package game

import "math"

// Tiên giới (spec §3.4 v4): the five realms above Đại Thừa. They ignore Linh khí (EXP)
// entirely — the only way up is **Tiên khí**, a special orb that only the Đài Tiên Khí on
// the map spray out (§8.5). The "sát mạng" rule of the previous draft is gone.

// TienRealmOf is the player's Tiên realm index (0-based), or -1 while still mortal.
//
// Tính thẳng từ **tổng Tiên khí** (spec v5.7 §3.4), không đọc `Level`. `Level` vẫn được giữ
// đồng bộ để phần còn lại của game (scale, số kiếm tối đa, hào quang) dùng, nhưng nguồn sự
// thật chỉ có một — nếu không thì hai con số sẽ lệch nhau đúng lúc bị chém.
func (g *Game) TienRealmOf(p *Player) int {
	if !g.BanksTienKhi(p) {
		return -1
	}
	return g.Prog.TienRealmForKhi(p.TienKhi)
}

// BanksTienKhi: người này đã sang đường tu Tiên khí chưa.
//
// Mốc là **Đại Thừa tầng 10 ĐẦY THANH** (spec v5.7 §3.4), không phải chỉ cần đứng ở Đại
// Thừa tầng 10. Khác biệt thật: trước đây vừa chạm tầng 10 là Linh khí nhặt được nữa cũng
// bị bỏ phí trong khi Tiên khí thì hiếm, nên người chơi kẹt cứng ở đó. Giờ họ nạp nốt thanh
// Đại Thừa bằng Linh khí như thường, đầy rồi mới chuyển sang săn Tiên khí.
//
// Đây cũng là ranh giới của HAI ĐƯỜNG TU TÁCH HẲN (anh chốt 2026-10-28, xem Q18): ai đã
// sang Tiên khí thì không hấp thụ Linh khí nữa, và ngược lại.
func (g *Game) BanksTienKhi(p *Player) bool {
	if !g.Cfg.Cultivation.TienEntryRequiresFullExp {
		return p.Level >= g.Prog.TopMortalLevel()
	}
	return p.TotalExp >= g.Prog.MaxExp()-1e-9
}

// CanAbsorbTienKhi / CanAbsorbExp: mỗi người chỉ hấp thụ đúng một loại tu vi.
func (g *Game) CanAbsorbTienKhi(p *Player) bool { return g.BanksTienKhi(p) }
func (g *Game) CanAbsorbExp(p *Player) bool     { return !g.BanksTienKhi(p) }

// GainTienKhi banks Tiên khí and promotes when a threshold is crossed. Returns how much
// actually went into the Tiên khí bar — 0 for anyone still on the Linh khí path.
func (g *Game) GainTienKhi(p *Player, points float64) float64 {
	if points <= 0 || p.Dead || !g.CanAbsorbTienKhi(p) {
		return 0
	}
	before := g.TienRealmOf(p)
	p.TienKhi += points
	g.overflowToScore(p)
	// Cảnh giới suy ra từ tổng Tiên khí, nên chỉ cần tính lại một lần — không còn vòng lặp
	// "cộng dồn rồi thăng từng bậc". Nhặt một cục 400 điểm thì nhảy thẳng mấy cảnh giới.
	if after := g.TienRealmOf(p); after > before {
		g.promoteTien(p, after)
	}
	return points
}

// overflowToScore xử lý phần Tiên khí vượt trần Đạo Tổ (1 500).
//
// Luật v5.7: Đạo Tổ **vẫn nạp thanh bình thường từ 1 001 lên 1 500**; chỉ phần vượt QUÁ
// 1 500 mới đổi Uy Danh (10:1). Bản v5.6 ghim thanh ở đúng mốc vào cảnh giới nên Đạo Tổ
// không bao giờ có phần dư để Thiêu Đốt — đó là Q26.
//
// Vẫn chặn thanh ở 1 500 chứ không cho phình mãi: `robTienKhi` tính `loss = tien_khi − đáy`,
// không chặn thì một người Đạo Tổ ngồi lâu thành cục mồi khổng lồ.
func (g *Game) overflowToScore(p *Player) {
	cap := g.Prog.TienMaxKhi()
	over := p.TienKhi - cap
	if over <= 0 {
		return
	}
	p.TienKhi = cap
	if rate := g.Cfg.Cultivation.TienOverflowScore; rate > 0 {
		g.addScore(p, "tien_overflow", over*rate)
	}
}

