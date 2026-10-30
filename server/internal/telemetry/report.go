package telemetry

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Báo cáo số liệu người chơi thật — đủ 7 mục của TELEMETRY_SPEC §4.1.
//
// Dùng chung cho `cmd/perfreport --real`, trang `/admin/perf` và bản tổng hợp tự động 2
// lần/ngày, để ba nơi không bao giờ nói ba con số khác nhau về cùng một khoảng thời gian.

// sampleCap: trần số mẫu nạp vào RAM cho một lần báo cáo.
//
// Phân vị phải tính trong Go (SQLite không có hàm percentile), nên phải nạp mẫu lên. 200 000
// mẫu ≈ 23 ngày chơi liên tục của 10 người — quá trần thì báo cáo NÓI RA là đã cắt, thay vì
// âm thầm đưa ra con số tính trên một phần dữ liệu.
const sampleCap = 200000

// matchWindow: hai mẫu cách nhau dưới ngần này giây thì coi là "cùng lúc" khi ghép
// client ↔ server. Mẫu server đi mỗi 10 giây nên 15 là đủ rộng để luôn bắt được mẫu gần
// nhất, mà vẫn đủ hẹp để không ghép nhầm sang nhịp kế tiếp.
const matchWindow = 15

type sampleRow struct {
	Session string
	// TS: đồng hồ CLIENT, mili giây. SrvTS: đồng hồ SERVER, giây.
	//
	// Giữ cả hai và không bao giờ lẫn: mọi phép GHÉP theo thời gian phải dùng `SrvTS`, vì
	// đồng hồ máy người chơi lệch tuỳ máy (có máy lệch vài phút) và ghép theo nó thì không
	// bao giờ khớp với mẫu server. Lúc đầu tôi đọc nhầm cột `server_ts` vào `TS`, mục 5 của
	// báo cáo im lặng trả "không có mẫu server cùng lúc" cho mọi dòng —
	// `TestGhepClientServerChiRaNghenODau` bắt được.
	TS                            int64
	SrvTS                         int64
	FPSAvg, FPSP1, FrameP95       float64
	Over33, Over100, LongTasks    int
	DrawCalls, Sprites, TexMB     float64
	Placeholder, Respawn2s        int
	PingP50, PingP95, PingMax     float64
	KbpsIn, KbpsOut               float64
	MaxPkt                        int
	Corrections                   float64
	Zone, Realm                   int
	CharsView, TienView           int
	Burning, Zoning               bool
	Quality, GPU, Browser, OSName string
	Mobile                        bool
}

type sessionRow struct {
	ID        string
	GuestHash string
	Build     string
	StartedAt int64
	DurSecs   float64
	NSamples  int
	FPSMedian float64
	Browser   string
	OSName    string
	GPU       string
	Mobile    bool
	Cores     int
	Quality   string
}

// Opts chọn khoảng thời gian cho báo cáo.
type Opts struct {
	From, To int64  // unix giây; To == 0 nghĩa là "tới bây giờ"
	Session  string // chỉ một phiên (bỏ qua From/To)
	Label    string // nhãn hiển thị cho khoảng, ví dụ "24h"
	Build    string // build id đang chạy, ghi vào dòng S6
}

// Data là toàn bộ số liệu một lần báo cáo cần, nạp một lần rồi dùng lại cho cả 7 mục.
type Data struct {
	Samples   []sampleRow
	Sessions  []sessionRow
	Events    []eventRow
	SrvSample []srvRow
	Truncated bool
	Opts      Opts
}

type eventRow struct {
	Session string
	TS      int64
	Kind    string
	Data    string
}

type srvRow struct {
	TS               int64
	Zone             int
	ZoneName         string
	TickAvg, TickP95 float64
	Humans, Bots     int
	KbpsOut          float64
	CPUPct, RSSMB    float64
	BuildID          string
}

