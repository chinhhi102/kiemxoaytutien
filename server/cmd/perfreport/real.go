package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/telemetry"
)

// Chế độ `--real`: in báo cáo số liệu NGƯỜI CHƠI THẬT (TELEMETRY_SPEC §4.1).
//
// Mở DB ở chế độ CHỈ ĐỌC: lệnh này thường chạy trong lúc server đang phục vụ người chơi, và
// một lệnh báo cáo không bao giờ được phép làm hỏng dữ liệu đang được ghi.

func reportReal(dbPath, since, session string) error {
	if dbPath == "" {
		repo := filepath.Dir(filepath.Dir(config.FindRepoFile("config/game.json")))
		dbPath = filepath.Join(repo, "server", "data", "telemetry.db")
	}
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("chưa có %s — server đã chạy với -telemetry chưa?", dbPath)
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer db.Close()

	dur, err := parseSinceArg(since)
	if err != nil {
		return fmt.Errorf("--since %q: %w", since, err)
	}
	now := time.Now()
	d, err := telemetry.Load(db, telemetry.Opts{
		From: now.Add(-dur).Unix(), To: now.Unix(),
		Session: session, Label: since, Build: buildIDOf(db),
	})
	if err != nil {
		return err
	}
	md := d.Markdown()
	if md == "" {
		// Không có phiên nào không phải là lỗi — chỉ là chưa ai vào chơi (§4.0).
		fmt.Printf("Không có phiên nào trong %s. Chưa có gì để báo cáo.\n", since)
		return nil
	}
	fmt.Print(md)
	return nil
}

// buildIDOf lấy build id gần nhất server đã ghi, để dòng S6 nói đúng nó đo bản nào.
func buildIDOf(db *sql.DB) string {
	var id string
	if err := db.QueryRow(`SELECT build_id FROM server_samples ORDER BY ts DESC LIMIT 1`).Scan(&id); err != nil {
		return "?"
	}
	return id
}

func parseSinceArg(v string) (time.Duration, error) {
	if strings.HasSuffix(v, "d") {
		var days int
		if _, err := fmt.Sscanf(v, "%dd", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(v)
}
