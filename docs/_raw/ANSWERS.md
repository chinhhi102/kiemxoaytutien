# Trả lời QUESTIONS.md (đã cập nhật vào docs/GAME_SPEC.md)

## Q1 — EXP khi chém chết người ở Luyện Khí tầng 1
**Chốt: (b) + thưởng hạ sát.** `loss = total_exp hiện tại của nạn nhân`; người chém nhận 50% loss + `kill_bonus_exp` (30, mới trong config), rơi 30% loss thành viên EXP, +1 kill. Spec mục 5.1.

## Q2 — Đường gió
**Chốt: (b).** Lệch < 60° → ×1.5; lệch > 120° → ×0.8; còn lại ×1.0. Thêm `wind_with_max_deg: 60`, `wind_against_min_deg: 120` vào config. Spec mục 8.3.

## Q3 — Rương Báu
**Giữ phương án tạm.** Người đầu tiên chạm vào → nổ 5 item văng xung quanh, ai lượm cũng được. Spec mục 6.

## Q4 — EXP từ quái
**Giữ phương án tạm, bổ sung:** viên EXP thuộc về người chém hạ quái trong 1.5s đầu (`orb_owner_secs`), sau đó ai cũng nhặt được — tránh bị cướp công. Spec mục 5.3.

## Q5 — Chết có rơi kiếm không
**Đổi: có rơi.** Chết (người và bot) → `floor(n * death_sword_drop)` kiếm (mặc định 0.5) văng ra thành item kiếm quanh xác. Người chém có phần thưởng, người khác có cơ hội "hôi của". Thêm `death_sword_drop: 0.5` vào config. Spec mục 7.

→ Việc cần làm: cập nhật config/game.json, sửa code Q1/Q2/Q4/Q5, thêm unit test tương ứng (xem checklist mục 17), xóa các câu đã trả lời khỏi QUESTIONS.md.

## Q6 — "1 animation/sheet" có áp dụng cho quái, cổng, VFX?
**Đồng ý (a), kèm quy tắc gộp** (đã ghi vào ART_PROMPTS §5.1):
- Nhân vật: 1 anim/ảnh (120 ảnh).
- Ảnh tĩnh (decor, huy hiệu, icon, skin kiếm): gộp thoải mái.
- Vật cản: 4 trạng thái của cùng 1 vật cản chung 1 ảnh; `break`/`regrow` tách riêng.
- Quái, cổng, Rương Báu, VFX nhiều frame: gộp các anim của **cùng 1 đối tượng**, **mỗi anim 1 hàng**, ghi rõ số frame từng hàng trong prompt. Không trộn 2 đối tượng có animation trong 1 ảnh.
- Ảnh gộp bị lệch → chỉ sinh lại phần lỗi dưới dạng tách riêng.

**Việc cần làm thêm trong `cmd/artgen`**: bộ cắt frame tìm từng hình theo vùng pixel liền nhau (connected components) thay vì chia lưới cứng, sau đó căn pivot (nhân vật: chân về y=460, không co giãn từng frame; vật thể/VFX: căn giữa). Sheet hơi lệch lưới vẫn dùng được → tiết kiệm quota sinh lại.

---

## Q7 — Nhịp Luyện Khí → Trúc Cơ
**Chốt 30–60s**, giữ `exp_orbs.cell = 160`. Spec §6.1, config và checklist §17 đã sửa cho khớp.

## Q8 — Số kiếm mất khi va chạm
**Đổi:** `clash_unit = 1` (bỏ nhân 5). Checklist §17 giữ đúng các con số 1:1, 1:2, 1:4.
**Luật mới:** chỉ đổi chiều quay khi **cùng cảnh giới**. Khác cảnh giới → không ai đổi chiều (config `flip_only_same_realm: true`). Đẩy văng `clash_knockback` vẫn áp dụng mọi trường hợp. Spec §5.2 đã viết lại.

