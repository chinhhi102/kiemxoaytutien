// Command loadtest đo đúng kịch bản nặng nhất của spec §13.6: một phòng đầy 40 nhân vật,
// trong đó M người đã lên Đạo Tổ với đủ 1 050 kiếm, rồi in ra bảng số để so với ngân sách.
//
//	go run ./cmd/loadtest                       # 40 người, 10 Đạo Tổ, 20 giây
//	go run ./cmd/loadtest -players 40 -tien 10 -secs 30
//
// Khác `cmd/stress` (quét từ 10 → 100 người tìm trần của máy): loadtest cố định đúng một
// kịch bản spec và in bảng đối chiếu ngân sách.
//
// Ngân sách §13.6: tick < 4 ms/phòng · snapshot < 8 KB/client · tổng < 40 KB/s.
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"time"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/netproto"
	"kiemxoay/server/internal/room"
)

// client đóng vai một người chơi: đếm byte nhận và đo ping đúng cách — gửi `ping` rồi tính
// lúc `pong` quay về, không đợi vòng lặp vẽ (spec §13.6).
type client struct {
	bytes   int64
	snaps   int64
	snapMax int
	pings   []float64
	sentAt  map[float64]time.Time
}

func (c *client) Send(b []byte) bool {
	c.bytes += int64(len(b))
	var head struct {
		T string  `msgpack:"t"`
		C float64 `msgpack:"c"`
	}
	if err := netproto.Decode(b, &head); err == nil {
		switch head.T {
		case "snap":
			c.snaps++
			if len(b) > c.snapMax {
				c.snapMax = len(b)
			}
		case "pong":
			if at, ok := c.sentAt[head.C]; ok {
				c.pings = append(c.pings, float64(time.Since(at).Microseconds())/1000)
				delete(c.sentAt, head.C)
			}
		}
	}
	return true
}

