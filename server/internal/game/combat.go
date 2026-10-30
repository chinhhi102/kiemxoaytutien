package game

import (
	"math"

	"kiemxoay/server/internal/geom"
	"kiemxoay/server/internal/world"
)

const (
	tagMonster uint8 = iota + 1
	tagObstacle
	tagClash
	tagFlip
)

// swordBreakFlushSecs: nhịp báo kiếm bể cho client, 6 lần/giây là đủ mượt mắt.
const swordBreakFlushSecs = 1.0 / 6

// canFight is true when a player can deal and receive sword interactions this tick.
func (g *Game) canFight(p *Player) bool { return !p.Dead && !p.InTransit(g.Time) }

// swordTouchesCircle returns the first sword of owner that swept over circle (c, r) this tick.
func (g *Game) swordTouchesCircle(owner *Player, c Vec, r float64) (int, bool) {
	found, ok := 0, false
	sweptHits(owner.Pos, owner.Ring, owner.Geom, c, r, func(i int) bool {
		found, ok = i, true
		return false
	})
	return found, ok
}

// resolveSwordBody applies spec §5.1: at most one hit per victim per tick.
//
// Lá chắn (§5.2) được kiểm ở đây chứ không chặn đường đi nữa: còn vòng kế ngoài cùng thì
// kiếm địch dù quét trùm lên thân cũng không ăn — phải mài cho rụng hết vòng ngoài đã.
func (g *Game) resolveSwordBody() {
	players := g.PlayerList()
	for _, b := range players {
		// Rụt trong mai thì không bị chém thân (spec §6.3).
		if !g.canFight(b) || b.Invulnerable(g.Time) || g.Shielded(b) || b.HasStatus(StShell, g.Time) {
			continue
		}
		for _, a := range players {
			if a == b || !g.canFight(a) {
				continue
			}
			if a.IsMinion || sameSide(a, b) {
				continue // đàn em chỉ mài kiếm, và không đụng tới phe mình (spec §6.3)
			}
			if _, hit := g.swordTouchesCircle(a, b.Pos, g.BodyR(b)); hit {
				g.applyBodyHit(a, b)
				break
			}
		}
	}
}

func (g *Game) applyBodyHit(a, b *Player) {
	pc := g.Cfg.Player
	a.Buffs[BuffStealth] = 0 // Ẩn Thân ends as soon as the owner's sword hits someone
	if g.HiddenInBush(b) {
		g.emote(b, "revealed") // spec §12: bubble when knocked out of hiding
	}
	b.InvulnUntil = g.Time + pc.InvulnOnHit
	b.RevealedUntil = g.Time + g.Cfg.Zones.BushRevealOnHitSecs
	push := b.Pos.Sub(a.Pos).Norm()
	b.KnockVel = push.Mul(pc.Knockback / pc.KnockbackSecs)
	b.KnockUntil = g.Time + pc.KnockbackSecs
	ev := Event{Kind: "Hit", A: a.ID, B: b.ID, X: float32(b.Pos.X), Y: float32(b.Pos.Y)}
	if b.HasBuff(BuffShield, g.Time) {
		b.Buffs[BuffShield] = 0
		ev.S = "shield"
		g.emit(ev)
		g.emote(b, "shield_break")
		return
	}
	g.emit(ev)
	dies := b.Level == 0
	if pc.HitDropsRealm {
		dies = g.Prog.Realm(b.Level) == 0 // Luyện Khí at any tier dies
	}
	if dies {
		g.kill(a, b)
		return
	}
	g.setMood(b, MoodHit, g.Cfg.Mood.HitSecs)
	oldRealm := g.Prog.Realm(b.Level)
	g.awardHit(a, b, oldRealm, false) // scored on the realm they had before being knocked down
	if g.Prog.IsTien(oldRealm) {
		g.robTienKhi(a, b) // Tiên giới: rớt cảnh giới + bị cướp Tiên khí (§3.4)
		return
	}
	after := g.expAfterHit(b)
	loss := b.TotalExp - after
	g.setTotalExp(b, after)
	g.GainExp(a, loss*pc.AttackerShare, true)
	g.scatterOrbs(b.Pos, loss*pc.DropShare, a.ID) // viên thuộc người chém 1.5s, không thì nạn nhân hút lại ngay
	if g.Prog.Realm(b.Level) < oldRealm {
		g.emit(Event{Kind: "KillFeed", Global: true, S: "realm_drop", A: a.ID, B: b.ID, N: g.Prog.Realm(b.Level)})
	}
}