## Q9 — `min_count = 0`
**Xác nhận phương án 1:** số kiếm về 0 chỉ là bị tước vũ khí, không chết. Lý do cần về 0: vòng kiếm dày chặn hết kiếm đối thủ, phải tước sạch kiếm mới chém được vào thân. Chém thân vẫn theo §5.1 (Trúc Cơ trở lên rớt 1 cảnh giới, Luyện Khí chết). Bot 0 kiếm → Flee. Sửa test checklist: "Luyện Khí còn 3 kiếm va Nguyên Anh → còn 0, Nguyên Anh nứt 3/4".

## Q10 — Phạm vi nhặt vật phẩm
**Chưa có câu trả lời — giữ nguyên 90**, chờ anh chốt.

## Q11 — EXP dư
Xếp hạng giờ dùng Uy Danh (spec §11) nên EXP sau trần không còn tác dụng → **giữ nguyên code hiện tại** (kẹp tại trần, bỏ phần dư). Đã ghi rõ vào spec §3.2.

→ Việc cần làm: cập nhật config (`clash_unit: 1`, `flip_only_same_realm: true`, `min_count: 0`, `cell: 160`), sửa logic đổi chiều trong va chạm kiếm, sửa unit test theo checklist §17, chuyển Q7–Q9, Q11 sang mục đã trả lời trong QUESTIONS.md.

---

## v3.6 — Vật cản cứng hơn
Nguyên nhân vật cản yếu: damage tính theo **từng thanh kiếm**, nên 60–400 kiếm cùng chém thì vật cản tan ngay.
1. Đổi sang tính theo **người**: mỗi người gây damage lên 1 vật cản tối đa 1 lần / 0.5s, `damage = (r + 1) × (1 + 0.5 × (số_vòng − 1))`. Nhiều người cùng chém thì cộng dồn. (spec §5.3)
2. HP mới: rào tre 30, sạp hàng 80, tường gạch 250, đá tảng 800 (spec §8.2, có bảng thời gian phá theo cảnh giới).
3. Config: `obstacles.hp`, thêm `dmg_cooldown_secs: 0.5`, `dmg_per_extra_ring: 0.5`.
4. Unit test theo checklist §17 (Luyện Khí phá rào ≈ 7.5s, Đại Thừa phá đá ≈ 4s).

---

## v3.7 — Tiên giới (5 cảnh giới mới sau Đại Thừa)
Đọc spec §3.4 (mới). Tóm tắt việc cần làm:
1. Config: thêm `tien_realms`, `tien_kills_per_step`, `tien_bot_kill_value`, `tien_same_victim_cooldown_secs`, `tien_scale_step`, `tien_extra_swords`; mở rộng `max_swords` tới 600.
2. Server: thêm bộ đếm `tien_kills` (float vì bot = 0.5). Từ Đại Thừa tầng 10 chỉ lên bằng sát mạng (mỗi bước 10/20/30/40/50). 1 sát mạng = chém trúng thân thành công; cùng người trong 60s không tính. Tiên giới không chia tầng.
3. Bị chém ở Tiên giới: rớt 1 cảnh giới, `tien_kills` về mức khởi điểm của cảnh giới mới; Chân Tiên rớt về Đại Thừa tầng 10 (EXP giữ trần, `tien_kills` = 0).
4. Scale Tiên giới: 5.2 → 6.0 (+0.2 mỗi cảnh giới). Kiếm tối đa +40 mỗi cảnh giới.
5. Client: HUD thanh Sát mạng thay thanh EXP; hiệu ứng Thiên Kiếp + kill feed phi thăng; hào quang Đạo Tổ; Đạo Tổ hiện trên minimap cả phòng; bảng phòng xếp Tiên giới trên Đại Thừa.
6. Uy Danh, bể kiếm, sát thương vật cản dùng nguyên công thức với r tới 13.
7. Skin kiếm & VFX mới trong ART_PROMPTS §7, §10 (dùng placeholder trước).
8. Unit test theo checklist §17.

---

