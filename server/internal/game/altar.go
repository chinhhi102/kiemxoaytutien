package game

import (
	"math"

	"kiemxoay/server/internal/geom"
	"kiemxoay/server/internal/world"
)

// Đài Tiên Khí (spec §8.5): five fountains that spray Tiên khí orbs. Four small ones sit by
// the sect yards near the corners; the Thiên Tiên Đài in the centre drops one huge orb every
// 150s and announces it to the whole room 10s early. The orbs are the only source of Tiên khí
// (§3.4), so everyone — not just Tiên giới players — has a reason to fight over them.

// ItemTienKhi is the item kind of a Tiên khí orb; Value is how many points it carries.
const ItemTienKhi = "tien_khi"

// altarState tracks one altar's timers.
type altarState struct {
	a         world.Altar
	nextSpray float64 // đợt phun thường tiếp theo
	nextBurst float64 // đợt phun lớn tiếp theo
	warned    bool    // đã báo trước đợt lớn chưa
}

// resetAltars rebuilds the altar timers from the map (called when a game starts).
func (g *Game) resetAltars() {
	ac := g.Cfg.Altars
	g.altars = g.altars[:0]
	for _, a := range g.Map.Altars {
		st := altarState{a: a}
		if a.Kind == "center" {
			st.nextBurst = g.Time + ac.Center.EverySecs
		} else {
			st.nextSpray = g.Time + ac.Small.EverySecs
			st.nextBurst = g.Time + ac.Small.BurstEverySecs
		}
		g.altars = append(g.altars, st)
	}
}

// updateAltars runs the spray timers (spec §8.5).
func (g *Game) updateAltars() {
	ac := g.Cfg.Altars
	for i := range g.altars {
		st := &g.altars[i]
		kind := ac.Small
		if st.a.Kind == "center" {
			kind = ac.Center
		}
		// Báo trước đợt lớn: đài nhỏ rung 3s, Thiên Tiên Đài loan toàn phòng 10s.
		if !st.warned && st.nextBurst > 0 && g.Time >= st.nextBurst-kind.WarnSecs {
			st.warned = true
			g.emit(Event{Kind: "AltarWarn", A: uint32(st.a.ID), S: st.a.Kind,
				N: int(kind.WarnSecs), Global: st.a.Kind == "center",
				X: float32(st.a.X), Y: float32(st.a.Y)})
		}
		if st.a.Kind != "center" && g.Time >= st.nextSpray {
			st.nextSpray = g.Time + kind.EverySecs
			g.sprayTienKhi(st.a, g.rndInt(kind.Orbs), kind.OrbValue)
		}
		if st.nextBurst > 0 && g.Time >= st.nextBurst {
			st.warned = false
			if st.a.Kind == "center" {
				st.nextBurst = g.Time + kind.EverySecs
				g.sprayTienKhi(st.a, 1, kind.OrbValue)
			} else {
				st.nextBurst = g.Time + kind.BurstEverySecs
				g.sprayTienKhi(st.a, kind.BurstSmall, kind.OrbValue)
				g.sprayTienKhi(st.a, kind.BurstMedium, kind.MediumValue)
			}
			g.emit(Event{Kind: "AltarBurst", A: uint32(st.a.ID), S: st.a.Kind,
				Global: st.a.Kind == "center", X: float32(st.a.X), Y: float32(st.a.Y)})
		}
	}
}

// rndInt picks from a [min, max] pair in config; a single value means a fixed count.
func (g *Game) rndInt(r []int) int {
	switch len(r) {
	case 0:
		return 1
	case 1:
		return r[0]
	default:
		lo, hi := r[0], r[1]
		if hi <= lo {
			return lo
		}
		return lo + g.Rng.Intn(hi-lo+1)
	}
}

