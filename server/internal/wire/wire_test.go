package wire

import (
	"math/rand"
	"testing"
)

// Nửa Go của bài đối chiếu §4.9. Nửa client ở `client/scripts/wire-check.ts`.
//
// Test ở đây KHÔNG thay được bài đối chiếu hai phía: nó chỉ chứng minh Go tự nhất quán với
// chính nó. Một lỗi đối xứng (ví dụ ghi và đọc cùng sai thứ tự byte) vẫn qua được. Nhưng nó
// chạy trong `go test` nên bắt lỗi sớm hơn hẳn, ngay lúc sửa schema.

func TestMaHoaRoiGiaiMaRaDungGiaTri(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 2000; i++ {
		want := Spawn{
			ID: r.Uint32(), DescVer: uint16(r.Intn(65536)), Name: randName(r),
			Char: uint8(r.Intn(256)), Title: uint8(r.Intn(256)),
			X: uint16(r.Intn(65536)), Y: uint16(r.Intn(65536)), Facing: uint8(r.Intn(256)),
			Level: uint8(r.Intn(256)), Scale: uint8(r.Intn(256)), Swords: uint16(r.Intn(65536)),
			RingPhase: uint16(r.Intn(65536)), RingDir: int8(r.Intn(256) - 128),
			Mood: uint8(r.Intn(256)), Buffs: uint16(r.Intn(65536)), Flags: uint16(r.Intn(65536)),
		}
		b := want.Encode(nil)
		got, n, err := DecodeSpawn(b)
		if err != nil {
			t.Fatalf("giải mã lỗi: %v", err)
		}
		if n != len(b) {
			t.Fatalf("đọc %d byte nhưng gói dài %d — hai phía đếm trường khác nhau", n, len(b))
		}
		if got != want {
			t.Fatalf("lượt %d: đi %+v về %+v", i, want, got)
		}
	}
}

// Update là message quan trọng nhất và cũng dễ sai nhất: trường chỉ xuất hiện khi bit bật.
func TestUpdateChiChoTruongCoBitBat(t *testing.T) {
	// Bitmask rỗng: gói chỉ còn id + mask = 6 byte. Đây là ca chạy nhiều nhất khi chơi thật
	// (entity đứng yên) và là chỗ tiết kiệm băng thông lớn nhất của cả giao thức.
	empty := Update{ID: 123, Changed: 0, X: 9999, Y: 8888}
	b := empty.Encode(nil)
	if len(b) != 6 {
		t.Fatalf("bitmask rỗng phải ra 6 byte, ra %d — trường không gửi vẫn bị ghi", len(b))
	}
	got, _, err := DecodeUpdate(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.X != 0 || got.Y != 0 {
		t.Errorf("bit tắt mà vẫn đọc ra giá trị: X=%d Y=%d", got.X, got.Y)
	}

	// Từng bit một: bật đúng một bit thì gói phải dài thêm đúng kích thước trường đó.
	sizes := []struct {
		bit  int
		size int
		name string
	}{
		{0, 2, "x"}, {1, 2, "y"}, {2, 1, "facing"}, {3, 1, "level"}, {4, 1, "scale"},
		{5, 2, "swords"}, {6, 2, "ring_phase"}, {7, 1, "ring_dir"}, {8, 1, "mood"},
		{9, 2, "buffs"}, {10, 2, "flags"},
	}
	for _, s := range sizes {
		u := Update{ID: 1, Changed: uint16(1 << s.bit)}
		if n := len(u.Encode(nil)); n != 6+s.size {
			t.Errorf("bật bit %s: gói %d byte, mong %d", s.name, n, 6+s.size)
		}
	}

	// Bật hết: đi và về phải giống hệt.
	full := Update{ID: 2, Changed: 0x7FF, X: 1, Y: 2, Facing: 3, Level: 4, Scale: 5,
		Swords: 6, RingPhase: 7, RingDir: -8, Mood: 9, Buffs: 10, Flags: 11}
	got2, n, err := DecodeUpdate(full.Encode(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got2 != full {
		t.Errorf("bật hết bit: đi %+v về %+v (đọc %d byte)", full, got2, n)
	}
}

// Gói cụt phải trả LỖI, không được đọc rác. Mạng cắt gói giữa chừng là chuyện thường.
func TestGoiCutTraLoiChuKhongDocRac(t *testing.T) {
	full := Spawn{ID: 1, Name: "Bé Tí", Swords: 400}.Encode(nil)
	for n := 0; n < len(full); n++ {
		if _, _, err := DecodeSpawn(full[:n]); err == nil {
			t.Fatalf("cắt còn %d/%d byte mà vẫn giải mã được — đang đọc rác", n, len(full))
		}
	}
	if _, _, err := DecodeSpawn(full); err != nil {
		t.Fatalf("gói đủ phải đọc được: %v", err)
	}
}

// Chuỗi dài quá 255 byte bị CẮT, không làm rơi gói. Và cắt phải theo BYTE, không theo ký tự.
func TestChuoiDaiBiCatTheoByte(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "Đ" // 2 byte UTF-8 mỗi chữ → 400 byte
	}
	b := Spawn{Name: long}.Encode(nil)
	got, _, err := DecodeSpawn(b)
	if err != nil {
		t.Fatalf("tên dài phải cắt chứ không làm rơi gói: %v", err)
	}
	if n := len([]byte(got.Name)); n > 255 {
		t.Errorf("tên sau khi cắt còn %d byte, phải ≤ 255", n)
	}
}

func TestSchemaHashCoVaOnDinh(t *testing.T) {
	// Hash rỗng nghĩa là generator chạy hỏng — và lúc đó bài kiểm hai phía mất tác dụng
	// vì nó so hai chuỗi rỗng với nhau rồi báo "khớp".
	if SchemaHash == "" {
		t.Fatal("SchemaHash rỗng")
	}
	if SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, schema khai 2", SchemaVersion)
	}
}

func randName(r *rand.Rand) string {
	names := []string{"", "A", "Đệ Tử", "Lão Quái Ham Ngủ85", "ユーザー", "Bé Tí"}
	return names[r.Intn(len(names))]
}
