package game

// Trạng thái có thời hạn (spec §6.3).
//
// Mười kỹ năng nhân vật tạo ra năm kiểu khống chế khác nhau — chậm, trói, đẩy lùi, trượt,
// bịt mắt — cộng ba kiểu tự tăng cường. Viết riêng mỗi kỹ năng một cách xử lý là mười chỗ
// phải nhớ hết hạn, mười chỗ phải kiểm "đang bị cái này thì có chịu cái kia không". Gom hết
// vào MỘT bảng thời hạn: thêm kỹ năng sau chỉ là thêm một hằng số.
//
// Khác `Buffs` (vật phẩm nhặt được, §6.2) ở chỗ: buff là phần thưởng tự chọn, còn trạng thái
// ở đây phần lớn là thứ người khác giáng lên mình.

const (
	// StSlow: chậm chân và chậm cả vòng kiếm (Ru Ngủ của Ông Ba Võng).
	StSlow = iota
	// StRoot: không di chuyển được, kiếm vẫn quay (Lưới Mì của Sư Phụ Mì Gói).
	StRoot
	// StBlind: trân châu dính đầy màn hình (Thằng Mập). Server chỉ đếm giờ; vẽ là việc client.
	StBlind
	// StSlip: trượt theo quán tính như trên băng (vệt trái cây của Cô Ba Gánh Hàng).
	StSlip
	// StBrace: gồng — độ bền kiếm ×2 và miễn mọi khống chế (Bé Tí Cơ Bắp).
	StBrace
	// StGhost: đi xuyên vật cản PHÁ ĐƯỢC (Anh Shipper). Không xuyên tường không phá được.
	StGhost
	// StShell: rụt vào mai — không bị bể kiếm, không bị chém thân (Cụ Rùa).
	StShell
	// StDash: đang lướt (Gánh Lướt của Cô Ba).
	StDash
	// StStun: choáng — không điều khiển, không dùng kỹ năng, không Thiêu Đốt; kiếm vẫn
	// quay (spec v5.5 §6.3: ai đi vào vệt trái cây của Cô Ba thì vừa trượt vừa choáng 1s).
	//
	// Tách khỏi StSlip vì hai thứ làm hai việc khác nhau: trượt là đổi CÁCH di chuyển (quán
	// tính), choáng là CẤM di chuyển. Gộp lại thì không nói được "trượt mà vẫn lái được"
	// nếu sau này spec đổi ý.
	StStun
	StatusCount
)

// StatusNames dùng cho log và event gửi xuống client.
var StatusNames = [StatusCount]string{"slow", "root", "blind", "slip", "brace", "ghost", "shell", "dash", "stun"}

// HasStatus: trạng thái này còn hiệu lực không.
func (p *Player) HasStatus(s int, now float64) bool { return p.Status[s] > now }

// controlStatuses là những trạng thái bị người khác giáng lên; Gồng miễn hết nhóm này
// (spec §6.3: "không bị đẩy lùi/chậm/trói").
var controlStatuses = []int{StSlow, StRoot, StSlip, StBlind, StStun}

// ApplyStatus giáng một trạng thái lên người chơi trong `secs` giây.
//
// Luôn lấy mốc hết hạn XA HƠN giữa cũ và mới, không cộng dồn: trúng Ru Ngủ hai lần liên
// tiếp thì vẫn chỉ chậm 3 giây kể từ lần sau, chứ không thành 6 giây — nếu không, hai người
// cùng kỹ năng có thể khoá cứng một người vô thời hạn.
func (g *Game) ApplyStatus(p *Player, s int, secs float64) {
	if p.Dead || secs <= 0 {
		return
	}
	if p.HasStatus(StBrace, g.Time) && isControl(s) {
		return // đang gồng: miễn khống chế
	}
	if until := g.Time + secs; until > p.Status[s] {
		p.Status[s] = until
	}
	if s == StBlind {
		g.emit(Event{Kind: "Blinded", A: p.ID, To: p.ID, F: float32(secs)})
	}
}

func isControl(s int) bool {
	for _, c := range controlStatuses {
		if c == s {
			return true
		}
	}
	return false
}

// ClearStatuses xoá sạch trạng thái — gọi khi chết và khi hồi sinh.
func (g *Game) ClearStatuses(p *Player) {
	for i := range p.Status {
		p.Status[i] = 0
	}
}

// onDeathClear xoá những thứ chết là mất: lượt kỹ năng và mọi trạng thái (spec §6.3 —
// "mất hết lượt khi chết", và không ai muốn hồi sinh xong vẫn đang bị trói).
//
// Rớt cảnh giới hay qua cổng thì KHÔNG mất lượt, nên việc xoá chỉ gắn vào đúng chỗ chết.
func (g *Game) onDeathClear(p *Player) {
	p.SkillCharges = 0
	p.Vel = Vec{}
	p.shellOn = false
	p.orbPicked, p.orbPickedID, p.orbPickedKind = 0, 0, ""
	g.ClearStatuses(p)
}

// BlindSightMult: người đang bị bịt mắt nhìn xa được bao nhiêu phần (spec §6.3 — "bot bị
// trúng: tầm nhìn còn 40%").
//
// Với người thật, cái che tầm nhìn là lớp trân châu trên màn hình. Với bot thì không có
// màn hình nào cả — nếu không rút tầm nhìn của chúng thì kỹ năng này **không làm gì** với
// bot, và bài đấu từng cặp cho Thằng Mập đúng 0 đối thủ bị khắc.
func (g *Game) BlindSightMult(p *Player) float64 {
	if p.HasStatus(StBlind, g.Time) {
		return g.Cfg.Skills.TranChauBotVision
	}
	return 1
}

// StatusBits gói trạng thái đang bật thành một số để nhét vào snapshot.
func (p *Player) StatusBits(now float64) uint16 {
	var bits uint16
	for i := 0; i < StatusCount; i++ {
		if p.Status[i] > now {
			bits |= 1 << i
		}
	}
	return bits
}