// Load nạp số liệu trong khoảng. Lỗi SQL trả về ngay — thà không có báo cáo còn hơn một
// báo cáo thiếu một nửa mà không ai biết.
func Load(db *sql.DB, o Opts) (*Data, error) {
	d := &Data{Opts: o}
	where, args := " WHERE s.server_ts BETWEEN ? AND ?", []any{o.From, o.To}
	if o.Session != "" {
		where, args = " WHERE s.session_id = ?", []any{o.Session}
	}
	rows, err := db.Query(`SELECT s.session_id,s.ts,s.server_ts,s.fps_avg,s.fps_p1,s.frame_p95,
		s.over33,s.over100,s.long_tasks,s.draw_calls,s.sprites,s.tex_mb,s.placeholder,s.respawn2s,
		s.ping_p50,s.ping_p95,s.ping_max,s.kbps_in,s.kbps_out,s.max_pkt,s.corrections,
		s.zone,s.realm,s.chars_view,s.tien_view,s.burning,s.zoning,
		COALESCE(ss.quality,''),COALESCE(ss.gpu,''),COALESCE(ss.browser,''),COALESCE(ss.os,''),COALESCE(ss.mobile,0)
		FROM samples s LEFT JOIN sessions ss ON ss.session_id = s.session_id`+where+
		` ORDER BY s.server_ts LIMIT ?`, append(args, sampleCap+1)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r sampleRow
		var burn, zon, mob int
		if err := rows.Scan(&r.Session, &r.TS, &r.SrvTS, &r.FPSAvg, &r.FPSP1, &r.FrameP95,
			&r.Over33, &r.Over100, &r.LongTasks, &r.DrawCalls, &r.Sprites, &r.TexMB, &r.Placeholder, &r.Respawn2s,
			&r.PingP50, &r.PingP95, &r.PingMax, &r.KbpsIn, &r.KbpsOut, &r.MaxPkt, &r.Corrections,
			&r.Zone, &r.Realm, &r.CharsView, &r.TienView, &burn, &zon,
			&r.Quality, &r.GPU, &r.Browser, &r.OSName, &mob); err != nil {
			return nil, err
		}
		r.Burning, r.Zoning, r.Mobile = burn != 0, zon != 0, mob != 0
		d.Samples = append(d.Samples, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(d.Samples) > sampleCap {
		d.Samples, d.Truncated = d.Samples[:sampleCap], true
	}

	sWhere, sArgs := " WHERE started_at BETWEEN ? AND ?", []any{o.From, o.To}
	if o.Session != "" {
		sWhere, sArgs = " WHERE session_id = ?", []any{o.Session}
	}
	srows, err := db.Query(`SELECT session_id,guest_hash,build,started_at,dur_secs,n_samples,fps_median,
		browser,os,gpu,mobile,cores,quality FROM sessions`+sWhere, sArgs...)
	if err != nil {
		return nil, err
	}
	defer srows.Close()
	for srows.Next() {
		var s sessionRow
		var mob int
		if err := srows.Scan(&s.ID, &s.GuestHash, &s.Build, &s.StartedAt, &s.DurSecs, &s.NSamples, &s.FPSMedian,
			&s.Browser, &s.OSName, &s.GPU, &mob, &s.Cores, &s.Quality); err != nil {
			return nil, err
		}
		s.Mobile = mob != 0
		d.Sessions = append(d.Sessions, s)
	}
	if err := srows.Err(); err != nil {
		return nil, err
	}

	eWhere, eArgs := " WHERE server_ts BETWEEN ? AND ?", []any{o.From, o.To}
	if o.Session != "" {
		eWhere, eArgs = " WHERE session_id = ?", []any{o.Session}
	}
	erows, err := db.Query(`SELECT session_id,server_ts,kind,data FROM events`+eWhere+` LIMIT 20000`, eArgs...)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var e eventRow
		if err := erows.Scan(&e.Session, &e.TS, &e.Kind, &e.Data); err != nil {
			return nil, err
		}
		d.Events = append(d.Events, e)
	}
	if err := erows.Err(); err != nil {
		return nil, err
	}

	vrows, err := db.Query(`SELECT ts,zone,zone_name,tick_avg,tick_p95,humans,bots,kbps_out,cpu_pct,rss_mb,build_id
		FROM server_samples WHERE ts BETWEEN ? AND ? ORDER BY ts LIMIT ?`, o.From, o.To, sampleCap)
	if err != nil {
		return nil, err
	}
	defer vrows.Close()
	for vrows.Next() {
		var s srvRow
		if err := vrows.Scan(&s.TS, &s.Zone, &s.ZoneName, &s.TickAvg, &s.TickP95, &s.Humans, &s.Bots,
			&s.KbpsOut, &s.CPUPct, &s.RSSMB, &s.BuildID); err != nil {
			return nil, err
		}
		d.SrvSample = append(d.SrvSample, s)
	}
	return d, vrows.Err()
}

// ---- thống kê cơ bản ----

func pct(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(float64(len(s)-1) * p)
	return s[i]
}

func median(v []float64) float64 { return pct(v, 0.5) }

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t / float64(len(v))
}

