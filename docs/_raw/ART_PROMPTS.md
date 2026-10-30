# KIẾM XOAY TU TIÊN — Art, Animation & Audio Prompt Pack

> Dùng cho tool sinh ảnh (minigen). Mọi asset 2D, nền trong suốt (PNG RGBA).
> Prompt viết tiếng Anh vì đa số model ảnh hiểu tốt hơn; mô tả tiếng Việt kèm theo để bạn chỉnh.

---

## 1. Quy chuẩn kỹ thuật (BẮT BUỘC cho mọi frame)

| Thuộc tính | Giá trị |
|---|---|
| Canvas mỗi frame | 512×512 px, nền trong suốt |
| Góc nhìn | 3/4 top-down, nhân vật hơi nghiêng **quay sang PHẢI** (client tự lật trái) |
| Tỉ lệ | Chibi, đầu to ≈ 40% chiều cao cơ thể |
| Vị trí | Nhân vật cao ~70% canvas, **chân chạm y = 460px, tâm x = 256px** (pivot cố định) |
| Viền | Viền đen dày 4–6px, tô màu phẳng + 1 lớp bóng (cel shading) |
| Không có | Chữ, watermark, nền, bóng đổ dưới chân (game tự vẽ bóng) |

Kích thước nhỏ hơn được: item/kiếm 256×256, VFX 256×256 hoặc 512×512.

Lưu ý về scale: nhân vật to tới 5× trong thế giới nhưng camera lùi ra tương ứng, nên trên màn hình chỉ to tối đa ~1.3× → sprite 512px là đủ nét, không cần vẽ bản to riêng. Vật cản lớn (đá tảng, tường) vẽ ở 1024×1024.

## 2. Style lock (dán vào ĐẦU mọi prompt)

Mục tiêu: **xấu mà mắc cười, bựa, vô duyên một cách đáng yêu** — không phải "cute sạch sẽ".

```
STYLE: 2D game sprite, grotesque comedic caricature, ugly-funny lowbrow slapstick cartoon,
huge head on a small scrawny or saggy body, shamelessly exaggerated facial features:
crooked or missing teeth, bulging or droopy mismatched eyes, big pores and oily shiny skin,
moles with a single long hair, sweat drops, rosy blotchy cheeks, double chin,
over-the-top smug or stupid facial expressions, cheap mismatched clothes with stains,
thick wobbly black outline, flat saturated colors with simple cel shading,
Vietnamese street comedy vibe mixed with wuxia cultivation fantasy,
transparent background, no text, no watermark, no ground shadow, centered, full body visible
```

Negative prompt (nếu tool hỗ trợ):
```
cute, kawaii, pretty, handsome, beautiful, elegant, clean, polished, heroic, idealized,
realistic, photo, 3d render, blurry, extra limbs, cropped body, text, logo, watermark,
background scenery, multiple characters, inconsistent outfit, nudity
```

### 2.1 Công thức "bựa" cho mỗi nhân vật
Mỗi nhân vật chọn đúng **3 "dấu ấn bựa" cố định** và nhắc lại nguyên văn trong MỌI prompt frame (đây là chìa khóa giữ nhất quán — chi tiết càng nhiều thì càng dễ lệch giữa các frame):
1. **Một khuyết điểm trên mặt** (răng, mắt, nốt ruồi, lông mũi, son lem...).
2. **Một thói quen vô duyên** (ngoáy mũi, ợ, gãi mông, liếm môi, hút rột rột...) → dùng cho `idle` và `chase`.
3. **Một món đồ rẻ tiền lố bịch** (dép tổ ong, dây chuyền xích chó, quần sịp hoa lộ ra...).

Biểu cảm phải **"quá lố 200%"**: cười thì toét tới mang tai, khóc thì nước mắt thành vòi, sợ thì mặt tái mét lồi mắt. Khi thấy frame nào "dễ thương" quá → sinh lại.

### 2.2 Giới hạn
Bựa nhưng không: khỏa thân, tình dục, máu me, chế giễu dân tộc/vùng miền/tôn giáo/khuyết tật hay giới tính. Hài đến từ thói quen và tình huống, không đến từ việc nhạo một nhóm người.

## 3. Quy trình giữ nhất quán khi sinh keyframe

1. **Bước 1 — Character reference sheet**: sinh 1 ảnh tham chiếu cho mỗi nhân vật (prompt mục 4.1). Chọn bản đẹp nhất, khóa seed.
2. **Bước 2 — Sinh frame**: mỗi frame dùng ảnh tham chiếu làm image reference + cùng seed + prompt frame (mục 4.2). Nếu tool hỗ trợ, sinh cả **sprite sheet dạng lưới** trong 1 ảnh (mục 4.3) — thường đồng đều hơn sinh lẻ.
3. **Bước 3 — Hậu kỳ** (tự động hóa bằng script):
   - Xóa nền (nếu model không ra nền trong suốt).
   - Căn pivot: tìm bounding box, đặt chân về y=460, x=256.
   - Chuẩn hóa màu theo palette của nhân vật (mục 5).
   - Xuất GIF preview để duyệt: `ffmpeg` hoặc `gifski`.
   - Ghép atlas: `tools/atlas-pack` → `character_atlas.png` + `character_atlas.json`.
4. **Bước 4 — Duyệt**: xem GIF loop, frame nào lệch trang phục/mặt thì sinh lại riêng frame đó.

## 4. Template prompt nhân vật

### 4.1 Reference sheet
```
{STYLE}
Character reference sheet of {CHARACTER_DESCRIPTION}.
Show 3 views in a row: front, 3/4 facing right, side facing right. Below: 6 facial expressions
(neutral, screaming, evil grin, panicked, crying, shocked). Same outfit and colors in every view.
Color palette: {PALETTE}. White background separated panels.
```

### 4.2 Một keyframe
```
{STYLE}
{CHARACTER_DESCRIPTION}, 3/4 view facing right.
Animation "{ANIM_NAME}", frame {i} of {N}: {POSE_DESCRIPTION_FOR_FRAME_i}.
Expression: {EXPRESSION}. Keep exact same outfit, colors, proportions as reference.
512x512, feet at bottom center, transparent background.
```

### 4.3 Sprite sheet dạng lưới
```
{STYLE}
Sprite sheet, {COLS}x{ROWS} grid, each cell 512x512, of {CHARACTER_DESCRIPTION} facing right,
animation "{ANIM_NAME}" in {N} sequential frames read left-to-right, top-to-bottom:
1) {pose1} 2) {pose2} ... {N}) {poseN}.
Identical character in every cell, same size, feet aligned on same baseline, transparent background.
```

