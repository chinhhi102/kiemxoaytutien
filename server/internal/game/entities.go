package game

import "kiemxoay/server/internal/geom"

type Vec = geom.Vec

// Buff kinds (spec §6). Their order is the bit order sent to clients.
const (
	BuffSpin = iota
	BuffHaste
	BuffMagnet
	BuffHard
	BuffShield
	BuffExp2
	BuffStealth
	BuffCount
)

var BuffNames = [BuffCount]string{"spin", "haste", "magnet", "hard", "shield", "exp2", "stealth"}

// BuffByName maps item kind → buff index.
func BuffByName(name string) (int, bool) {
	for i, n := range BuffNames {
		if n == name {
			return i, true
		}
	}
	return 0, false
}

// Mood names = animation names of spec §12.
const (
	MoodIdle         = "idle"
	MoodRun          = "run"
	MoodChase        = "chase"
	MoodFlee         = "flee"
	MoodHit          = "hit"
	MoodLevelup      = "levelup"
	MoodBreakthrough = "breakthrough"
	MoodRealmDrop    = "realm_drop"
	MoodTeleportOut  = "teleport_out"
	MoodTeleportIn   = "teleport_in"
	MoodDeath        = "death"
	MoodRespawn      = "respawn"
)

// MoodList là thứ tự dùng để gửi tâm trạng dạng CHỈ SỐ thay vì chuỗi (spec mục 27). Client
// có cùng danh sách này; **thêm vào CUỐI**, đừng chèn giữa, nếu không hai bên lệch nhau.
var MoodList = [...]string{
	MoodIdle, MoodRun, MoodChase, MoodFlee, MoodHit, MoodLevelup,
	MoodBreakthrough, MoodRealmDrop, MoodTeleportOut, MoodTeleportIn, MoodDeath, MoodRespawn,
}

// MoodIndex đổi tên tâm trạng thành chỉ số; tên lạ thì trả 0 (idle).
func MoodIndex(name string) uint8 {
	for i, m := range MoodList {
		if m == name {
			return uint8(i)
		}
	}
	return 0
}

// Transit tracks a player moving through a gate (spec §8.4).
type Transit struct {
	Dest     Vec
	DestGate int
	// DestZone: khu đích (spec §8.4 — 20 cổng của cả thế giới nằm chung một mạng).
	DestZone int
	OutAt    float64 // time the player reappears at Dest
	DoneAt   float64 // time transit (immunity from combat) ends
	Arrived  bool
	WarnSent bool
}

