package room

import (
	"kiemxoay/server/internal/bot"
	"kiemxoay/server/internal/game"
)

// updateBots implements the bot fill rules of spec §9.3.
//
// Hai con số, hai mục đích khác nhau:
//   - `room.bot_fill_target` là **trần của cả phòng** (người + bot). Thế giới 5 khu là để
//     bớt cảnh vừa vào đã bị dí, nhưng tổng số nhân vật vẫn phải có trần, nếu không mỗi
//     phòng thành năm phòng và tick server tăng gấp năm.
//   - `room.zone_min_chars` là **sàn của từng khu**. Không có nó thì bot dồn vào vài khu
//     và vẫn có khu trống trơn — người chơi qua cổng sang đó thấy một thế giới chết.
func (r *Room) updateBots(dt float64) {
	r.processRespawns()
	// Gom tên vừa rời của mọi khu — chạy ở pha ĐƠN LUỒNG, sau khi các khu đã tick xong.
	for _, z := range r.zones {
		for _, n := range z.departed {
			r.rememberName(n, z.G.Time)
		}
		z.departed = z.departed[:0]
	}
	// Vá sàn chạy MỖI TICK, không theo nhịp sinh bot. Bot đi lại giữa các khu bất cứ lúc
	// nào (qua cổng, hồi sinh), nên một khu có thể tụt xuống dưới sàn ngay giữa hai nhịp
	// sinh bot — mà nhịp đó là 2 giây. Buộc phải chờ hết nhịp mới sửa thì người chơi vừa
	// bước qua cổng có thể rơi vào một khu gần như trống. Việc này rẻ: 5 khu, vài phép so.
	//
	// Phòng đã chạm trần thì không sinh thêm được, nên cách duy nhất là ĐIỀU bot từ khu đông
	// sang — đó chính là việc `rebalanceZones` làm.
	r.rebalanceZones()
	r.botTimer += dt
	if r.botTimer < r.cfg.Bots.SpawnEverySecs {
		return
	}
	r.botTimer = 0
	// Người thật được ưu tiên trong trần phòng: họ vào bất cứ lúc nào, nên nếu tổng vượt
	// trần thì phải bớt bot NGAY, không đợi bot chết. Chờ chết thì một phòng đông người có
	// thể vượt trần suốt mấy phút.
	r.trimOverCap()
	room := r.Humans() + r.Bots()
	target := r.cfg.Room.BotFillTarget
	// Chỉ `bot_fill_target` mới chặn việc sinh bot. `max_humans` là trần NGƯỜI THẬT, hai
	// con số khác nhau: 20 người + 40 bot vẫn đúng ý "tối đa 20 người, tổng 60 nhân vật".
	// Bản cũ dừng sinh bot ngay khi đủ 20 người, nên phòng đông người lại thành phòng vắng.
	if room >= target {
		return
	}
	// Ưu tiên khu nào chưa đạt sàn; hết việc đó thì rải đều cho tới khi chạm trần phòng.
	budget := min(target-room, r.cfg.Bots.SpawnBatch)
	for i := 0; i < budget; i++ {
		z := r.zoneBelowFloor()
		if z == nil {
			z = r.thinnestByCount()
		}
		level, swords := r.randomBotLevel()
		r.spawnBotIn(z, level, swords)
	}
}

// trimOverCap bớt bot cho tới khi tổng nhân vật về đúng trần phòng. Bỏ bot ở khu ĐÔNG
// nhất trước, để việc cắt giảm không tạo ra khu trống.
func (r *Room) trimOverCap() {
	for r.Humans()+r.Bots() > r.cfg.Room.BotFillTarget {
		var fat *Zone
		for _, z := range r.zones {
			if len(z.brains) > 0 && (fat == nil || z.Count() > fat.Count()) {
				fat = z
			}
		}
		if fat == nil {
			return // không còn bot nào để bớt
		}
		for id := range fat.brains {
			fat.G.RemovePlayer(id)
			delete(fat.brains, id)
			break
		}
	}
}