## v3.8 — Số kiếm theo vòng + Linh khí / Tiên khí
Đọc spec §3.1, §3.4, §4.1a, §8.2.
1. Vòng thứ k (1-index) chứa `10 × k` kiếm. Mỗi cảnh giới mở thêm 1 vòng: cảnh giới r có `r + 1` vòng, `max_swords = 5 × (r+1) × (r+2)` → [10, 30, 60, 100, 150, 210, 280, 360, 450, 550, 660, 780, 910, 1050]. Config: `ring_capacity_step: 10`, bỏ `swords_per_ring` và `tien_extra_swords`.
2. Lấp vòng trong trước; vòng đầy chia đều `36°/k`, vòng ngoài chưa đầy chia đều `360°/m`. So le: vòng k ≥ 2 lệch nửa khe của chính nó. Bán kính/chiều dài vòng vẫn chỉ phụ thuộc k và scale.
3. UI: dưới Tiên giới đổi mọi chữ "EXP" thành **Linh khí**; từ Chân Tiên trở lên thanh tiến độ là **Tiên khí** (đầy bằng sát mạng, màu vàng kim). Giữ nguyên thanh, chỉ đổi tên/màu.
4. Sửa unit test & checklist §17 theo số mới (Luyện Khí 10 kiếm 1 vòng, kiếm thứ 11 mở vòng 2...). Bảng thời gian phá vật cản §8.2 đã tính lại theo số vòng mới.

---

## v3.9 — Hào quang Tiên giới
Đọc spec §3.4 (mục "Hào quang sau lưng") và ART_PROMPTS §10.
1. Config: thêm `tien_aura_colors`.
2. Client: lớp hào quang vẽ sau lưng nhân vật (dưới sprite nhân vật, dưới vòng kiếm, trên mặt đất), tâm ngang ngực, đường kính ≈ 2.2 × chiều cao nhân vật, to theo scale, nhịp thở ±5% chu kỳ 2s. Màu theo cảnh giới Tiên; Đạo Tổ dùng sprite âm dương riêng.
3. Ẩn Thân/bụi: hào quang ẩn/mờ theo nhân vật. Rớt cảnh giới: đổi màu ngay, về Đại Thừa thì tắt, kèm hiệu ứng vỡ.
4. Chưa có art: placeholder là hình tròn phát sáng (radial gradient) theo màu config.

---

## Q16 — Trường Kiếm
**Bỏ hẳn** (phương án 2). Spec v4.3 đã xóa khỏi §6.2 và §4.1a, v4.4 thay bằng **Cương Kiếm** — anh dán spec mới nhất vào `docs/_raw/`. Làm theo CHANGES_v4.md mục 20–23. Ảnh `assets/buffs/long.png` để đó, không dùng.

## Q17 — Bó Kiếm khi rớt cảnh giới
**Chấp nhận phương án 1.** Spec §5.2 đã ghi rõ: Bó Kiếm chủ yếu từ xác người chết và vật phẩm bản đồ. Luật chém thân theo số vòng (Q15) đã được ghi vào spec thay cho lá chắn chặn đường (config `body_hit_max_rings: 1`, bỏ `shield_ring`).

## Q18 — Linh khí / Tiên khí tách hẳn
**Giữ đúng như em đã làm.** Spec §3.4 đã sửa: không quy đổi qua lại, người phàm chém người Tiên giới chỉ nhận Uy Danh (và ngược lại). Bỏ `tien_khi_to_exp_base` khỏi config.

## Q19 — Tốc độ bể kiếm
**Giữ như em đã làm**: `clash_cooldown: 0` (bể mỗi tick), `flip_cooldown: 0.3`. Spec §5.2 đã sửa.

---

## Trả lời báo cáo đo (mục 24)

Nhận xét: client trong game thật đang ổn (60 fps, vẽ 1 ms). Chỗ vượt ngân sách là **server tick** và **băng thông** (61,7 KB/s trong game thật, đỉnh snapshot 38 KB). Trên 4G yếu, gói to và đều đặn dễ gây nghẽn hàng đợi → ping hiển thị tăng. Vì vậy ưu tiên mạng và server trước client.

