package room

import "testing"

// Bài kiểm cho vòng đời entity (NETCODE_SPEC §4.3, §4.4).
//
// Mục tiêu đo được của bước 3: **respawn trong 2 giây = 0**. Bước 2b đã đưa placeholder về
// 0/16 lượt nhưng respawn vẫn 2/16 lượt có 1, vì client phải ĐOÁN khi nào một người đã đi.
// Những test dưới đây canh đúng chỗ đó.

func snapAt(x, y uint16) entitySnap { return entitySnap{X: x, Y: y} }

// collect gom sự kiện theo loại cho dễ đọc trong thông báo lỗi.
func kinds(evs []Lifecycle) []LifecycleKind {
	out := make([]LifecycleKind, len(evs))
	for i, e := range evs {
		out[i] = e.Kind
	}
	return out
}

func TestVaoKhungThiSpawnMotLan(t *testing.T) {
	tr := newAOITracker()
	// Ngoài cả hai bán kính: không có gì.
	if evs := tr.Step(0, []entityIn{{ID: 1, Dist: 1000, Snap: snapAt(1, 1)}}); len(evs) != 0 {
		t.Fatalf("ngoài tầm mà vẫn gửi %v", kinds(evs))
	}
	// Lọt bán kính VÀO → Spawn.
	evs := tr.Step(0.1, []entityIn{{ID: 1, Dist: 100, Snap: snapAt(1, 1)}})
	if len(evs) != 1 || evs[0].Kind != LifeSpawn {
		t.Fatalf("mong 1 Spawn, nhận %v", kinds(evs))
	}
	// Đứng yên → KHÔNG gửi gì. Đây là phần tiết kiệm băng thông lớn nhất.
	if evs := tr.Step(0.2, []entityIn{{ID: 1, Dist: 100, Snap: snapAt(1, 1)}}); len(evs) != 0 {
		t.Fatalf("đứng yên mà vẫn gửi %v", kinds(evs))
	}
	// Nhúc nhích → Update, và chỉ bật đúng bit đã đổi.
	evs = tr.Step(0.3, []entityIn{{ID: 1, Dist: 100, Snap: snapAt(2, 1)}})
	if len(evs) != 1 || evs[0].Kind != LifeUpdate {
		t.Fatalf("mong 1 Update, nhận %v", kinds(evs))
	}
	if evs[0].Mask != 1 {
		t.Errorf("chỉ X đổi thì mask phải là 0b1, nhận %b", evs[0].Mask)
	}
}

// ĐÂY LÀ TEST QUAN TRỌNG NHẤT của bước 3, và nó canh đúng thứ HAI BÁN KÍNH sinh ra để làm:
// một người đứng GIỮA hai bán kính thì không bao giờ bị Despawn.
//
// Bản đầu tôi viết test này bằng cách cho entity đi qua đi lại giữa `aoiEnterMargin-10` và
// `aoiExitMargin-10`, rồi thử bỏ hai bán kính đi để xem test có đỏ không — nó vẫn XANH. Lý
// do: đặt hai lề bằng nhau thì hai mốc khoảng cách kia cũng bằng nhau, test không còn chạm
// tới ranh giới nào cả. Một test không đỏ được khi gỡ mất thứ nó canh là một test vô dụng.
func TestDungGiuaHaiBanKinhThiKhongBiDespawn(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}}) // vào tầm

	// Khoảng cách CỐ ĐỊNH, không tính từ hằng số.
	//
	// Lần trước tôi viết `mid := (aoiEnterMargin + aoiExitMargin) / 2` — và khi thử bỏ hai
	// bán kính đi để xem test có đỏ không, `mid` co theo luôn nên test vẫn xanh. Một test
	// tự điều chỉnh theo thứ nó đang canh thì canh cái gì cũng được.
	const mid = 600 // ngoài lề vào (400), trong lề ra (900)
	if mid <= aoiEnterMargin || mid >= aoiExitMargin {
		t.Fatalf("mốc %d phải nằm GIỮA hai lề (%.0f, %.0f) — sửa hằng số thì sửa cả mốc này",
			mid, aoiEnterMargin, aoiExitMargin)
	}
	despawns := 0
	for i := 1; i <= 200; i++ {
		for _, e := range tr.Step(float64(i)*0.05, []entityIn{{ID: 1, Dist: float64(mid), Snap: snapAt(uint16(i), 1)}}) {
			if e.Kind == LifeDespawn {
				despawns++
			}
		}
	}
	if despawns != 0 {
		t.Errorf("đứng giữa hai bán kính mà bị Despawn %d lần — lề ra phải rộng hơn lề vào", despawns)
	}
	if tr.Known() != 1 {
		t.Errorf("phải vẫn còn nhớ entity, Known = %d", tr.Known())
	}
}