### 4.3b Cách sinh mới (v6.1) — một thân, nhiều khuôn mặt

Thay cho việc sinh `run`/`chase`/`flee` riêng (xem GAME_SPEC §12.5):
1. **Thân không mặt**: `run` 6 frame + `idle` 4 frame, prompt thêm `face area left plain and neutral (closed mouth, simple eyes), same head position in every frame` — để lớp mặt chồng lên khớp.
2. **Bộ mặt** (cùng 1 ảnh lưới 4×2, mỗi ô 256×256, chỉ phần mặt nhìn hơi nghiêng phải, cùng kích cỡ và vị trí mắt):
   `neutral smug` · `screaming wide open mouth` · `evil grin narrowed eyes` · `panicked sweating bulging eyes` · `crying streams of tears` · `dizzy spiral eyes` · `shocked jaw drop` · `sleepy half-closed eyes`.
   Prompt: `{STYLE} sprite sheet 4x2 grid of facial expressions of {CHARACTER_DESCRIPTION}, face only, identical head shape, size and eye position in every cell, 3/4 view facing right, transparent background`.
3. Anim một lần (hit, death...) giữ nguyên bảng 4.4 nhưng **phải qua bước chuẩn hóa** của `tools/atlas-pack`.
4. Duyệt bằng GIF + ảnh onion-skin; frame có điểm độ giật đỏ → sinh lại riêng frame đó.

**Tùy chọn rig cắt mảnh (nâng cấp sau)**: 1 ảnh nhân vật tư thế chữ A (`A-pose, arms slightly away from body, legs apart, facing right 3/4, transparent background, 1024x1024`) rồi cắt thành đầu / thân / tay trái / tay phải / chân trái / chân phải (mỗi mảnh vẽ thêm phần khớp tròn ở chỗ nối).

### 4.4 Danh sách animation & keyframe

| Anim | Frames | Loop | Mô tả từng frame (dán vào POSE_DESCRIPTION) |
|---|---|---|---|
| `idle` | 4 | ✔ | 1 đứng thở ra · 2 hít vào vai nhô · 3 ngáp há to · 4 gãi đầu |
| `run` | 8 | ✔ | Chạy guồng chân cartoon, tay vung loạn xạ, **miệng há la hét**, tóc bay; 8 pha chu kỳ chạy (contact, down, pass, up × 2 chân) |
| `chase` | 8 | ✔ | Chạy rón rén nghiêng người tới trước, **cười nham hiểm** mắt híp, xoa hai tay vào nhau; 8 pha chạy |
| `flee` | 8 | ✔ | Chạy hoảng loạn, **hai tay giơ lên trời**, mồ hôi văng, mắt trợn, miệng méo; 8 pha chạy |
| `hit` | 4 | ✘ | 1 mặt bị ép bẹp sang trái · 2 người văng ra sau · 3 mắt xoắn ốc, sao quanh đầu · 4 lắc đầu tỉnh lại |
| `levelup` | 6 | ✘ | 1 thủ thế · 2–4 hào quang bùng lên, tạo dáng siêu anh hùng · 5 giơ ngón cái · 6 về thủ thế |
| `breakthrough` | 10 | ✘ | 1 ngồi thiền · 2–3 run rẩy · 4 tóc dựng đứng · 5–6 ngửa mặt hét lên trời · 7–8 sét đánh trúng người · 9 cháy xém tóc xù khói · 10 cười đắc thắng |
| `realm_drop` | 6 | ✘ | 1 sững sờ · 2 đầu gối khuỵu · 3 ngồi bệt · 4–5 khóc thành suối nước mắt · 6 lau mũi đứng dậy |
| `death` | 8 | ✘ | 1–2 bị đập · 3–4 dẹp lép như bánh tráng · 5 hồn trắng bay ra · 6–8 hồn vẫy tay bay lên |
| `respawn` | 6 | ✘ | 1 rơi từ trên cao · 2 sắp chạm đất · 3 đáp cắm mặt · 4 lộn nhào · 5–6 đứng dậy phủi bụi tạo dáng |
| `teleport_out` | 4 | ✘ | 1 giật mình · 2 người xoắn lại như bị hút · 3 chỉ còn cái đầu mếu máo · 4 còn lại đôi dép |
| `teleport_in` | 4 | ✘ | 1 bị phun ra giữa không trung · 2 chới với · 3 đáp đất loạng choạng · 4 mắt xoắn ốc chóng mặt |

Tổng: 76 frame / nhân vật.

## 5. Dàn nhân vật (nguyên bản, bản "bựa")

> Nhân vật là thiết kế riêng của bạn để tránh rủi ro bản quyền và hình ảnh người thật.

Mỗi nhân vật: **CHARACTER_DESCRIPTION** (dán vào prompt) + **gag riêng cho từng anim** (dán vào POSE_DESCRIPTION, thay cho mô tả chung ở bảng 4.4 khi phù hợp).

### 1. A Hớn Bánh Bao — `a_hon`
- **Prompt**: `a loud-mouthed scrawny street vendor with a giant steamed bun as a hat that has a big bite missing, two huge yellowish buck teeth with one chipped, a mole on his cheek with one long curly hair, sweat-stained yellowed sleeveless undershirt riding up over a round hairy belly button, baggy red shorts, mismatched flip-flops`
- **Palette**: #FFFFFF #E63946 #F4A261 #2B2D42
- `idle`: ngoáy mũi, nhìn ngón tay, búng gỉ đi · `chase`: thè lưỡi liếm mép, nhướng một bên mày, xoa tay · `hit`: chiếc răng thỏ mẻ văng ra · `death`: bẹp dí, cái bánh bao trên đầu bị cắn thêm một miếng.

### 2. Bà Tám Loa — `ba_tam_loa`
- **Prompt**: `a stout loud fish market lady holding a dented megaphone, bright red lipstick smeared far outside her lips and stuck on her teeth, two perfectly round blotchy pink blush circles, eyebrows drawn way too high in pencil, a tiny pencil stuck in her hair bun, floral blouse with fish scales stuck on it, wet rubber apron, tall green rubber boots`
- **Palette**: #2A9D8F #E9C46A #F28482 #264653
- `run`: hét vào loa, nước bọt văng thành tia · `chase`: chu môi son lem hôn gió đe dọa · `flee`: loa úp ngược vào mặt, chạy mù · `hit`: một vệt son in lên mặt kẻ chém (VFX riêng).