### 1. Tối ưu — đi tiếp, theo thứ tự này
1. **Mục 27 (mạng) trước**: delta snapshot, lượng tử hóa, item/EXP chỉ gửi sự kiện, **chia nhỏ ItemsAdded** (tối đa ~2 KB mỗi gói, phần còn lại gửi dần ở các tick sau, ưu tiên gần người chơi), không gửi viên nhỏ ngoài 1 500 khi camera lùi xa, 20/10 Hz theo khoảng cách, bỏ snapshot cũ khi client nghẽn.
2. **Mục 26 (server)**: chạy `pprof` trên loadtest trước, gửi tôi top 10 hàm tốn CPU, rồi mới sửa đúng chỗ (va chạm theo vòng, chỉ vòng ngoài cùng, không cấp phát mỗi tick).
3. **Mục 25 (client)**: chỉ làm phần rẻ mà lợi lớn — vẽ mỗi vòng kiếm thành 1 texture, BitmapText, cull. Web Worker và tự hạ chất lượng để sau, chỉ làm nếu đo thấy cần.
4. Đo lại bằng loadtest + F3 sau mỗi bước, ghi bảng trước/sau vào PROGRESS.md.
5. **Thêm 1 phép đo mạng thật**: chạy client với Chrome DevTools → Network throttling "Fast 4G" và "Slow 4G", ghi ping p95 và KB/s. Ghi luôn server đang đặt ở đâu (nước/khu vực) trong PROGRESS.md.

### 2. Thư viện
**Duyệt** `pixi-filters` và `revolt-fx`. Thêm vào CLAUDE.md quy tắc: *"Thư viện được spec ghi đích danh coi như đã duyệt; thư viện khác vẫn phải hỏi."*

### 3. Thứ tự các buổi tiếp theo
1. Tối ưu (mục 25–27 như trên).
2. Bảng tên — mục 37–41 (nhỏ, thấy ngay).
3. Ảnh đại diện + bản đồ sinh động — mục 42–47 (làm sau tối ưu để đo được chi phí bake chunk).
4. VFX Tiên giới — mục 29–33, rồi thần thú luôn lượn lờ — mục 34–36 (nếu chưa làm).

Mỗi buổi xong: test + build pass, cập nhật PROGRESS.md, báo số FPS/KB/s nếu có ảnh hưởng hiệu năng.

---

## Trả lời QUESTIONS (sau Q19)

### Tối ưu (CHANGES_v4 mục 25–27) — ĐI TIẾP
Đã đọc bảng đo. Đi tiếp theo thứ tự đã ghi ở mục "Trả lời báo cáo đo" phía trên trong file này (nếu `docs/_raw/ANSWERS.md` chưa có mục đó thì đây là bản đầy đủ):
1. Mục 27 (mạng) trước: chia nhỏ ItemsAdded ≤ 2 KB/gói ưu tiên gần, delta snapshot, lượng tử hoá, không gửi viên nhỏ ngoài 1 500 khi camera lùi xa, 20/10 Hz theo khoảng cách, bỏ snapshot cũ khi client nghẽn.
2. Mục 26 (server): pprof → sửa đúng chỗ (đã tìm ra lỗi SegSegDist2 — tốt).
3. Mục 25 (client): chỉ phần rẻ lợi lớn (vòng kiếm thành 1 texture, BitmapText, cull).
4. Đo lại loadtest + bài test 10 tab Chrome headless (thường + 4G yếu), ghi bảng trước/sau.

### Thư viện — DUYỆT
`pixi-filters` và `revolt-fx` được duyệt. Thêm vào CLAUDE.md: *"Thư viện được spec ghi đích danh coi như đã duyệt; thư viện khác vẫn phải hỏi."*

### Q20 — Va kiếm chậm lại sau khi sửa lỗi
**Chọn phương án 1: giữ nguyên luật đúng.** Lý do: tôi vừa phàn nàn "bị dí rất nhanh chết" — rất có thể chính là do lỗi này (kiếm cách cả trăm đơn vị vẫn tính là va). Không bù lại bằng cách đổi luật lúc này.
- Ghi số 84 lượt chết / 5 phút / 40 bot làm **mốc mới** trong PROGRESS.md (gạch các số đo cũ, ghi chú "đo trên luật va kiếm sai").
- Đo lại sau khi xong thế giới 5 khu + Bí Kíp. Nếu chơi thật thấy giao tranh quá lề mề, tôi sẽ quyết sau — ưu tiên phương án 3 (tăng `sword.width`) trước phương án 2.

