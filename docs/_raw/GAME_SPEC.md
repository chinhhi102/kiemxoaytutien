# KIẾM XOAY TU TIÊN — Game Design & Technical Spec (v2)

> Tên tạm (working title). File này dành cho Claude Code: đặt tại `docs/GAME_SPEC.md`.
> Mọi con số nằm trong `config/game.json` (mục 16) — **không hardcode**.
> Art/animation prompts: `docs/ART_PROMPTS.md`. Hướng dẫn setup: `docs/SETUP_GUIDE.md`.

**Thay đổi v6.1:** tên bot giống người thật (1 000 tên trong config/bot_names.txt, luật chọn tên & hành vi khi chết, §9.4); hoạt ảnh liền mạch — một thân nhiều khuôn mặt, chuẩn hóa frame tự động, bộ điều khiển anim mượt (§12.5).

**Thay đổi v6.0:** tối ưu mạng/hiệu năng chuyển sang tài liệu riêng `NETCODE_SPEC.md` — đồng bộ theo khu, Room Shared Cache, Protocol v2, cache nhiều lớp, chống nhấp nháy/placeholder (§13.3).

**Thay đổi v5.9:** làm rõ: chém người Tiên giới luôn rơi 30% Tiên khí mất ra đất, bất kể người chém là ai (§3.4).

**Thay đổi v5.8:** bị chém rớt 1 đại cảnh giới và thanh về **mức sàn** (tầng 1, 0%) của cảnh giới mới, không giữ tầng nữa (§5.1, §3.4).

**Thay đổi v5.7:** Tiên giới tính theo tổng Tiên khí (Chân Tiên 1–100 … Đạo Tổ 1 001–1 500), vào Tiên giới cần Đại Thừa 10 đầy thanh; Đạo Tổ nạp đầy 1 500 mới đổi Uy Danh, Thiêu Đốt được; bỏ cân bằng theo tỉ lệ thắng 1-đấu-1 (§3.4, §4.1c, §6.3).

**Thay đổi v5.6:** Thiêu Đốt bật thêm bằng nháy đúp giữ / chạm 2 lần giữ; Tiên giới đốt Tiên khí (§4.1c); bảng hướng dẫn có dòng Kỹ năng (§18.1).

**Thay đổi v5.5:** chỉnh kỹ năng — A Hớn hồi 20% kiếm tối đa; Bà Tám đẩy 500; Ông Ba Võng bán kính 500; Thằng Mập bịt mắt 5s + không nhặt đồ; Cô Ba trượt = choáng 1s; Shipper ×3 trong 5s, xuyên mọi vật cản trừ biên (§6.3).

**Thay đổi v5.4:** chỉnh kỹ năng — Gọi Đàn Em 10s, mỗi đàn em 50% số kiếm của đại ca; Lưới Mì trói 5s; Gồng Cơ 5s; Rụt Cổ 5s (§6.3).

**Thay đổi v5.3:** Tiên khí dư của Đạo Tổ đổi Uy Danh 10:1; va kiếm giữ nhịp đúng sau khi sửa lỗi SegSegDist2 (§3.4, §5.2).

**Thay đổi v5.2:** thêm Thiên Môn ở tâm mỗi khu (trừ Thiên Đình) — cổng một chiều đích cố định tới Thiên Đình (§8.4a).

**Thay đổi v5.1:** giữ 20 người/phòng (bot lấp theo phòng, sinh vào khu ít người nhất); kỹ năng chỉ dùng được khi nhặt vật phẩm Bí Kíp (tối đa 2 lượt); Thằng Mập đổi thành Phun Trân Châu bịt mắt (§6.3, §9).

**Thay đổi v5.0:** thế giới 5 khu vực tách biệt (mỗi khu = bản đồ cũ, thiết kế riêng), chỉ đi qua bằng cổng dịch chuyển đích ngẫu nhiên; sửa giật khi chạm vật cản (dùng chung mã va chạm, trượt dọc, gộp collider, hiệu chỉnh mượt); kỹ năng riêng cho 10 nhân vật; Đạo Tổ Tiên khí dư → Uy Danh (§3.4, §6.3, §8.0, §8.2a, §8.4, §9).

**Thay đổi v4.9:** ảnh đại diện nhân vật ở HUD thay số cấp (§12.4); bản đồ sinh động — trộn texture theo nhiễu, vùng hình hữu cơ (server dùng chung đa giác), decal, nước, ambient; cấm khối chữ nhật/tròn lộ ra (§8.6).

**Thay đổi v4.8:** thiết kế lại bảng tên trên đầu: ruy băng danh hiệu theo hạng, huy hiệu cảnh giới màu theo cảnh giới + chỉ báo nguy hiểm, tên to có viền, nền chung, kích thước cố định trên màn hình (§12.3).

**Thay đổi v4.7:** thần thú Tiên giới luôn lượn lờ quanh người (quỹ đạo nền + động tác ngẫu nhiên + bám theo có độ trễ) thay cho bay 1 vòng mỗi 20s (§3.5).

**Thay đổi v4.6:** bộ công cụ VFX — blend cộng, hào quang nhiều lớp, bloom cho lớp VFX, pixi-filters, particle editor (revolt-fx), texture CC0 (§3.6).

**Thay đổi v4.5:** Linh khí / Tiên khí tách hẳn, không quy đổi; bỏ lá chắn chặn đường, chém thân chỉ khi còn 1 vòng; bể kiếm tính mỗi tick, đổi chiều cooldown 0.3s (§3.4, §5.2). Xác nhận đã bỏ Trường Kiếm (v4.3).

**Thay đổi v4.4:** thêm vật phẩm Cương Kiếm (độ cứng: bền ×1.5 khi va kiếm, sát thương quái/vật cản ×2); mục tối ưu hiệu năng bắt buộc §13.6 (vẽ mỗi vòng kiếm thành 1 texture, delta snapshot, LOD, đo ping đúng).

**Thay đổi v4.3:** bỏ vật phẩm Trường Kiếm; tầm chém chỉ tăng theo số vòng và cảnh giới (§4.1a, §6.2).

**Thay đổi v4.2:** VFX thần thú bay lên từ chân cho cảnh giới Tiên (§3.5).

**Thay đổi v4.1:** độ bền kiếm mỗi cảnh giới +20% (Trúc Cơ vs Luyện Khí đổi 10 : 12), thay cho tỉ lệ 1 : (d+1) (§5.2).

**Thay đổi v4.0:** gộp toàn bộ thay đổi đã nhắn trực tiếp cho agent (tốc chạy, bất tử 5s, realm_mult 2.0, Thiêu Đốt Tinh Huyết, bảng phòng theo Uy Danh, danh hiệu Top 1–10, tin loan toàn server, bot ngang hạng 11, debug, thanh máu quái); không đẩy văng khi va kiếm + lá chắn vòng kế ngoài cùng; Tiên khí từ Đài Tiên Khí thay cho sát mạng; thuật toán sinh bản đồ tu tiên; sửa lỗi Tốc Kiếm.

**Thay đổi v3.9:** hào quang sau lưng cho các cảnh giới Tiên, mỗi cảnh giới một màu (§3.4).

**Thay đổi v3.8:** số kiếm theo vòng — vòng k chứa 10k kiếm, mỗi cảnh giới thêm 1 vòng (Luyện Khí 10 → Đạo Tổ 1 050); thanh tiến độ gọi là Linh khí / Tiên khí (§3.1, §3.4, §4.1a, §8.2).

**Thay đổi v3.7:** thêm 5 cảnh giới Tiên giới (Chân Tiên → Đạo Tổ) lên bằng sát mạng, không dùng EXP (§3.4).

**Thay đổi v3.6:** vật cản cứng hơn — damage tính theo người (0.5s/lần) thay vì theo từng kiếm, HP tăng (§5.3, §8.2).

**Thay đổi v3.5:** clash_unit = 1; chỉ đổi chiều quay khi cùng cảnh giới; số kiếm có thể về 0 (không chết); nhịp Luyện Khí → Trúc Cơ 30–60s (§5.2, §6.1).

**Thay đổi v3.4:** thêm minimap (§12.1) và danh hiệu Top 1–3 phòng (§11.3); kiếm quay chậm lại còn 1 vòng/giây, Tốc Kiếm trả về ×1.6 (§4.1b).

**Thay đổi v3.3:** bảng xếp hạng dùng điểm Uy Danh thay EXP, thêm bảng Sát Thần (lần chém trúng) và Thần Tốc (lên Đại Thừa nhanh nhất); Ẩn Thân giảm còn 5s, tàng hình 80% (§6.2, §11).

**Thay đổi v3.2:** bán kính vòng kiếm cố định, không phụ thuộc số kiếm; chỉ vòng mới mới rộng hơn (§4.1a).

**Thay đổi v3.1:** vòng kiếm vẽ kiểu cánh quạt như quảng cáo — so le góc, lưỡi nghiêng 35°, mỗi vòng một màu, vòng ngoài lùi ra xa hơn (§4.1a).

**Thay đổi v3:** kiếm dài gấp đôi, quay 300ms/vòng, kiếm tối đa 60→400 xếp vòng lồng nhau (20 kiếm/vòng), bị chém rớt 1 đại cảnh giới, EXP rải khắp bản đồ, vật phẩm ×3, vật cản chỉ chặn ở chân đế + y-sort.

**Thay đổi v2 so với v1:** đổi stack (Go server + TypeScript/PixiJS client trên web) · kiếm có thể bị bể khi va nhau (tỉ lệ theo chênh lệch cảnh giới) · scale 1 → 5 và tầm kiếm dài theo · chết ra màn hình chờ rồi match lại · bản đồ có địa hình, vật cản phá được (mọc lại sau 30s), cổng dịch chuyển · không login, nhập tên là chơi, có leaderboard · bộ vật phẩm buff có thời hạn.

---

## 1. Tổng quan

- Thể loại: multiplayer `.io` top-down 2D chạy trên **trình duyệt** (PC + điện thoại), vào là chơi, không cài đặt, không đăng nhập.
- Fantasy: dân phố tấu hài tu tiên bằng cách cho kiếm bay vòng quanh người.
- Core loop: **di chuyển → chém quái/vật cản → lượm EXP & vật phẩm → lên cảnh giới (to ra, kiếm dài và nhiều hơn) → chém người khác → bị chém thì rớt tầng, mất kiếm → tận dụng địa hình để lật kèo**.
- Điều khiển: WASD / phím mũi tên / giữ chuột trên PC; joystick ảo trên điện thoại. Không có nút tấn công.

## 2. Luồng màn hình

```
[Trang chủ] nhập tên + chọn nhân vật + nút "Vào chơi" + nút "Bảng xếp hạng"
      │
      ▼
[Đang tìm phòng...] → [Trong trận] ──chết──▶ [Màn hình chờ / Tổng kết]
                                                 ├─ "Chơi tiếp"  → matchmaking lại (có thể sang phòng khác)
                                                 ├─ "Bảng xếp hạng"
                                                 └─ "Đổi nhân vật" → Trang chủ
```

- Tên 2–16 ký tự, cho phép tiếng Việt có dấu, lọc từ tục, cho trùng tên.
- Tên & nhân vật đã chọn được nhớ trên máy (localStorage) cho lần sau.

## 3. Hệ thống tu luyện

### 3.1 Cảnh giới, kiếm tối đa, kích thước

`level_index L = r*10 + s` (r = cảnh giới 0..8, s = tầng 0..9) → 90 level.

```
scale(L) = 1 + 4 * L / 89          // 1.00 (Luyện Khí 1) → 5.00 (Đại Thừa 10), to ra ở MỖI tầng
```

**Mỗi cảnh giới mở thêm 1 vòng kiếm.** Vòng thứ k chứa `10 × k` kiếm (vòng 1: 10, vòng 2: 20, ... vòng 10: 100).
→ Cảnh giới r (0-index) có `r + 1` vòng, kiếm tối đa `= 5 × (r+1) × (r+2)`.

| r | Cảnh giới | Số vòng | Kiếm tối đa | Scale tầng 1 → tầng 10 |
|---|---|---|---|---|
| 0 | Luyện Khí | 1 | 10 | 1.00 → 1.40 |
| 1 | Trúc Cơ | 2 | 30 | 1.45 → 1.85 |
| 2 | Kim Đan | 3 | 60 | 1.90 → 2.30 |
| 3 | Nguyên Anh | 4 | 100 | 2.35 → 2.75 |
| 4 | Hóa Thần | 5 | 150 | 2.80 → 3.20 |
| 5 | Luyện Hư | 6 | 210 | 3.25 → 3.65 |
| 6 | Hợp Thể | 7 | 280 | 3.70 → 4.10 |
| 7 | Độ Kiếp | 8 | 360 | 4.15 → 4.55 |
| 8 | Đại Thừa | 9 | 450 | 4.60 → 5.00 |

Tiên giới tiếp tục quy tắc này (§3.4). Config: `ring_capacity_step = 10`; mảng `max_swords` tính sẵn từ công thức.

### 3.2 EXP (hiển thị cho người chơi: **Linh khí**)

```
exp_needed(r, s) = round(100 * 2^r * (1 + 0.08 * s))      // realm_mult = 2.0: mỗi cảnh giới tốn gấp đôi cảnh giới trước
```

| Cảnh giới | EXP/tầng (1 → 10) | Tổng cả cảnh giới | Cộng dồn |
|---|---|---|---|
| Luyện Khí | 100 → 172 | 1 360 | 1 360 |
| Trúc Cơ | 200 → 344 | 2 720 | 4 080 |
| Kim Đan | 400 → 688 | 5 440 | 9 520 |
| Nguyên Anh | 800 → 1 376 | 10 880 | 20 400 |
| Hóa Thần | 1 600 → 2 752 | 21 760 | 42 160 |
| Luyện Hư | 3 200 → 5 504 | 43 520 | 85 680 |
| Hợp Thể | 6 400 → 11 008 | 87 040 | 172 720 |
| Độ Kiếp | 12 800 → 22 016 | 174 080 | 346 800 |
| Đại Thừa | 25 600 → 44 032 | 348 160 | 694 960 |

**Triết lý tiến độ**: farm chỉ đủ để qua các cảnh giới đầu. Từ Kim Đan trở lên, đường lên chính là **cướp đoạt Linh khí của người khác** (§5.1: chém trúng 1 người = nhận 50% lượng EXP họ mất, mà lượng mất là nguyên 1 cảnh giới). Chém 1 người cao hơn mình 1–2 cảnh giới có thể nhảy vọt nhiều cảnh giới. Farm thuần lên Đại Thừa gần như không thể (nhiều giờ) — đó là chủ ý.

- Lưu `total_exp` cộng dồn, level suy ra từ bảng ngưỡng tích lũy tính sẵn.
- **Tên hiển thị**: dưới Tiên giới, mọi chữ "EXP" người chơi nhìn thấy đổi thành **Linh khí** (thanh Linh khí, viên Linh khí, "+20 Linh khí"). Trong code vẫn giữ tên `exp`.
- Đại Thừa tầng 10 **đầy thanh** (`total_exp = 694 960`, trần): Linh khí nhặt thêm bị bỏ. Lên tiếp phải hấp thụ **Tiên khí** (§3.4). Đại Thừa tầng 10 chưa đầy thanh thì vẫn nhặt Linh khí như thường.
- Lên tầng: glow nhỏ. **Đột phá cảnh giới**: anim `breakthrough`, sét, camera rung, đổi skin kiếm theo cảnh giới, nhân vật quanh đó hiện emote kinh ngạc.

### 3.4 Tiên giới — lên cảnh giới bằng Tiên khí

Sau Đại Thừa có thêm 5 cảnh giới. **Không dùng Linh khí (EXP)**, chỉ lên được bằng **Tiên khí** — loại viên đặc biệt chỉ phun ra từ các **Đài Tiên Khí** trên bản đồ (§8.5). Tiên giới **không chia tầng**. Cảnh giới Tiên được xác định thẳng từ **tổng Tiên khí đang có** (`tien_khi`, cộng dồn):

| r | Cảnh giới | Tiên khí (cộng dồn) | Trần | Kiếm tối đa | Số vòng | Scale |
|---|---|---|---|---|---|---|
| 9 | Chân Tiên | 1 – 100 | 100 | 550 | 10 | 5.2 |
| 10 | Kim Tiên | 101 – 300 | 300 | 660 | 11 | 5.4 |
| 11 | Thái Ất Ngọc Tiên | 301 – 600 | 600 | 780 | 12 | 5.6 |
| 12 | Đại La Kim Tiên | 601 – 1 000 | 1 000 | 910 | 13 | 5.8 |
| 13 | Đạo Tổ | 1 001 – 1 500 | 1 500 | 1 050 | 14 | 6.0 |

- **Điều kiện vào Tiên giới**: Đại Thừa tầng 10 **đầy thanh Linh khí** (694 960). Chỉ khi đó mới nhặt được Tiên khí; **viên Tiên khí đầu tiên (tien_khi = 1) → Chân Tiên ngay**.
- Lên cảnh giới khi Tiên khí **vượt trần** cảnh giới hiện tại: 100 → 101 là Kim Tiên, 300 → 301 là Thái Ất, 600 → 601 là Đại La, **1 000 → 1 001 là Đạo Tổ**.
- **Mốc khởi điểm** mỗi cảnh giới (đáy): Chân Tiên 1 · Kim Tiên 101 · Thái Ất 301 · Đại La 601 · Đạo Tổ 1 001.
- Config: `tien_khi_ceilings: [100, 300, 600, 1000, 1500]`.

**Ai nhặt được Tiên khí**
**Linh khí và Tiên khí tách hẳn nhau** (ranh giới: Đại Thừa tầng 10 đầy thanh):
- Từ **Đại Thừa tầng 10 đầy thanh** trở lên: chỉ hấp thụ **Tiên khí**. Viên Linh khí đi xuyên qua, không nhặt được.
- Dưới mức đó (kể cả Đại Thừa tầng 10 chưa đầy): chỉ hấp thụ **Linh khí**. Viên Tiên khí đi xuyên qua, không nhặt được.
- Viên không hấp thụ được vẫn nằm đó cho người dùng được tới nhặt; client vẽ mờ 30% với người không nhặt được để họ không chạy theo vô ích.
- Không có quy đổi qua lại giữa hai loại.

