package game

import (
	"math"
	"testing"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/world"
)

const dt = 1.0 / 30

// quietGame returns a game on an empty map with no random spawns.
func quietGame(t *testing.T, m *world.Map) *Game {
	t.Helper()
	cfg := config.MustLoadDefault()
	cfg.Room.MonsterCount = 0
	cfg.Items.CountPerRoom = 0
	cfg.Items.ChestEverySecs = 1e9
	cfg.ExpOrbs.Cell = 0 // no ambient orbs: tests count the orbs they drop
	if m == nil {
		m = world.EmptyMap(cfg.Room.MapSize)
	}
	return New(cfg, m, 1)
}

// place puts a player at pos with ring phase angle (sword 0's hilt points there), not
// sweeping, and clears spawn invulnerability.
func place(g *Game, p *Player, pos Vec, angle float64) {
	p.Pos, p.Ring.Angle, p.InvulnUntil = pos, angle, 0
	p.Ring.SweepLo, p.Ring.Sweep = angle, 0
	g.refreshDerived(p)
}

// bladeMid is the middle of p's sword 0.
func bladeMid(p *Player) Vec {
	h, t := SwordSegment(p.Pos, p.Ring, p.Geom, 0)
	return h.Add(t).Mul(0.5)
}

// touchSwords moves b so the middle of its sword 0 (phase angle) lies on a's sword 0.
func touchSwords(g *Game, a, b *Player, angle float64) {
	place(g, b, Vec{}, angle)
	place(g, b, bladeMid(a).Sub(bladeMid(b)), angle)
}

func countItems(g *Game, kind string) (n int, value float64) {
	for _, it := range g.Items {
		if it.Kind == kind {
			n++
			value += it.Value
		}
	}
	return n, value
}

func hasEvent(evs []Event, kind string) bool {
	for _, e := range evs {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// Spec v5.8 §5.1: bị chém rớt 1 đại cảnh giới và thanh về SÀN cảnh giới mới (tầng 1, 0%).
// Trúc Cơ tầng 2 → Luyện Khí tầng 1, 0% — không còn giữ tầng như bản cũ.
func TestBodyHitDropsToRealmFloor(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Level: 0, Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 11, Swords: 1})
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	b.TotalExp += 0.4 * g.Prog.ExpNeeded(11)
	place(g, b, bladeMid(a), math.Pi/2)
	before := b.TotalExp
	g.resolveSwordBody()
	want := g.Prog.Threshold(0) // sàn Luyện Khí = 0
	if b.Level != 0 || math.Abs(b.TotalExp-want) > 1e-9 {
		t.Fatalf("B level %d exp %v, muốn Luyện Khí tầng 1 (level 0) exp %v", b.Level, b.TotalExp, want)
	}
	loss := before - want
	if math.Abs(a.TotalExp-loss*0.5) > 1e-9 {
		t.Fatalf("A exp = %v, want %v", a.TotalExp, loss*0.5)
	}
	if !b.Invulnerable(g.Time) {
		t.Fatal("B should be invulnerable after a hit")
	}
	if !hasEvent(g.DrainEvents(), "RealmDrop") {
		t.Fatal("Trúc Cơ → Luyện Khí must emit RealmDrop")
	}
	if _, orbs := countItems(g, "orb"); math.Abs(orbs-math.Round(loss*0.3)) > 1e-9 {
		t.Fatalf("dropped orb EXP = %v, want %v", orbs, math.Round(loss*0.3))
	}
}

// Spec §5.1: swords over the new realm's max (60 at Luyện Khí) fall as bundles of ≤ 10.
func TestRealmDropSpillsExtraSwordsAsBundles(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Level: 0, Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 10, Swords: 25}) // Trúc Cơ: tối đa 30
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	place(g, b, bladeMid(a), math.Pi/16)
	a.InvulnUntil = 1e9 // B's larger ring would otherwise hit A first
	// Gọi thẳng §5.1: ở đây đang kiểm luật rớt cảnh giới, không kiểm cổng lá chắn §5.2
	// (25 kiếm = 2 vòng nên resolveSwordBody sẽ bỏ qua).
	g.applyBodyHit(a, b)
	if b.Ring.Count != 10 {
		t.Fatalf("B swords = %d, want 10 (Luyện Khí max)", b.Ring.Count)
	}
	n, total := countItems(g, "bundle")
	if n != 2 || total != 15 {
		t.Fatalf("bundles = %d holding %v swords, want 2 holding 15", n, total)
	}
}

// Spec §5.1: Luyện Khí at any tier dies when hit.
func TestLuyenKhiDiesAtAnyTier(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Level: 5, Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 7, Swords: 1})
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	place(g, b, bladeMid(a), math.Pi/2)
	g.resolveSwordBody()
	if !b.Dead || b.KilledBy != a.ID || a.Kills != 1 {
		t.Fatalf("B dead=%v killedBy=%d A kills=%d", b.Dead, b.KilledBy, a.Kills)
	}
	if !hasEvent(g.DrainEvents(), "Death") {
		t.Fatal("missing Death event")
	}
}