### Q22 — `tien_overflow_score`
**0.1 là đúng** (10 Tiên khí = 1 Uy Danh). Giữ như em đang để. Spec v5.3 §3.4 và config đã sửa thành 0.1; checklist: nạp 500 Tiên khí → +50 Uy Danh.

---

## Trả lời Q20–Q25 (bản gộp — thay thế mọi bản cũ)

> Lưu ý cho em: `docs/_raw/GAME_SPEC.md` và `docs/_raw/ANSWERS.md` anh vừa thay bằng bản mới nhất (spec v5.6). Nhiều câu hỏi gần đây (Q20, Q22, mục 25–33, và Q24 nhắc "sát mạng") là do `_raw` cũ. Từ v4.0, Tiên giới lên bằng **Tiên khí** từ Đài Tiên Khí, không còn sát mạng.

**Tối ưu (CHANGES_v4 25–27): ĐI TIẾP.** Thứ tự: mạng (27) → server (26, pprof) → client phần rẻ (25.1 vòng kiếm thành texture, BitmapText, cull). Đo lại loadtest + 10 tab Chrome headless (thường + 4G yếu), ghi trước/sau.

**Thư viện: DUYỆT** `pixi-filters`, `revolt-fx`. Thêm vào CLAUDE.md: *"Thư viện spec ghi đích danh coi như đã duyệt; thư viện khác vẫn hỏi."*

**Q20: phương án 1** — giữ luật va kiếm đúng. Ghi 84 lượt chết/5 phút/40 bot làm mốc mới, đánh dấu số cũ "đo trên luật sai". Chơi thử sau 5 khu + Bí Kíp rồi mới quyết có chỉnh không (nếu chỉnh thì ưu tiên tăng `sword.width`).

**Q22: 0.1** (10 Tiên khí = 1 Uy Danh). Spec đã sửa khớp.

**Q23:** đã sửa trong spec v5.6 — §4.1c thêm "nháy đúp rồi giữ / chạm 2 lần rồi giữ", §18.1 thêm dòng Kỹ năng.

**Q24: phương án 2** — spec §4.1c (đã có từ v4.0, nay ghi rõ hơn): ở Tiên giới đốt **Tiên khí**, mỗi giây 8% Tiên khí của bước hiện tại, không xuống dưới mốc khởi điểm cảnh giới đang đứng. Config `player.burn_tien_frac_per_sec: 0.08`. Sửa `burn_tien_test.go` theo luật này.

**Q25: chưa chỉnh cân bằng** — đúng ý anh, để người chơi phản hồi. Giữ bản `--duel` mới (đủ trần cảnh giới) làm chuẩn đo. Ghi bảng vào PROGRESS.md để so sau mỗi lần chỉnh.

---

## Q25 (lần 2) — Bỏ cách đo tỉ lệ thắng 1-đấu-1
Kỹ năng có loại tăng sức mạnh, có loại khắc chế/gây khó chịu → tỉ lệ thắng tay đôi không phản ánh giá trị. **Bỏ** việc cân bằng theo bảng này: xóa `cmd/sim --duel` (hoặc giữ làm công cụ debug nhưng không báo cáo, không dùng để chỉnh số). Spec §6.3 đã sửa: cân bằng theo phản hồi người chơi + số liệu trận thật.