**Bị chém khi đang ở Tiên giới (cướp đoạt Tiên khí)**
- Rớt 1 cảnh giới như thường (§5.1): Đạo Tổ → Đại La Kim Tiên ... Kim Tiên → Chân Tiên.
- Tiên khí **về mốc khởi điểm của cảnh giới mới**: `loss = tien_khi − đáy_mới`. Ví dụ Thái Ất đang 450 bị chém → Kim Tiên, Tiên khí về 101, `loss = 349`.
- **Chân Tiên bị chém** → về **sàn Đại Thừa** (tầng 1, 0%, `total_exp = 346 800`; phải nạp lại 348 160 Linh khí mới đầy Đại Thừa 10), Tiên khí về 0. Phần Linh khí mất tính theo §5.1, phần Tiên khí mất = toàn bộ Tiên khí đang có.
- Người chém (nếu đang hấp thụ được Tiên khí) nhận **50% loss** Tiên khí.
- **Luôn rơi 30% loss thành viên Tiên khí** quanh nạn nhân — **bất kể người chém là ai** (phàm, Tiên, bot) và bất kể nạn nhân rớt về đâu (kể cả Chân Tiên rớt về Đại Thừa). Đây là nguồn Tiên khí quan trọng ngoài Đài Tiên Khí, phải luôn xảy ra.
  - Chia thành viên: viên lớn 10 điểm trước, phần lẻ thành viên 2 điểm và 1 điểm; văng ra bán kính 200–500 quanh xác (như đài phun), tồn tại 60s như viên từ đài.
  - Kèm hiệu ứng "nổ Tiên khí" vàng kim + SFX riêng để cả khu thấy có Tiên khí rơi.
  - Viên rơi ra tuân luật hấp thụ thường: chỉ người đã vào Tiên giới (hoặc Đại Thừa 10 đầy thanh) nhặt được; người khác thấy mờ 30%.
- Người chém chưa hấp thụ được Tiên khí: **không nhận Tiên khí**, phần thưởng là Uy Danh (§11.1 — chém trúng Đạo Tổ = 1 400 × lật kèo). Tương tự, người Tiên giới chém người phàm không nhận Linh khí, chỉ nhận Uy Danh; phần rơi ra vẫn rơi bình thường.
- Kiếm thừa so với max mới rơi ra thành Bó Kiếm như thường.

**Đạo Tổ: nạp đầy rồi mới đổi Uy Danh**
- Đạo Tổ vẫn nhặt Tiên khí bình thường để nạp thanh từ 1 001 lên **trần 1 500**.
- **Chỉ khi thanh đang ở đúng 1 500** thì Tiên khí nhặt thêm mới đổi sang Uy Danh theo tỉ lệ **10 Tiên khí = 1 Uy Danh** (`tien_overflow_score` = 0.1). Thanh không vượt 1 500.
- Đạo Tổ dùng Thiêu Đốt → Tiên khí bị trừ như thường (từ 1 500 trở xuống, không dưới 1 001). Muốn đổi Uy Danh tiếp thì phải **nạp lại đầy 1 500** trước.

**Hiển thị & tấu hài**
- HUD: **giữ nguyên thanh tiến độ**, chỉ đổi tên: dưới Tiên giới gọi là **Linh khí**, từ Chân Tiên trở lên gọi là **Tiên khí** (màu vàng kim), thanh chạy từ đáy tới trần cảnh giới. Ví dụ: `Kim Tiên — Tiên khí 150/300`. Đạo Tổ đầy 1 500: thanh nhấp nháy vàng + chữ "Tiên khí dư → Uy Danh".
- Đột phá Tiên giới: hiệu ứng **Thiên Kiếp** (mây đen, sét đánh 3 lần liên tiếp, màn hình rung), kill feed toàn phòng: `"{A} vượt Thiên Kiếp, phi thăng {realm}!"`.
- **Hào quang sau lưng** (mọi cảnh giới Tiên, mỗi cảnh giới một màu):

  | Cảnh giới | Màu hào quang | Hiệu ứng thêm |
  |---|---|---|
  | Chân Tiên | Trắng bạc `#E8F1FF` | Vài đốm mây nhỏ trôi quanh |
  | Kim Tiên | Vàng kim `#FFD60A` | Tia sáng tỏa ra |
  | Thái Ất Ngọc Tiên | Xanh ngọc `#4CE0B3` | Hạt sáng bay lên |
  | Đại La Kim Tiên | Tím `#B15CFF` | Hai vòng sáng xoay ngược chiều nhau |
  | Đạo Tổ | Đen trắng âm dương + viền cầu vồng | Vòng âm dương xoay chậm, viền đổi màu liên tục |

  - Vẽ **sau lưng nhân vật** (dưới sprite nhân vật, trên mặt đất), tâm ở ngang ngực, đường kính ≈ 2.2 × chiều cao nhân vật, to theo `scale`. Dưới vòng kiếm.
  - Nhịp thở: phóng to/thu nhỏ ±5% và sáng/tối nhẹ theo chu kỳ 2s.
  - Ai cũng thấy. Ẩn Thân → mờ theo nhân vật (20%). Trong bụi → ẩn theo nhân vật.
  - Rớt cảnh giới → hào quang đổi màu ngay (về Đại Thừa thì tắt), kèm hiệu ứng vỡ vụn.
  - Màu nằm trong config `tien_aura_colors`.
- Đạo Tổ hiện trên minimap của cả phòng (giống Top 1) → cả phòng hợp sức săn.
- **Thần thú bay lên từ chân** (VFX riêng mỗi cảnh giới Tiên, xem §3.5).
- Uy Danh (§11.1) dùng nguyên công thức với `r` tới 13 (chém trúng Đạo Tổ = 1 400 × lật kèo). Đột phá Tiên giới vẫn `20 × r`.

**Lưu ý cân bằng**: Đạo Tổ có 14 vòng / 1 050 kiếm, tầm chém ≈ 1 900 đơn vị. Nếu thấy quá lớn, giảm `tien_scale_step` trong config.

### 3.5 VFX Thần thú Tiên giới

Mỗi cảnh giới Tiên có một thần thú, màu khớp với hào quang:

| Cảnh giới | Thần thú | Màu | Vệt theo sau |
|---|---|---|---|
| Chân Tiên | Tiên hạc | Trắng bạc | Mây trắng nhỏ |
| Kim Tiên | Kim long (rồng vàng) | Vàng kim | Tia lửa vàng |
| Thái Ất Ngọc Tiên | Ngọc long (rồng ngọc) | Xanh ngọc | Hạt sáng xanh |
| Đại La Kim Tiên | Tử phượng (phượng hoàng tím) | Tím | Lông vũ sáng |
| Đạo Tổ | **Song long âm dương** (1 trắng, 1 đen, quấn nhau) | Đen trắng + viền cầu vồng | Mây âm dương |

**Khi nào xuất hiện**
1. **Đột phá vào cảnh giới Tiên** (đi cùng Thiên Kiếp): thần thú to, chui lên từ dưới chân, **xoắn ốc quanh người 1.5 vòng** đi lên, bay vút lên trời rồi tan thành vệt sáng. Dài 2.5s. Camera rung nhẹ, mọi người trong tầm nhìn đều thấy.
2. **Luôn lượn lờ quanh người** (thường trực, liên tục, không tắt): sau màn đột phá, thần thú thu nhỏ còn 60% và **ở lại bay quanh chủ nhân suốt thời gian còn ở Tiên giới**:
   - **Quỹ đạo nền**: vòng elip quanh thân (bán kính 1.3–1.8 × chiều cao nhân vật, ép dẹt 0.35 theo góc nhìn), tốc độ chậm ~1 vòng / 4s; độ cao nhấp nhô lên xuống nhẹ (sóng sin chu kỳ 3s) → lúc vòng sau lưng, lúc lượn ra trước mặt.
   - **Đi theo chủ**: tâm quỹ đạo bám theo nhân vật có độ trễ (lerp 0.1/frame) → khi chạy, thần thú bị kéo dài ra phía sau như dải lụa rồi đuổi kịp.
   - **Động tác ngẫu nhiên** mỗi 8–15s, chọn 1: xoắn ốc vút lên rồi bổ nhào xuống · lượn hình số 8 quanh người · đổi chiều bay · lao ra xa 1 nhịp rồi vòng về · cuộn tròn quanh đầu chủ. Mỗi động tác 1.5–3s rồi về quỹ đạo nền (chuyển mượt, không giật).
   - **Phản ứng**: chủ bị chém trúng khi có khiên / nhặt Tiên khí → thần thú lóe sáng và lượn nhanh 1 vòng; chủ Thiêu Đốt Tinh Huyết → bay sát người, vệt đỏ.
   - Chiều sâu như cũ: nửa sau quỹ đạo vẽ dưới nhân vật, nửa trước vẽ trên.
   - Đạo Tổ: 2 con (trắng, đen) bay lệch pha nhau 180°, thỉnh thoảng quấn vào nhau.
3. **Bị chém rớt cảnh giới**: thần thú hiện ra gãy vụn thành mảnh sáng rơi xuống (0.8s).

**Cách làm: chuỗi đốt (procedural), không vẽ từng frame**
- Thần thú = **1 sprite đầu + 12–20 sprite thân (đốt) + 1 sprite đuôi** (hạc/phượng thì: thân + 2 cánh vỗ 4 frame + đuôi lông). Code cho đầu chạy theo quỹ đạo, các đốt sau đi theo vị trí đầu ở các thời điểm trước đó (như con rắn) → uốn lượn mượt, không cần sinh hàng chục frame, đổi màu bằng tint.
- **Quỹ đạo xoắn ốc**: `x = cx + R(t)·cos(θ(t))`, `y = feet_y − H(t) + R(t)·sin(θ(t))·0.35` (ép dẹt theo góc nhìn 3/4); `θ` tăng 1.5 vòng, `H` từ 0 lên `3 × chiều cao nhân vật`, `R` từ 0.6 → 1.0 × bán kính thân rồi bung ra khi bay vút lên.
- **Chiều sâu**: đốt nào đang ở nửa sau vòng xoắn (`sin θ < 0`) vẽ **dưới** sprite nhân vật, nửa trước vẽ **trên** → trông như rồng quấn quanh người thật.
- Mỗi đốt nhỏ dần về đuôi; đầu hơi lắc theo hướng bay; độ trong suốt tăng dần ở 0.4s cuối.
- Vệt theo sau: hạt particle sinh từ các đốt, tồn tại 0.5s.
- Tất cả kích thước nhân `scale` của nhân vật.
- Chỉ chạy ở client, kích hoạt bởi event `Breakthrough` / `RealmDrop` và đồng hồ client (bản thường trực) → không tốn băng thông.
- Ẩn Thân: thần thú mờ theo chủ (20%). Trong bụi: ẩn theo chủ.
- **Hiệu năng (vì giờ luôn bật)**: số đốt theo chất lượng — Cao 16 đốt + vệt hạt; Vừa 10 đốt, vệt hạt 50%; Thấp 6 đốt, không vệt. Thần thú của người ở xa (chủ nhỏ hơn 40 px trên màn hình) chỉ vẽ đầu + 4 đốt. Không giới hạn số con, nhưng vệt hạt chỉ bật cho tối đa 6 con gần mình nhất. Cập nhật quỹ đạo bằng toán thuần (không tạo object mới mỗi frame).
- Placeholder khi chưa có art: đầu = tam giác, đốt = hình tròn, tô màu theo bảng trên.

### 3.6 Bộ công cụ VFX (nâng chất lượng hiệu ứng Tiên giới)

**Vì sao đang xấu**: hào quang hiện là 1 đĩa phẳng + 1 vòng viền, blend thường, không phát sáng, không chuyển động nhiều lớp → nhìn như hình dán.

**Nguyên tắc** (áp cho mọi hiệu ứng ánh sáng: hào quang, thần thú, Thiên Kiếp, Đài Tiên Khí, viên Tiên khí):
1. **Blend cộng (`blendMode = 'add'`)** cho mọi phần phát sáng — màu chồng lên nhau sẽ sáng lên thay vì che nhau. Không tốn hiệu năng.
2. **Nhiều lớp, mỗi lớp chuyển động riêng** — không bao giờ dùng 1 hình tĩnh:
   - Lớp 1: quầng sáng mềm (radial gradient, blend cộng), thở ±8% chu kỳ 2s.
   - Lớp 2: vòng phù văn / bát quái quay chậm (1 vòng / 12s).
   - Lớp 3: tia sáng tỏa ra quay ngược chiều (1 vòng / 20s), độ trong 40%.
   - Lớp 4: hạt sáng bay lên từ chân (particle), tốc độ ngẫu nhiên, mờ dần.
   - Đạo Tổ: lớp âm dương quay + viền cầu vồng đổi màu theo thời gian (hue shift).
3. **Bloom chỉ cho lớp VFX**: đặt mọi hiệu ứng sáng vào 1 container riêng `vfxLayer`, gắn `AdvancedBloomFilter` (thư viện `pixi-filters`) **cho container đó thôi**, không gắn cho cả màn hình.
4. **Hiệu ứng khoảnh khắc** (`pixi-filters`): `ShockwaveFilter` (sóng xung kích khi đột phá Tiên / Thiên Tiên Đài phun), `GlowFilter` (viền sáng nhân vật Tiên giới), `GodrayFilter` (tia sáng trên Thiên Tiên Đài khi sắp phun).
5. **"Juice"**: đột phá có khựng hình 80ms (hit-stop), rung camera, chớp trắng 1 frame, chữ cảnh giới to bật ra (scale 0 → 1.3 → 1.0, easing back-out).

**Particle**: dùng thư viện particle có trình chỉnh sửa trực quan để thiết kế bằng mắt rồi xuất config JSON (không chỉnh số trong code):
- Ưu tiên **`revolt-fx`** (hỗ trợ PixiJS 8, có editor online, làm được chuỗi hiệu ứng theo thời gian).
- Phương án 2: `@spd789562/particle-emitter` (bản port `@pixi/particle-emitter` cho PixiJS 8, dùng ParticleContainer).
- Config hiệu ứng lưu ở `client/src/vfx/presets/*.json`.

**Texture hạt miễn phí (CC0, dùng thương mại được)**: Kenney Particle Pack, Brackeys VFX Bundle (itch.io, gồm flipbook). Ghi vào `assets/CREDITS.md`. Dùng các texture này cho tia lửa, khói, ánh sáng, vì chúng đẹp và ổn định hơn ảnh AI sinh cho hạt nhỏ; art AI giữ cho phần có "nhân vật" (đầu rồng, phượng, phù văn).

**Hiệu năng (gắn với §13.6)**:
- Chất lượng **Cao**: đủ bloom + filter + particle. **Vừa**: bloom chất lượng thấp, particle 50%. **Thấp**: tắt bloom/filter, chỉ blend cộng + particle 25%.
- Chỉ người trong khung nhìn mới có VFX; người ở xa (nhỏ hơn 40 px trên màn hình) chỉ còn lớp quầng sáng.
- Bloom filter chỉ 1 lần cho cả `vfxLayer`, không gắn filter riêng từng nhân vật.

## 4. Kiếm, vòng quay, tầm đánh

### 4.1a Vòng kiếm nhiều lớp (kiểu cánh quạt như quảng cáo)

Mục tiêu hình ảnh: nhiều vòng kiếm **so le, nghiêng, mỗi vòng một màu**, nhìn thành một dải dày xoáy như chong chóng. Tất cả các lớp đều lộ ra, không nấp sau nhau.

**Quy tắc cố định bán kính (BẮT BUỘC)**
- Bán kính và chiều dài của một vòng **chỉ phụ thuộc vào thứ tự vòng `k` và `scale`**, **KHÔNG phụ thuộc vào số kiếm `n`**. Thêm kiếm vào vòng đang có → chỉ lấp kín khe, vòng **không rộng ra**.
- Tầm chém chỉ tăng khi (a) mở **vòng mới** (kiếm thứ 11, 31, 61, 101...), hoặc (b) nhân vật **lên tầng/cảnh giới** (scale tăng). Không có vật phẩm nào tăng tầm chém.
- Xóa mọi công thức cũ kiểu `ring_radius = max(..., n * spacing / 2π)` (từ spec v1) khỏi code.
- Chuôi kiếm vòng 1 luôn sát người (`inner_r = (BODY_R + 6) * scale`), không có khoảng trống lớn giữa thân và kiếm.

Ví dụ ở cùng 1 tầng (scale không đổi): 1 kiếm, 5 kiếm, 10 kiếm → cùng 1 vòng, cùng bán kính. 11 kiếm → xuất hiện vòng 2 xa hơn. 30 kiếm → vẫn 2 vòng, bán kính không đổi.

**Chia vòng**
- Vòng thứ `k` (k = 0 là vòng 1) chứa tối đa `10 × (k+1)` kiếm: vòng 1 = 10, vòng 2 = 20, vòng 3 = 30... Kiếm lấp đầy vòng trong trước rồi mới sang vòng ngoài.
- Số vòng đang hiện = số vòng cần để chứa n kiếm (`R` nhỏ nhất sao cho `5R(R+1) ≥ n`), không vượt quá `r + 1` của cảnh giới.
- Trong 1 vòng, các kiếm chia đều 360°: vòng đầy có khoảng cách `36° / (k+1)`; vòng ngoài cùng chưa đầy (m kiếm) thì chia đều `360° / m`.

**Hình học từng vòng** (`R` = tổng số vòng hiện có, `base = SWORD_LEN * scale`, SWORD_LEN = 80)

