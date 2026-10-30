package game

import (
	"math"
	"sort"

	"kiemxoay/server/internal/geom"
)

const spawnsPerTick = 20

// randomItemKind picks a spawnable item by the weights of spec §6.
func (g *Game) randomItemKind() string {
	return weightedPick(g.Rng.Intn, g.Cfg.Items.Weights)
}

// weightedPick chooses a key from weights; keys are sorted so a seed gives a stable result.
func weightedPick(intn func(int) int, weights map[string]int) string {
	keys := make([]string, 0, len(weights))
	total := 0
	for k, w := range weights {
		keys = append(keys, k)
		total += w
	}
	sort.Strings(keys)
	if total <= 0 {
		return keys[0]
	}
	roll := intn(total)
	for _, k := range keys {
		roll -= weights[k]
		if roll < 0 {
			return k
		}
	}
	return keys[len(keys)-1]
}

// spawnItemNear drops an item at a random spot within spread of pos.
func (g *Game) spawnItemNear(kind string, value float64, pos Vec, spread float64) *Item {
	off := geom.FromAngle(g.Rng.Float64() * 2 * math.Pi).Mul(spread * (0.4 + 0.6*g.Rng.Float64()))
	at := g.resolveBlocking(pos.Add(off), g.Cfg.Items.ItemR)
	return g.addItem(kind, value, at)
}

func (g *Game) addItem(kind string, value float64, at Vec) *Item {
	if kind == "bundle" && value <= 0 {
		value = float64(g.Cfg.Items.BundleAmount)
	}
	it := &Item{ID: g.newID(), Kind: kind, Pos: at, Value: value, Region: -1}
	g.Items[it.ID] = it
	g.takeSlot(it)
	g.itemGrid.add(it)
	if kind == "orb" {
		g.orbCount++
	}
	return it
}

// SpawnItemForTest thả một vật phẩm ở đúng vị trí cho trước; chỉ dùng trong test.
func (g *Game) SpawnItemForTest(kind string, value float64, at Vec) *Item {
	return g.addItem(kind, value, at)
}

func (g *Game) removeItem(it *Item) {
	delete(g.Items, it.ID)
	g.freeSlot(it)
	g.itemGrid.remove(it)
	switch {
	case it.Region >= 0:
		g.orbRespawns = append(g.orbRespawns, pendingOrb{at: g.Time + g.Cfg.ExpOrbs.RespawnSecs, region: it.Region})
	case it.Kind == "orb":
		g.orbCount--
	case it.Pool:
		g.poolCount--
		g.itemRespawns = append(g.itemRespawns, g.Time+g.Cfg.Items.RespawnSecs)
	default:
	}
}

// PickupRadius is pickup_r * scale, × Nam Châm when active (spec §6).
func (g *Game) PickupRadius(p *Player) float64 {
	r := g.Cfg.Player.PickupR * p.Scale
	if p.HasBuff(BuffMagnet, g.Time) {
		r *= g.Cfg.Buffs["magnet"].Mult
	}
	return r
}

// CanPickUp: người này có nhặt được gì không.
//
// Chặn ở ĐÂY, ngay cửa vào của `resolvePickups`, chứ không rải vào từng loại vật phẩm:
// spec v5.5 nói trúng Phun Trân Châu thì **vật phẩm, Linh khí, Tiên khí và Bí Kíp đều đi
// xuyên qua** — một cửa thì không thể sót loại nào, rải ra bốn chỗ thì chắc chắn sẽ sót.
func (g *Game) CanPickUp(p *Player) bool {
	if p.IsMinion {
		return false // đàn em không nhặt đồ (spec v5.4 §6.3)
	}
	if g.Cfg.Skills.TranChauNoPickup && p.HasStatus(StBlind, g.Time) {
		return false
	}
	return true
}

func (g *Game) resolvePickups() {
	for _, p := range g.PlayerList() {
		if p.Dead || p.InTransit(g.Time) || !g.CanPickUp(p) {
			continue
		}
		r := g.PickupRadius(p) + g.Cfg.Items.ItemR
		r2 := r * r
		var got []*Item
		g.ItemsNear(p.Pos, r, r, func(it *Item) {
			if it.Pos.Dist2(p.Pos) <= r2 && g.canAbsorb(p, it) && (it.Owner == 0 || it.Owner == p.ID || g.Time >= it.OwnerUntil) {
				got = append(got, it)
			}
		})
		for _, it := range got {
			g.pickup(p, it)
		}
	}
}