func maxOf(v []float64) float64 {
	m := 0.0
	for _, x := range v {
		m = math.Max(m, x)
	}
	return m
}

func col(rows []sampleRow, f func(sampleRow) float64) []float64 {
	out := make([]float64, len(rows))
	for i, r := range rows {
		out[i] = f(r)
	}
	return out
}

func sumInt(rows []sampleRow, f func(sampleRow) int) int {
	t := 0
	for _, r := range rows {
		t += f(r)
	}
	return t
}

// num định dạng số cho bảng; không có mẫu thì in "–" chứ không in 0.00 (0 và "không đo
// được" là hai chuyện khác nhau, lẫn lộn hai thứ này là cách nhanh nhất để đọc sai bảng).
func num(v float64, n int, has bool) string {
	if !has {
		return "–"
	}
	return fmt.Sprintf("%.*f", n, v)
}

// deviceClass xếp một phiên vào nhóm của §4.1 mục 3.
//
// Phân biệt GPU rời / tích hợp bằng tên chuỗi WEBGL_debug_renderer_info trả về. Thô, nhưng
// đó là tất cả những gì trình duyệt cho biết, và nhóm sai còn hơn không chia nhóm.
func deviceClass(mobile bool, gpu string) string {
	if mobile {
		return "Điện thoại"
	}
	g := strings.ToLower(gpu)
	switch {
	case strings.Contains(g, "intel") || strings.Contains(g, "uhd") || strings.Contains(g, "iris") ||
		strings.Contains(g, "vega") && strings.Contains(g, "graphics") || strings.Contains(g, "swiftshader"):
		return "PC GPU tích hợp"
	case g == "":
		return "PC (không rõ GPU)"
	default:
		return "PC GPU rời"
	}
}

// ---- dòng S6 cho PERF_LOG + CSV ----

// S6 là một dòng tổng hợp số liệu người thật, đúng các cột của PERF_LOG §3 và §8.
type S6 struct {
	Date        string
	Build       string
	Label       string
	NSessions   int
	TickAvg     float64
	TickP95     float64
	CPUPct      float64
	KbpsAvg     float64
	KbpsPeak    float64
	PktMaxKB    float64
	PingP95     float64
	FPSAvg      float64
	FPSLow1     float64
	DrawCalls   float64
	Sprites     float64
	TexMB       float64
	Placeholder int
	RespawnMin  float64
	HasClient   bool
	HasServer   bool
	Notes       string
}