```
inner_r           = (BODY_R + 6) * scale                     // bán kính chuôi vòng 1
hilt_r(k)         = inner_r + HILT_STEP * k * base           // HILT_STEP = 0.1  → vòng ngoài lùi ra xa hơn
len(k)            = base * (1 + (RING_GROWTH - HILT_STEP) * k) // RING_GROWTH = 0.2 → mũi vòng sau xa hơn vòng trước 20% chiều dài gốc
stagger(k)        = k == 0 ? 0 : 0.5 * 360° / (10 * (k+1))    // so le nửa khe của chính vòng đó → kiếm vòng ngoài không trùng thẳng hàng với vòng 1
tilt              = TILT_DEG * dir                           // TILT_DEG = 35°, lưỡi ngả về phía sau so với chiều quay
angle(i, k, t)    = phase + dir * ω * t + slot_angle(i) + stagger(k)
ω                 = 2π / rev_secs * buff_spin                // rev_secs = 1.0 → 1 vòng/giây; buff_spin = 1.6 khi có Tốc Kiếm
```

- Mỗi thanh kiếm đặt chuôi tại điểm `(hilt_r(k), angle)` và chĩa ra theo hướng **bán kính xoay thêm `tilt`**, dài `len(k)`.
- Đổi chiều quay → `tilt` lật theo, tween 0.1s (lưỡi luôn ngả về phía sau, như kéo lê theo vòng xoáy).
- Tầm chém thực tế: `reach = max_k |tip(k)|` (khoảng cách từ tâm tới mũi kiếm xa nhất, tính hình học vì kiếm nghiêng). Tham khảo tầng 10, full kiếm: Luyện Khí ≈ 160 · Trúc Cơ ≈ 245 · Kim Đan ≈ 340 · Nguyên Anh ≈ 450 · Đại Thừa ≈ 1 200 · Đạo Tổ ≈ 1 900 đơn vị.
- Config `ring_growth_mode: "linear" | "compound"` vẫn giữ để thử.

**Màu từng vòng**
- Vòng k tô màu `ring_tints[k % số_màu]` (mặc định cam → tím → xanh lá → đỏ → xanh lơ → vàng). Skin kiếm theo cảnh giới vẫn giữ hình dáng, chỉ nhuộm màu theo vòng.
- Vòng 1 dùng màu gốc của skin (tint trắng) để người chơi nhận ra cảnh giới.

**Thứ tự vẽ**: vòng ngoài cùng vẽ trước (dưới), vòng 1 vẽ sau cùng (trên). Mỗi vòng có lớp spin blur cùng màu (mục 4.1b).

**Hitbox (server dùng đúng hình học trên)**
- Mỗi kiếm: capsule từ chuôi tới mũi theo hướng đã nghiêng, độ dày `sword_width = 12 * scale`.
- Thân: hình tròn `BODY_R * scale` (BODY_R = 30).
- Số kiếm tối thiểu **0** (có thể bị tước sạch kiếm). Bắt đầu: 1 kiếm. Thêm/bớt kiếm hoặc đổi số vòng (stagger thay đổi): tween vị trí 0.2s.

### 4.1b Va chạm và hiển thị khi kiếm quay
- **Tốc độ gốc: 1 vòng / 1 giây** (`rev_secs: 1.0`). Có buff Tốc Kiếm: 1 vòng / 0.625 giây.
- **Va chạm theo góc quét**: ở 30 tick/s, mỗi tick kiếm quay ~12° (~19° khi có buff). Vẫn kiểm tra **cung đã quét** từ tick trước tới tick này (sweep), không chỉ vị trí tức thời, để kiếm không "xuyên" qua người. Mục tiêu nằm trong vành khăn `[inner_r, reach]` và trong cung quét của bất kỳ kiếm nào → trúng.
- **Hiển thị**: ở 60 fps mỗi frame quay ~6°, nhỏ hơn nhiều so với khoảng cách 18° giữa 2 kiếm → nhìn rõ từng thanh kiếm quay, không còn hiệu ứng quay ngược. Vệt mờ (spin blur) chỉ cần nhẹ, mảnh, cùng màu vòng; đậm hơn khi có buff Tốc Kiếm.
- Dùng `ParticleContainer` của PixiJS cho kiếm (có thể tới hàng nghìn sprite trên màn hình).

### 4.1 Tốc độ di chuyển & camera

```
move_speed (đơn vị thế giới/s) = 195 * buff_haste(1.4 | 1.0) * burn(1.8 | 1.0) * zone_mult     // không còn nhân sqrt(scale)
camera: chiều cao vùng nhìn thấy (đơn vị thế giới) = max(900 * scale^0.85, 2.6 * reach)
```

- Tốc độ chạy **giống nhau ở mọi cảnh giới** trong thế giới. Chỉ vật phẩm (Thần Hành) và Thiêu Đốt Tinh Huyết (§4.1c) mới tăng tốc.
- Camera lùi theo cảnh giới nên trên màn hình người to trông đi chậm hơn người nhỏ → người nhỏ né được.

### 4.1c Thiêu Đốt Tinh Huyết (chạy nước rút)
- Giữ **Shift**, hoặc **nháy đúp rồi giữ** chuột (PC); giữ nút **🔥**, hoặc **chạm 2 lần rồi giữ** (điện thoại): tốc chạy × 1.8.
- Trong lúc giữ: mỗi giây đốt **8% EXP của tầng hiện tại** (`exp_needed(r, s) × 0.08`). Có thể rớt tầng nhưng **không bao giờ rớt cảnh giới**: tiến độ chạm đáy cảnh giới hiện tại (tầng 1, 0%) thì tự tắt.
- **Ở Tiên giới** (Chân Tiên trở lên): đốt **Tiên khí** thay cho EXP — mỗi giây 8% độ rộng khoảng Tiên khí của cảnh giới đang đứng (Chân Tiên 8/giây, Kim Tiên 16, Thái Ất 24, Đại La 32, **Đạo Tổ 40**), không bao giờ xuống dưới đáy cảnh giới đang đứng. Đạo Tổ đầy 1 500 đốt được bình thường; sau đó phải nạp lại 1 500 mới đổi Uy Danh tiếp (§3.4).
- Hiệu ứng: người bốc khói đỏ, mặt đỏ gay, vệt lửa sau chân, emote "Liều mạng!".

## 5. Chiến đấu

Thứ tự xử lý mỗi tick server:

### 5.1 Kiếm A chạm thân B — rớt 1 đại cảnh giới
(A ≠ B, B không bất tử, không có khiên, không đang dịch chuyển)
- Nếu B đang ở **Luyện Khí** (bất kỳ tầng nào) → B **chết** (mục 7). `loss = B.total_exp`; A nhận 50% loss + `kill_bonus_exp` (30), rơi 30% loss thành viên EXP, +1 kill.
- Ngược lại B **rớt 1 đại cảnh giới, thanh về mức sàn** của cảnh giới mới (tầng 1, 0%): `total_exp' = sàn(r − 1)` = tổng EXP cộng dồn của các cảnh giới trước đó. `loss = total_exp − total_exp'`.
  - Ví dụ: Trúc Cơ tầng 2 (bất kể tiến độ) → **Luyện Khí tầng 1, 0%**. Kim Đan tầng 7 → **Trúc Cơ tầng 1, 0%** (`total_exp' = 1 360`).
  - Sàn từng cảnh giới (cộng dồn, `realm_mult` 2.0): Luyện Khí 0 · Trúc Cơ 1 360 · Kim Đan 4 080 · Nguyên Anh 9 520 · Hóa Thần 20 400 · Luyện Hư 42 160 · Hợp Thể 85 680 · Độ Kiếp 172 720 · Đại Thừa 346 800.
- A nhận `50% * loss` EXP; rơi `30% * loss` dạng viên EXP quanh B.
- B bất tử **1s** (nhấp nháy), bị đẩy lùi khỏi A.
- Mỗi tick, mỗi nạn nhân tối đa 1 hit.
- Số kiếm > max của cảnh giới mới → kiếm thừa rơi xuống đất dưới dạng **bó kiếm** (mỗi bó tối đa 10 kiếm) để không tạo hàng trăm vật phẩm.
- Nếu B có buff Khiên: khiên vỡ, không mất gì, vẫn có 1s bất tử.

### 5.2 Kiếm A chạm kiếm B — bể kiếm

**Không đẩy văng** (`clash_knockback = 0`): hai bên mài kiếm vào nhau tới khi một bên hết lớp. VFX tia lửa + SFX "keng" tại điểm va.
- **Bể kiếm tính mỗi tick** (`clash_cooldown = 0`): ở 30 tick/s, hai người ngang nhau mài 1 giây là rụng ~30 kiếm mỗi bên (vd Kim Đan 60 kiếm → còn 30 sau 1s).
- **Đổi chiều quay** có cooldown riêng `flip_cooldown = 0.3s` (để vòng không giật 30 lần/giây).

**Luật chém thân theo số vòng** (thay cho lá chắn chặn đường đi ở bản trước):
- Hai người **đi xuyên qua vòng kiếm của nhau tự do**, không bị chặn, không bật ngược.
- Kiếm đối thủ **chỉ gây chém thân (§5.1) khi nạn nhân còn đúng 1 vòng** (≤ 10 kiếm). Còn từ 2 vòng trở lên thì thân "được che", kiếm chạm thân không tính.
- Kiếm bị bể luôn trừ từ **vòng ngoài cùng** trước. Vòng ngoài cùng hết kiếm → số vòng giảm 1.
- Hệ quả: Bó Kiếm khi rớt cảnh giới gần như không xảy ra (nạn nhân đã ≤ 10 kiếm lúc bị chém); Bó Kiếm đến chủ yếu từ xác người chết và vật phẩm trên bản đồ.

**Đổi chiều quay chỉ khi cùng cảnh giới**: `rA == rB` → cả hai vòng đổi chiều (`dir *= -1`). Khác cảnh giới → **không ai đổi chiều**, cả hai giữ nguyên chiều quay.

Sau đó tính bể kiếm theo **độ bền kiếm**: mỗi cảnh giới kiếm bền hơn cảnh giới ngay dưới **20%**.

```
durability(r) = 1.2^r                      // Luyện Khí 1.0 · Trúc Cơ 1.2 · Kim Đan 1.44 · Nguyên Anh 1.73 ...
D(người) = durability(r) × (1.5 nếu có buff Cương Kiếm)
k = D(bên cứng hơn) / D(bên mềm hơn) ≥ 1      // không buff: k = 1.2^d
```

- "Bên cao" / "bên thấp" trong bảng dưới nghĩa là bên có **D lớn hơn / nhỏ hơn**. Ví dụ Luyện Khí có Cương Kiếm (D = 1.5) va Trúc Cơ không buff (D = 1.2) → Luyện Khí là bên cứng hơn.
- D bằng nhau → mỗi bên mất 1.
- **Đổi chiều quay vẫn chỉ xét cảnh giới** (cùng cảnh giới mới đổi), không xét buff.

| Trường hợp | Kết quả mỗi lần va (u = `clash_unit` = 1) |
|---|---|
| Cùng cảnh giới (d = 0) | Mỗi bên mất `u` kiếm, cả hai đổi chiều |
| Khác cảnh giới | Bên thấp mất `u` kiếm; bên cao nhận `crack += lost / k`. Khi `crack ≥ 1` → bên cao mất `floor(crack)` kiếm, `crack -= floor(crack)`. Không ai đổi chiều |

Số kiếm có thể về **0** (`min_count = 0`). Lý do: vòng kiếm dày chặn hết kiếm đối thủ trước khi chạm tới thân, nên phải tước được sạch kiếm thì mới chém được vào người.
- 0 kiếm **không chết**: chỉ bị tước vũ khí. Chém trúng thân vẫn theo §5.1 (Trúc Cơ trở lên rớt 1 cảnh giới, Luyện Khí chết).
- Người 0 kiếm hiện emote hoảng loạn, nhặt item Kiếm/Bó Kiếm để có lại vòng. Bot 0 kiếm luôn chuyển sang Flee.
- Bên 0 kiếm không còn kiếm để va, nên không còn tính bể kiếm với họ.

Bảng tỉ lệ đổi kiếm (bên thấp mất : bên cao mất):

| Chênh lệch | Ví dụ | Tỉ lệ | Diễn giải |
|---|---|---|---|
| 0 | Kim Đan vs Kim Đan | 1 : 1 | Mỗi lần va mỗi bên −1, đổi chiều |
| 1 | Trúc Cơ vs Luyện Khí | 1.2 : 1 | **12 kiếm Luyện Khí bể → 10 kiếm Trúc Cơ bể** |
| 2 | Kim Đan vs Luyện Khí | 1.44 : 1 | ≈ 14 đổi 10 |
| 3 | Nguyên Anh vs Luyện Khí | 1.73 : 1 | ≈ 17 đổi 10 |
| 8 | Đại Thừa vs Luyện Khí | 4.3 : 1 | ≈ 43 đổi 10 |
| 13 | Đạo Tổ vs Luyện Khí | 10.7 : 1 | ≈ 107 đổi 10 |

→ Cảnh giới cao chỉ nhỉnh hơn mỗi bậc, đông kiếm vẫn có thể mài chết kiếm của người cao hơn 1–2 cảnh giới.

`crack` lưu trên người, reset về 0 khi chết. Hiển thị vết nứt nhẹ trên vòng kiếm khi `crack > 0`.

Kiếm bị bể: VFX vỡ vụn + SFX, mảnh văng ra (không thành item). Chủ nhân hiện emote "Kiếm của taaa!".

### 5.3 Kiếm chạm quái / vật cản
- Quái: 1 damage mỗi lần chạm (2 nếu có Cương Kiếm), cooldown mỗi cặp (kiếm, quái) 0.25s. Quái chết → **toàn bộ EXP rơi thành viên EXP** tại chỗ. Trong `orb_owner_secs` (1.5s) đầu, chỉ người chém hạ quái được nhặt; sau đó ai cũng nhặt được.
- Vật cản: tính damage **theo người, không theo từng thanh kiếm** (trước đây mỗi thanh kiếm gây damage riêng → 60–400 kiếm xay nát vật cản trong chớp mắt). Mỗi người chỉ gây damage lên 1 vật cản tối đa 1 lần / 0.5s, khi có ít nhất 1 kiếm của họ chạm vật cản:
  `damage = (r + 1) × (1 + 0.5 × (số_vòng − 1)) × (2 nếu có Cương Kiếm)` → mỗi vòng thêm +50% sức phá. Nhiều người cùng chém thì cộng dồn.
  **Kiếm xuyên qua vật cản và không bị mất/bể** khi chém vật cản.
- Kiếm không tương tác với chính chủ.

## 6. EXP rải khắp bản đồ & vật phẩm

### 6.1 Viên EXP môi trường (nguồn EXP chính)
- Bản đồ luôn phủ đều viên EXP: mật độ mặc định **1 viên / 140×140 đơn vị** (≈ 7 300 viên trên bản đồ 12 000²).
- Tỉ lệ: nhỏ +5 (80%) · vừa +20 (18%) · lớn +60 (2%).
- Viên bị nhặt → 5s sau mọc lại ở vị trí ngẫu nhiên **trong cùng ô lưới 1 000×1 000** (khu vực bị vét sạch sẽ tự đầy lại, không dồn về một chỗ).
- Bán kính hút đồ mặc định: `90 * scale`.
- Nhịp độ farm thuần chỉ để tham khảo, **không phải mục tiêu cân bằng**: lên cấp chủ yếu nhờ cướp Linh khí (§3.2). Dùng `cmd/sim --farm` để ghi lại số đo hiện tại vào `docs/PROGRESS.md` sau mỗi lần đổi config.
- Mạng: viên EXP đứng yên → chỉ gửi theo AOI khi vào tầm nhìn, sau đó chỉ gửi sự kiện nhặt/mọc.

### 6.2 Vật phẩm

| Item | Hiệu ứng | Thời hạn | Weight spawn |
|---|---|---|---|
| Kiếm | +3 kiếm. Nếu đã max → +20 EXP | – | 40 |
| Bó Kiếm | +15 kiếm (cũng là dạng rơi ra khi rớt cảnh giới/chết, khi đó chứa đúng số kiếm rơi) | – | 6 |
| Tốc Kiếm | **Chỉ** tăng tốc độ quay: ω × 1.6. **Không** đổi chiều dài/bán kính/tầm chém | 12s | 12 |
| Thần Hành | Tốc chạy × 1.4 | 10s | 12 |
| Nam Châm | Bán kính hút đồ × 3 | 15s | 8 |
| **Cương Kiếm** (độ cứng) | Kiếm cứng hơn: độ bền khi va kiếm × 1.5 (§5.2), sát thương lên quái và vật cản × 2 | 10s | 7 |
| Kim Cang (khiên) | Chặn 1 lần bị chém vào thân | 8s hoặc tới khi vỡ | 7 |
| **Bí Kíp** | +1 lượt dùng kỹ năng của nhân vật mình (tối đa 2), xem §6.3 | – | 10 |
| Song Bội Đan | EXP nhận × 2 | 20s | 5 |
| Ẩn Thân Phù | Tàng hình **80%**: người khác thấy mình và vòng kiếm mờ (độ đậm 20%), không thấy tên, không hiện trên minimap. Ở rất gần (120 × scale của người nhìn) thì hiện rõ 100%. Hết hiệu lực ngay khi kiếm mình chém trúng người | **5s** (tối đa, nhặt lại chỉ làm mới về 5s) | 3 |
| **Rương Báu** (hiếm) | Người đầu tiên **chạm vào** → nổ ra 8 item ngẫu nhiên văng xung quanh | – | 1 rương mỗi 60s, báo trước toàn phòng 10s |

- Số vật phẩm duy trì: **200/phòng**, item bị nhặt → 3s sau mọc lại chỗ khác.
- Nhặt lại buff đang có → làm mới thời gian (không cộng dồn).
- HUD hiển thị icon buff + vòng đếm ngược. Mỗi buff có aura màu riêng (trừ Ẩn Thân).
- Item spawn cách người chơi ≥ 300 đơn vị.

