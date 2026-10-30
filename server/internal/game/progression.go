package game

import (
	"math"
	"sort"

	"kiemxoay/server/internal/config"
)

// Progression holds the precomputed cultivation tables (spec §3).
type Progression struct {
	cfg        config.Cultivation
	maxSwords  []int
	needed     []float64 // needed[L] = EXP to go from level L to L+1
	cumulative []float64 // cumulative[L] = total EXP required to reach level L
}

func NewProgression(c config.Cultivation, maxSwords []int) *Progression {
	levels := len(c.Realms) * c.TiersPerRealm
	p := &Progression{cfg: c, maxSwords: maxSwords}
	p.needed = make([]float64, levels)
	p.cumulative = make([]float64, levels)
	sum := 0.0
	for L := 0; L < levels; L++ {
		r, s := L/c.TiersPerRealm, L%c.TiersPerRealm
		p.cumulative[L] = sum
		p.needed[L] = math.Round(c.ExpBase * math.Pow(c.RealmMult, float64(r)) * (1 + c.TierStep*float64(s)))
		sum += p.needed[L]
	}
	return p
}

func (p *Progression) Levels() int { return len(p.needed) }

func (p *Progression) Tiers() int { return p.cfg.TiersPerRealm }

// ExpNeeded returns exp_needed(r, s) for level index L.
func (p *Progression) ExpNeeded(L int) float64 {
	if L < 0 || L >= len(p.needed) {
		return 0
	}
	return p.needed[L]
}

// Threshold returns the cumulative EXP required to reach level L.
func (p *Progression) Threshold(L int) float64 {
	if L <= 0 {
		return 0
	}
	if L >= len(p.cumulative) {
		return p.cumulative[len(p.cumulative)-1] + p.needed[len(p.needed)-1]
	}
	return p.cumulative[L]
}

// LevelFromExp returns the level index reached with totalExp (capped at the last level).
func (p *Progression) LevelFromExp(totalExp float64) int {
	i := sort.Search(len(p.cumulative), func(i int) bool { return p.cumulative[i] > totalExp })
	if i == 0 {
		return 0
	}
	return i - 1
}

// MaxExp is the EXP at which progression stops counting (top of the final tier).
func (p *Progression) MaxExp() float64 { return p.Threshold(len(p.needed)) }

func (p *Progression) Realm(L int) int { return L / p.cfg.TiersPerRealm }

func (p *Progression) Tier(L int) int { return L % p.cfg.TiersPerRealm }

// MaxSwords returns max_swords[r]; realms past the end of the array add 40 each (spec §3.1).
func (p *Progression) MaxSwords(r int) int {
	if len(p.maxSwords) == 0 {
		return 1
	}
	if r < len(p.maxSwords) {
		return p.maxSwords[max(r, 0)]
	}
	return p.maxSwords[len(p.maxSwords)-1] + 40*(r-len(p.maxSwords)+1)
}

// DropRealmExp is total_exp after falling one whole realm while keeping the tier and the
// progress through it (spec §5.1): Trúc Cơ 2 at 40% → Luyện Khí 2 at 40%.
// DropRealmExp: bị chém thì rớt 1 đại cảnh giới và thanh về **MỨC SÀN** của cảnh giới mới
// (tầng 1, 0%) — spec v5.8 §5.1.
//
// Bản cũ giữ nguyên tầng và % tiến độ (Trúc Cơ tầng 2 → Luyện Khí tầng 2). Luật mới nặng
// hơn hẳn: Trúc Cơ tầng 2 → Luyện Khí **tầng 1, 0%**, Kim Đan tầng 7 → Trúc Cơ tầng 1 0%.
func (p *Progression) DropRealmExp(exp float64) float64 {
	L := p.LevelFromExp(exp)
	r := p.Realm(L)
	if r <= 0 {
		return 0
	}
	return p.Threshold((r - 1) * p.cfg.TiersPerRealm)
}

// Scale returns scale(L) = min + (max-min) * L / (levels-1); Tiên giới keeps growing by
// tien_scale_step per realm (spec §3.4: Chân Tiên 5.2 … Đạo Tổ 6.0).
func (p *Progression) Scale(L int) float64 {
	if r := p.Realm(L); p.IsTien(r) {
		return p.cfg.ScaleMax + p.cfg.TienScaleStep*float64(p.TienIndex(r)+1)
	}
	last := float64(len(p.needed) - 1)
	return p.cfg.ScaleMin + (p.cfg.ScaleMax-p.cfg.ScaleMin)*float64(L)/last
}

