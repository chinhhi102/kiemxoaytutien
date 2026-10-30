// Package netproto defines the MessagePack messages exchanged with the browser client (spec §13.3).
// Every message is a map with a "t" (type) key; short field names keep packets small.
package netproto

import (
	"github.com/vmihailenco/msgpack/v5"

	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/telemetry"
)

// Client → server message types.
const (
	TJoin  = "join"
	TInput = "in"
	TPing  = "ping"
	TDebug = "dbg"
	// TTelemetry: mẫu hiệu năng của người chơi thật (TELEMETRY_SPEC §3). Gói "lạnh" — server
	// KHÔNG xử lý nó trong vòng tick, chỉ đẩy sang goroutine ghi.
	TTelemetry = "tm"
	// TDescRequest: client xin gửi lại MÔ TẢ (tên + nhân vật) của một entity
	// (NETCODE_SPEC §5 Hiển thị mục 2).
	TDescRequest = "dreq"
)

// Envelope is decoded first to find the message type.
type Envelope struct {
	T string `msgpack:"t"`
}

type Join struct {
	T       string `msgpack:"t"`
	Name    string `msgpack:"name"`
	Char    string `msgpack:"char"`
	GuestID string `msgpack:"guest_id"`
	// Debug: client mở bằng ?debug=true nên muốn bật bảng phím test. Server chỉ chấp nhận
	// khi chính nó cũng chạy với -debug, và chỉ người tự thêm tham số mới thấy bảng này.
	Debug bool `msgpack:"debug,omitempty"`
}

// Input carries the movement direction quantised to 256 steps (angle_u8).
type Input struct {
	T      string `msgpack:"t"`
	Seq    uint32 `msgpack:"s"`
	Angle  uint8  `msgpack:"a"`
	Moving bool   `msgpack:"m"`
	Burn   bool   `msgpack:"b"` // đang giữ Thiêu Đốt Tinh Huyết (spec §4.1c)
	// Skill: người chơi vừa bấm nút kỹ năng (Space / nút trên điện thoại), spec §6.3.
	Skill bool `msgpack:"sk,omitempty"`
}

type Ping struct {
	T      string  `msgpack:"t"`
	Client float64 `msgpack:"c"`
}

// Debug commands are only honoured when the server runs with -debug.
type Debug struct {
	T   string `msgpack:"t"`
	Cmd string `msgpack:"cmd"`
	N   int    `msgpack:"n"`
}

// ObstacleState is sent in Welcome for obstacles that are not intact.
type ObstacleState struct {
	ID     int  `msgpack:"id"`
	HP     int  `msgpack:"hp"`
	Broken bool `msgpack:"b"`
}

// ProtocolVersion: tăng số này MỖI KHI đổi hình dạng gói tin (thêm/bớt/đổi kiểu một
// trường). Client so với số của chính nó và báo "tải lại trang" nếu lệch.
//
// Không có nó thì client mới gặp server cũ sẽ đọc rác rồi **đứng hình không báo gì** — đã
// dính đúng một lần: client build 16:40 nói `mood` là chỉ số, server chạy từ 14:56 vẫn gửi
// chuỗi, và người chơi chỉ thấy game treo.
// Thêm hẳn một LOẠI gói mới (như `tm` telemetry) thì không phải đổi hình dạng gói cũ, nên
// không tăng số: client cũ chỉ là không bao giờ gửi `tm`, còn server cũ gặp `tm` thì rơi vào
// nhánh `default` và bỏ qua. Cả hai chiều đều an toàn, không ai đọc rác.
const ProtocolVersion = 4

// Telemetry chở thông tin phiên, mẫu 10 giây và sự kiện (TELEMETRY_SPEC §2, §3).
//
// Ba thứ đi chung một gói để tiết kiệm: mỗi 10s chỉ có đúng một gói rời client, và lần đầu
// nó chở luôn thông tin phiên thay vì tốn thêm một gói riêng.
type Telemetry struct {
	T       string              `msgpack:"t"`
	Session *telemetry.Session  `msgpack:"s,omitempty"`
	Samples []*telemetry.Sample `msgpack:"m,omitempty"`
	Events  []*telemetry.Event  `msgpack:"e,omitempty"`
}

