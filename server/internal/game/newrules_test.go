package game

import (
	"math"
	"testing"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/world"
)

func rulesGame(t *testing.T) *Game {
	t.Helper()
	cfg := config.MustLoadDefault()
	cfg.Room.MonsterCount, cfg.Items.CountPerRoom = 0, 0
	return New(cfg, world.EmptyMap(cfg.Room.MapSize), 1)
}

// Lên Đạo Tổ rồi thì không còn cảnh giới để lên; Tiên khí nhặt thêm phải đổi thành Uy Danh
// chứ không bỏ phí, và thanh Tiên khí giữ nguyên ở mốc đỉnh.
func TestDaoToSurplusBecomesScore(t *testing.T) {
	g := rulesGame(t)
	top := g.Prog.TienCount() - 1
	// v5.7: Đạo Tổ phải nạp ĐẦY trần (1 500) rồi dư mới đổi Uy Danh — đứng ở đáy Đạo Tổ
	// (1 001) thì Tiên khí nhặt thêm vẫn dùng để nạp thanh.
	cap := g.Prog.TienMaxKhi()
	p := &Player{Level: g.Prog.TienLevel(top), TienKhi: cap, TotalExp: g.Prog.MaxExp()}
	g.refreshDerived(p)

	// Tỉ lệ chủ dự án chốt: 10 Tiên khí ăn 1 Uy Danh. Khoá lại ở đây vì đây là con số
	// cân bằng, đổi nhầm một chữ số là người chơi đỉnh bảng giàu lên gấp mười.
	if r := g.Cfg.Cultivation.TienOverflowScore; math.Abs(r-0.1) > 1e-9 {
		t.Fatalf("tien_overflow_score = %v, phải là 0,1 (10 Tiên khí = 1 Uy Danh)", r)
	}
	before := p.Score
	g.GainTienKhi(p, 250)

	want := 250 * g.Cfg.Cultivation.TienOverflowScore
	if got := p.Score - before; math.Abs(got-want) > 1e-9 {
		t.Fatalf("Uy Danh cộng thêm %.1f, muốn %.1f", got, want)
	}
	if math.Abs(p.TienKhi-cap) > 1e-9 {
		t.Fatalf("Tiên khí phình lên %.1f, phải giữ ở mốc Đạo Tổ %.1f", p.TienKhi, cap)
	}
	if p.ScoreBy["tien_overflow"] <= 0 {
		t.Fatal("không ghi nguồn điểm 'tien_overflow' — bảng tổng kết sẽ không giải thích được điểm ở đâu ra")
	}
}

// Chưa tới Đạo Tổ thì Tiên khí vẫn phải dùng để lên cảnh giới, KHÔNG được đổi sang Uy Danh.
func TestBelowDaoToStillLevelsUp(t *testing.T) {
	g := rulesGame(t)
	p := &Player{Level: g.Prog.TienLevel(0), TotalExp: g.Prog.MaxExp()}
	g.refreshDerived(p)

	// v5.7: lên cảnh giới khi VƯỢT TRẦN cảnh giới hiện tại (100 → 101 là Kim Tiên).
	need := g.Prog.TienCeil(0) + 1
	g.GainTienKhi(p, need)

	if g.TienRealmOf(p) != 1 {
		t.Fatalf("nhận đủ %.0f Tiên khí mà vẫn ở cảnh giới Tiên %d", need, g.TienRealmOf(p))
	}
	// Lên cảnh giới vốn đã được thưởng Uy Danh (§11.1), nên không so tổng điểm mà soi
	// đúng nguồn: chưa tới Đạo Tổ thì không được có đồng nào từ 'tien_overflow'.
	if p.ScoreBy["tien_overflow"] != 0 {
		t.Fatalf("chưa tới Đạo Tổ mà đã đổi Tiên khí dư sang Uy Danh: %.1f", p.ScoreBy["tien_overflow"])
	}
	if math.Abs(p.TienKhi-need) > 1e-9 {
		t.Fatalf("Tiên khí còn %.1f, phải giữ nguyên %.1f", p.TienKhi, need)
	}
}

// Cổng dịch chuyển đưa người chơi tới một cổng NGẪU NHIÊN khác cùng mạng, không phải cổng
// cố định. Thả cùng một người vào cùng một cổng nhiều lần thì phải ra ở nhiều chỗ khác nhau.
func TestGateExitIsRandom(t *testing.T) {
	g := rulesGame(t)
	g.Map.Gates = []world.Gate{
		{ID: 1, Network: "a", X: 1000, Y: 1000},
		{ID: 2, Network: "a", X: 5000, Y: 1000},
		{ID: 3, Network: "a", X: 1000, Y: 5000},
		{ID: 4, Network: "a", X: 5000, Y: 5000},
	}
	seen := map[int]int{}
	for i := 0; i < 200; i++ {
		d, ok := g.gateExit(g.Map.Gates[0])
		if !ok {
			t.Fatal("không tìm được cổng ra")
		}
		if d.GateID == 1 {
			t.Fatal("cổng ra trùng cổng vào — người chơi sẽ bị kẹt dịch chuyển tại chỗ")
		}
		seen[d.GateID]++
	}
	if len(seen) != 3 {
		t.Fatalf("chỉ ra ở %d cổng (%v), phải rải đều cả 3 cổng còn lại", len(seen), seen)
	}
	for id, n := range seen {
		if n < 30 { // 200/3 ≈ 67; dưới 30 là lệch hẳn chứ không phải may rủi
			t.Fatalf("cổng %d chỉ ra %d/200 lần — không ngẫu nhiên đều", id, n)
		}
	}
}

// Tắt random_exit thì quay về luật cũ: cổng đôi cố định.
func TestGateExitFixedWhenDisabled(t *testing.T) {
	g := rulesGame(t)
	g.Cfg.Gates.RandomExit = false
	g.Map.Gates = []world.Gate{
		{ID: 1, Network: "a", X: 1000, Y: 1000},
		{ID: 2, Network: "a", X: 5000, Y: 1000},
		{ID: 3, Network: "a", X: 1000, Y: 5000},
		{ID: 4, Network: "a", X: 5000, Y: 5000},
	}
	first, _ := g.gateExit(g.Map.Gates[0])
	for i := 0; i < 20; i++ {
		if d, _ := g.gateExit(g.Map.Gates[0]); d.GateID != first.GateID {
			t.Fatalf("tắt random_exit rồi mà cổng ra vẫn đổi: %d rồi %d", first.GateID, d.GateID)
		}
	}
}
