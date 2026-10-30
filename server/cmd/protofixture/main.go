// Command protofixture sinh bộ gói mẫu để client giải mã và đối chiếu — NETCODE_SPEC §4.9.
//
//	cd server && go run ./cmd/protofixture -o /tmp/wire.json
//	cd client && npm run check:wire -- /tmp/wire.json
//
// Cùng một ý với `cmd/collisionfixture` (GAME_SPEC §8.2a): hai phía cài cùng một thuật toán
// thì phải có một bài ép chúng chứng minh là giống nhau, bằng số, chạy được trong CI.
//
// Hai loại gói, và cần CẢ HAI:
//   - **Golden**: giá trị chọn tay, nhắm vào biên (0, giá trị lớn nhất, chuỗi rỗng, chuỗi
//     nhiều byte UTF-8, bitmask rỗng, bitmask đầy). Đây là những chỗ sai hay nấp.
//   - **Fuzz 10 000 gói**: giá trị ngẫu nhiên có seed cố định. Golden chỉ bắt được lỗi ta
//     NGHĨ RA; fuzz bắt được lỗi ta không nghĩ tới.
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"kiemxoay/server/internal/wire"
)

// Case là một gói mẫu: giá trị từng trường + chuỗi byte server mã hoá ra.
type Case struct {
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields"`
	Bytes  string         `json:"bytes"` // base64
}

// Fixture là toàn bộ bộ mẫu.
type Fixture struct {
	SchemaHash    string `json:"schema_hash"`
	SchemaVersion int    `json:"schema_version"`
	Cases         []Case `json:"cases"`
}