### 3. Ông Ba Võng — `ong_ba_vong`
- **Prompt**: `a skinny lazy old man with only two teeth left, nose hairs so long they look like a mustache, three strands of hair combed over a shiny bald head, half-closed sleepy eyes, toothpick in mouth, folded hammock carried on his back like a cape, saggy white tank top, striped pajama pants slipping down showing flowery underwear waistband`
- **Palette**: #8D99AE #EDF2F4 #6D597A #B56576
- `idle`: ngủ gật đứng, bong bóng mũi phồng xẹp · `flee`: quần tụt tới gối, vừa chạy vừa kéo · `hit`: ba sợi tóc bay lên dựng đứng · `breakthrough`: tóc mọc lại dày đặc trong 1 frame rồi rụng sạch.

### 4. Đại Ca Dép Tổ Ong — `dai_ca_dep`
- **Prompt**: `a pompous short gang boss in an oversized shiny purple suit with shoulder pads, tiny round black sunglasses, a gold chain as thick as a dog leash, a thin pencil mustache, slicked-back hair so greasy that flies buzz around it, a gold front tooth, cheap honeycomb plastic sandals with socks`
- **Palette**: #7209B7 #FFD60A #000000 #F72585
- `idle`: soi răng vàng vào gương con · `chase`: búng tay gọi đàn em không tồn tại, cười khẩy · `hit`: kính rơi lộ đôi mắt lé · `realm_drop`: vest bung hết cúc.

### 5. Thằng Mập Trà Sữa — `map_tra_sua`
- **Prompt**: `a round chubby teen with huge jiggly cheeks and a double chin, tapioca pearls stuck around his mouth, clutching a giant bubble tea cup with a fat straw, oversized hoodie with sweat-soaked armpits, tiny shorts, untied sneakers`
- **Palette**: #CDB4DB #A47148 #FFC8DD #3D405B
- `idle`: ợ ra một bong bóng · `chase`: hút trà sữa rột rột, mắt híp gian · `hit`: phun trân châu như súng máy · `death`: bẹp, ly trà sữa đổ thành vũng.

### 6. Cô Ba Gánh Hàng — `co_ba_ganh`
- **Prompt**: `a hyperactive young fruit seller with one thick unibrow, a huge gap-toothed grin, two stiff braids sticking straight out sideways, a crooked conical hat, balancing a bamboo shoulder pole with two overflowing fruit baskets, patched brown blouse, mismatched plastic slippers`
- **Palette**: #BC6C25 #DDA15E #606C38 #FEFAE0
- `chase`: xoay đòn gánh như cánh quạt trực thăng · `flee`: trái cây văng tung tóe rơi trúng đầu mình · `hit`: hai bím tóc cuộn xoắn lò xo.

### 7. Sư Phụ Mì Gói — `su_phu_mi`
- **Prompt**: `an eccentric old kung fu master whose long beard and eyebrows are made of curly instant noodles, cross-eyed, bright red bulbous nose, one eye squinting, stained orange robe with soup splashes, holding a pair of chopsticks like a weapon`
- **Palette**: #FB8500 #FFB703 #8ECAE6 #023047
- `idle`: gắp một sợi râu mì của mình ăn · `chase`: múa đũa, mắt lé nhìn hai hướng · `hit`: râu mì bị chém đứt văng tung tóe.

### 8. Bé Tí Cơ Bắp — `be_ti`
- **Prompt**: `a very tiny man with absurdly huge oiled shiny bodybuilder muscles, a chubby baby face with scruffy stubble, a sweatband, a tiny wrestling singlet, veins popping on his forehead, constantly flexing`
- **Palette**: #E76F51 #F4A261 #2A9D8F #1D3557
- `idle`: hôn bắp tay của mình · `chase`: gồng tới mức mặt đỏ, gân trán nổi · `hit`: cơ bắp xì hơi xẹp như bong bóng rồi phồng lại.

### 9. Anh Shipper Tia Chớp — `shipper`
- **Prompt**: `a sunburned delivery rider with a dented helmet, a pale sunglasses tan line across his face, a face mask pulled down under his chin, a fake overly polite grin, reflective vest, a giant delivery box on his back with a leaking sauce stain, phone in hand`
- **Palette**: #FFBE0B #FB5607 #3A86FF #222222
- `chase`: giơ điện thoại lên, mặt "anh ơi xuống lấy hàng" · `flee`: thùng hàng bật nắp, đồ ăn bay ra · `hit`: điện thoại văng, màn hình nứt.

### 10. Cụ Rùa Thiền — `cu_rua`
- **Prompt**: `a tiny ancient grandpa with eyebrows so long they drag on the ground, giant earlobes, a toothless peaceful smile, wearing a turtle shell backpack covered in cheap stickers, walking cane, flip-flops`
- **Palette**: #52B788 #D8F3DC #6F4E37 #F1FAEE
- `run`: chân guồng siêu nhanh nhưng người gần như không nhích · `flee`: rụt đầu và tay chân vào mai, mai tự trượt · `hit`: lông mày quấn quanh đầu như khăn.

Thêm nhân vật mới: giữ công thức **1 nghề/đặc điểm đời thường + 1 đạo cụ biểu tượng + 3 dấu ấn bựa (mục 2.1)**.

### 5.1 Sửa lỗi kỹ thuật thấy trong bản sinh thử
- **Lưới không đều** (ví dụ hàng trên 5 hình, hàng dưới 8 hình, kích thước nhân vật khác nhau): ghi rõ trong prompt `exactly {COLS} columns and {ROWS} rows, equal cell size, character the same height in every cell, feet on the same baseline`. **Nhân vật: mỗi sheet chỉ chứa 1 animation.**
- **Asset khác được gộp** để tiết kiệm quota, theo quy tắc:
  - Ảnh tĩnh 1 frame (decor, huy hiệu, icon item, skin kiếm, huy chương): gộp thoải mái nhiều món/1 ảnh.
  - Vật cản: 4 trạng thái của **cùng 1 vật cản** nên nằm chung 1 ảnh (giúp cùng góc nhìn, cùng kích thước); anim `break` và `regrow` tách riêng.
  - Quái, cổng, Rương Báu, VFX nhiều frame: gộp nhiều anim của **cùng 1 đối tượng** vào 1 ảnh, **mỗi anim 1 hàng riêng**, ghi rõ trong prompt `row 1: idle 4 frames, row 2: hurt 2 frames, ...`. Không trộn 2 đối tượng khác nhau trong 1 ảnh có animation.
  - Ảnh gộp nào ra lệch lưới → chỉ sinh lại phần lỗi dưới dạng sheet tách riêng.
