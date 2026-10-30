package main

import (
	"fmt"
	"math"

	"kiemxoay/server/internal/config"
)

// Sinh bản đồ "Tu Tiên Giới" theo 9 bước của spec §8.1.
//
// Bố cục cố định: Thiên Tiên Đài ở giữa, 4 sân môn phái có Đài Tiên Khí gần 4 góc, đường
// chính nối sân vào tâm, đường vành đai nối 4 vùng. Nội dung sinh cho MỘT góc phần tư rồi
// xoay 90° ba lần (công bằng mọi hướng), mỗi góc khoác một biome khác nhau.

// biome is one quadrant's flavour: which obstacles and decor it uses.
type biome struct {
	name    string
	ground  string   // tile nền dùng cho cả bản đồ khi đây là biome chủ đạo
	rubble  []string // vật cản rải rác
	cluster string   // vật cản xếp thành cụm
	fence   string   // vật cản xếp thành dãy dọc mép đường
	decor   []string
	alley   float64 // bề rộng hẻm đặc trưng
}

var biomes = []biome{
	// Góc 0 — Linh Sơn: núi đá, nhiều đá tảng thành cụm, ít bụi.
	{name: "Linh Sơn", ground: "dirt", rubble: []string{"boulder"}, cluster: "boulder",
		fence: "brick", decor: []string{"rock_small", "pine"}, alley: 350},
	// Góc 1 — Trúc Lâm: rừng tre, nhiều bụi rậm + rào tre, chỗ phục kích.
	{name: "Trúc Lâm", ground: "grass", rubble: []string{"fence"}, cluster: "fence",
		fence: "fence", decor: []string{"bamboo", "bush_small"}, alley: 250},
	// Góc 2 — Phàm Trấn: làng, nhà khung tường gạch có cửa, rào tre quanh vườn.
	{name: "Phàm Trấn", ground: "grass", rubble: []string{"cart"}, cluster: "brick",
		fence: "fence", decor: []string{"lantern", "well"}, alley: 250},
	// Góc 3 — Hắc Thị: chợ, lưới sạp hàng tạo hẻm hẹp.
	{name: "Hắc Thị", ground: "concrete", rubble: []string{"stall", "cart"}, cluster: "stall",
		fence: "stall", decor: []string{"lantern", "banner"}, alley: 150},
}

// gen holds everything one generation run needs.
type gen struct {
	b     *builder
	cfg   config.Mapgen
	size  float64
	roads []seg // đoạn đường: không đặt vật cản lên
}

// seg is a road segment with a width; roads are a chain of these.
type seg struct {
	a, b Vec2
	w    float64
}

// Vec2 is a plain point (cmd/mapgen does not import the game packages).
type Vec2 struct{ X, Y float64 }

func (p Vec2) add(q Vec2) Vec2     { return Vec2{p.X + q.X, p.Y + q.Y} }
func (p Vec2) sub(q Vec2) Vec2     { return Vec2{p.X - q.X, p.Y - q.Y} }
func (p Vec2) mul(k float64) Vec2  { return Vec2{p.X * k, p.Y * k} }
func (p Vec2) len() float64        { return math.Hypot(p.X, p.Y) }
func (p Vec2) dist(q Vec2) float64 { return p.sub(q).len() }

// rot90 rotates a point around the map centre, q times (step 3: đối xứng xoay 4 phía).
func (g *gen) rot90(p Vec2, q int) Vec2 {
	c := g.size / 2
	x, y := p.X-c, p.Y-c
	for i := 0; i < q; i++ {
		x, y = -y, x
	}
	return Vec2{x + c, y + c}
}

// onRoad reports whether a point (plus its radius) sits on any road, where nothing is placed.
func (g *gen) onRoad(p Vec2, r float64) bool {
	for _, s := range g.roads {
		if pointSegDist(p, s.a, s.b) <= s.w/2+r {
			return true
		}
	}
	return false
}

func pointSegDist(p, a, b Vec2) float64 {
	ab := b.sub(a)
	l2 := ab.X*ab.X + ab.Y*ab.Y
	if l2 < 1e-9 {
		return p.dist(a)
	}
	t := ((p.X-a.X)*ab.X + (p.Y-a.Y)*ab.Y) / l2
	t = math.Max(0, math.Min(1, t))
	return p.dist(a.add(ab.mul(t)))
}