### 6.3 Kỹ năng nhân vật — mở bằng vật phẩm **Bí Kíp**

Mỗi nhân vật có **1 kỹ năng riêng**, nhưng **không tự dùng được**: phải nhặt vật phẩm **Bí Kíp** để có lượt dùng.
- **Bí Kíp** là vật phẩm chung (cuộn sách cổ), không gắn với nhân vật nào. Ai nhặt cũng được **+1 lượt dùng kỹ năng của chính nhân vật mình**. Tối đa giữ **2 lượt**; đã đủ 2 mà nhặt thêm → +20 Linh khí (Tiên giới: +2 Tiên khí).
- Phím **Space** (PC) / nút kỹ năng cạnh nút 🔥 (điện thoại). Nút hiện icon kỹ năng của nhân vật + **số lượt đang có**; 0 lượt thì nút xám. Giữa 2 lần dùng khóa 3s.
- Mất hết lượt khi chết. Không mất khi rớt cảnh giới hay qua cổng.
- Bán kính/khoảng cách hiệu ứng **nhân theo scale** người dùng; thời gian hiệu ứng không đổi.
- Luật cân bằng chung: kỹ năng **không** trực tiếp đổi cảnh giới/EXP, **không** chém thân thay kiếm, **không** xuyên luật "còn 1 vòng mới chém được thân" (§5.2).
- Không dùng được khi đang dịch chuyển; đang Ẩn Thân mà dùng thì lộ (hết Ẩn Thân).

| Nhân vật | Kỹ năng | Hiệu ứng |
|---|---|---|
| A Hớn Bánh Bao | **Bánh Bao Thần** | Ăn 1 cái bánh bao: **hồi ngay 20% số kiếm tối đa** của cảnh giới hiện tại (không vượt tối đa, lấp vào vòng trong trước), ợ một tiếng to |
| Bà Tám Loa | **Loa Phường** | Hét vào loa: sóng âm hình nón 60° dài 500 phía trước, **đẩy lùi** địch **500** và làm vòng kiếm của họ **đổi chiều** |
| Ông Ba Võng | **Ru Ngủ** | Đung đưa võng hát ru: địch trong bán kính **500** **chậm 40%**, vòng kiếm quay **chậm 50%** trong 3s (hiện "zzz" trên đầu) |
| Đại Ca Dép Tổ Ong | **Gọi Đàn Em** | Huýt sáo gọi 2 đàn em (NPC nhỏ hơn đại ca 30%, cùng cảnh giới) chạy theo **10s**. **Mỗi đàn em có số kiếm = 50% số kiếm hiện tại của đại ca** (làm tròn xuống, tối thiểu 1, xếp vòng như người chơi), lao vào mài kiếm đối thủ gần nhất. Đàn em không chém thân được, không nhặt đồ, không qua cổng; hết kiếm hoặc hết giờ thì biến mất (khói + câu "Đại ca ơi em đi đây"). Đại ca không bị trừ kiếm khi gọi |
| Thằng Mập Trà Sữa | **Phun Trân Châu (bịt mắt)** | Phun 1 loạt trân châu hình nón 45° dài 700 phía trước. Người trúng bị **trân châu dính đầy màn hình 5s** và **không nhặt được gì** trong 5s đó (vật phẩm, Linh khí, Tiên khí, Bí Kíp — đi xuyên qua) (xem dưới) |
| Cô Ba Gánh Hàng | **Gánh Lướt** | Lướt nhanh 600 theo hướng đi (0.3s), rơi trái cây dọc đường tạo vệt trơn 4s: địch đi vào bị **trượt theo quán tính 1s và choáng (stun)**: không điều khiển được, không dùng kỹ năng/Thiêu Đốt trong 1s (kiếm vẫn quay) |
| Sư Phụ Mì Gói | **Lưới Mì** | Ném lưới mì vào điểm cách 400 phía trước, bán kính 250: địch dính bị **trói 5s** (không di chuyển, kiếm vẫn quay) |
| Bé Tí Cơ Bắp | **Gồng Cơ** | Gồng **5s**: độ bền kiếm **× 2** (cộng dồn với Cương Kiếm), không bị đẩy lùi/chậm/trói |
| Anh Shipper Tia Chớp | **Giao Hỏa Tốc** | Tốc chạy **× 3** trong **5s** và **đi xuyên mọi vật cản** (kể cả vách đá, rặng tre, tường không phá được), **trừ biên khu vực**. Hết 5s mà đang kẹt trong vật cản → được đẩy mượt ra chỗ trống gần nhất trong 0.3s |
| Cụ Rùa Thiền | **Rụt Cổ** | Rụt vào mai **5s**: đứng yên, **không bị bể kiếm, không bị chém thân**; ra khỏi mai bật sóng đẩy địch xung quanh 200 |

**Trân châu dính màn hình** (hiệu ứng của Thằng Mập):
- Chỉ hiện trên màn hình **người bị trúng** (lớp overlay phía client, server gửi event `Blinded {secs}`).
- 25–40 viên trân châu đen bóng kèm vệt trà sữa **bắn "bẹp" lên màn hình** ở vị trí ngẫu nhiên, phủ khoảng **55–65%** diện tích, dày hơn ở giữa; mỗi viên **trượt chậm xuống** để lại vệt, rồi rơi/tan dần trong 5s.
- Trong 5s đó người bị trúng **không nhặt được bất cứ thứ gì** (server bỏ qua nhặt đồ); quanh nhân vật hiện icon 🚫 nhỏ. Có tiếng "bẹp bẹp" + "rột rột".
- Không che HUD (thanh tiến độ, nút, minimap) để người chơi vẫn biết mình đang ở đâu.
- Bot bị trúng: tầm nhìn còn 40% và không nhặt đồ trong 5s.
- Trúng lần nữa khi đang bị: làm mới thời gian, thêm 10 viên.

- Mỗi kỹ năng có câu thoại tấu hài khi dùng (bong bóng thoại) + SFX riêng.
- Bot nhặt Bí Kíp như vật phẩm thường và dùng kỹ năng theo luật đơn giản (vd Cụ Rùa dùng Rụt Cổ khi còn 1 vòng và có địch gần; Bà Tám dùng Loa khi địch trong nón).
- Server tính mọi hiệu ứng; client nhận event `SkillCast {id, skill, pos, dir}`, `SkillEffect`, `Blinded`.
- **Không cân bằng theo tỉ lệ thắng 1-đấu-1.** Kỹ năng có loại tăng sức mạnh, có loại để khắc chế/gây khó chịu, nên tỉ lệ thắng tay đôi không phản ánh giá trị thật. Cân bằng dựa trên **phản hồi người chơi** và số liệu trận thật (tỉ lệ nhặt Bí Kíp rồi dùng, hiệu quả trong giao tranh đông người).

## 7. Chết & màn hình chờ

- **Rơi kiếm khi chết** (người và bot): `floor(n * death_sword_drop)` kiếm (mặc định 50%) văng ra quanh xác dưới dạng Bó Kiếm (mỗi bó ≤ 10). Ví dụ chết (ở Luyện Khí) với 10 kiếm → 1 bó 5 kiếm.
- **Người thật**: anim `death` 1s → rời phòng → **Màn hình Tổng kết** hiển thị:
  - Bị ai chém (tên + nhân vật), câu tấu hài random ("Bạn đã được tiễn về quê nuôi cá").
  - Cảnh giới cao nhất đạt được, tổng EXP đỉnh, số người đã chém, thời gian sống, hạng trong phòng.
  - Nút: **Chơi tiếp** (matchmaking lại ngay, giữ tên + nhân vật), **Bảng xếp hạng**, **Đổi nhân vật**.
- Khi vào lại: Luyện Khí tầng 1, 1 kiếm, không buff, **bất tử 5s**, spawn ở chỗ an toàn (không có ai trong 800 đơn vị).
- **Bot**: chết thì xử lý theo mục 9.3 (respawn tại chỗ hoặc bị xóa).
- Kill feed tấu hài trong trận: `"{A} vừa tiễn {B} về quê nuôi cá"`, `"{B} bị {A} chém rớt xuống {realm}"`, `"{A} đột phá {realm}, cả phòng run rẩy"`, `"{A} chui cổng ra liền bị {B} chém, đen thôi đỏ quên đi"`.

## 8. Bản đồ & địa hình

### 8.0 Thế giới 5 khu vực

Mỗi phòng là **1 thế giới gồm 5 khu vực tách biệt**. Mỗi khu vực to bằng toàn bộ bản đồ cũ (12 000 × 12 000) → tổng diện tích gấp 5 lần, giảm cảnh vừa vào đã bị dí.

- Các khu vực **không nối nhau bằng đường đi** (tường biên mỗi khu không qua được). **Chỉ đi qua khu khác bằng cổng dịch chuyển** (§8.4).
- Mỗi khu vực có thiết kế, bố cục và luật địa hình riêng:

| # | Khu vực | Bố cục | Điểm đặc biệt | Đài Tiên Khí |
|---|---|---|---|---|
| 1 | **Thiên Đình** | Bố cục hiện tại (§8.1): Thiên Tiên Đài giữa, 4 sân môn phái, đường chính + vành đai, 4 biome nhỏ | Khu "trung tâm", nơi duy nhất có Thiên Tiên Đài → điểm nóng giao tranh | 4 nhỏ + Thiên Tiên Đài |
| 2 | **Linh Sơn Cốc** | Các dãy vách đá (không phá được) chạy dài chia khu thành nhiều **hẻm núi**, nối bằng **đèo hẹp** và **cầu treo** (đoạn đường hẹp 200–300) | Đánh nhau ở đèo/cầu, người to khó qua đèo hẹp | 2 đài trên 2 "đỉnh núi" (cao nguyên nhỏ có 2 lối lên) |
| 3 | **Trúc Lâm Mê Cung** | Rặng tre dày (không phá được) tạo **mê cung** với nhiều ngõ cụt, bụi rậm khắp nơi, vài ao | Phục kích, tầm nhìn ngắn: trong khu này bụi rậm nhiều gấp 3 | 2 đài ở 2 khoảng trống giữa mê cung |
| 4 | **Phàm Trấn** | **Lưới phố** kiểu thị trấn: nhà (khung tường gạch có cửa), chợ đêm (sạp hàng tạo hẻm 150/250/350), quảng trường nhỏ | Nhiều vật cản phá được nhất → thay đổi địa hình liên tục | 2 đài ở 2 quảng trường |
| 5 | **Băng Nguyệt Hồ** | Hồ băng lớn ở giữa + đảo băng, bờ tuyết xung quanh, ít vật cản | **Mặt băng trơn**: trên băng di chuyển có quán tính (trượt, khó dừng/đổi hướng); bờ tuyết làm chậm 0.8 | 2 đài trên 2 đảo băng |

- Mỗi khu dùng thuật toán sinh riêng (`mapgen` nhận `zone_type`), vẫn tuân thủ: đường/quảng trường chính không vật cản, kiểm tra liên thông **trong khu**, §8.6 (không khối hình học lộ ra). Khu 1 giữ nguyên thuật toán §8.1.
- **Băng trơn**: vận tốc mới = vận tốc cũ × 0.92 + hướng input × gia tốc nhỏ mỗi tick (tức là trượt dài); thiết kế art cần mặt băng bóng, vết trượt.
- **Vào trận** (lần đầu hoặc "Chơi tiếp"): spawn ở **khu ít người thật nhất** (hòa → ngẫu nhiên), chỗ an toàn trong khu đó.
- HUD: tên khu vực hiện cạnh minimap; vào khu mới hiện chữ lớn mờ dần giữa màn hình ("— Trúc Lâm Mê Cung —", 1.5s).
- Minimap chỉ vẽ khu đang đứng. Nhấn **M** (phóng to): thêm cột bên cạnh liệt kê 5 khu + số người đang ở mỗi khu (không hiện vị trí).
- Kill feed: tin từ khu khác có tiền tố tên khu `[Linh Sơn Cốc]`. Bảng xếp hạng phòng, danh hiệu: tính chung cả thế giới.

### 8.1 Khu 1 — Thiên Đình: thuật toán sinh bản đồ "Tu Tiên Giới"

Bản đồ 12 000 × 12 000, tường biên không phá được. Mỗi phòng sinh 1 bản đồ từ `seed` (lưu lại để debug/tái hiện). Bản đồ Tiled vẽ tay (nếu có) được ưu tiên hơn bản đồ sinh.

**Bố cục cố định (giống một "tu tiên giới" có môn phái, đường núi, tiên đài)**

```
┌───────────────────────────────────────────┐
│ [Đài TK]  Linh Sơn         Trúc Lâm [Đài TK]│
│     ╲      (đá tảng)       (bụi, tre)  ╱     │
│      ╲___                         ___╱      │
│          ╲____   ┌─────────┐  ____╱          │
│               ╲__│ Thiên   │_╱               │
│                  │ Tiên Đài│                 │
│               ╱‾‾│ (giữa)  │‾╲               │
│          ____╱   └─────────┘  ╲____          │
│      ___╱                          ╲___      │
│     ╱      Phàm Trấn          Hắc Thị  ╲     │
│ [Đài TK]  (làng, rào tre)   (chợ, hẻm)[Đài TK]│
└───────────────────────────────────────────┘
```

- **Thiên Tiên Đài** ở trung tâm: quảng trường bán kính 900, nền đá sáng, không vật cản bên trong, viền là vòng cột/tường thấp có 4 lối vào (theo 4 đường chính).
- **4 Đài Tiên Khí** gần 4 góc (cách mỗi cạnh 15% bản đồ), mỗi đài nằm trong một **sân môn phái**: tường không phá được bao quanh, có 2–3 cổng ra vào, bán kính sân 700.
- **4 vùng (biome) theo 4 góc phần tư**, dùng lại art hiện có:
  - Linh Sơn (núi đá): nhiều đá tảng thành cụm, ít bụi.
  - Trúc Lâm (rừng tre): nhiều bụi rậm + rào tre, chỗ phục kích.
  - Phàm Trấn (làng): các "ngôi nhà" là khung tường gạch hình chữ nhật có cửa, rào tre quanh vườn.
  - Hắc Thị (chợ): lưới sạp hàng/xe đẩy tạo **hẻm hẹp** 150 / 250 / 350.
  - Ranh giới giữa các vùng dùng **Voronoi có nhiễu** (không phải đường thẳng) để trông tự nhiên.

**Các bước thuật toán**
1. **Khung**: đặt Thiên Tiên Đài, 4 sân môn phái + Đài Tiên Khí, 4 điểm vào của sân.
2. **Đường chính**: nối mỗi sân môn phái tới Thiên Tiên Đài bằng đường cong (Bezier + nhiễu nhẹ), rộng 500. Thêm **đường vành đai** vòng tròn bán kính ~3 500 nối 4 vùng. Đường lát đá sáng màu, **không đặt vật cản trên đường**.
3. **Đối xứng xoay 4 phía**: sinh nội dung 1 góc phần tư, xoay 90° cho 3 góc còn lại (công bằng cho mọi hướng), rồi mỗi góc áp **biome riêng** (đổi loại vật cản/decor) và xê dịch ngẫu nhiên nhỏ (±150) để không lộ là bản sao.
4. **Vật cản theo biome**: dùng **Poisson-disk sampling** (khoảng cách tối thiểu theo loại) + **cụm** (mỗi cụm 3–7 vật cản). Tường gạch xếp thành khung nhà có lỗ cửa; rào tre xếp thành dãy liền dọc mép đường; sạp hàng xếp lưới tạo hẻm.
5. **Vùng địa hình**: bụi rậm thành cụm ở mép rừng và cạnh đường (chỗ phục kích); bùn ở chỗ trũng giữa vùng; đường gió chạy dọc vài đoạn đường chính hướng về tâm.
6. **Cổng dịch chuyển**: 4 cổng mỗi khu (khu 1: 1 cổng trong mỗi sân môn phái; khu khác: rải đều 4 phía), thuộc mạng cổng chung của cả thế giới (§8.4). Cổng chỉ đặt ở chỗ trống, cách vật cản ≥ 300.
7. **Decor**: Poisson-disk thưa (≤ 1 món / 600×600), không đặt trên đường, không đè vật cản.
8. **Kiểm tra hợp lệ** (sai thì sinh lại với seed+1, tối đa 10 lần):
   - Lưới đi bộ 100×100: mọi ô trống phải **liên thông** (flood fill) — không có khu bị nhốt.
   - Diện tích trống ≥ 65%.
   - Không vật cản trong bán kính 300 quanh cổng, Đài Tiên Khí, lối vào sân.
   - Mỗi biome có ít nhất 1 hẻm/lối hẹp và 1 cụm bụi.
9. Xuất bản đồ ra cùng định dạng object như Tiled (`.tmj` JSON) để client/server dùng chung và để xem lại trong Tiled.

Thêm lệnh `go run ./cmd/mapgen --seed N --png out.png` xuất ảnh xem trước để duyệt nhanh.

### 8.2 Vật cản (chặn thân, có độ cứng)

| Loại | HP | Ghi chú |
|---|---|---|
| Hàng rào tre | 30 | Mỏng, dài |
| Sạp hàng / xe đẩy | 80 | Vỡ ra: 30% rơi 1 item |
| Tường gạch | 250 | |
| Đá tảng | 800 | Rất lớn |
| Tường đá cổ / biên map | ∞ | Không phá được |

Thời gian 1 người (đủ kiếm tối đa của cảnh giới) phá vỡ, theo công thức §5.3:

| Cảnh giới (số vòng) | Sức phá/giây | Rào tre | Sạp hàng | Tường gạch | Đá tảng |
|---|---|---|---|---|---|
| Luyện Khí (1) | 2 | 15s | 40s | 125s | 400s |
| Trúc Cơ (2) | 6 | 5s | 13s | 42s | 133s |
| Kim Đan (3) | 12 | 2.5s | 6.7s | 21s | 67s |
| Nguyên Anh (4) | 20 | 1.5s | 4s | 12.5s | 40s |
| Hóa Thần (5) | 30 | 1s | 2.7s | 8.3s | 27s |
| Đại Thừa (9) | 90 | 0.3s | 0.9s | 2.8s | 8.9s |
| Đạo Tổ (14) | 210 | tức thì | 0.4s | 1.2s | 3.8s |

