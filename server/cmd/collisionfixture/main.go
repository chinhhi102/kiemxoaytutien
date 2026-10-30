// Command collisionfixture sinh dữ liệu đối chiếu va chạm giữa server và client (spec §8.2a).
//
// Server (Go) và client dự đoán (TypeScript) phải cho ra **cùng một vị trí**. Hai bản cài
// đặt khác ngôn ngữ thì không có cách nào bảo đảm bằng mắt; cách duy nhất là chạy cùng một
// bộ đầu vào qua cả hai rồi so.
//
// Lệnh này ghi ra 1 000 bước đi ngẫu nhiên **sát vật cản** kèm vị trí server tính được.
// Script `client/scripts/collision-check.ts` phát lại qua mã client và báo lỗi nếu lệch quá
// 0,5 đơn vị.
//
//	cd server && go run ./cmd/collisionfixture > /tmp/fixture.json
//	cd client && npm run check:collision -- /tmp/fixture.json
package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"math/rand"
	"os"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/world"
)

type oneCase struct {
	// Vị trí trước khi đi, góc đi, cỡ thân, và vị trí server tính được sau MỘT bước.
	//
	// Kiểm cả bước đi chứ không chỉ phép đẩy ra khỏi vật cản: tốc độ còn phụ thuộc vùng
	// địa hình (bùn ×0.6, đường gió ×1.5/×0.8), mà vùng bây giờ là ĐA GIÁC. Bản kiểm cũ chỉ
	// so va chạm nên không thấy client thiếu hẳn phép xét đa giác — đi giữa đồng trống vẫn
	// ra 20 lần hiệu chỉnh mỗi giây.
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	VX     float64 `json:"vx"`
	VY     float64 `json:"vy"`
	Angle  float64 `json:"a"`
	Scale  float64 `json:"sc"`
	DT     float64 `json:"dt"`
	Moving bool    `json:"mv"`
	// Ghost: đang Giao Hỏa Tốc — xuyên mọi vật cản và chạy ×3 (spec v5.5 §6.3). Phải có
	// trong bài đối chiếu vì nó đổi CẢ hai thứ quyết định vị trí: tốc độ và va chạm.
	Ghost bool    `json:"gh"`
	EX    float64 `json:"ex"`
	EY    float64 `json:"ey"`
	EVX   float64 `json:"evx"`
	EVY   float64 `json:"evy"`
}

type fixture struct {
	Map   *world.Map `json:"map"`
	Cases []oneCase  `json:"cases"`
}

