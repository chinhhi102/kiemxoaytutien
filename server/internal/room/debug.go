package room

import (
	"log"

	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/netproto"
)

// Debug applies a test shortcut (M1 requirement); only reachable when the server runs with -debug.
func (r *Room) Debug(c *Client, d netproto.Debug) {
	r.Do(func() {
		z := r.zoneOf(c.PlayerID)
		if z == nil {
			log.Printf("dbg %q bỏ qua: người chơi %d không còn trong phòng", d.Cmd, c.PlayerID)
			return
		}
		p, ok := z.G.Players[c.PlayerID]
		if !ok || p.Dead {
			// Im lặng bỏ qua ở đây từng làm bài test 10 tab ra bảng số sai suốt hai lượt:
			// lệnh lên Đạo Tổ gửi đi lúc nhân vật đang nằm chờ hồi sinh thì rơi vào hư vô.
			log.Printf("dbg %q bỏ qua: người chơi %d %s", d.Cmd, c.PlayerID,
				map[bool]string{true: "đã chết", false: "không còn trong phòng"}[ok])
			return
		}
		switch d.Cmd {
		case "exp":
			z.G.GainExp(p, float64(d.N), false)
		case "realm":
			// Jump to the start of the next realm. Từ Đại Thừa tầng 10 trở lên không hấp thụ
			// Linh khí nữa (§3.4) nên phải cấp thẳng Tiên khí, không thì phím test tắc ở đó
			// và không bao giờ thử được Tiên giới.
			if z.G.BanksTienKhi(p) {
				// Cấp đủ Tiên khí để vượt TRẦN cảnh giới hiện tại — vượt trần mới lên cảnh
				// giới mới (spec v5.7). Chưa có Tiên khí nào thì một điểm là vào Chân Tiên.
				idx := z.G.TienRealmOf(p)
				if ceil := z.G.Prog.TienCeil(idx); idx < 0 {
					z.G.GainTienKhi(p, 1)
				} else if idx+1 < z.G.Prog.TienCount() {
					z.G.GainTienKhi(p, ceil+1-p.TienKhi)
				}
				return
			}
			next := (z.G.Prog.Realm(p.Level) + 1) * z.G.Prog.Tiers()
			z.G.GainExp(p, z.G.Prog.Threshold(next)-p.TotalExp, false)
		case "max":
			// Lên thẳng Đạo Tổ với đủ kiếm. Chỉ dùng cho công cụ đo (tools/browsertest) —
			// bấm phím "realm" 14 lần rồi "sword" 105 lần thì vừa chậm vừa hay lệch.
			top := z.G.Prog.TienLevel(len(r.cfg.Cultivation.TienRealms) - 1)
			z.G.ForceState(p, top, z.G.Prog.MaxSwords(z.G.Prog.Realm(top)), p.Pos)
			if d.N == 0 {
				p.InvulnUntil = 0 // mặc định: đánh nhau thật
			}
			// n = 1 → giữ bất tử. Công cụ đo cần giữ nguyên cảnh NẶNG NHẤT suốt pha đo;
			// để chúng chém chết nhau thì sau vài giây cả phòng về Luyện Khí 1 kiếm và số
			// đo thành vô nghĩa (đã dính đúng lỗi này một lần).
		case "tien":
			// Nạp Tiên khí để thử luật Tiên giới (đổi dư sang Uy Danh, cướp đoạt §3.4).
			z.G.GainTienKhi(p, float64(max(1, d.N)))
		case "sword":
			z.G.AddSwords(p, max(1, d.N))
		case "skill":
			// Cấp lượt kỹ năng mà không phải đi tìm Bí Kíp. Không có lệnh này thì muốn thử
			// một kỹ năng phải chạy quanh bản đồ chờ Bí Kíp rơi — đủ lâu để không ai thử.
			for i := 0; i < max(1, d.N); i++ {
				z.G.AddSkillCharge(p)
			}
		case "die":
			z.G.DebugKill(p)
		case "buff":
			if d.N >= 0 && d.N < game.BuffCount {
				z.G.ApplyBuff(p, game.BuffNames[d.N])
			}
		case "bot":
			// Thả một bot ngay cạnh mình để thử đánh nhau; d.N = cảnh giới (0 = như mình).
			realm := d.N
			if realm <= 0 {
				realm = z.G.Prog.Realm(p.Level)
			}
			realm = min(realm, len(r.cfg.Cultivation.Realms)-1)
			b := r.spawnBotIn(z, realm*z.G.Prog.Tiers(), z.G.Prog.MaxSwords(realm)/4+1)
			b.Pos = p.Pos.Add(z.G.Vec(z.G.Geom(p).Reach+120, 0))
			b.InvulnUntil = 0
		default:
		}
	})
}
