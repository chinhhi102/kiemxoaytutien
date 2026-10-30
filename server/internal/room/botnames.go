package room

import (
	"fmt"
	"strings"
)

// Đặt tên bot sao cho **không phân biệt được với người thật** (spec v6.1 §9.4).
//
// Vì sao đáng làm: bot chiếm phần lớn một phòng (trần 60 nhân vật, tối đa 20 người thật).
// Nếu tên bot nhìn là biết ngay thì cả bảng xếp hạng và kill feed đều lộ ra "đây là phòng
// toàn máy", và cảm giác đông vui mà hệ thống bot dựng lên sụp hết.

// nameMemorySecs: tên vừa rời phòng bị "giữ chỗ" ngần này giây trước khi được dùng lại.
//
// Spec nói 10 phút. Lý do có mốc này: không có nó thì một bot chết rồi bot mới vào mang
// đúng tên vừa biến mất — người chơi đang nhìn kill feed sẽ thấy cùng một cái tên chết đi
// sống lại liên tục, và đó là dấu hiệu lộ bot rõ nhất.
const nameMemorySecs = 600

// nameVariantChance: tỉ lệ thêm biến thể nhỏ vào tên, để hai phòng khác nhau không lặp y
// hệt một danh sách tên (spec §9.4).
const nameVariantChance = 0.15

// takenNames gom mọi tên đang dùng trong phòng (người thật lẫn bot, mọi khu).
func (r *Room) takenNames() map[string]bool {
	out := make(map[string]bool, 64)
	for _, z := range r.zones {
		for _, p := range z.G.Players {
			out[strings.ToLower(p.Name)] = true
		}
	}
	return out
}

// rememberName ghi lại một tên vừa rời phòng để không dùng lại ngay.
func (r *Room) rememberName(name string, now float64) {
	if r.recentNames == nil {
		r.recentNames = map[string]float64{}
	}
	r.recentNames[strings.ToLower(name)] = now + nameMemorySecs
}

// pickBotName rút một tên chưa dùng trong phòng và chưa rời đi gần đây.
//
// Rút NGẪU NHIÊN rồi thử, thay vì trộn cả danh sách 1 000 tên: phòng nhiều nhất 60 nhân vật
// nên xác suất trùng rất thấp, vài lần thử là xong. Trộn cả danh sách mỗi lần sinh bot thì
// tốn hơn hẳn mà không được gì.
func (r *Room) pickBotName(now float64) string {
	names := r.cfg.Bots.Names
	if len(names) == 0 {
		return fmt.Sprintf("Khach%03d", r.rng.Intn(1000))
	}
	taken := r.takenNames()
	free := func(n string) bool {
		key := strings.ToLower(n)
		if taken[key] {
			return false
		}
		until, seen := r.recentNames[key]
		return !seen || now >= until
	}
	for try := 0; try < 40; try++ {
		base := names[r.rng.Intn(len(names))]
		// Kiểm TÊN GỐC trước, rồi mới thêm biến thể.
		//
		// Làm ngược lại thì biến thể lách qua được bộ nhớ tên: "Khoa_09" vừa chết, bot mới
		// vào tên "Khoa.09" — người chơi nhìn kill feed vẫn thấy đúng một người chết đi
		// sống lại, chỉ khác một dấu chấm. Test `TestRecentNameNotReused` bắt được đúng chỗ này.
		if !free(base) {
			continue
		}
		if r.rng.Float64() < nameVariantChance {
			if v := varyName(base, r.rng.Intn(100)); free(v) {
				return v
			}
		}
		return base
	}
	// Rút ngẫu nhiên không ra thì QUÉT cả danh sách. Chỉ xảy ra khi gần hết tên còn trống,
	// và lúc đó quét vẫn rẻ (1 000 chuỗi) mà cho ra một cái tên THẬT — hơn hẳn việc bịa
	// "ShadowMonk48", thứ đọc ra là biết máy đặt.
	start := r.rng.Intn(len(names))
	for i := 0; i < len(names); i++ {
		if n := names[(start+i)%len(names)]; free(n) {
			return n
		}
	}
	// Hết sạch tên: gắn thêm số cho chắc chắn không trùng. Thà một cái tên hơi nhạt còn hơn
	// hai nhân vật cùng tên đứng cạnh nhau trong bảng xếp hạng.
	return fmt.Sprintf("%s%d", names[r.rng.Intn(len(names))], r.rng.Intn(90)+10)
}

// varyName thêm biến thể nhỏ: đổi `_` ↔ `.` hoặc gắn 1–2 chữ số (spec §9.4).
//
// Cố ý KHÔNG đụng tới chữ cái và dấu tiếng Việt: biến thể là để tên không lặp y hệt giữa
// các phòng, không phải để bịa ra tên mới — bịa thì dễ ra thứ đọc không giống người đặt.
func varyName(n string, salt int) string {
	switch {
	case strings.Contains(n, "_"):
		return strings.Replace(n, "_", ".", 1)
	case strings.Contains(n, "."):
		return strings.Replace(n, ".", "_", 1)
	case salt%3 == 0:
		return fmt.Sprintf("%s%d", n, salt%10)
	default:
		return fmt.Sprintf("%s%02d", n, salt)
	}
}

// pickBotChar chọn nhân vật cho bot theo PHÂN BỐ NGƯỜI CHƠI THẬT đang ở trong phòng
// (spec §9.4), không rải đều.
//
// Rải đều là một dấu hiệu lộ bot tinh vi hơn tên: người chơi thật dồn vào vài nhân vật họ
// thích, nên một phòng mà mười nhân vật xuất hiện đều tăm tắp thì nhìn là thấy máy. Chưa có
// người thật nào trong phòng thì rải đều — không có gì để bắt chước.
func (r *Room) pickBotChar() string {
	chars := r.cfg.Characters
	if len(chars) == 0 {
		return ""
	}
	counts := map[string]int{}
	total := 0
	for _, z := range r.zones {
		for id, p := range z.G.Players {
			if _, human := z.clients[id]; human {
				counts[p.Char]++
				total++
			}
		}
	}
	if total == 0 {
		return chars[r.rng.Intn(len(chars))]
	}
	// Cộng 1 cho mọi nhân vật (làm trơn Laplace): nhân vật chưa ai chọn vẫn có cơ hội, nếu
	// không thì một phòng mở đầu bằng hai người cùng chọn A Hớn sẽ thành phòng toàn A Hớn.
	pick := r.rng.Intn(total + len(chars))
	for _, c := range chars {
		w := counts[c] + 1
		if pick < w {
			return c
		}
		pick -= w
	}
	return chars[r.rng.Intn(len(chars))]
}
