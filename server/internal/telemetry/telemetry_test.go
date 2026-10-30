package telemetry

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// open mở một DB telemetry tạm cho test. Dùng file thật chứ không dùng ":memory:" vì
// `sweep` đo dung lượng bằng os.Stat — DB trong RAM không có file để đo.
func open(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "t.db"), "test-build")
	if err != nil {
		t.Fatalf("mở: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// flush đợi goroutine ghi xử lý hết hàng đợi, KHÔNG đóng store (còn phải đọc lại DB).
// Đợi theo `time.Sleep` là cách viết test chớp tắt; `Sync` đợi đúng cái cần đợi.
func flush(t *testing.T, st *Store) {
	t.Helper()
	st.Sync()
}

func TestGhiVaDocLaiMotPhien(t *testing.T) {
	st := open(t)
	now := float64(time.Now().UnixMilli())
	st.AddSession(&Session{ID: "s1", Browser: "Chrome", OS: "Windows", GPU: "NVIDIA RTX", Quality: "high"}, "iphash")
	st.AddSample(&Sample{Session: "s1", TS: now, FPSAvg: 58, FPSP1: 42, PingP95: 30, KbpsIn: 12, CharsView: 8})
	st.AddEvent(&Event{Session: "s1", TS: now, Kind: "zone_switch", Data: "120ms"})
	st.AddEnd(&End{Session: "s1", DurSecs: 120, NSamples: 12, FPSMedian: 57})
	flush(t, st)

	d, err := Load(st.DB(), Opts{From: 0, To: time.Now().Unix() + 60, Label: "test"})
	if err != nil {
		t.Fatalf("đọc: %v", err)
	}
	if len(d.Sessions) != 1 || len(d.Samples) != 1 || len(d.Events) != 1 {
		t.Fatalf("mong 1/1/1, nhận %d phiên, %d mẫu, %d sự kiện", len(d.Sessions), len(d.Samples), len(d.Events))
	}
	if got := d.Sessions[0].DurSecs; got != 120 {
		t.Errorf("tóm tắt kết thúc không được ghi đè vào phiên: dur = %v, mong 120", got)
	}
	if got := d.Samples[0].Browser; got != "Chrome" {
		t.Errorf("mẫu phải ghép được với phiên để biết thiết bị: browser = %q", got)
	}
}

// guest_id KHÔNG được vào DB ở dạng thô (§1.3). Test này canh đúng luật ẩn danh đó.
func TestHashKhongLoNguyenBan(t *testing.T) {
	raw := "guest-abc-123"
	h := Hash(raw)
	if h == "" || strings.Contains(h, raw) {
		t.Fatalf("băm %q ra %q — vẫn lộ chuỗi gốc", raw, h)
	}
	if Hash(raw) != h {
		t.Error("băm phải ổn định: hai phiên của cùng một người cần ghép được với nhau")
	}
	if Hash("") != "" {
		t.Error("chuỗi rỗng phải ra rỗng, không ra băm của chuỗi rỗng")
	}
}

// Hàng đợi đầy thì VỨT chứ không chặn (luật 2 của gói này). Chặn ở đây nghĩa là vòng đọc
// WebSocket của một người đang chơi bị treo — telemetry tự gây ra đúng cái lag nó đi đo.
func TestDayThiVutChuKhongChan(t *testing.T) {
	st := open(t)
	defer st.Close()
	done := make(chan struct{})
	go func() {
		for i := 0; i < queueSize*3; i++ {
			st.AddSample(&Sample{Session: "s", FPSAvg: float64(i)})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("AddSample bị chặn — hàng đợi đầy phải vứt, không được đợi")
	}
}

func TestSummarizeDungTrungViTheoPhienKhongPhaiTheoMau(t *testing.T) {
	st := open(t)
	now := float64(time.Now().UnixMilli())
	// Phiên A: 100 mẫu, 20 FPS (người ngồi lâu, máy yếu).
	for i := 0; i < 100; i++ {
		st.AddSample(&Sample{Session: "A", TS: now, FPSAvg: 20, FPSP1: 20})
	}
	// Phiên B và C: mỗi phiên 1 mẫu, 60 FPS.
	st.AddSample(&Sample{Session: "B", TS: now, FPSAvg: 60, FPSP1: 60})
	st.AddSample(&Sample{Session: "C", TS: now, FPSAvg: 60, FPSP1: 60})
	st.AddSession(&Session{ID: "A"}, "")
	st.AddSession(&Session{ID: "B"}, "")
	st.AddSession(&Session{ID: "C"}, "")
	flush(t, st)

	d, err := Load(st.DB(), Opts{From: 0, To: time.Now().Unix() + 60})
	if err != nil {
		t.Fatal(err)
	}
	s := d.Summarize()
	if s.NSessions != 3 {
		t.Fatalf("mong 3 phiên, nhận %d", s.NSessions)
	}
	// Trung bình THEO MẪU sẽ ra ~20.8 (phiên A át hết). Trung vị THEO PHIÊN phải ra 60:
	// hai trong ba người chơi mượt, và bảng phải nói đúng điều đó.
	if s.FPSAvg != 60 {
		t.Errorf("FPS = %v; phiên dài không được át các phiên khác (mong 60)", s.FPSAvg)
	}
}

// Dòng S6 phải có ĐÚNG số cột của bảng PERF_LOG §3, nếu không chép vào là vỡ bảng.
func TestDongS6DungSoCot(t *testing.T) {
	// Header thật của bảng PERF_LOG §3. Dòng S6 lệch một cột là bảng vỡ khi chép vào.
	const header = "| # | Ngày | Commit | Lượt / thay đổi | Kịch bản | Tick TB / p95 (ms) | " +
		"KB/s TB / đỉnh | Gói max (KB) | FPS TB / 1% | Vẽ (ms) | Draw call | Placeholder | " +
		"Respawn/phút | Δ chính | Nút thắt còn lại |"
	row := S6{Date: "2026-10-30", Build: "b1", Label: "24h", NSessions: 3, HasClient: true, HasServer: true}.MarkdownRow()
	want, got := strings.Count(header, "|"), strings.Count(row, "|")
	if got != want {
		t.Errorf("dòng S6 có %d dấu | , header có %d:\n%s\n%s", got, want, header, row)
	}
}

func TestItMauThiTuDanhDau(t *testing.T) {
	st := open(t)
	now := float64(time.Now().UnixMilli())
	st.AddSession(&Session{ID: "only"}, "")
	st.AddSample(&Sample{Session: "only", TS: now, FPSAvg: 60})
	flush(t, st)
	d, _ := Load(st.DB(), Opts{From: 0, To: time.Now().Unix() + 60})
	if n := d.Summarize().Notes; !strings.Contains(n, "ít mẫu") {
		t.Errorf("1 phiên phải được đánh dấu ít mẫu, nhận %q", n)
	}
}

// CSV phải có đúng số cột của header docs/perf_log.csv + cột n_sessions của v6.2.
func TestCSVDungSoCot(t *testing.T) {
	const header = "run,date,commit,label,scenario,tick_avg_ms,tick_p95_ms,cpu_pct,kbps_avg,kbps_peak," +
		"pkt_max_kb,ping_p95_ms,fps_avg,fps_low1,draw_ms,draw_calls,sprites,tex_mb,placeholder_frames," +
		"respawn_per_min,zone_switch_cached_ms,zone_switch_cold_ms,deaths_per_bot_5min,notes,n_sessions"
	want := strings.Count(header, ",")
	got := strings.Count(S6{Date: "2026-10-30"}.CSVRow(1, "abc1234"), ",")
	if got != want {
		t.Errorf("dòng CSV có %d dấu phẩy, header có %d", got, want)
	}
}

// Không có phiên nào → KHÔNG tạo báo cáo (§4.0). "Không ai chơi" không phải một kết quả đo.
func TestKhongCoPhienThiKhongCoBaoCao(t *testing.T) {
	st := open(t)
	flush(t, st)
	d, err := Load(st.DB(), Opts{From: 0, To: time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if md := d.Markdown(); md != "" {
		t.Errorf("mong chuỗi rỗng, nhận %d ký tự", len(md))
	}
}

// Ngưỡng §4.3: một khung placeholder hay một lần respawn trong 2s ở build thật là phải báo đỏ.
func TestNguongCanhBaoBatPlaceholderVaRespawn(t *testing.T) {
	st := open(t)
	now := float64(time.Now().UnixMilli())
	st.AddSession(&Session{ID: "s"}, "")
	st.AddSample(&Sample{Session: "s", TS: now, FPSAvg: 60, FPSP1: 60, Placeholder: 1, Respawn2s: 1})
	flush(t, st)
	d, _ := Load(st.DB(), Opts{From: 0, To: time.Now().Unix() + 60})
	md := d.Markdown()
	for _, want := range []string{"khung placeholder", "respawn trong 2s"} {
		if !strings.Contains(md, want) {
			t.Errorf("báo cáo thiếu cảnh báo %q", want)
		}
	}
}

// Mục 5 phải KẾT LUẬN được nghẽn ở đâu, không chỉ liệt kê số.
func TestGhepClientServerChiRaNghenODau(t *testing.T) {
	st := open(t)
	ts := time.Now().Unix() / 10 * 10
	st.AddSession(&Session{ID: "s"}, "")
	// Client tụt còn 12 FPS, và CÙNG LÚC server tick p95 20 ms → lỗi ở server.
	st.AddSample(&Sample{Session: "s", TS: float64(ts) * 1000, FPSAvg: 15, FPSP1: 12, PingP95: 40})
	st.AddServerSample(&ServerSample{TS: ts, Zone: 0, TickAvg: 15, TickP95: 20, Humans: 5})
	flush(t, st)
	d, _ := Load(st.DB(), Opts{From: 0, To: time.Now().Unix() + 60})
	md := d.Markdown()
	if !strings.Contains(md, "**server**") {
		t.Errorf("tick 20 ms cùng lúc client tụt 12 FPS phải kết luận nghẽn ở server:\n%s", section(md, "## 5"))
	}
}

func TestDonDuLieuQuaHan(t *testing.T) {
	st := open(t)
	old := time.Now().AddDate(0, 0, -retainDays-5).Unix()
	if _, err := st.DB().Exec(`INSERT INTO samples (session_id,ts,server_ts,fps_avg) VALUES ('old',0,?,60)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`INSERT INTO samples (session_id,ts,server_ts,fps_avg) VALUES ('new',0,?,60)`, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	st.sweep()
	var n int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM samples WHERE session_id='old'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("mẫu quá %d ngày phải bị xoá, còn %d", retainDays, n)
	}
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM samples WHERE session_id='new'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("mẫu mới bị xoá nhầm: còn %d", n)
	}
}

func TestDeviceClassPhanNhom(t *testing.T) {
	cases := []struct {
		mobile    bool
		gpu, want string
	}{
		{true, "Apple A15", "Điện thoại"},
		{false, "ANGLE (Intel, Intel(R) UHD Graphics)", "PC GPU tích hợp"},
		{false, "ANGLE (AMD, AMD Radeon RX 570)", "PC GPU rời"},
		{false, "", "PC (không rõ GPU)"},
		{false, "SwiftShader", "PC GPU tích hợp"},
	}
	for _, c := range cases {
		if got := deviceClass(c.mobile, c.gpu); got != c.want {
			t.Errorf("deviceClass(%v, %q) = %q, mong %q", c.mobile, c.gpu, got, c.want)
		}
	}
}

// section cắt một mục khỏi báo cáo để thông báo lỗi của test đọc được.
func section(md, head string) string {
	i := strings.Index(md, head)
	if i < 0 {
		return md
	}
	rest := md[i:]
	if j := strings.Index(rest[len(head):], "\n## "); j >= 0 {
		return rest[:len(head)+j]
	}
	return rest
}

func TestPercentile(t *testing.T) {
	v := []float64{}
	for i := 1; i <= 100; i++ {
		v = append(v, float64(i))
	}
	if got := pct(v, 0.95); got < 94 || got > 97 {
		t.Errorf("p95 của 1..100 = %v, mong quanh 95", got)
	}
	if got := median(v); got < 49 || got > 52 {
		t.Errorf("trung vị = %v, mong quanh 50", got)
	}
	if got := pct(nil, 0.5); got != 0 {
		t.Errorf("rỗng phải ra 0, nhận %v", got)
	}
}

func TestNextRollupLuonOPhiaTruoc(t *testing.T) {
	// Kiểm chéo bằng nhiều mốc trong ngày: mốc tiếp theo phải luôn SAU thời điểm hiện tại,
	// và phải là 12:00 hoặc 23:30 — không được trả về quá khứ (vòng lặp sẽ quay điên cuồng).
	for _, hm := range [][2]int{{0, 1}, {11, 59}, {12, 0}, {12, 1}, {23, 29}, {23, 31}} {
		now := time.Date(2026, 10, 30, hm[0], hm[1], 0, 0, vnLocation)
		next := nextRollup(now)
		if !next.After(now) {
			t.Errorf("từ %02d:%02d ra %v — không ở phía trước", hm[0], hm[1], next)
		}
		h, m := next.Hour(), next.Minute()
		if !((h == 12 && m == 0) || (h == 23 && m == 30)) {
			t.Errorf("từ %02d:%02d ra %02d:%02d — không phải mốc 12:00 hay 23:30", hm[0], hm[1], h, m)
		}
	}
}

func TestParseSince(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"1h":  time.Hour,
		"24h": 24 * time.Hour,
		"7d":  7 * 24 * time.Hour,
		"90m": 90 * time.Minute,
	} {
		got, err := parseSince(in)
		if err != nil || got != want {
			t.Errorf("parseSince(%q) = %v, %v; mong %v", in, got, err, want)
		}
	}
	if _, err := parseSince("bậy"); err == nil {
		t.Error("chuỗi sai phải báo lỗi")
	}
}

func TestZeroKhacVoiKhongDoDuoc(t *testing.T) {
	// 0 và "không đo được" là hai chuyện khác nhau. Lẫn lộn là cách nhanh nhất để đọc sai
	// bảng: "tick 0.00 ms" đọc ra là server nhanh thần tốc, trong khi sự thật là không có
	// mẫu server nào.
	if got := num(0, 2, false); got != "–" {
		t.Errorf("không có số liệu phải in \"–\", nhận %q", got)
	}
	if got := num(0, 2, true); got != "0.00" {
		t.Errorf("đo được đúng 0 phải in 0.00, nhận %q", got)
	}
	row := S6{HasServer: false, HasClient: true, FPSAvg: 60}.MarkdownRow()
	if !strings.Contains(row, "– / –") {
		t.Errorf("không có mẫu server thì cột tick phải là – / –:\n%s", row)
	}
}

func TestVuotTranMauThiNoiRa(t *testing.T) {
	// Cắt dữ liệu trong im lặng thì báo cáo đọc ra như thể đã bao hết — nguy hiểm hơn hẳn
	// việc không có báo cáo.
	d := &Data{Truncated: true, Opts: Opts{Label: "7d"},
		Sessions: []sessionRow{{ID: "s"}}, Samples: []sampleRow{{Session: "s"}}}
	if md := d.Markdown(); !strings.Contains(md, fmt.Sprintf("Vượt trần %d mẫu", sampleCap)) {
		t.Error("báo cáo bị cắt phải nói rõ là đã cắt")
	}
}
