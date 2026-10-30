package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Điểm "độ giật" của hoạt ảnh (GAME_SPEC §12.5 B mục 5).
//
// Vấn đề nó đo: frame sinh bằng AI lệch nhau — to nhỏ khác nhau, nhân vật nằm lệch chỗ,
// màu sắc đổi — nên khi chạy liên tiếp thì nhân vật "giật, nhảy lung tung". Mắt người nhận
// ra ngay nhưng không nói được frame nào hỏng, và với 10 nhân vật × ~15 animation thì soi
// bằng mắt là không khả thi.
//
// Spec chốt ba thành phần, so TỪNG CẶP frame liên tiếp: lệch tâm, lệch chiều cao, khác màu.
// Điểm 0–100, cao là tệ.
//
// ⚠ Ngưỡng quy đổi dưới đây do tôi chọn, spec không quy định. Vì vậy hãy đọc BẢNG XẾP HẠNG
// (animation nào tệ hơn animation nào) chứ đừng đọc con số tuyệt đối như một chân lý —
// chủ dự án xem bảng rồi chỉnh ngưỡng nếu thấy lệch so với mắt mình.

const (
	// jitterCenterFull: tâm nhân vật xê dịch bằng ngần này phần cạnh ô thì tính 100 điểm.
	// 6% của ô 128 px ≈ 7.7 px — xê dịch cỡ đó giữa hai frame liền nhau là thấy rõ bằng mắt.
	jitterCenterFull = 0.06
	// jitterHeightFull: chiều cao đổi ngần này phần so với trung vị thì tính 100 điểm.
	jitterHeightFull = 0.08
	// jitterColorFull: màu trung bình lệch ngần này (trên thang 255) thì tính 100 điểm.
	jitterColorFull = 18.0
	// jitterRed: từ điểm này trở lên là BÁO ĐỎ, đề xuất sinh lại riêng frame đó.
	jitterRed = 45.0
	// poseOff: hộp bao lệch quá ngần này so với trung vị của animation thì frame đó **vẽ sai
	// dáng** — báo riêng, KỂ CẢ khi điểm giật tổng chưa vượt ngưỡng (chủ dự án chốt 30/10).
	//
	// Vì sao phải có, và vì sao phải xét cả TỈ LỆ chứ không chỉ cao/rộng: `dep_lao idle_03`
	// vẽ con dép NẰM BẸP trong khi ba frame kia dép đứng. Đo ra chiều cao lệch 19%, chiều
	// rộng lệch 20% — cả hai đều LỌT dưới ngưỡng 30%. Chỉ tỉ lệ rộng/cao là lộ: 1.13 so với
	// trung vị 0.76, lệch 49%. Thiếu phép xét tỉ lệ thì luật này bỏ sót đúng cái frame sinh
	// ra nó.
	poseOff = 0.30
	// outlierFlat / outlierFull: một bước nhảy gấp dưới `outlierFlat` lần bước nhảy thông
	// thường của chính animation là bình thường (0 điểm); gấp `outlierFull` lần là chắc chắn
	// hỏng (100 điểm). Xem worstOutlier.
	outlierFlat = 1.6
	outlierFull = 4.0
)

// frameStat là những con số rút ra từ một frame để so với frame kế bên.
type frameStat struct {
	name    string
	cx, cy  float64 // trọng tâm phần không trong suốt
	height  float64 // chiều cao hộp bao phần không trong suốt
	boxW    float64 // chiều rộng hộp bao — dùng cho phép xét "sai dáng"
	r, g, b float64 // màu trung bình phần không trong suốt
	opaque  int
	width   float64
	cell    float64
}