// Summarize rút một dòng S6 từ dữ liệu đã nạp.
//
// Dùng TRUNG VỊ THEO PHIÊN chứ không trung bình theo mẫu (§4.1 mục 2): một người ngồi chơi
// 2 tiếng sẽ sinh gấp 60 lần số mẫu của người vào 2 phút, và trung bình theo mẫu thì bảng
// chỉ còn nói về đúng cái máy của người ngồi lâu nhất.
func (d *Data) Summarize() S6 {
	s := S6{Date: time.Unix(d.Opts.To, 0).Format("2006-01-02"), Build: d.Opts.Build, Label: d.Opts.Label}
	byS := map[string][]sampleRow{}
	for _, r := range d.Samples {
		byS[r.Session] = append(byS[r.Session], r)
	}
	s.NSessions = len(byS)
	if len(d.Samples) > 0 {
		s.HasClient = true
		var fpsAvg, fpsLow, ping, kbpsIn, draw, spr, tex []float64
		for _, rows := range byS {
			fpsAvg = append(fpsAvg, median(col(rows, func(r sampleRow) float64 { return r.FPSAvg })))
			// FPS 1% thấp: lấy cái TỆ NHẤT của phiên, không lấy trung vị. Chỉ số này tồn tại
			// để bắt cú khựng, mà trung vị thì xoá sạch cú khựng.
			fpsLow = append(fpsLow, pct(col(rows, func(r sampleRow) float64 { return r.FPSP1 }), 0.1))
			ping = append(ping, maxOf(col(rows, func(r sampleRow) float64 { return r.PingP95 })))
			kbpsIn = append(kbpsIn, median(col(rows, func(r sampleRow) float64 { return r.KbpsIn })))
			draw = append(draw, median(col(rows, func(r sampleRow) float64 { return r.DrawCalls })))
			spr = append(spr, median(col(rows, func(r sampleRow) float64 { return r.Sprites })))
			tex = append(tex, median(col(rows, func(r sampleRow) float64 { return r.TexMB })))
		}
		s.FPSAvg, s.FPSLow1 = median(fpsAvg), median(fpsLow)
		s.PingP95 = median(ping)
		s.KbpsAvg = median(kbpsIn)
		s.KbpsPeak = maxOf(col(d.Samples, func(r sampleRow) float64 { return r.KbpsIn }))
		s.PktMaxKB = maxOf(col(d.Samples, func(r sampleRow) float64 { return float64(r.MaxPkt) })) / 1024
		s.DrawCalls, s.Sprites, s.TexMB = median(draw), median(spr), median(tex)
		s.Placeholder = sumInt(d.Samples, func(r sampleRow) int { return r.Placeholder })
		respawn := sumInt(d.Samples, func(r sampleRow) int { return r.Respawn2s })
		// Mỗi mẫu là 10 giây, nên số mẫu / 6 là số phút đã quan sát.
		if mins := float64(len(d.Samples)) / 6; mins > 0 {
			s.RespawnMin = float64(respawn) / mins
		}
	}
	if len(d.SrvSample) > 0 {
		s.HasServer = true
		var avg, p95, cpu []float64
		for _, r := range d.SrvSample {
			avg = append(avg, r.TickAvg)
			p95 = append(p95, r.TickP95)
			cpu = append(cpu, r.CPUPct)
		}
		s.TickAvg, s.TickP95, s.CPUPct = mean(avg), pct(p95, 0.95), median(cpu)
	}
	if s.NSessions < 5 {
		s.Notes = fmt.Sprintf("ít mẫu, tham khảo (N=%d)", s.NSessions)
	}
	return s
}

// MarkdownRow in dòng chép thẳng vào bảng PERF_LOG §3.
func (s S6) MarkdownRow() string {
	return fmt.Sprintf("| – | %s | `%s` | Người thật %s (N=%d phiên) | S6 | %s / %s | %s / %s | %s | %s / %s | – | %s | %s | %s | – | %s |",
		s.Date, s.Build, s.Label, s.NSessions,
		num(s.TickAvg, 2, s.HasServer), num(s.TickP95, 2, s.HasServer),
		num(s.KbpsAvg, 1, s.HasClient), num(s.KbpsPeak, 1, s.HasClient),
		num(s.PktMaxKB, 1, s.HasClient),
		num(s.FPSAvg, 0, s.HasClient), num(s.FPSLow1, 0, s.HasClient),
		num(s.DrawCalls, 0, s.HasClient),
		num(float64(s.Placeholder), 0, s.HasClient),
		num(s.RespawnMin, 2, s.HasClient),
		orDash(s.Notes))
}

// CSVRow in dòng cho docs/perf_log.csv (§8 + cột n_sessions của v6.2).
func (s S6) CSVRow(run int, commit string) string {
	f := func(v float64, n int, has bool) string {
		if !has {
			return ""
		}
		return fmt.Sprintf("%.*f", n, v)
	}
	return strings.Join([]string{
		fmt.Sprintf("%d", run), s.Date, commit, fmt.Sprintf("%q", "Người thật "+s.Label), "S6",
		f(s.TickAvg, 2, s.HasServer), f(s.TickP95, 2, s.HasServer), f(s.CPUPct, 1, s.HasServer),
		f(s.KbpsAvg, 1, s.HasClient), f(s.KbpsPeak, 1, s.HasClient), f(s.PktMaxKB, 1, s.HasClient),
		f(s.PingP95, 1, s.HasClient), f(s.FPSAvg, 1, s.HasClient), f(s.FPSLow1, 1, s.HasClient),
		"", f(s.DrawCalls, 0, s.HasClient), f(s.Sprites, 0, s.HasClient), f(s.TexMB, 1, s.HasClient),
		fmt.Sprintf("%d", s.Placeholder), f(s.RespawnMin, 2, s.HasClient),
		"", "", "", fmt.Sprintf("%q", s.Notes), fmt.Sprintf("%d", s.NSessions),
	}, ",")
}