## Q26 — Đạo Tổ & Tiên khí: luật mới (spec v5.7 §3.4, §4.1c)
Thay toàn bộ cách tính Tiên giới:
1. **Vào Tiên giới**: chỉ khi Đại Thừa tầng 10 **đầy thanh** (`total_exp = 694 960`). Chưa đầy → vẫn nhặt Linh khí, không nhặt được Tiên khí (`BanksTienKhi` đổi điều kiện từ "Đại Thừa 10" sang "exp đầy trần").
2. **Cảnh giới Tiên = theo tổng Tiên khí cộng dồn**, trần `tien_khi_ceilings: [100, 300, 600, 1000, 1500]`: 1–100 Chân Tiên · 101–300 Kim Tiên · 301–600 Thái Ất · 601–1000 Đại La · 1001–1500 Đạo Tổ. **Viên Tiên khí đầu tiên → Chân Tiên ngay.** Bỏ `tien_khi_per_step`.
3. **Đạo Tổ**: nạp thanh từ 1 001 tới 1 500; **chỉ khi đang đúng 1 500** thì Tiên khí nhặt thêm mới đổi Uy Danh (10:1). Thiêu Đốt trừ Tiên khí bình thường (không dưới 1 001) — phải nạp lại 1 500 mới đổi tiếp. → Đạo Tổ Thiêu Đốt được (thay phương án tạm 1 của em).
4. Thiêu Đốt Tiên giới: 8% độ rộng khoảng của cảnh giới/giây (Chân Tiên 8, Kim Tiên 16, Thái Ất 24, Đại La 32, Đạo Tổ 40), không dưới đáy.
5. Bị chém: Tiên khí về **đáy cảnh giới mới** (1 / 101 / 301 / 601 / 1 001). Chân Tiên bị chém → Đại Thừa 10 thanh 0% (`total_exp = 650 928`), Tiên khí 0.
6. Sửa `TestDaoToHasNoTienKhiHeadroom` thành test luật mới; thêm test theo checklist §17.

---

## v5.8 — Bị chém: rớt 1 đại cảnh giới, thanh về mức sàn
Thay luật "rớt 1 cảnh giới, giữ nguyên tầng + % tiến độ" (§5.1 cũ):
- Bị chém (khi không chết) → `total_exp = sàn(cảnh giới mới)` = tầng 1, thanh 0%. Ví dụ Trúc Cơ tầng 2 → Luyện Khí tầng 1 0%; Kim Đan tầng 7 → Trúc Cơ tầng 1 0% (1 360).
- `loss = total_exp cũ − sàn mới` (người chém nhận 50%, rơi 30% như cũ).
- Tiên giới: Tiên khí về đáy cảnh giới mới (như v5.7). **Chân Tiên bị chém → sàn Đại Thừa** (tầng 1, 0%, `total_exp = 346 800`), Tiên khí 0 — thay cho "Đại Thừa 10 thanh 0%" ở v5.7.
- Sửa test & checklist §17 theo luật mới.

---

## v5.9 — LỖI: chém người Tiên giới không rơi Tiên khí
Anh thấy chém người cấp Tiên không có Tiên khí rơi ra. Spec §3.4 luôn yêu cầu rơi 30% — đây là lỗi, sửa:
1. Tái hiện trước: dùng `?debug=true` nạp cho bot 1 lên Kim Tiên (vd 250 Tiên khí), cho người chơi chém trúng — ghi lại có event rơi viên không, viên có được gửi xuống client không, client có vẽ không (viên Tiên khí có thể đang bị lọc bởi AOI/LOD "không gửi viên nhỏ khi camera lùi xa", hoặc bị vẽ mờ 30% rồi chìm vào nền). Báo tôi nguyên nhân.
2. Luật đúng (spec v5.9 §3.4): **luôn** rơi `floor(30% × loss_tiên_khí)` thành viên Tiên khí quanh nạn nhân, **bất kể người chém là phàm/Tiên/bot**, kể cả Chân Tiên rớt về Đại Thừa (loss = toàn bộ Tiên khí). Chia viên 10 → 2 → 1, văng bán kính 200–500, tồn tại 60s, có hiệu ứng nổ vàng + SFX.
3. Viên Tiên khí **không bị loại bởi LOD/AOI "viên nhỏ"** (mục 27) — Tiên khí luôn được gửi khi trong khung nhìn.
4. Test: Chân Tiên 80 bị chém → rơi tổng 24; Thái Ất 450 → Kim Tiên, rơi 104; người chém phàm vẫn làm rơi đủ.