// expAfterHit is the victim's total_exp after a body hit (spec §5.1): one whole realm down,
// same tier, same progress through the tier. With hit_drops_realm off it is one tier down.
func (g *Game) expAfterHit(b *Player) float64 {
	if !g.Cfg.Player.HitDropsRealm {
		return math.Max(0, b.TotalExp-g.Prog.ExpNeeded(b.Level-1))
	}
	return g.Prog.DropRealmExp(b.TotalExp)
}

// kill handles a Luyện Khí player being cut down (spec §5.1, §7).
func (g *Game) kill(a, b *Player) {
	pc := g.Cfg.Player
	// Scored first: the EXP reward below can push the attacker up a realm, which would
	// shrink their lật_kèo multiplier for this very kill (spec §11.1).
	g.awardHit(a, b, g.Prog.Realm(b.Level), true)
	// Spec §5.1: at Luyện Khí the "loss" is the victim's whole EXP, plus a kill bonus.
	loss := b.TotalExp
	g.GainExp(a, loss*pc.AttackerShare+pc.KillBonusExp, true)
	g.scatterOrbs(b.Pos, loss*pc.DropShare, a.ID) // viên thuộc người chém 1.5s, không thì nạn nhân hút lại ngay
	g.dropDeathSwords(b)
	a.Kills++
	victimRank := g.RankOf(b)
	b.Dead, b.DiedAt, b.KilledBy = true, g.Time, a.ID
	g.onDeathClear(b)
	b.KillerName, b.KillerChar = a.Name, a.Char
	b.Moving = false
	g.setMood(b, MoodDeath, pc.DeathAnimSecs)
	g.emit(Event{Kind: "Death", A: a.ID, B: b.ID, X: float32(b.Pos.X), Y: float32(b.Pos.Y)})
	feed := "kill"
	if b.lastTeleportIn > 0 && g.Time-b.lastTeleportIn < g.Cfg.Gates.FeedSecs {
		feed = "gate_kill"
	}
	// N = hạng của nạn nhân trong bảng phòng lúc bị giết; hạ một người trong Top 5 thì
	// dòng này được loan ra toàn server (spec §11.4). victimRank lấy TRƯỚC khi b.Dead = true.
	g.emit(Event{Kind: "KillFeed", Global: true, S: feed, A: a.ID, B: b.ID, N: victimRank})
}

// DebugKill kills a player without an attacker (debug key only).
func (g *Game) DebugKill(b *Player) {
	g.dropDeathSwords(b)
	b.Dead, b.DiedAt, b.KilledBy, b.Moving = true, g.Time, 0, false
	g.onDeathClear(b)
	b.KillerName, b.KillerChar = "", ""
	g.setMood(b, MoodDeath, g.Cfg.Player.DeathAnimSecs)
	g.emit(Event{Kind: "Death", B: b.ID, X: float32(b.Pos.X), Y: float32(b.Pos.Y)})
}

// resolveSwordSword applies spec §5.2 to every pair of overlapping rings.
// sameSide: hai người này cùng phe không — đại ca với đàn em của mình, hoặc hai đàn em
// cùng chủ.
//
// Thiếu phép kiểm này thì Gọi Đàn Em thành kỹ năng TỰ SÁT: hai đàn em đứng sát đại ca sẽ
// mài chính vòng kiếm của anh ta. Bài đấu từng cặp cho Đại Ca Dép đúng 1,4% tỉ lệ thắng —
// thấp nhất bảng — và đó là lý do.
func sameSide(a, b *Player) bool {
	switch {
	case a.IsMinion && b.IsMinion:
		return a.MinionOf == b.MinionOf
	case a.IsMinion:
		return a.MinionOf == b.ID
	case b.IsMinion:
		return b.MinionOf == a.ID
	}
	return false
}

