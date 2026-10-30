package game

import (
	"math"
	"sort"

	"kiemxoay/server/internal/geom"
)

// MoveSpeed implements spec §4.1 without the zone factor.
// Tốc độ KHÔNG phụ thuộc tu vi; chỉ buff Thần Hành (nhặt được) và Thiêu Đốt Tinh Huyết
// (đốt EXP, §4.1c) mới tăng tốc.
func (g *Game) MoveSpeed(p *Player) float64 {
	s := g.Cfg.Player.MoveBase
	if p.HasBuff(BuffHaste, g.Time) {
		s *= g.Cfg.Buffs["haste"].Mult
	}
	if g.Burning(p) {
		s *= g.Cfg.Burn.Mult
	}
	// Kỹ năng (spec §6.3): Ru Ngủ làm chậm, Giao Hỏa Tốc làm nhanh, Lưới Mì trói chết chân.
	if p.HasStatus(StSlow, g.Time) {
		s *= g.Cfg.Skills.RuNguMove
	}
	if p.HasStatus(StGhost, g.Time) {
		s *= g.Cfg.Skills.GhostSpeed
	}
	if p.HasStatus(StRoot, g.Time) || p.HasStatus(StShell, g.Time) || p.HasStatus(StStun, g.Time) {
		s = 0 // bị trói, đang rụt trong mai, hoặc đang choáng
	}
	return s
}

// burnFloor is the lowest total_exp Thiêu Đốt Tinh Huyết may take a player down to: the
// start of their current realm, so burning can cost tiers but never a whole cảnh giới
// (rớt cảnh giới vẫn chỉ đến từ việc bị chém, §5.1).
func (g *Game) burnFloor(p *Player) float64 {
	if !g.Cfg.Burn.KeepRealm {
		return 0
	}
	return g.Prog.Threshold(g.Prog.Realm(p.Level) * g.Prog.Tiers())
}

// tienFloor is the burn floor on the Tiên khí path: the entry mark of the realm they are
// standing in, so burning never drops a Tiên giới player a whole cảnh giới (spec v5.6 §4.1c).
//
// Đại Thừa tầng 10 chưa vào Tiên giới nên đáy là 0 — tức là chưa nhặt được viên Tiên khí
// nào thì chưa đốt được, đúng như spec nói.
func (g *Game) tienFloor(p *Player) float64 {
	idx := g.TienRealmOf(p)
	if idx < 0 {
		return 0
	}
	return g.Prog.TienEntry(idx)
}

// tienBurnPerSec là lượng Tiên khí đốt mỗi giây: 8% **độ rộng khoảng** của cảnh giới đang
// đứng (spec v5.7 §4.1c) → Chân Tiên 8, Kim Tiên 16, Thái Ất 24, Đại La 32, Đạo Tổ 40.
//
// Theo độ rộng khoảng chứ không theo một con số chung: cảnh giới càng cao thì khoảng càng
// rộng, nên chạy nước rút luôn tốn đúng 8% quãng đường của chính cảnh giới ấy — người ở
// đỉnh không được chạy rẻ hơn người mới vào.
func (g *Game) tienBurnPerSec(p *Player) float64 {
	idx := g.TienRealmOf(p)
	if idx < 0 {
		return 0
	}
	return g.Cfg.Burn.TienFracPerSec * g.Prog.TienWidth(idx)
}

// Burning reports whether the player is actually burning essence right now: they asked for
// it, they are moving, and they still have cultivation left to spend (spec §4.1c).
//
// **Hai đường tu, hai loại nhiên liệu** (spec v5.6): người phàm đốt EXP, người đã sang Tiên
// khí đốt Tiên khí. Trước v5.6 chỉ có nhánh EXP, mà người Tiên giới đứng đúng ở đáy cảnh
// giới nên `TotalExp > burnFloor` luôn sai — lên Tiên giới là mất hẳn khả năng chạy nhanh
// (xem Q24, đã được trả lời bằng chính v5.6).
func (g *Game) Burning(p *Player) bool {
	if !p.Burning || !p.Moving || p.Dead || p.InTransit(g.Time) || g.Cfg.Burn.Mult <= 1 {
		return false
	}
	if p.HasStatus(StStun, g.Time) {
		return false // đang choáng thì không Thiêu Đốt được (spec v5.5 §6.3)
	}
	if g.BanksTienKhi(p) {
		return p.TienKhi > g.tienFloor(p)
	}
	return p.TotalExp > g.burnFloor(p)
}