// DescRequest: client thiếu mô tả của entity nên không biết vẽ nhân vật nào.
//
// Vì sao cần: tên và nhân vật chỉ gửi LẦN ĐẦU client thấy một người; server nhớ bằng
// `knownPlayers`. Nếu client vì bất kỳ lý do gì mà mất mô tả đó (vứt view, mất gói, tải lại
// atlas), server không hề biết và sẽ KHÔNG BAO GIỜ gửi lại — client kẹt vĩnh viễn với một
// entity không biết vẽ gì. Trước đây nó vẽ placeholder; spec §5 cấm, nên phải hỏi lại.
type DescRequest struct {
	T   string   `msgpack:"t"`
	IDs []uint32 `msgpack:"ids"`
}

type Welcome struct {
	T         string          `msgpack:"t"`
	ID        uint32          `msgpack:"id"`
	Room      string          `msgpack:"room"`
	Map       string          `msgpack:"map"`
	TickHz    int             `msgpack:"tick_hz"`
	Time      float64         `msgpack:"time"`
	Obstacles []ObstacleState `msgpack:"obs"`
	Debug     bool            `msgpack:"debug"`
	Proto     int             `msgpack:"pv"`
	// Zone / ZoneName: khu đang ở (spec §8.0). Gửi lại mỗi lần sang khu khác, vì sang khu
	// là đổi hẳn bản đồ.
	Zone     int    `msgpack:"zn"`
	ZoneName string `msgpack:"znm,omitempty"`
}

// Player flags.
const (
	FlagInvuln = 1 << iota
	FlagBush
	FlagStealth
	FlagTransit
	FlagDead
	FlagBot
	FlagRevealed
	FlagBurning // Thiêu Đốt Tinh Huyết đang bật (spec §4.1c)
)

type PlayerState struct {
	ID     uint32  `msgpack:"id"`
	X      float32 `msgpack:"x"`
	Y      float32 `msgpack:"y"`
	Facing float32 `msgpack:"f"`
	Level  int     `msgpack:"L"`
	Count  int     `msgpack:"n"`
	Phase  float32 `msgpack:"ph"`
	Dir    int8    `msgpack:"d"`
	Tilt   float32 `msgpack:"tl"` // blade lean (rad); ring geometry is derived from scale, count and buffs (spec §13.3)
	Scale  float32 `msgpack:"sc"`
	// Mood gửi dạng CHỈ SỐ thay vì chuỗi (spec mục 27). Chuỗi "breakthrough" tốn 13 byte mỗi
	// người mỗi gói, 20 lần/giây — mà nó hầu như không đổi.
	Mood  uint8 `msgpack:"md"`
	Buffs uint8 `msgpack:"bf"`
	Flags uint8 `msgpack:"fl"`
	// St: trạng thái do kỹ năng gây ra, mỗi bit một trạng thái (spec §6.3, game.StatusNames).
	// Gửi cho mọi người vì client phải vẽ "zzz", mai rùa, vệt lướt… của người khác.
	St    uint16  `msgpack:"st,omitempty"`
	Crack float32 `msgpack:"ck,omitempty"`
	// Exp chỉ gửi cho CHÍNH người đó: không ai cần biết EXP của người khác, HUD chỉ vẽ của
	// mình. Trước đây gửi cho tất cả, tốn 7 byte × (số người thấy được) mỗi gói.
	Exp float32 `msgpack:"xp,omitempty"`
	// Vx/Vy: vận tốc, chỉ gửi cho CHÍNH người đó và chỉ khi đang trượt trên băng (§8.0).
	// Client cần nó để phát lại dự đoán từ đúng vận tốc server đang giữ; thiếu thì mỗi lần
	// nhận snapshot là dự đoán lại từ vận tốc 0 và nhân vật giật trên cả mặt hồ.
	Vx float32 `msgpack:"vx,omitempty"`
	Vy float32 `msgpack:"vy,omitempty"`
	// Charges: số lượt kỹ năng đang giữ, chỉ gửi cho chính chủ (spec §6.3).
	Charges int `msgpack:"skc,omitempty"`
	Title   int `msgpack:"ti,omitempty"` // room rank badge 1..3 (spec §11.3), 0 = none
	// Tiên giới (spec §3.4): sát mạng đã có / cần, để HUD vẽ thanh Tiên khí. Chỉ gửi cho
	// chính người đó; người khác chỉ cần Level để biết cảnh giới và màu hào quang.
	TienHave float32 `msgpack:"th,omitempty"`
	TienNeed float32 `msgpack:"tn,omitempty"`
	Name     string  `msgpack:"nm,omitempty"` // only the first time a client sees this player
	Char     string  `msgpack:"ch,omitempty"`
}