type Player struct {
	ID      uint32
	Name    string
	Char    string
	GuestID string
	IsBot   bool

	Pos        Vec
	Facing     float64
	InputAngle float64
	Moving     bool
	// Burning: người chơi đang giữ phím Thiêu Đốt Tinh Huyết (spec §4.1c).
	Burning   bool
	BurnedExp float64 // tổng EXP đã đốt trong mạng này, hiện ở màn Tổng kết
	LastSeq   uint32

	TotalExp float64
	Level    int
	Ring     Ring
	Geom     RingGeom
	Scale    float64
	// ViewHW/ViewHH: nửa khung nhìn, tính sẵn trong refreshDerived (xem refreshView).
	// Đừng đọc thẳng — gọi g.ViewHalfExtents(p) để nó tự tính nếu chưa có.
	// Vel: vận tốc hiện tại, CHỈ dùng trên mặt băng (spec §8.0). Ngoài băng luôn bằng 0 —
	// trên đất thả phím là dừng ngay, không có quán tính.
	Vel    Vec
	ViewHW float64
	ViewHH float64

	PeakExp   float64
	PeakLevel int
	Kills     int
	SpawnedAt float64

	// Uy Danh scoring for this life (spec §11.1).
	Score        float64
	ScoreBy      map[string]float64 // points per source, for the Tổng kết breakdown
	Hits         int                // body hits landed on other players (bảng Sát Thần)
	HitCooldown  map[string]float64 // victim key → time the same victim scores again
	ScoredRealms map[int]bool       // realms already rewarded this life
	Top1Secs     float64            // seconds held as room top 1
	DaiThuaSecs  float64            // seconds from spawn to Đại Thừa; 0 = never reached
	// TienKhi: Tiên khí cộng dồn — thứ duy nhất đưa người chơi lên Tiên giới (spec §3.4 v4).
	TienKhi float64

	// Title is the room rank badge: 0 = none, 1..3 = Top 1..3 (spec §11.3).
	Title        int
	pendingTitle int
	titleSince   float64

	InvulnUntil float64
	Buffs       [BuffCount]float64 // expiry time; 0 = inactive
	// Status: trạng thái có thời hạn do kỹ năng gây ra (spec §6.3, xem status.go).
	Status [StatusCount]float64
	// SkillCharges: số lượt dùng kỹ năng đang giữ, nhặt Bí Kíp mà có (§6.3). Tối đa 2.
	SkillCharges int
	// SkillReadyAt: khoá 3 giây giữa hai lần dùng (§6.3 — không có hồi chiêu tự động).
	SkillReadyAt float64
	// shellOn: đang trong mai rùa — dùng để bắt đúng khoảnh khắc RA khỏi mai.
	shellOn bool
	// ghostOn + unstick*: Giao Hỏa Tốc hết hạn giữa lòng vật cản thì trượt mượt ra chỗ
	// trống trong 0,3 giây (spec v5.5 §6.3). Xem `updateUnstick`.
	ghostOn      bool
	unstickFrom  Vec
	unstickTo    Vec
	unstickUntil float64
	// Đàn em của Đại Ca (§6.3): sống có hạn, không chém thân được.
	IsMinion    bool
	MinionOf    uint32
	MinionUntil float64
	KnockVel    Vec
	KnockUntil  float64

	Dead       bool
	DiedAt     float64
	KilledBy   uint32
	KillerName string
	KillerChar string

	Mood      string
	MoodUntil float64 // forced moods (hit, levelup…) hold until this time
	MoodSince float64

	Bush           int     // bush zone id, -1 when not in a bush
	RevealedUntil  float64 // bush reveal after being hit
	GateReadyAt    float64
	lastTeleportIn float64
	Transit        *Transit
	EmoteReady     map[string]float64

	// Kiếm bể dồn lại rồi mới báo một lần (xem flushSwordBreaks). Bể kiếm tính mỗi tick nên
	// nếu mỗi cây gãy bắn một sự kiện thì 20 người đánh nhau đẻ ra ~1900 sự kiện/giây —
	// server vẫn nhẹ nhưng client phải dựng từng ấy hiệu ứng + tiếng động và tụt khung hình.
	breakPending int
	// orbPicked*: gộp lượt nhặt viên trong một tick (xem `flushOrbPickups`).
	orbPicked     int
	orbPickedID   uint32
	orbPickedKind string
	breakFlushAt  float64
}

func (p *Player) HasBuff(b int, now float64) bool { return p.Buffs[b] > now }

func (p *Player) Invulnerable(now float64) bool { return p.InvulnUntil > now }

// InTransit is true while the player is inside a gate and immune to combat.
func (p *Player) InTransit(now float64) bool { return p.Transit != nil && now < p.Transit.DoneAt }

// Hidden reports whether the player is out of sight because they are mid-teleport.
func (p *Player) Hidden(now float64) bool { return p.Transit != nil && !p.Transit.Arrived }

type Monster struct {
	ID        uint32
	Kind      string
	Pos       Vec
	R         float64
	HP, MaxHP int
	Exp       float64
	Heading   float64
	TurnAt    float64
	FleeUntil float64
	FleeFrom  Vec
}

// Item kinds: "sword", "bundle" (Value = swords), buff names, "orb" (EXP orb, Value = EXP), "chest".
type Item struct {
	ID         uint32
	Kind       string
	Pos        Vec
	Value      float64
	Region     int    // ambient EXP orb: respawn region index (spec §6.1); -1 otherwise
	Pool       bool   // part of the room's count_per_room item pool, respawns respawn_secs after pickup
	Owner      uint32 // only this player may pick it up until OwnerUntil
	OwnerUntil float64
	ExpireAt   float64 // > 0: viên tự tan vào lúc này (Tiên khí, spec §8.5)

	// cell/slot: vị trí của viên này trong itemGrid, để xoá O(1) mà không phải dò.
	cell int
	slot int
	// slotIdx: chỉ số dày đặc toàn cục (xem Game.itemSlots), để client nhớ bằng slice.
	slotIdx int
}

// ObstacleState is the dynamic state of a destructible obstacle (spec §8.2).
type ObstacleState struct {
	HP       int
	Partial  float64 // leftover fractional damage (spec §5.3 damage is not a whole number)
	Broken   bool
	BrokenAt float64
	Warned   bool
}