// sectCentre is quadrant q's sect yard centre (step 1).
func (g *gen) sectCentre(q int) Vec2 {
	in := g.cfg.SectInset * g.size
	return g.rot90(Vec2{in, in}, q)
}

// ---- bước 1: khung ---------------------------------------------------------

// frame places the Thiên Tiên Đài plaza, the four sect yards with their Đài Tiên Khí, and
// the yard entrances (spec §8.1 bước 1).
func (g *gen) frame() {
	c := g.size / 2
	plaza := g.cfg.CenterPlazaR
	// Quảng trường trung tâm: vòng cột thấp, chừa 4 lối vào theo 4 đường chính.
	g.ringWall(Vec2{c, c}, plaza, 140, []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2}, 600)
	g.b.altar("center", c, c, 220, g.cfg.CenterPlazaR*0.6)
	g.b.zone("road", c-plaza, c-plaza, 2*plaza, 2*plaza, true)

	for q := 0; q < 4; q++ {
		s := g.sectCentre(q)
		r := g.cfg.SectR
		// Sân môn phái: tường bao không phá được, chừa 3 cổng ra vào.
		gaps := []float64{math.Pi/4 + float64(q)*math.Pi/2, math.Pi / 6, -math.Pi / 3}
		g.ringWall(s, r, 140, gaps, 420)
		g.b.altar("small", s.X, s.Y, 150, 300)
		g.b.zone("road", s.X-r*0.7, s.Y-r*0.7, r*1.4, r*1.4, true)
	}
}

// ringWall draws a circle of indestructible wall blocks with openings at the given angles.
func (g *gen) ringWall(c Vec2, r, thick float64, gaps []float64, gapArc float64) {
	n := int(2 * math.Pi * r / (thick * 0.9))
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		skip := false
		for _, ga := range gaps {
			d := math.Abs(math.Atan2(math.Sin(a-ga), math.Cos(a-ga)))
			if d*r < gapArc/2 {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		x, y := c.X+math.Cos(a)*r, c.Y+math.Sin(a)*r
		g.b.wall(x-thick/2, y-thick/2, thick, thick)
	}
}

// ---- bước 2: đường chính + vành đai ---------------------------------------

// roadsStep connects every sect yard to the centre with a gently curved road and lays the
// ring road that ties the four quadrants together (spec §8.1 bước 2).
func (g *gen) roadsStep() {
	c := Vec2{g.size / 2, g.size / 2}
	w := g.cfg.RoadW
	for q := 0; q < 4; q++ {
		s := g.sectCentre(q)
		// Bezier nhẹ: điểm điều khiển lệch khỏi đường thẳng để đường cong tự nhiên.
		mid := s.add(c).mul(0.5)
		off := Vec2{-(c.Y - s.Y), c.X - s.X}.mul(0.18)
		ctrl := mid.add(off)
		g.bezierRoad(s, ctrl, c, w)
	}
	// Đường vành đai nối 4 vùng.
	rr := g.cfg.RingRoadR
	steps := 64
	var prev Vec2
	for i := 0; i <= steps; i++ {
		a := 2 * math.Pi * float64(i) / float64(steps)
		p := Vec2{c.X + math.Cos(a)*rr, c.Y + math.Sin(a)*rr}
		if i > 0 {
			g.addRoad(prev, p, w*0.8)
		}
		prev = p
	}
}

func (g *gen) bezierRoad(p0, p1, p2 Vec2, w float64) {
	steps := 18
	prev := p0
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1 - t
		p := p0.mul(u * u).add(p1.mul(2 * u * t)).add(p2.mul(t * t))
		g.addRoad(prev, p, w)
		prev = p
	}
}

