package room

import "math"

// Vùng quan tâm có TRỄ và vòng đời entity tường minh — NETCODE_SPEC §4.3, §4.4.
//
// Đây là thứ bước 2b còn thiếu. Bước 2b vá ở client: "vắng mặt quá 2 giây thì coi như đã
// đi". Cách đó không bao giờ đạt được ngân sách respawn = 0, vì client KHÔNG PHÂN BIỆT
// ĐƯỢC "server bỏ qua mình ở tick lẻ" với "người đó rời đi thật". Đo trên `:9999` sau 2b:
// placeholder đã về 0/16 lượt, nhưng respawn vẫn 2/16 lượt có 1.
//
// Hai thay đổi đóng nốt khoảng hở đó:
//
//  1. **Hai bán kính** (§4.4). Vào khung nhìn ở bán kính nhỏ, ra khỏi ở bán kính LỚN HƠN.
//     Một bán kính duy nhất nghĩa là người đứng đúng mép cứ vào–ra–vào–ra mỗi tick.
//  2. **`Despawn` tường minh** (§4.3). Server nói rõ "người này đi rồi", nên client không
//     phải đoán nữa.

const (
	// aoiEnterMargin / aoiExitMargin: lề vào và lề ra, tính bằng đơn vị thế giới (§4.4).
	// Lề ra LỚN HƠN lề vào — đó chính là phần "trễ" chống rung ở mép.
	aoiEnterMargin = 400.0
	aoiExitMargin  = 900.0
	// aoiLingerSecs: ra khỏi bán kính ngoài rồi còn GIỮ thêm ngần này giây mới `Despawn`.
	// Vào lại trong khoảng đó thì chỉ `Update` tiếp, không `Spawn` lại (§4.4).
	aoiLingerSecs = 2.0
	// keyframeSecs: mỗi ngần này giây gửi lại state ĐẦY ĐỦ của entity trong tầm, để tự sửa
	// sai lệch tích luỹ của chuỗi delta (§4.3).
	keyframeSecs = 2.0
)

// DespawnReason theo §4.3.
const (
	DespawnOutOfView = 0
	DespawnDied      = 1
	DespawnTeleport  = 2
	DespawnLeft      = 3
)

// tracked là những gì server NHỚ đã gửi cho một client về một entity.
//
// Giữ bản state cuối đã gửi chứ không chỉ một cờ "đã biết": delta chỉ tính được khi biết
// client đang giữ con số nào.
type tracked struct {
	// leftAt: lúc entity ra khỏi bán kính NGOÀI. 0 nghĩa là đang trong tầm.
	leftAt float64
	// lastFull: lần cuối gửi state đầy đủ (spawn hoặc keyframe).
	lastFull float64
	// sent là bản state cuối client đã nhận — gốc để tính delta.
	sent entitySnap
}

// entitySnap là state đã LƯỢNG TỬ của một entity, đúng như nó đi trên dây (§4.6).
//
// Lượng tử TRƯỚC khi so sánh, không phải sau: so bằng số thực thì vị trí nhúc nhích
// 0.001 đơn vị cũng tính là "đã đổi" và delta không tiết kiệm được gì.
type entitySnap struct {
	X, Y      uint16
	Facing    uint8
	Level     uint8
	Scale     uint8
	Swords    uint16
	RingPhase uint16
	RingDir   int8
	Mood      uint8
	Buffs     uint16
	Flags     uint16
}

// changedMask trả về bitmask những trường khác nhau, theo đúng thứ tự bit của
// `proto/schema.txt` message `Update`.
func (a entitySnap) changedMask(b entitySnap) uint16 {
	var m uint16
	set := func(bit int, diff bool) {
		if diff {
			m |= 1 << bit
		}
	}
	set(0, a.X != b.X)
	set(1, a.Y != b.Y)
	set(2, a.Facing != b.Facing)
	set(3, a.Level != b.Level)
	set(4, a.Scale != b.Scale)
	set(5, a.Swords != b.Swords)
	set(6, a.RingPhase != b.RingPhase)
	set(7, a.RingDir != b.RingDir)
	set(8, a.Mood != b.Mood)
	set(9, a.Buffs != b.Buffs)
	set(10, a.Flags != b.Flags)
	return m
}

// LifecycleKind là loại sự kiện vòng đời.
type LifecycleKind uint8

const (
	LifeSpawn LifecycleKind = iota
	LifeUpdate
	LifeDespawn
)

// Lifecycle là một sự kiện gửi cho client về một entity.
type Lifecycle struct {
	Kind   LifecycleKind
	ID     uint32
	Snap   entitySnap
	Mask   uint16 // chỉ dùng với LifeUpdate
	Reason uint8  // chỉ dùng với LifeDespawn
}

// aoiTracker giữ, cho MỘT client, biết đã gửi gì về từng entity.
type aoiTracker struct {
	seen map[uint32]*tracked
}