func orDash(s string) string {
	if s == "" {
		return "–"
	}
	return s
}

// ---- báo cáo đầy đủ 7 mục ----

// Markdown dựng báo cáo §4.1. Trả về chuỗi rỗng khi KHÔNG CÓ PHIÊN NÀO — §4.0 nói rõ lúc đó
// không tạo file, và "không ai chơi" không phải là một kết quả đo.
func (d *Data) Markdown() string {
	if len(d.Sessions) == 0 && len(d.Samples) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Số liệu người chơi thật — %s\n\n", d.Opts.Label)
	fmt.Fprintf(&b, "Khoảng: %s → %s · build `%s`\n\n",
		time.Unix(d.Opts.From, 0).Format("2006-01-02 15:04"),
		time.Unix(d.Opts.To, 0).Format("2006-01-02 15:04"), d.Opts.Build)
	if d.Truncated {
		fmt.Fprintf(&b, "> ⚠ Vượt trần %d mẫu — các con số dưới đây tính trên %d mẫu ĐẦU của khoảng, không phải toàn bộ.\n\n", sampleCap, sampleCap)
	}
	d.sec1Overview(&b)
	d.sec2S6(&b)
	d.sec3ByDevice(&b)
	d.sec4BySituation(&b)
	d.sec5ClientServer(&b)
	d.sec6WorstSessions(&b)
	d.sec7Errors(&b)
	d.secAlerts(&b)
	return b.String()
}

func (d *Data) sec1Overview(b *strings.Builder) {
	guests, devices := map[string]bool{}, map[string]int{}
	hours := 0.0
	for _, s := range d.Sessions {
		guests[s.GuestHash] = true
		hours += s.DurSecs / 3600
		devices[deviceClass(s.Mobile, s.GPU)]++
	}
	fmt.Fprintf(b, "## 1. Tổng quan\n\n- Phiên: **%d** · người khác nhau: **%d** · tổng giờ chơi: **%.2f h**\n",
		len(d.Sessions), len(guests), hours)
	if len(devices) > 0 {
		fmt.Fprintf(b, "- Thiết bị: %s\n", joinCounts(devices))
	}
	brows := map[string]int{}
	for _, s := range d.Sessions {
		if s.Browser != "" {
			brows[s.Browser]++
		}
	}
	if len(brows) > 0 {
		fmt.Fprintf(b, "- Trình duyệt: %s\n", joinCounts(brows))
	}
	ended := 0
	for _, s := range d.Sessions {
		if s.DurSecs > 0 {
			ended++
		}
	}
	// Phiên không có tóm tắt = tab bị đóng mà sendBeacon không tới nơi (hoặc còn đang chơi).
	// Nói ra để không ai tưởng tổng giờ chơi ở trên là đầy đủ.
	if miss := len(d.Sessions) - ended; miss > 0 {
		fmt.Fprintf(b, "- %d phiên chưa có tóm tắt kết thúc (đang chơi, hoặc sendBeacon không tới) → tổng giờ chơi là mức SÀN.\n", miss)
	}
	b.WriteString("\n")
}

func (d *Data) sec2S6(b *strings.Builder) {
	s := d.Summarize()
	b.WriteString("## 2. Dòng S6 (chép vào PERF_LOG §3)\n\n```\n")
	b.WriteString(s.MarkdownRow())
	b.WriteString("\n```\n\nDòng CSV (`docs/perf_log.csv`):\n\n```\n")
	b.WriteString(s.CSVRow(0, ""))
	b.WriteString("\n```\n\n")
}