func TestShieldBlocksOneHit(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 15, Swords: 1})
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	place(g, b, bladeMid(a), math.Pi/2)
	g.ApplyBuff(b, "shield")
	g.resolveSwordBody()
	if b.Level != 15 || b.HasBuff(BuffShield, g.Time) || !b.Invulnerable(g.Time) {
		t.Fatalf("shield: level %d shield %v invuln %v", b.Level, b.HasBuff(BuffShield, g.Time), b.Invulnerable(g.Time))
	}
}

func TestOneHitPerVictimPerTick(t *testing.T) {
	g := quietGame(t, nil)
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 20, Swords: 1})
	place(g, b, Vec{X: 5000, Y: 5000}, math.Pi/2)
	for _, x := range []float64{-1, 1} {
		a := g.AddPlayer(SpawnOptions{Name: "A", Swords: 1})
		ang := math.Pi // at the right of B, point left
		if x > 0 {
			ang = 0
		}
		place(g, a, Vec{}, ang)
		place(g, a, b.Pos.Sub(bladeMid(a)), ang)
	}
	g.resolveSwordBody()
	if b.Level != 10 {
		t.Fatalf("B level = %d, want 10 (only one hit per tick)", b.Level)
	}
}

// Spec §5.2: same realm → each loses clash_unit swords, both flip and are knocked apart.
func TestClashFlipsBothRings(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Swords: 8})
	b := g.AddPlayer(SpawnOptions{Name: "B", Swords: 8})
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	touchSwords(g, a, b, math.Pi)
	g.resolveSwordSword()
	if a.Ring.Dir != -1 || b.Ring.Dir != -1 {
		t.Fatalf("dirs = %v, %v; want both flipped", a.Ring.Dir, b.Ring.Dir)
	}
	if a.Ring.Count != 7 || b.Ring.Count != 7 {
		t.Fatalf("cùng cảnh giới phải mất đúng clash_unit = 1 mỗi bên: %d, %d", a.Ring.Count, b.Ring.Count)
	}
	// Spec §5.2: không còn đẩy văng, hai người đứng nguyên mài nhau.
	if a.KnockUntil > g.Time || b.KnockUntil > g.Time {
		t.Fatalf("va kiếm không được đẩy hai người ra: %+v %+v", a.KnockVel, b.KnockVel)
	}
	g.resolveSwordSword() // cooldown 0.3s prevents an immediate second clash
	if a.Ring.Dir != -1 {
		t.Fatal("clash cooldown not respected")
	}
}

// Spec §5.2: hai vòng kiếm chạm nhau thì đứng yên mài nhau cho tới khi một bên hết kiếm —
// không ai bị đẩy ra, và bên yếu hơn cạn kiếm trước.
func TestClashGrindsUntilOneSideRunsOut(t *testing.T) {
	g := quietGame(t, nil)
	low := g.AddPlayer(SpawnOptions{Name: "Low", Level: 0, Swords: 6})
	high := g.AddPlayer(SpawnOptions{Name: "High", Level: 30, Swords: 6})
	place(g, low, Vec{X: 5000, Y: 5000}, 0)
	touchSwords(g, low, high, math.Pi)
	start := low.Pos
	for i := 0; i < 30*10 && low.Ring.Count > 0; i++ {
		g.resolveSwordSword()
		g.Time += dt
	}
	if low.Ring.Count != 0 {
		t.Fatalf("bên yếu còn %d kiếm, phải bị mài sạch", low.Ring.Count)
	}
	if high.Ring.Count == 0 {
		t.Fatal("bên mạnh cũng hết kiếm: tỉ lệ đổi kiếm sai")
	}
	if low.Pos.Dist(start) > 1e-9 {
		t.Fatalf("va kiếm làm người bị dịch chuyển: %+v → %+v", start, low.Pos)
	}
}

// Spec §5.2 (v3.5): khác cảnh giới thì không ai đổi chiều quay, chỉ bể kiếm.
func TestClashAcrossRealmsDoesNotFlip(t *testing.T) {
	g := quietGame(t, nil)
	low := g.AddPlayer(SpawnOptions{Name: "Low", Level: 0, Swords: 8})
	high := g.AddPlayer(SpawnOptions{Name: "High", Level: 30, Swords: 8})
	place(g, low, Vec{X: 5000, Y: 5000}, 0)
	touchSwords(g, low, high, math.Pi)
	g.resolveSwordSword()
	if low.Ring.Dir != 1 || high.Ring.Dir != 1 {
		t.Fatalf("khác cảnh giới mà vẫn đổi chiều: %v, %v", low.Ring.Dir, high.Ring.Dir)
	}
	// v4.1: bên thấp mất đúng clash_unit = 1; bên cao mới chỉ nứt 1/1.728 nên chưa mất kiếm.
	if low.Ring.Count != 7 || high.Ring.Count != 8 {
		t.Fatalf("số kiếm sau va = %d, %d; muốn 7, 8", low.Ring.Count, high.Ring.Count)
	}
}

