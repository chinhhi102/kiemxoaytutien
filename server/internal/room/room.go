// Package room runs one goroutine per room: tick, bots, snapshots, matchmaking (spec §9, §13.3).
package room

import (
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/netproto"
	"kiemxoay/server/internal/store"
	"kiemxoay/server/internal/world"
)

// Recorder persists finished human lives (implemented by store.Store).
type Recorder interface {
	Record(l store.Life) error
}

type pendingRespawn struct {
	id uint32
	at float64
}

// Room là MỘT thế giới gồm nhiều khu vực (spec §8.0). Mỗi khu có bản đồ riêng, mô phỏng
// riêng, người chơi riêng; đi giữa các khu chỉ bằng cổng dịch chuyển (§8.4).
//
// Những gì tính chung cả thế giới nằm ở đây: bảng xếp hạng phòng, danh hiệu, số bot cần
// giữ, hàng chờ hồi sinh. Những gì thuộc về một khu nằm trong Zone.
type Room struct {
	ID      string
	Created time.Time
	cfg     *config.Config
	zones   []*Zone
	mgr     *Manager
	inbox   chan func()
	// transfers: người chờ sang khu khác, gom lại rồi chuyển một lượt ở RANH GIỚI TICK —
	// lúc không khu nào đang chạy, nên không cần khoá (xem Zone).
	transfers []transfer
	recorder  Recorder
	rng       *rand.Rand
	botTimer  float64
	// recentNames: tên vừa rời phòng → thời điểm được phép dùng lại (spec §9.4).
	recentNames map[string]float64
	lbTimer     float64
	emptyFor    float64
	reserved    int // humans counted by the manager (guarded by Manager.mu)
	closed      bool
	snapEvery   float64
	// rankAt: hạng của từng người, chụp MỘT LẦN đầu mỗi tick.
	//
	// Bảng xếp hạng tính chung cả thế giới, nên tính nó phải đọc người chơi của mọi khu —
	// làm việc đó trong goroutine của một khu là đọc dữ liệu đúng lúc khu khác đang sửa.
	// Hạng trên bảng Tổng kết chậm một tick thì không ai nhận ra; đua dữ liệu thì có.
	rankAt map[uint32]int
}

// transfer: chuyển một nhân vật sang khu khác ở ranh giới tick, giữ nguyên mọi trạng thái.
type transfer struct {
	playerID uint32
	from     *Zone
	to       *Zone
	at       game.Vec
	destGate int
}

func newRoom(id string, cfg *config.Config, maps []*world.Map, mgr *Manager, rec Recorder, seed int64) *Room {
	r := &Room{
		ID: id, Created: time.Now(), cfg: cfg, mgr: mgr,
		inbox:    make(chan func(), 1024),
		recorder: rec, rng: rand.New(rand.NewSource(seed + 1)),
		snapEvery: 1 / float64(cfg.Net.SnapshotHz),
	}
	for i, m := range maps {
		r.zones = append(r.zones, newZone(i, cfg, m, seed+int64(i)*7919))
	}
	// Nối 20 cổng của cả thế giới thành MỘT mạng chung (spec §8.4). Khu không biết gì về
	// khu khác; phòng là nơi duy nhất nhìn thấy toàn bộ, nên nó làm người chỉ đường.
	for _, z := range r.zones {
		z.G.ZoneIndex = z.Index
		z.G.GateRouter = r.routeGate
		z.G.MoveToZone = r.moveFromZone(z)
	}
	return r
}

// Zones là danh sách khu của phòng (chỉ đọc, dùng cho HUD và công cụ đo).
func (r *Room) Zones() []*Zone { return r.zones }

// Zone trả khu thứ i (test và công cụ đo dùng).
func (r *Room) Zone(i int) *Zone { return r.zones[i] }

// ZoneOf trả khu mà nhân vật này đang ở — test và công cụ đo dùng, vì người chơi được thả
// vào khu ít người nhất nên không đoán trước được họ ở đâu.
func (r *Room) ZoneOf(id uint32) *Zone { return r.zoneOf(id) }

// Ranking xếp hạng CHUNG CẢ THẾ GIỚI (spec §8.0: bảng xếp hạng phòng và danh hiệu tính
// chung, không theo khu) — gộp người của mọi khu rồi xếp theo Uy Danh, hoà thì theo EXP.
func (r *Room) Ranking() []*game.Player {
	if len(r.zones) == 1 {
		return r.zones[0].G.Ranking()
	}
	var out []*game.Player
	for _, z := range r.zones {
		out = append(out, z.G.Ranking()...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].TotalExp > out[j].TotalExp
	})
	return out
}