---

## Trả lời báo cáo tối ưu 25–27 & NETCODE_PLAN

**Tối ưu 26–27: ĐẠT, chấp nhận.** Ghi vào PERF_LOG (lượt mới, kèm bảng mổ gói theo trường: 82% ItemPicked). Luật va kiếm "từ vòng ngoài vào cho tới khi đủ 1 vòng đầy" — **chấp nhận**, đã ghi vào GAME_SPEC v6.2 §13.4. Cảm ơn vì không sửa test cho qua.

**NETCODE — duyệt đổi thứ tự (NETCODE_SPEC v1.1 §8):**
1. Làm **chống nhấp nháy & placeholder trước** bằng bản vá rẻ: client chỉ hủy entity khi có `Despawn` hoặc im lặng > 2s; thêm `DescRequest` khi thiếu descriptor; không bao giờ vẽ placeholder ở build thật. Đo với nhiều người + bot quanh mép khung nhìn. Gửi tôi số "biến mất rồi quay lại trong 2s" và "% nhân vật vẽ placeholder" trước/sau.
2. Bước "tách khu" thu nhỏ thành **Room Shared Cache** + kiểm chứng log gói.

**3 điểm em nêu:**
- Bước 1 nhỏ hơn spec tưởng → đồng ý, đã gộp như trên.
- Trần 96 MB quá nhiều cho điện thoại → spec v1.1: PC Cao 96 / PC Vừa 64 / điện thoại 24 (Thấp 16) MB; texture vòng điện thoại tối đa 1024 px; đầy trần thì hạ độ phân giải trước khi loại bỏ.
- Bẫy Service Worker → spec v1.1 §5 L3 thêm luật an toàn: chỉ cache file có hash; index/manifest/JS chính network-first; không skipWaiting giữa trận; server từ chối build cũ; công tắc `/sw-kill`; test build A → B không đứng hình. Bước 7 chỉ làm sau khi qua test này.

**4 điểm còn lại trong NETCODE_PLAN** (anh chưa dán cho tôi): ghi rõ phương án em đề xuất cho từng điểm vào QUESTIONS.md, tôi sẽ trả lời. Trong lúc chờ, làm bước 1 (không phụ thuộc 4 điểm đó).

---

## Trả lời NETCODE_PLAN (c) — đủ 7 điểm

1. Bước 1 nhỏ hơn → **đồng ý**, đã thu thành Room Shared Cache (spec v1.1 §8 bước 2).
2. Lượng tử hóa vị trí → **đồng ý theo hướng em**: người khác/quái/item dùng `u16` thang 0,18 (đúng spec); **vị trí của chính mình** trong `SelfState` giữ chính xác cao (fixed-point 1/256 hoặc float32). Đổi ở bước 3, đo bộ đếm hiệu chỉnh phải ≈ 0. (spec v1.2 §4.6)
3. Ngân sách 6 KB → **tạm chấp nhận < 8 KB** cho tới hết bước 3; sau Protocol v2 phải về < 6 KB. (spec v1.2 §1)
4. Trần texture → đã sửa spec v1.1: PC 96/64, điện thoại 24 (Thấp 16) MB.
5. Service Worker → đã sửa spec v1.1 §5 L3 (manifest/index network-first, giữ cổng chặn version, `/sw-kill`, test build A→B).
6. Bài đối chiếu trước khi xóa v1 → **đồng ý và nâng lên bắt buộc trước khi bật v2 cho bất kỳ ai**. Thêm: dùng **1 file schema** + generator sinh cả mã hóa Go lẫn giải mã TS (không viết tay DataView hai phía), golden vectors + fuzz 10 000 gói trong CI, `schema_hash` trong `Hello` lệch thì từ chối. (spec v1.2 §4.9)
7. AOI giữ 2s → **đồng ý**: bản rẻ ở client trước, bước 3 thay bằng `Despawn` tường minh. (spec v1.2 §8)