// burnExp charges the cost of this tick's sprint (spec §4.1c).
func (g *Game) burnExp(p *Player, dt float64) {
	if !g.Burning(p) {
		return
	}
	if g.BanksTienKhi(p) {
		cost := g.tienBurnPerSec(p) * dt
		after := math.Max(g.tienFloor(p), p.TienKhi-cost)
		// Dùng chung `BurnedExp` cho cả hai đường tu: màn Tổng kết hỏi "đã thiêu đốt bao
		// nhiêu tu vi", và với người Tiên giới thì tu vi của họ chính là Tiên khí.
		p.BurnedExp += p.TienKhi - after
		p.TienKhi = after
		return
	}
	cost := g.Cfg.Burn.ExpFracPerSec * g.Prog.ExpNeeded(p.Level) * dt
	after := math.Max(g.burnFloor(p), p.TotalExp-cost)
	p.BurnedExp += p.TotalExp - after
	g.setTotalExp(p, after)
}

// ZoneMult returns the terrain speed factor for moving along dir at pos (spec §8.3).
func (g *Game) ZoneMult(pos Vec, dir Vec) float64 {
	mult := 1.0
	g.QueryZones(pos, 1, func(id int) {
		z := g.Map.Zones[id]
		if !z.Shape.Contains(pos) {
			return
		}
		switch z.Kind {
		case "mud", "water":
			// Ao/suối làm chậm như bùn (spec §8.6 mục 4); client vẽ mặt nước gợn sóng.
			mult *= g.Cfg.Zones.MudMult
		case "snow":
			// Bờ tuyết quanh hồ băng (spec §8.0, khu Băng Nguyệt Hồ).
			mult *= g.Cfg.Zones.SnowMult
		case "wind":
			mult *= g.windMult(dir, z.WindDir)
		default:
		}
	})
	return mult
}

// windMult: within wind_with_max_deg of the wind → with; beyond wind_against_min_deg → against;
// otherwise (crosswind) unchanged (spec §8.3).
func (g *Game) windMult(dir Vec, windDir float64) float64 {
	cos := geom.Clamp(dir.Dot(geom.FromAngle(windDir)), -1, 1)
	deg := math.Acos(cos) * 180 / math.Pi
	switch {
	case deg < g.Cfg.Zones.WindWithMaxDeg:
		return g.Cfg.Zones.WindWith
	case deg > g.Cfg.Zones.WindAgainstMinDeg:
		return g.Cfg.Zones.WindAgainst
	default:
		return 1
	}
}

// BushAt returns the id of the bush zone containing pos, or -1.
func (g *Game) BushAt(pos Vec) int {
	found := -1
	g.QueryZones(pos, 1, func(id int) {
		z := g.Map.Zones[id]
		if z.Kind == "bush" && z.Shape.Contains(pos) {
			found = id
		}
	})
	return found
}

func (g *Game) movePlayers(dt float64) {
	for _, p := range g.PlayerList() {
		if p.Dead || p.InTransit(g.Time) && !p.Transit.Arrived {
			continue
		}
		g.burnExp(p, dt) // trả giá trước, rồi mới đi với tốc độ đã cộng (§4.1c)
		g.movePlayer(p, dt)
		p.Bush = g.BushAt(p.Pos)
		g.checkGate(p)
	}
}

func (g *Game) movePlayer(p *Player, dt float64) {
	if p.Moving {
		p.Facing = p.InputAngle
	}
	// Trượt vì vệt trái cây của Cô Ba cũng dùng đúng cơ chế mặt băng (spec §6.3) — một
	// luật quán tính, hai nguồn gây ra.
	if g.OnIce(p.Pos) || p.HasStatus(StSlip, g.Time) {
		g.slideOnIce(p, dt)
	} else if p.Moving {
		dir := geom.FromAngle(p.InputAngle)
		p.Pos = p.Pos.Add(dir.Mul(g.MoveSpeed(p) * g.ZoneMult(p.Pos, dir) * dt))
		p.Vel = Vec{}
	} else {
		p.Vel = Vec{}
	}
	if p.KnockUntil > g.Time {
		p.Pos = p.Pos.Add(p.KnockVel.Mul(dt))
	}
	// Người chơi KHÔNG chặn đường nhau (spec §5.2 là luật chém, xem shield.go): đi tới gần
	// ai cũng vẫn điều khiển được bình thường, hai vòng kiếm chồng lên nhau mà mài.
	p.Pos = g.resolveBlockingFor(p, p.Pos, g.BodyR(p))
}

// OnIce: điểm này có nằm trên mặt băng không (spec §8.0, khu Băng Nguyệt Hồ).
func (g *Game) OnIce(pos Vec) bool {
	on := false
	g.QueryZones(pos, 1, func(id int) {
		if z := g.Map.Zones[id]; z.Kind == "ice" && z.Shape.Contains(pos) {
			on = true
		}
	})
	return on
}