// canAbsorb: viên Linh khí và viên Tiên khí không lẫn sang nhau (§3.4). Người không hấp thụ
// được thì đi xuyên qua, viên vẫn nằm đó cho người dùng được tới nhặt.
func (g *Game) canAbsorb(p *Player, it *Item) bool {
	switch it.Kind {
	case "orb":
		return g.CanAbsorbExp(p)
	case ItemTienKhi:
		return g.CanAbsorbTienKhi(p)
	default:
		return true
	}
}

// ItemSkillScroll là vật phẩm Bí Kíp (spec §6.3).
const ItemSkillScroll = "skill_scroll"

// Viên Linh khí / Tiên khí **gộp thành một sự kiện mỗi người mỗi tick**, và chỉ gửi cho
// chính người nhặt. Hai lớp, vì một lớp không đủ:
//
//  1. Chỉ gửi cho người nhặt — đo được: trong gói snapshot lớn nhất, `ItemPicked` chiếm
//     **173 sự kiện / 8,1 KB / 74% gói**, do sáu mươi nhân vật hút viên liên tục và mỗi
//     lượt phát một sự kiện 48 byte cho mọi người trong tầm nhìn.
//  2. Nhưng lọc theo người xem vẫn chưa đủ: một người Đạo Tổ (scale 6, vòng nhặt 540) quét
//     qua bãi viên dày sinh ra **443 lượt nhặt của riêng mình trong một gói** — vẫn 21 KB.
//     Nên phải gộp lại, đúng cách `flushSwordBreaks` đã làm với kiếm gãy.
//
// Người ngoài không mất gì đáng kể: viên biến mất vẫn báo qua danh sách `ir` (5 byte), chỉ
// mất hiệu ứng "viên bay hút vào người khác" — một chấm nhỏ cách đó cả nghìn đơn vị. Vật
// phẩm hiếm (kiếm, rương, buff, Bí Kíp) vẫn phát nguyên vẹn cho cả phòng: chúng đáng xem
// và chỉ lác đác vài lượt mỗi phút.
func isOrbKind(kind string) bool { return kind == "orb" || kind == ItemTienKhi }

// flushOrbPickups phát một sự kiện gộp cho mỗi người vừa nhặt viên trong tick này.
//
// `B` mang id viên CUỐI CÙNG để client còn vẽ được một vệt bay hút, `N` là tổng số viên để
// client biết chỉ kêu một tiếng thay vì bốn trăm tiếng.
func (g *Game) flushOrbPickups() {
	for _, p := range g.PlayerList() {
		if p.orbPicked == 0 {
			continue
		}
		g.emit(Event{Kind: "ItemPicked", A: p.ID, B: p.orbPickedID, S: p.orbPickedKind,
			N: p.orbPicked, To: p.ID, X: float32(p.Pos.X), Y: float32(p.Pos.Y)})
		p.orbPicked, p.orbPickedID, p.orbPickedKind = 0, 0, ""
	}
}

func (g *Game) pickup(p *Player, it *Item) {
	g.removeItem(it)
	if isOrbKind(it.Kind) {
		p.orbPicked++
		p.orbPickedID, p.orbPickedKind = it.ID, it.Kind
	} else {
		g.emit(Event{Kind: "ItemPicked", A: p.ID, B: it.ID, S: it.Kind,
			X: float32(it.Pos.X), Y: float32(it.Pos.Y)})
	}
	switch it.Kind {
	case "orb":
		g.GainExp(p, it.Value, true)
	case "sword":
		g.AddSwords(p, g.Cfg.Items.SwordAmount)
	case "bundle":
		g.AddSwords(p, int(it.Value))
	case ItemTienKhi:
		// Chỉ người từ Đại Thừa tầng 10 trở lên nhặt được (canAbsorb đã lọc ở trên).
		g.GainTienKhi(p, it.Value)
		g.emit(Event{Kind: "TienKhiPicked", A: p.ID, To: p.ID, N: int(it.Value), S: "khi",
			X: float32(it.Pos.X), Y: float32(it.Pos.Y)})
	case ItemSkillScroll:
		// Bí Kíp: +1 lượt dùng kỹ năng của CHÍNH nhân vật mình (spec §6.3). Đủ 2 lượt rồi
		// thì không bỏ phí — đổi sang tu vi, giống luật quá tay của các vật phẩm khác.
		if g.AddSkillCharge(p) {
			g.emit(Event{Kind: "SkillCharge", A: p.ID, To: p.ID, N: p.SkillCharges,
				X: float32(p.Pos.X), Y: float32(p.Pos.Y)})
			break
		}
		if g.BanksTienKhi(p) {
			g.GainTienKhi(p, g.Cfg.Skills.OvercapTien)
		} else {
			g.GainExp(p, p.TotalExp+g.Cfg.Skills.OvercapExp, false)
		}
	case "chest":
		for i := 0; i < g.Cfg.Items.ChestItems; i++ {
			g.spawnItemNear(g.randomItemKind(), 0, it.Pos, g.Cfg.Items.ScatterR*2)
		}
		g.emote(p, "chest")
	default:
		g.ApplyBuff(p, it.Kind)
	}
}