// Spec §6.2: a sword item gives +3, a bundle +15, capped at the realm max; at max → +20 EXP.
func TestSwordPickupOvercapGivesExp(t *testing.T) {
	g := quietGame(t, nil)
	// Luyện Khí giờ tối đa 10 kiếm (spec §3.1): đã đầy thì nhặt kiếm đổi thành EXP.
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 10})
	g.pickup(p, g.addItem("sword", 0, p.Pos))
	if p.Ring.Count != 10 || p.TotalExp != 20 {
		t.Fatalf("count %d exp %v, want 10 swords and +20 EXP", p.Ring.Count, p.TotalExp)
	}
	q := g.AddPlayer(SpawnOptions{Name: "Q", Swords: 1})
	g.pickup(q, g.addItem("sword", 0, q.Pos))
	if q.Ring.Count != 4 {
		t.Fatalf("count %d, want 4", q.Ring.Count)
	}
	g.pickup(q, g.addItem("bundle", 0, q.Pos))
	if q.Ring.Count != 10 {
		t.Fatalf("count %d after a bundle, want 10 (chạm trần Luyện Khí)", q.Ring.Count)
	}
	big := g.AddPlayer(SpawnOptions{Name: "Big", Level: 20, Swords: 55}) // Kim Đan: trần 60
	g.pickup(big, g.addItem("bundle", 0, big.Pos))
	if big.Ring.Count != 60 {
		t.Fatalf("bundle must stop at the realm max: %d", big.Ring.Count)
	}
}

func TestBuffExpiresAndRefreshes(t *testing.T) {
	g := quietGame(t, nil)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	g.ApplyBuff(p, "haste")
	for g.Time < 5 {
		g.Tick(dt)
	}
	g.ApplyBuff(p, "haste") // refresh, not stack
	if exp := p.Buffs[BuffHaste]; math.Abs(exp-(g.Time+10)) > 1e-9 {
		t.Fatalf("haste expiry %v, want %v", exp, g.Time+10)
	}
	for g.Time < 14.9 {
		g.Tick(dt)
	}
	if !p.HasBuff(BuffHaste, g.Time) {
		t.Fatal("haste ended early")
	}
	for g.Time < 15.2 {
		g.Tick(dt)
	}
	if p.HasBuff(BuffHaste, g.Time) {
		t.Fatal("haste did not expire")
	}
}

func TestExp2DoublesGains(t *testing.T) {
	g := quietGame(t, nil)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	g.ApplyBuff(p, "exp2")
	g.GainExp(p, 30, true)
	if p.TotalExp != 60 {
		t.Fatalf("exp = %v, want 60", p.TotalExp)
	}
}

// Spec v4.3 bỏ hẳn Trường Kiếm: không vật phẩm nào kéo dài kiếm nữa, và `long` không còn
// là một buff hợp lệ.
func TestNoBuffExtendsTheBlade(t *testing.T) {
	g := quietGame(t, nil)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	l0 := p.Geom.Len(0)
	for _, name := range BuffNames {
		g.ApplyBuff(p, name)
		if math.Abs(p.Geom.Len(0)-l0) > 1e-9 {
			t.Fatalf("buff %q đổi chiều dài kiếm %v → %v", name, l0, p.Geom.Len(0))
		}
	}
	if _, ok := BuffByName("long"); ok {
		t.Fatal("vẫn còn buff \"long\": spec v4.3 đã bỏ Trường Kiếm")
	}
	if _, ok := g.Cfg.Items.Weights["long"]; ok {
		t.Fatal("config vẫn còn thả vật phẩm Trường Kiếm")
	}
}

// Spec v4.4 §5.3: Cương Kiếm nhân đôi sức phá vật cản.
func TestHardBuffDoublesObstacleDamage(t *testing.T) {
	g := quietGame(t, nil)
	p := g.AddPlayer(SpawnOptions{Name: "P", Level: 20, Swords: 30})
	before := g.ObstacleDamage(p)
	g.ApplyBuff(p, "hard")
	if after := g.ObstacleDamage(p); math.Abs(after-before*g.Cfg.Buffs["hard"].DamageMult) > 1e-9 {
		t.Fatalf("sức phá vật cản %v → %v, muốn ×%v", before, after, g.Cfg.Buffs["hard"].DamageMult)
	}
}

func obstacleMap(kind string, hp int) *world.Map {
	m := world.EmptyMap(12000)
	m.Obstacles = []world.Obstacle{{ID: 0, Kind: kind, MaxHP: hp, Shape: world.Shape{X: 6000, Y: 6000, W: 200, H: 40}}}
	return m
}

