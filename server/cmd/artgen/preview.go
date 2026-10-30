package main

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GIF xem trước + ảnh onion-skin — GAME_SPEC §12.5 B mục 6.
//
// Bảng điểm nói animation nào tệ, nhưng không nói TỆ NHƯ THẾ NÀO. Hai thứ này để duyệt bằng
// mắt: GIF cho thấy nó chạy ra sao, onion-skin chồng mờ mọi frame lên nhau nên lệch tâm và
// lệch chiều cao hiện ra thành bóng ma lệch khỏi thân — nhìn một cái là thấy.

// previewBG: nền của ảnh xem trước.
//
// Phải là nền ĐẶC, không trong suốt: GIF chỉ có 1 bit alpha nên vùng trong suốt sẽ nhảy lem
// nhem giữa các frame và làm người xem tưởng chính art bị lỗi. Xám trung tính để không bênh
// nhân vật sáng hay nhân vật tối.
var previewBG = color.RGBA{R: 0x44, G: 0x48, B: 0x4c, A: 0xff}

// writeGIF ghi một animation thành GIF lặp.
func writeGIF(path string, frames []*image.NRGBA, fps int) error {
	if len(frames) == 0 {
		return nil
	}
	if fps <= 0 {
		fps = 12
	}
	// GIF tính thời gian theo đơn vị 1/100 giây, tối thiểu 2 (nhiều trình xem ép 10 fps khi
	// delay < 2). Phát CHẬM 2× so với trong game để mắt bắt được chỗ giật.
	delay := max(2, 200/fps)
	g := &gif.GIF{LoopCount: 0}
	for _, f := range frames {
		b := f.Bounds()
		pm := image.NewPaletted(b, palette.Plan9)
		draw.Draw(pm, b, &image.Uniform{C: previewBG}, image.Point{}, draw.Src)
		draw.Draw(pm, b, f, b.Min, draw.Over)
		g.Image = append(g.Image, pm)
		g.Delay = append(g.Delay, delay)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return gif.EncodeAll(out, g)
}

// writeOnion ghi ảnh onion-skin: mọi frame chồng mờ lên nhau.
//
// Đây là ảnh nói nhiều nhất về độ giật. Animation chuẩn cho ra một thân người rõ nét với vài
// bóng mờ ở tay chân; animation lệch cho ra một đống bóng ma lệch nhau cả thân — không cần
// biết điểm số cũng thấy ngay.
func writeOnion(path string, frames []*image.NRGBA) error {
	if len(frames) == 0 {
		return nil
	}
	b := frames[0].Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, &image.Uniform{C: previewBG}, image.Point{}, draw.Src)
	// Alpha mỗi lớp = 1/số frame, nhưng không dưới 0.12: 10 lớp × 10% thì mờ tới mức không
	// thấy gì, mà không thấy gì thì ảnh này vô dụng.
	a := 1.0 / float64(len(frames))
	if a < 0.12 {
		a = 0.12
	}
	for _, f := range frames {
		mask := image.NewUniform(color.Alpha{A: uint8(a * 255)})
		draw.DrawMask(out, b, f, b.Min, mask, b.Min, draw.Over)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return savePNG(path, out)
}

// buildPreview xuất GIF + onion-skin cho những animation được chỉ định.
//
// `want` dạng "co_ba_ganh/idle,dep_lao/idle"; rỗng = mọi animation CÒN ĐỎ sau chuẩn hoá
// (đó là những cái cần duyệt bằng mắt, xuất hết 149 cái thì không ai xem).
func buildPreview(assets, want string, outDir string) error {
	targets := map[string]bool{}
	for _, t := range strings.Split(want, ",") {
		if t = strings.TrimSpace(t); t != "" {
			targets[t] = true
		}
	}
	if len(targets) == 0 {
		rep, err := buildJitterIn(assets, "frames_norm", false)
		if err != nil {
			return err
		}
		for _, a := range rep.Anims {
			score := a.Avg
			if !a.Loop {
				score = a.Outlier
			}
			if score >= jitterRed {
				targets[a.Char+"/"+a.Anim] = true
			}
		}
	}
	if len(targets) == 0 {
		fmt.Println("Không animation nào còn đỏ — không có gì cần xuất.")
		return nil
	}
	n := 0
	for _, sub := range []string{"frames", "frames_norm"} {
		dirs, _ := filepath.Glob(filepath.Join(assets, "characters", "*", sub))
		more, _ := filepath.Glob(filepath.Join(assets, "monsters", "*", sub))
		for _, dir := range append(dirs, more...) {
			char := filepath.Base(filepath.Dir(dir))
			files, _ := filepath.Glob(filepath.Join(dir, "*.png"))
			sort.Strings(files)
			byAnim := map[string][]string{}
			for _, f := range files {
				if a, ok := animOf(f); ok {
					byAnim[a] = append(byAnim[a], f)
				}
			}
			for anim, fs := range byAnim {
				if !targets[char+"/"+anim] {
					continue
				}
				imgs := make([]*image.NRGBA, 0, len(fs))
				for _, f := range fs {
					im, err := loadImage(f)
					if err != nil {
						return err
					}
					imgs = append(imgs, im)
				}
				// "truoc" = frame gốc, "sau" = frame đã chuẩn hoá. Tên file nói rõ để khi gửi
				// cho chủ dự án không phải giải thích cái nào là cái nào.
				tag := "truoc"
				if sub == "frames_norm" {
					tag = "sau"
				}
				base := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s", char, anim, tag))
				if err := writeGIF(base+".gif", imgs, fpsFor(anim)); err != nil {
					return err
				}
				if err := writeOnion(base+"_onion.png", imgs); err != nil {
					return err
				}
				n += 2
			}
		}
	}
	fmt.Printf("Đã xuất %d tệp vào %s (mỗi animation: GIF + onion-skin, bản trước và sau chuẩn hoá).\n", n, outDir)
	return nil
}
