package room

import (
	"strings"
	"testing"

	"kiemxoay/server/internal/netproto"
)

// Luật đặt tên bot (spec v6.1 §9.4). Mục đích của cả nhóm test này chỉ có một: **không
// được nhìn là biết đâu là bot**.

// Nguồn tên phải lấy từ `config/bot_names.txt`, không phải danh sách ngắn trong game.json.
func TestBotNamesLoadedFromFile(t *testing.T) {
	_, _, cfg := testManager(t)
	if n := len(cfg.Bots.Names); n < 500 {
		t.Fatalf("chỉ nạp được %d tên bot; `config/bot_names.txt` phải có khoảng 1 000", n)
	}
	for _, bad := range []string{"bot", "Bot", "BOT"} {
		for _, n := range cfg.Bots.Names {
			if strings.Contains(n, bad) {
				t.Errorf("tên %q lộ là bot", n)
			}
		}
	}
}

// Không hai nhân vật nào trong cùng phòng trùng tên.
func TestBotNamesUniqueInRoom(t *testing.T) {
	m, _, cfg := testManager(t)
	conn := &recConn{}
	m.Join(NewClient(conn, false), netproto.Join{Name: "NguoiThat", Char: "a_hon"})
	r := m.AllRooms()[0]
	dt := 1 / float64(cfg.Net.TickHz)
	// Chạy đủ lâu để phòng lấp đầy bot.
	for i := 0; i < int(60/dt); i++ {
		r.Step(dt)
		if r.Humans()+r.Bots() >= cfg.Room.BotFillTarget {
			break
		}
	}
	seen := map[string]string{}
	n := 0
	for _, z := range r.Zones() {
		for _, p := range z.Game().Players {
			n++
			key := strings.ToLower(p.Name)
			if other, dup := seen[key]; dup {
				t.Errorf("trùng tên %q (đã có ở %s)", p.Name, other)
			}
			seen[key] = p.Name
		}
	}
	if n < 10 {
		t.Fatalf("phòng mới có %d nhân vật, chưa đủ để kiểm trùng tên", n)
	}
}

// Tên vừa rời phòng không được dùng lại ngay — người chơi nhìn kill feed sẽ thấy cùng một
// cái tên chết đi sống lại liên tục, và đó là dấu hiệu lộ bot rõ nhất.
func TestRecentNameNotReused(t *testing.T) {
	m, _, _ := testManager(t)
	m.Join(NewClient(&recConn{}, false), netproto.Join{Name: "P", Char: "a_hon"})
	r := m.AllRooms()[0]

	// Giữ chỗ gần hết danh sách, chừa đúng một tên.
	now := 100.0
	keep := r.cfg.Bots.Names[0]
	for _, n := range r.cfg.Bots.Names[1:] {
		r.rememberName(n, now)
	}
	got := r.pickBotName(now)
	if !strings.HasPrefix(strings.ToLower(got), strings.ToLower(keep)) {
		t.Errorf("rút được %q trong khi chỉ còn %q chưa bị giữ chỗ", got, keep)
	}

	// Qua 10 phút thì các tên cũ dùng lại được.
	later := now + nameMemorySecs + 1
	fresh := map[string]bool{}
	for i := 0; i < 50; i++ {
		fresh[r.pickBotName(later)] = true
	}
	if len(fresh) < 5 {
		t.Errorf("sau 10 phút chỉ rút được %d tên khác nhau — bộ nhớ tên không hết hạn", len(fresh))
	}
}

// Biến thể chỉ đụng tới ký tự trang trí và chữ số, không bịa ra chữ mới.
func TestVaryNameKeepsLetters(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"huy_pr0", "huy.pr0"},
		{"meo.22", "meo_22"},
	} {
		if got := varyName(c.in, 0); got != c.want {
			t.Errorf("varyName(%q) = %q, muốn %q", c.in, got, c.want)
		}
	}
	// Tên không có `_` hay `.` thì chỉ được GẮN THÊM số, phần chữ giữ nguyên.
	for _, salt := range []int{0, 1, 7, 42} {
		got := varyName("Bé Thỏ", salt)
		if !strings.HasPrefix(got, "Bé Thỏ") {
			t.Errorf("varyName làm hỏng phần chữ: %q", got)
		}
	}
}

// Bot chết thì phần lớn quay lại CÙNG TÊN sau vài giây (như người bấm "Chơi tiếp"), phần
// còn lại rời hẳn phòng. Hồi sinh đúng 3 giây mọi lần là nhịp máy, nhìn kill feed thấy ngay.
func TestBotDeathBehaviourConfig(t *testing.T) {
	_, _, cfg := testManager(t)
	b := cfg.Bots
	if len(b.RespawnSecsRange) != 2 || b.RespawnSecsRange[0] != 4 || b.RespawnSecsRange[1] != 12 {
		t.Errorf("respawn_secs_range = %v, spec §9.4 nói 4–12 giây", b.RespawnSecsRange)
	}
	if b.RejoinSameNameChance != 0.7 {
		t.Errorf("rejoin_same_name_chance = %v, spec nói 0.7", b.RejoinSameNameChance)
	}
	if b.IdleAfterJoinChance != 0.05 {
		t.Errorf("idle_after_join_chance = %v, spec nói 0.05", b.IdleAfterJoinChance)
	}
}
