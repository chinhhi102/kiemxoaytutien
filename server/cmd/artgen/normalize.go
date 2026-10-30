package main

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Chuẩn hoá frame tự động — GAME_SPEC §12.5 B mục 1–3.
//
// Vấn đề: frame sinh bằng AI lệch nhau. Cùng một nhân vật, frame này cao hơn frame kia vài
// phần trăm, đứng lệch sang trái vài pixel, chân không cùng một đường. Chạy liên tiếp thì
// thành "giật, nhảy lung tung" — bảng điểm `artgen jitter` cho thấy **tâm và chiều cao** là
// hai thành phần áp đảo, màu sắc chỉ phụ.
//
// Ba phép chuẩn hoá, làm theo đúng thứ tự này vì mỗi phép dựa trên kết quả phép trước:
//
//  1. **Cùng tỉ lệ cho cả animation**: đo chiều cao đầu+thân từng frame, lấy TRUNG VỊ, co
//     giãn mọi frame về chiều cao đó. Trung vị chứ không phải trung bình: một frame hỏng
//     cao gấp rưỡi sẽ kéo lệch trung bình, còn trung vị thì không.
//  2. **Neo chân**: đặt đáy bàn chân về đúng một đường y, căn x theo TRUNG ĐIỂM HAI CHÂN.
//     Không căn theo hộp bao: tay vung ra làm hộp bao lệch hẳn sang một bên, và đó chính là
//     thứ làm nhân vật "trượt ngang" khi chạy.
//  3. **Neo đầu**: đỉnh đầu giữa hai frame liên tiếp không được lệch quá 4 px ngoài nhịp
//     nhún; lệch hơn thì ghi ra để biết frame nào cần sinh lại.
//
// Ảnh gốc KHÔNG bị ghi đè: kết quả ra `frames_norm/`, và `buildAtlas` ưu tiên thư mục đó khi
// có. Chuẩn hoá là phép đoán — phải luôn quay về gốc được để so và để chạy lại khi sửa luật.

const (
	// normAlphaMin: alpha tối thiểu để tính là "có nhân vật". Mép khử răng cưa có alpha rất
	// thấp; đếm nó vào thì mọi phép đo bị kéo theo độ dày viền chứ không theo hình người.
	normAlphaMin = 0x40
	// bodyWidthFrac: một hàng được coi là THÂN khi nó rộng ít nhất ngần này so với hàng rộng
	// nhất. Tay giơ lên và chân duỗi ra là những hàng hẹp, bị loại — đúng ý spec "bỏ tay chân".
	bodyWidthFrac = 0.25
	// feetBandFrac: dải đáy dùng để tìm trung điểm hai chân, tính theo chiều cao nhân vật.
	feetBandFrac = 0.12
	// headDriftMax: đỉnh đầu lệch quá ngần này pixel giữa hai frame là đáng ngờ (spec mục 3).
	headDriftMax = 4.0
	// scaleClamp: co giãn tối đa cho phép (±15%).
	//
	// Ngưỡng này KHÔNG có trong spec và nó cứu cả tính năng. Áp mục B.2 nguyên văn ("co giãn
	// mọi frame về chiều cao trung vị") lên art thật cho ra **co giãn 896%, dịch 4 116 px
	// trên canvas 128 px** — vì `death` cố ý biến hình: nhân vật bẹp xuống rồi hồn nhỏ bay
	// lên, ép về cùng chiều cao là thổi cái hồn to bằng người.
	//
	// Luật thay thế: chuẩn hoá chỉ được sửa NHẸ. Frame cần sửa nhiều hơn 15% không phải là
	// frame "lệch tỉ lệ" mà là frame vẽ khác hẳn — co giãn không chữa được, phải sinh lại.
	// Nên chỗ đó ghi vào danh sách đáng ngờ thay vì bóp méo ảnh.
	scaleClamp = 0.15
	// shiftClamp: dịch tối đa, tính theo phần cạnh canvas. Cùng lý do với scaleClamp.
	shiftClamp = 0.12
)