func (d *Data) sec3ByDevice(b *strings.Builder) {
	b.WriteString("## 3. Theo nhóm thiết bị\n\n| Nhóm | Phiên | FPS trung vị | FPS 1% thấp | Ping p95 | KB/s |\n|---|---|---|---|---|---|\n")
	cls := map[string][]sampleRow{}
	sessCls := map[string]map[string]bool{}
	for _, r := range d.Samples {
		c := deviceClass(r.Mobile, r.GPU)
		cls[c] = append(cls[c], r)
		if sessCls[c] == nil {
			sessCls[c] = map[string]bool{}
		}
		sessCls[c][r.Session] = true
	}
	for _, c := range sortedKeys(cls) {
		rows := cls[c]
		fmt.Fprintf(b, "| %s | %d | %.0f | %.0f | %.0f ms | %.1f |\n", c, len(sessCls[c]),
			median(col(rows, func(r sampleRow) float64 { return r.FPSAvg })),
			pct(col(rows, func(r sampleRow) float64 { return r.FPSP1 }), 0.1),
			median(col(rows, func(r sampleRow) float64 { return r.PingP95 })),
			median(col(rows, func(r sampleRow) float64 { return r.KbpsIn })))
	}
	b.WriteString("\n")
}

func (d *Data) sec4BySituation(b *strings.Builder) {
	b.WriteString("## 4. Chỗ nào làm tụt FPS\n\n")
	bucketTable(b, "Số nhân vật trong khung", d.Samples, func(r sampleRow) string {
		switch {
		case r.CharsView <= 5:
			return "0–5"
		case r.CharsView <= 15:
			return "6–15"
		default:
			return "16+"
		}
	})
	bucketTable(b, "Số người Tiên giới trong khung", d.Samples, func(r sampleRow) string {
		switch {
		case r.TienView == 0:
			return "0"
		case r.TienView <= 2:
			return "1–2"
		default:
			return "3+"
		}
	})
	bucketTable(b, "Khu", d.Samples, func(r sampleRow) string { return fmt.Sprintf("khu %d", r.Zone) })
	bucketTable(b, "Mức chất lượng", d.Samples, func(r sampleRow) string { return orDash(r.Quality) })
	bucketTable(b, "Đang đổi khu", d.Samples, func(r sampleRow) string {
		if r.Zoning {
			return "đang đổi khu"
		}
		return "bình thường"
	})
}

func bucketTable(b *strings.Builder, title string, rows []sampleRow, key func(sampleRow) string) {
	buckets := map[string][]sampleRow{}
	for _, r := range rows {
		buckets[key(r)] = append(buckets[key(r)], r)
	}
	if len(buckets) == 0 {
		return
	}
	fmt.Fprintf(b, "**%s**\n\n| | Mẫu | FPS trung vị | FPS 1%% thấp | Vẽ p95 (ms) |\n|---|---|---|---|---|\n", title)
	for _, k := range sortedKeys(buckets) {
		v := buckets[k]
		fmt.Fprintf(b, "| %s | %d | %.0f | %.0f | %.1f |\n", k, len(v),
			median(col(v, func(r sampleRow) float64 { return r.FPSAvg })),
			pct(col(v, func(r sampleRow) float64 { return r.FPSP1 }), 0.1),
			median(col(v, func(r sampleRow) float64 { return r.FrameP95 })))
	}
	b.WriteString("\n")
}

