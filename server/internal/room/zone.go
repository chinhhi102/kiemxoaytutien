package room

import (
	"math/rand"

	"kiemxoay/server/internal/bot"
	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/netproto"
	"kiemxoay/server/internal/world"
)

// Zone là MỘT khu vực của phòng (spec §8.0): một thế giới 12 000 × 12 000 riêng, mô phỏng
// riêng, danh sách người chơi riêng, vùng nhìn (AOI) riêng.
//
// Năm khu **không nối nhau bằng đường đi** — tường biên mỗi khu không qua được — nên hai
// khu không bao giờ phải biết gì về nhau trong một tick. Nhờ vậy mỗi khu chạy được trên
// một goroutine mà không cần khoá: việc duy nhất đi qua ranh giới là **chuyển người giữa
// các khu**, và việc đó làm ở ranh giới tick, lúc không khu nào đang chạy.
//
// Những thứ tính chung CẢ THẾ GIỚI (bảng xếp hạng phòng, danh hiệu, Uy Danh) vẫn nằm ở
// Room, không nằm ở đây.
type Zone struct {
	// Index 0..4 theo bảng §8.0; Name là tên hiển thị cho HUD và kill feed.
	Index int
	Name  string
	G     *game.Game

	clients map[uint32]*Client
	brains  map[uint32]*bot.Brain
	pending []game.Event

	// itemBuf dùng lại giữa các lần dựng snapshot — spec §13.4 cấm cấp phát mỗi tick.
	itemBuf []itemCand
	// stepMs: thời gian xử lý một tick của riêng khu này, gửi kèm snapshot cho overlay F3.
	stepMs  float32
	tick    uint32
	snapAcc float64

	// rng và transfers RIÊNG cho từng khu: năm khu chạy song song, dùng chung một bộ sinh
	// số ngẫu nhiên hay chung một slice là đua dữ liệu ngay. Gom lại sau khi mọi khu xong.
	rng       *rand.Rand
	transfers []transfer
	// respawns: bot của KHU NÀY đang chờ hồi sinh. Để ở phòng thì năm goroutine cùng ghi
	// một slice — `-race` bắt được ngay khi phòng đông bot.
	respawns []pendingRespawn
	// departed: tên bot vừa rời khu này, chờ `updateBots` gom vào bộ nhớ tên của phòng.
	// Gom ở đây vì `onBotDeath` chạy trong goroutine của khu (xem ghi chú ở đó).
	departed []string
	// namesAt/namesDue: lần cuối gửi lại tên người chơi. Thuộc về KHU, không phải phòng —
	// năm khu chạy song song nên để ở phòng là hai goroutine cùng ghi một ô nhớ.
	namesAt  float64
	namesDue bool
	// feeds: tin kill feed khu này vừa sinh ra. Kill feed đi KHẮP PHÒNG (spec §8.0), mà
	// gửi sang client của khu khác ngay trong goroutine của mình là ghi vào kết nối người
	// khác đúng lúc khu kia đang ghi — đua dữ liệu. Gom lại, gửi sau khi mọi khu xong.
	feeds []netproto.Feed
}

// ZoneMapIDs: id bản đồ của từng khu, do `cmd/mapgen` sinh ra (spec §8.0).
var ZoneMapIDs = []string{"thien_dinh", "linh_son_coc", "truc_lam", "pham_tran", "bang_nguyet_ho"}

// ZoneNames là tên 5 khu theo đúng thứ tự bảng §8.0.
var ZoneNames = []string{"Thiên Đình", "Linh Sơn Cốc", "Trúc Lâm Mê Cung", "Phàm Trấn", "Băng Nguyệt Hồ"}

// ZoneName trả tên khu thứ i, an toàn với chỉ số ngoài khoảng.
func ZoneName(i int) string {
	if i < 0 || i >= len(ZoneNames) {
		return ""
	}
	return ZoneNames[i]
}

func newZone(index int, cfg *config.Config, m *world.Map, seed int64) *Zone {
	return &Zone{
		Index:   index,
		Name:    ZoneName(index),
		G:       game.New(cfg, m, seed),
		clients: map[uint32]*Client{},
		brains:  map[uint32]*bot.Brain{},
		rng:     rand.New(rand.NewSource(seed*31 + 17)),
	}
}

// StepMs là thời gian xử lý một tick của riêng khu này (trung bình trượt), cho công cụ đo.
func (z *Zone) StepMs() float32 { return z.stepMs }

// Game mở Game của khu cho công cụ đo và test.
func (z *Zone) Game() *game.Game { return z.G }

// Humans đếm người thật đang ở khu này — `clients` chỉ chứa người, bot nằm ở `brains`.
// Dùng để chọn khu ít người nhất khi vào trận (§8.0).
func (z *Zone) Humans() int { return len(z.clients) }

// Count là tổng nhân vật (người + bot) trong khu — dùng để rải bot cho đều.
func (z *Zone) Count() int { return len(z.clients) + len(z.brains) }