// routeGate chọn cổng ra trong MẠNG CHUNG của cả thế giới (spec §8.4): bước vào một cổng
// thì ra ở một cổng bất kỳ khác — có thể cùng khu, có thể khu khác — nhưng không bao giờ
// ra lại chính cổng vừa vào.
//
// Với 20 cổng chia đều 5 khu, khoảng 80% lần dịch chuyển sẽ sang khu khác, đúng như spec
// mong muốn: cổng là đường duy nhất đi giữa các khu.
func (r *Room) routeGate(fromZone int, gate world.Gate) (game.GateDest, bool) {
	// **Thiên Môn** (spec §8.4a): một chiều, luôn ra ở Bến Thiên Môn cố định của khu mình
	// tại Thiên Đình. Đây là lối đi CHẮC CHẮN cho người muốn leo cao, ngược hẳn với 4 cổng
	// thường vốn cố tình giấu đích đến.
	if gate.Heaven && len(r.zones) > 1 {
		heaven := r.zones[0]
		if dock := heaven.G.Map.DockFor(fromZone); dock != nil {
			return game.GateDest{Zone: heaven.Index, GateID: dock.ID, At: game.Vec{X: dock.X, Y: dock.Y}}, true
		}
		// Khu Thiên Đình chưa có bến cho khu này: thả vào giữa khu còn hơn nuốt mất người.
		return game.GateDest{Zone: heaven.Index, GateID: -1,
			At: game.Vec{X: heaven.G.Map.Size / 2, Y: heaven.G.Map.Size / 2}}, true
	}
	if !r.cfg.Gates.RandomExit || len(r.zones) <= 1 {
		return game.GateDest{}, false // để khu tự xử theo luật cũ
	}
	type cand struct {
		zone int
		g    world.Gate
	}
	pool := make([]cand, 0, len(r.zones)*4)
	for _, z := range r.zones {
		for _, o := range z.G.Map.Gates {
			if z.Index == fromZone && o.ID == gate.ID {
				continue // không ra lại chính cổng vừa bước vào
			}
			if o.Heaven || o.Dock {
				continue // Thiên Môn và Bến Thiên Môn không nằm trong bể ngẫu nhiên (§8.4a)
			}
			if r.cfg.Gates.ExitPool == "other_zone" && z.Index == fromZone {
				continue
			}
			pool = append(pool, cand{z.Index, o})
		}
	}
	if len(pool) == 0 {
		return game.GateDest{}, false
	}
	// Dùng rng CỦA KHU gọi tới, không dùng rng của phòng: năm khu chạy song song.
	c := pool[r.zones[fromZone].rng.Intn(len(pool))]
	return game.GateDest{Zone: c.zone, GateID: c.g.ID, At: game.Vec{X: c.g.X, Y: c.g.Y}}, true
}

// moveFromZone dựng hàm "chuyển nhân vật sang khu khác" gắn CỨNG với khu gọi nó; việc bê
// thật làm ở ranh giới tick.
//
// Khu nguồn phải là tham số, không được đi tìm. Bản trước gọi `zoneOf(p.ID)` để tìm, mà
// `zoneOf` quét map `Players` của **mọi khu** — trong khi hàm này chạy từ trong goroutine
// của một khu, lúc bốn khu kia đang tick và có thể đang xoá người khỏi map của chúng. Đó là
// đọc map song song với ghi map: `go test -race` bắt được, và ngoài đời là
// `fatal error: concurrent map read and map write` — sập cả server, không phải một lỗi nhỏ.
//
// Mà cũng không cần tìm: hàm này do khu nào cài thì chạy cho đúng người của khu ấy.
func (r *Room) moveFromZone(from *Zone) func(*game.Player, int, game.Vec) {
	return func(p *game.Player, zone int, at game.Vec) {
		if zone < 0 || zone >= len(r.zones) || zone == from.Index {
			return
		}
		r.queueTransfer(transfer{playerID: p.ID, from: from, to: r.zones[zone], at: at})
	}
}

// Send chuyển một nhân vật sang khu khác ở ranh giới tick (spec §8.0, §8.4). Mọi trạng thái
// — kiếm, buff, nứt, Tiên khí, Uy Danh — giữ nguyên; chỉ đổi bản đồ và vị trí.
func (r *Room) queueTransfer(t transfer) { t.from.transfers = append(t.from.transfers, t) }