// sec5ClientServer ghép những đoạn client tụt FPS với tick server CÙNG LÚC ĐÓ (§4.1 mục 5).
//
// Đây là mục trả lời câu đắt nhất: tụt vì máy người chơi, vì server, hay vì mạng. Không có
// nó thì mỗi lần ai đó báo "game giật" lại phải đoán.
func (d *Data) sec5ClientServer(b *strings.Builder) {
	b.WriteString("## 5. Ghép client ↔ server\n\n")
	if len(d.SrvSample) == 0 {
		b.WriteString("Không có mẫu server trong khoảng này.\n\n")
		return
	}
	// Ghép theo KHOẢNG DUNG SAI, không chia ô 10 giây.
	//
	// Chia ô là sai và sai âm thầm: mẫu server ghi vào một giây bất kỳ, mẫu client ghi vào
	// một giây khác, nên hai cái cách nhau 2 giây vẫn rơi vào hai ô khác nhau (1793340158 →
	// ô ...150, còn 1793340161 → ô ...160) và phép ghép không bao giờ khớp. Cả mục 5 im lặng
	// trả "không có mẫu server cùng lúc" cho mọi dòng — đúng như test bắt được.
	srv := append([]srvRow(nil), d.SrvSample...)
	sort.Slice(srv, func(i, j int) bool { return srv[i].TS < srv[j].TS })
	type bad struct {
		r       sampleRow
		tick    float64
		humans  int
		hasTick bool
	}
	var list []bad
	for _, r := range d.Samples {
		if r.FPSP1 >= 30 && r.PingP95 <= 150 {
			continue
		}
		e := bad{r: r}
		// Mẫu server đi mỗi 10 giây, nên bất cứ mẫu nào cách dưới 15 giây đều là "cùng lúc".
		lo := sort.Search(len(srv), func(i int) bool { return srv[i].TS >= r.SrvTS-matchWindow })
		for i := lo; i < len(srv) && srv[i].TS <= r.SrvTS+matchWindow; i++ {
			e.tick = math.Max(e.tick, srv[i].TickP95)
			e.humans += srv[i].Humans
			e.hasTick = true
		}
		list = append(list, e)
	}
	if len(list) == 0 {
		b.WriteString("Không có mẫu nào tụt dưới 30 FPS hay ping quá 150 ms. Không có gì để ghép.\n\n")
		return
	}
	sort.Slice(list, func(i, j int) bool { return list[i].r.FPSP1 < list[j].r.FPSP1 })
	if len(list) > 10 {
		list = list[:10]
	}
	b.WriteString("| Lúc | FPS 1% | Ping p95 | Tick server p95 | Người | Kết luận |\n|---|---|---|---|---|---|\n")
	for _, e := range list {
		verdict := "client"
		switch {
		case !e.hasTick:
			verdict = "không có mẫu server cùng lúc"
		case e.tick > 8:
			verdict = "**server**"
		case e.r.PingP95 > 150 && e.tick <= 8:
			verdict = "**mạng**"
		}
		fmt.Fprintf(b, "| %s | %.0f | %.0f ms | %s | %d | %s |\n",
			time.Unix(e.r.SrvTS, 0).Format("15:04:05"), e.r.FPSP1, e.r.PingP95,
			num(e.tick, 2, e.hasTick), e.humans, verdict)
	}
	b.WriteString("\n")
}

func (d *Data) sec6WorstSessions(b *strings.Builder) {
	b.WriteString("## 6. Top 5 phiên tệ nhất\n\n")
	type worst struct {
		s     sessionRow
		low   float64
		rows  int
		chars float64
	}
	byS := map[string][]sampleRow{}
	for _, r := range d.Samples {
		byS[r.Session] = append(byS[r.Session], r)
	}
	var list []worst
	for _, s := range d.Sessions {
		rows := byS[s.ID]
		if len(rows) == 0 {
			continue
		}
		list = append(list, worst{s: s,
			low:   pct(col(rows, func(r sampleRow) float64 { return r.FPSP1 }), 0.1),
			rows:  len(rows),
			chars: median(col(rows, func(r sampleRow) float64 { return float64(r.CharsView) }))})
	}
	if len(list) == 0 {
		b.WriteString("Chưa có phiên nào có mẫu.\n\n")
		return
	}
	sort.Slice(list, func(i, j int) bool { return list[i].low < list[j].low })
	if len(list) > 5 {
		list = list[:5]
	}
	evByS := map[string][]string{}
	for _, e := range d.Events {
		evByS[e.Session] = append(evByS[e.Session], e.Kind)
	}
	b.WriteString("| Phiên | Thiết bị | FPS 1% | Mẫu | Nhân vật/khung | Sự kiện |\n|---|---|---|---|---|---|\n")
	for _, w := range list {
		ev := "–"
		if k := evByS[w.s.ID]; len(k) > 0 {
			ev = strings.Join(dedup(k), ", ")
		}
		fmt.Fprintf(b, "| `%s` | %s %s | %.0f | %d | %.0f | %s |\n",
			short(w.s.ID), deviceClass(w.s.Mobile, w.s.GPU), w.s.Browser, w.low, w.rows, w.chars, ev)
	}
	b.WriteString("\n")
}