// statFrame đo một ảnh frame.
//
// Mọi phép đo chỉ tính pixel KHÔNG TRONG SUỐT: nền trong suốt chiếm phần lớn canvas, gộp nó
// vào thì mọi frame đều ra trọng tâm giữa ô và màu xám như nhau — tức là đo xong không phân
// biệt được gì.
func statFrame(img image.Image, name string) frameStat {
	b := img.Bounds()
	s := frameStat{name: name, cell: float64(b.Dx())}
	minY, maxY := math.MaxInt32, -1
	minX, maxX := math.MaxInt32, -1
	var sx, sy, sr, sg, sb float64
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cr, cg, cb, ca := img.At(x, y).RGBA()
			// Bỏ cả pixel gần trong suốt: mép khử răng cưa có alpha rất thấp, đếm vào thì
			// trọng tâm bị kéo theo độ dày viền chứ không theo nhân vật.
			if ca < 0x4000 {
				continue
			}
			n++
			sx += float64(x)
			sy += float64(y)
			sr += float64(cr >> 8)
			sg += float64(cg >> 8)
			sb += float64(cb >> 8)
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
		}
	}
	if n == 0 {
		return s
	}
	s.opaque = n
	s.cx, s.cy = sx/float64(n), sy/float64(n)
	s.r, s.g, s.b = sr/float64(n), sg/float64(n), sb/float64(n)
	s.height = float64(maxY - minY + 1)
	s.boxW = float64(maxX - minX + 1)
	s.width = float64(b.Dx())
	return s
}

// PairScore là điểm giữa hai frame liên tiếp.
type PairScore struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Center float64 `json:"center"`
	Height float64 `json:"height"`
	Color  float64 `json:"color"`
	Score  float64 `json:"score"`
	Red    bool    `json:"red"`
}

// AnimScore là điểm của một animation của một nhân vật.
type AnimScore struct {
	Char   string  `json:"char"`
	Anim   string  `json:"anim"`
	Loop   bool    `json:"loop"`
	Frames int     `json:"frames"`
	Avg    float64 `json:"avg"`
	Worst  float64 `json:"worst"`
	// Outlier / OutlierFrame: frame lạc lõng nhất và điểm của nó (xem outlierRatio).
	// Đây là con số dùng để quyết định SINH LẠI frame nào; `Avg`/`Worst` chỉ nói animation
	// nào đáng xem trước.
	Outlier      float64 `json:"outlier"`
	OutlierFrame string  `json:"outlier_frame"`
	// BadPose: frame vẽ SAI DÁNG (hộp bao lệch hẳn tỉ lệ so với phần còn lại của animation).
	// Tách khỏi điểm giật vì nó là một loại hỏng khác: không phải "lệch nhẹ giữa hai frame"
	// mà là "frame này vẽ ra một thứ khác".
	BadPose []string `json:"bad_pose,omitempty"`
	// Excepted: chủ dự án đã duyệt là giật CÓ CHỦ ĐÍCH (assets/jitter_exceptions.json).
	// Vẫn có điểm đầy đủ, chỉ không tính vào số "đỏ" — để nếu art đổi khiến điểm vọt lên thì
	// vẫn nhìn thấy, thay vì bị giấu đi vĩnh viễn.
	Excepted bool        `json:"excepted,omitempty"`
	Reason   string      `json:"exception_reason,omitempty"`
	Pairs    []PairScore `json:"pairs"`
}

// JitterReport là toàn bộ kết quả, ghi ra assets/_raw/jitter.json.
type JitterReport struct {
	Threshold float64     `json:"threshold"`
	Anims     []AnimScore `json:"anims"`
}

// Exception là một animation được chủ dự án duyệt là "giật có chủ đích".
type Exception struct {
	Char       string `json:"char"`
	Anim       string `json:"anim"`
	Reason     string `json:"reason"`
	ApprovedBy string `json:"approved_by"`
	Date       string `json:"date"`
}