// addRoad ghi một đoạn đường và vẽ nền đá sáng cho nó.
//
// Vùng "road" là hình CHỮ NHẬT, nên một đoạn chéo không thể vẽ bằng một hộp bao: hộp bao
// của đường chéo là cả hình vuông giữa hai đầu. Đã dính thật — một con phố chéo biến gần
// hết Phàm Trấn thành mặt đường. Vì vậy đoạn chéo được cắt thành nhiều khúc ngắn, mỗi khúc
// một hộp nhỏ ôm sát đường.
func (g *gen) addRoad(a, b Vec2, w float64) {
	g.roads = append(g.roads, seg{a: a, b: b, w: w})
	steps := 1
	if dx, dy := math.Abs(b.X-a.X), math.Abs(b.Y-a.Y); math.Min(dx, dy) > w/2 {
		steps = int(math.Max(dx, dy)/(w/2)) + 1
	}
	for i := 0; i < steps; i++ {
		p := Vec2{a.X + (b.X-a.X)*float64(i)/float64(steps), a.Y + (b.Y-a.Y)*float64(i)/float64(steps)}
		q := Vec2{a.X + (b.X-a.X)*float64(i+1)/float64(steps), a.Y + (b.Y-a.Y)*float64(i+1)/float64(steps)}
		lo := Vec2{math.Min(p.X, q.X), math.Min(p.Y, q.Y)}
		hi := Vec2{math.Max(p.X, q.X), math.Max(p.Y, q.Y)}
		g.b.zone("road", lo.X-w/2, lo.Y-w/2, hi.X-lo.X+w, hi.Y-lo.Y+w, false)
	}
}

// ---- bước 3+4: nội dung 1 góc, xoay 4 phía, vật cản theo biome -------------

// quadrantContent fills quadrant 0 and mirrors it to the other three, each with its own
// biome (spec §8.1 bước 3, 4).
func (g *gen) quadrantContent() {
	// Poisson-disk trong góc phần tư 0 (tam giác từ tâm ra góc trên-trái).
	pts := g.poisson(620, 300)
	for q := 0; q < 4; q++ {
		bm := biomes[q%len(biomes)]
		for i, p := range pts {
			rp := g.rot90(p, q)
			// Xê dịch nhỏ để 4 góc không lộ là bản sao.
			j := g.cfg.Jitter
			rp = rp.add(Vec2{g.b.rnd(-j, j), g.b.rnd(-j, j)})
			if g.onRoad(rp, 200) {
				continue
			}
			switch i % 4 {
			case 0:
				g.cluster(bm, rp)
			case 1:
				g.fenceRun(bm, rp)
			case 2:
				g.house(bm, rp)
			default:
				g.b.obstacle(bm.rubble[i%len(bm.rubble)], rp.X, rp.Y, 160, 120, 200)
			}
		}
		g.alleyFor(bm, q)
		g.bushCluster(q)
	}
}

// poisson is a simple Poisson-disk sample inside quadrant 0, min distance apart.
func (g *gen) poisson(minDist, pad float64) []Vec2 {
	lo, _ := g.b.playable()
	c := g.size / 2
	var out []Vec2
	for tries := 0; tries < 900 && len(out) < 40; tries++ {
		p := Vec2{g.b.rnd(lo+pad, c-pad), g.b.rnd(lo+pad, c-pad)}
		ok := true
		for _, o := range out {
			if o.dist(p) < minDist {
				ok = false
				break
			}
		}
		// Không lấn quảng trường trung tâm hay sân môn phái.
		if ok && p.dist(Vec2{c, c}) > g.cfg.CenterPlazaR+400 && p.dist(g.sectCentre(0)) > g.cfg.SectR+400 {
			out = append(out, p)
		}
	}
	return out
}

// cluster drops 3–7 obstacles of one kind in a tight group (spec §8.1 bước 4).
func (g *gen) cluster(bm biome, at Vec2) {
	n := 3 + g.b.rng.Intn(5)
	for i := 0; i < n; i++ {
		a := g.b.rnd(0, 2*math.Pi)
		d := g.b.rnd(60, 260)
		x, y := at.X+math.Cos(a)*d, at.Y+math.Sin(a)*d
		if bm.cluster == "boulder" {
			g.b.boulder(x, y, g.b.rnd(70, 130), 60)
		} else {
			g.b.obstacle(bm.cluster, x, y, 180, 140, 60)
		}
	}
}

// fenceRun lays a continuous run of fence along a line (spec §8.1 bước 4: "dãy liền").
func (g *gen) fenceRun(bm biome, at Vec2) {
	long := g.b.rnd(600, 1400)
	if g.b.rng.Intn(2) == 0 {
		g.b.obstacle(bm.fence, at.X, at.Y, long, 110, 80)
	} else {
		g.b.obstacle(bm.fence, at.X, at.Y, 110, long, 80)
	}
}