func (d *Data) sec7Errors(b *strings.Builder) {
	b.WriteString("## 7. Lỗi\n\n")
	type agg struct {
		n     int
		sess  map[string]bool
		first string
	}
	errs := map[string]*agg{}
	for _, e := range d.Events {
		if e.Kind != "error" {
			continue
		}
		key := firstLine(e.Data)
		a := errs[key]
		if a == nil {
			a = &agg{sess: map[string]bool{}, first: e.Data}
			errs[key] = a
		}
		a.n++
		a.sess[e.Session] = true
	}
	if len(errs) == 0 {
		b.WriteString("Không có lỗi nào được báo.\n\n")
		return
	}
	keys := sortedKeysAgg(errs)
	b.WriteString("| Lỗi | Lần | Phiên |\n|---|---|---|\n")
	for _, k := range keys {
		fmt.Fprintf(b, "| %s | %d | %d |\n", k, errs[k].n, len(errs[k].sess))
	}
	b.WriteString("\n")
}

// secAlerts in các ngưỡng §4.3 đã vượt. Đứng CUỐI báo cáo chứ không đầu: đọc xong số liệu
// rồi mới tới kết luận, và kết luận thì cần nói rõ dựa trên con số nào.
func (d *Data) secAlerts(b *strings.Builder) {
	b.WriteString("## ⚠ Ngưỡng cảnh báo (§4.3)\n\n")
	var hits []string
	cls := map[string][]sampleRow{}
	for _, r := range d.Samples {
		cls[deviceClass(r.Mobile, r.GPU)] = append(cls[deviceClass(r.Mobile, r.GPU)], r)
	}
	for _, c := range sortedKeys(cls) {
		rows := cls[c]
		low := 0
		for _, r := range rows {
			if r.FPSP1 < 30 {
				low++
			}
		}
		if share := float64(low) / float64(len(rows)); share > 0.10 {
			hits = append(hits, fmt.Sprintf("**%s**: %.0f%% mẫu có FPS 1%% dưới 30 (ngưỡng 10%%)", c, share*100))
		}
	}
	if n := sumInt(d.Samples, func(r sampleRow) int { return r.Placeholder }); n > 0 {
		hits = append(hits, fmt.Sprintf("**%d khung placeholder** ở build thật (ngân sách 0)", n))
	}
	if n := sumInt(d.Samples, func(r sampleRow) int { return r.Respawn2s }); n > 0 {
		hits = append(hits, fmt.Sprintf("**%d lần entity respawn trong 2s** (ngân sách 0)", n))
	}
	lost := 0
	for _, e := range d.Events {
		if e.Kind == "webgl_context_lost" {
			lost++
		}
	}
	if lost > 0 {
		hits = append(hits, fmt.Sprintf("**%d lần mất ngữ cảnh WebGL**", lost))
	}
	if len(d.Samples) > 0 {
		if p := median(col(d.Samples, func(r sampleRow) float64 { return r.PingP95 })); p > 150 {
			hits = append(hits, fmt.Sprintf("**Ping p95 %.0f ms trên LAN** — LAN mà ping cao là nghẽn client hoặc server, không phải mạng", p))
		}
	}
	if len(hits) == 0 {
		b.WriteString("Không vượt ngưỡng nào.\n")
		return
	}
	for _, h := range hits {
		fmt.Fprintf(b, "- %s\n", h)
	}
}

// ---- tiện ích nhỏ ----

func joinCounts(m map[string]int) string {
	parts := make([]string, 0, len(m))
	for _, k := range sortedKeysInt(m) {
		parts = append(parts, fmt.Sprintf("%s %d", k, m[k]))
	}
	return strings.Join(parts, " · ")
}

func sortedKeys(m map[string][]sampleRow) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeysInt(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return m[out[i]] > m[out[j]] })
	return out
}

func sortedKeysAgg[T any](m map[string]*T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func dedup(v []string) []string {
	seen, out := map[string]bool{}, []string{}
	for _, s := range v {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}