func TestObstacleBreaksAndRegrowsWhenClear(t *testing.T) {
	g := quietGame(t, obstacleMap("fence", 30))
	g.DamageObstacle(0, 30)
	if !g.Obstacles[0].Broken || g.ObstacleSolid(0) {
		t.Fatal("fence should be broken and passable")
	}
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, Vec{X: 6100, Y: 6020}, 0) // standing on the rubble
	for g.Time < 31 {
		g.Tick(dt)
		p.Pos = Vec{X: 6100, Y: 6020}
	}
	if !g.Obstacles[0].Broken {
		t.Fatal("obstacle regrew on top of a player")
	}
	p.Pos = Vec{X: 3000, Y: 3000}
	g.Tick(dt)
	if g.Obstacles[0].Broken || g.Obstacles[0].HP != 30 {
		t.Fatalf("obstacle should regrow with full HP: %+v", g.Obstacles[0])
	}
}

func TestSwordDamagesObstacleAndSurvives(t *testing.T) {
	g := quietGame(t, obstacleMap("brick", 250))
	p := g.AddPlayer(SpawnOptions{Name: "P", Level: 20, Swords: 1})
	place(g, p, Vec{}, -math.Pi/2)
	place(g, p, Vec{X: 6100, Y: 6020}.Sub(bladeMid(p).Sub(p.Pos)), -math.Pi/2)
	g.hitObstacles(p)
	// Kim Đan (r=2), 1 kiếm → 1 vòng: damage = (2+1) × 1 = 3 (spec §5.3).
	if g.Obstacles[0].HP != 250-3 {
		t.Fatalf("brick HP = %d, want 247", g.Obstacles[0].HP)
	}
	if p.Ring.Count != 1 {
		t.Fatal("sword lost on obstacle")
	}
}

// Spec §5.3 (v3.6): damage tính theo người, không theo từng kiếm. Thêm kiếm trong cùng một
// vòng không làm phá nhanh hơn; chỉ thêm VÒNG mới tăng sức phá (+50%/vòng).
func TestObstacleDamagePerPlayerNotPerSword(t *testing.T) {
	hitOnceAt := func(swords int) float64 {
		g := quietGame(t, obstacleMap("boulder", 800))
		p := g.AddPlayer(SpawnOptions{Name: "P", Level: 0, Swords: swords})
		place(g, p, Vec{}, -math.Pi/2)
		place(g, p, Vec{X: 6100, Y: 6020}.Sub(bladeMid(p).Sub(p.Pos)), -math.Pi/2)
		g.hitObstacles(p)
		return float64(800-g.Obstacles[0].HP) + g.Obstacles[0].Partial
	}
	// Vòng 1 chứa 10 kiếm: 1 kiếm và 10 kiếm đều là 1 vòng nên sức phá phải bằng nhau.
	one, ten := hitOnceAt(1), hitOnceAt(10)
	if one != ten {
		t.Fatalf("1 kiếm gây %v, 10 kiếm (vẫn 1 vòng) gây %v — phải bằng nhau", one, ten)
	}
	thirty := hitOnceAt(30) // đủ 2 vòng → ×1.5
	if math.Abs(thirty-one*1.5) > 1e-9 {
		t.Fatalf("2 vòng gây %v, muốn %v (gấp 1.5 lần 1 vòng)", thirty, one*1.5)
	}
}

// Spec §8.2: một người Luyện Khí đủ 60 kiếm (3 vòng) phá rào tre ≈ 7.5s.
func TestObstacleBreakTimeMatchesSpec(t *testing.T) {
	g := quietGame(t, obstacleMap("fence", 30))
	p := g.AddPlayer(SpawnOptions{Name: "P", Level: 0, Swords: 60})
	place(g, p, Vec{}, -math.Pi/2)
	place(g, p, Vec{X: 6100, Y: 6020}.Sub(bladeMid(p).Sub(p.Pos)), -math.Pi/2)
	start := g.Time
	for i := 0; i < 30*20 && !g.Obstacles[0].Broken; i++ {
		g.hitObstacles(p)
		g.Time += dt
	}
	took := g.Time - start
	if !g.Obstacles[0].Broken {
		t.Fatal("Luyện Khí 60 kiếm phải phá được rào tre")
	}
	if took < 6.5 || took > 8.5 {
		t.Fatalf("phá rào tre mất %.1fs, spec ghi ≈7.5s", took)
	}
}

func TestBodyBlockedByObstacle(t *testing.T) {
	g := quietGame(t, obstacleMap("stone_wall", 0))
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, Vec{X: 6100, Y: 5900}, 0)
	p.Moving, p.InputAngle = true, math.Pi/2
	for i := 0; i < 60; i++ {
		g.Tick(dt)
	}
	if p.Pos.Y > 6000-g.BodyR(p)+1 {
		t.Fatalf("player walked into the wall: y=%v", p.Pos.Y)
	}
}

func TestGateTeleportsWithoutInvulnerability(t *testing.T) {
	m := world.EmptyMap(12000)
	m.Gates = []world.Gate{{ID: 0, X: 2000, Y: 2000, Network: "a"}, {ID: 1, X: 9000, Y: 9000, Network: "a"}}
	g := quietGame(t, m)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, Vec{X: 2000, Y: 2000}, 0)
	g.Tick(dt)
	if !p.InTransit(g.Time) {
		t.Fatal("expected transit")
	}
	for g.Time < g.Cfg.Gates.TransitSecs+2*dt {
		g.Tick(dt)
	}
	if p.Pos.Dist(Vec{X: 9000, Y: 9000}) > 400 {
		t.Fatalf("player at %+v, want near exit gate", p.Pos)
	}
	if p.Invulnerable(g.Time) || p.InTransit(g.Time) {
		t.Fatal("no invulnerability after leaving a gate")
	}
}

