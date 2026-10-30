package room

import (
	"sort"
	"sync"
)

// tickWindow: số tick giữ lại để tính trung bình và p95. 300 tick ở 30 Hz = đúng 10 giây,
// bằng nhịp lấy mẫu của TELEMETRY_SPEC §3.
const tickWindow = 300

// tickStats giữ thời gian xử lý của những tick gần nhất của MỘT khu.
//
// Vì sao cần nó khi đã có `Zone.StepMs()`: `stepMs` là trung bình trượt, đọc một con số đó
// không bao giờ ra được p95 — mà p95 mới là thứ người chơi cảm thấy (một tick 40 ms giữa
// trăm tick 2 ms là một cú khựng nhìn thấy được, nhưng trung bình trượt nuốt mất nó).
//
// Có khoá riêng vì goroutine ghi telemetry đọc nó trong khi goroutine phòng đang ghi vào.
// Khoá mỗi tick mỗi khu = 150 lần/giây, không đáng kể so với một tick vài ms.
type tickStats struct {
	mu   sync.Mutex
	ring [tickWindow]float64
	n    int // số tick đã ghi (để biết ring đã đầy chưa)
	at   int
}

func (t *tickStats) add(ms float64) {
	t.mu.Lock()
	t.ring[t.at] = ms
	t.at = (t.at + 1) % tickWindow
	if t.n < tickWindow {
		t.n++
	}
	t.mu.Unlock()
}

// snapshot trả về (trung bình, p95) của cửa sổ hiện tại. n == 0 thì trả 0, 0.
func (t *tickStats) snapshot() (avg, p95 float64) {
	t.mu.Lock()
	n := t.n
	buf := make([]float64, n)
	copy(buf, t.ring[:n])
	t.mu.Unlock()
	if n == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range buf {
		sum += v
	}
	sort.Float64s(buf)
	idx := n * 95 / 100
	if idx >= n {
		idx = n - 1
	}
	return sum / float64(n), buf[idx]
}

// TickStats là số liệu tick 10 giây gần nhất của khu (trung bình, p95, tính bằng ms).
// Gọi được từ goroutine bất kỳ.
func (z *Zone) TickStats() (avg, p95 float64) { return z.ticks.snapshot() }