// Đi qua đi lại NGAY TẠI mép lề vào: phần này do hạn trễ 2 giây lo, không phải hai bán kính.
func TestDiQuaDiLaiOMepKhongSpawnLai(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}})

	spawns := 0
	for i := 0; i < 200; i++ {
		d := aoiEnterMargin - 10
		if i%2 == 1 {
			d = aoiExitMargin + 100 // ra hẳn ngoài CẢ hai lề
		}
		for _, e := range tr.Step(float64(i)*0.05, []entityIn{{ID: 1, Dist: d, Snap: snapAt(uint16(i), 1)}}) {
			if e.Kind == LifeSpawn {
				spawns++
			}
		}
	}
	// 200 tick × 0.05s = 10 giây → tối đa 5 keyframe. Nhiều hơn nghĩa là đang spawn lại thật.
	if spawns > 6 {
		t.Errorf("đi qua đi lại sinh ra %d lần gửi đầy đủ — phải chỉ còn keyframe", spawns)
	}
}

func TestRaHanTroiMoiDespawn(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}})

	far := []entityIn{{ID: 1, Dist: aoiExitMargin + 100, Snap: snapAt(1, 1)}}
	// Ra ngoài bán kính NGOÀI nhưng chưa quá hạn trễ: chưa được Despawn.
	for _, at := range []float64{0.1, 1.0, 1.9} {
		if evs := tr.Step(at, far); len(evs) != 0 {
			t.Fatalf("t=%.1f: mới ra %.1f giây mà đã %v", at, at, kinds(evs))
		}
	}
	// Quá hạn → Despawn, lý do out_of_view.
	evs := tr.Step(2.1, far)
	if len(evs) != 1 || evs[0].Kind != LifeDespawn {
		t.Fatalf("quá hạn trễ phải Despawn, nhận %v", kinds(evs))
	}
	if evs[0].Reason != DespawnOutOfView {
		t.Errorf("lý do = %d, mong out_of_view (%d)", evs[0].Reason, DespawnOutOfView)
	}
	if tr.Known() != 0 {
		t.Error("Despawn rồi mà vẫn còn nhớ")
	}
}

func TestVaoLaiTrongHanThiKhongSpawnLai(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}})
	// Ra ngoài 1.5 giây (chưa quá hạn 2s)…
	tr.Step(0.1, []entityIn{{ID: 1, Dist: aoiExitMargin + 100, Snap: snapAt(1, 1)}})
	tr.Step(1.5, []entityIn{{ID: 1, Dist: aoiExitMargin + 100, Snap: snapAt(1, 1)}})
	// …rồi vào lại: phải là Update, KHÔNG phải Spawn.
	evs := tr.Step(1.6, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(9, 9)}})
	for _, e := range evs {
		if e.Kind == LifeSpawn {
			t.Fatal("vào lại trong hạn trễ mà vẫn Spawn — đây đúng là lỗi nhấp nháy")
		}
	}
	// Và sau đó đồng hồ trễ phải được xoá: ra lại thì bấm giờ từ đầu, không Despawn ngay.
	if evs := tr.Step(1.7, []entityIn{{ID: 1, Dist: aoiExitMargin + 100, Snap: snapAt(9, 9)}}); len(evs) != 0 {
		t.Errorf("ra lại phải bấm giờ lại, nhận %v", kinds(evs))
	}
}

