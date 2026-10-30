package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"kiemxoay/server/internal/netproto"
	"kiemxoay/server/internal/room"
	"kiemxoay/server/internal/telemetry"
)

// Nhận và ghi số liệu người chơi thật (TELEMETRY_SPEC).
//
// Toàn bộ file này chạy NGOÀI vòng tick. Chỗ duy nhất chạm vào goroutine phòng là
// `sampleRooms`, và nó chỉ đọc vài con số rồi thoát ngay.

const (
	// serverSampleEvery: nhịp server tự ghi số liệu của chính mình (§3).
	serverSampleEvery = 10 * time.Second
	// maxEventsPerMin: trần sự kiện mỗi kết nối. Spec §2.3 đặt trần ở client; server kiểm lại
	// vì một tab bị lỗi lặp vô tận sẽ bắn `error` hàng nghìn lần và làm ngập DB.
	maxEventsPerMin = 40
	// maxSamplesPerMsg: client gửi bù tối đa 30 mẫu (§3); cho dư một chút rồi cắt.
	maxSamplesPerMsg = 40
	// maxEndBody: thân POST /telemetry/end — sendBeacon gửi vài trăm byte.
	maxEndBody = 4 << 10
)

// handleTelemetry nhận gói `tm` từ một client đang chơi.
func (s *server) handleTelemetry(c *room.Client, ipHash string, data []byte) {
	if s.telemetry == nil {
		return // build/chạy không bật telemetry: bỏ qua trong im lặng
	}
	var tm netproto.Telemetry
	if netproto.Decode(data, &tm) != nil {
		return
	}
	if tm.Session != nil {
		// guest_hash băm Ở SERVER từ guest_id mà client đã gửi lúc join, không nhận chuỗi
		// băm sẵn từ client: client tự băm thì mỗi bản build lỗi một kiểu và hai phiên của
		// cùng một người sẽ không ghép được với nhau nữa.
		tm.Session.GuestHash = telemetry.Hash(c.GuestID)
		s.telemetry.AddSession(tm.Session, ipHash)
	}
	for i, m := range tm.Samples {
		if i >= maxSamplesPerMsg {
			break
		}
		s.telemetry.AddSample(m)
	}
	for _, e := range tm.Events {
		if !c.AllowEvent() {
			break
		}
		s.telemetry.AddEvent(e)
	}
}

// handleTelemetryEnd nhận tóm tắt phiên lúc đóng tab (§2.4).
//
// Đi qua HTTP chứ không qua WebSocket vì lúc đóng tab WebSocket đã đứt; `sendBeacon` là thứ
// duy nhất trình duyệt còn bảo đảm gửi đi.
func (s *server) handleTelemetryEnd(w http.ResponseWriter, r *http.Request) {
	if s.telemetry == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxEndBody))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var end telemetry.End
	if json.Unmarshal(body, &end) != nil || end.Session == "" {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	s.telemetry.AddEnd(&end)
	// 204: sendBeacon không đọc phần thân trả về, và tab đang đóng rồi.
	w.WriteHeader(http.StatusNoContent)
}

// runServerSampler ghi số liệu của chính server mỗi 10 giây (§3).
func (s *server) runServerSampler(stop <-chan struct{}) {
	t := time.NewTicker(serverSampleEvery)
	defer t.Stop()
	var last cpuSnap
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			cpu, now := cpuPercent(&last)
			rss := rssMB()
			s.sampleRooms(now, cpu, rss)
		}
	}
}

// sampleRooms xin từng phòng một con số của từng khu.
//
// Phải đi qua `Room.Do` chứ không đọc thẳng: số người và số bot nằm trong map do goroutine
// phòng sở hữu, đọc từ ngoài là data race thật (go test -race bắt ngay). Closure chỉ đọc vài
// `len()` rồi đẩy vào hàng đợi telemetry, nên phần thêm vào tick là không đáng kể.
func (s *server) sampleRooms(now int64, cpu, rss float64) {
	for _, rm := range s.mgr.AllRooms() {
		rm := rm
		rm.Do(func() {
			for i, z := range rm.Zones() {
				avg, p95 := z.TickStats()
				humans := z.Humans()
				s.telemetry.AddServerSample(&telemetry.ServerSample{
					TS: now, Zone: i, ZoneName: z.Name,
					TickAvg: avg, TickP95: p95,
					Humans: humans, Bots: z.Count() - humans,
					KbpsOut: rm.ZoneKbpsOut(z, serverSampleEvery.Seconds()),
					CPUPct:  cpu, RSSMB: rss,
					BuildID: s.telemetry.BuildID(),
				})
			}
		})
	}
}

// cpuSnap giữ mốc CPU lần đo trước để tính phần trăm giữa hai lần.
type cpuSnap struct {
	at  time.Time
	cpu time.Duration
}

// cpuPercent trả về %CPU của tiến trình kể từ lần gọi trước, và mốc thời gian hiện tại.
//
// Dùng thời gian CPU của chính tiến trình (/proc/self/stat), không dùng tải toàn máy: trên
// máy dev còn trình duyệt, Vite và vài thứ khác đang chạy, tải toàn máy nói lên chuyện khác.
func cpuPercent(last *cpuSnap) (float64, int64) {
	now := time.Now()
	cur := procCPU()
	var pct float64
	if !last.at.IsZero() {
		if wall := now.Sub(last.at); wall > 0 {
			pct = float64(cur-last.cpu) / float64(wall) * 100
		}
	}
	last.at, last.cpu = now, cur
	return pct, now.Unix()
}

// procCPU đọc tổng thời gian CPU (user + sys) của tiến trình từ /proc/self/stat.
// Không đọc được (không phải Linux) thì trả 0 — báo cáo sẽ hiện 0 thay vì số bịa.
func procCPU() time.Duration {
	b, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	// /proc/self/stat: trường 14 (utime) và 15 (stime), đơn vị clock tick. Tên tiến trình ở
	// trường 2 có thể chứa khoảng trắng và dấu ngoặc, nên cắt từ sau `) ` rồi mới tách.
	s := string(b)
	i := lastIndexByte(s, ')')
	if i < 0 {
		return 0
	}
	fields := splitFields(s[i+1:])
	if len(fields) < 14 {
		return 0
	}
	ticks := atoiSafe(fields[11]) + atoiSafe(fields[12]) // utime, stime (đã lệch 2 vì cắt)
	// Linux đặt cứng USER_HZ = 100 cho giao diện /proc, không phụ thuộc HZ của nhân.
	return time.Duration(ticks) * 10 * time.Millisecond
}

func rssMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / (1 << 20)
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func splitFields(s string) []string {
	out := []string{}
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\n' {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

func atoiSafe(s string) int64 {
	var n int64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return n
		}
		n = n*10 + int64(s[i]-'0')
	}
	return n
}

// startTelemetry mở DB telemetry nếu được bật. Mở hỏng thì LOG rồi chạy tiếp không có
// telemetry — mất số liệu còn hơn không ai vào chơi được.
func startTelemetry(path, buildID string, on bool) *telemetry.Store {
	if !on {
		return nil
	}
	st, err := telemetry.Open(path, buildID)
	if err != nil {
		log.Printf("telemetry: không mở được %s (%v) — chạy tiếp, KHÔNG thu số liệu", path, err)
		return nil
	}
	log.Printf("telemetry: đang thu vào %s (build %s)", path, buildID)
	return st
}