// ApplyBuff starts or refreshes a timed buff; picking it again never stacks (spec §6).
func (g *Game) ApplyBuff(p *Player, name string) {
	b, ok := BuffByName(name)
	if !ok {
		return
	}
	cfg := g.Cfg.Buffs[name]
	secs := cfg.Secs
	if cfg.MaxSecs > 0 && secs > cfg.MaxSecs {
		secs = cfg.MaxSecs // Ẩn Thân: nhặt lại chỉ làm mới, không vượt max_secs (spec §6.2)
	}
	p.Buffs[b] = g.Time + secs
	g.refreshDerived(p)
	g.emit(Event{Kind: "BuffStart", A: p.ID, S: name, N: int(g.Cfg.Buffs[name].Secs * 1000), X: float32(p.Pos.X), Y: float32(p.Pos.Y)})
}

// updateBuffs expires buffs and emits BuffEnd.
func (g *Game) updateBuffs(p *Player) {
	changed := false
	for b := 0; b < BuffCount; b++ {
		if p.Buffs[b] != 0 && p.Buffs[b] <= g.Time {
			p.Buffs[b] = 0
			changed = true
			g.emit(Event{Kind: "BuffEnd", A: p.ID, S: BuffNames[b], X: float32(p.Pos.X), Y: float32(p.Pos.Y)})
		}
	}
	if changed {
		g.refreshDerived(p)
	}
}

// maintainSpawns keeps count_per_room pool items: the first fill is immediate, a picked item
// comes back respawn_secs later somewhere else (spec §6.2).
func (g *Game) maintainSpawns() {
	ic := g.Cfg.Items
	due := 0
	for len(g.itemRespawns) > due && g.itemRespawns[due] <= g.Time {
		due++
	}
	g.itemRespawns = g.itemRespawns[due:]
	missing := ic.CountPerRoom - g.poolCount - len(g.itemRespawns)
	for i := 0; i < min(spawnsPerTick, missing); i++ {
		it := g.addItem(g.randomItemKind(), 0, g.randomFreePoint(ic.ItemR, ic.SpawnMinDist, 10))
		it.Pool = true
		g.poolCount++
	}
	g.respawnOrbs()
	for i := 0; i < spawnsPerTick && len(g.Monsters) < g.Cfg.Room.MonsterCount; i++ {
		g.spawnMonster()
	}
	g.updateChest()
}

func (g *Game) spawnMonster() {
	weights := map[string]int{}
	for name, k := range g.Cfg.Monsters.Kinds {
		if k.Max > 0 && name == "boss" && g.bossCount >= k.Max {
			continue
		}
		weights[name] = k.Weight
	}
	kind := weightedPick(g.Rng.Intn, weights)
	k := g.Cfg.Monsters.Kinds[kind]
	m := &Monster{ID: g.newID(), Kind: kind, R: k.Radius, HP: k.HP, MaxHP: k.HP, Exp: k.Exp,
		Pos: g.randomFreePoint(k.Radius, g.Cfg.Items.SpawnMinDist, 10), Heading: g.Rng.Float64() * 2 * math.Pi}
	if kind == "boss" {
		g.bossCount++
	}
	g.Monsters[m.ID] = m
}

// updateChest announces a Rương Báu 10s early, then spawns it (spec §6).
func (g *Game) updateChest() {
	ic := g.Cfg.Items
	if g.chestAt == nil && g.Time >= g.chestNextAt-ic.ChestWarnSecs {
		at := g.randomFreePoint(ic.ItemR*2, ic.SpawnMinDist, 20)
		g.chestAt = &at
		g.emit(Event{Kind: "ChestIncoming", Global: true, N: int(ic.ChestWarnSecs), X: float32(at.X), Y: float32(at.Y)})
	}
	if g.chestAt != nil && g.Time >= g.chestNextAt {
		g.addItem("chest", 0, *g.chestAt)
		g.chestAt = nil
		g.chestNextAt += ic.ChestEverySecs
	}
}

func (g *Game) updateTimers() {
	for _, p := range g.PlayerList() {
		if p.Dead {
			continue
		}
		g.updateBuffs(p)
		g.updateTransit(p)
	}
	g.updateObstacles()
}