func (g *Game) resolveSwordSword() {
	players := g.PlayerList()
	for i, a := range players {
		// Spec §5.2: someone stripped to 0 swords has nothing left to clash with.
		if !g.canFight(a) || a.Ring.Count == 0 {
			continue
		}
		for _, b := range players[i+1:] {
			if !g.canFight(b) || b.Ring.Count == 0 {
				continue
			}
			if sameSide(a, b) {
				continue // đàn em không mài kiếm với đại ca và với nhau (spec §6.3)
			}
			// Bể kiếm tính mỗi tick khi hai vòng còn chạm nhau (clash_cooldown = 0), nên
			// hai vòng ngoài 30 kiếm ngang cảnh giới mài sạch nhau trong đúng 1 giây.
			// Đổi chiều quay vẫn giữ cooldown riêng để vòng kiếm không giật liên hồi.
			if at, ok := g.ringsTouch(a, b); ok && g.cooldownReady(pairKey(tagClash, a.ID, b.ID, 0), g.Cfg.Sword.ClashCooldown) {
				g.clash(a, b, at)
			}
		}
	}
}

// ringsTouch finds a contact point between the swept swords of a and b (spec §4.1b). A dense
// ring sweeps its whole annulus each tick, so it is treated as a solid disc of radius reach.
func (g *Game) ringsTouch(a, b *Player) (Vec, bool) {
	d := a.Pos.Dist(b.Pos)
	if d > a.Geom.Outer()+b.Geom.Outer() {
		return Vec{}, false
	}
	denseA, denseB := denseRing(a.Ring, a.Geom), denseRing(b.Ring, b.Geom)
	switch {
	case denseA && denseB:
		if d > a.Geom.Reach+b.Geom.Reach {
			return Vec{}, false
		}
		t := a.Geom.Reach / math.Max(a.Geom.Reach+b.Geom.Reach, 1e-9)
		return a.Pos.Add(b.Pos.Sub(a.Pos).Mul(t)), true
	case denseA:
		return swordEntersDisc(b, a.Pos, a.Geom.Reach)
	case denseB:
		return swordEntersDisc(a, b.Pos, b.Geom.Reach)
	default:
		return g.sparseRingsTouch(a, b)
	}
}

// swordEntersDisc reports whether any sword of p, sampled along its sweep, enters disc (c, r).
func swordEntersDisc(p *Player, c Vec, r float64) (Vec, bool) {
	n := sweepSteps(p.Ring)
	lim := (r + p.Geom.Width/2) * (r + p.Geom.Width/2)
	ct, st := math.Cos(p.Ring.Tilt), math.Sin(p.Ring.Tilt)
	for s := 0; s <= n; s++ {
		ph := sweepPhase(p.Ring, s, n)
		for i := 0; i < p.Ring.Count; i++ {
			h, t, _ := swordAtRot(p.Pos, ph, p.Ring, p.Geom, i, ct, st)
			if geom.PointSegDist2(c, h, t) <= lim {
				return h.Add(t).Mul(0.5), true
			}
		}
	}
	return Vec{}, false
}

// blade là một thanh kiếm đã tính sẵn vị trí, kèm tâm và nửa chiều dài để lọc thô.
type blade struct {
	hilt, tip Vec
	mid       Vec
	half      float64
}