func (p *Progression) RealmName(r int) string {
	if r < 0 {
		return ""
	}
	if r < len(p.cfg.Realms) {
		return p.cfg.Realms[r]
	}
	if i := p.TienIndex(r); i < len(p.cfg.TienRealms) {
		return p.cfg.TienRealms[i]
	}
	return ""
}

// ---- Tiên giới (spec §3.4) -------------------------------------------------
//
// Tiên realms sit above Đại Thừa and have no tiers, so they occupy only the first level of
// their 10-level block: Chân Tiên = level 90, Kim Tiên = 100 … That keeps Realm(L) = L/10
// working everywhere; EXP never reaches them because GainExp caps at MaxExp.

// IsTien reports whether realm index r is a Tiên realm.
func (p *Progression) IsTien(r int) bool { return r >= len(p.cfg.Realms) }

// TienIndex is the 0-based position inside tien_realms.
func (p *Progression) TienIndex(r int) int { return r - len(p.cfg.Realms) }

// TienCount is how many Tiên realms exist.
func (p *Progression) TienCount() int { return len(p.cfg.TienRealms) }

// TopMortalLevel is Đại Thừa tầng 10 — where sát mạng starts counting.
func (p *Progression) TopMortalLevel() int { return len(p.needed) - 1 }

// TienLevel is the level index of Tiên realm idx (0-based).
func (p *Progression) TienLevel(idx int) int {
	return (len(p.cfg.Realms) + idx) * p.cfg.TiersPerRealm
}

// Tiên giới tính theo TỔNG Tiên khí cộng dồn (spec v5.7 §3.4):
//
//	1–100 Chân Tiên · 101–300 Kim Tiên · 301–600 Thái Ất · 601–1000 Đại La · 1001–1500 Đạo Tổ
//
// Tiên khí là nguồn sự thật, cảnh giới suy ra từ nó. Bản v5.6 cộng dồn từng chặng và giữ
// cảnh giới trong `Level`, nên có HAI nguồn sự thật phải tự đồng bộ với nhau — bị chém là
// chỗ chúng lệch.

// TienCeil là trần Tiên khí của cảnh giới Tiên idx (100, 300, 600, 1000, 1500).
func (p *Progression) TienCeil(idx int) float64 {
	c := p.cfg.TienKhiCeilings
	if len(c) == 0 {
		return 0
	}
	return c[min(max(idx, 0), len(c)-1)]
}

// TienEntry là MỐC KHỞI ĐIỂM (đáy) của cảnh giới Tiên idx: 1, 101, 301, 601, 1001.
func (p *Progression) TienEntry(idx int) float64 {
	if idx <= 0 {
		return 1
	}
	return p.TienCeil(idx-1) + 1
}

// TienWidth là độ rộng khoảng Tiên khí của cảnh giới idx (100, 200, 300, 400, 500) — mốc
// để tính Thiêu Đốt 8%/giây (spec v5.7 §4.1c).
func (p *Progression) TienWidth(idx int) float64 {
	if idx <= 0 {
		return p.TienCeil(0)
	}
	return p.TienCeil(idx) - p.TienCeil(idx-1)
}

// TienRealmForKhi trả cảnh giới Tiên (0-based) ứng với tổng Tiên khí này; -1 nếu chưa đủ
// một điểm nào, tức chưa vào Tiên giới.
//
// **Viên Tiên khí đầu tiên là vào Chân Tiên ngay** (spec nói rõ) — nên mốc là `>= 1`, không
// phải `> 0` cho vui: 0,5 điểm vẫn chưa phải Chân Tiên.
func (p *Progression) TienRealmForKhi(khi float64) int {
	if khi < 1 {
		return -1
	}
	for i := range p.cfg.TienKhiCeilings {
		if khi <= p.cfg.TienKhiCeilings[i] {
			return i
		}
	}
	return len(p.cfg.TienKhiCeilings) - 1
}

// TienMaxKhi là trần cao nhất (1 500) — Đạo Tổ nạp tới đây thì dư mới đổi Uy Danh.
func (p *Progression) TienMaxKhi() float64 { return p.TienCeil(p.TienCount() - 1) }
