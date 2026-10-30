package main

import (
	"fmt"
	"strings"
)

// Post-processing modes.
const (
	modeSprite = "sprite" // one object → fitted onto a transparent canvas
	modeSheet  = "sheet"  // many frames / objects in rows → separate PNGs
	modeTile   = "tile"   // opaque square texture
	modeWide   = "wide"   // opaque illustration, keep aspect
	modeRaw    = "raw"    // keep as-is (reference sheets)
)

// Seg is one animation (or set of objects) inside a sheet. Frames are read left-to-right,
// top-to-bottom across the sheet's rows; segments consume frames in order.
type Seg struct {
	Anim   string   `json:"anim"`
	Frames int      `json:"frames"`
	Out    string   `json:"out"`             // output directory
	Names  []string `json:"names,omitempty"` // explicit file names instead of {anim}_{NN}
	W      int      `json:"w"`
	H      int      `json:"h"`
	Pivot  string   `json:"pivot"`          // "feet" (y=460 on 512) | "center"
	Each   bool     `json:"each,omitempty"` // scale every frame on its own (different objects)
	// HeightK is the tallest frame's height relative to standing idle. Each animation is a
	// separate image, so without it a crouched run would be scaled up to idle height.
	HeightK float64 `json:"height_k,omitempty"`
	Poses   string  `json:"-"`
}

// Job is one minigen request plus how to turn its image into game assets.
// One image holds several animations to save the minigen quota (~60-100 images / 5h).
type Job struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Prompt   string `json:"prompt"`
	Mode     string `json:"mode"`
	Out      string `json:"out"` // sprite/tile/raw/wide output file
	W        int    `json:"w"`
	H        int    `json:"h"`
	Pivot    string `json:"pivot"`
	Rows     []int  `json:"rows,omitempty"` // sheet: frames per row
	Segs     []Seg  `json:"segs,omitempty"`
}

// Style lock from ART_PROMPTS §2 ("bựa": ugly-funny, not cute). Gemini cannot output alpha, so
// we ask for flat magenta (ART_PROMPTS §5.1) and key it out in post-processing.
const bgMagenta = "pure flat magenta #FF00FF background filling everything around the subject"

const style = "STYLE: 2D game sprite, grotesque comedic caricature, ugly-funny lowbrow slapstick cartoon, " +
	"huge head on a small scrawny or saggy body, shamelessly exaggerated facial features: crooked or missing teeth, " +
	"bulging or droopy mismatched eyes, big pores and oily shiny skin, moles with a single long hair, sweat drops, " +
	"rosy blotchy cheeks, double chin, over-the-top smug or stupid facial expressions exaggerated 200 percent, " +
	"cheap mismatched clothes with stains, thick wobbly black outline, flat saturated colors with simple cel shading, " +
	"Vietnamese street comedy vibe mixed with wuxia cultivation fantasy, " + bgMagenta +
	", no text, no watermark, no ground shadow, full body visible."

// Objects (swords, props, effects) keep the cartoon look without the face caricature rules.
const styleObject = "STYLE: 2D game asset, lowbrow slapstick cartoon, thick wobbly black outline, flat saturated colors with simple cel shading, " +
	"Vietnamese street comedy vibe mixed with wuxia cultivation fantasy, " + bgMagenta + ", no text, no watermark, no ground shadow. " +
	"Do not use magenta or pink-purple in the subject itself."

const styleOpaque = "STYLE: 2D mobile game art, cartoon, flat colors with simple cel shading, vibrant saturated palette, " +
	"clean vector-like look, no text, no watermark."

const negative = " Avoid: cute, kawaii, pretty, handsome, beautiful, elegant, clean, polished, heroic, idealized, realistic, photo, " +
	"3d render, blurry, extra limbs, cropped body, text, letters, numbers, logo, watermark, background scenery, " +
	"multiple different characters, inconsistent outfit, nudity, grid lines, frame borders."