// sparseRingsTouch tests sword pairs at matching sub-steps of both sweeps (few swords only).
//
// Ba điều quan trọng về tốc độ ở đây — pprof trên server thật với 10 người chơi cho thấy
// riêng hàm này ăn **83% CPU server**, nên nó đáng được viết kỹ:
//
//  1. Kiếm của b tính SẴN một lần mỗi bước quét, không tính lại trong vòng lặp của a. Bản
//     cũ gọi `swordAt(b…)` đúng `a.Count × b.Count` lần thay vì `b.Count` lần.
//  2. Bỏ sớm những lưỡi nằm ở phía ĐỐI DIỆN. Hai người đứng cạnh nhau thì chỉ một cung nhỏ
//     của mỗi vòng có cơ hội chạm; lưỡi nào cách tâm đối phương xa hơn `bán kính ngoài +
//     nửa lưỡi + bề rộng` thì không thanh nào với tới.
//  3. So khoảng cách giữa hai TÂM lưỡi trước khi gọi `SegSegDist2`: một phép trừ và hai
//     phép nhân, rẻ hơn hàng chục lần.
//
// outerSlots trả khoảng chỉ số kiếm của **vòng ngoài cùng** [from, to).
//
// Spec mục 26: kiếm–kiếm chỉ xét vòng ngoài cùng. Lý do hình học: các vòng đồng tâm, vòng
// ngoài có bán kính lớn nhất nên nó là thứ chạm nhau TRƯỚC — hai vòng trong chỉ gặp được
// nhau sau khi hai vòng ngoài đã lồng vào nhau, mà lúc đó va chạm đã được tính rồi.
//
// Khác biệt về công sức là cả một bậc: Đạo Tổ có 1 050 kiếm chia 14 vòng, vòng ngoài cùng
// chỉ 140 kiếm. Hai người Đạo Tổ đánh nhau là **1 050 × 1 050 → 140 × 140**, ít hơn 56 lần.
// Một chi tiết không bỏ được: vòng ngoài cùng lúc sắp mài xong chỉ còn **vài cây kiếm lẻ**,
// và hai cây lẻ trên hai vòng tròn lớn thì hiếm khi gặp nhau — mài tắc hẳn. Đo được: hai
// Kim Đan 60 kiếm, tới tick 28 vòng ngoài còn đúng 2 kiếm thì hai tick cuối không va lần
// nào, và sau 1 giây còn 32 kiếm thay vì 30 như §5.2 đòi.
//
// Nên lấy từ vòng ngoài cùng **đi xuống cho tới khi đủ một vòng đầy**: mặt chạm luôn liền
// mạch. Thực tế là 1–2 vòng, vẫn ít hơn 14 vòng của Đạo Tổ cả chục lần.
func outerSlots(ring Ring, g RingGeom) (int, int) {
	rings := g.RingsFor(ring.Count)
	if rings <= 0 {
		return 0, 0
	}
	k := rings - 1
	from := g.RingStart(k)
	for k > 0 && ring.Count-from < g.RingCap(k) {
		k--
		from = g.RingStart(k)
	}
	return min(from, ring.Count), ring.Count
}

func (g *Game) sparseRingsTouch(a, b *Player) (Vec, bool) {
	n := max(sweepSteps(a.Ring), sweepSteps(b.Ring))
	reach := (a.Geom.Width + b.Geom.Width) / 2
	reach2 := reach * reach
	outerA, outerB := a.Geom.Outer(), b.Geom.Outer()
	ctA, stA := math.Cos(a.Ring.Tilt), math.Sin(a.Ring.Tilt)
	ctB, stB := math.Cos(b.Ring.Tilt), math.Sin(b.Ring.Tilt)
	fromA, toA := outerSlots(a.Ring, a.Geom)
	fromB, toB := outerSlots(b.Ring, b.Geom)
	for s := 0; s <= n; s++ {
		pa, pb := sweepPhase(a.Ring, s, n), sweepPhase(b.Ring, s, n)
		bb := g.bladeBuf[:0]
		for j := fromB; j < toB; j++ {
			// `length` lấy thẳng từ hình học, không đo lại bằng Hypot: cùng một con số mà
			// Hypot một mình ăn 11% CPU server.
			h, t, length := swordAtRot(b.Pos, pb, b.Ring, b.Geom, j, ctB, stB)
			mid, half := h.Add(t).Mul(0.5), length/2
			if lim := outerA + half + reach; mid.Dist2(a.Pos) > lim*lim {
				continue
			}
			bb = append(bb, blade{hilt: h, tip: t, mid: mid, half: half})
		}
		g.bladeBuf = bb
		if len(bb) == 0 {
			continue
		}
		for i := fromA; i < toA; i++ {
			p1, q1, lenA := swordAtRot(a.Pos, pa, a.Ring, a.Geom, i, ctA, stA)
			m1, half1 := p1.Add(q1).Mul(0.5), lenA/2
			if lim := outerB + half1 + reach; m1.Dist2(b.Pos) > lim*lim {
				continue
			}
			for k := range bb {
				gap := half1 + bb[k].half + reach
				if m1.Dist2(bb[k].mid) > gap*gap {
					continue
				}
				if geom.SegSegDist2(p1, q1, bb[k].hilt, bb[k].tip) <= reach2 {
					return p1.Add(q1).Add(bb[k].hilt).Add(bb[k].tip).Mul(0.25), true
				}
			}
		}
	}
	return Vec{}, false
}