// applyTransfers chạy SAU khi mọi khu đã tick xong, nên không khu nào đang đọc dữ liệu của
// mình lúc nhân vật bị bê đi.
func (r *Room) applyTransfers() {
	r.transfers = r.transfers[:0]
	for _, z := range r.zones {
		r.transfers = append(r.transfers, z.transfers...)
		z.transfers = z.transfers[:0]
	}
	if len(r.transfers) == 0 {
		return
	}
	for _, t := range r.transfers {
		p, ok := t.from.G.Players[t.playerID]
		if !ok {
			continue
		}
		c := t.from.clients[t.playerID]
		b := t.from.brains[t.playerID]
		t.from.G.Detach(t.playerID)
		delete(t.from.clients, t.playerID)
		delete(t.from.brains, t.playerID)

		p.Pos = t.at
		t.to.G.Attach(p)
		if c != nil {
			t.to.clients[t.playerID] = c
			c.resetKnown()
			c.send(r.welcomeFor(t.to, p, c))
		}
		if b != nil {
			t.to.brains[t.playerID] = b
		}
	}
	r.transfers = r.transfers[:0]
}

// zoneOf trả về khu mà nhân vật này đang ở, nil nếu không còn trong phòng.
//
// ⚠ Quét map `Players` của MỌI khu, nên **chỉ được gọi ngoài lúc các khu đang tick** — tức
// là từ trong `r.Do(...)` hoặc sau `wg.Wait()`. Gọi nó từ trong goroutine của một khu là
// đọc map song song với khu khác đang ghi (đã dính đúng lỗi này một lần, xem `moveFromZone`).
func (r *Room) zoneOf(id uint32) *Zone {
	for _, z := range r.zones {
		if _, ok := z.G.Players[id]; ok {
			return z
		}
	}
	return nil
}

// thinnestZone chọn khu ÍT NGƯỜI THẬT nhất để thả người mới vào (spec §8.0); hoà thì bốc
// ngẫu nhiên để hai người vào cùng lúc không dồn hết vào một khu.
func (r *Room) thinnestZone() *Zone {
	best := []*Zone{}
	low := 1 << 30
	for _, z := range r.zones {
		switch n := z.Humans(); {
		case n < low:
			low, best = n, []*Zone{z}
		case n == low:
			best = append(best, z)
		}
	}
	if len(best) == 0 {
		return nil
	}
	return best[r.rng.Intn(len(best))]
}

// Do runs fn on the room goroutine.
func (r *Room) Do(fn func()) {
	select {
	case r.inbox <- fn:
	default:
		log.Printf("room %s inbox full, dropping message", r.ID)
	}
}

// Run ticks the room until it closes (no humans for empty_close_secs).
func (r *Room) Run() {
	dt := 1 / float64(r.cfg.Net.TickHz)
	t := time.NewTicker(time.Duration(float64(time.Second) * dt))
	defer t.Stop()
	for range t.C {
		r.Step(dt)
		if r.closed {
			return
		}
	}
}

// Step advances one tick; exported so cmd/sim can run rooms without real time.
// Pong trả lời ping ngay lập tức, đúng như cmd/server/ws.go làm trong handler WebSocket —
// để cmd/loadtest đo được ping không dính thời gian chờ vòng lặp (spec §13.6).
func (r *Room) Pong(c *Client, stamp float64) {
	if b, err := netproto.Encode(netproto.Pong{T: "pong", Client: stamp, Server: float64(time.Now().UnixMilli())}); err == nil {
		c.Conn.Send(b)
	}
}

// Clients là danh sách người đang trong phòng (gộp mọi khu); cmd/stress dùng để bơm input.
func (r *Room) Clients() []*Client {
	out := []*Client{}
	for _, z := range r.zones {
		for _, c := range z.clients {
			out = append(out, c)
		}
	}
	return out
}

// slowStepMs: một tick lâu hơn ngần này thì in ra xem thời gian đi đâu. Ngân sách §13.6 là
// 4 ms; đặt ngưỡng cao hơn hẳn để chỉ bắt những lần thật sự bất thường.
const slowStepMs = 25.0