func main() {
	players := flag.Int("players", 40, "số nhân vật trong phòng")
	tien := flag.Int("tien", 10, "bao nhiêu người ở Đạo Tổ với đủ kiếm")
	secs := flag.Float64("secs", 20, "số giây game cần đo")
	seed := flag.Int64("seed", 1, "random seed")
	cpuProf := flag.String("cpuprofile", "", "ghi hồ sơ CPU ra file này để chạy `go tool pprof`")
	flag.Parse()

	if *cpuProf != "" {
		f, err := os.Create(*cpuProf)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	cfg := config.MustLoadDefault()
	cfg.Room.MaxHumans = max(cfg.Room.MaxHumans, *players) // dồn hết vào MỘT phòng
	m := room.NewManager(cfg, nil, nil)
	m.Start = func(*room.Room) {}
	rng := rand.New(rand.NewSource(*seed))

	cs := make([]*client, 0, *players)
	for i := 0; i < *players; i++ {
		c := &client{sentAt: map[float64]time.Time{}}
		cs = append(cs, c)
		m.Join(room.NewClient(c, false), netproto.Join{Name: fmt.Sprintf("Nguoi%03d", i), Char: "a_hon"})
	}
	rooms := m.AllRooms()
	r := rooms[0]
	dt := 1 / float64(cfg.Net.TickHz)
	// Client vào phòng qua hàng đợi, chỉ được nạp trong Step → phải chạy vài tick trước khi
	// đụng tới danh sách người chơi, nếu không sẽ ép cảnh giới cho một phòng rỗng.
	for k := 0; k < 5; k++ {
		r.Step(dt)
	}

	// Ép M người lên Đạo Tổ đủ kiếm và đứng chụm lại — đây là cảnh nặng nhất của game.
	//
	// Thế giới 5 khu (spec §8.0): người chơi rải khắp các khu, nên nhóm Đạo Tổ cũng chụm
	// theo TỪNG KHU chứ không dồn hết vào khu 0 — dồn một chỗ thì bốn khu kia rỗng và con
	// số đo được không giống phòng thật.
	prog := r.Zone(0).G.Prog
	top := prog.TienLevel(len(cfg.Cultivation.TienRealms) - 1)
	maxSwords := prog.MaxSwords(prog.Realm(top))
	type forcedOne struct {
		p  *game.Player
		z  *room.Zone
		at game.Vec
	}
	var forced []forcedOne
	center := game.Vec{X: cfg.Room.MapSize / 2, Y: cfg.Room.MapSize / 2}
	total := 0
	for _, z := range r.Zones() {
		total += len(z.Game().Players)
	}
	i := 0
	for _, z := range r.Zones() {
		inZone := 0
		for _, p := range z.Game().Players {
			if i >= *tien {
				break
			}
			at := center.Add(game.Vec{X: float64(inZone%4) * 300, Y: float64(inZone/4) * 300})
			z.Game().ForceState(p, top, maxSwords, at)
			forced = append(forced, forcedOne{p, z, at})
			inZone++
			i++
		}
	}
	fmt.Printf("Phòng: %d nhân vật (%d bot) rải %d khu, %d người Đạo Tổ %d kiếm, %d phòng\n",
		total, total-*players, len(r.Zones()), i, maxSwords, len(rooms))

	// Chờ phòng lấp đủ bot rồi mới bấm giờ: trần phòng là `room.bot_fill_target` (60), mà
	// bot sinh theo nhịp `bots.spawn_every_secs` nên cần tới vài chục giây game. Đo lúc
	// phòng mới có người thật thì con số đẹp mà vô nghĩa.
	for k := 0; k < int(120/dt); k++ {
		r.Step(dt)
		full := true
		for _, z := range r.Zones() {
			full = full && z.Count() >= cfg.Room.ZoneMinChars
		}
		if full && r.Humans()+r.Bots() >= cfg.Room.BotFillTarget {
			break
		}
	}
	total = 0
	for _, z := range r.Zones() {
		total += len(z.Game().Players)
	}
	fmt.Printf("Đã lấp đầy: %d nhân vật (%d người, %d bot)\n", total, r.Humans(), r.Bots())
	for k := 0; k < int(3/dt); k++ { // 3 giây cho vật phẩm mọc đầy rồi mới bấm giờ
		r.Step(dt)
	}
	for _, c := range cs {
		c.bytes, c.snaps, c.snapMax, c.pings = 0, 0, 0, nil
	}

	ticks := int(*secs / dt)
	durs := make([]float64, 0, ticks)
	start := time.Now()
	for k := 0; k < ticks; k++ {
		if k%6 == 0 {
			for _, c := range r.Clients() {
				c.SetInput(netproto.Input{T: "in", Angle: uint8(rng.Intn(256)), Moving: true})
			}
		}
		if k%30 == 0 { // ping 1 lần/giây như client thật
			for idx, c := range cs {
				stamp := float64(k*1000 + idx)
				c.sentAt[stamp] = time.Now()
				r.Pong(room.NewClient(c, false), stamp)
			}
		}
		// Giữ nhóm Đạo Tổ ở nguyên chỗ và nguyên số kiếm: ta đang đo cảnh nặng nhất, không
		// phải đo xem họ mài nhau bao lâu thì hết kiếm.
		for _, f := range forced {
			f.z.Game().ForceState(f.p, top, maxSwords, f.at)
		}
		t0 := time.Now()
		r.Step(dt)
		durs = append(durs, float64(time.Since(t0).Microseconds())/1000)
	}
	wall := time.Since(start).Seconds()
	report(cs, durs, wall, *secs, total)
	// Tick của TỪNG KHU (spec v5 mục 10): ngân sách §13.6 nói về cả phòng, nhưng khi một
	// khu chậm hẳn thì phải thấy ngay là khu nào, không thì chỉ biết "phòng chậm".
	fmt.Println("\nTick từng khu (trung bình trượt lúc kết thúc):")
	for _, z := range r.Zones() {
		fmt.Printf("  %-20s %6.2f ms · %d nhân vật\n", z.Name, z.StepMs(), len(z.Game().Players))
	}
}

func report(cs []*client, durs []float64, wall, secs float64, n int) {
	sort.Float64s(durs)
	sum := 0.0
	for _, d := range durs {
		sum += d
	}
	var bytes int64
	maxSnap := 0
	var pings []float64
	for _, c := range cs {
		bytes += c.bytes
		if c.snapMax > maxSnap {
			maxSnap = c.snapMax
		}
		pings = append(pings, c.pings...)
	}
	sort.Float64s(pings)
	kbs := float64(bytes) / 1024 / secs / float64(max(len(cs), 1))
	avg := sum / float64(max(len(durs), 1))
	p95 := durs[int(math.Min(float64(len(durs)-1), float64(len(durs))*0.95))]
	pingP95 := 0.0
	if len(pings) > 0 {
		pingP95 = pings[int(math.Min(float64(len(pings)-1), float64(len(pings))*0.95))]
	}

	fmt.Printf("\n%-34s %12s %12s %s\n", "Chỉ số", "Đo được", "Ngân sách", "")
	row := func(name, got, want string, ok bool) {
		mark := "ĐẠT"
		if !ok {
			mark = "VƯỢT"
		}
		fmt.Printf("%-34s %12s %12s %s\n", name, got, want, mark)
	}
	row("Tick server (trung bình)", fmt.Sprintf("%.2f ms", avg), "< 4 ms", avg < 4)
	row("Tick server (p95)", fmt.Sprintf("%.2f ms", p95), "< 4 ms", p95 < 4)
	row("Snapshot lớn nhất / client", fmt.Sprintf("%.1f KB", float64(maxSnap)/1024), "< 8 KB", maxSnap < 8*1024)
	row("Băng thông xuống / client", fmt.Sprintf("%.1f KB/s", kbs), "< 40 KB/s", kbs < 40)
	row("Ping p95 (không tính địa lý)", fmt.Sprintf("%.2f ms", pingP95), "< 20 ms", pingP95 < 20)
	fmt.Printf("\n%d nhân vật · CPU dùng %.0f%% thời gian thực · %d tick đo\n",
		n, wall/secs*100, len(durs))
}