// slideOnIce di chuyển theo QUÁN TÍNH (spec §8.0):
//
//	vận tốc mới = vận tốc cũ × friction + hướng input × accel × dt
//
// Khác hẳn trên đất, nơi thả phím là dừng ngay. Trên băng, thả phím thì vẫn trôi tiếp, và
// đổi hướng phải "lái" chứ không bẻ ngoặt được — đó là cái khó mà khu này bán.
//
// Hàm này phải khớp từng con số với `iceStep` bên client (spec §8.2a: dùng chung một thuật
// toán), nếu không server sẽ kéo nhân vật về mỗi tick suốt thời gian ở trên hồ.
func (g *Game) slideOnIce(p *Player, dt float64) {
	ic := g.Cfg.Ice
	p.Vel = p.Vel.Mul(ic.Friction)
	if p.Moving {
		// Buff tốc và Thiêu Đốt vẫn ăn vào gia tốc, không ăn vào ma sát: trên băng người
		// nhanh hơn thì tăng tốc nhanh hơn, chứ không phải dừng giỏi hơn.
		boost := g.MoveSpeed(p) / math.Max(g.Cfg.Player.MoveBase, 1e-9)
		p.Vel = p.Vel.Add(geom.FromAngle(p.InputAngle).Mul(ic.Accel * boost * dt))
	}
	p.Pos = p.Pos.Add(p.Vel.Mul(dt))
}

// resolveBlocking pushes a circle out of solid obstacles and keeps it inside the playable
// area — never in the void outside the border wall (spec §8.1).
//
// Trước đây chỗ này chỉ kẹp vào [r, size-r] tức cả dải lề ngoài tường biên, nên một cú
// knockback hay một lần PushOut chọn nhầm mặt ngoài của tường là người chơi đứng luôn bên
// ngoài bản đồ và không có gì kéo họ vào lại.
// resolveBlockingFor là resolveBlocking có xét kỹ năng.
//
// Giao Hỏa Tốc (spec v5.5 §6.3) đi xuyên **mọi** vật cản — kể cả vách đá, rặng tre, tường
// không phá được. Thứ duy nhất còn chặn là **biên khu**: cho xuyên luôn biên thì Shipper ra
// đứng ngoài bản đồ, chỗ không có đất, không có viên, và không gì kéo họ vào lại.
//
// Bản v5.0 chỉ cho xuyên vật cản phá được, nên phải lọc theo `MaxHP == 0`; giờ không còn
// phân biệt nữa.
func (g *Game) resolveBlockingFor(p *Player, pos Vec, r float64) Vec {
	if p.unstickUntil > g.Time {
		// Đang được đẩy mượt ra khỏi vật cản: vị trí do `updateUnstick` quyết, đẩy ra ở đây
		// nữa là bật ra tức thời — đúng cái giật mà spec bảo tránh.
		return g.clampToPlay(pos, r)
	}
	if !p.HasStatus(StGhost, g.Time) {
		return g.resolveBlocking(pos, r)
	}
	return g.clampToPlay(pos, r)
}

func (g *Game) resolveBlocking(pos Vec, r float64) Vec {
	pos = g.clampToPlay(pos, r)
	for iter := 0; iter < 3; iter++ {
		// Đẩy ra theo THỨ TỰ ID TĂNG DẦN, không theo thứ tự lưới không gian trả về.
		//
		// Đẩy lần lượt ra khỏi nhiều vật cản là phép KHÔNG GIAO HOÁN: đứng ở góc hai vật
		// cản, đẩy A trước rồi B cho ra chỗ khác hẳn đẩy B trước rồi A. Lưới không gian
		// duyệt theo ô nên thứ tự phụ thuộc vị trí, còn client duyệt theo mảng — hai bên ra
		// hai kết quả khác nhau. Bài đối chiếu 1 000 bước (cmd/collisionfixture) bắt được
		// 4 ca lệch, ca nặng nhất **173,7 đơn vị**, đủ để người chơi thấy giật nảy.
		ids := g.obsBuf[:0]
		g.QueryObstacles(pos, r, func(id int) {
			if g.ObstacleSolid(id) {
				ids = append(ids, id)
			}
		})
		g.obsBuf = ids
		sort.Ints(ids)
		moved := false
		for _, id := range ids {
			if np, hit := g.Map.Obstacles[id].Block().PushOut(pos, r); hit {
				pos, moved = np, true
			}
		}
		if !moved {
			break
		}
	}
	return g.clampToPlay(pos, r)
}

// MoveStepForTest chạy đúng MỘT bước di chuyển như `movePlayer` cho công cụ đối chiếu
// client/server (cmd/collisionfixture). Không dùng trong luật chơi.
//
// Phải đi qua cùng đường với movePlayer — tốc độ gốc, nhân theo vùng địa hình, rồi đẩy ra
// khỏi vật cản — nếu không thì bài đối chiếu sẽ bỏ sót đúng những chỗ hai bên hay lệch.
func (g *Game) MoveStepForTest(from, vel Vec, angle, scale, dt float64, moving, ghost bool) (Vec, Vec) {
	p := &Player{Pos: from, Vel: vel, InputAngle: angle, Moving: moving, Scale: scale}
	p.Level = 0
	if ghost {
		p.Status[StGhost] = g.Time + 1
	}
	g.movePlayerForTest(p, dt)
	return p.Pos, p.Vel
}