// Cổng ghép cặp 2 chiều là luật CŨ của §8.4. Chủ dự án đã đổi: mặc định bây giờ cổng bắn
// ngẫu nhiên (xem `gates.random_exit`, test trong newrules_test.go). Cách ghép cặp vẫn giữ
// nguyên và vẫn phải đúng, vì đó là đường chạy khi tắt `random_exit`.
func TestGatesAreTwoWayPairsWhenRandomOff(t *testing.T) {
	m := world.EmptyMap(12000)
	m.Gates = []world.Gate{
		{ID: 0, X: 2000, Y: 2000, Network: "ring"}, {ID: 1, X: 9000, Y: 9000, Network: "ring"},
		{ID: 2, X: 2000, Y: 9000, Network: "ring"}, {ID: 3, X: 9000, Y: 2000, Network: "ring"},
	}
	for _, gate := range m.Gates {
		exit, ok := m.GateExit(gate)
		if !ok {
			t.Fatalf("cổng %d không có cổng đích", gate.ID)
		}
		back, ok := m.GateExit(exit)
		if !ok || back.ID != gate.ID {
			t.Fatalf("cổng %d → %d nhưng %d → %d: không phải 2 chiều", gate.ID, exit.ID, exit.ID, back.ID)
		}
	}
	// Và luật thực tế cũng đưa người chơi đúng tới cổng đã ghép cặp — khi đã tắt bắn ngẫu nhiên.
	g := quietGame(t, m)
	g.Cfg.Gates.RandomExit = false
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, Vec{X: 2000, Y: 2000}, 0)
	for g.Time < g.Cfg.Gates.TransitSecs+2*dt {
		g.Tick(dt)
	}
	if p.Pos.Dist(Vec{X: 9000, Y: 9000}) > 400 {
		t.Fatalf("ra ở %+v, muốn gần cổng ghép cặp (9000, 9000)", p.Pos)
	}
}

func TestMoveSpeedFormula(t *testing.T) {
	g := quietGame(t, nil)
	base := g.Cfg.Player.MoveBase
	low := g.AddPlayer(SpawnOptions{Name: "Low", Level: 0, Swords: 1})
	high := g.AddPlayer(SpawnOptions{Name: "High", Level: 89, Swords: 1})
	// Tu vi không làm tăng tốc độ: cảnh giới thấp nhất và cao nhất chạy bằng nhau.
	if s := g.MoveSpeed(low); math.Abs(s-base) > 1e-9 {
		t.Fatalf("speed cảnh giới thấp = %v, muốn %v", s, base)
	}
	if s := g.MoveSpeed(high); math.Abs(s-base) > 1e-9 {
		t.Fatalf("speed cảnh giới cao = %v, muốn %v (không tăng theo tu vi)", s, base)
	}
	// Chỉ vật phẩm (buff Thần Hành) mới tăng tốc.
	low.Buffs[BuffHaste] = g.Time + 10
	if s := g.MoveSpeed(low); math.Abs(s-base*g.Cfg.Buffs["haste"].Mult) > 1e-9 {
		t.Fatalf("speed khi có buff = %v", s)
	}
}

func TestKillRewardAndDeathSwordDrop(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Level: 5, Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 0, Swords: 50})
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	place(g, b, bladeMid(a), math.Pi/2+math.Pi/5)
	a.InvulnUntil = 1e9
	b.TotalExp = 80
	aBefore := a.TotalExp
	g.applyBodyHit(a, b) // 50 kiếm = 3 vòng: bỏ qua cổng lá chắn §5.2, kiểm riêng luật chết
	if !b.Dead {
		t.Fatal("B should die at Luyện Khí")
	}
	if got, want := a.TotalExp-aBefore, 80*0.5+30; math.Abs(got-want) > 1e-9 {
		t.Fatalf("killer gained %v, want %v (50%% + kill bonus)", got, want)
	}
	bundles, swords := countItems(g, "bundle")
	_, orbs := countItems(g, "orb")
	if bundles != 3 || swords != 25 {
		t.Fatalf("corpse dropped %d bundles / %v swords, want 3 bundles holding floor(50*0.5)=25", bundles, swords)
	}
	if orbs != 24 {
		t.Fatalf("dropped orb EXP %v, want 24 (30%% of 80)", orbs)
	}
}

func TestWindAngles(t *testing.T) {
	g := quietGame(t, nil)
	cases := []struct {
		deg  float64
		want float64
	}{{0, 1.5}, {45, 1.5}, {90, 1}, {119, 1}, {130, 0.8}, {180, 0.8}}
	for _, c := range cases {
		dir := Vec{X: math.Cos(c.deg * math.Pi / 180), Y: math.Sin(c.deg * math.Pi / 180)}
		if got := g.windMult(dir, 0); got != c.want {
			t.Errorf("%v° off the wind → %v, want %v", c.deg, got, c.want)
		}
	}
}