// sprayTienKhi throws n orbs out of the altar like a fountain: random angle, radius
// spray_r[0]..spray_r[1] (spec §8.5).
func (g *Game) sprayTienKhi(a world.Altar, n int, value float64) {
	ac := g.Cfg.Altars
	lo, hi := 300.0, 700.0
	if len(ac.SprayR) >= 2 {
		lo, hi = ac.SprayR[0], ac.SprayR[1]
	}
	for i := 0; i < n; i++ {
		ang := g.Rng.Float64() * 2 * math.Pi
		d := lo + g.Rng.Float64()*(hi-lo)
		g.addTienKhiOrb(Vec{X: a.X, Y: a.Y}.Add(geom.FromAngle(ang).Mul(d)), value)
	}
}

// scatterTienKhi drops robbed Tiên khí around a victim (spec §3.4). It splits the total into
// orbs no bigger than the medium orb so a single pickup cannot take it all back.
// Chia viên **10 → 2 → 1** và văng ra bán kính 200–500 (spec v5.9 §3.4).
//
// Bán kính là chỗ đã hỏng và khó thấy: bản cũ dùng `items.scatter_r` = 90, trong khi người
// Tiên giới có scale 5–6 nên thân rộng hơn 150 và cao hơn 400 pixel — mà vật phẩm vẽ DƯỚI
// nhân vật (`zIndex = y − 1000`). Kết quả: viên rơi đúng, gửi đúng, nhưng nằm khuất hẳn
// dưới cái xác to đùng, nên chủ dự án chém xong "không thấy Tiên khí rơi ra".
//
// Test khoá lại phần SỐ LIỆU (tổng điểm rơi); phần nhìn thấy được thì bán kính lo.
func (g *Game) scatterTienKhi(at Vec, total float64) {
	c := g.Cfg.Cultivation
	lo := math.Max(1, c.TienDropScatterMin)
	hi := math.Max(lo, c.TienDropScatterMax)
	left := math.Floor(total)
	for _, unit := range []float64{10, 2, 1} {
		for left >= unit {
			ang := g.Rng.Float64() * 2 * math.Pi
			d := lo + g.Rng.Float64()*(hi-lo)
			g.addTienKhiOrb(at.Add(geom.FromAngle(ang).Mul(d)), unit)
			left -= unit
		}
	}
}

// addTienKhiOrb places one orb, clamped inside the map, with a lifetime (§8.5: 60s then it
// fades; Tiên khí never respawns somewhere else).
func (g *Game) addTienKhiOrb(at Vec, value float64) {
	if value <= 0 {
		return
	}
	at = g.clampToMap(at)
	it := g.addItem(ItemTienKhi, value, at)
	it.ExpireAt = g.Time + math.Max(1, g.Cfg.Altars.OrbLifeSec)
}

// clampToMap keeps a point inside the playable area so orbs never land behind the border wall.
func (g *Game) clampToMap(p Vec) Vec {
	lo, hi := g.PlayBounds(g.Cfg.Items.ItemR + 20)
	return Vec{X: geom.Clamp(p.X, lo, hi), Y: geom.Clamp(p.Y, lo, hi)}
}

// expireItems removes Tiên khí orbs whose lifetime ran out (spec §8.5).
func (g *Game) expireItems() {
	for _, it := range g.Items {
		if it.ExpireAt > 0 && g.Time >= it.ExpireAt {
			g.removeItem(it)
			g.emit(Event{Kind: "ItemExpired", A: it.ID, X: float32(it.Pos.X), Y: float32(it.Pos.Y)})
		}
	}
}

// AltarMarkers is what the minimap needs: where the altars are and which are about to burst
// (spec §8.5, §12.1).
type AltarMarker struct {
	X, Y    float64
	Kind    string
	Warning bool
}

func (g *Game) AltarMarkers() []AltarMarker {
	out := make([]AltarMarker, 0, len(g.altars))
	for _, st := range g.altars {
		out = append(out, AltarMarker{X: st.a.X, Y: st.a.Y, Kind: st.a.Kind, Warning: st.warned})
	}
	return out
}