- **Cắt frame**: tool cắt nên tìm từng hình theo vùng pixel liền nhau (connected components / bounding box) thay vì chia lưới cứng, rồi căn lại pivot. Như vậy sheet hơi lệch lưới vẫn dùng được, đỡ phải sinh lại. Với nhân vật, giữ nguyên tỉ lệ gốc (không tự co giãn từng frame), chỉ căn chân về y=460.
- **Hướng nhìn lẫn lộn** (lúc chính diện, lúc nghiêng): thêm `every frame facing right in 3/4 view`.
- **Nền trắng**: script hậu kỳ xóa nền, nhưng viền trắng quanh nhân vật sẽ lộ trên nền cỏ → yêu cầu nền `pure flat magenta #FF00FF background` để xóa sạch hơn (nếu tool không ra được nền trong suốt).

## 6. Quái (PvE)

Mỗi quái: `idle` 4 frame (lắc lư), `hurt` 2 frame, `death` 6 frame (vỡ tung thành viên EXP). Không cần chừa vùng trống in số HP (game dùng thanh máu nhỏ trên đầu); nếu prompt bên dưới còn cụm `blank ... area` thì bỏ đi.

| Loại | Prompt |
|---|---|
| Nhỏ — Bánh Bao Cáu | `{STYLE} an angry living steamed bun creature with tiny legs, steam puffing from head, round belly area left blank in the center` |
| Nhỏ — Dép Lào Biết Đi | `{STYLE} a sentient flip-flop sandal creature hopping, googly eyes, sticking tongue out, blank round circle on its sole` |
| Vừa — Nồi Cơm Điện Cắn | `{STYLE} a rice cooker monster with its lid as a biting mouth full of rice teeth, blank round panel on front` |
| Vừa — Gà Hoảng Loạn | `{STYLE} a panicking chubby chicken monster, feathers flying, bulging eyes, blank round badge on chest` |
| Lớn — Muỗi Khổng Lồ | `{STYLE} a giant fat mosquito monster with a straw-like nose, tiny wings, smug face, blank round area on belly` |
| Boss — Vua Ghế Nhựa | `{STYLE} a towering boss made of stacked red plastic stools, crown of a plastic basin, angry eyes, blank big circle in the middle` |

## 7. Kiếm theo cảnh giới (skin đổi khi đột phá)

Canvas 256×256, kiếm **thẳng đứng, mũi hướng LÊN**, cán ở dưới. Thêm bản `glow` 4 frame loop nếu muốn.

**Để nhuộm màu theo vòng được** (game tô mỗi vòng một màu): thân lưỡi kiếm vẽ bằng **màu sáng, gần trắng/xám nhạt** với bóng đổ xám, chỉ chuôi kiếm và chi tiết nhỏ mới có màu riêng. Thêm vào prompt: `blade painted in very light grey and white tones so it can be color-tinted in game, colored details only on the handle`.

Tùy chọn giống quảng cáo — **lưỡi liềm cong**: `{STYLE} a single curved crescent-shaped flying blade, like a sharp crescent moon, thick at the middle and pointed at both ends, light grey and white tones for tinting, vertical, transparent background`. Có thể dùng thay skin kiếm cho 1–2 cảnh giới để đổi gió.

```
{STYLE} single game weapon icon, {SWORD}, vertical, tip pointing up, centered, transparent background
```

| Cảnh giới | {SWORD} |
|---|---|
| Luyện Khí | `a pair of bamboo chopsticks tied together as a sword` |
| Trúc Cơ | `a kitchen cleaver with a wooden handle` |
| Kim Đan | `a peach wood taoist sword with red tassel` |
| Nguyên Anh | `a polished steel jian sword` |
| Hóa Thần | `a flaming red-orange sword with fire trail` |
| Luyện Hư | `an ice crystal blue sword with frost sparkles` |
| Hợp Thể | `a crackling lightning yellow sword` |
| Độ Kiếp | `a dark purple cursed sword with smoky aura` |
| Đại Thừa | `a radiant jade and gold celestial sword glowing white` |
| Chân Tiên | `an elegant celestial sword wrapped in soft white clouds` |
| Kim Tiên | `a pure golden immortal sword with floating golden runes` |
| Thái Ất Ngọc Tiên | `a translucent white jade sword with a faint inner glow` |
| Đại La Kim Tiên | `a cosmic sword with a starry night sky blade and gold edges` |
| Đạo Tổ | `a sword made of swirling yin-yang black and white energy, radiant halo` |

## 8. Vật phẩm

| Asset | Prompt |
|---|---|
| Bong bóng kiếm | `{STYLE} a glossy transparent soap bubble containing a small glowing sword, shiny highlight, 256x256` |
| Bó Kiếm (`item_bundle` → `items/bundle.png`) | `{STYLE} a glossy transparent soap bubble containing a bundle of five small swords tied together with a red ribbon, swords fanned out slightly, golden glow, shiny highlight` (đã sinh theo bản này — giữ nguyên). Trong game to hơn bong bóng kiếm thường 20%; bó rơi từ xác in số kiếm lên bong bóng |
| EXP nhỏ/vừa/lớn | `{STYLE} a glowing {green|blue|gold} spirit orb gem, sparkly, 128x128` |

Vật phẩm buff: mỗi cái là 1 bong bóng chứa icon, cùng 1 template:
```
{STYLE} a glossy transparent soap bubble containing {ICON}, tinted {COLOR}, shiny highlight, 256x256
```

| Item | {ICON} | {COLOR} | Aura trên nhân vật (VFX loop 4 frame) |
|---|---|---|---|
| Tốc Kiếm | `a cyan spinning whirlwind around a small sword` | cyan | vòng gió xanh quanh vòng kiếm |
| Thần Hành | `a winged sandal` | orange | vệt khói chạy sau chân |
| Nam Châm | `a red horseshoe magnet` | red | sóng tròn hút vào |
| Cương Kiếm | `a thick heavy steel sword with a diamond sparkle on the blade` | silver-blue | lưỡi kiếm lóe ánh thép, tia lấp lánh chạy dọc vòng kiếm |
| Kim Cang | `a golden buddhist-style round shield` | gold | quả cầu vàng trong suốt bao quanh người |
| Song Bội Đan | `a glowing pill with "x2" shape made of light` | green | đốm sáng xanh bay lên |
| Ẩn Thân Phù | `a paper talisman with swirling ink` | grey | (chỉ chủ nhân thấy mình mờ 40%) |

