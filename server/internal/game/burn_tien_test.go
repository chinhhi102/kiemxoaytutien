package game

import (
	"math"
	"testing"
)

// Tiên giới theo spec v5.7/v5.8/v5.9 (§3.4, §4.1c, §5.1).
//
// Con số trong các test này lấy thẳng từ checklist §17 của spec, không tự đặt — có thế thì
// test mới là bản dịch của spec chứ không phải bản chụp của code.

// tienPlayer dựng một người đã vào Tiên giới với đúng ngần này Tiên khí.
func tienPlayer(t *testing.T, g *Game, khi float64) *Player {
	t.Helper()
	p := g.AddPlayer(SpawnOptions{Name: "T", Char: g.Cfg.Characters[0]})
	p.InvulnUntil = 0
	// Đầy thanh Đại Thừa tầng 10 mới nhặt được Tiên khí (spec v5.7).
	g.ForceState(p, g.Prog.TopMortalLevel(), 10, p.Pos)
	p.TotalExp = g.Prog.MaxExp()
	if !g.BanksTienKhi(p) {
		t.Fatal("đầy thanh Đại Thừa 10 mà vẫn chưa sang đường tu Tiên khí")
	}
	g.GainTienKhi(p, khi)
	return p
}

// Đại Thừa tầng 10 CHƯA đầy thanh thì vẫn tu Linh khí, không nhặt được Tiên khí.
func TestTienEntryNeedsFullBar(t *testing.T) {
	g := rulesGame(t)
	p := g.AddPlayer(SpawnOptions{Name: "A", Char: g.Cfg.Characters[0]})
	g.ForceState(p, g.Prog.TopMortalLevel(), 10, p.Pos)
	p.TotalExp = g.Prog.MaxExp() - 1

	if g.BanksTienKhi(p) {
		t.Error("chưa đầy thanh mà đã sang đường Tiên khí")
	}
	if !g.CanAbsorbExp(p) {
		t.Error("chưa đầy thanh thì phải còn nhặt được Linh khí")
	}
	if g.CanAbsorbTienKhi(p) {
		t.Error("chưa đầy thanh mà đã nhặt được Tiên khí")
	}

	g.GainExp(p, 1, false)
	if !g.BanksTienKhi(p) {
		t.Error("đầy thanh rồi mà vẫn chưa sang đường Tiên khí")
	}
}

// Cảnh giới Tiên suy ra thẳng từ tổng Tiên khí; viên đầu tiên là vào Chân Tiên ngay.
func TestTienRealmFromTotalKhi(t *testing.T) {
	g := rulesGame(t)
	for _, c := range []struct {
		khi  float64
		want int
		name string
	}{
		{0, -1, "chưa có Tiên khí"},
		{1, 0, "1 → Chân Tiên"},
		{100, 0, "100 → vẫn Chân Tiên"},
		{101, 1, "101 → Kim Tiên"},
		{300, 1, "300 → vẫn Kim Tiên"},
		{301, 2, "301 → Thái Ất"},
		{600, 2, "600 → vẫn Thái Ất"},
		{601, 3, "601 → Đại La"},
		{1000, 3, "1000 → vẫn Đại La"},
		{1001, 4, "1001 → Đạo Tổ"},
		{1500, 4, "1500 → vẫn Đạo Tổ"},
	} {
		if got := g.Prog.TienRealmForKhi(c.khi); got != c.want {
			t.Errorf("%s: ra %d, muốn %d", c.name, got, c.want)
		}
	}
}

// Nạp Tiên khí thì cảnh giới (và `Level`) đi theo.
func TestTienPromotesByKhi(t *testing.T) {
	g := rulesGame(t)
	p := tienPlayer(t, g, 1)
	if got := g.TienRealmOf(p); got != 0 {
		t.Fatalf("1 Tiên khí phải là Chân Tiên, ra idx %d", got)
	}
	if want := g.Prog.TienLevel(0); p.Level != want {
		t.Errorf("level %d, muốn %d — `Level` phải đi theo Tiên khí", p.Level, want)
	}
	g.GainTienKhi(p, 100) // 101 → Kim Tiên
	if got := g.TienRealmOf(p); got != 1 {
		t.Errorf("101 Tiên khí phải là Kim Tiên, ra idx %d", got)
	}
	if want := g.Prog.TienLevel(1); p.Level != want {
		t.Errorf("level %d, muốn %d", p.Level, want)
	}
}