func TestMonsterOrbsReservedForKiller(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Swords: 1})
	place(g, a, Vec{X: 2000, Y: 2000}, 0)
	place(g, b, Vec{X: 7000, Y: 7000}, 0)
	m := &Monster{ID: g.newID(), Kind: "small", Pos: Vec{X: 7000, Y: 7000}, Exp: 10}
	g.killMonster(m, a)
	g.resolvePickups()
	if b.TotalExp != 0 {
		t.Fatalf("B stole reserved orbs: exp %v", b.TotalExp)
	}
	g.Time += g.Cfg.Player.OrbOwnerSecs + 0.01
	g.resolvePickups()
	if b.TotalExp != 10 {
		t.Fatalf("after the reservation B should collect 10 EXP, got %v", b.TotalExp)
	}
}

// Spec §17: a scale-5 player cannot squeeze through a 150-wide alley; a small one can.
func TestNarrowAlleyBlocksBigPlayers(t *testing.T) {
	m := world.EmptyMap(12000)
	m.Obstacles = []world.Obstacle{
		{ID: 0, Kind: "stone_wall", Shape: world.Shape{X: 5000, Y: 4000, W: 120, H: 2000}},
		{ID: 1, Kind: "stone_wall", Shape: world.Shape{X: 5270, Y: 4000, W: 120, H: 2000}},
	}
	g := quietGame(t, m)
	small := g.AddPlayer(SpawnOptions{Name: "S", Level: 0, Swords: 1})
	big := g.AddPlayer(SpawnOptions{Name: "B", Level: 89, Swords: 1})
	place(g, small, Vec{X: 5195, Y: 3800}, 0)
	place(g, big, Vec{X: 5195, Y: 3500}, 0)
	for _, p := range []*Player{small, big} {
		p.Moving, p.InputAngle = true, math.Pi/2
		p.InvulnUntil = 1e9 // only movement is under test
	}
	for i := 0; i < 30*25; i++ { // 25s: đủ để người nhỏ đi hết hẻm ở tốc độ 130/s
		g.Tick(dt)
		small.Pos.X, big.Pos.X = 5195, 5195 // walk straight down the alley axis
	}
	if small.Pos.Y < 6000 {
		t.Fatalf("small player stuck at y=%v", small.Pos.Y)
	}
	if big.Pos.Y > 4000 {
		t.Fatalf("big player squeezed into the alley: y=%v", big.Pos.Y)
	}
}

func TestStealthAndBushVisibility(t *testing.T) {
	m := world.EmptyMap(12000)
	m.Zones = []world.Zone{{ID: 0, Kind: "bush", Shape: world.Shape{Circle: true, X: 6000, Y: 6000, R: 300}}}
	g := quietGame(t, m)
	viewer := g.AddPlayer(SpawnOptions{Name: "V", Swords: 1})
	hider := g.AddPlayer(SpawnOptions{Name: "H", Swords: 1})
	place(g, viewer, Vec{X: 5400, Y: 6000}, 0)
	place(g, hider, Vec{X: 6000, Y: 6000}, 0)
	g.Tick(dt) // updates bush membership
	if g.CanSee(viewer, hider) {
		t.Fatal("player in a bush must be hidden from outside")
	}
	viewer.Pos = Vec{X: 6000 - 140, Y: 6000}
	if !g.CanSee(viewer, hider) {
		t.Fatal("within 150 × scale the bush no longer hides")
	}
	hider.RevealedUntil = g.Time + 2
	viewer.Pos = Vec{X: 5400, Y: 6000}
	if !g.CanSee(viewer, hider) {
		t.Fatal("being hit reveals the player for 2s")
	}
	// Ẩn Thân (spec §6.2 từ v3.3): vẫn gửi xuống client, chỉ bị vẽ mờ khi ở xa.
	hider.RevealedUntil = 0
	hider.Pos = Vec{X: 8000, Y: 6000}
	g.Tick(dt)
	g.ApplyBuff(hider, "stealth")
	viewer.Pos = Vec{X: 7500, Y: 6000}
	if !g.CanSee(viewer, hider) {
		t.Fatal("người Ẩn Thân vẫn phải được gửi xuống (vẽ mờ), không biến mất")
	}
	if !g.StealthFaded(viewer, hider) {
		t.Fatal("cách 500 đơn vị thì người Ẩn Thân phải bị vẽ mờ")
	}
	viewer.Pos = Vec{X: 7900, Y: 6000}
	if g.StealthFaded(viewer, hider) {
		t.Fatal("cách 100 đơn vị (< reveal_r) thì phải hiện rõ 100%")
	}
}