func TestChetThiDespawnNgayKhongChoTre(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}})
	// Chết là sự kiện TỨC THÌ: chờ 2 giây rồi mới xoá nghĩa là xác đứng im giữa màn hình.
	evs := tr.Step(0.1, []entityIn{{ID: 1, Dist: 0, Gone: true, Reason: DespawnDied}})
	if len(evs) != 1 || evs[0].Kind != LifeDespawn || evs[0].Reason != DespawnDied {
		t.Fatalf("chết phải Despawn ngay với lý do died, nhận %v", evs)
	}
}

func TestBienMatKhoiDanhSachCungDespawn(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}})
	// Entity không còn trong danh sách ứng viên (đã sang khu khác): phải báo đi, nếu không
	// client giữ mãi một hình đứng im.
	evs := tr.Step(0.1, nil)
	if len(evs) != 1 || evs[0].Kind != LifeDespawn || evs[0].Reason != DespawnLeft {
		t.Fatalf("mong Despawn/left, nhận %v", evs)
	}
}

func TestKeyframeDinhKy(t *testing.T) {
	tr := newAOITracker()
	tr.Step(0, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}})
	full := 0
	for i := 1; i <= 100; i++ { // 10 giây ở 10 Hz
		at := float64(i) * 0.1
		for _, e := range tr.Step(at, []entityIn{{ID: 1, Dist: 0, Snap: snapAt(1, 1)}}) {
			if e.Kind == LifeSpawn {
				full++
			}
		}
	}
	// 10 giây / 2 giây = 5 keyframe (±1 tuỳ mốc chia).
	if full < 4 || full > 6 {
		t.Errorf("10 giây phải có khoảng 5 keyframe, nhận %d", full)
	}
}

func TestMaskKhopThuTuBitCuaSchema(t *testing.T) {
	// Thứ tự bit PHẢI khớp `proto/schema.txt` message `Update`. Lệch một bit là client đọc
	// giá trị của trường khác — và sai im lặng, không có lỗi nào được ném ra.
	base := entitySnap{}
	cases := []struct {
		bit  int
		name string
		mod  func(*entitySnap)
	}{
		{0, "x", func(s *entitySnap) { s.X = 1 }},
		{1, "y", func(s *entitySnap) { s.Y = 1 }},
		{2, "facing", func(s *entitySnap) { s.Facing = 1 }},
		{3, "level", func(s *entitySnap) { s.Level = 1 }},
		{4, "scale", func(s *entitySnap) { s.Scale = 1 }},
		{5, "swords", func(s *entitySnap) { s.Swords = 1 }},
		{6, "ring_phase", func(s *entitySnap) { s.RingPhase = 1 }},
		{7, "ring_dir", func(s *entitySnap) { s.RingDir = 1 }},
		{8, "mood", func(s *entitySnap) { s.Mood = 1 }},
		{9, "buffs", func(s *entitySnap) { s.Buffs = 1 }},
		{10, "flags", func(s *entitySnap) { s.Flags = 1 }},
	}
	for _, c := range cases {
		other := base
		c.mod(&other)
		if m := base.changedMask(other); m != uint16(1<<c.bit) {
			t.Errorf("đổi %s cho mask %b, mong bit %d", c.name, m, c.bit)
		}
	}
}

func TestChebyshevKhongNhanNhamGocManHinh(t *testing.T) {
	// Khung nhìn là hình CHỮ NHẬT. Dùng khoảng cách Euclid thì người nằm chéo ngoài góc vẫn
	// được tính là gần, và họ sẽ hiện ra ở chỗ không nhìn thấy.
	hw, hh := 100.0, 50.0
	if d := chebyshevOutside(50, 20, hw, hh); d != 0 {
		t.Errorf("trong khung phải ra 0, nhận %v", d)
	}
	if d := chebyshevOutside(150, 0, hw, hh); d != 50 {
		t.Errorf("ra ngoài 50 theo trục X, nhận %v", d)
	}
	if d := chebyshevOutside(0, 90, hw, hh); d != 40 {
		t.Errorf("ra ngoài 40 theo trục Y, nhận %v", d)
	}
	// Chéo góc: lấy trục lệch NHIỀU hơn.
	if d := chebyshevOutside(150, 90, hw, hh); d != 50 {
		t.Errorf("chéo góc phải lấy trục lệch nhiều hơn (50), nhận %v", d)
	}
}