func (r *Room) Step(dt float64) {
	stepStart := time.Now()
	r.drainInbox()
	r.snapshotRanks()
	// Mỗi khu một GOROUTINE (spec v5 mục 4). An toàn vì trong một tick không khu nào đọc
	// dữ liệu của khu khác: việc duy nhất đi qua ranh giới là chuyển người, và nó gom vào
	// hàng chờ riêng của từng khu rồi chạy SAU khi mọi khu đã xong.
	if len(r.zones) == 1 {
		r.stepZone(r.zones[0], dt)
	} else {
		var wg sync.WaitGroup
		for _, z := range r.zones {
			wg.Add(1)
			go func(z *Zone) {
				defer wg.Done()
				r.stepZone(z, dt)
			}(z)
		}
		wg.Wait()
	}
	r.flushFeeds()
	r.applyTransfers()
	r.updateBots(dt)
	r.lbTimer += dt
	if r.lbTimer >= 1/float64(r.cfg.Net.LeaderboardHz) {
		r.lbTimer = 0
		r.sendLeaderboards()
	}
	r.checkEmpty(dt)
	if ms := float64(time.Since(stepStart).Microseconds()) / 1000; ms > slowStepMs {
		log.Printf("tick chậm %.1f ms cả phòng (%d khu, %d người, %d bot)", ms, len(r.zones), r.Humans(), r.Bots())
	}
}

// stepZone chạy một tick cho MỘT khu.
func (r *Room) stepZone(z *Zone, dt float64) {
	stepStart := time.Now()
	r.applyInputs(z)
	tIn := time.Now()
	for _, b := range z.brains {
		b.Update(z.G)
	}
	tBrains := time.Now()
	z.G.Tick(dt)
	tGame := time.Now()
	z.tick++
	z.pending = append(z.pending, z.G.DrainEvents()...)
	r.handleDeaths(z)
	tDeaths := time.Now()
	z.snapAcc += dt
	if z.snapAcc >= r.snapEvery-1e-9 {
		z.snapAcc -= r.snapEvery
		r.broadcast(z)
	}
	tCast := time.Now()
	if ms := float64(tCast.Sub(stepStart).Microseconds()) / 1000; ms > slowStepMs {
		log.Printf("tick chậm %.1f ms ở khu %s: input %.1f · bot %.1f (%d) · game %.1f · chết %.1f · gửi %.1f (%d người)",
			ms, z.Name, float64(tIn.Sub(stepStart).Microseconds())/1000,
			float64(tBrains.Sub(tIn).Microseconds())/1000, len(z.brains),
			float64(tGame.Sub(tBrains).Microseconds())/1000,
			float64(tDeaths.Sub(tGame).Microseconds())/1000,
			float64(tCast.Sub(tDeaths).Microseconds())/1000, len(z.clients))
	}
	// Trung bình trượt để con số trên overlay không nhảy loạn (spec §13.6).
	z.stepMs = z.stepMs*0.9 + float32(time.Since(stepStart).Microseconds())/1000*0.1
}

func (r *Room) drainInbox() {
	for {
		select {
		case fn := <-r.inbox:
			fn()
		default:
			return
		}
	}
}

func (r *Room) applyInputs(z *Zone) {
	for id, c := range z.clients {
		seq, ang, mov, burn, skill, ok := c.takeInput()
		if !ok {
			continue
		}
		z.G.SetInput(id, seq, ang, mov, burn)
		if skill {
			if p, alive := z.G.Players[id]; alive {
				z.G.CastSkill(p)
			}
		}
	}
}

// addHuman spawns the client's player at Luyện Khí 1 with 1 sword (spec §7).
func (r *Room) addHuman(c *Client, j netproto.Join) {
	// Bảng phím test chỉ hiện cho người tự mở bằng ?debug=true (và server phải cho phép).
	c.Debug = c.Debug || (r.mgr != nil && r.mgr.AllowDebug && j.Debug)
	// Vào trận ở khu ÍT NGƯỜI THẬT NHẤT (spec §8.0) — cả điểm của việc có 5 khu là để
	// người mới không vừa vào đã bị dí.
	z := r.thinnestZone()
	p := z.G.AddPlayer(game.SpawnOptions{Name: j.Name, Char: j.Char, GuestID: j.GuestID, Swords: r.cfg.Sword.StartCount})
	c.PlayerID = p.ID
	c.resetKnown()
	z.clients[p.ID] = c
	c.setRoom(r)
	c.send(r.welcomeFor(z, p, c))
}

// welcomeFor dựng gói welcome cho một nhân vật ở một khu. Gọi cả lúc vào trận lẫn lúc
// chuyển khu — sang khu khác là đổi hẳn bản đồ, client phải tải lại từ đầu.
func (r *Room) welcomeFor(z *Zone, p *game.Player, c *Client) netproto.Welcome {
	w := netproto.Welcome{T: "welcome", ID: p.ID, Room: r.ID, Map: z.G.Map.ID,
		TickHz: r.cfg.Net.TickHz, Time: z.G.Time, Debug: c.Debug, Proto: netproto.ProtocolVersion,
		Zone: z.Index, ZoneName: z.Name}
	for id, st := range z.G.Obstacles {
		if st.Broken || st.HP != z.G.Map.Obstacles[id].MaxHP {
			w.Obstacles = append(w.Obstacles, netproto.ObstacleState{ID: id, HP: st.HP, Broken: st.Broken})
		}
	}
	return w
}