// Không có buff Ẩn Thân thì không ai bị coi là tàng hình — kể cả chính mình (nếu không
// game sẽ vẽ mờ nhân vật của người chơi suốt trận).
func TestStealthFadedFalseWithoutBuff(t *testing.T) {
	g := quietGame(t, nil)
	me := g.AddPlayer(SpawnOptions{Name: "Me", Swords: 1})
	other := g.AddPlayer(SpawnOptions{Name: "Other", Swords: 1})
	place(g, me, Vec{X: 6000, Y: 6000}, 0)
	place(g, other, Vec{X: 9000, Y: 6000}, 0)
	if g.StealthFaded(me, me) {
		t.Fatal("tự nhìn mình mà bị coi là Ẩn Thân")
	}
	if g.StealthFaded(me, other) {
		t.Fatal("người không có buff mà bị coi là Ẩn Thân")
	}
}

// Spec §6.2: Ẩn Thân tối đa max_secs, nhặt lại chỉ làm mới chứ không cộng dồn.
func TestStealthCappedAtMaxSecs(t *testing.T) {
	g := quietGame(t, nil)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	cfg := g.Cfg.Buffs["stealth"]
	g.ApplyBuff(p, "stealth")
	g.ApplyBuff(p, "stealth")
	if want := g.Time + cfg.MaxSecs; math.Abs(p.Buffs[BuffStealth]-want) > 1e-9 {
		t.Fatalf("hạn Ẩn Thân = %v, muốn %v", p.Buffs[BuffStealth], want)
	}
}

// Spec §10: bot chỉ phát hiện người Ẩn Thân trong 50% tầm nhìn bình thường.
func TestBotSightHalvedAgainstStealth(t *testing.T) {
	g := quietGame(t, nil)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	if m := g.BotSightMult(p); m != 1 {
		t.Fatalf("không Ẩn Thân thì hệ số tầm nhìn = %v, muốn 1", m)
	}
	g.ApplyBuff(p, "stealth")
	if m := g.BotSightMult(p); m != g.Cfg.Buffs["stealth"].BotDetectMult {
		t.Fatalf("hệ số tầm nhìn khi Ẩn Thân = %v, muốn %v", m, g.Cfg.Buffs["stealth"].BotDetectMult)
	}
}

func TestStealthEndsWhenOwnerHitsSomeone(t *testing.T) {
	g := quietGame(t, nil)
	a := g.AddPlayer(SpawnOptions{Name: "A", Swords: 1})
	b := g.AddPlayer(SpawnOptions{Name: "B", Level: 15, Swords: 1})
	place(g, a, Vec{X: 5000, Y: 5000}, 0)
	place(g, b, bladeMid(a), math.Pi/2)
	g.ApplyBuff(a, "stealth")
	g.resolveSwordBody()
	if a.HasBuff(BuffStealth, g.Time) {
		t.Fatal("Ẩn Thân must end when the owner's sword hits a player")
	}
}

func TestChestAnnouncedThenBurstsIntoItems(t *testing.T) {
	g := quietGame(t, nil)
	g.Cfg.Items.ChestEverySecs = 20
	g.chestNextAt = 20
	var announced bool
	for g.Time < 21 {
		g.Tick(dt)
		for _, e := range g.DrainEvents() {
			if e.Kind == "ChestIncoming" {
				announced = true
				if g.Time > 20-g.Cfg.Items.ChestWarnSecs+0.1 {
					t.Fatalf("chest announced late at %v", g.Time)
				}
			}
		}
	}
	var chest *Item
	for _, it := range g.Items {
		if it.Kind == "chest" {
			chest = it
		}
	}
	if !announced || chest == nil {
		t.Fatalf("announced %v chest %v", announced, chest)
	}
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, chest.Pos, 0)
	g.resolvePickups()
	n := 0
	for _, it := range g.Items {
		if it.Kind != "chest" && it.Kind != "orb" {
			n++
		}
	}
	// Picking up the chest also scoops up items that land within pickup range.
	if _, still := g.Items[chest.ID]; still || n+p.Ring.Count-1 < 1 {
		t.Fatalf("chest should burst into items (left %d)", n)
	}
}

func TestBreakthroughShocksNearbyAndBushRevealEmotes(t *testing.T) {
	m := world.EmptyMap(12000)
	m.Zones = []world.Zone{{ID: 0, Kind: "bush", Shape: world.Shape{Circle: true, X: 6000, Y: 6000, R: 300}}}
	g := quietGame(t, m)
	p := g.AddPlayer(SpawnOptions{Name: "P", Level: 9, Swords: 1})
	near := g.AddPlayer(SpawnOptions{Name: "N", Swords: 1})
	far := g.AddPlayer(SpawnOptions{Name: "F", Swords: 1})
	place(g, p, Vec{X: 3000, Y: 3000}, 0)
	place(g, near, Vec{X: 3300, Y: 3000}, 0)
	place(g, far, Vec{X: 9000, Y: 9000}, 0)
	g.DrainEvents()
	g.GainExp(p, g.Prog.ExpNeeded(9), false)
	shocked := map[uint32]bool{}
	for _, e := range g.DrainEvents() {
		if e.Kind == "Emote" && e.S == "shocked" {
			shocked[e.A] = true
		}
	}
	if !shocked[near.ID] || shocked[far.ID] || shocked[p.ID] {
		t.Fatalf("shocked = %v", shocked)
	}
	// A player hiding in the bush gets a "revealed" bubble when hit.
	hider := g.AddPlayer(SpawnOptions{Name: "H", Level: 15, Swords: 1})
	att := g.AddPlayer(SpawnOptions{Name: "A", Swords: 1})
	place(g, hider, Vec{X: 6000, Y: 6000}, math.Pi/2)
	place(g, att, Vec{X: 6000 - 100, Y: 6000}, 0)
	hider.Bush = 0
	g.DrainEvents()
	g.applyBodyHit(att, hider)
	found := false
	for _, e := range g.DrainEvents() {
		if e.Kind == "Emote" && e.S == "revealed" && e.A == hider.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("missing revealed emote")
	}
}