// MonsterState: `k`, `mx`, `r` KHÔNG BAO GIỜ đổi trong đời một con quái, nên chỉ gửi lần đầu
// client thấy nó (spec mục 27). Trước đây gửi lại 20 lần/giây: riêng chuỗi `kind` đã 10 byte
// mỗi con mỗi gói.
// Toạ độ quái và vật phẩm gửi dạng **số nguyên đơn vị thế giới** (`uint16`), không phải
// `float32` (spec mục 27 "lượng tử hoá").
//
// Bản đồ rộng 12 000 nên lọt gọn trong uint16, và sai số tối đa nửa đơn vị — trên màn hình
// là 1/60 bán kính thân người, không ai thấy. Đổi lại mỗi toạ độ còn 3 byte thay vì 5.
//
// ⚠ Cố ý **không** lượng tử hoá toạ độ NGƯỜI CHƠI: client dùng vị trí server gửi về để đối
// chiếu với phần nó tự dự đoán (§8.2a). Thêm nửa đơn vị sai số vào đó là mỗi gói lại lệch
// một ít, và bộ đếm "hiệu chỉnh/giây" vừa kéo về 0 sẽ chạy lại. Người chơi cũng chỉ chiếm
// 11% gói, không đáng để đánh đổi.
type MonsterState struct {
	ID uint32 `msgpack:"id"`
	X  uint16 `msgpack:"x"`
	Y  uint16 `msgpack:"y"`
	// HP: **chỉ gửi khi ĐỔI** (spec mục 27 "delta snapshot"). Thiếu trường này = máu giữ
	// nguyên. Quái chết bị xoá khỏi danh sách nên máu luôn ≥ 1, không nhầm với 0 được.
	HP   int     `msgpack:"hp,omitempty"`
	Kind string  `msgpack:"k,omitempty"`
	Max  int     `msgpack:"mx,omitempty"`
	R    float32 `msgpack:"r,omitempty"`
}

type ItemState struct {
	ID uint32 `msgpack:"id"`
	X  uint16 `msgpack:"x"`
	Y  uint16 `msgpack:"y"`
	// Kind gửi dạng CHỈ SỐ trong `ItemKinds`, không phải chuỗi: "skill_scroll" tốn 13 byte
	// mỗi viên mỗi gói, mà một gói có thể có 60 vật phẩm mới vào tầm nhìn.
	Kind  uint8   `msgpack:"k"`
	Value float32 `msgpack:"v,omitempty"`
}

// ItemKinds: thứ tự này là GIAO THỨC. Thêm loại mới thì thêm vào CUỐI, đừng chèn giữa —
// client tra theo chỉ số (`client/src/net/protocol.ts` ITEM_KINDS phải khớp từng dòng).
var ItemKinds = []string{"orb", "sword", "bundle", "chest", "tien_khi", "skill_scroll",
	"spin", "haste", "magnet", "hard", "shield", "exp2", "stealth"}

var itemKindIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(ItemKinds))
	for i, k := range ItemKinds {
		m[k] = uint8(i)
	}
	return m
}()

// ItemKindID tra chỉ số của một loại vật phẩm. Loại lạ (chưa thêm vào bảng) trả 0 và sẽ vẽ
// như viên Linh khí — xấu nhưng không làm sập client.
func ItemKindID(kind string) uint8 { return itemKindIndex[kind] }

// QuantPos kẹp và làm tròn một toạ độ thế giới về uint16.
func QuantPos(v float64) uint16 {
	switch {
	case v <= 0:
		return 0
	case v >= 65535:
		return 65535
	default:
		return uint16(v + 0.5)
	}
}

// Snapshot is sent at snapshot_hz. Items are diffed per client (added / removed).
type Snapshot struct {
	T            string         `msgpack:"t"`
	Tick         uint32         `msgpack:"tk"`
	Time         float64        `msgpack:"tm"`
	Ack          uint32         `msgpack:"ack"`
	Players      []PlayerState  `msgpack:"p"`
	Monsters     []MonsterState `msgpack:"m"`
	ItemsAdded   []ItemState    `msgpack:"ia,omitempty"`
	ItemsRemoved []uint32       `msgpack:"ir,omitempty"`
	Events       []game.Event   `msgpack:"ev,omitempty"`
	// SrvMs: thời gian server xử lý một tick của phòng, mili-giây. Chỉ để đo (spec §13.6),
	// client hiện lên overlay F3 để biết nghẽn nằm ở server hay ở máy mình.
	SrvMs float32 `msgpack:"sv,omitempty"`
}