// frameShape là hình học rút ra từ một frame.
type frameShape struct {
	top, bottom int     // hàng trên cùng / dưới cùng có nhân vật
	bodyTop     int     // đỉnh phần đầu+thân (bỏ tay giơ cao)
	feetX       float64 // trung điểm hai chân
	ok          bool
}

// shapeOf đo hình học một frame.
func shapeOf(img image.Image) frameShape {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rowMin := make([]int, h)
	rowMax := make([]int, h)
	rowN := make([]int, h)
	for i := range rowMin {
		rowMin[i] = w
		rowMax[i] = -1
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if _, _, _, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA(); a>>8 >= normAlphaMin {
				if x < rowMin[y] {
					rowMin[y] = x
				}
				if x > rowMax[y] {
					rowMax[y] = x
				}
				rowN[y]++
			}
		}
	}
	s := frameShape{top: -1, bottom: -1}
	maxW := 0
	for y := 0; y < h; y++ {
		if rowN[y] == 0 {
			continue
		}
		if s.top < 0 {
			s.top = y
		}
		s.bottom = y
		if rw := rowMax[y] - rowMin[y] + 1; rw > maxW {
			maxW = rw
		}
	}
	if s.top < 0 {
		return s // frame rỗng
	}
	// bodyTop: hàng đầu tiên từ trên xuống đủ rộng để là thân, không phải tay giơ lên.
	s.bodyTop = s.top
	for y := s.top; y <= s.bottom; y++ {
		if rowN[y] > 0 && float64(rowMax[y]-rowMin[y]+1) >= float64(maxW)*bodyWidthFrac {
			s.bodyTop = y
			break
		}
	}
	// feetX: trung điểm của dải đáy. Dải đáy chứa hai bàn chân; lấy trung điểm min/max x của
	// dải đó cho ra điểm giữa hai chân, không bị tay vung kéo lệch.
	band := int(math.Max(1, float64(s.bottom-s.top+1)*feetBandFrac))
	lo, hi := w, -1
	for y := s.bottom; y > s.bottom-band && y >= 0; y-- {
		if rowN[y] == 0 {
			continue
		}
		if rowMin[y] < lo {
			lo = rowMin[y]
		}
		if rowMax[y] > hi {
			hi = rowMax[y]
		}
	}
	if hi < 0 {
		return s
	}
	s.feetX = float64(lo+hi) / 2
	s.ok = true
	return s
}

// bodyHeight là chiều cao đầu+thân (bỏ tay chân giơ ra).
func (s frameShape) bodyHeight() float64 { return float64(s.bottom - s.bodyTop + 1) }

// NormStat là kết quả chuẩn hoá một animation, để in báo cáo.
type NormStat struct {
	Char, Anim  string
	Frames      int
	ScaleSpread float64 // chênh lệch tỉ lệ lớn nhất phải co giãn (%), trước khi chuẩn hoá
	ShiftMaxPx  float64 // dịch ngang lớn nhất phải bù (px)
	HeadDrift   float64 // lệch đỉnh đầu lớn nhất còn lại sau chuẩn hoá (px)
	Suspect     string  // frame đáng ngờ (đầu vẫn trôi quá headDriftMax)
	// JitBefore / JitAfter: điểm giật trước và sau chuẩn hoá. Skipped = chuẩn hoá làm TỆ HƠN
	// nên đã bỏ, giữ nguyên ảnh gốc.
	JitBefore, JitAfter float64
	Skipped             bool
}