func main() {
	mapsDir := flag.String("maps", config.FindRepoFile("maps"), "thư mục bản đồ Tiled")
	mapID := flag.String("map", "", "id bản đồ (rỗng = bản đồ đầu tiên)")
	n := flag.Int("n", 1000, "số bước đi cần sinh")
	seed := flag.Int64("seed", 20261029, "random seed")
	out := flag.String("o", "", "ghi ra file này (rỗng = stdout)")
	flag.Parse()

	cfg := config.MustLoadDefault()
	maps, err := world.LoadDir(*mapsDir, world.LoadOptions{
		HP: cfg.Obstacles.HP, FootprintW: cfg.Obstacles.FootprintW, FootprintH: cfg.Obstacles.FootprintH,
	})
	if err != nil || len(maps) == 0 {
		log.Fatalf("không đọc được bản đồ: %v", err)
	}
	m := maps[0]
	for _, c := range maps {
		if c.ID == *mapID {
			m = c
		}
	}

	g := game.New(cfg, m, *seed)
	rng := rand.New(rand.NewSource(*seed))
	f := fixture{Map: m, Cases: make([]oneCase, 0, *n)}
	dt := 1 / float64(max(cfg.Net.TickHz, 1))

	// Vùng làm ĐỔI TỐC ĐỘ (bùn, nước, gió) phải được thử riêng. Lần đầu viết bài này tôi
	// rải điểm quanh vật cản và rìa bản đồ, kết quả 0/1 000 ca rơi vào bùn hay nước — bài
	// kiểm báo "đạt" trong khi client vẫn thiếu hẳn luật vùng nước. Bài kiểm không chạm
	// tới chỗ cần kiểm thì nó chỉ là một dòng chữ xanh vô nghĩa.
	var speedZones []world.Zone
	for _, z := range m.Zones {
		if z.Kind == "mud" || z.Kind == "water" || z.Kind == "wind" || z.Kind == "snow" || z.Kind == "ice" {
			speedZones = append(speedZones, z)
		}
	}
	log.Printf("bản đồ %s: %d vật cản, %d vùng đổi tốc độ", m.ID, len(m.Obstacles), len(speedZones))

	for len(f.Cases) < *n {
		var from game.Vec
		roll := rng.Float64()
		// Một phần ba số ca đặt trong vùng đổi tốc độ, phần còn lại chia cho sát vật cản và
		// sát rìa bản đồ — ba chỗ hai bên dễ lệch nhau nhất, cũng là chỗ người chơi kêu giật.
		if len(speedZones) > 0 && roll < 0.33 {
			z := speedZones[rng.Intn(len(speedZones))].Shape
			for try := 0; try < 50; try++ {
				q := game.Vec{X: z.X + rng.Float64()*math.Max(z.W, 1), Y: z.Y + rng.Float64()*math.Max(z.H, 1)}
				if z.Circle {
					a := rng.Float64() * 2 * math.Pi
					d := z.R * math.Sqrt(rng.Float64())
					q = game.Vec{X: z.X + math.Cos(a)*d, Y: z.Y + math.Sin(a)*d}
				}
				// Chấp nhận cả điểm ngoài đa giác nhưng trong hộp bao: đó chính là chỗ
				// client và server hay nghĩ khác nhau.
				from = q
				break
			}
		} else if len(m.Obstacles) > 0 && roll < 0.78 {
			o := m.Obstacles[rng.Intn(len(m.Obstacles))].Block()
			cx, cy := o.X+o.W/2, o.Y+o.H/2
			if o.Circle {
				cx, cy = o.X, o.Y
			}
			reach := math.Max(math.Max(o.W, o.H), o.R) + 120
			from = game.Vec{X: cx + (rng.Float64()*2-1)*reach, Y: cy + (rng.Float64()*2-1)*reach}
		} else {
			edge := m.PlayMargin + 200
			from = game.Vec{X: rng.Float64() * m.Size, Y: rng.Float64() * m.Size}
			switch rng.Intn(4) {
			case 0:
				from.Y = rng.Float64() * edge
			case 1:
				from.Y = m.Size - rng.Float64()*edge
			case 2:
				from.X = rng.Float64() * edge
			default:
				from.X = m.Size - rng.Float64()*edge
			}
		}
		a := rng.Float64() * 2 * math.Pi
		scale := 1 + rng.Float64()*2 // thân từ cỡ Luyện Khí tới Đạo Tổ
		moving := rng.Intn(5) > 0    // một phần năm số ca là "thả phím": trên băng vẫn trôi
		// Vận tốc đầu vào khác 0 để kiểm đúng phần QUÁN TÍNH, không chỉ bước đầu tiên.
		vel := game.Vec{X: rng.Float64()*400 - 200, Y: rng.Float64()*400 - 200}
		// Một phần tư số ca đang Giao Hỏa Tốc. Phải có tỉ lệ đáng kể chứ không phải vài ca
		// lẻ: chỗ dễ lệch nhất là **ngay trong lòng vật cản**, nơi server cho đi qua còn
		// client (nếu quên) đẩy bật ra.
		ghost := rng.Intn(4) == 0
		end, endVel := g.MoveStepForTest(from, vel, a, scale, dt, moving, ghost)
		f.Cases = append(f.Cases, oneCase{X: from.X, Y: from.Y, VX: vel.X, VY: vel.Y, Angle: a,
			Scale: scale, DT: dt, Moving: moving, Ghost: ghost,
			EX: end.X, EY: end.Y, EVX: endVel.X, EVY: endVel.Y})
	}

	b, err := json.Marshal(f)
	if err != nil {
		log.Fatal(err)
	}
	if *out == "" {
		os.Stdout.Write(b)
		return
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("%d bước đi → %s (bản đồ %s)", len(f.Cases), *out, m.ID)
}