func main() {
	out := flag.String("o", "wire_fixture.json", "file kết quả")
	fuzz := flag.Int("fuzz", 10000, "số gói ngẫu nhiên (spec §4.9 yêu cầu 10 000)")
	seed := flag.Int64("seed", 7, "seed cố định — bài kiểm phải lặp lại được")
	flag.Parse()

	f := Fixture{SchemaHash: wire.SchemaHash, SchemaVersion: wire.SchemaVersion}
	f.Cases = append(f.Cases, goldenCases()...)
	f.Cases = append(f.Cases, fuzzCases(rand.New(rand.NewSource(*seed)), *fuzz)...)

	b, err := json.Marshal(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("schema %s v%d · %d gói mẫu → %s (%.0f KB)\n",
		f.SchemaHash, f.SchemaVersion, len(f.Cases), *out, float64(len(b))/1024)
}

// goldenCases: giá trị chọn tay, nhắm thẳng vào biên.
func goldenCases() []Case {
	var cs []Case
	add := func(name string, fields map[string]any, bytes []byte) {
		cs = append(cs, Case{Msg: name, Fields: fields, Bytes: base64.StdEncoding.EncodeToString(bytes)})
	}

	// Spawn: toàn 0, rồi toàn giá trị lớn nhất, rồi tên nhiều byte UTF-8.
	for _, sp := range []wire.Spawn{
		{},
		{ID: 0xFFFFFFFF, DescVer: 0xFFFF, Name: "", Char: 255, Title: 255, X: 65535, Y: 65535,
			Facing: 255, Level: 255, Scale: 255, Swords: 65535, RingPhase: 65535, RingDir: -128,
			Mood: 255, Buffs: 65535, Flags: 65535},
		// Tên tiếng Việt có dấu: mỗi chữ tốn 2–3 byte UTF-8. Đây là chỗ bộ đếm độ dài hay sai
		// vì người ta đếm KÝ TỰ thay vì BYTE — và lỗi đó chỉ lộ ra với tên tiếng Việt.
		{ID: 42, Name: "Đệ Tử Hay Khóc", Char: 3, Level: 9, Swords: 400},
		{ID: 7, Name: "A", RingDir: 1, Scale: 50},
	} {
		add("Spawn", map[string]any{
			"id": sp.ID, "descVer": sp.DescVer, "name": sp.Name, "char": sp.Char, "title": sp.Title,
			"x": sp.X, "y": sp.Y, "facing": sp.Facing, "level": sp.Level, "scale": sp.Scale,
			"swords": sp.Swords, "ringPhase": sp.RingPhase, "ringDir": sp.RingDir,
			"mood": sp.Mood, "buffs": sp.Buffs, "flags": sp.Flags,
		}, sp.Encode(nil))
	}

	// Update: bitmask RỖNG (không trường nào đi theo) và bitmask ĐẦY.
	// Bitmask rỗng là ca quan trọng nhất của cả giao thức — entity đứng yên thì gói chỉ còn
	// 6 byte, và đó là chỗ tiết kiệm băng thông lớn nhất. Đọc sai nó là hỏng mọi thứ.
	empty := wire.Update{ID: 1, Changed: 0}
	add("Update", updateFields(empty), empty.Encode(nil))

	full := wire.Update{ID: 2, Changed: 0x7FF, X: 1, Y: 2, Facing: 3, Level: 4, Scale: 5,
		Swords: 6, RingPhase: 7, RingDir: -8, Mood: 9, Buffs: 10, Flags: 11}
	add("Update", updateFields(full), full.Encode(nil))

	// Chỉ một bit — ca hay gặp nhất khi chạy thật (chỉ vị trí đổi).
	onlyXY := wire.Update{ID: 3, Changed: 0b11, X: 1234, Y: 5678}
	add("Update", updateFields(onlyXY), onlyXY.Encode(nil))

	for _, d := range []wire.Desc{{}, {ID: 9, DescVer: 3, Name: "Lão Quái Ham Ngủ85", Char: 7, Title: 2}} {
		add("Desc", map[string]any{"id": d.ID, "descVer": d.DescVer, "name": d.Name, "char": d.Char, "title": d.Title}, d.Encode(nil))
	}
	for _, d := range []wire.Despawn{{}, {ID: 123, Reason: 3}} {
		add("Despawn", map[string]any{"id": d.ID, "reason": d.Reason}, d.Encode(nil))
	}
	// SelfState: số ÂM là chỗ i32 hay bị đọc thành u32.
	for _, s := range []wire.SelfState{{}, {AckSeq: 99, X: -123456, Y: 123456, Vx: -1, Vy: 1}} {
		add("SelfState", map[string]any{"ackSeq": s.AckSeq, "x": s.X, "y": s.Y, "vx": s.Vx, "vy": s.Vy}, s.Encode(nil))
	}
	for _, i := range []wire.Input{{}, {Seq: 4294967295, Angle: 128, Buttons: 7}} {
		add("Input", map[string]any{"seq": i.Seq, "angle": i.Angle, "buttons": i.Buttons}, i.Encode(nil))
	}
	return cs
}

func updateFields(u wire.Update) map[string]any {
	m := map[string]any{"id": u.ID, "changed": u.Changed}
	// CHỈ đưa vào những trường bit đang bật — đúng như client sẽ giải mã. Đưa cả trường không
	// gửi vào thì bài đối chiếu so với một giá trị không bao giờ đi trên dây.
	type bit struct {
		name string
		i    int
		val  any
	}
	for _, b := range []bit{
		{"x", 0, u.X}, {"y", 1, u.Y}, {"facing", 2, u.Facing}, {"level", 3, u.Level},
		{"scale", 4, u.Scale}, {"swords", 5, u.Swords}, {"ringPhase", 6, u.RingPhase},
		{"ringDir", 7, u.RingDir}, {"mood", 8, u.Mood}, {"buffs", 9, u.Buffs}, {"flags", 10, u.Flags},
	} {
		if u.Changed&(1<<b.i) != 0 {
			m[b.name] = b.val
		}
	}
	return m
}

// fuzzCases sinh gói ngẫu nhiên. Seed cố định nên hỏng là lặp lại được.
func fuzzCases(r *rand.Rand, n int) []Case {
	names := []string{"", "A", "Đệ Tử", "Lão Quái Ham Ngủ85", "ユーザー", "a.b_c", "Bé Tí", "~Kikix"}
	var cs []Case
	for i := 0; i < n; i++ {
		switch i % 6 {
		case 0:
			sp := wire.Spawn{
				ID: r.Uint32(), DescVer: uint16(r.Intn(65536)), Name: names[r.Intn(len(names))],
				Char: uint8(r.Intn(256)), Title: uint8(r.Intn(256)),
				X: uint16(r.Intn(65536)), Y: uint16(r.Intn(65536)), Facing: uint8(r.Intn(256)),
				Level: uint8(r.Intn(256)), Scale: uint8(r.Intn(256)), Swords: uint16(r.Intn(65536)),
				RingPhase: uint16(r.Intn(65536)), RingDir: int8(r.Intn(256) - 128),
				Mood: uint8(r.Intn(256)), Buffs: uint16(r.Intn(65536)), Flags: uint16(r.Intn(65536)),
			}
			cs = append(cs, Case{Msg: "Spawn", Fields: map[string]any{
				"id": sp.ID, "descVer": sp.DescVer, "name": sp.Name, "char": sp.Char, "title": sp.Title,
				"x": sp.X, "y": sp.Y, "facing": sp.Facing, "level": sp.Level, "scale": sp.Scale,
				"swords": sp.Swords, "ringPhase": sp.RingPhase, "ringDir": sp.RingDir,
				"mood": sp.Mood, "buffs": sp.Buffs, "flags": sp.Flags,
			}, Bytes: b64(sp.Encode(nil))})
		case 1:
			u := wire.Update{
				ID: r.Uint32(), Changed: uint16(r.Intn(1 << 11)),
				X: uint16(r.Intn(65536)), Y: uint16(r.Intn(65536)), Facing: uint8(r.Intn(256)),
				Level: uint8(r.Intn(256)), Scale: uint8(r.Intn(256)), Swords: uint16(r.Intn(65536)),
				RingPhase: uint16(r.Intn(65536)), RingDir: int8(r.Intn(256) - 128),
				Mood: uint8(r.Intn(256)), Buffs: uint16(r.Intn(65536)), Flags: uint16(r.Intn(65536)),
			}
			cs = append(cs, Case{Msg: "Update", Fields: updateFields(u), Bytes: b64(u.Encode(nil))})
		case 2:
			d := wire.Desc{ID: r.Uint32(), DescVer: uint16(r.Intn(65536)),
				Name: names[r.Intn(len(names))], Char: uint8(r.Intn(256)), Title: uint8(r.Intn(256))}
			cs = append(cs, Case{Msg: "Desc", Fields: map[string]any{
				"id": d.ID, "descVer": d.DescVer, "name": d.Name, "char": d.Char, "title": d.Title,
			}, Bytes: b64(d.Encode(nil))})
		case 3:
			d := wire.Despawn{ID: r.Uint32(), Reason: uint8(r.Intn(4))}
			cs = append(cs, Case{Msg: "Despawn", Fields: map[string]any{"id": d.ID, "reason": d.Reason}, Bytes: b64(d.Encode(nil))})
		case 4:
			s := wire.SelfState{AckSeq: r.Uint32(),
				X: int32(r.Int63n(1<<32) - (1 << 31)), Y: int32(r.Int63n(1<<32) - (1 << 31)),
				Vx: int32(r.Int63n(1<<32) - (1 << 31)), Vy: int32(r.Int63n(1<<32) - (1 << 31))}
			cs = append(cs, Case{Msg: "SelfState", Fields: map[string]any{
				"ackSeq": s.AckSeq, "x": s.X, "y": s.Y, "vx": s.Vx, "vy": s.Vy}, Bytes: b64(s.Encode(nil))})
		default:
			in := wire.Input{Seq: r.Uint32(), Angle: uint8(r.Intn(256)), Buttons: uint8(r.Intn(8))}
			cs = append(cs, Case{Msg: "Input", Fields: map[string]any{
				"seq": in.Seq, "angle": in.Angle, "buttons": in.Buttons}, Bytes: b64(in.Encode(nil))})
		}
	}
	return cs
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