// hardMult is the Cương Kiếm multiplier for p right now (spec v4.4 §6.2): "durability" cho
// va kiếm, "damage" cho chém quái và vật cản. 1 khi không có buff.
func (g *Game) hardMult(p *Player, which string) float64 {
	// Gồng Cơ cộng dồn với Cương Kiếm (spec §6.3) — hai nguồn độ bền nhân vào nhau.
	brace := 1.0
	if which == "durability" && p.HasStatus(StBrace, g.Time) {
		brace = g.Cfg.Skills.BraceDurability
	}
	if !p.HasBuff(BuffHard, g.Time) {
		return brace
	}
	b := g.Cfg.Buffs["hard"]
	m := b.DurabilityMult
	if which == "damage" {
		m = b.DamageMult
	}
	if m <= 0 {
		return brace
	}
	return m * brace
}

func (g *Game) clash(a, b *Player, at Vec) {
	ra, rb := g.Prog.Realm(a.Level), g.Prog.Realm(b.Level)
	// Spec §5.2: va kiếm KHÔNG đẩy hai người ra nữa — vòng kiếm chỉ chặn không cho chém
	// vào thân, hai bên đứng mài nhau cho tới khi một bên hết kiếm trước.
	// Đặt clash_knockback > 0 trong config là bật lại kiểu đẩy văng cũ.
	if g.Cfg.Sword.ClashKnockback > 0 {
		g.pushApart(a, b)
	}
	// Spec §5.2 (v3.5): only an even match sends both rings spinning the other way; against a
	// different realm nobody flips, so the stronger ring keeps grinding in the same direction.
	if (!g.Cfg.Sword.FlipOnlySameRealm || ra == rb) && g.cooldownReady(pairKey(tagFlip, a.ID, b.ID, 0), g.Cfg.Sword.FlipCooldown) {
		a.Ring.Dir, b.Ring.Dir = -a.Ring.Dir, -b.Ring.Dir
		// MỘT sự kiện cho một lần va, không phải một cho mỗi bên.
		//
		// Hai vòng va nhau ở đúng một điểm, và client xưa nay vẫn gộp hai sự kiện đó lại để
		// vẽ đúng một tia lửa — tức là server gửi gấp đôi rồi client vứt một nửa. Đo được
		// trong gói snapshot lớn nhất: **56 sự kiện RingFlip, 2,0 KB, 24% gói**.
		//
		// Chiều quay mới KHÔNG mất đi: nó nằm sẵn trong trường `d` của mỗi người, gửi mỗi
		// gói snapshot. Sự kiện này chỉ để vẽ tia lửa và kêu tiếng va.
		g.emit(Event{Kind: "RingFlip", A: a.ID, B: b.ID, N: int(a.Ring.Dir), X: float32(at.X), Y: float32(at.Y)})
	}
	out := ResolveClash(g.Cfg.Sword,
		ClashSide{Realm: ra, Count: a.Ring.Count, Crack: a.Ring.Crack, DurMult: g.hardMult(a, "durability")},
		ClashSide{Realm: rb, Count: b.Ring.Count, Crack: b.Ring.Crack, DurMult: g.hardMult(b, "durability")})
	a.Ring.Crack, b.Ring.Crack = out.CrackA, out.CrackB
	g.breakSwords(a, out.LostA)
	g.breakSwords(b, out.LostB)
}