// Đạo Tổ: nạp thanh bình thường tới 1 500; CHỈ khi đang đúng 1 500 thì dư mới đổi Uy Danh.
func TestDaoToFillsThenConverts(t *testing.T) {
	g := rulesGame(t)
	p := tienPlayer(t, g, 1001) // vừa vào Đạo Tổ
	if got := g.TienRealmOf(p); got != 4 {
		t.Fatalf("1001 Tiên khí phải là Đạo Tổ, ra idx %d", got)
	}
	scoreBefore := p.Score

	g.GainTienKhi(p, 100) // 1101, chưa đầy
	if p.TienKhi != 1101 {
		t.Errorf("Tiên khí %.0f, muốn 1101 — Đạo Tổ phải nạp được thanh", p.TienKhi)
	}
	if p.Score != scoreBefore {
		t.Error("chưa đầy 1 500 mà đã cộng Uy Danh")
	}

	g.GainTienKhi(p, 399) // đúng 1500
	if p.TienKhi != 1500 || p.Score != scoreBefore {
		t.Errorf("Tiên khí %.0f / Uy Danh +%.1f — đúng 1 500 thì chưa được cộng gì",
			p.TienKhi, p.Score-scoreBefore)
	}

	g.GainTienKhi(p, 500) // dư 500 → +50 Uy Danh (10:1)
	if p.TienKhi != 1500 {
		t.Errorf("Tiên khí %.0f — thanh không được vượt 1 500", p.TienKhi)
	}
	if got := p.Score - scoreBefore; math.Abs(got-50) > 1e-6 {
		t.Errorf("dư 500 Tiên khí được %.1f Uy Danh, muốn 50 (10:1)", got)
	}
}

// Đạo Tổ Thiêu Đốt được, và đốt xong phải nạp lại đầy mới đổi Uy Danh tiếp (Q26 đã xong).
func TestDaoToCanBurn(t *testing.T) {
	g := rulesGame(t)
	p := tienPlayer(t, g, 1500)
	p.Moving, p.Burning = true, true
	if !g.Burning(p) {
		t.Fatal("Đạo Tổ đầy thanh mà không Thiêu Đốt được — đây chính là Q26")
	}
	g.burnExp(p, 1.0)
	if want := 1500 - 40.0; math.Abs(p.TienKhi-want) > 1e-6 {
		t.Errorf("đốt 1 giây còn %.1f Tiên khí, muốn %.0f (8%% × 500 = 40/giây)", p.TienKhi, want)
	}
	if g.TienRealmOf(p) != 4 {
		t.Error("đốt một giây mà rớt khỏi Đạo Tổ")
	}
}

// Tốc độ đốt = 8% ĐỘ RỘNG KHOẢNG của cảnh giới: 8 · 16 · 24 · 32 · 40 (spec v5.7 §4.1c).
func TestTienBurnRatePerRealm(t *testing.T) {
	g := rulesGame(t)
	for _, c := range []struct {
		khi  float64
		want float64
		name string
	}{
		{50, 8, "Chân Tiên"},
		{200, 16, "Kim Tiên"},
		{400, 24, "Thái Ất"},
		{800, 32, "Đại La"},
		{1200, 40, "Đạo Tổ"},
	} {
		p := tienPlayer(t, g, c.khi)
		if got := g.tienBurnPerSec(p); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("%s đốt %.1f/giây, muốn %.0f", c.name, got, c.want)
		}
		g.RemovePlayer(p.ID)
	}
}

// Đốt không bao giờ xuống dưới đáy cảnh giới đang đứng.
func TestTienBurnStopsAtRealmFloor(t *testing.T) {
	g := rulesGame(t)
	p := tienPlayer(t, g, 350) // Thái Ất, đáy 301
	p.Moving, p.Burning = true, true
	for i := 0; i < 50; i++ {
		g.burnExp(p, 1.0)
	}
	if p.TienKhi < 301-1e-9 {
		t.Errorf("Tiên khí tụt xuống %.1f, đáy Thái Ất là 301", p.TienKhi)
	}
	if g.TienRealmOf(p) != 2 {
		t.Error("Thiêu Đốt làm rớt cảnh giới — spec cấm")
	}
	if g.Burning(p) {
		t.Error("chạm đáy rồi mà vẫn còn đốt")
	}
}

// Checklist §17: Thái Ất 450 bị chém → Kim Tiên, Tiên khí 101; người chém (Tiên giới)
// +174; rơi 104 điểm viên Tiên khí.
func TestCutTienGioiThaiAt450(t *testing.T) {
	g := rulesGame(t)
	victim := tienPlayer(t, g, 450)
	att := tienPlayer(t, g, 1)
	attBefore := att.TienKhi

	g.robTienKhi(att, victim)

	if victim.TienKhi != 101 {
		t.Errorf("nạn nhân còn %.0f Tiên khí, muốn 101 (đáy Kim Tiên)", victim.TienKhi)
	}
	if got := g.TienRealmOf(victim); got != 1 {
		t.Errorf("nạn nhân ở idx %d, muốn 1 (Kim Tiên)", got)
	}
	// loss = 450 − 101 = 349; cướp 50% = 174.5
	if got := att.TienKhi - attBefore; math.Abs(got-174.5) > 0.6 {
		t.Errorf("người chém nhận %.1f Tiên khí, muốn ≈174", got)
	}
	if got := totalTienOrbs(g); got != 104 {
		t.Errorf("rơi %d điểm Tiên khí, muốn 104 (30%% của 349)", got)
	}
}