→ Người nhỏ gần như không phá nổi tường gạch, đá tảng → dùng làm chỗ nấp thật sự. Người khổng lồ cảnh giới cao thì càn quét được — đúng cảm giác "đại năng".

- **Hai vùng va chạm khác nhau cho mỗi vật cản:**
  - **Chân đế (footprint)** — chặn thân nhân vật và quái. Chỉ là phần sát mặt đất: mặc định hình elip/chữ nhật rộng 90% sprite, cao **30% sprite, nằm ở đáy**. Trong Tiled đặt được thủ công bằng thuộc tính `footprint` (nếu không có thì tính tự động như trên).
  - **Thân hình (visual bounds)** — vùng kiếm chém trúng để gây damage (đúng với cái người chơi nhìn thấy).
- **Sắp xếp lớp vẽ theo chân (y-sort)**: nhân vật đứng phía sau (y chân nhỏ hơn đáy vật cản) được vẽ khuất sau vật cản; phần vật cản che lên nhân vật mình thì mờ đi 50% để vẫn thấy mình.
- **Vật cản dạng dãy** (hàng rào, tường): art phải liền mạch, nối tiếp nhau không hở (tile ngang). Chỗ nào art có khe hở thì footprint cũng phải hở theo — người chơi thấy đi qua được thì phải đi qua được.
- Kiếm đi xuyên vật cản và gây damage (mục 5.3).
- Hiển thị 3 trạng thái theo HP: nguyên vẹn (>66%), nứt (33–66%), sắp vỡ (<33%).
- **Vỡ** → thành đống đổ nát, đi qua được. Sau **30s** mọc lại:
  - 3s trước khi mọc: hiệu ứng nhấp nháy cảnh báo.
  - Nếu có nhân vật/quái đứng trong vùng lúc mọc → chờ tới khi trống rồi mới mọc (không kẹt người).
  - Mọc lại với HP đầy.

### 8.2a Di chuyển mượt khi chạm vật cản (sửa lỗi giật)

Nguyên nhân thường gặp của giật/khựng khi đi sát vật cản: (1) client dự đoán va chạm khác server (khác hình footprint, quên góc xoay ±6°/lật) → server kéo nhân vật về → giật; (2) đẩy ra khỏi vật cản bằng cách "bật" theo pháp tuyến → rung qua lại giữa 2 vật cản sát nhau; (3) khe giữa các đoạn rào/tường liền nhau làm vướng như bậc thềm; (4) sửa sai vị trí bằng cách nhảy thẳng tới vị trí server.

Bắt buộc:
- **Dùng chung 1 mã va chạm** cho server và client dự đoán (port cùng thuật toán, cùng dữ liệu footprint đã tính sẵn **sau khi** xoay/lật, cùng số thực). Có test so khớp: 1 000 lần di chuyển ngẫu nhiên sát vật cản → vị trí client và server lệch < 0.5 đơn vị.
- **Trượt dọc** (slide): khi chạm, bỏ thành phần vận tốc đi vào vật cản, giữ thành phần song song bề mặt. Lặp tối đa 3 lần trong 1 tick để xử lý góc hai vật cản. Không bao giờ bật ngược.
- Footprint trơn: hình tròn / capsule / đa giác lồi có bo góc; các đoạn rào/tường liền nhau **gộp thành 1 collider** dài, không để khe/bậc giữa các đoạn.
- Vật cản vỡ/mọc lại: khi mọc lại trùng người, đẩy người ra mượt trong 0.2s (không dịch tức thời).
- **Hiệu chỉnh dự đoán mượt**: lệch < 30 đơn vị → kéo dần về trong 100ms; lệch lớn hơn mới nhảy thẳng.
- Debug `H`: vẽ footprint client (xanh) và server (đỏ) chồng lên nhau, có bộ đếm "số lần hiệu chỉnh/giây" trong F3.

### 8.3 Vùng địa hình (không chặn)

| Vùng | Hiệu ứng | Dùng để lật kèo |
|---|---|---|
| Bụi rậm | Ai đứng trong bụi thì người ngoài bụi không thấy (trừ khi đứng gần 150 × scale). Bị chém trúng thì lộ 2s | Phục kích, trốn |
| Bùn lầy | Tốc chạy × 0.6 | Dụ kẻ đuổi vào bùn |
| Đường gió | Lệch so với hướng gió < 60° → × 1.5 (theo gió); > 120° → × 0.8 (ngược gió); còn lại × 1.0 (ngang gió) | Đường thoát nhanh |
| Hẻm hẹp | Các lối đi rộng 150 / 250 / 350 giữa vật cản không phá được | Người to (thân rộng) không chui lọt, người nhỏ trốn vào được |

### 8.4 Cổng dịch chuyển (đích ngẫu nhiên, nối 5 khu)
- Mỗi khu có **4 cổng** → cả thế giới 20 cổng, cùng 1 mạng. **Không còn ghép cặp cố định.**
- Bước vào 1 cổng → ra ở **1 cổng ngẫu nhiên khác trong cả thế giới** (có thể cùng khu hoặc khu khác; không bao giờ ra lại chính cổng vừa vào). Với 20 cổng, ~80% lần dịch chuyển sẽ sang khu khác. Config `gates.exit_pool`: `"all"` (mặc định) | `"other_zone"` (luôn sang khu khác). Tắt ngẫu nhiên: `gates.random_exit: false` (khi đó ra cổng kế tiếp theo vòng).
- Kích hoạt khi tâm nhân vật đi vào bán kính cổng (150). Anim `teleport_out` 0.4s → xuất hiện ở cổng đích với anim `teleport_in` 0.4s.
- Cổng đích **lóe sáng 0.5s trước** khi người xuất hiện (cảnh báo nhẹ cho người đứng gần — vẫn dễ bị bất ngờ).
- **Không bất tử khi vừa ra cổng.** Trong lúc đang dịch chuyển (0.8s) không bị chém và không chém được.
- Cooldown mỗi người: 8s mới dùng cổng tiếp được.
- Người to cỡ nào cũng dùng được.
- Mỗi cổng có màu theo khu nó đang đứng (để biết mình đang ở khu nào), nhưng **không cho biết** sẽ ra ở đâu.
- Chuyển khu: server chuyển nhân vật sang mô phỏng của khu đích ở ranh giới tick (mọi trạng thái — kiếm, buff, crack, Tiên khí — giữ nguyên).

### 8.4a Thiên Môn — cổng thẳng tới Thiên Đình

Mỗi khu **trừ Thiên Đình** có thêm **1 Thiên Môn ở chính giữa khu** (tổng 4). Đây là lối đi **chắc chắn** tới Thiên Đình cho người muốn leo cao (Thiên Tiên Đài, 4 Đài Tiên Khí nhỏ), khác với 4 cổng thường đi ngẫu nhiên.

- **Một chiều, đích cố định**: bước vào Thiên Môn → luôn ra ở Thiên Đình. Từ Thiên Đình muốn rời đi thì dùng cổng thường (ngẫu nhiên).
- **Điểm ra ở Thiên Đình**: 4 **Bến Thiên Môn** nằm ngoài rìa quảng trường Thiên Tiên Đài (cách tâm ~1 400, mỗi bến ở đầu 1 con đường chính). Mỗi khu nối cố định với 1 bến (Linh Sơn Cốc → bến Bắc, Trúc Lâm → Đông, Phàm Trấn → Nam, Băng Nguyệt Hồ → Tây). Bến là bục đá nhỏ có cột sáng, không vật cản trong bán kính 400.
- Cùng luật cổng thường: kích hoạt khi tâm nhân vật vào bán kính (200), dịch chuyển 0.8s không chém/không bị chém, **không bất tử khi ra**, bến đích lóe sáng 0.5s trước. Cooldown chung với cổng thường (8s).
- **Vị trí trong từng khu** (mapgen phải chừa quảng trường bán kính 500 ở tâm khu, có ít nhất 2 đường dẫn tới, kiểm tra liên thông):
  - Linh Sơn Cốc: trên một bình nguyên giữa các hẻm núi, 2–3 đèo dẫn vào.
  - Trúc Lâm Mê Cung: khoảng trống trung tâm mê cung (đường vào ngoằn ngoèo — muốn lên Thiên Đình phải tìm đường).
  - Phàm Trấn: quảng trường chính giữa thị trấn.
  - Băng Nguyệt Hồ: đảo băng giữa hồ (phải trượt băng mới tới).
- **Nhận diện**: to gấp đôi cổng thường, cổng đá trắng – mái vàng, cột sáng vàng bắn lên trời (thấy từ xa), biển "Thiên Đình ↑". Minimap: icon vàng riêng, luôn hiện. Khi đứng trong khu, góc minimap có mũi tên nhỏ chỉ hướng tới Thiên Môn.
- Bot: dùng Thiên Môn khi đã ở Đại Thừa trở lên hoặc khi Thiên Tiên Đài sắp phun (xác suất cao), còn lại hiếm khi dùng.
- Kill feed khi người trong Top 3 bước qua Thiên Môn: `"{A} ({danh hiệu}) vừa bước qua Thiên Môn — Thiên Đình dậy sóng!"`.

### 8.5 Đài Tiên Khí

| Đài | Vị trí | Phun thường | Phun lớn |
|---|---|---|---|
| **Đài Tiên Khí nhỏ** (khu 1: ×4, mỗi khu khác: ×2 — tổng 12) | Khu 1: giữa 4 sân môn phái; khu khác: theo bảng §8.0 | Mỗi **5s**: 2–3 viên nhỏ (2 điểm) | Mỗi **30s**: 15 viên nhỏ + 2 viên vừa (10 điểm) |
| **Thiên Tiên Đài** (×1, chỉ khu 1) | Trung tâm khu Thiên Đình | – | Mỗi **150s**: 1 viên **khổng lồ** (150 điểm) |

- Viên văng ra ngẫu nhiên trong bán kính 300–700 quanh đài (theo cung, như đài phun nước).
- **Báo trước**: đài nhỏ rung + sáng dần 3s trước đợt phun lớn; Thiên Tiên Đài thông báo toàn phòng **10s trước** (`"Thiên Tiên Đài sắp giáng Tiên khí!"`) + đếm ngược trên đỉnh đài + icon trên minimap.
- Viên Tiên khí khổng lồ: to, sáng vàng kim, bay lơ lửng nhấp nhô; ai chạm trước nhận cả viên.
- Viên Tiên khí không mọc lại ở chỗ khác; tồn tại tối đa 60s rồi tan.
- Đài là địa hình không phá được, kiếm đi xuyên. Không vật cản quanh đài 300 đơn vị.
- Minimap: hiện 5 đài; đài sắp phun lớn thì nhấp nháy.

### 8.6 Hình ảnh bản đồ sinh động (không dùng khối hình học)

**Quy tắc cứng**: bản đồ hiển thị **không được có hình chữ nhật thẳng cạnh hay hình tròn hoàn hảo** nào lộ ra (vùng nền, bùn, bụi, đường, sân...). Hình học đơn giản chỉ được dùng cho va chạm và chế độ debug (`H`).

**1. Mặt đất: trộn texture theo nhiễu (texture splatting)**
- Mỗi biome có 2–3 texture nền (vd Linh Sơn: đá xám, sỏi, rêu; Trúc Lâm: cỏ, lá khô, đất ẩm). Trộn bằng **mặt nạ nhiễu** (Perlin/Simplex, 2 tầng tần số) → mảng màu loang tự nhiên thay vì ô vuông.
- Ranh giới giữa 2 biome: dải chuyển tiếp rộng 400–600, trộn theo nhiễu (không phải đường thẳng hay đường cong trơn).
- **Bake sẵn** lúc vào phòng thành các tấm `RenderTexture` 2048×2048 (chunk), chỉ vẽ chunk trong khung nhìn. Không trộn lại mỗi frame.

**2. Vùng địa hình có hình dạng hữu cơ**
- Bùn, ao, bụi, sân, vùng gió... sinh dạng **đa giác méo theo nhiễu**: `r(θ) = r0 × (1 + 0.25 × noise(θ))`, làm mượt (Chaikin 2 lần), 24–40 đỉnh.
- **Server dùng đúng đa giác này** cho logic (điểm-trong-đa-giác) → hình thấy và vùng tác dụng khớp nhau. `mapgen` xuất đỉnh đa giác trong file bản đồ.
- Mép vùng: viền mềm (feather 24–48 px bake sẵn) + 1 dải viền texture riêng (mép bùn có vệt nứt, mép ao có cỏ lau, mép đường có sỏi vụn).
- Đường đá: vẽ theo đường cong với mép răng cưa nhẹ, thỉnh thoảng có viên đá lát lệch, rêu mọc kẽ.

**3. Decal & trang trí rải theo biome** (bake vào chunk, không có va chạm)
- Mỗi chunk 30–60 decal ngẫu nhiên (Poisson-disk), **xoay / lật / co giãn 0.7–1.3 / đổi sắc nhẹ** để không lặp: vết nứt, khóm cỏ, lá rụng, sỏi, vũng nước nhỏ, rễ cây, hoa dại nhỏ màu nhạt, vết bánh xe, mảnh ngói vỡ...
- Quanh vật cản: bóng đổ mềm + vụn nhỏ dưới chân (tường gạch có gạch vỡ, đá tảng có sỏi, sạp hàng có rác) → vật cản "cắm" vào đất thay vì nổi lên.
- Vật cản cùng loại: xoay nhẹ ±6°, lật ngang ngẫu nhiên, 2–3 biến thể sprite.

**4. Nước & điểm nhấn**
- Mỗi bản đồ 2–4 **ao / suối nhỏ** (đa giác hữu cơ, chỉ là vùng làm chậm như bùn hoặc trang trí không va chạm), mặt nước có shader gợn sóng nhẹ + phản chiếu lấp lánh.
- Biome Linh Sơn có **vách đá / vực** dọc biên (thay tường biên thẳng), Trúc Lâm có hàng tre dày làm biên.

**5. Sự sống (ambient, chỉ client, không ảnh hưởng gameplay)**
- **Bóng mây** trôi chậm trên mặt đất (2–4 bóng mờ cỡ lớn).
- Theo biome: Trúc Lâm lá tre rơi + đom đóm; Linh Sơn sương mù mỏng trôi; Phàm Trấn gà chạy, bướm; Hắc Thị đèn lồng đung đưa; Thiên Tiên Đài hạt sáng bay lên.
- Chim bay ngang màn hình thỉnh thoảng (bóng chim dưới đất).
- Giới hạn tổng số hạt ambient theo chất lượng (Cao 150 / Vừa 60 / Thấp 0).

**6. Màu sắc** vẫn tuân thủ ART_PROMPTS §9.0: nền nhạt, tương phản thấp; vật chơi được (EXP, item, quái, nhân vật) luôn nổi bật hơn nền.

## 9. Phòng, matchmaking, bot

### 9.1 Phòng
- Tối đa **20 người thật**/phòng (giữ như cũ, chưa tăng cho tới khi tối ưu server xong). 1 phòng = 1 thế giới 5 khu. Mỗi khu mô phỏng ở 1 goroutine riêng, chỉ trao đổi khi có người qua cổng.
- Phòng tự hủy khi 0 người thật trong 60s.

### 9.2 Matchmaking
```
on_join(player):
  candidates = rooms where humans < 20
  room = candidate có humans lớn nhất; hòa → phòng tạo sớm nhất
  if none: room = create_room(random_map)
```

### 9.3 Bot
```
every 2s per room:                         // tổng cả thế giới, như cũ
  if humans < 20 and humans + bots < 20:
      spawn min(20 - humans - bots, 2) bots vào KHU ĐANG ÍT NHÂN VẬT NHẤT
on human join/leave:  KHÔNG xóa bot
on bot death:
  if humans + (bots - 1) >= 20: remove bot
  else: respawn bot sau 3s ở khu đang ít nhân vật nhất
```
- Bot spawn **ngang hạng 11 của phòng** (`bots.spawn_rank = 11`): lấy cảnh giới + tầng của người đang đứng hạng 11 bảng phòng (theo Uy Danh); phòng chưa đủ 11 người thì lấy người cuối bảng. Không có trần cảnh giới. Số kiếm ngẫu nhiên [1, max].
- Bot **không vào leaderboard toàn cục** (server biết, người chơi không biết).
- **Tên bot phải không phân biệt được với người thật** (§9.4).
- Bot cũng dùng cổng: khi bỏ chạy (Flee) gần cổng, hoặc khu đang có quá ít mục tiêu/vật phẩm (xác suất thấp, cooldown 60s).

### 9.4 Đặt tên bot

- Nguồn: `config/bot_names.txt` — 1 000 tên theo đúng kiểu nickname game thủ Việt hay dùng: tên thật (có/không dấu, kèm năm sinh như `2k7`, `_09`), tên ở nhà (Tèo, Bống, Mít...), biệt danh tu tiên hài (Sư Huynh Nợ Tiền, Tán Tu Lang Thang...), câu nói đùa (Đừng gank em, Wifi hàng xóm...), tên tiếng Anh ghép (SleepyNinja, LazyPanda...), kiểu teencode chữ thường (`tho2k9`, `meo.22`), và một ít có ký tự trang trí ASCII (`_Max.`, `^^Vy_`). Không có chữ "bot", không có tên người nổi tiếng hay nhân vật có bản quyền.
- **Ký tự trang trí chỉ dùng ASCII** (`_ . ~ ^ x`) vì BitmapFont của bảng tên (§12.3) có thể không có glyph ★ ♥ ✿. Tên người chơi thật có ký tự lạ ngoài font → vẽ bằng font dự phòng, không được hiện ô vuông.
- **Luật chọn tên trong 1 room**:
  - Rút ngẫu nhiên không lặp; không trùng tên bất kỳ người/bot nào đang trong room hoặc vừa rời trong 10 phút.
  - 15% số lần: thêm biến thể nhỏ (đổi `_` ↔ `.`, thêm 1–2 chữ số) để tên không lặp y hệt giữa các room.
  - Phân bố nhân vật bot giống phân bố nhân vật người chơi chọn (lấy thống kê 7 ngày, chưa có thì đều).