// RemoveClient is called when a connection closes.
func (r *Room) RemoveClient(c *Client) {
	r.Do(func() {
		z := r.zoneOf(c.PlayerID)
		if z == nil {
			return
		}
		if _, ok := z.clients[c.PlayerID]; !ok {
			return
		}
		if p, ok := z.G.Players[c.PlayerID]; ok && !p.Dead {
			r.record(z, p)
		}
		r.detach(z, c)
	})
}

func (r *Room) detach(z *Zone, c *Client) {
	z.G.RemovePlayer(c.PlayerID)
	delete(z.clients, c.PlayerID)
	c.PlayerID = 0
	c.setRoom(nil)
	r.mgr.release(r)
}

// handleDeaths sends the Tổng kết screen after the death animation and records the life.
func (r *Room) handleDeaths(z *Zone) {
	for _, p := range z.G.PlayerList() {
		if !p.Dead || z.G.Time-p.DiedAt < r.cfg.Player.DeathAnimSecs {
			continue
		}
		if c, ok := z.clients[p.ID]; ok {
			rank := r.rankOf(p)
			r.record(z, p)
			c.send(netproto.Dead{T: "dead", KillerName: p.KillerName, KillerChar: p.KillerChar, PeakLevel: p.PeakLevel,
				PeakExp: p.PeakExp, Kills: p.Kills, SurviveSec: p.DiedAt - p.SpawnedAt, Rank: rank,
				Score: p.Score, Hits: p.Hits, ScoreBy: p.ScoreBy, BurnedExp: p.BurnedExp,
				DaiThuaSecs: p.DaiThuaSecs, Top1Secs: p.Top1Secs})
			r.detach(z, c)
			continue
		}
		if b, ok := z.brains[p.ID]; ok && !z.respawnQueued(b.ID) {
			r.onBotDeath(z, p)
		}
	}
}

func (r *Room) record(z *Zone, p *game.Player) {
	if p.IsBot || r.recorder == nil {
		return
	}
	end := p.DiedAt
	if !p.Dead {
		end = z.G.Time
	}
	err := r.recorder.Record(store.Life{GuestID: p.GuestID, Name: p.Name, Character: p.Char,
		Score: p.Score, Hits: p.Hits, PeakExp: p.PeakExp, PeakLevel: p.PeakLevel, Kills: p.Kills,
		DaiThuaSecs: p.DaiThuaSecs, Top1Secs: p.Top1Secs,
		SurviveSecs: end - p.SpawnedAt, Map: z.G.Map.ID, EndedAt: time.Now().Unix()})
	if err != nil {
		log.Printf("record life: %v", err)
	}
}

// snapshotRanks chụp hạng của mọi người trong phòng, chạy một luồng ở đầu tick.
func (r *Room) snapshotRanks() {
	if r.rankAt == nil {
		r.rankAt = map[uint32]int{}
	}
	clear(r.rankAt)
	for i, p := range r.Ranking() {
		r.rankAt[p.ID] = i + 1
	}
}

// rankOf is the player's 1-based rank in the room board (spec §11.2: Uy Danh, then exp).
// Xếp hạng tính CHUNG CẢ THẾ GIỚI, không theo khu (spec §8.0); đọc từ bản chụp đầu tick.
func (r *Room) rankOf(p *game.Player) int {
	if rank, ok := r.rankAt[p.ID]; ok {
		return rank
	}
	return len(r.rankAt) + 1
}

func (r *Room) checkEmpty(dt float64) {
	if r.Humans() > 0 || r.mgr.humansOf(r) > 0 {
		r.emptyFor = 0
		return
	}
	r.emptyFor += dt
	if r.emptyFor >= r.cfg.Room.EmptyCloseSecs && r.mgr.tryClose(r) {
		r.closed = true
	}
}

// Humans returns connected human players across every zone (room goroutine only).
func (r *Room) Humans() int {
	n := 0
	for _, z := range r.zones {
		n += len(z.clients)
	}
	return n
}

// Bots returns the number of bot brains across every zone, including bots waiting to respawn.
func (r *Room) Bots() int {
	n := 0
	for _, z := range r.zones {
		n += len(z.brains)
	}
	return n
}