// Checklist §17: Chân Tiên bị chém → sàn Đại Thừa (tầng 1, 0%), Tiên khí 0.
func TestCutChanTienFallsToDaiThuaFloor(t *testing.T) {
	g := rulesGame(t)
	victim := tienPlayer(t, g, 80)
	att := g.AddPlayer(SpawnOptions{Name: "A", Char: g.Cfg.Characters[1]})

	g.robTienKhi(att, victim)

	if victim.TienKhi != 0 {
		t.Errorf("Tiên khí còn %.0f, muốn 0", victim.TienKhi)
	}
	wantLevel := (len(g.Cfg.Cultivation.Realms) - 1) * g.Prog.Tiers()
	if victim.Level != wantLevel {
		t.Errorf("level %d, muốn %d (Đại Thừa tầng 1)", victim.Level, wantLevel)
	}
	if want := g.Prog.Threshold(wantLevel); math.Abs(victim.TotalExp-want) > 1e-6 {
		t.Errorf("total_exp %.0f, muốn %.0f (sàn Đại Thừa)", victim.TotalExp, want)
	}
	// Checklist: Chân Tiên 80 Tiên khí bị chém → rơi 24.
	if got := totalTienOrbs(g); got != 24 {
		t.Errorf("rơi %d điểm Tiên khí, muốn 24 (30%% của 80)", got)
	}
}

// v5.9: người chém là PHÀM (không nhặt được Tiên khí) thì vẫn phải rơi đủ 30%.
func TestMortalAttackerStillDropsTienKhi(t *testing.T) {
	g := rulesGame(t)
	victim := tienPlayer(t, g, 80)
	att := g.AddPlayer(SpawnOptions{Name: "A", Char: g.Cfg.Characters[1]}) // Luyện Khí

	g.robTienKhi(att, victim)

	if att.TienKhi != 0 {
		t.Errorf("người phàm nhận %.0f Tiên khí — spec nói không nhận được", att.TienKhi)
	}
	if got := totalTienOrbs(g); got != 24 {
		t.Errorf("rơi %d điểm Tiên khí, muốn 24 — phần rơi ra đất không phụ thuộc người chém", got)
	}
}

// Viên rơi ra phải văng ĐỦ XA khỏi xác: người Tiên giới to gấp 5–6 lần và vật phẩm vẽ dưới
// nhân vật, nên viên rơi sát chân là nằm khuất hẳn — đó là lỗi "chém mà không thấy rơi".
func TestTienOrbsScatterFarEnough(t *testing.T) {
	g := rulesGame(t)
	victim := tienPlayer(t, g, 450)
	victim.Pos = Vec{X: 5000, Y: 5000}
	att := tienPlayer(t, g, 1)

	g.robTienKhi(att, victim)

	minD := g.Cfg.Cultivation.TienDropScatterMin
	n := 0
	for _, it := range g.Items {
		if it.Kind != ItemTienKhi {
			continue
		}
		n++
		if d := it.Pos.Dist(victim.Pos); d < minD-1e-6 {
			t.Errorf("viên rơi cách xác %.0f, tối thiểu phải %.0f", d, minD)
		}
	}
	if n == 0 {
		t.Fatal("không có viên nào rơi")
	}
}

func totalTienOrbs(g *Game) int {
	total := 0.0
	for _, it := range g.Items {
		if it.Kind == ItemTienKhi {
			total += it.Value
		}
	}
	return int(total)
}

// Người phàm vẫn đốt EXP như cũ.
func TestMortalStillBurnsExp(t *testing.T) {
	g := rulesGame(t)
	p := g.AddPlayer(SpawnOptions{Name: "A", Char: g.Cfg.Characters[0]})
	g.GainExp(p, g.Prog.ExpNeeded(p.Level)*0.9, false)
	before := p.TotalExp

	p.Moving, p.Burning = true, true
	if !g.Burning(p) {
		t.Fatal("người phàm còn EXP tầng mà không đốt được")
	}
	g.burnExp(p, 1.0)
	if p.TotalExp >= before {
		t.Error("đốt mà EXP không giảm")
	}
}