- **Hành vi giống người khi chết** (người thật chết → màn Tổng kết → bấm Chơi tiếp mất vài giây):
  - 70%: hồi sinh **cùng tên** sau ngẫu nhiên 4–12s (như người bấm "Chơi tiếp"), vào khu ít người như người thật.
  - 30%: "rời phòng" — bot biến mất, bot mới (tên mới) vào sau 5–20s.
  - Thỉnh thoảng (5%) bot đứng yên 3–8s sau khi vào (như người đang đọc bảng hướng dẫn).
- Kill feed, tin loan, danh hiệu, Uy Danh hiển thị cho bot y hệt người thật.

## 10. Bot AI

Quyết định lại mỗi 0.3–0.6s (random), tìm đường bằng A* trên lưới 100 đơn vị (cập nhật khi vật cản vỡ/mọc).

| State | Điều kiện | Hành vi |
|---|---|---|
| Farm | Mặc định | Tới quái, item, vật cản yếu gần nhất |
| Hunt | Mục tiêu trong 600 × scale có `power` < 0.8× mình | Đuổi, nhắm vòng kiếm vào thân; tránh lao kiếm vào kiếm đối thủ cùng/cao cảnh giới hơn |
| Flee | Có kẻ `power` > 1.25× trong 500 × scale | Chạy ngược hướng; ưu tiên hẻm hẹp (nếu mình nhỏ hơn), bụi rậm, cổng |
| Ambush | Mình nhỏ, gần bụi rậm/cổng | Nấp chờ kẻ yếu đi ngang |
| Wander | Không có gì | Đi ngẫu nhiên |

- `power = sword_count * (r + 1) * reach`.
- Tham số tính cách random mỗi bot: `reaction_delay` 0.2–0.6s, `aim_noise`, `aggression`.
- Bot chỉ "thấy" những gì người chơi cùng vị trí thấy được (tôn trọng bụi rậm). Người đang Ẩn Thân: bot chỉ phát hiện trong 50% tầm nhìn bình thường.

## 11. Leaderboard (không cần login)

EXP **không** dùng để xếp hạng toàn cục vì có trần (Đại Thừa tầng 10) → ai chơi lâu cũng bằng nhau. Thay bằng điểm **Uy Danh** thưởng cho việc đánh nhau, lật kèo và giữ ngôi.

### 11.1 Điểm Uy Danh (tính trong 1 mạng, server tính)

| Hành động | Điểm |
|---|---|
| **Chém trúng người** (làm họ rớt cảnh giới hoặc chết) | `100 × (cảnh giới nạn nhân + 1) × lật_kèo` |
| Thưởng thêm khi chém chết | +50 |
| **Lật kèo**: nạn nhân cao cảnh giới hơn mình | `lật_kèo = 1 + 0.5 × (r_nạn_nhân − r_mình)`, tối thiểu 1 |
| **Đột phá** cảnh giới mới (lần đầu trong mạng đó) | `20 × r_mới` |
| **Giữ Top 1 phòng** | +1 điểm mỗi giây |

- Ví dụ: Trúc Cơ (r=1) chém trúng Nguyên Anh (r=3) → `100 × 4 × (1 + 0.5×2) = 800`. Đại Thừa chém Luyện Khí → `100 × 1 × 1 + 50 = 150`.
- Nạn nhân là **bot** → nhận 50% điểm.
- **Chống cày điểm**: chém trúng cùng 1 người (cùng `guest_id`) lần nữa trong vòng 60s → 0 điểm.
- HUD hiển thị Uy Danh hiện tại; mỗi lần cộng điểm hiện số bay lên (+800 LẬT KÈO!).
- Màn hình Tổng kết hiển thị bảng chi tiết điểm theo từng nguồn.

### 11.2 Các bảng xếp hạng

- **Trong phòng** (realtime, 1 lần/giây): top 10 xếp theo **Uy Danh** của mạng hiện tại, mỗi dòng hiện kèm cảnh giới + tầng (để biết ai mạnh mà né/săn). Top 1–10 có danh hiệu (§11.3).
- **Toàn cục**, tab **Hôm nay / Tuần này / Mọi thời đại**, 3 bảng:
  - **Uy Danh** (bảng chính): điểm Uy Danh cao nhất trong 1 mạng.
  - **Sát Thần**: số lần chém trúng người nhiều nhất trong 1 mạng.
  - **Thần Tốc**: thời gian ngắn nhất từ lúc vào trận tới Đại Thừa (chỉ ghi những mạng đã đạt Đại Thừa; xếp tăng dần).
- Lưu mỗi "mạng" vào SQLite: `guest_id, name, character, score, hits, kills, peak_level, secs_to_dai_thua (null nếu chưa đạt), top1_secs, survive_secs, map, ended_at`.
- API: `GET /api/leaderboard?period=day|week|all&board=score|hits|speed` → top 50, cache 10s.
- Điểm do server tính → client không gian lận được. Bot không được ghi.

### 11.3 Danh hiệu Top 1–10 phòng

| Hạng | Danh hiệu | Màu |
|---|---|---|
| Top 1 | **Bần tăng chưa ngán ai bao giờ** + vương miện | Vàng kim |
| Top 2 | **Ngon thì nhào vô** | Bạc |
| Top 3 | **Chỗ này nhiều máu nè** | Đồng |
| Top 4–10 | Theo mảng `titles` trong config (giữ đúng bộ danh hiệu đang chạy trong code) | Trắng |

- Danh hiệu hiện **phía trên tên** nhân vật trong trận và cạnh tên trong bảng phòng.
- Chống nhấp nháy: giữ hạng mới liên tục 3s mới đổi danh hiệu.
- Kill feed khi lên ngôi / mất Top 1 (như cũ). Người Ẩn Thân / trong bụi: danh hiệu ẩn theo.

### 11.4 Tin loan toàn server
- Khi một người **đang trong Top 5 phòng** bị chém trúng (rớt cảnh giới hoặc chết), **mọi phòng trên server** đều nhận tin trên kill feed, ví dụ: `"[Phòng 3] {A} vừa hạ {B} — 'Ngon thì nhào vô' — rớt xuống {realm}!"`.
- Giới hạn tối đa 1 tin/giây toàn server (gộp nếu nhiều), không hiện tin của chính phòng mình 2 lần.

## 12. Trạng thái nhân vật, animation & tấu hài

Server tính `mood` và gửi xuống (mọi người thấy giống nhau). Giữ mood tối thiểu 0.5s để không nhấp nháy.

| Anim | Khi nào | Tấu hài |
|---|---|---|
| `idle` | Đứng yên | Ngáp, gãi đầu |
| `run` | Chạy, không có địch gần | La toáng "AAAA" |
| `chase` | Đang tiến lại kẻ yếu hơn trong tầm | Cười nham hiểm, xoa tay |
| `flee` | Đang chạy khỏi kẻ mạnh hơn | Giơ tay hoảng loạn, mồ hôi văng |
| `hit` | Bị chém (0.6s) | Mặt bẹp, sao quay quanh đầu |
| `levelup` | Lên tầng | Tạo dáng, lóe sáng |
| `breakthrough` | Lên cảnh giới | Sét đánh, tóc dựng |
| `realm_drop` | Rớt cảnh giới | Ngồi bệt khóc |
| `teleport_out` / `teleport_in` | Dịch chuyển | Bị hút xoáy / văng ra chóng mặt |
| `death` | Chết | Dẹp lép như bánh tráng, hồn bay lên |
| `respawn` | Vào trận | Rơi từ trời xuống, cắm mặt, đứng dậy phủi bụi |

- Emote bubble khi: mất kiếm, khiên vỡ, nhặt Rương Báu, bị lộ khỏi bụi. Mỗi loại có cooldown.
- Client tự lật sprite trái/phải theo hướng chạy.

### 12.3 Bảng tên trên đầu (nameplate): Danh hiệu · Cảnh giới · Tên

**Vấn đề hiện tại**: 3 dòng chữ trơn nằm đè lên vòng kiếm nhiều màu → khó đọc, không có cấp bậc thị giác.

**Bố cục** (từ trên xuống, căn giữa trên đầu nhân vật):

```
      ╭──────────────────────────────╮
      │ 👑  Bần tăng chưa ngán ai bao giờ │   ← Ruy băng danh hiệu (chỉ Top 1–10)
      ╰──────────────────────────────╯
          ╭───────────────╮
          │ ☯ ĐẠO TỔ   ▲ │                ← Huy hiệu cảnh giới (dạng viên thuốc)
          ╰───────────────╯
          TestThThuTianThu                ← Tên (to nhất, đậm nhất)
```

Cả cụm có **1 tấm nền chung** bo góc, đen 45%, viền mảnh sáng 1px → chữ luôn nổi trên mọi nền (kiếm, cỏ, đá). Khoảng cách các dòng 2–4px, cụm cách đỉnh đầu 8px.

**1. Ruy băng danh hiệu** (chỉ Top 1–10)

| Hạng | Nền ruy băng | Chữ | Hiệu ứng |
|---|---|---|---|
| Top 1 | Ruy băng hai đuôi xẻ, đỏ son viền vàng | Vàng kim chuyển sắc (sáng → đậm), viền nâu đậm 2px | Vương miện nhỏ bên trái; **vệt sáng lướt qua chữ mỗi 3s**; 2–3 đốm lấp lánh quanh ruy băng |
| Top 2 | Ruy băng xanh đen viền bạc | Bạc chuyển sắc | Vệt sáng lướt qua mỗi 5s |
| Top 3 | Ruy băng nâu viền đồng | Đồng chuyển sắc | Không |
| Top 4–10 | Dải mảnh trong suốt 60% | Trắng ngà, chữ nhỏ hơn | Không |

- Lên hạng: ruy băng **bật ra** (scale 0 → 1.2 → 1.0, 0.35s) + đốm sáng tung ra. Mất hạng: ruy băng rách đôi rơi xuống (0.4s).

**2. Huy hiệu cảnh giới** (dạng viên thuốc bo tròn)
- Bên trái: biểu tượng nhỏ của cảnh giới (dùng lại 14 huy hiệu cảnh giới, thu nhỏ). Giữa: tên cảnh giới **in hoa**, chữ đậm; phàm giới thêm số tầng nhỏ phía sau dạng chỉ số (`KIM ĐAN ⁵`).
- Màu nền theo cảnh giới (chữ là tông đậm cùng màu hoặc trắng nếu nền tối):

| Cảnh giới | Màu | | Cảnh giới | Màu |
|---|---|---|---|---|
| Luyện Khí | Xám xanh `#8D99AE` | | Hợp Thể | Cam `#F77F00` |
| Trúc Cơ | Xanh lá `#57CC99` | | Độ Kiếp | Đỏ sấm `#E63946` |
| Kim Đan | Vàng đồng `#E9C46A` | | Đại Thừa | Đen, viền & chữ vàng `#FFD60A` |
| Nguyên Anh | Xanh lam `#4EA8DE` | | Tiên giới (5 cấp) | Màu hào quang tương ứng (§3.4) |
| Hóa Thần | Tím `#9D4EDD` | | | |
| Luyện Hư | Xanh ngọc `#2EC4B6` | | | |

- **Tiên giới**: viền huy hiệu là dải màu chảy vòng quanh (gradient xoay), chữ phát sáng nhẹ; Đạo Tổ viền cầu vồng đổi màu liên tục.
- **Chỉ báo nguy hiểm** (chỉ hiện trên người khác, bên phải huy hiệu): ▲ đỏ = cao cảnh giới hơn bạn · ● vàng = ngang · ▼ xanh = thấp hơn. Giúp quyết định né hay săn trong nửa giây.
- Lên tầng: số tầng lăn lên (0.3s). Đột phá: huy hiệu chớp trắng + phình to 1.3 rồi về.

**3. Tên người chơi**
- Dòng **to nhất**: chữ đậm, trắng, viền đen 3px + bóng đổ nhẹ phía dưới.
- **Tên của chính bạn**: vàng ấm + mũi tên nhỏ ▼ phía trên cụm (dễ tìm mình giữa đám đông).
- Bot hiển thị giống người (không lộ là bot).

**Kích thước cố định trên màn hình** (không phóng to theo scale nhân vật, không co theo camera):

| | PC | Điện thoại |
|---|---|---|
| Tên | 18 px | 15 px |
| Cảnh giới | 13 px | 11 px |
| Danh hiệu | 13 px (Top 1: 15 px) | 11 px (Top 1: 13 px) |

**Font**: phải hỗ trợ đầy đủ dấu tiếng Việt. Tên + cảnh giới: **Be Vietnam Pro** (ExtraBold / Bold). Danh hiệu: 1 font trang trí có bộ ký tự Vietnamese trên Google Fonts (lọc theo ngôn ngữ Vietnamese; agent tự kiểm tra hiển thị đủ dấu, ví dụ thử chuỗi "Bần tăng chưa ngán ai bao giờ"), nếu không có thì dùng Be Vietnam Pro Italic.

**Gọn màn hình (declutter)**
- Nhân vật nhỏ hơn 40 px trên màn hình: chỉ hiện **tên** (+ vương miện nếu Top 1).
- Hơn 12 bảng tên trên màn hình: ẩn ruy băng Top 4–10 của người ở xa.
- 2 bảng tên chồng nhau: bảng của người ở xa mờ 50%.
- Ẩn Thân / trong bụi: ẩn cả cụm (như cũ).

**Hiệu năng**
- Dựng mỗi bảng tên thành **1 texture cache** (vẽ lại chỉ khi nội dung đổi: hạng, cảnh giới, tầng). Hiệu ứng động (vệt sáng, viền chảy, đốm lấp lánh) là sprite riêng chồng lên, không vẽ lại texture.
- Chữ dùng BitmapFont sinh sẵn (gồm đủ ký tự tiếng Việt), không dùng `Text` thường.
- Chất lượng Thấp: tắt vệt sáng, đốm lấp lánh, viền chảy (giữ màu tĩnh).

### 12.4 Ảnh đại diện ở góc thông tin (thay số cấp)

Thay vòng tròn số "14" ở góc trên trái bằng **ảnh đại diện nhân vật**:
- **Chân dung** nhân vật đang chơi (cắt từ đầu tới vai), trong khung tròn 64 px (PC) / 52 px (điện thoại). Biểu cảm đổi theo trạng thái: bình thường · cười nham hiểm khi đang đuổi · hoảng khi bị chém / còn 1 vòng kiếm · khóc khi rớt cảnh giới (dùng 4 chân dung).
- **Viền khung = màu cảnh giới** (bảng §12.3). Tiên giới: viền là hào quang động theo màu §3.4; Đạo Tổ viền âm dương xoay.
- **Huy hiệu nhỏ** góc dưới phải khung: biểu tượng cảnh giới + số tầng (phàm giới) — thay cho con số to cũ.
- Vương miện nhỏ trên khung nếu đang Top 1; ruy băng hạng nhỏ nếu Top 2–10.
- Bị chém: khung rung + chớp đỏ. Đột phá: khung phát sáng + phình ra.
- Chưa có art: dùng frame `idle_00` của nhân vật, tự cắt phần đầu (vùng trên 45% sprite) làm chân dung tạm.

### 12.5 Hoạt ảnh liền mạch & tự nhiên

**Vấn đề**: các frame sinh bằng AI lệch nhau (to nhỏ, vị trí, dáng, nét vẽ khác nhau) → nhân vật "giật, nhảy lung tung". Mỗi mood (run / chase / flee) lại là 1 bộ frame AI riêng → đổi mood là đổi cả dáng người, nhìn như 3 nhân vật khác nhau.

**Giải pháp gồm 3 lớp:**

**A. Một thân, nhiều khuôn mặt** (bỏ việc mỗi mood một bộ chạy riêng)
- Mỗi nhân vật chỉ có **1 chu kỳ chạy** (`run`, 6 frame) và **1 chu kỳ đứng** (`idle`, 4 frame) cho phần thân.
- Biểu cảm (bình thường, la hét, cười nham hiểm, hoảng, khóc, choáng...) là **lớp mặt riêng** chồng lên đầu (sprite mặt 6–8 cái, cùng kích thước, cùng vị trí neo). Đổi mood = **chỉ đổi mặt**, thân vẫn chạy liền mạch.
- `chase` / `flee` khác `run` bằng **tư thế thủ tục** (procedural): chase ngả người tới trước 8° + hai tay xoa (lớp tay nếu có); flee ngả ra sau 6° + nảy cao hơn + giọt mồ hôi văng (particle).
- Anim một lần (hit, levelup, breakthrough, death, respawn, teleport, skill) vẫn là frame riêng nhưng phải qua bước chuẩn hóa B.

**B. Chuẩn hóa frame tự động** (`tools/atlas-pack`, chạy trên mọi frame trước khi ghép atlas)
1. **Neo chân**: tìm đường đáy của 2 bàn chân, đặt về y = 460, căn giữa x theo **trung điểm hai chân** (không theo hộp bao — tay vung ra làm hộp bao lệch).
2. **Cùng tỉ lệ cho cả animation**: đo chiều cao phần đầu+thân (bỏ tay chân) từng frame, lấy **trung vị**, co giãn mọi frame của animation về cùng chiều cao đó (không co giãn từng frame riêng lẻ).
3. **Neo đầu**: vị trí đầu giữa các frame liên tiếp không được lệch quá 4 px (ngoài nhịp nhún); lệch hơn → căn lại hoặc đánh dấu frame hỏng.
4. **Cùng bảng màu & nét viền**: lượng tử hóa về palette của nhân vật (ART_PROMPTS §5), chuẩn hóa độ dày viền.
5. **Điểm "độ giật"**: so từng cặp frame liên tiếp (lệch tâm, lệch chiều cao, khác màu) → điểm 0–100. Frame vượt ngưỡng → báo đỏ trong `assets:report`, đề xuất sinh lại riêng frame đó.
6. Xuất GIF xem trước + ảnh **onion-skin** (chồng mờ các frame) để duyệt bằng mắt.

