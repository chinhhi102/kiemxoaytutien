package game

import (
	"math"
	"testing"

	"kiemxoay/server/internal/config"
)

func testProg(t *testing.T) *Progression {
	t.Helper()
	c := config.MustLoadDefault()
	return NewProgression(c.Cultivation, c.Sword.MaxSwords)
}

func TestExpNeededMatchesSpecTable(t *testing.T) {
	p := testProg(t)
	cases := []struct {
		L    int
		want float64
	}{
		{0, 100}, {9, 172}, {10, 200}, {19, 344}, {20, 400}, {29, 688},
		{30, 800}, {39, 1376}, {80, 25600}, {89, 44032},
	}
	for _, c := range cases {
		if got := p.ExpNeeded(c.L); got != c.want {
			t.Errorf("ExpNeeded(%d) = %v, want %v", c.L, got, c.want)
		}
	}
}

// Spec §3.2: mỗi cảnh giới cần gấp đôi EXP của cảnh giới trước.
func TestEachRealmCostsDoubleThePrevious(t *testing.T) {
	p := testProg(t)
	realmTotal := func(r int) float64 {
		sum := 0.0
		for s := 0; s < p.Tiers(); s++ {
			sum += p.ExpNeeded(r*p.Tiers() + s)
		}
		return sum
	}
	for r := 1; r < 9; r++ {
		prev, cur := realmTotal(r-1), realmTotal(r)
		if math.Abs(cur-prev*2) > 2 {
			t.Errorf("cảnh giới %d cần %v, muốn gấp đôi cảnh giới trước (%v → %v)", r, cur, prev, prev*2)
		}
		// Từng tầng cũng phải gấp đôi tầng tương ứng của cảnh giới trước.
		for s := 0; s < p.Tiers(); s++ {
			a, b := p.ExpNeeded((r-1)*p.Tiers()+s), p.ExpNeeded(r*p.Tiers()+s)
			if math.Abs(b-a*2) > 1 {
				t.Errorf("cảnh giới %d tầng %d cần %v, muốn %v", r, s+1, b, a*2)
			}
		}
	}
}

func TestRealmTotalsMatchSpec(t *testing.T) {
	p := testProg(t)
	want := []float64{1360, 2720, 5440, 10880, 21760, 43520, 87040, 174080, 348160}
	for r, w := range want {
		sum := 0.0
		for s := 0; s < 10; s++ {
			sum += p.ExpNeeded(r*10 + s)
		}
		// Spec table rounds each tier; allow ±2 for cumulative rounding.
		if math.Abs(sum-w) > 2 {
			t.Errorf("realm %d total = %v, want ≈%v", r, sum, w)
		}
	}
}

func TestLevelFromExp(t *testing.T) {
	p := testProg(t)
	if L := p.LevelFromExp(0); L != 0 {
		t.Fatalf("0 exp → level %d", L)
	}
	if L := p.LevelFromExp(99.9); L != 0 {
		t.Fatalf("99.9 exp → level %d", L)
	}
	if L := p.LevelFromExp(100); L != 1 {
		t.Fatalf("100 exp → level %d", L)
	}
	if L := p.LevelFromExp(p.Threshold(10)); L != 10 {
		t.Fatalf("threshold(10) → level %d", L)
	}
	if L := p.LevelFromExp(1e9); L != 89 {
		t.Fatalf("huge exp → level %d, want 89", L)
	}
}

func TestScaleRange(t *testing.T) {
	p := testProg(t)
	if s := p.Scale(0); s != 1 {
		t.Errorf("scale(0) = %v", s)
	}
	if s := p.Scale(89); math.Abs(s-5) > 1e-9 {
		t.Errorf("scale(89) = %v, want 5", s)
	}
	if p.Scale(9) <= p.Scale(0) {
		t.Error("Luyện Khí 10 must be bigger than Luyện Khí 1")
	}
	if s := p.Scale(10); math.Abs(s-1.449) > 0.01 {
		t.Errorf("Trúc Cơ 1 scale = %v, want ≈1.45", s)
	}
}

func TestMaxSwords(t *testing.T) {
	p := testProg(t)
	// Spec §3.1: max_swords[r] = 5(r+1)(r+2) — vừa đủ r+1 vòng theo luật §4.1a.
	for r, want := range []int{10, 30, 60, 100, 150, 210, 280, 360, 450, 550, 660, 780, 910, 1050} {
		if got := p.MaxSwords(r); got != want {
			t.Errorf("MaxSwords(%d) = %d, want %d", r, got, want)
		}
	}
}

// Spec v5.8 §5.1: bị chém thì rớt 1 đại cảnh giới và thanh về **MỨC SÀN** của cảnh giới mới
// (tầng 1, 0%) — không còn giữ tầng và % tiến độ như bản cũ.
func TestDropRealmGoesToRealmFloor(t *testing.T) {
	p := testProg(t)
	for _, c := range []struct {
		name string
		exp  float64
		want float64
	}{
		// Trúc Cơ tầng 2, 25% → Luyện Khí tầng 1, 0%.
		{"Trúc Cơ t2 → Luyện Khí t1", p.Threshold(11) + 0.25*p.ExpNeeded(11), 0},
		// Kim Đan tầng 7 → Trúc Cơ tầng 1 0% = 1 360 (con số spec ghi thẳng trong §5.1).
		{"Kim Đan t7 → Trúc Cơ t1", p.Threshold(26) + 0.9*p.ExpNeeded(26), 1360},
	} {
		got := p.DropRealmExp(c.exp)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: ra %.0f, muốn %.0f", c.name, got, c.want)
		}
		if tier := p.Tier(p.LevelFromExp(got)); got > 0 && tier != 0 {
			t.Errorf("%s: rơi vào tầng %d, phải là tầng 1 (chỉ số 0)", c.name, tier+1)
		}
	}
}

// Sàn từng cảnh giới, ghi thẳng trong spec v5.8 §5.1. Khoá lại vì `realm_mult` đổi một chữ
// số là cả bảng này lệch mà không có gì báo.
func TestRealmFloors(t *testing.T) {
	p := testProg(t)
	for r, want := range []float64{0, 1360, 4080, 9520, 20400, 42160, 85680, 172720, 346800} {
		if got := p.Threshold(r * 10); math.Abs(got-want) > 1e-9 {
			t.Errorf("sàn cảnh giới %d = %.0f, spec ghi %.0f", r, got, want)
		}
	}
}
