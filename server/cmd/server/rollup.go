package main

import (
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/telemetry"
)

// Tổng hợp tự động 2 lần/ngày và trang xem nhanh (TELEMETRY_SPEC §4.0, §4.2).

// runRollupScheduler chạy tổng hợp đúng 12:00 và 23:30 giờ Việt Nam.
func (s *server) runRollupScheduler(stop <-chan struct{}) {
	for {
		next := telemetry.NextRollup(time.Now())
		t := time.NewTimer(time.Until(next))
		select {
		case <-stop:
			t.Stop()
			return
		case <-t.C:
			if err := s.runRollup(next); err != nil {
				log.Printf("telemetry: tổng hợp lỗi: %v", err)
			}
		}
	}
}

// runRollup tổng hợp từ lần chạy trước tới `at`, ghi docs/perf/real/ và nối 1 dòng CSV.
func (s *server) runRollup(at time.Time) error {
	if s.telemetry == nil {
		return nil
	}
	db := s.telemetry.DB()
	var from int64
	if err := db.QueryRow(`SELECT COALESCE(MAX(until_ts),0) FROM rollups`).Scan(&from); err != nil {
		return err
	}
	if from == 0 {
		// Lần đầu: lấy 12 giờ gần nhất thay vì toàn bộ lịch sử, để bản tổng hợp đầu tiên vẫn
		// là một khoảng có nghĩa chứ không trộn mọi thứ từ ngày khai sinh DB.
		from = at.Add(-12 * time.Hour).Unix()
	}
	to := at.Unix()
	d, err := telemetry.Load(db, telemetry.Opts{
		From: from, To: to, Build: s.telemetry.BuildID(),
		Label: fmt.Sprintf("%s → %s", time.Unix(from, 0).In(telemetry.VNLocation()).Format("02/01 15:04"), at.Format("02/01 15:04")),
	})
	if err != nil {
		return err
	}
	// Ghi mốc TRƯỚC khi quyết định có tạo file hay không: khoảng không có ai chơi vẫn phải
	// được đánh dấu là đã xét, nếu không lần sau lại quét lại đúng khoảng rỗng đó mãi mãi.
	if _, err := db.Exec(`INSERT INTO rollups (ran_at, until_ts) VALUES (?,?)`, time.Now().Unix(), to); err != nil {
		return err
	}
	md := d.Markdown()
	if md == "" {
		log.Printf("telemetry: tổng hợp %s — không có người chơi, không tạo file", at.Format("15:04"))
		return nil
	}
	dir := repoPath("docs", "perf", "real")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := filepath.Join(dir, at.Format("2006-01-02_1504")+".md")
	if err := os.WriteFile(name, []byte(md), 0o644); err != nil {
		return err
	}
	if err := appendCSV(repoPath("docs", "perf_log.csv"), d.Summarize()); err != nil {
		log.Printf("telemetry: nối CSV: %v", err)
	}
	log.Printf("telemetry: đã ghi %s (%d phiên)", name, len(d.Sessions))
	return nil
}

func repoPath(parts ...string) string {
	repo := filepath.Dir(filepath.Dir(config.FindRepoFile("config/game.json")))
	return filepath.Join(append([]string{repo}, parts...)...)
}

// appendCSV nối một dòng S6 vào docs/perf_log.csv.
func appendCSV(path string, s telemetry.S6) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, s.CSVRow(0, ""))
	return err
}

// handleAdminPerf là trang xem nhanh §4.2: CHỈ LAN + mật khẩu trong biến môi trường.
func (s *server) handleAdminPerf(w http.ResponseWriter, r *http.Request) {
	if s.telemetry == nil {
		http.Error(w, "telemetry tắt", http.StatusNotFound)
		return
	}
	if !isLANAddr(clientIP(r)) {
		// 404 chứ không 403: ngoài LAN thì không cần biết trang này tồn tại.
		http.NotFound(w, r)
		return
	}
	pass := os.Getenv("KX_ADMIN_PASS")
	if pass == "" {
		http.Error(w, "chưa đặt KX_ADMIN_PASS — trang này tắt cho tới khi có mật khẩu", http.StatusForbidden)
		return
	}
	if _, got, ok := r.BasicAuth(); !ok || got != pass {
		w.Header().Set("WWW-Authenticate", `Basic realm="kx perf"`)
		http.Error(w, "cần mật khẩu", http.StatusUnauthorized)
		return
	}
	since := 24 * time.Hour
	if v := r.URL.Query().Get("since"); v != "" {
		if dur, err := telemetry.ParseSince(v); err == nil {
			since = dur
		}
	}
	now := time.Now()
	d, err := telemetry.Load(s.telemetry.DB(), telemetry.Opts{
		From: now.Add(-since).Unix(), To: now.Unix(),
		Session: r.URL.Query().Get("session"),
		Label:   since.String(), Build: s.telemetry.BuildID(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	md := d.Markdown()
	if md == "" {
		md = "# Chưa có số liệu\n\nKhông có phiên nào trong khoảng này."
	}
	written, dropped := s.telemetry.Stats()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>Kiếm Xoay — số liệu thật</title>
<style>body{font:14px/1.5 system-ui,sans-serif;margin:2rem auto;max-width:70rem;padding:0 1rem}
pre{background:#f4f4f4;padding:.75rem;overflow-x:auto;white-space:pre-wrap}
table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:.2rem .5rem}
@media(prefers-color-scheme:dark){body{background:#1a1a1a;color:#eee}pre{background:#262626}td,th{border-color:#444}}</style>
<p><a href="?since=1h">1h</a> · <a href="?since=24h">24h</a> · <a href="?since=7d">7d</a>
 — đã ghi %d bản ghi, vứt %d (hàng đợi đầy)</p><pre>%s</pre>`, written, dropped, html.EscapeString(md))
}

// isLANAddr: chỉ mạng nội bộ mới xem được trang quản trị (§4.2).
func isLANAddr(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