Rương Báu: `{STYLE} an ornate wooden treasure chest wrapped in red ribbons, glowing golden light leaking out, 512x512` — anim `idle` 4 frame rung lắc, `open` 6 frame bung nắp, đồ bay ra.

Icon buff cho HUD: dùng lại icon ở trên, nền tròn, 128×128.

Bong bóng: thêm anim `float` 4 frame (nhấp nhô) và `pop` 4 frame.

## 9. Bản đồ

### 9.0 Quy tắc dễ nhìn (QUAN TRỌNG — đọc trước khi sinh bất kỳ asset bản đồ nào)
Người chơi phải nhìn ra **quái, viên EXP, vật phẩm, kiếm** trong nửa giây. Vì vậy chia 2 tầng màu:

| Tầng | Quy tắc |
|---|---|
| **Nền & trang trí** (đất, cỏ, decor) | Màu **nhạt, ít bão hòa, độ tương phản thấp**, ít chi tiết nhỏ. **Không hoa, không chấm màu sặc sỡ** (dễ lẫn với viên EXP/item). Viền mảnh, màu tối của chính nó (không viền đen dày). |
| **Thứ chơi được** (nhân vật, quái, EXP, item, kiếm, cổng) | Màu **bão hòa, sáng**, viền đen dày, có bóng đổ dưới chân + quầng sáng nhẹ. |

- Mỗi biome tối đa 3 màu nền gần nhau. Ví dụ Sân làng: xanh rêu nhạt #8DA77A, #9DB58A, #7E9A6C.
- Viên EXP/item dùng màu **không có trong nền**: EXP = xanh lơ phát sáng, vàng kim; item = bong bóng trắng bóng.
- Decor: thưa (không quá 1 món / 600×600 đơn vị), nhỏ, không to hơn nhân vật quá 2 lần trừ khi là vật cản.

**Sửa nhanh không cần sinh lại art**: client áp `ColorMatrixFilter` lên lớp nền — giảm bão hòa ~50%, giảm sáng ~10%, giảm tương phản ~20%. Áp riêng cho lớp nền + decor, không áp cho lớp gameplay.


Nền: tile vuông **seamless/tileable** 512×512, nhìn top-down.

```
{STYLE without "transparent background"} seamless tileable top-down ground texture, {BIOME}, cartoon game style,
no objects, no characters, even lighting, edges must tile perfectly
```

| Biome | {BIOME} |
|---|---|
| Sân làng | `soft muted sage green grass, very low contrast, subtle large brush strokes, no flowers, no small colorful dots, calm and plain` |
| Khu chợ | `pale warm grey concrete, very low contrast, few faint cracks, plain and calm` |
| Bãi đất | `light muted sandy brown dirt, very low contrast, only a few faint large pebbles, plain and calm` |

Thêm vào cuối prompt nền: `desaturated, low detail, designed as a background that does not distract from gameplay objects`.

Decor (nền trong suốt, 256×256 hoặc 512×512, chỉ để trang trí, không chặn đường):
`a red plastic stool`, `an old bicycle leaning`, `a pile of watermelons`, `a street food cart`, `a banana tree`,
`a rusty water jar`, `a broken TV`, `a lazy sleeping dog`, `a bush with flowers`, `a stone lantern`.

Tường biên: `{STYLE} a seamless horizontal stone and bamboo fence segment, top-down 3/4 view, tileable left-right`.

### 9.0b Bộ texture & decal cho bản đồ sinh động (GAME_SPEC §8.6)

Mọi texture nền: thêm cuối prompt `seamless tileable top-down, desaturated, low contrast, low detail, no objects`.

| Biome | Texture nền (2–3 cái, 512 tileable) |
|---|---|
| Linh Sơn | `grey rocky ground` · `fine gravel` · `patchy moss on stone` |
| Trúc Lâm | `soft muted grass` · `dry bamboo leaves litter on soil` · `damp dark earth` |
| Phàm Trấn | `trampled dirt path ground` · `short village grass` |
| Hắc Thị | `old worn cobblestone` · `dusty packed earth` |
| Thiên Tiên Đài | `pale jade stone floor` (đã có) |

Decal (nền trong suốt, 256×256 trừ khi ghi khác, màu nhạt hòa vào nền, viền mảnh không đen dày):
`ground crack lines` · `small grass tuft` · `scattered fallen leaves` · `cluster of pebbles` · `small shallow puddle` · `exposed tree roots` · `tiny pale wildflowers` · `cart wheel tracks (512x128)` · `broken roof tile shards` · `scattered brick rubble` · `bamboo leaves pile` · `moss patch` · `footprints in dirt`.
Prompt mẫu: `top-down game decal, {item}, soft muted colors, blends into the ground, thin outline, transparent background`.

Dải viền vùng (512×128, tileable trái-phải, nền trong suốt): `muddy edge with dry cracks` · `pond shore with reeds and pebbles` · `stone path edge with broken gravel` · `bush cluster edge with leaves`.

Nước & biên: `top-down seamless calm water texture, soft teal, subtle ripples` (512 tileable) · `{STYLE} top-down rocky cliff edge segment, tileable left-right, 1024x512` · `{STYLE} dense bamboo grove wall segment, tileable left-right, 1024x512`.

Ambient (nhỏ, 64–128 px, vài frame): `falling bamboo leaf` (4) · `firefly glow dot` · `butterfly flapping` (4) · `small chicken running` (4) · `swaying paper lantern` (4) · `bird silhouette flying` (4) · `soft cloud shadow blob, black at 20% opacity, 1024x1024`.

### 9.1 Vật cản phá được (mỗi loại 4 trạng thái)

Quy tắc vẽ vật cản: **đáy phẳng, đặt chắc trên mặt đất** (game chỉ tính va chạm ở 30% phía dưới). Hàng rào/tường phải **liền mạch nối tiếp được** (tileable trái-phải), không có khe hở giả. Màu vật cản trung tính hơn nhân vật nhưng đậm hơn nền.

Mỗi vật cản cần 4 ảnh cùng góc nhìn, cùng kích thước canvas: `intact` · `cracked` · `nearly_broken` · `rubble` (đống đổ nát, nhìn là biết đi qua được). Thêm 1 anim `break` 6 frame (nổ vụn) và `regrow` 6 frame (mọc lại từ đất, kèm ánh sáng).