// pushApart knocks both players away from each other so their rings separate (spec §5.2).
func (g *Game) pushApart(a, b *Player) {
	sc := g.Cfg.Sword
	secs := math.Max(sc.ClashKnockbackSecs, 1e-3)
	away := b.Pos.Sub(a.Pos).Norm()
	if away.Len() == 0 {
		away = Vec{X: 1}
	}
	b.KnockVel, b.KnockUntil = away.Mul(sc.ClashKnockback/secs), g.Time+secs
	a.KnockVel, a.KnockUntil = away.Mul(-sc.ClashKnockback/secs), g.Time+secs
}

func (g *Game) breakSwords(p *Player, n int) {
	if n <= 0 {
		return
	}
	if p.HasStatus(StShell, g.Time) {
		return // đang rụt trong mai: không bị bể kiếm (spec §6.3)
	}
	lost := g.LoseSwords(p, n)
	if lost > 0 {
		p.breakPending += lost
		if p.Ring.Count == 0 {
			g.emote(p, "no_swords") // spec §5.2: bị tước sạch kiếm → hoảng loạn
		} else {
			g.emote(p, "sword_lost")
		}
	}
}

// flushSwordBreaks gộp số kiếm bể của mỗi người thành MỘT sự kiện mỗi swordBreakFlushSecs,
// kèm tổng số cây đã gãy. Client vẽ một vụ nổ to theo N thay vì N vụ nổ nhỏ.
func (g *Game) flushSwordBreaks() {
	for _, p := range g.PlayerList() {
		if p.breakPending <= 0 {
			continue
		}
		// Rụng sạch kiếm thì báo ngay, đó là khoảnh khắc đáng xem.
		if g.Time < p.breakFlushAt && p.Ring.Count > 0 {
			continue
		}
		g.emit(Event{Kind: "SwordBreak", A: p.ID, N: p.breakPending, X: float32(p.Pos.X), Y: float32(p.Pos.Y)})
		p.breakPending = 0
		p.breakFlushAt = g.Time + swordBreakFlushSecs
	}
}

// resolveSwordWorld applies spec §5.3: swords vs monsters and obstacles.
func (g *Game) resolveSwordWorld() {
	for _, a := range g.PlayerList() {
		if !g.canFight(a) {
			continue
		}
		g.hitMonsters(a)
		g.hitObstacles(a)
	}
}

func (g *Game) hitMonsters(a *Player) {
	outer := a.Geom.Outer()
	for _, m := range g.Monsters {
		if m.Pos.Dist(a.Pos) > outer+m.R {
			continue
		}
		hits := 0
		sweptHits(a.Pos, a.Ring, a.Geom, m.Pos, m.R, func(i int) bool {
			if g.cooldownReady(pairKey(tagMonster, a.ID, m.ID, i), g.Cfg.Monsters.HitCooldown) {
				hits++
			}
			return hits < m.HP
		})
		if hits == 0 {
			continue
		}
		// Cương Kiếm: mỗi lần chạm gây 2 damage thay vì 1 (spec v4.4 §5.3).
		hits = int(float64(hits) * g.hardMult(a, "damage"))
		m.HP -= hits
		m.FleeUntil, m.FleeFrom = g.Time+g.Cfg.Monsters.FleeSecs, a.Pos
		if m.HP <= 0 {
			g.killMonster(m, a)
			continue
		}
		g.emit(Event{Kind: "MonsterHurt", A: m.ID, N: m.HP, X: float32(m.Pos.X), Y: float32(m.Pos.Y)})
	}
}