// sheetPrompt describes a multi-row sheet: which frames go on which row and what each segment shows.
func sheetPrompt(st, subject string, rows []int, segs []Seg, sameCharacter bool) string {
	var b strings.Builder
	total := 0
	for _, n := range rows {
		total += n
	}
	fmt.Fprintf(&b, "%s Sprite sheet of %s with %d frames in total. ", st, subject, total)
	if cols, ok := uniformGrid(rows); ok {
		fmt.Fprintf(&b, "Exactly %d %s and %d %s, equal cell size. ", cols, plural(cols, "column"), len(rows), plural(len(rows), "row"))
	} else {
		for i, n := range rows {
			fmt.Fprintf(&b, "Row %d has exactly %d frames. ", i+1, n)
		}
	}
	b.WriteString("Frames are read left-to-right, top-to-bottom: ")
	for _, s := range segs {
		fmt.Fprintf(&b, "[%s, %d frames] %s. ", s.Anim, s.Frames, s.Poses)
	}
	if sameCharacter {
		b.WriteString("Identical character in every frame: same outfit, colors, proportions, every frame facing right in 3/4 view, character the same height in every cell, feet on the same baseline. ")
	}
	b.WriteString("Leave plenty of empty background space between frames and between rows so every frame is clearly separated; no grid lines, no frame numbers.")
	b.WriteString(negative)
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// uniformGrid reports whether every row has the same number of frames.
func uniformGrid(rows []int) (int, bool) {
	for _, n := range rows[1:] {
		if n != rows[0] {
			return 0, false
		}
	}
	return rows[0], len(rows) > 0
}

type anim struct {
	name, poses string
	frames      int
}

var charAnims = map[string]anim{
	"idle":         {"idle", "1) standing breathing out 2) breathing in, shoulders raised 3) yawning with mouth wide open 4) scratching head, bored sleepy face", 4},
	"run":          {"run", "cartoon run cycle with spinning legs, arms flailing wildly, mouth wide open screaming AAAA, hair flying; 8 phases: contact, down, pass, up for each leg", 8},
	"chase":        {"chase", "sneaky tiptoe run leaning forward, evil grin with squinting eyes, rubbing both hands together; 8 run phases", 8},
	"flee":         {"flee", "panicked run with both arms raised to the sky, sweat drops flying, bulging eyes, wobbly mouth; 8 run phases", 8},
	"hit":          {"hit", "1) face squashed flat to the left 2) body knocked backwards 3) spiral eyes with stars around the head 4) shaking head recovering", 4},
	"levelup":      {"levelup", "1) ready stance 2) golden aura bursting 3) superhero pose with aura 4) bigger aura heroic pose 5) thumbs up 6) back to ready stance", 6},
	"breakthrough": {"breakthrough", "1) meditating seated 2) trembling 3) trembling harder 4) hair standing straight up 5) looking up screaming at the sky 6) screaming harder 7) struck by lightning 8) lightning engulfs body 9) singed frizzy smoking hair 10) triumphant laugh", 10},
	"realm_drop":   {"realm_drop", "1) stunned 2) knees buckling 3) sitting flat on the ground 4) crying rivers of tears 5) crying harder 6) wiping nose and standing up", 6},
	"death":        {"death", "1) getting smashed 2) smashed harder 3) flattened like rice paper 4) completely flat 5) white ghost soul rising out 6) ghost waving goodbye floating up 7) ghost floating higher 8) ghost almost gone at the top", 8},
	"respawn":      {"respawn", "1) falling from high above 2) about to hit the ground 3) landing face first 4) somersault 5) standing up dusting off 6) striking a pose", 6},
	"teleport_out": {"teleport_out", "1) startled 2) body twisting like being sucked into a vortex 3) only the sobbing head left 4) only a pair of flip-flops left", 4},
	"teleport_in":  {"teleport_in", "1) spat out in mid-air 2) flailing 3) landing staggering 4) spiral eyes dizzy", 4},
	// Kỹ năng riêng của từng nhân vật (ART_PROMPTS §11.3). Tư thế cụ thể lấy từ
	// `skillPoses` bên dưới và thay vào chỗ này, nên chuỗi ở đây chỉ là khung chung.
	"skill": {"skill", "1) wind-up 2) wind-up further 3) release 4) peak of the move 5) follow-through 6) back to standing", 6},
}

// skillPoses: tư thế anim `skill` của từng nhân vật, đúng bảng ART_PROMPTS §11.3.
var skillPoses = map[string]string{
	"a_hon":       "1) holding a whole steamed bun 2) stuffing the entire bun into the mouth 3) cheeks puffed out hugely 4) swallowing, eyes bulging 5) burping out a cloud of smoke 6) satisfied grin patting belly",
	"ba_tam_loa":  "1) raising a dented megaphone 2) inhaling deeply, chest out 3) screaming into the megaphone, hair blown backwards 4) screaming harder, face red 5) sound waves bursting out 6) lowering the megaphone smugly",
	"ong_ba_vong": "1) unrolling a hammock 2) lying down in the hammock 3) swinging gently, eyes closed 4) swinging more, a snot bubble forming 5) big snot bubble, fast asleep 6) one eye peeking open",
	"dai_ca_dep":  "1) looking around 2) raising two fingers to the mouth 3) whistling loudly 4) whistling harder, pointing forward 5) crossing arms waiting 6) smug nod",
	"map_tra_sua": "1) sipping bubble tea 2) cheeks filling up with pearls 3) cheeks hugely puffed 4) spitting pearls like a machine gun 5) still spitting, head shaking 6) wiping mouth with the back of the hand",
	"co_ba_ganh":  "1) crouching with the carrying pole 2) pushing off 3) dashing sideways, pole horizontal 4) fruit flying off the baskets 5) skidding to a stop 6) straightening up panting",
	"su_phu_mi":   "1) holding a noodle net on two giant chopsticks 2) winding up 3) throwing the net forward 4) the net spreading open in the air 5) pointing at where it landed 6) folding arms proudly",
	"be_ti":       "1) standing relaxed 2) clenching fists 3) flexing both biceps, veins popping 4) flexing harder, face red, body glowing 5) roaring 6) holding the flex pose",
	"shipper":     "1) gripping imaginary handlebars 2) twisting the throttle 3) delivery box puffing smoke 4) leaning forward hard, lightning behind 5) nearly horizontal from speed 6) skidding to a stop",
	"cu_rua":      "1) sitting cross-legged 2) starting to pull limbs in 3) head half retracted 4) fully retracted, only the shell left 5) shell glowing 6) head popping back out",
}

// poseHeight: tallest frame of each animation relative to the standing idle pose.
var poseHeight = map[string]float64{
	"idle": 1, "run": 0.88, "chase": 0.85, "flee": 0.95, "hit": 0.92, "levelup": 1,
	"breakthrough": 1.15, "realm_drop": 1, "death": 0.95, "respawn": 1, "teleport_out": 1, "teleport_in": 1,
	"skill": 1.05,
}

// charSheets: one animation per image (ART_PROMPTS §5.1) laid out on an exact grid.
var charSheets = []struct {
	anim     string
	rows     []int
	priority int
}{
	{"idle", []int{4}, 1},
	{"run", []int{4, 4}, 1},
	{"hit", []int{4}, 2},
	{"death", []int{4, 4}, 2},
	{"chase", []int{4, 4}, 3},
	{"flee", []int{4, 4}, 3},
	{"levelup", []int{3, 3}, 4},
	{"realm_drop", []int{3, 3}, 4},
	{"breakthrough", []int{5, 5}, 4},
	{"respawn", []int{3, 3}, 4},
	{"teleport_out", []int{4}, 4},
	{"teleport_in", []int{4}, 4},
	{"skill", []int{3, 3}, 3}, // kỹ năng riêng (§11.3) — ưu tiên 3 vì gameplay đã chạy
}

func characterJobs() []Job {
	var jobs []Job
	for _, ch := range characters {
		subject := ch.desc + ", color palette " + ch.palette
		jobs = append(jobs, Job{
			Name: "char_" + ch.id + "_ref", Priority: 9, Mode: modeRaw, Out: "characters/" + ch.id + "/ref_sheet.png",
			Prompt: style + " Character reference sheet of " + subject + ". Show 3 views in a row: front, 3/4 facing right, " +
				"side facing right. Below: 6 facial expressions (neutral, screaming, evil grin, panicked, crying, shocked). " +
				"Same outfit and colors in every view, separated panels." + negative,
		})
		for _, cs := range charSheets {
			an := charAnims[cs.anim]
			poses := an.poses
			if cs.anim == "skill" {
				if sp, ok := skillPoses[ch.id]; ok {
					poses = sp // tư thế kỹ năng riêng của nhân vật (§11.3)
				}
			}
			if gag, ok := ch.gags[cs.anim]; ok {
				poses += "; comedy gag of this character in this animation: " + gag
			}
			segs := []Seg{{Anim: an.name, Frames: an.frames, Out: "characters/" + ch.id + "/frames", W: 512, H: 512, Pivot: "feet",
				HeightK: poseHeight[an.name], Poses: poses}}
			jobs = append(jobs, Job{Name: "char_" + ch.id + "_" + cs.anim, Priority: cs.priority, Mode: modeSheet,
				Rows: cs.rows, Segs: segs, Prompt: sheetPrompt(style, subject, cs.rows, segs, true)})
		}
	}
	return jobs
}

// One image per monster: idle 4 / hurt 2 + death 6 (ART_PROMPTS §6).
func monsterJobs() []Job {
	var jobs []Job
	for _, m := range monsters {
		out := "monsters/" + m.id + "/frames"
		segs := []Seg{
			{Anim: "idle", Frames: 4, Out: out, W: 512, H: 512, Pivot: "feet", Poses: "1) leaning left 2) upright 3) leaning right 4) upright squashed"},
			{Anim: "hurt", Frames: 2, Out: out, W: 512, H: 512, Pivot: "feet", Poses: "1) flinching with angry tears 2) squashed in pain"},
			{Anim: "death", Frames: 6, Out: out, W: 512, H: 512, Pivot: "feet", Poses: "1) inflating 2) cracking 3) bursting 4) exploding into glowing spirit orbs 5) orbs scattering 6) only sparkles left"},
		}
		rows := []int{4, 2, 6} // one animation per row (ART_PROMPTS §5.1)
		jobs = append(jobs, Job{Name: "mon_" + m.id, Priority: 1, Mode: modeSheet, Rows: rows, Segs: segs,
			Prompt: sheetPrompt(style, m.prompt+", 3/4 top-down view facing right", rows, segs, true)})
	}
	return jobs
}

var swords = []string{
	"a pair of bamboo chopsticks tied together as a sword",
	"a kitchen cleaver with a wooden handle",
	"a peach wood taoist sword with red tassel",
	"a polished steel jian sword",
	"a flaming red-orange sword with fire trail",
	"an ice crystal blue sword with frost sparkles",
	"a crackling lightning yellow sword",
	"a dark purple cursed sword with smoky aura",
	"a radiant jade and gold celestial sword glowing white",
}

// Swords and item bubbles were generated one per image before the sheet format existed.
func swordJobs() []Job {
	var jobs []Job
	for i, s := range swords {
		jobs = append(jobs, Job{
			Name: fmt.Sprintf("sword_realm_%d", i), Priority: 0, Mode: modeSprite, Out: fmt.Sprintf("swords/realm_%d.png", i),
			W: 256, H: 256, Pivot: "center",
			Prompt: styleObject + " Single game weapon icon, " + s + ", perfectly vertical, tip pointing straight up, handle at the bottom, centered. " +
				// ART_PROMPTS §7: the game tints each ring, so the blade must be near-white.
				"Blade painted in very light grey and white tones so it can be color-tinted in game, colored details only on the handle." + negative,
		})
	}
	return jobs
}

var buffIcons = []struct{ id, icon, color string }{
	{"spin", "a cyan spinning whirlwind around a small sword", "cyan"},
	{"haste", "a winged sandal", "orange"},
	{"magnet", "a red horseshoe magnet", "red"},
	{"hard", "a thick heavy steel sword with a diamond sparkle on the blade", "silver-blue"},
	{"shield", "a golden buddhist-style round shield", "gold"},
	{"exp2", "a glowing magic pill made of light", "green"},
	{"stealth", "a paper talisman with swirling ink", "grey"},
}

func itemJobs() []Job {
	sprite := func(name, out string, size int, prompt string) Job {
		return Job{Name: name, Priority: 0, Mode: modeSprite, Out: out, W: size, H: size, Pivot: "center", Prompt: styleObject + " " + prompt + negative}
	}
	jobs := []Job{
		sprite("item_sword", "items/sword.png", 128, "A glossy transparent soap bubble containing a small glowing sword, shiny highlight."),
		// Bó Kiếm — chủ dự án duyệt 30/10, prompt lấy NGUYÊN VĂN từ ART_PROMPTS §8 bảng
		// "Vật phẩm". Năm thanh kiếm (không phải ba), xoè nhẹ, buộc ruy băng đỏ, hào quang
		// vàng — để phân biệt với bong bóng kiếm đơn.
		sprite("item_bundle", "items/bundle.png", 128,
			"A glossy transparent soap bubble containing a bundle of five small swords tied together with a red ribbon, swords fanned out slightly, golden glow, shiny highlight."),
	}
	for _, c := range []struct{ id, color string }{{"orb_small", "green"}, {"orb_medium", "blue"}, {"orb_large", "gold"}} {
		jobs = append(jobs, sprite("item_"+c.id, "items/"+c.id+".png", 128, "A glowing "+c.color+" spirit orb gem, sparkly, round."))
	}
	for _, b := range buffIcons {
		jobs = append(jobs, sprite("buff_"+b.id, "buffs/"+b.id+".png", 128,
			"A glossy transparent soap bubble containing "+b.icon+", tinted "+b.color+", shiny highlight."))
	}
	chest := "an ornate wooden treasure chest wrapped in red ribbons, glowing golden light leaking out, 3/4 top-down view"
	segs := []Seg{
		{Anim: "idle", Frames: 4, Out: "items/chest", W: 512, H: 512, Pivot: "feet", Poses: "1) still 2) rattling left 3) still 4) rattling right"},
		{Anim: "open", Frames: 6, Out: "items/chest", W: 512, H: 512, Pivot: "feet", Poses: "1) closed shaking 2) lid cracking open 3) lid bursting open with light 4) items flying out 5) light fading 6) open empty chest"},
	}
	jobs = append(jobs, Job{Name: "chest", Priority: 2, Mode: modeSheet, Rows: []int{4, 6}, Segs: segs,
		Prompt: sheetPrompt(styleObject, chest, []int{4, 6}, segs, true)})
	return jobs
}

var decorList = []string{"a red plastic stool", "an old bicycle leaning", "a pile of watermelons", "a street food cart", "a banana tree",
	"a rusty water jar", "a broken TV", "a lazy sleeping dog", "a bush with flowers", "a stone lantern"}

func mapJobs() []Job {
	tile := func(id, biome string) Job {
		return Job{Name: "tile_" + id, Priority: 1, Mode: modeTile, Out: "tiles/" + id + ".png", W: 512, H: 512,
			Prompt: styleOpaque + " Seamless tileable top-down ground texture, " + biome + ", square image filling the whole frame, cartoon game style, " +
				"no objects, no characters, even lighting, edges must tile perfectly. " +
				// ART_PROMPTS §9.0: calm background that never competes with EXP orbs and items.
				"Desaturated, low detail, designed as a background that does not distract from gameplay objects."}
	}
	jobs := []Job{
		tile("grass", "soft muted sage green grass, very low contrast, subtle large brush strokes, no flowers, no small colorful dots, calm and plain"),
		tile("concrete", "pale warm grey concrete, very low contrast, few faint cracks, plain and calm"),
		tile("dirt", "light muted sandy brown dirt, very low contrast, only a few faint large pebbles, plain and calm"),
		tile("mud", "muddy brown puddle texture with bubbles"),
		{Name: "zone_bush", Priority: 1, Mode: modeSprite, Out: "zones/bush.png", W: 512, H: 512, Pivot: "center",
			Prompt: styleObject + " Top-down view of a dense leafy bush cluster, round, lush green, centered." + negative},
	}
	names := make([]string, len(decorList))
	for i, d := range decorList {
		names[i] = decorID(d)
	}
	segs := []Seg{
		{Anim: "decor", Frames: 10, Out: "decor", Names: names, W: 128, H: 128, Pivot: "center", Each: true,
			Poses: "ten different props in this order: " + strings.Join(decorList, ", ")},
		{Anim: "wall", Frames: 1, Out: "decor", Names: []string{"wall_border"}, W: 1024, H: 256, Pivot: "center", Each: true,
			Poses: "one wide seamless horizontal stone and bamboo fence segment, tileable left-right"},
	}
	rows := []int{5, 5, 1}
	wind := []Seg{{Anim: "wind", Frames: 4, Out: "zones", W: 256, H: 256, Pivot: "center",
		Poses: "a white cartoon wind gust arrow with swirl lines and light blue outline pointing right, sliding right a little more each frame"}}
	jobs = append(jobs,
		Job{Name: "decor_sheet", Priority: 3, Mode: modeSheet, Rows: rows, Segs: segs,
			Prompt: sheetPrompt(styleObject, "top-down 3/4 view decoration props", rows, segs, false)},
		Job{Name: "zone_wind", Priority: 3, Mode: modeSheet, Rows: []int{4}, Segs: wind,
			Prompt: sheetPrompt(styleObject, "a wind gust effect", []int{4}, wind, false)})
	return jobs
}

func decorID(d string) string {
	for _, p := range []string{"a ", "an "} {
		d = strings.TrimPrefix(d, p)
	}
	return strings.ReplaceAll(d, " ", "_")
}

// Obstacle canvases are square: the client keeps the art's aspect and repeats it along long
// fences and walls instead of stretching it to the hitbox.
//
// 512 là cỡ ĐÚNG, đo ngày 30/10 chứ không ước lượng: ở mức thu phóng gần nhất (Luyện Khí)
// quầy hàng vẽ ra 220 px, xe đẩy 245 px, tường đá 230 px — tức ảnh 512 chỉ dư khoảng 2 lần,
// vừa đủ để không bị rỗ khi camera kéo gần.
//
// `boulder` từng để 1024 và đó là lỗi: ngoài bản đồ nó rộng trung bình 208 đơn vị, CÒN NHỎ
// HƠN quầy hàng (246) vốn chỉ dùng 512. Một mình boulder (8 ảnh + 6 vỡ + 6 mọc lại) chiếm
// 80 MB texture GPU trong tổng 607 MB mà client phải nạp — xem ghi chú ở atlas.go.
var obstacles = []struct {
	id, desc string
	w, h     int
}{
	{"fence", "a straight horizontal bamboo fence segment tied with rope", 512, 512},
	{"stall", "a street market stall with a striped canopy and fruit baskets", 512, 512},
	{"cart", "a wooden street food cart with a small stove", 512, 512},
	{"brick", "a short horizontal red brick wall segment with moss", 512, 512},
	{"boulder", "a huge round grey boulder with cartoon crack lines", 512, 512},
	{"stone_wall", "an ancient mossy horizontal stone wall with carved patterns", 512, 512},
}

func obstacleJobs() []Job {
	var jobs []Job
	for _, o := range obstacles {
		dir := "obstacles/" + o.id
		subject := "a top-down 3/4 view game obstacle, " + o.desc
		states := []Seg{{Anim: "states", Frames: 4, Out: dir, Names: []string{"intact", "cracked", "nearly_broken", "rubble"},
			W: o.w, H: o.h, Pivot: "center",
			Poses: "1) intact and solid 2) with visible cracks 3) heavily damaged, crumbling, pieces missing 4) flat pile of rubble and debris on the ground"}}
		jobs = append(jobs, Job{Name: "obs_" + o.id + "_states", Priority: 1, Mode: modeSheet, Rows: []int{2, 2}, Segs: states,
			Prompt: sheetPrompt(styleObject, subject, []int{2, 2}, states, true)})
		// Đoạn DỌC: client xếp gạch theo chiều dài nên ảnh ngang kéo dọc trông sai hẳn
		// (hàng rào nằm ngang chồng lên nhau). Mỗi vật cản cần thêm một bộ 4 trạng thái vẽ
		// theo chiều dọc, tên file `<state>_vertical.png`.
		// "vertical" dễ bị hiểu thành "dựng đứng" (một tấm panel đứng), trong khi cái cần là
		// cùng vật cản đó XOAY 90° TRÊN MẶT ĐẤT để chạy ra xa máy quay. Nên mô tả theo hướng
		// chạy và bắt nó chạm mép trên + mép dưới khung hình thì xếp gạch mới nối liền.
		vdesc := strings.ReplaceAll(o.desc, "horizontal ", "")
		vdesc = strings.ReplaceAll(vdesc, " horizontal", "")
		vstates := []Seg{{Anim: "states", Frames: 4, Out: dir,
			Names: []string{"intact_vertical", "cracked_vertical", "nearly_broken_vertical", "rubble_vertical"},
			W:     o.w, H: o.h, Pivot: "center",
			Poses: "1) intact and solid 2) with visible cracks 3) heavily damaged, crumbling, pieces missing 4) flat pile of rubble and debris on the ground"}}
		jobs = append(jobs, Job{Name: "obs_" + o.id + "_states_vertical", Priority: 1, Mode: modeSheet, Rows: []int{2, 2}, Segs: vstates,
			Prompt: sheetPrompt(styleObject, "a top-down 3/4 view game obstacle, "+vdesc+
				", rotated 90 degrees flat on the ground so its length runs AWAY from the viewer, "+
				"going from the bottom edge of the frame straight up to the top edge; the shape is TALL and NARROW, "+
				"it touches both the top and the bottom edge so copies stacked on top of each other join seamlessly; "+
				"this is the same object as the horizontal version, only turned, not stood upright on its end",
				[]int{2, 2}, vstates, true)})
		if o.id == "stone_wall" {
			continue
		}
		for _, an := range []Seg{
			{Anim: "break", Frames: 6, Out: dir + "/break", W: o.w, H: o.h, Pivot: "center", Poses: "1) shaking 2) cracking 3) bursting apart 4) debris flying 5) dust cloud 6) rubble pile"},
			{Anim: "regrow", Frames: 6, Out: dir + "/regrow", W: o.w, H: o.h, Pivot: "center", Poses: "1) glowing rubble 2) pieces rising with light 3) half rebuilt glowing 4) almost whole 5) whole with sparkles 6) intact and solid"},
		} {
			segs := []Seg{an}
			jobs = append(jobs, Job{Name: "obs_" + o.id + "_" + an.Anim, Priority: 4, Mode: modeSheet, Rows: []int{3, 3}, Segs: segs,
				Prompt: sheetPrompt(styleObject, subject, []int{3, 3}, segs, true)})
		}
	}
	return jobs
}

// One image per gate colour: idle 6 / warn 4 / burst 6 (ART_PROMPTS §9.3).
func gateJobs() []Job {
	var jobs []Job
	for _, color := range []string{"purple", "blue", "orange"} {
		out := "gates/" + color
		segs := []Seg{
			{Anim: "idle", Frames: 6, Out: out, W: 512, H: 512, Pivot: "center", Poses: "slowly swirling vortex, rotating a little more each frame"},
			{Anim: "warn", Frames: 4, Out: out, W: 512, H: 512, Pivot: "center", Poses: "flashing brighter and brighter, intense glow"},
			{Anim: "burst", Frames: 6, Out: out, W: 512, H: 512, Pivot: "center", Poses: "bursting with light rays expanding then fading"},
		}
		rows := []int{6, 4, 6}
		jobs = append(jobs, Job{Name: "gate_" + color, Priority: 1, Mode: modeSheet, Rows: rows, Segs: segs,
			Prompt: sheetPrompt(styleObject, "a top-down magical teleport portal on the ground, swirling "+color+
				" vortex inside a carved stone ring with taoist symbols, glowing", rows, segs, true)})
	}
	return jobs
}

func vfxSeg(id string, n int, poses string) Seg {
	return Seg{Anim: id, Frames: n, Out: "vfx/" + id, W: 256, H: 256, Pivot: "center", Poses: poses}
}

// vfxJobs: one image per effect (never two animated effects in one image, ART_PROMPTS §5.1).
func vfxJobs() []Job {
	list := []struct {
		seg  Seg
		rows []int
		pri  int
	}{
		{vfxSeg("clash_spark", 6, "comic sword clash spark burst, white and yellow star shapes, expanding then fading"), []int{3, 3}, 2},
		{vfxSeg("sword_shatter", 6, "a cartoon sword shattering into sharp shards flying outward"), []int{3, 3}, 2},
		{vfxSeg("death_smoke", 6, "comic white-grey smoke puff cloud with dark outline, expanding then dissipating"), []int{3, 3}, 2},
		{vfxSeg("levelup_aura", 6, "rising golden aura ring with sparkles"), []int{3, 3}, 3},
		{vfxSeg("lightning", 6, "thick cartoon lightning bolt striking down from the top"), []int{3, 3}, 3},
		{vfxSeg("shield_break", 6, "golden bubble shield cracking and bursting into glass shards"), []int{3, 3}, 3},
		{vfxSeg("stun_stars", 4, "ring of small yellow cartoon stars orbiting"), []int{4}, 3},
		{vfxSeg("run_dust", 4, "small brown dust puffs behind running feet"), []int{4}, 4},
	}
	var jobs []Job
	for _, v := range list {
		segs := []Seg{v.seg}
		jobs = append(jobs, Job{Name: "vfx_" + v.seg.Anim, Priority: v.pri, Mode: modeSheet, Rows: v.rows, Segs: segs,
			Prompt: sheetPrompt(styleObject, "a game visual effect", v.rows, segs, false)})
	}
	return append(jobs, Job{Name: "vfx_crack", Priority: 3, Mode: modeSprite, Out: "vfx/crack.png", W: 256, H: 256, Pivot: "center",
		Prompt: styleObject + " Thin dark grey crack lines overlay pattern like broken glass, sparse, centered." + negative})
}

var badgeThemes = []string{"bamboo green", "kitchen steel grey", "peach wood red", "polished silver", "flame orange",
	"ice crystal blue", "lightning yellow", "cursed dark purple", "celestial jade and gold"}

func uiJobs() []Job {
	badgeNames := make([]string, len(badgeThemes))
	for i := range badgeThemes {
		badgeNames[i] = fmt.Sprintf("badge_%d", i)
	}
	segs := []Seg{
		{Anim: "badge", Frames: 9, Out: "ui", Names: badgeNames, W: 256, H: 256, Pivot: "center", Each: true,
			Poses: "nine round game badge emblems for cultivation ranks, each with a small sword symbol and no letters, themes in order: " + strings.Join(badgeThemes, ", ")},
		{Anim: "medal", Frames: 3, Out: "ui", Names: []string{"medal_gold", "medal_silver", "medal_bronze"}, W: 128, H: 128, Pivot: "center", Each: true,
			Poses: "a gold medal, a silver medal and a bronze medal, each with a tiny sword emblem and no letters"},
	}
	rows := []int{5, 4, 3}
	return []Job{
		{Name: "ui_badges", Priority: 2, Mode: modeSheet, Rows: rows, Segs: segs,
			Prompt: sheetPrompt(styleObject, "game UI icons", rows, segs, false)},
		{Name: "ui_speech", Priority: 2, Mode: modeSprite, Out: "ui/speech_bubble.png", W: 512, H: 256, Pivot: "center",
			Prompt: styleObject + " An empty white comic speech bubble with thick black outline, wide oval, tail at bottom left, empty inside, centered." + negative},
		{Name: "ui_scroll", Priority: 3, Mode: modeSprite, Out: "ui/scroll.png", W: 1024, H: 768, Pivot: "center",
			Prompt: styleObject + " An empty unrolled ancient paper scroll with wooden handles on left and right, horizontal, nothing written, centered." + negative},
		{Name: "ui_home_bg", Priority: 2, Mode: modeWide, Out: "ui/home_bg.png", W: 1920, H: 1080,
			Prompt: styleOpaque + " Wide 16:9 illustration of a chaotic cartoon Vietnamese village square with many chibi characters " +
				"spinning flying swords around them, sunny, bamboo and street stalls, no text."},
	}
}

// Thần thú Tiên giới (ART_PROMPTS §10.1, GAME_SPEC §3.5). Vẽ bằng tông trắng/xám để code
// nhuộm màu theo từng cảnh giới; mỗi con dựng từ đầu + đốt thân + đuôi nên sinh cả ba phần
// trong CÙNG một ảnh cho đồng nét và đồng độ dày viền (đúng mẹo ghi ở §10.1).
const beastTone = ", painted in white and light grey tones for color tinting, isolated single part"

var beasts = []struct {
	id, desc string
}{
	{"dragon", "a fierce but funny chinese dragon: 1) side view head facing right with long whiskers, small antlers and an open mouth with a goofy grin 2) one round body segment seen from the side with overlapping scales and a small spiky fin on top, symmetrical so it can be chained repeatedly 3) side view tail tip facing left with a flowing fin like a flame"},
	{"crane", "a flying immortal crane: 1) side view head on a long neck stretched forward facing right 2) one body segment seen from the side with layered white feathers, symmetrical so it can be chained repeatedly 3) side view tail tip facing left with long trailing feathers"},
	{"phoenix", "a flying phoenix: 1) side view head facing right with a plumed crest 2) one body segment seen from the side with layered glowing feathers, symmetrical so it can be chained repeatedly 3) side view tail tip facing left with long flowing tail feathers"},
}

// skillArtJobs: icon kỹ năng, vật phẩm Bí Kíp, VFX và art riêng cho 4 khu mới
// (ART_PROMPTS §11.3, §11.4 — CHANGES_v5 mục 15).
func skillArtJobs() []Job {
	var jobs []Job
	// Vật phẩm Bí Kíp + icon HUD.
	jobs = append(jobs,
		Job{Name: "item_skill_scroll", Priority: 2, Mode: modeSprite, Out: "items/skill_scroll.png", W: 256, H: 256, Pivot: "center",
			Prompt: styleObject + " A glossy transparent bubble containing an ancient rolled-up martial arts scroll with a red ribbon, glowing" + negative},
		Job{Name: "icon_skill_scroll", Priority: 3, Mode: modeSprite, Out: "ui/icon_skill_scroll.png", W: 128, H: 128, Pivot: "center",
			Prompt: styleObject + " Round HUD icon of an ancient rolled-up scroll with a red ribbon, bold simple shape, no text" + negative})

	// Icon kỹ năng của từng nhân vật (§11.3).
	for _, k := range []struct{ id, icon string }{
		{"a_hon", "a steamed bun with a bite taken out of it"},
		{"ba_tam_loa", "a megaphone with sound waves"},
		{"ong_ba_vong", "a hammock with zzz above it"},
		{"dai_ca_dep", "a mouth whistling with two fingers"},
		{"map_tra_sua", "bubble tea pearls flying"},
		{"co_ba_ganh", "a fruit basket with speed lines"},
		{"su_phu_mi", "a net woven from noodles"},
		{"be_ti", "a flexed bicep"},
		{"shipper", "a lightning bolt on a delivery box"},
		{"cu_rua", "a turtle shell"},
	} {
		jobs = append(jobs, Job{Name: "icon_skill_" + k.id, Priority: 3, Mode: modeSprite,
			Out: "ui/skills/" + k.id + ".png", W: 128, H: 128, Pivot: "center",
			Prompt: styleObject + " Round skill icon, " + k.icon + ", bold simple shape, no text" + negative})
	}

	// Sprite đàn em của Đại Ca (§11.3) — anim `run` 8 khung.
	minionSegs := []Seg{{Anim: "run", Frames: 8, Out: "characters/dan_em/frames", W: 512, H: 512, Pivot: "feet",
		HeightK: 0.88, Poses: charAnims["run"].poses}}
	jobs = append(jobs, Job{Name: "char_dan_em_run", Priority: 3, Mode: modeSheet, Rows: []int{4, 4}, Segs: minionSegs,
		Prompt: sheetPrompt(style, "a short skinny gangster underling in a cheap purple vest and black sunglasses, smaller than a normal character", []int{4, 4}, minionSegs, true)})

	// Overlay trân châu: 6 biến thể viên bẹp + vệt trà sữa (§11.3).
	for i := 1; i <= 6; i++ {
		jobs = append(jobs, Job{Name: fmt.Sprintf("vfx_boba_%d", i), Priority: 4, Mode: modeSprite,
			Out: fmt.Sprintf("vfx/boba_%d.png", i), W: 256, H: 256, Pivot: "center",
			Prompt: styleObject + " A glossy black tapioca pearl splattered flat on glass with a milk tea smear, front view" + negative})
	}
	jobs = append(jobs, Job{Name: "vfx_boba_drip", Priority: 4, Mode: modeSprite, Out: "vfx/boba_drip.png", W: 128, H: 512, Pivot: "center",
		Prompt: styleObject + " A milk tea drip trail running vertically down glass" + negative})

	// Art riêng cho 4 khu mới (§11.4).
	for _, z := range []struct {
		id, desc string
		w, h     int
	}{
		{"cliff_wall", "A tall rocky cliff wall segment seen from a top-down 3/4 view, tileable left to right", 1024, 512},
		{"rope_bridge", "A wooden rope suspension bridge segment seen from above, tileable", 512, 256},
		{"bamboo_wall", "A very dense bamboo grove wall segment seen from above, tileable left to right", 1024, 512},
		{"bamboo_corner", "A very dense bamboo grove corner piece seen from above, L-shaped", 512, 512},
		{"village_roof", "A small village house roof of grey tiles seen from directly above", 1024, 1024},
		{"lantern_string", "A string of paper lanterns seen from above", 512, 256},
		{"ice_island_edge", "A small ice island edge segment seen from above", 512, 256},
	} {
		jobs = append(jobs, Job{Name: "zone_" + z.id, Priority: 4, Mode: modeSprite, Out: "obstacles/" + z.id + ".png",
			W: z.w, H: z.h, Pivot: "center", Prompt: styleObject + " " + z.desc + negative})
	}
	for _, t := range []struct{ id, desc string }{
		{"mountain_rock", "grey mountain rock with patches of moss"},
		{"bamboo_floor", "damp forest floor covered with bamboo leaves"},
		{"cobblestone", "old cobblestone street"},
		{"ice", "smooth glossy pale blue ice with subtle cracks and reflections"},
		{"snow", "soft deep snow ground"},
	} {
		jobs = append(jobs, Job{Name: "tile_" + t.id, Priority: 4, Mode: modeTile, Out: "tiles/" + t.id + ".png",
			W: 512, H: 512, Pivot: "center",
			Prompt: styleObject + " Seamless tileable top-down ground texture: " + t.desc + ", flat even lighting, no shadows" + negative})
	}
	for _, d := range []struct{ id, desc string }{
		{"ice_skid", "a pair of curved ice skid marks"},
		{"snow_pile", "a small pile of snow"},
		{"frozen_reeds", "a few frozen reeds poking through ice"},
	} {
		jobs = append(jobs, Job{Name: "decal_" + d.id, Priority: 5, Mode: modeSprite, Out: "decals/" + d.id + ".png",
			W: 256, H: 256, Pivot: "center", Prompt: styleObject + " Top-down ground decal: " + d.desc + negative})
	}
	return jobs
}

func beastJobs() []Job {
	var jobs []Job
	for _, b := range beasts {
		segs := []Seg{{Anim: "parts", Frames: 3, Out: "beasts", Each: true,
			Names: []string{b.id + "_head", b.id + "_body", b.id + "_tail"},
			W:     512, H: 512, Pivot: "center",
			Poses: "three separate parts side by side: head, one body segment, tail"}}
		jobs = append(jobs, Job{Name: "beast_" + b.id, Priority: 3, Mode: modeSheet, Rows: []int{3}, Segs: segs,
			Prompt: sheetPrompt(styleObject, b.desc+beastTone, []int{3}, segs, true)})
	}
	jobs = append(jobs,
		Job{Name: "beast_dragon_leg", Priority: 5, Mode: modeSprite, Out: "beasts/dragon_leg.png", W: 128, H: 128, Pivot: "center",
			Prompt: styleObject + " A small four-clawed dragon leg, side view" + beastTone + negative},
		Job{Name: "beast_phoenix_feather", Priority: 5, Mode: modeSprite, Out: "beasts/phoenix_feather.png", W: 256, H: 256, Pivot: "center",
			Prompt: styleObject + " One long flowing glowing phoenix tail feather, horizontal" + beastTone + negative})
	for _, m := range []struct{ id, desc string }{
		{"cloud", "a tiny white wisp of cloud"},
		{"spark", "a tiny bright spark of fire"},
		{"glow", "a tiny round glowing light mote"},
		{"feather", "a tiny single glowing feather"},
	} {
		jobs = append(jobs, Job{Name: "beast_mote_" + m.id, Priority: 5, Mode: modeSprite, Out: "beasts/mote_" + m.id + ".png",
			W: 64, H: 64, Pivot: "center", Prompt: styleObject + " " + m.desc + " particle, white" + beastTone + negative})
	}
	return jobs
}

// Nền bản đồ, decal và sự sống (ART_PROMPTS §9.0b, GAME_SPEC §8.6). Nền phải NHẠT và ÍT
// chi tiết: vật chơi được (EXP, item, quái, nhân vật) luôn phải nổi hơn nền (§9.0).
const seamless = ", seamless tileable top-down, desaturated, low contrast, low detail, no objects"

var biomeTiles = []struct{ id, desc string }{
	{"rocky_ground", "grey rocky ground"},
	{"fine_gravel", "fine gravel"},
	{"mossy_stone", "patchy moss on stone"},
	{"soft_grass", "soft muted grass"},
	{"bamboo_litter", "dry bamboo leaves litter on soil"},
	{"damp_earth", "damp dark earth"},
	{"dirt_path", "trampled dirt path ground"},
	{"village_grass", "short village grass"},
	{"worn_cobble", "old worn cobblestone"},
	{"dusty_earth", "dusty packed earth"},
}

var decals = []struct{ id, desc string }{
	{"ground_cracks", "ground crack lines"},
	{"grass_tuft", "small grass tuft"},
	{"fallen_leaves", "scattered fallen leaves"},
	{"pebbles", "cluster of pebbles"},
	{"puddle", "small shallow puddle"},
	{"tree_roots", "exposed tree roots"},
	{"wildflowers", "tiny pale wildflowers"},
	{"roof_tiles", "broken roof tile shards"},
	{"brick_rubble", "scattered brick rubble"},
	{"bamboo_leaves", "bamboo leaves pile"},
	{"moss_patch", "moss patch"},
	{"footprints", "footprints in dirt"},
}

var zoneEdges = []struct{ id, desc string }{
	{"mud_edge", "muddy edge with dry cracks"},
	{"pond_shore", "pond shore with reeds and pebbles"},
	{"stone_path_edge", "stone path edge with broken gravel"},
	{"bush_edge", "bush cluster edge with leaves"},
}

var plateParts = []struct {
	id, desc string
	w, h     int
}{
	{"ribbon_top1", "empty horizontal ribbon banner with forked tails, vermilion red with gold trim, no text", 1024, 256},
	{"ribbon_top2", "empty horizontal ribbon banner with forked tails, dark navy with silver trim, no text", 1024, 256},
	{"ribbon_top3", "empty horizontal ribbon banner with forked tails, brown with bronze trim, no text", 1024, 256},
	{"crown_small", "tiny golden crown icon with red gems", 128, 128},
	{"realm_pill", "empty rounded pill-shaped badge, white with light grey bevel for tinting, no text", 512, 128},
	{"shine_streak", "soft diagonal white light streak", 256, 64},
}

var portraitMoods = []struct{ id, desc string }{
	{"neutral", "neutral smug face"},
	{"evil", "evil grin, chasing"},
	{"panic", "panicked, sweating"},
	{"cry", "crying with streams of tears"},
}

func mapArtJobs() []Job {
	var jobs []Job
	for _, t := range biomeTiles {
		jobs = append(jobs, Job{Name: "tile_" + t.id, Priority: 2, Mode: modeTile, Out: "tiles/" + t.id + ".png",
			W: 512, H: 512, Prompt: styleObject + " " + t.desc + seamless})
	}
	for _, d := range decals {
		jobs = append(jobs, Job{Name: "decal_" + d.id, Priority: 3, Mode: modeSprite, Out: "decals/" + d.id + ".png",
			W: 256, H: 256, Pivot: "center",
			Prompt: styleObject + " top-down game decal, " + d.desc + ", soft muted colors, blends into the ground, thin outline" + negative})
	}
	for _, e := range zoneEdges {
		jobs = append(jobs, Job{Name: "decal_" + e.id, Priority: 4, Mode: modeSprite, Out: "decals/" + e.id + ".png",
			W: 512, H: 128, Pivot: "center",
			Prompt: styleObject + " top-down zone border strip, " + e.desc + ", tileable left to right, soft muted colors" + negative})
	}
	for _, p := range plateParts {
		jobs = append(jobs, Job{Name: "plate_" + p.id, Priority: 3, Mode: modeSprite, Out: "ui/" + p.id + ".png",
			W: p.w, H: p.h, Pivot: "center", Prompt: style + " " + p.desc + negative})
	}
	for _, ch := range characters {
		for _, m := range portraitMoods {
			jobs = append(jobs, Job{Name: "portrait_" + ch.id + "_" + m.id, Priority: 5, Mode: modeSprite,
				Out: "characters/" + ch.id + "/portrait_" + m.id + ".png", W: 512, H: 512, Pivot: "center",
				Prompt: style + " " + ch.desc + ", head and shoulders portrait, facing the viewer slightly turned right, centered, " + m.desc + negative})
		}
	}
	return jobs
}

func allJobs() []Job {
	var jobs []Job
	for _, f := range []func() []Job{skillArtJobs, swordJobs, itemJobs, characterJobs, bodyFaceJobs, monsterJobs, mapJobs, obstacleJobs, gateJobs, vfxJobs, beastJobs, mapArtJobs, uiJobs} {
		jobs = append(jobs, f()...)
	}
	return jobs
}