type LeaderEntry struct {
	ID    uint32  `msgpack:"id"`
	Name  string  `msgpack:"nm"`
	Exp   float32 `msgpack:"xp"`
	Level int     `msgpack:"L"`
	Score float32 `msgpack:"sc"`
	Title int     `msgpack:"ti,omitempty"`
}

// Leaderboard goes out once a second. It also carries the minimap's live markers, which are
// the only positions a client is allowed to know outside its own view (spec §12.1).
type Leaderboard struct {
	// ZoneCounts: số nhân vật ở mỗi khu, cho bảng phóng to bằng phím M (spec §8.0).
	ZoneCounts []int         `msgpack:"zc,omitempty"`
	T          string        `msgpack:"t"`
	Top        []LeaderEntry `msgpack:"top"`
	Rank       int           `msgpack:"rank"`
	Total      int           `msgpack:"total"`
	MyScore    float32       `msgpack:"ms"`
	// Top1 is the leader's position for the minimap crown; absent while they are hiding.
	Top1    bool    `msgpack:"t1,omitempty"`
	Top1X   float32 `msgpack:"t1x,omitempty"`
	Top1Y   float32 `msgpack:"t1y,omitempty"`
	HasBau  bool    `msgpack:"ch,omitempty"` // a Rương Báu is on the map (or warned)
	ChestX  float32 `msgpack:"chx,omitempty"`
	ChestY  float32 `msgpack:"chy,omitempty"`
	ChestIn bool    `msgpack:"chw,omitempty"` // still in the 10s warning, not spawned yet
	// Đài Tiên Khí cho minimap (spec §8.5, §12.1): x, y, kind, đang sắp phun lớn.
	Altars []AltarMark `msgpack:"al,omitempty"`
}

// AltarMark is one Đài Tiên Khí on the minimap.
type AltarMark struct {
	X    float32 `msgpack:"x"`
	Y    float32 `msgpack:"y"`
	Kind string  `msgpack:"k"`
	Warn bool    `msgpack:"w,omitempty"`
}

// Feed is a kill-feed line with names resolved on the server (spec §7).
// Top > 0 means the victim was that high in their room board and the line was announced
// to every room on the server (spec §11.4).
type Feed struct {
	T     string `msgpack:"t"`
	Kind  string `msgpack:"k"`
	A     string `msgpack:"a"`
	B     string `msgpack:"b,omitempty"`
	Realm int    `msgpack:"r"`
	Top   int    `msgpack:"tp,omitempty"`
	Room  string `msgpack:"rm,omitempty"` // phòng xảy ra, chỉ gửi khi loan toàn server
	// Zone: khu xảy ra chuyện (spec §8.0). Client thêm tiền tố "[tên khu]" khi tin đến từ
	// khu khác — không có nó thì người chơi tưởng vừa có người chết ngay cạnh mình.
	Zone     int    `msgpack:"zn"`
	ZoneName string `msgpack:"znm,omitempty"`
}

// Dead is the Tổng kết screen payload (spec §7, §11.1).
type Dead struct {
	T           string             `msgpack:"t"`
	KillerName  string             `msgpack:"killer"`
	KillerChar  string             `msgpack:"killer_char"`
	PeakLevel   int                `msgpack:"peak_level"`
	PeakExp     float64            `msgpack:"peak_exp"`
	Kills       int                `msgpack:"kills"`
	SurviveSec  float64            `msgpack:"survive"`
	Rank        int                `msgpack:"rank"`
	Score       float64            `msgpack:"score"`
	Hits        int                `msgpack:"hits"`
	BurnedExp   float64            `msgpack:"burned_exp"` // EXP đã đốt (spec §4.1c)
	ScoreBy     map[string]float64 `msgpack:"score_by"`   // điểm theo từng nguồn (§11.1)
	DaiThuaSecs float64            `msgpack:"dai_thua_secs"`
	Top1Secs    float64            `msgpack:"top1_secs"`
}

type Pong struct {
	T      string  `msgpack:"t"`
	Client float64 `msgpack:"c"`
	Server float64 `msgpack:"s"`
}

type Error struct {
	T   string `msgpack:"t"`
	Msg string `msgpack:"msg"`
}

func Encode(v any) ([]byte, error) { return msgpack.Marshal(v) }

func Decode(b []byte, v any) error { return msgpack.Unmarshal(b, v) }