// killMonster drops the monster's EXP as orbs reserved for the killer for orb_owner_secs (spec §5.3).
func (g *Game) killMonster(m *Monster, killer *Player) {
	delete(g.Monsters, m.ID)
	if m.Kind == "boss" {
		g.bossCount--
	}
	g.emit(Event{Kind: "MonsterDie", A: m.ID, S: m.Kind, X: float32(m.Pos.X), Y: float32(m.Pos.Y)})
	g.scatterOrbs(m.Pos, m.Exp, killer.ID)
	k := g.Cfg.Monsters.Kinds[m.Kind]
	if g.Rng.Float64() < k.Drop {
		for i := 0; i < k.DropItems; i++ {
			g.spawnItemNear(g.randomItemKind(), 0, m.Pos, m.R+40)
		}
	}
}

// ObstacleDamage is what one player does to an obstacle per hit (spec §5.3, v3.6): it scales
// with realm and ring count, NOT with the number of swords — otherwise 400 blades would
// shred a boulder instantly.
func (g *Game) ObstacleDamage(a *Player) float64 {
	rings := max(a.Geom.Rings, 1)
	if a.Ring.Count == 0 {
		return 0
	}
	// Cương Kiếm nhân đôi sức phá vật cản (spec v4.4 §5.3).
	return float64(g.Prog.Realm(a.Level)+1) * (1 + g.Cfg.Obstacles.DmgPerExtraRng*float64(rings-1)) * g.hardMult(a, "damage")
}

func (g *Game) hitObstacles(a *Player) {
	if a.Ring.Count == 0 {
		return
	}
	outer := a.Geom.Outer()
	n := sweepSteps(a.Ring)
	dmg := g.ObstacleDamage(a)
	g.QueryObstacles(a.Pos, outer, func(id int) {
		st := &g.Obstacles[id]
		o := g.Map.Obstacles[id]
		if o.MaxHP == 0 || st.Broken || !o.Shape.OverlapsCircle(a.Pos, outer) {
			return
		}
		// One damage tick per player per obstacle per dmg_cooldown_secs, however many
		// swords happen to be sweeping through it this tick.
		key := pairKey(tagObstacle, a.ID, uint32(id), 0)
		if g.onCooldown(key) {
			return
		}
		for i := 0; i < a.Ring.Count; i++ {
			if g.sweepHitsShape(a, i, n, o.Shape) {
				g.startCooldown(key, g.Cfg.Obstacles.DmgCooldown)
				g.damageObstacleBy(id, dmg)
				return
			}
		}
	})
}

// sweepHitsShape samples sword i along this tick's sweep against an obstacle's visual bounds
// (spec §8.2: swords hit what the player sees, bodies are blocked only by the footprint).
func (g *Game) sweepHitsShape(a *Player, i, n int, sh world.Shape) bool {
	for s := 0; s <= n; s++ {
		p, q := swordAt(a.Pos, sweepPhase(a.Ring, s, n), a.Ring, a.Geom, i)
		if sh.SegmentHits(p, q, a.Geom.Width/2) {
			return true
		}
	}
	return false
}

// damageObstacleBy applies fractional damage (spec §5.3 gives values like 13.5 per hit):
// the remainder is carried on the obstacle so nothing is silently rounded away.
func (g *Game) damageObstacleBy(id int, dmg float64) {
	if dmg <= 0 {
		return
	}
	st := &g.Obstacles[id]
	st.Partial += dmg
	whole := int(st.Partial + 1e-9)
	if whole <= 0 {
		return
	}
	st.Partial -= float64(whole)
	g.DamageObstacle(id, whole)
}