func newAOITracker() *aoiTracker { return &aoiTracker{seen: map[uint32]*tracked{}} }

// entityIn mô tả một entity ứng viên và khoảng cách của nó tới người xem.
type entityIn struct {
	ID   uint32
	Dist float64 // khoảng cách Chebyshev đã trừ nửa khung nhìn: ≤0 là trong khung
	Snap entitySnap
	// Gone: entity không còn tồn tại (chết / rời phòng). Luôn `Despawn` ngay, không chờ trễ.
	Gone   bool
	Reason uint8
}

// Step tính danh sách sự kiện vòng đời cần gửi cho client này.
//
// `now` là thời gian game. Hàm thuần: cùng đầu vào cho cùng đầu ra, nên test được bằng số
// mà không cần dựng cả một phòng.
func (t *aoiTracker) Step(now float64, ents []entityIn) []Lifecycle {
	var out []Lifecycle
	alive := make(map[uint32]bool, len(ents))

	for _, e := range ents {
		alive[e.ID] = true
		st, known := t.seen[e.ID]

		if e.Gone {
			if known {
				delete(t.seen, e.ID)
				out = append(out, Lifecycle{Kind: LifeDespawn, ID: e.ID, Reason: e.Reason})
			}
			continue
		}

		inEnter := e.Dist <= aoiEnterMargin
		inExit := e.Dist <= aoiExitMargin

		if !known {
			// CHƯA biết: chỉ nhận vào khi lọt bán kính VÀO. Dùng bán kính ra để nhận thì
			// hai bán kính thành vô nghĩa.
			if !inEnter {
				continue
			}
			t.seen[e.ID] = &tracked{lastFull: now, sent: e.Snap}
			out = append(out, Lifecycle{Kind: LifeSpawn, ID: e.ID, Snap: e.Snap})
			continue
		}

		if !inExit {
			// Ra khỏi bán kính NGOÀI: bấm giờ, chưa `Despawn` vội.
			if st.leftAt == 0 {
				st.leftAt = now
			}
			if now-st.leftAt >= aoiLingerSecs {
				delete(t.seen, e.ID)
				out = append(out, Lifecycle{Kind: LifeDespawn, ID: e.ID, Reason: DespawnOutOfView})
			}
			// Trong lúc chờ vẫn KHÔNG gửi update: người đó đã ngoài tầm nhìn, gửi là phí.
			continue
		}

		// Vào lại trong lúc chờ → chỉ tiếp tục, KHÔNG `Spawn` lại (§4.4). Đây đúng là chỗ
		// bước 2b làm sai và sinh ra "respawn trong 2 giây".
		st.leftAt = 0

		if now-st.lastFull >= keyframeSecs {
			// Keyframe: gửi lại đầy đủ để sai lệch tích luỹ không trôi mãi (§4.3).
			st.lastFull = now
			st.sent = e.Snap
			out = append(out, Lifecycle{Kind: LifeSpawn, ID: e.ID, Snap: e.Snap})
			continue
		}

		if m := st.sent.changedMask(e.Snap); m != 0 {
			st.sent = e.Snap
			out = append(out, Lifecycle{Kind: LifeUpdate, ID: e.ID, Snap: e.Snap, Mask: m})
		}
		// Không đổi gì thì KHÔNG gửi gì cả. Đó là phần tiết kiệm lớn nhất: phòng đông thì
		// phần lớn entity đứng yên hoặc chỉ nhúc nhích dưới một bước lượng tử.
	}

	// Entity biến mất khỏi danh sách (đổi khu, rời phòng) mà không ai báo: coi như đã đi.
	for id := range t.seen {
		if !alive[id] {
			delete(t.seen, id)
			out = append(out, Lifecycle{Kind: LifeDespawn, ID: id, Reason: DespawnLeft})
		}
	}
	return out
}

// Forget xoá hẳn trí nhớ (đổi khu, kết nối lại) — lần sau mọi entity đều `Spawn` lại.
func (t *aoiTracker) Forget() { t.seen = map[uint32]*tracked{} }

// Known là số entity client đang giữ — overlay F3 và test đọc.
func (t *aoiTracker) Known() int { return len(t.seen) }

// chebyshevOutside trả về khoảng cách từ `pos` ra NGOÀI khung nhìn (0 nếu đang ở trong).
//
// Dùng Chebyshev (lớn hơn trong hai trục) chứ không dùng Euclid: khung nhìn là hình chữ
// nhật, và khoảng cách Euclid sẽ nhận vào người nằm ngoài góc màn hình.
func chebyshevOutside(dx, dy, hw, hh float64) float64 {
	ox := math.Abs(dx) - hw
	oy := math.Abs(dy) - hh
	return math.Max(0, math.Max(ox, oy))
}