Kế hoạch (b) **duyệt**. Bắt đầu Bước 2a → 2b ngay. Bài đo `flicker.mjs` phải có bot đi qua lại mép khung nhìn 30s (như em đã lưu ý), ghi trước/sau vào PERF_LOG.

---

## Trả lời bảng điểm độ giật (CHANGES_v6_1 mục 5)

1. **Ngưỡng 45: giữ tạm.** Tôi sẽ tự kiểm bằng phím `J` và báo lại nếu cần chỉnh. Để dễ kiểm, thêm vào chế độ `J`: phím `[` / `]` chuyển qua lần lượt từng animation trong bảng (xếp theo điểm), phát lặp chậm 0.5× ngay giữa màn hình, kèm điểm — tôi xem nhanh top 10 và vài con điểm ~30 để chốt ngưỡng.
2. **Đồng ý tách 2 nhóm** (anim lặp đọc TB; anim một lần đọc "lạc lõng"). Spec GAME_SPEC v6.3 §12.5 B.5 đã ghi lại.
3. **Đảo thứ tự: B → C → A.**
   - **B trước** (chuẩn hóa frame trong `tools/atlas-pack`): chạy cho toàn bộ art, rồi **chấm lại** bảng điểm. Gửi tôi bảng trước/sau + GIF trước/sau của co_ba_ganh `idle` và dep_lao `idle`.
   - Anim nào **sau B vẫn đỏ** → đó là do bản thân frame vẽ khác dáng, không chuẩn hóa được: đưa vào danh sách sinh lại (ART_JOBS), ghi rõ frame nào. Dự đoán: co_ba_ganh `idle`, a_hon `teleport_in_01`, ong_ba_vong `realm_drop_05`.
   - Rồi **C** (bộ điều khiển: crossfade, giữ pha, fps theo tốc độ, lật có trễ, chuyển động phụ).
   - **A** (một thân nhiều mặt) để sau cùng vì cần art mới.

---

## Trả lời báo cáo B + C (build 9cff870)

**B + C: duyệt.** Ba chỗ em tự chặn so với spec (trần ±15% co giãn, chấm cả 2 bản neo chân/không neo rồi giữ bản tốt hơn, phép kiểm kích thước mảnh khi cắt) — **chấp nhận**, ghi chú vào GAME_SPEC §12.5 B ở lần cập nhật spec tới. `check:items` tốt — thêm vào CI cùng `check:assets`.

**1. dep_lao `idle` (56.8):** tôi sẽ tự xem bằng `J` + `]`. Trong lúc chờ, cho phép **đánh dấu ngoại lệ có chủ đích** trong `assets/jitter_exceptions.json` (tên anim + lý do + người duyệt) — anim có trong danh sách không tính đỏ nhưng vẫn hiện điểm. Chưa thêm dep_lao vào cho tới khi tôi xác nhận.

**2. Bó Kiếm (`item_bundle`): ĐƯA VÀO, giữ đúng prompt em đã đặt** (3 kiếm bắt chéo buộc vải đỏ, cùng canvas với `item_sword`). ART_PROMPTS §8 đã ghi lại cho khớp. Sinh ảnh luôn. Trong game vẽ to hơn bong bóng kiếm thường 20%; bó rơi từ xác in số kiếm lên bong bóng. `sword` và `tien_khi` giữ vẽ tay như em đề xuất.

**3. Nhờ push lên GitHub** mọi commit đã deploy (`9cff870` chưa có trên `main` — tôi chỉ thấy tới `a8a9cc8`), kèm `docs/anim_preview/`, để bên Claude chat đọc được onion-skin và GIF.

**Tiếp theo:** netcode **bước 2b** (cấm placeholder build thật + đóng nút thắt B3), song song telemetry (hướng C) nếu chưa xong. **A** (một thân nhiều mặt) để sau, như đã chốt.


> Ghi chú: `docs/QUESTIONS.md` bản tôi đọc còn ghi "việc lớn tiếp theo: tối ưu 25–27" — phần đó đã xong (mốc #4). Cập nhật lại QUESTIONS.md cho đúng hiện trạng; câu Bó Kiếm chuyển sang QUESTIONS_DONE.