```
{STYLE} top-down 3/4 view game obstacle, {OBSTACLE}, state: {STATE}, transparent background, {SIZE}
STATE: "intact and solid" | "with visible cracks" | "heavily damaged, crumbling, pieces missing" | "flat pile of rubble and debris on the ground"
```

| Loại | {OBSTACLE} | {SIZE} |
|---|---|---|
| Hàng rào tre | `a straight bamboo fence segment tied with rope` | 1024x256 |
| Sạp hàng | `a street market stall with a striped canopy and fruit baskets` | 512x512 |
| Xe đẩy | `a wooden street food cart with a small stove` | 512x512 |
| Tường gạch | `a short red brick wall segment with moss` | 1024x512 |
| Đá tảng | `a huge round grey boulder with cartoon cracks lines` | 1024x1024 |
| Tường đá cổ (không phá) | `an ancient mossy stone wall with carved patterns` | 1024x512 |

### 9.2 Vùng địa hình

| Vùng | Prompt | Ghi chú |
|---|---|---|
| Bụi rậm | `{STYLE} top-down dense leafy bush cluster, round, lush green, 512x512` | Vẽ 2 lớp: thân bụi + lớp lá trên cùng bán trong suốt để thấy mờ mờ người bên trong (chỉ với chính mình) |
| Bùn lầy | `seamless tileable top-down muddy brown puddle texture with bubbles` | Tile 512, viền bằng mask mềm trong code |
| Đường gió | `{STYLE} a white cartoon wind gust arrow with swirl lines` | Anim 4 frame chạy dọc làn gió |

### 9.2b Bố cục tu tiên giới (cho thuật toán sinh bản đồ)

Dùng lại art cũ cho vật cản/decor; chỉ cần sinh thêm:

| Asset | Prompt | Kích thước |
|---|---|---|
| Đường đá (tile) | `seamless tileable top-down pale stone path made of large flat worn flagstones, muted warm grey, very low contrast, plain` | 512, tileable |
| Mép đường | `seamless horizontal top-down edge strip where pale stone path meets muted grass, soft irregular border, tileable left-right` | 512×128 |
| Nền Thiên Tiên Đài | `seamless tileable top-down polished pale jade stone floor with faint carved cloud patterns, very low contrast` | 512, tileable |
| Cột/tường quanh Thiên Tiên Đài | `{STYLE} top-down 3/4 view short white marble balustrade segment with cloud carvings, tileable left-right` | 1024×256 |
| Tường sân môn phái | `{STYLE} top-down 3/4 view ancient sect courtyard wall with grey tiles roof, tileable left-right` | 1024×512 |
| Cổng sân môn phái | `{STYLE} top-down 3/4 view wooden sect gate archway with a blank signboard, red pillars` | 1024×1024 |

### 9.2d Thiên Môn (GAME_SPEC §8.4a)

| Asset | Prompt | Anim |
|---|---|---|
| Thiên Môn | `{STYLE} top-down 3/4 view grand celestial gate, white marble pillars with a golden curved roof, swirling golden portal inside, clouds at the base, a blank signboard on top, 1024x1024` | `idle` 6 loop (xoáy vàng), `burst` 6 |
| Cột sáng | `vertical soft golden light beam rising into the sky, additive glow, transparent background, 256x1024` | `idle` 4 loop |
| Bến Thiên Môn (ở Thiên Đình) | `{STYLE} top-down small round white stone landing platform with golden cloud carvings and a faint light pillar, 512x512` | `warn` 4 (lóe trước khi có người ra) |
| Icon minimap | `{STYLE} tiny golden gate icon, 64x64` | – |

### 9.2c Đài Tiên Khí

| Asset | Prompt | Anim |
|---|---|---|
| Đài nhỏ | `{STYLE} top-down 3/4 view small round stone altar platform with a glowing golden spring of light in the center, carved taoist symbols, 1024x1024` | `idle` 6 loop (ánh sáng phun nhẹ), `charge` 6 (sáng dần trước đợt lớn), `burst` 6 (phun mạnh) |
| Thiên Tiên Đài | `{STYLE} top-down view grand circular celestial altar with three tiers, gold and jade, a beam of light rising from the center, clouds around the rim, 2048x2048` | `idle` 6 loop, `charge` 8, `burst` 8 |
| Viên Tiên khí nhỏ / vừa | `{STYLE} glowing golden celestial energy orb with a tiny cloud swirl inside, sparkly, {128x128 / 192x192}` | `float` 4 loop |
| Viên Tiên khí khổng lồ | `{STYLE} huge radiant golden celestial energy orb with swirling clouds and light rays, majestic, 512x512` | `float` 6 loop |

### 9.3 Cổng dịch chuyển

```
{STYLE} top-down magical teleport portal on the ground, swirling {COLOR} vortex inside a carved stone ring
with taoist symbols, glowing, 512x512, frame {i} of {N}
```

| Anim | Frames | Mô tả |
|---|---|---|
| `idle` | 6 loop | Xoáy chậm |
| `warn` | 4 | Lóe sáng mạnh trước khi có người chui ra (0.5s) |
| `burst` | 6 | Bùng sáng khi người chui vào/ra |

Mỗi cặp/mạng cổng một màu (tím, xanh lam, cam) để người chơi đoán được cổng nối với nhau.

## 10. VFX

| VFX | Frames | Prompt |
|---|---|---|
| Tia lửa va kiếm | 6 | `{STYLE} comic sword clash spark burst, white and yellow star shapes, frame {i} of 6 expanding then fading` |
| Sao choáng | 4 loop | `{STYLE} ring of small yellow cartoon stars orbiting, frame {i} of 4` |
| Hào quang lên cấp | 6 | `{STYLE} rising golden aura ring with sparkles, frame {i} of 6` |
| Thiên Kiếp (Tiên giới) | 8 | `{STYLE} dark storm cloud gathering then three thick cartoon lightning bolts striking down in a row, frame {i} of 8` |
| Hào quang Tiên (dùng chung, game tự nhuộm màu) | 6 loop | `{STYLE} large round glowing halo disc seen from the front, soft radiant light rays, painted in white and light grey only so it can be color-tinted in game, transparent background, 1024x1024, frame {i} of 6` |
| Hào quang Đạo Tổ (riêng) | 6 loop | `{STYLE} large glowing yin-yang halo disc with a thin rainbow rim, soft light rays, slowly rotating, transparent background, 1024x1024, frame {i} of 6` |
| Hạt trang trí hào quang | 4 loop mỗi loại | mây nhỏ (Chân Tiên), tia sáng (Kim Tiên), hạt sáng bay lên (Thái Ất), vòng sáng mảnh (Đại La) — vẽ trắng/xám để nhuộm |