// rebalanceZones điều bot từ khu đông sang khu đang dưới sàn.
//
// Chỗ dễ sai nhất ở đây là **việc chuyển khu chỉ xảy ra ở ranh giới tick**, nên `z.Count()`
// KHÔNG nhúc nhích trong lúc vòng lặp này chạy. Bản trước đọc thẳng `Count()` nên cả năm
// lượt đều thấy y hệt tình trạng cũ, xếp năm lệnh chuyển cùng một chiều, và khu cho đi bị
// rút quá tay thành khu thiếu mới — lượt sau kéo ngược về, cứ thế qua lại. Triệu chứng là
// phòng đủ 60 nhân vật mà vẫn luôn có một khu 8–9 người, lúc này lúc khác.
//
// Nên ở đây phải tự cộng sổ phần đã xếp hàng (`delta`) và quyết định theo con số DỰ KIẾN.
func (r *Room) rebalanceZones() {
	delta := make(map[*Zone]int, len(r.zones))
	count := func(z *Zone) int { return z.Count() + delta[z] }
	floor := r.cfg.Room.ZoneMinChars

	for i := 0; i < len(r.zones); i++ {
		var thin *Zone
		for _, z := range r.zones {
			if count(z) >= floor {
				continue
			}
			if thin == nil || count(z) < count(thin) {
				thin = z
			}
		}
		if thin == nil {
			return // mọi khu đã đạt sàn, hoặc sẽ đạt sau khi hàng chờ chạy
		}
		var fat *Zone
		for _, z := range r.zones {
			if z == thin || len(z.brains) == 0 {
				continue
			}
			if fat == nil || count(z) > count(fat) {
				fat = z
			}
		}
		// Hai điều kiện, hai lý do khác nhau:
		//   - khu kia phải thật sự dư, nếu không bot chỉ nhấp nháy giữa hai khu cùng thiếu;
		//   - cho đi xong nó không được tụt xuống dưới sàn, nếu không ta chỉ chuyển chỗ
		//     thiếu chứ không chữa được gì.
		if fat == nil || count(fat) <= count(thin)+1 || count(fat)-1 < floor {
			return
		}
		id, ok := fat.spareBot()
		if !ok {
			return
		}
		r.queueTransfer(transfer{playerID: id, from: fat, to: thin,
			at: game.Vec{X: thin.G.Map.Size / 2, Y: thin.G.Map.Size / 2}})
		delta[fat]--
		delta[thin]++
	}
}

// spareBot bốc một bot của khu **chưa nằm trong hàng chờ chuyển đi**.
//
// Không lọc thì vòng lặp của `rebalanceZones` có thể bốc trúng cùng một con mấy lượt liền
// (duyệt map là ngẫu nhiên): lệnh thứ hai trở đi bị bỏ qua lúc chạy thật, nên khu thiếu
// nhận được ít hơn số đã trừ trong sổ.
func (z *Zone) spareBot() (uint32, bool) {
	for id := range z.brains {
		queued := false
		for _, t := range z.transfers {
			if t.playerID == id {
				queued = true
				break
			}
		}
		if !queued {
			return id, true
		}
	}
	return 0, false
}

// zoneBelowFloor trả về khu thiếu người nhất trong số các khu chưa đạt sàn, nil nếu đủ cả.
func (r *Room) zoneBelowFloor() *Zone {
	var worst *Zone
	for _, z := range r.zones {
		if z.Count() >= r.cfg.Room.ZoneMinChars {
			continue
		}
		if worst == nil || z.Count() < worst.Count() {
			worst = z
		}
	}
	return worst
}

func (r *Room) spawnBot() *game.Player {
	level, swords := r.randomBotLevel()
	return r.spawnBotAt(level, swords)
}

// spawnBotAt thả bot vào khu ÍT NHÂN VẬT NHẤT (spec v5 mục 8) để năm khu không bị lệch —
// một khu chật ních trong khi khu khác trống trơn thì thế giới rộng cũng vô nghĩa.
func (r *Room) spawnBotAt(level, swords int) *game.Player {
	return r.spawnBotIn(r.thinnestByCount(), level, swords)
}

func (r *Room) spawnBotIn(z *Zone, level, swords int) *game.Player {
	if z == nil {
		return nil
	}
	// Tên và nhân vật theo luật §9.4 (xem `botnames.go`): không trùng trong phòng, không
	// dùng lại tên vừa rời đi 10 phút, nhân vật theo phân bố người chơi thật.
	p := z.G.AddPlayer(game.SpawnOptions{
		Name:  r.pickBotName(z.G.Time),
		Char:  r.pickBotChar(),
		IsBot: true, Level: level, Swords: swords,
	})
	z.brains[p.ID] = bot.New(p.ID, r.rng.Int63(), z.G)
	return p
}