// movePlayerForTest chạy đúng phần DI CHUYỂN của movePlayer (không có đẩy văng, không có
// va kiếm) để công cụ đối chiếu so với client.
func (g *Game) movePlayerForTest(p *Player, dt float64) {
	// Trượt vì vệt trái cây của Cô Ba cũng dùng đúng cơ chế mặt băng (spec §6.3) — một
	// luật quán tính, hai nguồn gây ra.
	if g.OnIce(p.Pos) || p.HasStatus(StSlip, g.Time) {
		g.slideOnIce(p, dt)
	} else if p.Moving {
		dir := geom.FromAngle(p.InputAngle)
		// Giao Hỏa Tốc nhân tốc độ — phần này cũng phải có ở bài đối chiếu, nếu không
		// client dự đoán Shipper chạy ×3 mà bài kiểm vẫn báo "đạt".
		speed := g.Cfg.Player.MoveBase
		if p.HasStatus(StGhost, g.Time) {
			speed *= g.Cfg.Skills.GhostSpeed
		}
		p.Pos = p.Pos.Add(dir.Mul(speed * g.ZoneMult(p.Pos, dir) * dt))
		p.Vel = Vec{}
	} else {
		p.Vel = Vec{}
	}
	p.Pos = g.resolveBlockingFor(p, p.Pos, g.Cfg.Player.BodyR*p.Scale)
}

// clampToPlay keeps a circle of radius r inside the border wall.
func (g *Game) clampToPlay(pos Vec, r float64) Vec {
	lo, hi := g.PlayBounds(r)
	if hi < lo { // bản đồ bé hơn hai lần lề: lùi về kẹp theo cả bản đồ
		lo, hi = r, g.Map.Size-r
	}
	return Vec{X: geom.Clamp(pos.X, lo, hi), Y: geom.Clamp(pos.Y, lo, hi)}
}

// circleFree reports whether a circle overlaps no solid obstacle.
func (g *Game) circleFree(pos Vec, r float64) bool {
	free := true
	g.QueryObstacles(pos, r, func(id int) {
		if g.ObstacleSolid(id) && g.Map.Obstacles[id].Block().OverlapsCircle(pos, r) {
			free = false
		}
	})
	return free
}

// PlayBounds is the area anything may spawn in: inside the border wall, never in the void
// outside it (spec §8.1).
func (g *Game) PlayBounds(pad float64) (lo, hi float64) {
	m := g.Map.PlayMargin + pad
	return m, g.Map.Size - m
}

// randomFreePoint returns a random point not inside obstacles, preferring far from players.
func (g *Game) randomFreePoint(r, minPlayerDist float64, tries int) Vec {
	best, bestD := Vec{X: g.Map.Size / 2, Y: g.Map.Size / 2}, -1.0
	lo, hi := g.PlayBounds(r + 50)
	for i := 0; i < tries; i++ {
		p := Vec{X: lo + g.Rng.Float64()*(hi-lo), Y: lo + g.Rng.Float64()*(hi-lo)}
		if !g.circleFree(p, r) {
			continue
		}
		d := g.nearestPlayerDist(p)
		if d >= minPlayerDist {
			return p
		}
		if d > bestD {
			best, bestD = p, d
		}
	}
	return best
}

func (g *Game) nearestPlayerDist(p Vec) float64 {
	best := math.Inf(1)
	for _, o := range g.Players {
		if !o.Dead {
			best = math.Min(best, o.Pos.Dist(p))
		}
	}
	return best
}

func (g *Game) safeSpawn() Vec {
	return g.randomFreePoint(g.Cfg.Player.BodyR*2, g.Cfg.Room.SafeSpawnR, 60)
}

func (g *Game) moveMonsters(dt float64) {
	mc := g.Cfg.Monsters
	for _, m := range g.Monsters {
		speed := mc.WanderSpeed
		if m.FleeUntil > g.Time {
			m.Heading = m.Pos.Sub(m.FleeFrom).Angle()
			speed = mc.FleeSpeed
		} else if g.Time >= m.TurnAt {
			m.Heading = g.Rng.Float64() * 2 * math.Pi
			m.TurnAt = g.Time + 2 + g.Rng.Float64()*3
		}
		next := g.resolveBlocking(m.Pos.Add(geom.FromAngle(m.Heading).Mul(speed*dt)), m.R)
		if next.Dist2(m.Pos) < 1e-6 {
			m.TurnAt = g.Time // stuck against a wall: pick a new heading next tick
		}
		m.Pos = next
	}
}