// normalizeDir chuẩn hoá mọi animation trong một thư mục `frames/`, ghi ra `frames_norm/`.
func normalizeDir(framesDir string) ([]NormStat, error) {
	outDir := filepath.Join(filepath.Dir(framesDir), "frames_norm")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(filepath.Join(framesDir, "*.png"))
	sort.Strings(files)
	byAnim := map[string][]string{}
	for _, f := range files {
		if a, ok := animOf(f); ok {
			byAnim[a] = append(byAnim[a], f)
		}
	}
	char := filepath.Base(filepath.Dir(framesDir))
	var out []NormStat
	for _, anim := range sortedAnimNames(byAnim) {
		st, err := normalizeAnim(char, anim, byAnim[anim], outDir)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

func normalizeAnim(char, anim string, files []string, outDir string) (NormStat, error) {
	stat := NormStat{Char: char, Anim: anim, Frames: len(files)}
	imgs := make([]*image.NRGBA, 0, len(files))
	shapes := make([]frameShape, 0, len(files))
	heights := make([]float64, 0, len(files))
	for _, f := range files {
		im, err := loadImage(f)
		if err != nil {
			return stat, fmt.Errorf("%s: %w", f, err)
		}
		sh := shapeOf(im)
		imgs = append(imgs, im)
		shapes = append(shapes, sh)
		if sh.ok {
			heights = append(heights, sh.bodyHeight())
		}
	}
	if len(heights) == 0 {
		// Không đo được gì (frame rỗng hết): chép nguyên, đừng bịa.
		for i, f := range files {
			if err := savePNG(filepath.Join(outDir, filepath.Base(f)), imgs[i]); err != nil {
				return stat, err
			}
		}
		return stat, nil
	}
	target := medianOf(heights)

	// Đường chân chung: trung vị đáy của cả animation. Dùng trung vị chứ không dùng một
	// frame làm chuẩn — chọn nhầm frame hỏng làm chuẩn thì kéo lệch cả animation.
	bottoms := make([]float64, 0, len(shapes))
	feet := make([]float64, 0, len(shapes))
	for _, sh := range shapes {
		if sh.ok {
			bottoms = append(bottoms, float64(sh.bottom))
			feet = append(feet, sh.feetX)
		}
	}
	targetBottom := medianOf(bottoms)
	targetFeetX := medianOf(feet)

	heads := make([]float64, len(imgs))
	for i, im := range imgs {
		sh := shapes[i]
		name := filepath.Base(files[i])
		if !sh.ok {
			if err := savePNG(filepath.Join(outDir, name), im); err != nil {
				return stat, err
			}
			continue
		}
		k := target / sh.bodyHeight()
		// Anim MỘT LẦN đổi kích thước là chuyện cố ý (death bẹp xuống rồi thành hồn nhỏ,
		// teleport_out tan dần). Không đụng vào tỉ lệ của chúng — chỉ anim LẶP mới phải
		// giữ nguyên cỡ người qua mọi frame.
		if !loopAnim(anim) {
			k = 1
		}
		raw := math.Abs(k-1) * 100
		if k < 1-scaleClamp || k > 1+scaleClamp {
			// Quá ngưỡng: KHÔNG bóp méo, ghi vào danh sách sinh lại.
			k = 1
			if stat.Suspect == "" {
				stat.Suspect = strings.TrimSuffix(name, ".png") + " (lệch tỉ lệ)"
			}
		}
		stat.ScaleSpread = math.Max(stat.ScaleSpread, raw)
		// Dịch sao cho sau khi co giãn, đáy chân và trung điểm hai chân về đúng mốc chung.
		dx := targetFeetX - sh.feetX*k
		dy := targetBottom - float64(sh.bottom)*k
		lim := float64(im.Bounds().Dx()) * shiftClamp
		if math.Abs(dx) > lim || math.Abs(dy) > lim {
			dx, dy = 0, 0
			if stat.Suspect == "" {
				stat.Suspect = strings.TrimSuffix(name, ".png") + " (lệch vị trí)"
			}
		}
		stat.ShiftMaxPx = math.Max(stat.ShiftMaxPx, math.Max(math.Abs(dx), math.Abs(dy)))
		outImg := transformFrame(im, k, dx, dy)
		heads[i] = float64(shapeOf(outImg).bodyTop)
		if err := savePNG(filepath.Join(outDir, name), outImg); err != nil {
			return stat, err
		}
	}
	// TỰ KIỂM: chuẩn hoá chỉ được giữ lại khi nó THẬT SỰ giảm điểm giật.
	//
	// Vì sao bắt buộc phải có bước này: phép "neo theo trung điểm hai chân" và "đo chiều cao
	// đầu+thân" được nghĩ ra cho nhân vật hình người. Đo thật thì `run` của người cải thiện
	// mạnh (su_phu_mi 48.9 → 28.8) nhưng `idle` của **rùa, cua, gà** lại TỆ ĐI gấp đôi —
	// chúng không có hai bàn chân ở đáy để mà neo, và "đầu+thân" cũng không tách được.
	//
	// Thay vì đi liệt kê tay con nào hình người con nào không (sai sót và sẽ lỗi thời khi
	// thêm nhân vật), cứ chấm điểm cả hai bản rồi GIỮ BẢN TỐT HƠN. Luật này tự đúng với mọi
	// nhân vật tương lai, và bảo đảm chuẩn hoá không bao giờ làm hỏng thêm thứ gì.
	stat.JitBefore = animScore(imgs, files, anim)
	normImgs := make([]*image.NRGBA, len(files))
	for i, f := range files {
		im, err := loadImage(filepath.Join(outDir, filepath.Base(f)))
		if err != nil {
			return stat, err
		}
		normImgs[i] = im
	}
	stat.JitAfter = animScore(normImgs, files, anim)
	if stat.JitAfter > stat.JitBefore {
		stat.Skipped = true
		for i, f := range files {
			if err := savePNG(filepath.Join(outDir, filepath.Base(f)), imgs[i]); err != nil {
				return stat, err
			}
		}
		return stat, nil
	}

	// Mục 3: đỉnh đầu còn trôi bao nhiêu SAU khi chuẩn hoá.
	//
	// Chỉ GHI LẠI con số, KHÔNG dùng nó để kết luận "frame hỏng". Thử dùng thì ra 102/149
	// animation bị gắn cờ — một danh sách dài như vậy thì không ai đọc, và phần lớn là nhiễu:
	// ngưỡng 4 px của spec tính trên canvas 512, mà `idle` có động tác ngáp ngửa đầu nên đỉnh
	// đầu dịch thật chứ không phải lỗi.
	//
	// Danh sách sinh lại lấy theo tiêu chí của chính chủ dự án (ANSWERS mục 3): **animation
	// nào SAU chuẩn hoá vẫn đỏ**. Co giãn và dịch đã không chữa được thì đúng là phải vẽ lại.
	for i := 1; i < len(heads); i++ {
		stat.HeadDrift = math.Max(stat.HeadDrift, math.Abs(heads[i]-heads[i-1]))
	}
	if stat.JitAfter >= jitterRed {
		stat.Suspect = "CÒN ĐỎ → sinh lại"
	}
	return stat, nil
}

// animScore chấm điểm giật của một loạt ảnh trong RAM, dùng đúng công thức của `jitter.go`.
func animScore(imgs []*image.NRGBA, files []string, anim string) float64 {
	stats := make([]frameStat, 0, len(imgs))
	heights := make([]float64, 0, len(imgs))
	for i, im := range imgs {
		st := statFrame(im, strings.TrimSuffix(filepath.Base(files[i]), ".png"))
		stats = append(stats, st)
		if st.height > 0 {
			heights = append(heights, st.height)
		}
	}
	medianH := medianOf(heights)
	pairs := make([]PairScore, 0, len(stats))
	sum := 0.0
	for i := 1; i < len(stats); i++ {
		p := scorePair(stats[i-1], stats[i], medianH)
		pairs = append(pairs, p)
		sum += p.Score
	}
	if len(pairs) == 0 {
		return 0
	}
	if loopAnim(anim) {
		return sum / float64(len(pairs))
	}
	_, out := worstOutlier(pairs)
	return out
}

// transformFrame co giãn quanh gốc ảnh rồi dịch, giữ nguyên khung canvas.
//
// Lấy mẫu NGƯỢC (duyệt điểm đích, tìm điểm nguồn) và nội suy song tuyến tính: duyệt xuôi thì
// khi phóng to sẽ để lại lỗ thủng giữa các điểm.
func transformFrame(src *image.NRGBA, k, dx, dy float64) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	if k <= 0 {
		draw.Draw(dst, b, src, b.Min, draw.Src)
		return dst
	}
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := (float64(x) - dx) / k
			sy := (float64(y) - dy) / k
			if sx < 0 || sy < 0 || sx > float64(w-1) || sy > float64(h-1) {
				continue
			}
			setBilinear(dst, src, x, y, sx, sy)
		}
	}
	return dst
}