// DamageObstacle reduces an obstacle's HP and breaks it at zero (spec §8.2).
func (g *Game) DamageObstacle(id, dmg int) {
	st := &g.Obstacles[id]
	o := g.Map.Obstacles[id]
	if o.MaxHP == 0 || st.Broken {
		return
	}
	st.HP = max(0, st.HP-dmg)
	c := o.Shape.Center()
	if st.HP > 0 {
		g.emit(Event{Kind: "ObstacleDamaged", A: uint32(id), N: st.HP, X: float32(c.X), Y: float32(c.Y)})
		return
	}
	st.Broken, st.BrokenAt, st.Warned = true, g.Time, false
	g.gridDirty = true
	g.emit(Event{Kind: "ObstacleBroken", A: uint32(id), X: float32(c.X), Y: float32(c.Y)})
	if (o.Kind == "stall" || o.Kind == "cart") && g.Rng.Float64() < g.Cfg.Obstacles.StallItemDrop {
		g.spawnItemNear(g.randomItemKind(), 0, c, 30)
	}
}

// updateObstacles regrows broken obstacles once their area is clear (spec §8.2).
func (g *Game) updateObstacles() {
	oc := g.Cfg.Obstacles
	for id := range g.Obstacles {
		st := &g.Obstacles[id]
		if !st.Broken {
			continue
		}
		c := g.Map.Obstacles[id].Shape.Center()
		if !st.Warned && g.Time >= st.BrokenAt+oc.RegrowSecs-oc.RegrowWarnSecs {
			st.Warned = true
			g.emit(Event{Kind: "ObstacleRegrow", A: uint32(id), S: "warn", X: float32(c.X), Y: float32(c.Y)})
		}
		if g.Time < st.BrokenAt+oc.RegrowSecs || g.obstacleOccupied(id) {
			continue
		}
		st.Broken, st.HP, st.Partial = false, g.Map.Obstacles[id].MaxHP, 0
		g.gridDirty = true
		g.emit(Event{Kind: "ObstacleRegrow", A: uint32(id), X: float32(c.X), Y: float32(c.Y)})
	}
}

// obstacleOccupied reports whether any body or monster stands where obstacle id would regrow.
func (g *Game) obstacleOccupied(id int) bool {
	sh := g.Map.Obstacles[id].Block()
	for _, p := range g.Players {
		if !p.Dead && sh.OverlapsCircle(p.Pos, g.BodyR(p)) {
			return true
		}
	}
	for _, m := range g.Monsters {
		if sh.OverlapsCircle(m.Pos, m.R) {
			return true
		}
	}
	return false
}

// dropDeathSwords spills floor(n * death_sword_drop) swords around a corpse as bundles (spec §7).
func (g *Game) dropDeathSwords(b *Player) {
	n := int(math.Floor(float64(b.Ring.Count) * g.Cfg.Player.DeathSwordDrop))
	b.Ring.Count -= n
	g.dropBundles(b.Pos, n, b.Geom.Reach)
}

// dropBundles drops n swords as Bó Kiếm of at most drop_bundle_max swords each (spec §5.1).
func (g *Game) dropBundles(pos Vec, n int, spread float64) {
	per := max(g.Cfg.Sword.DropBundleMax, 1)
	for n > 0 {
		k := min(n, per)
		g.spawnItemNear("bundle", float64(k), pos, spread)
		n -= k
	}
}

// scatterOrbs drops total EXP as orbs around pos, largest sizes first. A non-zero owner
// reserves the orbs for that player for orb_owner_secs.
func (g *Game) scatterOrbs(pos Vec, total float64, owner uint32) {
	vals := g.Cfg.Items.OrbValues
	sizes := []float64{vals["large"], vals["medium"], vals["small"]}
	remaining := math.Round(total)
	for _, v := range sizes {
		for remaining >= v && v > 0 && g.orbCount < g.Cfg.Items.MaxOrbs {
			g.ownItem(g.spawnItemNear("orb", v, pos, g.Cfg.Items.ScatterR), owner)
			remaining -= v
		}
	}
	if remaining > 0 && g.orbCount < g.Cfg.Items.MaxOrbs {
		g.ownItem(g.spawnItemNear("orb", remaining, pos, g.Cfg.Items.ScatterR), owner)
	}
}

func (g *Game) ownItem(it *Item, owner uint32) {
	if owner != 0 {
		it.Owner, it.OwnerUntil = owner, g.Time+g.Cfg.Player.OrbOwnerSecs
	}
}
