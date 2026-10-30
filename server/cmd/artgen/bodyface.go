package main

import "strings"

// Art cho hướng A — "một thân, nhiều khuôn mặt" (ART_PROMPTS §4.3b, GAME_SPEC §12.5 A).
//
// Hiện mỗi mood (`run` / `chase` / `flee`) là một bộ frame AI riêng, nên đổi mood là đổi cả
// dáng người — nhìn như ba nhân vật khác nhau. Cách mới: **một** chu kỳ chạy + **một** chu kỳ
// đứng cho phần thân, còn biểu cảm là lớp mặt riêng chồng lên đầu. Đổi mood = chỉ đổi mặt.
//
// Chủ dự án chốt 30/10: sinh art trước cho **2–3 nhân vật để thử**, CHƯA code phần A.
// Sinh hết 10 nhân vật rồi mới phát hiện cách sinh không ăn thua thì tốn cả trăm lượt quota;
// ba nhân vật là đủ để biết lớp mặt có khớp vào thân hay không.

// bodyFaceChars: ba nhân vật thử.
//
// Chọn có lý do chứ không lấy bừa: `a_hon` là nhân vật DỰ PHÒNG (thiếu atlas ai cũng vẽ bằng
// nó, nên nó phải tốt nhất), còn `su_phu_mi` và `dai_ca_dep` là hai chu kỳ `run` chấm giật
// cao nhất bảng — nếu cách mới chữa được hai cái đó thì mới đáng áp cho cả mười.
var bodyFaceChars = []string{"a_hon", "su_phu_mi", "dai_ca_dep"}

// faceExpressions theo đúng thứ tự ô trong lưới 4×2 của ART_PROMPTS §4.3b.
var faceExpressions = []string{
	"neutral smug", "screaming wide open mouth", "evil grin narrowed eyes", "panicked sweating bulging eyes",
	"crying streams of tears", "dizzy spiral eyes", "shocked jaw drop", "sleepy half-closed eyes",
}

// faceNeutral là câu thêm vào prompt THÂN để chừa chỗ cho lớp mặt.
//
// Không có câu này thì bộ sinh vẽ luôn biểu cảm vào thân, và lớp mặt chồng lên sẽ thành hai
// cái mặt đè nhau.
const faceNeutral = "face area left plain and neutral (closed mouth, simple eyes), same head position in every frame"

// bodyFaceJobs dựng job cho hướng A: thân không mặt + bộ 8 khuôn mặt.
func bodyFaceJobs() []Job {
	var jobs []Job
	for _, ch := range characters {
		if !contains(bodyFaceChars, ch.id) {
			continue
		}
		subject := ch.desc + ", color palette " + ch.palette
		out := "characters/" + ch.id + "/body"

		// Thân: run 6 + idle 4, một animation một hàng (ART_PROMPTS §5.1).
		segs := []Seg{
			{Anim: "run", Frames: 6, Out: out, W: 512, H: 512, Pivot: "feet",
				Poses: "cartoon run cycle with spinning legs and arms swinging, " + faceNeutral +
					"; 6 phases: contact, down, pass, up for each leg"},
			{Anim: "idle", Frames: 4, Out: out, W: 512, H: 512, Pivot: "feet",
				Poses: "1) standing breathing out 2) breathing in with shoulders raised 3) weight shifted left 4) weight shifted right, " + faceNeutral},
		}
		rows := []int{6, 4}
		jobs = append(jobs, Job{
			Name: "bodyA_" + ch.id, Priority: 4, Mode: modeSheet, Rows: rows, Segs: segs,
			Prompt: sheetPrompt(style, subject+" with a plain neutral face", rows, segs, true),
		})

		// Bộ mặt: lưới 4×2, mỗi ô một biểu cảm. Prompt lấy nguyên văn §4.3b.
		faceSegs := []Seg{{
			Anim: "face", Frames: len(faceExpressions), Out: "characters/" + ch.id + "/faces",
			Names: faceExpressions2Names(), W: 256, H: 256, Pivot: "center", Each: true,
			Poses: "eight facial expressions in this order: " + strings.Join(faceExpressions, ", "),
		}}
		jobs = append(jobs, Job{
			Name: "facesA_" + ch.id, Priority: 4, Mode: modeSheet, Rows: []int{4, 4}, Segs: faceSegs,
			// Prompt ép CHẶT phần "chỉ có đầu".
			//
			// Bản theo đúng chữ của §4.3b ("face only") KHÔNG ăn thua: bộ sinh vẽ ra nguyên
			// người với biểu cảm khác nhau — đo bằng mắt trên lượt sinh đầu ngày 30/10, cả 8
			// ô đều là toàn thân. Mà toàn thân thì không chồng lớp lên thân được, tức là hỏng
			// đúng mục đích của hướng A.
			//
			// Nên nói thẳng bằng ngôn ngữ ảnh: cận cảnh, cắt ở cổ, KHÔNG vai, KHÔNG thân,
			// KHÔNG tay — và nhắc lại trong phần `Avoid`, vì phủ định đặt một lần thường bị
			// bỏ qua.
			Prompt: style + " sprite sheet 4x2 grid of HEAD CLOSE-UPS of " + subject +
				". EXTREME CLOSE-UP OF THE HEAD ONLY, cropped at the neck: no shoulders, no torso, no arms, no legs, no body." +
				" The head fills each cell. Identical head shape, identical head size and identical eye position in every cell," +
				" 3/4 view facing right, transparent background." +
				" Exactly 4 columns and 2 rows, equal cell size, generous empty space between cells." +
				" Expressions in reading order: " + strings.Join(faceExpressions, ", ") +
				negative + " Avoid also: full body, whole character, shoulders, torso, arms, legs, standing figure.",
		})
	}
	return jobs
}

// faceExpressions2Names đổi tên biểu cảm thành tên file (`neutral smug` → `smug`).
//
// Lấy TỪ CUỐI làm tên: "neutral smug" → `smug`, "screaming wide open mouth" → `mouth` thì
// sai nghĩa, nên dùng bảng đặt tay cho rõ ràng.
func faceExpressions2Names() []string {
	return []string{"smug", "scream", "evil", "panic", "cry", "dizzy", "shock", "sleepy"}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
