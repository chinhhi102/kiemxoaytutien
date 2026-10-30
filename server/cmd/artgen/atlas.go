package main

import (
	"encoding/json"
	"image"
	"image/draw"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// atlasCell: cạnh một ô khung trong atlas, tính bằng pixel.
//
// Để 256 làm NGỘP BỘ NHỚ TEXTURE của máy người chơi thật. Đo ngày 30/10 trên server :9999:
// client nạp 419 ảnh = **607 MB** texture GPU, riêng 11 atlas nhân vật đã 220 MB (mỗi atlas
// 2560×2304 ×4 byte = 22.5 MB). Laptop dùng GPU tích hợp chia sẻ VRAM với hệ thống thường
// chỉ được vài trăm MB cho một tab, nên driver vứt texture và cả mặt đất lẫn nhân vật hiện
// ra TRẮNG TRƠN chỉ còn nét viền — đúng ảnh chụp của đội test lúc 12:18.
//
// 128 là đủ, đo bằng số chứ không ước lượng: nhân vật vẽ ra màn hình cao **87 px** ở Luyện
// Khí và **30 px** ở Đạo Tổ (camera lùi ra theo cảnh giới). Ô 256 tức là thừa 3–8 lần. Ô 128
// vẫn còn dư so với 87 px, mà tốn đúng 1/4: 220 MB → 55 MB.
//
// Đổi số này KHÔNG phải sửa client: client đọc `cell` trong atlas.json và co theo chiều cao
// texture thật (`k = h / sp.texture.height` trong render/player.ts), nên cỡ nhân vật trên
// màn hình không đổi.
const atlasCell = 128

var frameRe = regexp.MustCompile(`^([a-z_]+)_(\d{2})\.png$`)

// animFPS follows ART_PROMPTS §13: idle 6, locomotion 12, one-shots 12–15.
var animFPS = map[string]int{"idle": 6, "run": 12, "chase": 12, "flee": 12, "hurt": 10}

var loopAnims = map[string]bool{"idle": true, "run": true, "chase": true, "flee": true}

type atlasRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type atlasAnim struct {
	Frames []string `json:"frames"`
	FPS    int      `json:"fps"`
	Loop   bool     `json:"loop"`
}

type atlasFile struct {
	Cell   int                  `json:"cell"`
	PivotX float64              `json:"pivot_x"`
	PivotY float64              `json:"pivot_y"`
	Frames map[string]atlasRect `json:"frames"`
	Anims  map[string]atlasAnim `json:"anims"`
}

// buildAtlases packs characters/*/frames and monsters/*/frames into atlas.png + atlas.json.
func buildAtlases(assets string) error {
	dirs, _ := filepath.Glob(filepath.Join(assets, "characters", "*", "frames"))
	more, _ := filepath.Glob(filepath.Join(assets, "monsters", "*", "frames"))
	for _, d := range append(dirs, more...) {
		// Ưu tiên frame ĐÃ CHUẨN HOÁ nếu có (GAME_SPEC §12.5 B: chuẩn hoá chạy TRƯỚC khi
		// ghép atlas). Không có thì dùng frame gốc, nên repo chưa chạy `normalize` vẫn ghép
		// được atlas như cũ.
		if n := filepath.Join(filepath.Dir(d), "frames_norm"); dirHasPNG(n) {
			d = n
		}
		if err := buildAtlas(d); err != nil {
			return err
		}
	}
	return nil
}

// dirHasPNG: thư mục có tồn tại và có ít nhất một ảnh.
func dirHasPNG(dir string) bool {
	m, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	return len(m) > 0
}

func buildAtlas(framesDir string) error {
	files, _ := filepath.Glob(filepath.Join(framesDir, "*.png"))
	sort.Strings(files)
	if len(files) == 0 {
		return nil
	}
	cols := int(math.Ceil(math.Sqrt(float64(len(files)))))
	rows := (len(files) + cols - 1) / cols
	sheet := image.NewNRGBA(image.Rect(0, 0, cols*atlasCell, rows*atlasCell))
	af := atlasFile{Cell: atlasCell, PivotX: 0.5, PivotY: feetY, Frames: map[string]atlasRect{}, Anims: map[string]atlasAnim{}}
	for i, f := range files {
		img, err := loadImage(f)
		if err != nil {
			return err
		}
		x, y := (i%cols)*atlasCell, (i/cols)*atlasCell
		draw.Draw(sheet, image.Rect(x, y, x+atlasCell, y+atlasCell), resize(img, img.Bounds(), atlasCell, atlasCell), image.Point{}, draw.Src)
		name := strings.TrimSuffix(filepath.Base(f), ".png")
		af.Frames[name] = atlasRect{x, y, atlasCell, atlasCell}
		if m := frameRe.FindStringSubmatch(filepath.Base(f)); m != nil {
			a := af.Anims[m[1]]
			a.Frames = append(a.Frames, name)
			a.FPS, a.Loop = fpsFor(m[1]), loopAnims[m[1]]
			af.Anims[m[1]] = a
		}
	}
	parent := filepath.Dir(framesDir)
	if err := savePNG(filepath.Join(parent, "atlas.png"), sheet); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(af, "", " ")
	return os.WriteFile(filepath.Join(parent, "atlas.json"), b, 0o644)
}

func fpsFor(anim string) int {
	if f, ok := animFPS[anim]; ok {
		return f
	}
	return 12
}

// writeManifest lists every usable asset so the client can fall back to placeholders.
func writeManifest(assets string) error {
	var files []string
	err := filepath.WalkDir(assets, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(assets, p)
		rel = filepath.ToSlash(rel)
		// Bỏ qua CẢ `frames` lẫn `frames_norm`: đó là ảnh nguồn, game chỉ dùng `atlas.png`.
		// Quên `frames_norm` ở đây là thêm ~900 ảnh 512 px vào manifest, client preload hết,
		// và bộ nhớ texture vừa cắt từ 607 MB xuống 378 MB sẽ vọt lại ngay.
		if d.IsDir() && (rel == "_raw" || d.Name() == "frames" || d.Name() == "frames_norm") {
			return filepath.SkipDir
		}
		if !d.IsDir() && (strings.HasSuffix(rel, ".png") || strings.HasSuffix(rel, ".json") ||
			strings.HasSuffix(rel, ".mp3") || strings.HasSuffix(rel, ".ogg") || strings.HasSuffix(rel, ".wav")) && rel != "manifest.json" {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)
	b, _ := json.MarshalIndent(map[string]any{"files": files}, "", " ")
	return os.WriteFile(filepath.Join(assets, "manifest.json"), b, 0o644)
}