func setBilinear(dst, src *image.NRGBA, x, y int, sx, sy float64) {
	x0, y0 := int(sx), int(sy)
	fx, fy := sx-float64(x0), sy-float64(y0)
	x1, y1 := min(x0+1, src.Bounds().Dx()-1), min(y0+1, src.Bounds().Dy()-1)
	var acc [4]float64
	for i, p := range [4]struct {
		x, y int
		w    float64
	}{
		{x0, y0, (1 - fx) * (1 - fy)},
		{x1, y0, fx * (1 - fy)},
		{x0, y1, (1 - fx) * fy},
		{x1, y1, fx * fy},
	} {
		_ = i
		o := src.PixOffset(src.Bounds().Min.X+p.x, src.Bounds().Min.Y+p.y)
		// Nhân trước với alpha rồi chia lại ở cuối: trộn thẳng RGB của một điểm trong suốt
		// (thường là đen) vào điểm đục sẽ viền đen quanh nhân vật.
		a := float64(src.Pix[o+3])
		acc[0] += float64(src.Pix[o]) * a * p.w
		acc[1] += float64(src.Pix[o+1]) * a * p.w
		acc[2] += float64(src.Pix[o+2]) * a * p.w
		acc[3] += a * p.w
	}
	d := dst.PixOffset(dst.Bounds().Min.X+x, dst.Bounds().Min.Y+y)
	if acc[3] <= 0 {
		return
	}
	dst.Pix[d] = clampByte(acc[0] / acc[3])
	dst.Pix[d+1] = clampByte(acc[1] / acc[3])
	dst.Pix[d+2] = clampByte(acc[2] / acc[3])
	dst.Pix[d+3] = clampByte(acc[3])
}

func clampByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// buildNormalized chạy chuẩn hoá cho mọi nhân vật và quái.
func buildNormalized(assets string) ([]NormStat, error) {
	dirs, _ := filepath.Glob(filepath.Join(assets, "characters", "*", "frames"))
	more, _ := filepath.Glob(filepath.Join(assets, "monsters", "*", "frames"))
	var all []NormStat
	for _, d := range append(dirs, more...) {
		st, err := normalizeDir(d)
		if err != nil {
			return nil, err
		}
		all = append(all, st...)
	}
	return all, nil
}

func printNorm(stats []NormStat) {
	sort.Slice(stats, func(i, j int) bool { return stats[i].ScaleSpread > stats[j].ScaleSpread })
	fmt.Printf("\nCHUẨN HOÁ FRAME (GAME_SPEC §12.5 B.1–3) — %d animation\n\n", len(stats))
	sort.Slice(stats, func(i, j int) bool {
		return (stats[i].JitBefore - stats[i].JitAfter) > (stats[j].JitBefore - stats[j].JitAfter)
	})
	fmt.Printf("%-15s %-14s %5s %8s %8s %8s  %s\n", "Nhân vật", "Anim", "Khung", "Giật cũ", "Giật mới", "Δ", "Ghi chú")
	shown := 0
	for _, s := range stats {
		if math.Abs(s.JitBefore-s.JitAfter) < 1 && s.Suspect == "" {
			continue // gần như không đổi, không cần in
		}
		shown++
		if shown > 25 {
			break
		}
		note := s.Suspect
		if s.Skipped {
			note = "BỎ QUA (chuẩn hoá làm tệ hơn) " + note
		}
		fmt.Printf("%-15s %-14s %5d %8.1f %8.1f %8.1f  %s\n",
			s.Char, s.Anim, s.Frames, s.JitBefore, s.JitAfter, s.JitAfter-s.JitBefore, note)
	}
	sus, skip := 0, 0
	var be, af float64
	for _, s := range stats {
		if s.Suspect != "" {
			sus++
		}
		if s.Skipped {
			skip++
		}
		be += s.JitBefore
		af += s.JitAfter
	}
	n := float64(len(stats))
	log.Printf("chuẩn hoá xong: %d animation · điểm giật TB %.1f → %.1f · %d animation bỏ qua vì chuẩn hoá làm tệ hơn · %d animation còn frame đáng ngờ (cần sinh lại)",
		len(stats), be/n, af/n, skip, sus)
}