### 10.1 Thần thú Tiên giới (ghép đốt bằng code, xem GAME_SPEC §3.5)

Mọi phần vẽ **nhìn ngang, hướng sang PHẢI**, nền trong suốt, viền đen dày, tông **trắng/xám nhạt** để game nhuộm màu (trừ Đạo Tổ vẽ màu thật). Thêm vào cuối mỗi prompt: `painted in white and light grey tones for color tinting, transparent background, isolated single part`.

| Phần | Prompt | Kích thước |
|---|---|---|
| Đầu rồng | `{STYLE} side view head of a fierce but funny chinese dragon facing right, long whiskers, small antlers, open mouth with a goofy grin` | 512×512 |
| Đốt thân rồng | `{STYLE} one round segment of a chinese dragon body seen from the side, overlapping scales, a small spiky fin on top, symmetrical, can be chained repeatedly` | 256×256 |
| Đuôi rồng | `{STYLE} side view tail tip of a chinese dragon facing left, flowing fin like a flame` | 256×256 |
| Chân rồng (tùy chọn, gắn ở đốt 3 và 8) | `{STYLE} small four-clawed dragon leg, side view` | 128×128 |
| Tiên hạc | `{STYLE} side view flying crane, long neck stretched forward facing right, wings {up|middle|down|middle}, frame {i} of 4` | 512×512 |
| Phượng hoàng | `{STYLE} side view flying phoenix facing right, long flowing tail feathers, wings {up|middle|down|middle}, frame {i} of 4` | 512×512 |
| Lông đuôi phượng (đốt) | `{STYLE} one long flowing glowing phoenix tail feather, horizontal` | 256×128 |
| Song long Đạo Tổ | Dùng lại đầu/đốt/đuôi rồng nhưng sinh 2 bộ màu thật: `pure white glowing dragon` và `pure black dragon with white glowing outline` | như trên |
| Hạt vệt sau | mây nhỏ / tia lửa / hạt sáng / lông vũ — `{STYLE} tiny {item} particle, white, 64x64` | 64×64 |

Mẹo giữ đều nét: sinh **đầu + đốt + đuôi trong cùng 1 ảnh** (`three separate parts side by side: head, one body segment, tail`) để cùng phong cách và độ dày viền, rồi để tool cắt tự tách.
| Sét đột phá | 6 | `{STYLE} thick cartoon lightning bolt striking down from the top, frame {i} of 6` |
| Khói chết | 6 | `{STYLE} comic white smoke puff cloud, frame {i} of 6 expanding then dissipating` |
| Bụi chạy | 4 | `{STYLE} small dust puffs behind running feet, frame {i} of 4` |

| Kiếm bể | 6 | `{STYLE} a cartoon sword shattering into sharp shards flying outward, frame {i} of 6` |
| Vết nứt vòng kiếm | 1 | `{STYLE} thin white crack lines overlay, transparent, 256x256` (chồng lên kiếm khi bị nứt) |
| Khiên vỡ | 6 | `{STYLE} golden bubble shield cracking and bursting into glass shards, frame {i} of 6` |

Bong bóng thoại (UI): sinh khung trống `{STYLE} empty white comic speech bubble with black outline`; chữ do game render (dùng font hỗ trợ tiếng Việt).
Gợi ý câu thoại: `run`: "AAAAA!", "Tránh ra!"; `chase`: "Hehehe...", "Đứng lại đó!"; `flee`: "Cứu tôi với!", "Mẹ ơi!"; `hit`: "Ối giời!", "Đau quá!"; `breakthrough`: "Ta đã ngộ rồi!!!"; mất kiếm: "Kiếm của taaa!"; ra cổng: "Đây là đâu???"; nấp bụi: "Suỵt...".

## 11. UI

- 9 huy hiệu cảnh giới: `{STYLE} round game badge emblem for cultivation rank "{tier theme}", 256x256` (theme theo màu skin kiếm mục 7, KHÔNG để model viết chữ — chữ do game render).
- Joystick ảo: đế tròn mờ + núm tròn.
- Thanh EXP, khung leaderboard, khung kill feed: vẽ đơn giản bằng code hoặc 9-slice.
- Trang chủ: logo game (chữ do bạn tự thiết kế hoặc render bằng font), ảnh nền: `{STYLE without transparent} wide illustration of a chaotic cartoon village square with many chibi characters spinning swords around them, sunny, no text`.
- Màn hình Tổng kết: khung giấy cuộn cổ kính `{STYLE} an empty unrolled ancient paper scroll with wooden handles, transparent background`; ảnh "hồn bay" của nhân vật dùng lại frame cuối anim `death`.
- Bảng xếp hạng: 3 huy chương top 1–3 `{STYLE} {gold|silver|bronze} medal with a tiny sword emblem, 128x128`.

### 11.1 Bảng tên (GAME_SPEC §12.3)

| Asset | Prompt | Ghi chú |
|---|---|---|
| Ruy băng Top 1 | `{STYLE} empty horizontal ribbon banner with forked tails, vermilion red with gold trim, no text, transparent background, 1024x256` | Cắt 9-slice (giữ 2 đuôi, kéo giãn phần giữa) |
| Ruy băng Top 2 | như trên, `dark navy with silver trim` | |
| Ruy băng Top 3 | như trên, `brown with bronze trim` | |
| Vương miện nhỏ | `{STYLE} tiny golden crown icon with red gems, 128x128` | |
| Nền huy hiệu cảnh giới | `{STYLE} empty rounded pill-shaped badge, white with light grey bevel for tinting, no text, 512x128` | 9-slice, game nhuộm màu |
| Vệt sáng lướt | `soft diagonal white light streak, transparent background, 256x64` | Blend cộng, chạy ngang qua chữ |

### 11.2 Chân dung HUD (GAME_SPEC §12.4)

Mỗi nhân vật 4 chân dung, cùng prompt nhân vật ở mục 5 + `{STYLE} head and shoulders portrait, facing the viewer slightly turned right, centered, {EXPRESSION}, transparent background, 512x512`:
`neutral smug face` · `evil grin, chasing` · `panicked, sweating` · `crying with streams of tears`.

### 11.3 Kỹ năng nhân vật (GAME_SPEC §6.3)

Vật phẩm **Bí Kíp**: `{STYLE} a glossy transparent bubble containing an ancient rolled-up martial arts scroll with a red ribbon, glowing, 256x256` + icon HUD 128×128.