// house builds a brick frame with a door gap (spec §8.1 bước 4: "khung nhà có lỗ cửa").
func (g *gen) house(bm biome, at Vec2) {
	w, h := g.b.rnd(500, 900), g.b.rnd(400, 700)
	t := 110.0
	door := 260.0
	g.b.obstacle("brick", at.X, at.Y, w, t, 40)                         // tường trên
	g.b.obstacle("brick", at.X, at.Y+h-t, (w-door)/2, t, 40)            // tường dưới trái
	g.b.obstacle("brick", at.X+(w+door)/2, at.Y+h-t, (w-door)/2, t, 40) // tường dưới phải
	g.b.obstacle("brick", at.X, at.Y, t, h, 40)                         // tường trái
	g.b.obstacle("brick", at.X+w-t, at.Y, t, h, 40)                     // tường phải
}

// alleyFor guarantees each biome has its signature narrow alley (step 8 kiểm tra).
func (g *gen) alleyFor(bm biome, q int) {
	lo, _ := g.b.playable()
	base := Vec2{lo + 900, lo + 2600}
	p := g.rot90(base, q)
	g.b.alley(p.X, p.Y, 1400, bm.alley, q%2 == 0)
}

// bushCluster guarantees each biome has at least one bush patch for ambushes.
func (g *gen) bushCluster(q int) {
	lo, _ := g.b.playable()
	p := g.rot90(Vec2{lo + 2100, lo + 1200}, q)
	for i := 0; i < 4; i++ {
		a := g.b.rnd(0, 2*math.Pi)
		d := g.b.rnd(0, 420)
		g.b.zone("bush", p.X+math.Cos(a)*d-250, p.Y+math.Sin(a)*d-250, 500, 500, true)
	}
}

// ---- bước 5: vùng địa hình -------------------------------------------------

func (g *gen) terrainZones() {
	c := g.size / 2
	for q := 0; q < 4; q++ {
		// Bùn ở chỗ trũng giữa vùng — đa giác méo, KHÔNG phải ellipse (spec §8.6).
		p := g.rot90(Vec2{c - 2600, c - 2600}, q)
		g.b.organicZone("mud", p.X, p.Y, 470, 0.72)
		// Mỗi góc bản đồ một cái ao: vùng làm chậm như bùn nhưng client vẽ mặt nước gợn
		// sóng (spec §8.6 mục 4). 2–4 ao mỗi bản đồ.
		if q < 3 {
			w := g.rot90(Vec2{c - 1500, c - 3400}, q)
			g.b.organicZone("water", w.X, w.Y, 330, 0.66)
		}
		// Đường gió chạy dọc đường chính, thổi về tâm.
		s := g.sectCentre(q)
		dir := math.Atan2(c-s.Y, c-s.X) * 180 / math.Pi
		mid := s.add(Vec2{c, c}).mul(0.5)
		g.b.windLane(mid.X-700, mid.Y-250, 1400, 500, dir)
	}
}

// ---- bước 6: cổng dịch chuyển ---------------------------------------------

// gatesStep links each sect yard to the diagonally opposite one, plus a four-gate network on
// the ring road (spec §8.1 bước 6).
// gatesStep đặt **4 cổng** trong Thiên Đình, mỗi cổng ở một sân môn phái (spec §8.1 bước 6).
//
// Cả 20 cổng của thế giới thuộc MỘT mạng chung (§8.4), nên không còn chia "mạng sect/ring"
// như bản cũ — cách cũ còn đặt tới 8 cổng ở khu này, lệch con số 4/khu của spec.
func (g *gen) gatesStep() {
	for q := 0; q < 4; q++ {
		g.gateNear(g.sectCentre(q), gateNet, "gold")
	}
}