**C. Bộ điều khiển hoạt ảnh lúc chạy (client)**
- **Chuyển mượt**: đổi animation có crossfade 80–120 ms (không cắt cứng).
- **Giữ pha chu kỳ**: run ↔ chase ↔ flee dùng chung chu kỳ thân nên **không reset về frame 0** khi đổi mood.
- **Tốc độ anim theo tốc độ chạy thật**: fps chu kỳ chạy = tốc độ di chuyển / độ dài bước chân → không bị "trượt chân" (chân bước chậm mà người lướt nhanh). Thiêu Đốt, Thần Hành, băng trơn đều tự khớp.
- **Thời gian giữ tối thiểu**: mood giữ ít nhất 0.5 s (đã có), biểu cảm mặt giữ ít nhất 0.8 s.
- **Quay đầu trái/phải**: không lật tức thì — co ngang 0.08 s (scale X 1 → 0.2) rồi lật và giãn lại; chỉ lật khi hướng đổi > 30° và giữ > 0.12 s (chống lật qua lại khi chạy dọc).
- **Chuyển động phụ thủ tục** (áp lên mọi anim): nhún theo bước chạy (sóng sin đồng bộ pha chân), nghiêng người theo hướng rẽ (tối đa 6°), squash & stretch khi dừng/đáp đất (±6%), thở nhẹ khi đứng. Biên độ nhỏ, đồng nhất cho mọi nhân vật → cảm giác cùng 1 "chất".
- **Không giật theo mạng**: trạng thái anim lấy từ vị trí đã **nội suy** (không từ gói tin thô); nhận mood mới từ server thì đợi tới mốc frame gần nhất mới chuyển.
- Debug `?debug=true` phím `J`: hiện tên anim, frame, pha, điểm độ giật của nhân vật đang chọn.

**Nâng cấp sau (nếu A–C vẫn chưa đủ)**: chuyển sang **rig cắt mảnh 2D** — từ 1 ảnh nhân vật chuẩn, cắt đầu / thân / 2 tay / 2 chân thành sprite riêng có điểm khớp, animation bằng xoay/dịch các mảnh trong code. Mọi frame dùng **đúng 1 bộ hình** nên tuyệt đối đồng nhất; đổi skin chỉ cần thay bộ mảnh.

### 12.2 Máu quái
- **Thanh máu nhỏ** trên đầu quái (không còn số to đè lên mặt quái). Chỉ hiện khi quái đã mất máu; đổi màu xanh → vàng → đỏ theo % máu.
- Art quái không cần chừa vùng trống giữa bụng nữa.

### 12.1 Minimap

- Góc **trên bên trái** (dưới thanh cảnh giới), hình vuông bo góc, nền mờ 60%. Kích thước: 180px trên PC, 120px trên điện thoại. Chạm/nhấn phím `M` → phóng to giữa màn hình (bấm lại để thu nhỏ).
- **Hiển thị:**
  - Địa hình tĩnh vẽ sẵn 1 lần từ bản đồ Tiled: tường không phá được, đá tảng, bụi rậm, bùn, đường gió (màu nhạt, đơn giản).
  - Cổng dịch chuyển: chấm tròn theo màu cặp/mạng cổng.
  - **Mình**: mũi tên chỉ hướng di chuyển, màu nổi bật.
  - Khung chữ nhật mảnh thể hiện vùng camera đang nhìn.
  - **Top 1 phòng**: icon vương miện (cả phòng đều thấy → khuyến khích đi săn).
  - **Rương Báu**: icon rương nhấp nháy khi đang có, kể cả lúc báo trước 10s.
- **Không hiển thị** người chơi/bot khác (trừ Top 1), quái, item, viên EXP → giữ yếu tố bất ngờ và phục kích. Top 1 đang Ẩn Thân hoặc trong bụi thì ẩn khỏi minimap.
- Vật cản phá được không cần vẽ (đổi trạng thái liên tục).
- Dữ liệu: vị trí mình lấy ở client; vị trí Top 1 và Rương Báu gửi kèm gói `Leaderboard` (1 lần/giây).

## 13. Kiến trúc kỹ thuật

### 13.1 Stack đã chọn

| Phần | Công nghệ | Lý do |
|---|---|---|
| Server | **Go** (bản ổn định mới nhất) | Hiệu năng cao, code dễ đọc, compile nhanh, ra 1 file chạy duy nhất, Claude viết Go rất ổn định |
| WebSocket | `github.com/coder/websocket` | Gọn, hiện đại |
| Mã hóa gói tin | MessagePack (`vmihailenco/msgpack/v5` ↔ `@msgpack/msgpack`) | Nhỏ hơn JSON nhiều, dễ debug hơn binary tự chế |
| Database | SQLite qua `modernc.org/sqlite` | Không cần cài DB server, không cần CGO |
| Client | **TypeScript + Vite + PixiJS** | Chạy ngay trên trình duyệt PC & điện thoại (đúng kiểu `.io`), PixiJS vẽ hàng nghìn sprite 2D mượt bằng WebGL |
| Âm thanh | `howler.js` | Xử lý được việc điện thoại chặn auto-play |
| Bản đồ | Tiled (`.tmj`) | Vẽ map bằng kéo thả, không cần code |

Nguyên tắc: **server quyết định mọi thứ** (authoritative). Client chỉ gửi hướng di chuyển và vẽ lại những gì server gửi.

### 13.2 Cấu trúc repo
```
kiem-xoay/
├── CLAUDE.md
├── docs/            GAME_SPEC.md · ART_PROMPTS.md · SETUP_GUIDE.md · PROGRESS.md
├── config/game.json                 # mọi con số cân bằng; server đọc, client lấy qua GET /api/config
├── maps/            lang_tre.tmj · cho_dem.tmj · nui_da.tmj
├── server/
│   ├── go.mod
│   ├── cmd/server/main.go           # HTTP + WebSocket + phục vụ file client đã build
│   ├── cmd/sim/main.go              # chạy giả lập không giao diện với N bot để cân bằng game
│   └── internal/
│       ├── game/      # luật chơi thuần: entity, tick, va chạm, exp, kiếm, buff (có unit test)
│       ├── world/     # bản đồ, vật cản, vùng, cổng, lưới tìm đường
│       ├── bot/       # AI
│       ├── room/      # phòng, matchmaking, bot fill
│       ├── netproto/  # định nghĩa message
│       └── store/     # SQLite, leaderboard
├── client/
│   ├── package.json · vite.config.ts · index.html
│   └── src/
│       ├── main.ts
│       ├── net/       # kết nối WebSocket, giải mã, nội suy
│       ├── render/    # PixiJS: nhân vật, kiếm, bản đồ, VFX
│       ├── ui/        # trang chủ, HUD, màn hình chết, leaderboard, joystick
│       └── audio/
└── assets/          # xem ART_PROMPTS.md
```

### 13.3 Mạng

> **Đã được thay thế bởi `NETCODE_SPEC.md` (Protocol v2, cache nhiều lớp, đồng bộ theo khu).** Nội dung dưới đây chỉ giữ để tham khảo; chỗ nào mâu thuẫn thì theo NETCODE_SPEC.

- Tick server 30 Hz, mỗi phòng 1 goroutine. Snapshot gửi client 20 Hz.
- Client → Server: `Join{name, character, guest_id}`, `Input{seq, angle_u8, moving}` (30 Hz), `Ping`.
- Server → Client: `Welcome`, `Snapshot`, `Event`, `Leaderboard` (1 Hz).
- **Không gửi vị trí từng thanh kiếm**: gửi `phase, dir, n, ω, scale, phase_tick` (client tự tính số vòng, chiều dài từng vòng); client tự tính góc. Đổi chiều → event `RingFlip`.
- Event: `Hit, SwordBreak, Death, LevelUp, Breakthrough, RealmDrop, RingFlip, ObstacleDamaged, ObstacleBroken, ObstacleRegrow, TeleportOut, TeleportIn, ItemPicked, BuffStart, BuffEnd, ChestIncoming, KillFeed`.
- AOI: chỉ gửi những gì nằm trong vùng nhìn của người đó (+ lề). Người **trong bụi** không được gửi cho người không được thấy (chống hack nhìn xuyên). Người **Ẩn Thân** vẫn được gửi (vì vẫn hiện mờ 20%), kèm cờ `stealth`.
- Client: nội suy người khác (trễ 100ms); tự dự đoán chuyển động của mình + hiệu chỉnh theo server.

### 13.4 Va chạm (server)
- Lưới không gian (spatial hash) cho thân, quái, item, viên EXP, vật cản. Vật cản tĩnh index sẵn.
- **Không bao giờ duyệt từng thanh kiếm** để tìm va chạm. Mỗi vòng là 1 vật thể cứng (cùng tốc độ, cùng pha) → tính theo **vòng**: kiểm tra mục tiêu có nằm trong vành khăn của vòng không, rồi tính thẳng chỉ số kiếm gần nhất theo góc (O(1)), cộng cung quét trong tick.
- Kiếm–kiếm: luật chơi chỉ tính va chạm giữa **vòng ngoài cùng** của mỗi bên (kiếm bể luôn trừ từ vòng ngoài cùng, §5.2) → chỉ xét cặp vòng ngoài cùng của 2 người có vành khăn chồng nhau. Các vòng trong bỏ qua hoàn toàn, kể cả khi hai người đi xuyên vào nhau.
- Kiếm–quái / vật cản / viên EXP: chỉ dùng vòng tròn `reach` để lọc qua lưới, rồi kiểm tra vành khăn.
- Không cấp phát bộ nhớ mỗi tick (dùng lại slice, `sync.Pool` cho buffer gói tin).

### 13.6 Ngân sách hiệu năng & tối ưu (bắt buộc)

**Mục tiêu** (phòng đầy 40 nhân vật, trong đó 10 ở Tiên giới đủ kiếm ~1 000 thanh):

| Chỉ số | Mục tiêu |
|---|---|
| Server: thời gian 1 tick / phòng | < 4 ms (tick 33 ms) |
| Gói snapshot / client | < 8 KB, tổng < 40 KB/s |
| Client: FPS máy tầm trung / điện thoại tầm trung | ≥ 60 / ≥ 40 |
| Ping hiển thị (không tính địa lý) | tăng < 20 ms so với phòng trống |

**Đo trước khi sửa**
- Phím `F3` (chỉ với `?debug=true`): FPS, ms/frame, số sprite đang vẽ, số draw call, KB/s nhận, kích thước snapshot, tick server ms (server gửi kèm), số entity trong AOI.
- **Ping phải đo đúng**: server gắn timestamp vào `Pong`, client trả lời ngay trong handler WebSocket (hoặc Web Worker), không chờ vòng lặp vẽ. Nếu ping chỉ tăng khi FPS tụt → nghẽn ở client, không phải mạng.
- `cmd/loadtest`: mở N client giả (headless) vào 1 phòng, ép M người lên Đạo Tổ đủ kiếm, in tick ms, KB/s, p95 ping.

**Client (thường là thủ phạm chính)**
1. **Mỗi vòng kiếm vẽ thành 1 texture, rồi chỉ xoay texture đó.** Vòng kiếm quay như 1 khối cứng → vẽ toàn bộ kiếm của vòng (kể cả nghiêng, so le, màu, spin blur) vào 1 `RenderTexture` **một lần**, mỗi frame chỉ đổi `rotation` của 1 sprite. 1 050 kiếm → 14 sprite. Chỉ vẽ lại texture khi số kiếm của vòng đổi (giới hạn tối đa 5 lần/giây/người), khi đổi chiều (lật tilt), khi đổi skin/màu.
2. **LOD theo kích thước trên màn hình**: kích thước texture vòng theo đường kính thực trên màn hình (tối đa 2048). Vật nhỏ hơn 3 px trên màn hình (viên EXP khi camera lùi xa) → không vẽ; thay bằng lớp "bụi sáng" chung.
3. **Cull**: không vẽ gì ngoài màn hình (+ lề).
4. Chữ (tên, danh hiệu, số điểm bay lên) dùng `BitmapText`, không dùng `Text` (Text tạo texture mới mỗi lần đổi chữ). Bảng xếp hạng/HUD cập nhật tối đa 1–2 lần/giây.
5. Viên EXP, item, hạt VFX dùng `ParticleContainer` + 1 atlas → ít draw call.
6. Giải mã MessagePack trong **Web Worker**, luồng chính chỉ nhận dữ liệu đã giải mã.
7. Không tạo object mới mỗi frame (pool cho sprite, particle, vector).
8. **Tự hạ chất lượng**: FPS < 45 trong 3s → tắt spin blur, tắt thần thú thường trực, giảm particle 50%, hạ độ phân giải texture vòng; FPS ổn lại 10s → bật dần. Có tùy chọn Chất lượng: Cao / Vừa / Thấp.

**Server & mạng**
9. Lượng tử hóa: vị trí `u16` (bản đồ 12 000 → bước ~0.2), góc `u8`/`u16`, scale `u8`. Không gửi float64.
10. **Delta snapshot**: chỉ gửi trường thay đổi so với snapshot client đã xác nhận; entity đứng yên không gửi lại.
11. Viên EXP/Tiên khí/item: chỉ gửi sự kiện xuất hiện/biến mất, không nằm trong snapshot định kỳ. Camera lùi xa (Tiên giới) → **không gửi viên EXP nhỏ** ngoài bán kính 1 500 quanh người chơi (dù nằm trong khung nhìn), chỉ gửi viên vừa/lớn, Tiên khí, item.
12. Tần suất theo khoảng cách: entity gần (trong 1/2 khung nhìn) 20 Hz, xa 10 Hz.
13. Giới hạn tối đa 150 entity động mỗi snapshot, ưu tiên gần và to.
14. Kiểm tra nghẽn ghi: nếu bộ đệm gửi của 1 client đầy (mạng yếu) → bỏ snapshot cũ, chỉ gửi bản mới nhất (không xếp hàng làm trễ dồn).

**Ghi chú**: nếu mọi người đều ping ~200 ms kể cả phòng trống thì nguyên nhân là **khoảng cách địa lý** tới server (ví dụ người chơi ở Việt Nam, server ở Mỹ/Âu) → cần đặt server gần người chơi (Singapore / Việt Nam), không phải tối ưu code.

### 13.5 An toàn cơ bản
- Kiểm tra & giới hạn mọi input từ client; rate limit tin nhắn và số kết nối mỗi IP.
- Lọc tên tục, giới hạn độ dài.

## 14. Quái (PvE)

Quái **không gây sát thương**, đi lang thang, chạy khi bị chém, hiện số HP trên người.

| Loại | HP | EXP | Drop item | Weight |
|---|---|---|---|---|
| Nhỏ | 3 | 10 | 10% | 60 |
| Vừa | 8 | 30 | 25% | 30 |
| Lớn | 20 | 90 | 60% | 9 |
| Boss mini | 80 | 400 | 100% (3 item) | 1, tối đa 1/phòng |

Duy trì ~150 quái/phòng (bản đồ lớn hơn v1).

## 15. Milestones

Mỗi milestone là 1 phiên làm việc với Claude Code (prompt mẫu trong `SETUP_GUIDE.md`).

| # | Nội dung | Xong khi |
|---|---|---|
| M0 | Tạo repo, server Go "hello", client Vite + PixiJS kết nối WebSocket | Mở trình duyệt thấy chấm tròn di chuyển theo phím, vị trí do server gửi |
| M1 | Vòng kiếm, quái, item kiếm, EXP, 9 cảnh giới, scale + camera. Hình placeholder | Chơi 1 mình, lên Trúc Cơ thấy to ra, kiếm dài ra, camera lùi ra |
| M2 | PvP: chém thân, rớt tầng, bể kiếm theo cảnh giới, chết → màn hình Tổng kết → chơi tiếp | Mở 2 tab đánh nhau được |
| M3 | Phòng, matchmaking, bot fill, bot AI cơ bản, trang chủ nhập tên, leaderboard SQLite | 1 tab = 1 phòng có 19 bot; trang leaderboard có dữ liệu |
| M4 | Bản đồ Tiled: vật cản phá được + mọc lại, bụi, bùn, gió, hẻm, cổng; bot biết dùng địa hình | Chơi được trên 3 bản đồ |
| M5 | Toàn bộ vật phẩm buff + Rương Báu; netcode mượt (nội suy, dự đoán, AOI, ẩn thân) | Giả lập ping 150ms vẫn mượt |
| M6 | Ghép art, animation, mood, VFX, SFX, UI hoàn chỉnh | Đủ anim mục 12 |
| M7 | Tối ưu điện thoại, joystick, deploy lên VPS có HTTPS | Bạn bè vào được bằng link |

Sau mỗi milestone: chạy `go run ./cmd/sim` để xem cân bằng (thời gian trung bình lên mỗi cảnh giới, số kiếm trung bình, số lần chết).

## 16. Config mặc định (`config/game.json`)