// promoteTien moves a player up to Tiên realm idx with the Thiên Kiếp announcement.
func (g *Game) promoteTien(p *Player, idx int) {
	p.Level = g.Prog.TienLevel(idx)
	p.PeakLevel = max(p.PeakLevel, p.Level)
	g.refreshDerived(p)
	realm := g.Prog.Realm(p.Level)
	g.awardBreakthrough(p, realm)
	g.setMood(p, MoodBreakthrough, g.Cfg.Mood.BreakthroughSecs)
	g.emit(Event{Kind: "Breakthrough", A: p.ID, N: p.Level, S: "tien",
		X: float32(p.Pos.X), Y: float32(p.Pos.Y)})
	g.emit(Event{Kind: "KillFeed", Global: true, S: "tien_ascend", A: p.ID, N: realm})
	g.shockNearby(p)
}

// robTienKhi applies spec §3.4 when a Tiên giới player is cut: they drop one realm, their
// Tiên khí falls back to the new realm's entry threshold, the attacker robs tien_rob_share
// of the loss and tien_drop_share scatters as orbs around the victim.
func (g *Game) robTienKhi(attacker, victim *Player) {
	idx := g.TienRealmOf(victim)
	if idx < 0 {
		return
	}
	c := g.Cfg.Cultivation
	var entry float64
	if idx == 0 {
		// Chân Tiên bị chém → về **SÀN Đại Thừa** (tầng 1, thanh 0%), mất sạch Tiên khí
		// (spec v5.8). Bản v5.7 cho về Đại Thừa tầng 10 đầy thanh, tức là chỉ cần nhặt một
		// viên Tiên khí là lên lại Chân Tiên ngay — rớt cảnh giới mà gần như không mất gì.
		victim.Level = g.Prog.Realm(g.Prog.TopMortalLevel()) * g.Prog.Tiers()
		victim.TotalExp = g.Prog.Threshold(victim.Level)
	} else {
		victim.Level = g.Prog.TienLevel(idx - 1)
		entry = g.Prog.TienEntry(idx - 1)
	}
	loss := math.Max(0, victim.TienKhi-entry)
	victim.TienKhi = entry
	if idx == 0 {
		victim.TienKhi = 0 // rớt hẳn khỏi Tiên giới thì thanh Tiên khí về 0, không phải 1
	}
	g.refreshDerived(victim)
	g.setMood(victim, MoodRealmDrop, g.Cfg.Mood.RealmDropSecs)
	g.emit(Event{Kind: "RealmDrop", A: victim.ID, N: victim.Level,
		X: float32(victim.Pos.X), Y: float32(victim.Pos.Y)})
	g.dropExcessSwords(victim)
	g.emit(Event{Kind: "KillFeed", Global: true, S: "realm_drop", A: attacker.ID, B: victim.ID,
		N: g.Prog.Realm(victim.Level)})
	if loss <= 0 {
		return
	}
	// Người chém chỉ CƯỚP được Tiên khí khi bản thân đã sang đường tu đó (spec §3.4);
	// `GainTienKhi` tự lo điều kiện ấy.
	g.GainTienKhi(attacker, loss*c.TienRobShare)
	// Nhưng phần RƠI RA ĐẤT thì **luôn luôn** xảy ra, bất kể người chém là phàm, Tiên hay
	// bot (spec v5.9 §3.4). Đây là nguồn Tiên khí quan trọng ngoài Đài Tiên Khí: người phàm
	// hạ được một Đạo Tổ mà không rơi gì thì cả khu mất một cơ hội lớn.
	g.scatterTienKhi(victim.Pos, loss*c.TienDropShare)
	g.emit(Event{Kind: "SkillEffect", A: victim.ID, S: "tien_burst",
		X: float32(victim.Pos.X), Y: float32(victim.Pos.Y), N: int(loss * c.TienDropShare)})
}

// TienProgress is what the HUD bar shows: Tiên khí banked inside the current realm and how
// wide that realm is (spec §3.4 "Tiên khí 150/300").
//
// Đạo Tổ KHÔNG còn trả (0, 0) nữa: từ v5.7 họ vẫn nạp thanh 1 001 → 1 500, và chính thanh
// đó quyết định lúc nào Tiên khí dư mới đổi sang Uy Danh — giấu thanh đi thì người chơi
// không biết còn bao xa mới tới mốc đổi.
func (g *Game) TienProgress(p *Player) (have, at float64) {
	idx := g.TienRealmOf(p)
	if idx < 0 {
		return 0, 0
	}
	from := g.Prog.TienEntry(idx)
	return math.Max(0, p.TienKhi-from+1), g.Prog.TienWidth(idx)
}