// Spec §8.2: bodies are blocked only by the footprint (bottom 30%); a player can walk across
// the top part of a boulder's sprite.
func TestFootprintBlocksOnlyTheBase(t *testing.T) {
	m := world.EmptyMap(12000)
	sh := world.Shape{X: 6000, Y: 6000, W: 400, H: 400}
	m.Obstacles = []world.Obstacle{{ID: 0, Kind: "boulder", Shape: sh, Foot: world.Footprint(sh, 0.9, 0.3)}}
	g := quietGame(t, m)
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, Vec{X: 6200, Y: 6100}, 0) // inside the top of the sprite
	if np := g.resolveBlocking(p.Pos, g.BodyR(p)); np != p.Pos {
		t.Fatalf("upper part of the sprite must not block: pushed to %+v", np)
	}
	if np := g.resolveBlocking(Vec{X: 6200, Y: 6350}, g.BodyR(p)); np.Dist(Vec{X: 6200, Y: 6350}) < 1 {
		t.Fatal("footprint must block")
	}
}

// Spec §6.1: ambient orbs cover the map and regrow in the same 1000×1000 region after 5s.
func TestAmbientOrbsRegrowInSameRegion(t *testing.T) {
	cfg := config.MustLoadDefault()
	cfg.Room.MonsterCount, cfg.Items.CountPerRoom = 0, 0
	g := New(cfg, world.EmptyMap(cfg.Room.MapSize), 1)
	n, _ := countItems(g, "orb")
	cells := int(cfg.Room.MapSize / cfg.ExpOrbs.Cell)
	if n < cells*cells*9/10 {
		t.Fatalf("ambient orbs = %d, want about %d", n, cells*cells)
	}
	var orb *Item
	for _, it := range g.Items {
		if it.Region >= 0 {
			orb = it
			break
		}
	}
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	place(g, p, orb.Pos, 0)
	g.pickup(p, orb)
	g.RemovePlayer(p.ID)
	for g.Time < cfg.ExpOrbs.RespawnSecs-0.1 {
		g.Tick(dt)
	}
	if m, _ := countItems(g, "orb"); m != n-1 {
		t.Fatalf("orb regrew early: %d vs %d", m, n-1)
	}
	for g.Time < cfg.ExpOrbs.RespawnSecs+0.1 {
		g.Tick(dt)
	}
	if m, _ := countItems(g, "orb"); m != n {
		t.Fatalf("orb did not regrow: %d vs %d", m, n)
	}
	size := cfg.ExpOrbs.RespawnRegion
	for _, it := range g.Items {
		if it.Region == orb.Region && (int(it.Pos.X/size) != int(orb.Pos.X/size) || int(it.Pos.Y/size) != int(orb.Pos.Y/size)) {
			t.Fatalf("orb regrew outside its region: %+v", it.Pos)
		}
	}
}

// Spec §6.2: the room keeps count_per_room items; a picked one returns respawn_secs later.
func TestItemPoolRespawnsAfterDelay(t *testing.T) {
	cfg := config.MustLoadDefault()
	cfg.Room.MonsterCount, cfg.ExpOrbs.Cell, cfg.Items.ChestEverySecs = 0, 0, 1e9
	cfg.Items.CountPerRoom = 10
	g := New(cfg, world.EmptyMap(cfg.Room.MapSize), 1)
	g.Tick(dt)
	if g.poolCount != 10 {
		t.Fatalf("pool = %d, want 10", g.poolCount)
	}
	p := g.AddPlayer(SpawnOptions{Name: "P", Swords: 1})
	for _, it := range g.Items {
		g.pickup(p, it)
		break
	}
	g.RemovePlayer(p.ID)
	t0 := g.Time
	for g.Time < t0+cfg.Items.RespawnSecs-0.2 {
		g.Tick(dt)
	}
	if g.poolCount != 9 {
		t.Fatalf("item came back early: pool %d", g.poolCount)
	}
	for g.Time < t0+cfg.Items.RespawnSecs+0.2 {
		g.Tick(dt)
	}
	if g.poolCount != 10 {
		t.Fatalf("item did not come back: pool %d", g.poolCount)
	}
}