Mỗi nhân vật thêm anim **`skill`** 6 frame (dùng template §4.3) + 1 icon kỹ năng 128×128 (`{STYLE} round skill icon, {ICON}, bold simple shape, no text`) + VFX riêng:

| Nhân vật | Pose anim `skill` | Icon | VFX |
|---|---|---|---|
| A Hớn | nhét cả cái bánh bao vào miệng, má phồng, ợ ra khói | steamed bun with a bite | hơi nóng bốc lên, kiếm bay ngược về vòng |
| Bà Tám Loa | ưỡn ngực hét vào loa, tóc bay ngược | megaphone with sound waves | sóng âm hình nón (vòng cung trắng lan ra) |
| Ông Ba Võng | nằm võng đung đưa, mắt nhắm, bong bóng mũi | hammock with zzz | vòng chữ "zzz" + sóng ru màu tím nhạt |
| Đại Ca | huýt sáo bằng 2 ngón tay, chỉ tay | whistle with two fingers | 2 đàn em nhỏ (vest tím rẻ tiền, kính đen) — sinh thêm sprite đàn em `run` 8 frame |
| Thằng Mập | phồng má phun như súng máy | bubble tea pearls flying | viên trân châu đen bay + vệt; **overlay màn hình**: 6–8 biến thể `glossy black tapioca pearl splattered flat on glass with a milk tea smear, front view, transparent background, 256x256` + `milk tea drip trail, vertical, 128x512` |
| Cô Ba | lướt ngang, đòn gánh nằm ngang, trái cây văng | fruit basket with speed lines | vệt bụi + vỏ chuối/trái cây trên đất |
| Sư Phụ Mì | quăng lưới bằng đôi đũa | noodle net | lưới sợi mì vàng chụp xuống |
| Bé Tí | gồng cơ, gân nổi, mặt đỏ, người phát sáng | flexed bicep | hào quang cam, tia gân nổi |
| Shipper | rồ ga tưởng tượng, thùng hàng phụt khói | lightning bolt on delivery box | vệt tia chớp sau lưng |
| Cụ Rùa | rụt đầu tay chân vào mai | turtle shell | mai phát sáng xanh, sóng đẩy khi bật ra |

### 11.4 Art riêng cho 4 khu mới (GAME_SPEC §8.0)

| Khu | Cần thêm |
|---|---|
| Linh Sơn Cốc | `{STYLE} top-down 3/4 view tall rocky cliff wall segment, tileable left-right, 1024x512` · `top-down wooden rope suspension bridge segment, tileable, 512x256` · nền `grey mountain rock with patches of moss` |
| Trúc Lâm Mê Cung | `{STYLE} top-down very dense bamboo grove wall segment, tileable left-right, 1024x512` · góc rặng tre (L-shape) · nền `damp forest floor with bamboo leaves` |
| Phàm Trấn | `{STYLE} top-down small village house roof (grey tiles), 1024x1024` (nhà nhìn từ trên, tường gạch là va chạm) · `paper lanterns string` · nền `old cobblestone street` |
| Băng Nguyệt Hồ | nền `seamless top-down smooth glossy pale blue ice with subtle cracks and reflections` · `seamless top-down soft snow ground` · decal `ice skid marks`, `snow pile`, `frozen reeds` · `{STYLE} small ice island edge segment` |

## 12. Âm thanh (SFX / BGM) — nguồn miễn phí

| Nguồn | Ghi chú giấy phép |
|---|---|
| kenney.nl | Đa số CC0, dùng thương mại thoải mái (có cả pack VFX/UI) |
| opengameart.org | Lọc theo CC0 hoặc CC-BY (CC-BY cần ghi credit) |
| freesound.org | Lọc theo license CC0; CC-BY cần credit; tránh CC-BY-NC nếu thương mại |
| pixabay.com (sound effects & music) | Giấy phép Pixabay, cho phép dùng thương mại |

**Luôn lưu file `CREDITS.md` ghi nguồn + license từng file.**

Danh sách SFX cần:
- `sword_spin_loop` (vù vù nhẹ, pitch tăng theo ω), `sword_clash` (keng ×3 biến thể), `sword_hit_body` (bốp, chát).
- `pickup_sword`, `pickup_buff`, `pickup_exp` (ting, pitch tăng nếu lượm liên tục), `buff_end` (xì hơi).
- `sword_break` (vỡ kính), `shield_break`, `chest_incoming` (trống báo), `chest_open`.
- `obstacle_hit` (cộc gỗ / đá), `obstacle_break` (đổ sập), `obstacle_regrow` (mọc lên "pựt").
- `gate_in` (hút "xoẹt"), `gate_out` (phụt "bùm"), `gate_warn`.
- `mud_step` (lép nhép), `bush_rustle` (sột soạt), `wind_loop`.
- `monster_hurt`, `monster_pop`.
- `levelup` (ding), `breakthrough` (sấm + "hoàaa"), `realm_drop` (kèn "wah wah" thất bại).
- `death` (tiếng "bẹp" + còi hoạt hình), `respawn` (rơi "vèoo" + đáp "bịch").
- Voice tấu hài (có thể tự thu bằng giọng team hoặc dùng TTS): la hét khi chạy, cười nham hiểm, kêu cứu, ối giời — mỗi loại 3–5 biến thể.
- BGM: 1 bài loop vui nhộn kiểu nhạc cổ phong/ngũ cung tốc độ nhanh.

## 13. Cấu trúc thư mục & đặt tên

```
assets/
├── characters/{char_id}/
│   ├── ref_sheet.png
│   ├── frames/{anim}_{00..N}.png
│   ├── atlas.png
│   └── atlas.json          # { "anims": { "run": { "frames": [...], "fps": 12, "loop": true } } }
├── monsters/{monster_id}/...
├── swords/realm_{0..8}.png
├── items/  buffs/  vfx/  tiles/  decor/  ui/
├── obstacles/{type}/{intact|cracked|nearly_broken|rubble}.png + break/ + regrow/
├── zones/  gates/{color}/
├── audio/sfx/  audio/voice/{char_id}/  audio/bgm/
└── CREDITS.md
```

FPS gợi ý: `idle` 6, `run`/`chase`/`flee` 12, các anim một lần 12–15.
`char_id` dạng snake_case không dấu: `a_hon`, `ba_tam_loa`, `ong_ba_vong`, `dai_ca_dep`, `map_tra_sua`, `co_ba_ganh`, `su_phu_mi`, `be_ti`, `shipper`, `cu_rua`.