// thinnestByCount: khu ít nhân vật nhất (người + bot), hoà thì bốc ngẫu nhiên.
func (r *Room) thinnestByCount() *Zone {
	best := []*Zone{}
	low := 1 << 30
	for _, z := range r.zones {
		switch n := z.Count(); {
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

// randomBotLevel decides what a new bot spawns as (spec §9.3).
//
// Bots slot in at `spawn_rank` (11) — level with whoever sits just below the room's Top 10 —
// so they never show up strong enough to bully the leaders. A room without that many players
// yet has nobody to measure against, so its bots start at Luyện Khí instead. There is no
// realm ceiling: in a room full of Nguyên Anh, rank 11 is Nguyên Anh and so are new bots.
//
// Xếp hạng lấy của CẢ PHÒNG, không phải của một khu: bot vào khu vắng vẫn phải vừa sức với
// mặt bằng chung, nếu không thì khu vắng thành chỗ nuôi quân toàn bot yếu.
func (r *Room) randomBotLevel() (int, int) {
	prog := r.zones[0].G.Prog
	level := r.rng.Intn(prog.Tiers()) // mặc định: Luyện Khí, tầng ngẫu nhiên
	if n := r.cfg.Bots.SpawnRank; n >= 1 {
		if ranking := r.Ranking(); len(ranking) >= n {
			level = ranking[n-1].Level
		}
	}
	realm := prog.Realm(level)
	return level, 1 + r.rng.Intn(prog.MaxSwords(realm))
}

// onBotDeath bắt chước hành vi NGƯỜI THẬT khi chết (spec v6.1 §9.4).
//
// Người thật chết thì thấy màn Tổng kết, đọc vài giây rồi bấm "Chơi tiếp" — hoặc tắt game.
// Bot cũ hồi sinh sau đúng 3 giây, mọi lần, mọi con: nhìn kill feed là thấy ngay nhịp máy.
// Giờ 70% quay lại **cùng tên** sau 4–12 giây (như bấm Chơi tiếp), 30% rời hẳn phòng và một
// bot khác tên vào sau — đúng như người chơi bỏ đi và người mới vào.
func (r *Room) onBotDeath(z *Zone, p *game.Player) {
	// ⚠ Hàm này chạy TRONG goroutine của khu (`handleDeaths` ← `stepZone`), nên chỉ được
	// đụng dữ liệu CỦA KHU. Dùng `r.rng` hay ghi thẳng `r.recentNames` ở đây là đọc-ghi
	// song song với bốn khu kia — `go test -race` bắt được ngay, và ngoài đời là
	// `fatal error: concurrent map writes` làm sập server (đã dính đúng vậy hôm 29/10).
	// Tên người vừa rời gom vào hàng chờ của khu, `updateBots` gom lại sau khi mọi khu
	// tick xong.
	if r.Humans()+r.Bots()-1 >= r.cfg.Room.BotFillTarget {
		z.departed = append(z.departed, p.Name)
		z.G.RemovePlayer(p.ID)
		delete(z.brains, p.ID)
		return
	}
	bc := r.cfg.Bots
	if z.rng.Float64() >= bc.RejoinSameNameChance {
		// "Rời phòng": bỏ hẳn con này. `updateBots` sẽ lấp chỗ trống bằng một bot MỚI, tên
		// mới — nhịp lấp do `spawn_every_secs` lo, nên khoảng trống tự nhiên 5–20 giây.
		z.departed = append(z.departed, p.Name)
		z.G.RemovePlayer(p.ID)
		delete(z.brains, p.ID)
		return
	}
	lo, hi := bc.RespawnSecsRange[0], bc.RespawnSecsRange[1]
	wait := lo + z.rng.Float64()*(hi-lo)
	z.respawns = append(z.respawns, pendingRespawn{id: p.ID, at: z.G.Time + wait})
}

// respawnQueued chỉ hỏi hàng chờ CỦA KHU đó — gọi từ goroutine của chính khu ấy.
func (z *Zone) respawnQueued(id uint32) bool {
	for _, q := range z.respawns {
		if q.id == id {
			return true
		}
	}
	return false
}

// processRespawns hồi sinh bot tới hạn. Bot hồi sinh **chuyển sang khu ít nhân vật nhất**
// (spec v5 mục 8), không ở lại chỗ cũ — nếu không thì khu đông sẽ tự nuôi lại chính nó.
func (r *Room) processRespawns() {
	for _, z := range r.zones {
		keep := z.respawns[:0]
		for _, q := range z.respawns {
			if z.G.Time < q.at {
				keep = append(keep, q)
				continue
			}
			p, ok := z.G.Players[q.id]
			if !ok {
				continue
			}
			level, swords := r.randomBotLevel()
			// Chỉ điều đi khi khu cũ có người để dư. Khu đang đúng sàn mà cho đi thì chỗ
			// thiếu chỉ chạy sang chỗ khác chứ không mất — cùng một kiểu sai với
			// `rebalanceZones` bản cũ.
			if to := r.thinnestByCount(); to != nil && to != z && z.Count() > r.cfg.Room.ZoneMinChars {
				z.G.RemovePlayer(p.ID)
				delete(z.brains, p.ID)
				r.spawnBotIn(to, level, swords)
				continue
			}
			z.G.Respawn(p, level, swords)
		}
		z.respawns = keep
	}
}