// gateNear puts a gate on a clear spot near p (≥ 300 from obstacles, spec §8.1 bước 6).
func (g *gen) gateNear(p Vec2, net, color string) {
	// Nới dần bán kính tìm. Thà cổng lệch xa tâm sân còn hơn phá luật "không vật cản trong
	// bán kính 300 quanh cổng" — đặt bừa vào tâm sân là đâm thẳng vào tường bao sân.
	for d := 900.0; d < 3600; d += 300 {
		for try := 0; try < 60; try++ {
			a := g.b.rnd(0, 2*math.Pi)
			x, y := p.X+math.Cos(a)*d, p.Y+math.Sin(a)*d
			if g.b.free(rect{x - 300, y - 300, x + 300, y + 300}, 0) && g.b.gate(x, y, net, color) {
				return
			}
		}
	}
}

// ---- bước 7: decor ---------------------------------------------------------

func (g *gen) decorStep() {
	for q := 0; q < 4; q++ {
		bm := biomes[q%len(biomes)]
		for _, sprite := range bm.decor {
			g.b.decor(sprite, 14)
		}
	}
}

// ---- bước 8: kiểm tra hợp lệ ----------------------------------------------

// check runs the §8.1 bước 8 validation. A failure means "sinh lại với seed+1".
func (g *gen) check() error {
	if err := g.b.validate(); err != nil {
		return err
	}
	open, total, reachable := g.floodFill(100)
	if total == 0 {
		return fmt.Errorf("lưới rỗng")
	}
	if ratio := float64(open) / float64(total); ratio < g.cfg.MinOpenRatio {
		return fmt.Errorf("diện tích trống %.0f%%, cần ≥ %.0f%%", ratio*100, g.cfg.MinOpenRatio*100)
	}
	// Spec đòi "mọi ô trống phải liên thông". Vài ô lẻ kẹt trong khe tường là không tránh
	// khỏi và không ảnh hưởng lối đi, nên chỉ báo lỗi khi phần bị nhốt đáng kể (> 1%).
	if trapped := open - reachable; float64(trapped) > float64(open)*0.01 {
		return fmt.Errorf("có %d/%d ô trống bị nhốt (%.1f%%, cho phép ≤ 1%%)",
			trapped, open, float64(trapped)/float64(open)*100)
	}
	for _, o := range g.b.layers["gates"] {
		if err := g.clearAround(o.X+o.Width/2, o.Y+o.Height/2, 300, "cổng"); err != nil {
			return err
		}
	}
	for _, o := range g.b.layers["altars"] {
		if err := g.clearAround(o.X+o.Width/2, o.Y+o.Height/2, 300, "đài"); err != nil {
			return err
		}
	}
	return nil
}

// clearAround checks nothing solid sits within r of a point.
func (g *gen) clearAround(x, y, r float64, what string) error {
	for _, s := range g.b.solid {
		if s.x0 < x+r && x-r < s.x1 && s.y0 < y+r && y-r < s.y1 {
			return fmt.Errorf("vật cản trong bán kính %.0f quanh %s tại (%.0f,%.0f)", r, what, x, y)
		}
	}
	return nil
}

// floodFill walks the open cells of a `cell`-sized grid from the map centre and returns how
// many cells are open in total and how many the walk reached.
func (g *gen) floodFill(cell float64) (open, total, reachable int) {
	lo, hi := g.b.playable()
	n := int((hi - lo) / cell)
	if n <= 0 {
		return 0, 0, 0
	}
	blocked := make([]bool, n*n)
	for i := 0; i < n*n; i++ {
		x := lo + (float64(i%n)+0.5)*cell
		y := lo + (float64(i/n)+0.5)*cell
		for _, s := range g.b.solid {
			if s.x0 <= x && x <= s.x1 && s.y0 <= y && y <= s.y1 {
				blocked[i] = true
				break
			}
		}
		if !blocked[i] {
			open++
		}
	}
	total = n * n
	start := (n/2)*n + n/2
	if blocked[start] {
		return open, total, 0
	}
	seen := make([]bool, n*n)
	seen[start] = true
	stack := []int{start}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		reachable++
		x, y := i%n, i/n
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= n || ny >= n {
				continue
			}
			j := ny*n + nx
			if !seen[j] && !blocked[j] {
				seen[j] = true
				stack = append(stack, j)
			}
		}
	}
	return open, total, reachable
}

// generate runs the whole §8.1 pipeline once.
// generate sinh khu 1 (Thiên Đình) theo §8.1; các khu khác xem generateZone trong zones.go.
func generate(b *builder, cfg config.Mapgen) error { return generateZone(b, cfg, 0) }