// loadExceptions đọc assets/jitter_exceptions.json.
//
// Không có file, file hỏng, hay danh sách rỗng đều KHÔNG phải lỗi — chỉ là chưa ai duyệt
// ngoại lệ nào. Nhưng nếu file có mà đọc hỏng thì phải NÓI RA: im lặng coi như rỗng nghĩa là
// một anim đã được duyệt bỗng lại báo đỏ, và không ai hiểu vì sao.
func loadExceptions(assets string) map[string]Exception {
	out := map[string]Exception{}
	path := filepath.Join(assets, "jitter_exceptions.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var doc struct {
		Exceptions []Exception `json:"exceptions"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		log.Printf("CẢNH BÁO: %s hỏng (%v) — coi như không có ngoại lệ nào", path, err)
		return out
	}
	for _, e := range doc.Exceptions {
		out[e.Char+"/"+e.Anim] = e
	}
	return out
}

func clamp100(v float64) float64 {
	return math.Min(100, math.Max(0, v))
}

// scorePair quy ba độ lệch về thang 0–100 rồi lấy TRUNG BÌNH.
//
// Trung bình chứ không lấy lớn nhất: một animation chạy thì trọng tâm xê dịch là chuyện bình
// thường (chân guồng, tay vung), nên lấy lớn nhất sẽ báo đỏ mọi chu kỳ chạy và bảng mất giá
// trị. Ba thành phần vẫn in riêng để biết nó giật vì lý do gì.
func scorePair(a, b frameStat, medianH float64) PairScore {
	p := PairScore{From: a.name, To: b.name}
	if a.opaque == 0 || b.opaque == 0 {
		// Frame rỗng là hỏng hẳn, không phải "hơi giật".
		p.Score, p.Red = 100, true
		return p
	}
	cell := a.cell
	if cell <= 0 {
		cell = 1
	}
	d := math.Hypot(a.cx-b.cx, a.cy-b.cy)
	p.Center = clamp100(d / (jitterCenterFull * cell) * 100)
	if medianH > 0 {
		p.Height = clamp100(math.Abs(a.height-b.height) / medianH / jitterHeightFull * 100)
	}
	dc := (math.Abs(a.r-b.r) + math.Abs(a.g-b.g) + math.Abs(a.b-b.b)) / 3
	p.Color = clamp100(dc / jitterColorFull * 100)
	p.Score = (p.Center + p.Height + p.Color) / 3
	p.Red = p.Score >= jitterRed
	return p
}

// loopAnim: animation chạy vòng lặp. Tách riêng vì hai nhóm phải đọc bằng hai cách khác nhau.
func loopAnim(name string) bool { return loopAnims[name] }

// worstOutlier tìm frame LẠC LÕNG NHẤT: bước nhảy của nó lớn gấp mấy lần bước nhảy
// THÔNG THƯỜNG của chính animation đó, quy về thang 0–100.
//
// Vì sao phải so tương đối thay vì so với một ngưỡng cố định: `death` cố ý biến hình mạnh
// (nhân vật bẹp xuống rồi hồn trắng bay lên), nên mọi cặp frame của nó đều vượt mọi ngưỡng
// tuyệt đối — đo thử trên art hiện có ra **94/102 animation một lần báo đỏ**, bảng mất sạch
// giá trị. Lấy trung vị bước nhảy của chính animation làm chuẩn thì thang đo tự co giãn:
// một animation biến hình đều có mọi bước xấp xỉ nhau nên không frame nào nổi lên; chỉ frame
// sinh hỏng mới nhảy gấp mấy lần phần còn lại. Đó đúng là thứ spec cần để "đề xuất sinh lại
// RIÊNG frame đó".
//
// Quy đổi: gấp `outlierFlat` lần trung vị = 0 điểm, gấp `outlierFull` lần = 100 điểm.
func worstOutlier(pairs []PairScore) (string, float64) {
	if len(pairs) < 3 {
		// Dưới 3 cặp thì "bước nhảy thông thường" không có nghĩa — hai cặp thì cái nào cũng
		// là một nửa dữ liệu. Thà không chấm còn hơn chấm bừa.
		return "", 0
	}
	vals := make([]float64, len(pairs))
	for i, p := range pairs {
		vals[i] = p.Score
	}
	med := medianOf(vals)
	if med <= 0 {
		return "", 0
	}
	best, bestScore := "", 0.0
	for _, p := range pairs {
		ratio := p.Score / med
		sc := clamp100((ratio - outlierFlat) / (outlierFull - outlierFlat) * 100)
		if sc > bestScore {
			best, bestScore = p.To, sc
		}
	}
	return best, bestScore
}

// badPoseFrames tìm frame vẽ SAI DÁNG: hộp bao lệch hẳn so với trung vị của animation.
//
// Xét ba thứ — chiều cao, chiều rộng, và TỈ LỆ rộng/cao. Tỉ lệ là thứ quan trọng nhất và dễ
// bị bỏ quên: một frame vẽ con dép nằm bẹp có cao và rộng đều lệch chưa tới ngưỡng, nhưng
// tỉ lệ thì lệch gấp rưỡi.
//
// Bỏ qua animation dưới 3 frame và animation MỘT LẦN biến hình (death, teleport): ở đó hộp
// bao đổi hẳn là chuyện cố ý, báo đỏ chỉ tạo tiếng ồn.
func badPoseFrames(stats []frameStat) []string {
	if len(stats) < 3 {
		return nil
	}
	// Tính trung vị trên frame KHÁC NHAU, không trên chuỗi phát.
	//
	// Chuỗi phát có thể lặp lại một frame (xem `anim_overrides.json`), và frame lặp hai lần
	// thì đếm hai lần vào trung vị — đủ để kéo lệch chuẩn rồi báo oan một frame bình thường.
	// Đã dính: sau khi cho `dep_lao idle` chạy 00→01→02→01, frame `idle_02` bị gắn cờ "sai
	// dáng" chỉ vì `idle_01` được đếm đôi.
	seen := map[string]bool{}
	hs, ws, rs := []float64{}, []float64{}, []float64{}
	for _, st := range stats {
		if st.opaque == 0 || st.height <= 0 || st.boxW <= 0 || seen[st.name] {
			continue
		}
		seen[st.name] = true
		hs = append(hs, st.height)
		ws = append(ws, st.boxW)
		rs = append(rs, st.boxW/st.height)
	}
	if len(hs) < 3 {
		return nil
	}
	mh, mw, mr := medianOf(hs), medianOf(ws), medianOf(rs)
	var bad []string
	flagged := map[string]bool{}
	for _, st := range stats {
		if st.opaque == 0 || st.height <= 0 || st.boxW <= 0 || flagged[st.name] {
			continue
		}
		off := func(v, med float64) float64 {
			if med <= 0 {
				return 0
			}
			return math.Abs(v/med - 1)
		}
		if off(st.height, mh) > poseOff || off(st.boxW, mw) > poseOff || off(st.boxW/st.height, mr) > poseOff {
			flagged[st.name] = true
			bad = append(bad, st.name)
		}
	}
	return bad
}

// animOf tách tên animation khỏi tên file frame ("run_03.png" → "run").
func animOf(file string) (string, bool) {
	m := frameRe.FindStringSubmatch(filepath.Base(file))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// buildJitter chấm điểm mọi animation của mọi nhân vật và quái, ghi assets/_raw/jitter.json.
//
// `sub` chọn thư mục frame: "frames" (gốc) hay "frames_norm" (đã chuẩn hoá). Có hai lựa chọn
// để so ĐƯỢC trước/sau — không so được thì không biết chuẩn hoá có tác dụng hay không.
func buildJitter(assets string) (*JitterReport, error) { return buildJitterIn(assets, "frames", true) }

func buildJitterIn(assets, sub string, write bool) (*JitterReport, error) {
	dirs, _ := filepath.Glob(filepath.Join(assets, "characters", "*", sub))
	more, _ := filepath.Glob(filepath.Join(assets, "monsters", "*", sub))
	rep := &JitterReport{Threshold: jitterRed}
	exc := loadExceptions(assets)
	// Chấm đúng thứ tự game THỰC SỰ PHÁT, không phải thứ tự file trong thư mục.
	//
	// Thiếu dòng này thì bỏ một frame hỏng ra khỏi vòng lặp xong, bảng điểm vẫn giữ nguyên
	// con số cũ mãi mãi — và người đọc bảng sẽ tưởng việc sửa không có tác dụng.
	ovs := loadAnimOverrides(assets)
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
		// `dir` = assets/<nhóm>/<tên>/frames_norm → nhóm là thư mục cha của thư mục cha.
		// Lần đầu tôi viết `filepath.Dir(dir)` một lần và ra khoá "dep_lao/dep_lao".
		group := filepath.Base(filepath.Dir(filepath.Dir(dir))) + "/" + char
		for _, anim := range sortedAnimNames(byAnim) {
			fs := applyOverride(byAnim[anim], ovs[group+"/"+anim], dir)
			if len(fs) < 2 {
				continue // một frame thì không có cặp nào để so
			}
			stats := make([]frameStat, 0, len(fs))
			heights := make([]float64, 0, len(fs))
			for _, f := range fs {
				img, err := loadImage(f)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", f, err)
				}
				st := statFrame(img, strings.TrimSuffix(filepath.Base(f), ".png"))
				stats = append(stats, st)
				if st.height > 0 {
					heights = append(heights, st.height)
				}
			}
			medianH := medianOf(heights)
			sc := AnimScore{Char: char, Anim: anim, Loop: loopAnim(anim), Frames: len(fs)}
			sum := 0.0
			for i := 1; i < len(stats); i++ {
				p := scorePair(stats[i-1], stats[i], medianH)
				sc.Pairs = append(sc.Pairs, p)
				sum += p.Score
				sc.Worst = math.Max(sc.Worst, p.Score)
			}
			// CHỈ xét anim lặp. Anim một lần (death, teleport, levelup, respawn) cố ý đổi
			// hộp bao — chạy phép này lên chúng ra 64 animation bị gắn cờ, gần như toàn bộ
			// là nhiễu, và một danh sách như thế thì không ai đọc.
			if loopAnim(anim) {
				sc.BadPose = badPoseFrames(stats)
			}
			if len(sc.Pairs) > 0 {
				sc.Avg = sum / float64(len(sc.Pairs))
			}
			sc.OutlierFrame, sc.Outlier = worstOutlier(sc.Pairs)
			if e, ok := exc[char+"/"+anim]; ok {
				sc.Excepted, sc.Reason = true, e.Reason
			}
			rep.Anims = append(rep.Anims, sc)
		}
	}
	sort.Slice(rep.Anims, func(i, j int) bool {
		if rep.Anims[i].Avg != rep.Anims[j].Avg {
			return rep.Anims[i].Avg > rep.Anims[j].Avg // tệ nhất lên đầu
		}
		return rep.Anims[i].Char+rep.Anims[i].Anim < rep.Anims[j].Char+rep.Anims[j].Anim
	})
	if !write {
		return rep, nil
	}
	// Bản ĐẦY ĐỦ vào `_raw` (thư mục nháp, server KHÔNG phục vụ ra ngoài): có từng cặp frame,
	// dùng khi ngồi soi một animation cụ thể.
	full, err := json.MarshalIndent(rep, "", " ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(assets, "_raw", "jitter.json"), full, 0o644); err != nil {
		return nil, err
	}
	// Bản RÚT GỌN vào `assets/` để phím debug `J` trong game tải được.
	//
	// Phải là file riêng chứ không phải cùng file: `_raw` bị chặn phục vụ có chủ đích (nó
	// chứa sheet gốc và ảnh nháp), và bản đầy đủ nặng 183 KB — không có lý do bắt trình
	// duyệt tải ngần ấy chỉ để hiện một con số trên đầu nhân vật.
	slim := JitterReport{Threshold: rep.Threshold}
	for _, a := range rep.Anims {
		slim.Anims = append(slim.Anims, AnimScore{
			Char: a.Char, Anim: a.Anim, Loop: a.Loop, Frames: a.Frames,
			Avg: round1(a.Avg), Worst: round1(a.Worst), Outlier: round1(a.Outlier), OutlierFrame: a.OutlierFrame,
		})
	}
	b, err := json.Marshal(slim)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(assets, "jitter.json"), b, 0o644); err != nil {
		return nil, err
	}
	return rep, nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// writeJitterTo ghi báo cáo ra một đường dẫn tuỳ ý (dùng cho bản sau chuẩn hoá).
func writeJitterTo(assets, rel string, rep *JitterReport) error {
	b, err := json.MarshalIndent(rep, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(assets, filepath.FromSlash(rel)), b, 0o644)
}

// applyOverride đổi danh sách file frame theo thứ tự đã ép (nếu có).
//
// Frame nào ép nhắc tới mà không tồn tại thì bỏ qua — cùng cách xử lý với `buildAtlas`, để
// bảng điểm và atlas không bao giờ nói hai chuyện khác nhau.
func applyOverride(files []string, ov AnimOverride, dir string) []string {
	if len(ov.Frames) == 0 {
		return files
	}
	have := map[string]string{}
	for _, f := range files {
		have[strings.TrimSuffix(filepath.Base(f), ".png")] = f
	}
	out := make([]string, 0, len(ov.Frames))
	for _, name := range ov.Frames {
		if p, ok := have[name]; ok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return files
	}
	_ = dir
	return out
}

func sortedAnimNames(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// medianOf là TRUNG VỊ đúng nghĩa: số chẵn phần tử thì lấy trung bình hai phần tử giữa.
//
// Bản cũ lấy phần tử giữa TRÊN, và sai lệch đó không vô hại: `dep_lao idle` có 4 frame với
// tỉ lệ 0.54 / 0.70 / 0.82 / 1.13 — lấy giữa trên ra 0.82 thay vì 0.76, đủ để `idle_01`
// vượt ngưỡng 30% và bị báo "sai dáng" oan, trong khi chủ dự án nhìn bằng mắt thì thấy nó
// bình thường. Một phép kiểm báo oan là một phép kiểm sắp bị phớt lờ.
func medianOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// printJitter in bảng điểm, TÁCH HAI NHÓM.
//
// Tách vì hai nhóm phải đọc bằng hai cách khác nhau, và gộp chung thì bảng vô dụng:
//   - Anim LẶP (idle/run/chase/flee): chạy liên tục nên giật là thấy ngay. Cột "TB" đáng tin,
//     điểm cao = animation thật sự tệ.
//   - Anim MỘT LẦN (death, teleport, levelup...): vốn PHẢI biến hình mạnh, nên cột "TB" luôn
//     cao và không nói lên gì. Nhóm này đọc cột "Lạc lõng" — frame nhảy ra khỏi xu hướng của
//     hai frame kề bên, tức frame đáng sinh lại.
func printJitter(rep *JitterReport, limit int) {
	fmt.Printf("\nĐIỂM ĐỘ GIẬT (GAME_SPEC §12.5 B.5) — 0 tốt, 100 tệ\n")
	fmt.Println("Ngưỡng quy đổi do agent chọn, spec không quy định: đọc THỨ HẠNG, đừng đọc con số tuyệt đối.")

	loops, once := []AnimScore{}, []AnimScore{}
	for _, a := range rep.Anims {
		if a.Loop {
			loops = append(loops, a)
		} else {
			once = append(once, a)
		}
	}
	printGroup("ANIM LẶP (idle / run / chase / flee) — đọc cột TB", loops, limit, false, rep.Threshold)
	sort.Slice(once, func(i, j int) bool { return once[i].Outlier > once[j].Outlier })
	printGroup("ANIM MỘT LẦN (death, teleport, levelup...) — đọc cột LẠC LÕNG", once, limit, true, rep.Threshold)

	redLoop := 0
	for _, a := range loops {
		if a.Avg >= rep.Threshold && !a.Excepted {
			redLoop++
		}
	}
	redOnce := 0
	for _, a := range once {
		if a.Outlier >= rep.Threshold && !a.Excepted {
			redOnce++
		}
	}
	fmt.Printf("\nTổng: %d anim lặp (%d tệ) · %d anim một lần (%d có frame lạc lõng).\n",
		len(loops), redLoop, len(once), redOnce)
	fmt.Println("Chi tiết đầy đủ: assets/_raw/jitter.json")
}

func printGroup(title string, rows []AnimScore, limit int, byOutlier bool, threshold float64) {
	fmt.Printf("\n%s\n", title)
	fmt.Printf("%-15s %-13s %5s %6s %7s %9s  %s\n", "Nhân vật", "Anim", "Khung", "TB", "Tệ nhất", "Lạc lõng", "Chỗ tệ nhất")
	for i, a := range rows {
		if limit > 0 && i >= limit {
			fmt.Printf("  ... còn %d dòng nữa (xem jitter.json)\n", len(rows)-limit)
			break
		}
		key := a.Avg
		if byOutlier {
			key = a.Outlier
		}
		mark := " "
		if key >= threshold {
			mark = "!"
		}
		if a.Excepted {
			// Vẫn in điểm, chỉ đổi dấu: "đã duyệt" khác hẳn "đạt ngưỡng", và lẫn hai thứ
			// đó là mất luôn khả năng thấy art xuống cấp về sau.
			mark = "~"
		}
		worst := PairScore{}
		for _, p := range a.Pairs {
			if p.Score > worst.Score {
				worst = p
			}
		}
		where := fmt.Sprintf("%s→%s tâm %.0f cao %.0f màu %.0f", worst.From, worst.To, worst.Center, worst.Height, worst.Color)
		if byOutlier && a.OutlierFrame != "" {
			where = fmt.Sprintf("frame %s lệch khỏi xu hướng", a.OutlierFrame)
		}
		fmt.Printf("%-15s %-13s %5d %6.1f %7.1f %9.1f %s %s\n", a.Char, a.Anim, a.Frames, a.Avg, a.Worst, a.Outlier, mark, where)
	}
}