```json
{
  "cultivation": {
    "realms": ["Luyện Khí","Trúc Cơ","Kim Đan","Nguyên Anh","Hóa Thần","Luyện Hư","Hợp Thể","Độ Kiếp","Đại Thừa"],
    "tien_realms": ["Chân Tiên","Kim Tiên","Thái Ất Ngọc Tiên","Đại La Kim Tiên","Đạo Tổ"],
    "tien_khi_ceilings": [100, 300, 600, 1000, 1500], "tien_entry_requires_full_exp": true,
    "tien_rob_share": 0.5, "tien_drop_share": 0.3,
    "tien_scale_step": 0.2, "tien_overflow_score": 0.1,
    "tien_aura_colors": ["#E8F1FF", "#FFD60A", "#4CE0B3", "#B15CFF", "yin_yang"],
    "tiers_per_realm": 10,
    "exp_base": 100, "realm_mult": 2.0, "tier_step": 0.08,
    "scale_min": 1.0, "scale_max": 5.0
  },
  "sword": {
    "start_count": 1, "min_count": 0,
    "max_swords": [10, 30, 60, 100, 150, 210, 280, 360, 450, 550, 660, 780, 910, 1050],
    "ring_capacity_step": 10, "length": 80, "width": 12, "hilt_gap": 6,
    "ring_growth": 0.2, "ring_growth_mode": "linear", "hilt_step": 0.1, "tilt_deg": 35,
    "ring_tints": ["#FFFFFF", "#C77DFF", "#7CFC6A", "#FF6B6B", "#4CC9F0", "#FFD60A"],
    "rev_secs": 1.0, "flip_cooldown": 0.3,
    "clash_unit": 1, "flip_only_same_realm": true, "clash_knockback": 0, "clash_cooldown": 0, "body_hit_max_rings": 1, "drop_bundle_max": 10,
    "durability_per_realm": 1.2, "durability_mode": "compound"
  },
  "player": {
    "body_r": 30, "move_base": 195, "move_scale_sqrt": false, "pickup_r": 90,
    "invuln_on_hit": 1.0, "invuln_on_spawn": 5.0,
    "burn_speed_mult": 1.8, "burn_exp_per_sec": 0.08, "burn_tien_frac_per_sec": 0.08,
    "attacker_share": 0.5, "drop_share": 0.3, "kill_bonus_exp": 30,
    "death_sword_drop": 0.5, "orb_owner_secs": 1.5,
    "camera_view_h": 900, "camera_exp": 0.85, "camera_reach_mult": 2.6,
    "hit_drops_realm": true
  },
  "buffs": {
    "spin":   { "mult": 1.6, "secs": 12 },
    "haste":  { "mult": 1.4, "secs": 10 },
    "magnet": { "mult": 3.0, "secs": 15 },
    "hard":   { "durability_mult": 1.5, "damage_mult": 2.0, "secs": 10 },
    "shield": { "secs": 8 },
    "exp2":   { "mult": 2.0, "secs": 20 },
    "stealth":{ "secs": 5, "max_secs": 5, "visible_alpha": 0.2, "reveal_r": 120, "bot_detect_mult": 0.5 }
  },
  "items": {
    "weights": { "sword": 40, "bundle": 6, "spin": 12, "haste": 12, "magnet": 8, "hard": 7, "shield": 7, "skill_scroll": 10, "exp2": 5, "stealth": 3 },
    "sword_item_amount": 3, "bundle_amount": 15,
    "count_per_room": 200, "respawn_secs": 3, "overcap_exp": 20, "chest_every_secs": 60, "chest_items": 8
  },
  "exp_orbs": {
    "cell": 140, "values": [5, 20, 60], "weights": [80, 18, 2],
    "respawn_secs": 5, "respawn_region": 1000
  },
  "obstacles": {
    "footprint_w": 0.9, "footprint_h": 0.3, "occlusion_alpha": 0.5,
    "regrow_secs": 30, "regrow_warn_secs": 3,
    "hp": { "fence": 30, "stall": 80, "brick": 250, "boulder": 800 },
    "dmg_cooldown_secs": 0.5, "dmg_per_extra_ring": 0.5,
    "stall_item_drop": 0.3
  },
  "zones": { "mud_mult": 0.6, "wind_with": 1.5, "wind_against": 0.8, "wind_with_max_deg": 60, "wind_against_min_deg": 120, "bush_reveal_r": 150, "bush_reveal_on_hit_secs": 2 },
  "gates": { "radius": 150, "transit_secs": 0.8, "exit_warn_secs": 0.5, "cooldown_secs": 8, "per_zone": 4, "random_exit": true, "exit_pool": "all",
    "heaven_gate": { "enabled": true, "radius": 200, "landing_r_from_center": 1400, "landing_clear_r": 400, "zone_center_plaza_r": 500, "map": { "linh_son_coc": "north", "truc_lam_me_cung": "east", "pham_tran": "south", "bang_nguyet_ho": "west" } } },
  "titles": ["Bần tăng chưa ngán ai bao giờ", "Ngon thì nhào vô", "Chỗ này nhiều máu nè"],
  "title_hold_secs": 3,
  "minimap": { "size_pc": 180, "size_mobile": 120, "show_top1": true, "show_chest": true, "show_others": false },
  "score": {
    "hit_base": 100, "kill_bonus": 50, "upset_per_realm": 0.5,
    "breakthrough_per_realm": 20, "top1_per_sec": 1,
    "bot_victim_mult": 0.5, "same_victim_cooldown_secs": 60
  },
  "altars": {
    "small": { "count": 4, "every_secs": 5, "orbs": [2, 3], "orb_value": 2, "burst_every_secs": 30, "burst_small": 15, "burst_medium": 2, "medium_value": 10, "warn_secs": 3 },
    "center": { "every_secs": 150, "orb_value": 150, "warn_secs": 10 },
    "spray_r": [300, 700], "orb_lifetime_secs": 60, "clear_r": 300
  },
  "mapgen": {
    "center_plaza_r": 900, "sect_r": 700, "sect_inset": 0.15, "road_w": 500, "ring_road_r": 3500,
    "symmetry": "rot4", "jitter": 150, "min_open_ratio": 0.65, "max_retries": 10
  },
  "bots": { "spawn_rank": 11 },
  "zones_def": ["thien_dinh", "linh_son_coc", "truc_lam_me_cung", "pham_tran", "bang_nguyet_ho"],
  "ice": { "friction": 0.92, "accel": 0.25, "snow_mult": 0.8 },
  "skills": { "key": "Space", "max_charges": 2, "lockout_secs": 3, "overcap_exp": 20, "overcap_tien_khi": 2, "blind": { "secs": 5, "no_pickup": true, "pearls": [25, 40], "cover": [0.55, 0.65], "bot_vision_mult": 0.4 } },
  "movement": { "slide_iterations": 3, "smooth_correct_max": 30, "smooth_correct_ms": 100 },
  "room": {
    "max_humans": 20, "bot_fill_target": 20, "zones": 5, "map_size": 12000,
    "monster_count": 150, "empty_close_secs": 60, "safe_spawn_r": 800
  },
  "net": { "tick_hz": 30, "snapshot_hz": 20, "interp_ms": 100 }
}
```

Độ bền kiếm: `durability_mode: "compound"` → `k = 1.2^d` (mặc định). Đổi sang `"linear"` → `k = 1 + 0.2 × d` nếu muốn khoảng cách giữa cảnh giới thấp và cao nhỏ hơn nữa (Đạo Tổ vs Luyện Khí chỉ 3.6 : 1).

## 18. Chế độ debug

- Mở game với `?debug=true` trên URL → bật phím debug: `E` +100 EXP, `R` lên cảnh giới, `K` +1 kiếm, `1–7` bật buff, `X` tự chết, **`B` thả 1 bot ngay cạnh mình**, phím vẽ hitbox capsule. Không có `?debug=true` thì mọi phím debug bị tắt (cả client lẫn server từ chối).
- Có thể tắt hẳn debug trên server production bằng biến môi trường.

### 18.1 Bảng hướng dẫn điều khiển lúc vào trận
- Vừa vào trận: hiện bảng mờ (nền 50%) giữa-dưới màn hình liệt kê điều khiển (di chuyển; Thiêu Đốt: Shift / nháy đúp giữ / 🔥; **Kỹ năng: Space / nút kỹ năng** (cần Bí Kíp); M minimap...). Tự mờ dần sau 5s (trùng thời gian bất tử) hoặc biến mất khi người chơi bắt đầu di chuyển.

## 17. Checklist test chấp nhận

- [ ] Không có tên bot nào chứa "bot"; 2 bot trong cùng room không trùng tên; bot chết thì 70% quay lại cùng tên sau 4–12s.
- [ ] Đổi run → chase → flee: thân chạy liên tục không reset frame, chỉ đổi mặt + tư thế; quay trái/phải không lật qua lại khi chạy dọc; tốc độ bước chân khớp tốc độ di chuyển.
- [ ] `assets:report` có điểm độ giật mọi animation; không frame nào vượt ngưỡng trong build thật.

- [ ] Thế giới 5 khu, mỗi khu 12 000²; không đi bộ sang khu khác được; 200 lượt qua cổng: ra đều các cổng còn lại, không bao giờ ra lại cổng vừa vào, ~80% sang khu khác.
- [ ] Người mới vào spawn ở khu ít người nhất; tổng người + bot mỗi phòng vẫn 20, bot sinh vào khu ít nhân vật nhất.
- [ ] Băng Nguyệt Hồ: trên băng trượt có quán tính.
- [ ] 4 khu (trừ Thiên Đình) có Thiên Môn ở tâm, đường tới thông; bước vào luôn ra ở đúng Bến Thiên Môn của khu đó ở Thiên Đình; minimap có icon + mũi tên chỉ hướng.
- [ ] Đi sát/đi dọc tường, rào, đá tảng: không giật, không rung; F3 'hiệu chỉnh/giây' ≈ 0 khi chỉ đi sát vật cản.
- [ ] Không có Bí Kíp thì nút kỹ năng xám; nhặt Bí Kíp → +1 lượt (tối đa 2), dùng đúng kỹ năng của nhân vật mình; chết thì mất lượt; kỹ năng không làm chém thân người còn ≥ 2 vòng.
- [ ] Trúng Phun Trân Châu: màn hình bị che ~60% trong 5s, không nhặt được gì trong 5s; HUD và minimap không bị che.
- [ ] Shipper dùng Giao Hỏa Tốc đi xuyên vách đá được nhưng không ra khỏi biên khu; hết giờ trong vật cản thì được đẩy ra chỗ trống.

- [ ] Hai Kim Đan 60 kiếm mài nhau 1 giây → mỗi bên còn ~30 kiếm; đi xuyên qua nhau được; chỉ người còn 1 vòng mới bị chém thân.
- [ ] Tốc Kiếm chỉ làm kiếm quay nhanh hơn, tầm chém không đổi.
- [ ] Luyện Khí có Cương Kiếm va Trúc Cơ không buff → Trúc Cơ mất 1 mỗi lần va, Luyện Khí nứt 1/1.25; phá rào tre nhanh gấp đôi.
- [ ] `cmd/loadtest` 40 nhân vật, 10 người Đạo Tổ đủ kiếm: tick < 4 ms, snapshot < 8 KB, FPS ≥ 60 trên PC tầm trung, ping tăng < 20 ms.
- [ ] Không còn vật phẩm Trường Kiếm trên bản đồ, HUD, config, debug (phím 1–7 bỏ buff này).
- [ ] Giữ Shift: chạy ×1.8, mất 8% EXP tầng/giây, không bao giờ rớt cảnh giới.
- [ ] Đài nhỏ phun mỗi 5s, đợt lớn mỗi 30s; Thiên Tiên Đài báo trước 10s và phun viên 150 điểm mỗi 150s.
- [ ] mapgen: 10 seed liên tiếp đều qua kiểm tra liên thông, có Thiên Tiên Đài, 4 sân môn phái, đường chính không vật cản.
- [ ] `?debug=true` mới dùng được phím debug; `B` thả bot.

- [ ] Luyện Khí tầng 1 nhỏ hơn rõ ràng so với Luyện Khí tầng 10; Đại Thừa tầng 10 scale = 5.
- [ ] Kiếm dài ra và vòng rộng ra theo scale.
- [ ] Cùng 1 tầng: 1 → 10 kiếm, bán kính vòng không đổi; kiếm thứ 11 mới mở vòng 2 rộng hơn.
- [ ] Luyện Khí tối đa 10 kiếm = 1 vòng; Trúc Cơ 30 = 2 vòng (10 + 20); Kim Đan 60 = 3 vòng; các vòng so le, mỗi vòng một màu, đều nhìn thấy; lưỡi nghiêng về phía sau và lật khi đổi chiều.
- [ ] Mũi kiếm vòng 3 xa tâm hơn vòng 1; hitbox server khớp với hình vẽ (bật chế độ debug vẽ capsule để so).
- [ ] Kiếm quay đúng 1 vòng / 1 giây (0.625 giây khi có Tốc Kiếm), nhìn rõ từng thanh, không bị hiệu ứng quay ngược.
- [ ] Nhân vật đi sát mép trên tảng đá được (chỉ bị chặn ở chân đế), đứng sau đá thì bị che.
- [ ] Farm không đánh nhau: Luyện Khí → Trúc Cơ trong 30–60s (đo bằng cmd/sim).
- [ ] 2 người cùng cảnh giới va kiếm → mỗi bên mất 1 kiếm, cả hai đổi chiều quay.
- [ ] Trúc Cơ va Luyện Khí: sau 12 lần va, Luyện Khí mất 12 kiếm, Trúc Cơ mất 10; **không ai đổi chiều**.
- [ ] Nguyên Anh va Luyện Khí: Luyện Khí mất 1 mỗi lần va, Nguyên Anh mất 1 sau mỗi ≈1.73 lần (crack cộng 1/1.728).
- [ ] Người 0 kiếm vẫn sống; bị chém thân thì rớt cảnh giới (Luyện Khí thì chết).
- [ ] Bị chém ở Trúc Cơ tầng 2 → Luyện Khí tầng 1 thanh 0%; Kim Đan tầng 7 → Trúc Cơ tầng 1 thanh 0%; kiếm vượt trần rơi ra thành bó.
- [ ] Bị chém ở Luyện Khí (tầng bất kỳ) → màn hình Tổng kết, bấm "Chơi tiếp" vào lại được; người chém nhận 50% EXP nạn nhân + 30; xác rơi 50% số kiếm.
- [ ] Viên EXP từ quái chỉ người chém nhặt được trong 1.5s đầu.
- [ ] Đi chéo 90° so với gió → tốc độ không đổi.
- [ ] Vật cản bị chém vỡ → đi qua được → 30s sau mọc lại, không kẹt người.
- [ ] Kiếm chém vật cản không bị mất.
- [ ] Luyện Khí đủ kiếm phá rào tre ≈ 15s; Đại Thừa phá đá tảng ≈ 9s.
- [ ] Vào cổng → ra ở cổng khác, cổng đích lóe sáng trước, không có bất tử khi ra.
- [ ] Người scale lớn không chui lọt hẻm 150.
- [ ] Buff hết hạn đúng giờ, nhặt lại thì làm mới thời gian.
- [ ] Ẩn Thân: người khác thấy mờ 20%, không tên; tối đa 5s; chém trúng người là hiện ngay.
- [ ] Trúc Cơ chém trúng Nguyên Anh → +800 Uy Danh; chém lại cùng người trong 60s → +0.
- [ ] Bảng Thần Tốc chỉ có những mạng đã đạt Đại Thừa.
- [ ] Đại Thừa tầng 10 chưa đầy thanh: nhặt Linh khí bình thường, không nhặt được Tiên khí. Đầy 694 960: nhặt 1 Tiên khí → Chân Tiên; 101 → Kim Tiên; 1 001 → Đạo Tổ. Người dưới Đại Thừa 10 đi qua viên Tiên khí không nhặt được (viên mờ 30%); người Tiên giới đi qua viên Linh khí không nhặt được.
- [ ] Đột phá Kim Tiên: rồng vàng chui lên từ chân, quấn quanh người 1.5 vòng (nửa sau khuất sau nhân vật), bay vút lên, 2.5s; sau đó rồng nhỏ (60%) luôn lượn lờ quanh người, mỗi 8–15s làm 1 động tác ngẫu nhiên, khi chạy thì bị kéo dài phía sau.
- [ ] Lên Chân Tiên thấy hào quang trắng bạc sau lưng; mỗi cảnh giới Tiên một màu; rớt về Đại Thừa thì tắt hào quang.
- [ ] Thái Ất đang 450 bị chém → Kim Tiên, Tiên khí 101; người chém (Tiên giới) +174, rơi 104 điểm viên Tiên khí. Chân Tiên bị chém → Đại Thừa tầng 1 thanh 0% (346 800), Tiên khí 0.
- [ ] Chém người Tiên giới (người chém là phàm, Tiên hoặc bot đều được): luôn có viên Tiên khí vàng văng ra quanh xác, tổng = 30% Tiên khí nạn nhân mất (Chân Tiên 80 Tiên khí bị chém → rơi 24).
- [ ] Đạo Tổ: nhặt Tiên khí khi chưa đủ 1 500 → nạp thanh, không cộng Uy Danh; đủ 1 500 → +1 Uy Danh mỗi 10 Tiên khí; Thiêu Đốt kéo thanh xuống, phải nạp lại 1 500 mới đổi tiếp.
- [ ] Không còn vùng nền/bùn/bụi/đường nào là hình chữ nhật thẳng cạnh hay hình tròn hoàn hảo; ranh giới biome loang tự nhiên; đứng sát mép bùn thì hiệu ứng chậm khớp đúng mép đa giác.
- [ ] HUD góc trên trái là chân dung nhân vật, viền màu cảnh giới, biểu cảm đổi theo trạng thái.
- [ ] Bảng tên đọc rõ trên nền vòng kiếm nhiều màu; tên mình màu vàng có ▼; người cao cảnh giới hơn mình có ▲ đỏ; Top 1 có ruy băng đỏ-vàng + vương miện + vệt sáng lướt.
- [ ] Top 1/2/3 phòng hiện đúng danh hiệu trên đầu và trong bảng; đổi hạng phải giữ 3s mới đổi danh hiệu.
- [ ] Minimap thấy mình, khung camera, cổng, Top 1, Rương Báu; không thấy người khác; Top 1 vào bụi thì biến khỏi minimap.
- [ ] Phòng 5 người → 15 bot; thêm người không xóa bot; người thứ 21 sang phòng mới.
- [ ] Leaderboard ghi đúng 3 bảng sau khi chết, bot không có trong bảng toàn cục.
