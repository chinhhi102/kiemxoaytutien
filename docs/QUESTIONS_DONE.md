# Câu hỏi đã xong (lưu trữ)

Những câu đã được trả lời — giữ lại để tra sau, không cần đọc nữa.
Câu **đang chờ** nằm ở `docs/QUESTIONS.md`.

> Mỗi câu đúng **một dòng**. Trước đây Q7–Q14 nằm cả trong bảng lẫn trong phần "Chi tiết"
> bên dưới, nên đọc file này là đọc hai lần cùng một nội dung. Cần chi tiết đầy đủ thì tra
> `docs/_raw/ANSWERS.md` (lời chủ dự án) và `docs/PROGRESS.md` (việc đã làm).

| Câu | Nội dung | Kết luận |
|---|---|---|
| Q1–Q6 | Thưởng hạ sát, gió theo góc, viên EXP thuộc người chém, chết rơi kiếm, 1 anim/ảnh | Trả lời trong `_raw/ANSWERS.md`, đã áp dụng |
| Q7 | Nhịp Luyện Khí → Trúc Cơ | Chốt 30–60s; sau đó đổi lại, xem Q13 |
| Q8 | Checklist §17 ghi số kiếm mất theo bản cũ | Spec sửa: `clash_unit` = 1, checklist khớp |
| Q9 | `min_count = 0` có làm chết không | Không. 0 kiếm chỉ bị tước vũ khí; chỉ Luyện Khí bị chém thân mới chết |
| Q10 | Phạm vi nhặt vật phẩm | Giữ `pickup_r` = 90. Luật vốn đã đúng: kiếm không nhặt được đồ |
| Q11 | "Tính exp dư ra làm…" (câu bị cắt) | §3.2: Đại Thừa 10 thì EXP thừa bị bỏ, xếp hạng dùng Uy Danh |
| Q12 | Tốc độ chạy 130 hay 260×√scale | Gộp vào Q14; code giữ 130, sau đó Q13 đổi lên 195 |
| Q13 | Nhịp farm quá chậm | Chấp nhận nhịp hiện tại — lên cấp chủ yếu nhờ cướp Linh khí. `move_base` → 195, `exp_orbs.cell` → 140 |
| Q14 | `_raw` thiếu 13 mục so với code | Spec v4.0 gộp đủ; `realm_mult` giữ 2.0 |
| Q15 | Lá chắn §5.2 làm không ai chém chết được ai | Hiểu nhầm lá chắn thành tường chắn đường. Chuyển thành **luật chém**: người chơi không chặn đường nhau, còn 1 vòng mới bị chém thân |
| Q16 | Trường Kiếm | Bỏ hẳn (v4.3); v4.4 thay bằng **Cương Kiếm** |
| Q17 | Bó Kiếm khi rớt cảnh giới | Phương án 1 — Bó Kiếm chủ yếu từ xác người chết và vật phẩm bản đồ |
| Q18 | Linh khí / Tiên khí | Tách hẳn, không quy đổi qua lại. Bỏ `tien_khi_to_exp_base` |
| Q19 | Tốc độ bể kiếm | Giữ `clash_cooldown: 0`, `flip_cooldown: 0.3` |
| Q20 | Sửa lỗi va kiếm làm trận chậm hẳn, có chỉnh cân bằng không | **Phương án 1** — giữ luật đúng. Mốc mới: 84 lượt chết / 5 phút / 40 bot; số cũ (608/10 phút) đo trên luật sai. Chơi thử rồi mới quyết; nếu chỉnh thì ưu tiên tăng `sword.width` |
| Q21 | Cổng ngẫu nhiên + Đạo Tổ đổi Tiên khí dư | Cổng là **đích ngẫu nhiên** (không phải cổng tự đổi chỗ), §8.4 viết lại thành mạng chung 20 cổng. Đạo Tổ đổi dư sang Uy Danh, ghi vào §3.4 |
| Q22 | `tien_overflow_score`: 1.0 hay 0.1 | **0.1** (10 Tiên khí = 1 Uy Danh) |
| Q23 | Spec còn ghi cách bật Thiêu Đốt cũ | Spec v5.6 §4.1c thêm "nháy đúp rồi giữ / chạm 2 lần rồi giữ"; §18.1 thêm dòng Kỹ năng |
| Q24 | Lên Tiên giới là mất Thiêu Đốt | **Phương án 2** — Tiên giới đốt **Tiên khí**, 8% mỗi giây, không dưới đáy cảnh giới (spec v5.6, sau đó v5.7 đổi mốc 8% sang độ rộng khoảng) |
| Q25 | Bảng tỉ lệ thắng 10×10 sau v5.4/v5.5 | **Bỏ hẳn cách cân bằng theo tỉ lệ thắng 1-đấu-1** — kỹ năng có loại tăng sức mạnh, có loại khắc chế, tay đôi không phản ánh giá trị. Đã xoá `cmd/sim --duel`; cân bằng theo phản hồi người chơi + số liệu trận thật (§6.3) |
| Q26 | Đạo Tổ vẫn không Thiêu Đốt được | Thay toàn bộ cách tính Tiên giới (spec v5.7): cảnh giới theo **tổng Tiên khí cộng dồn**, Đạo Tổ nạp 1 001 → 1 500 và chỉ đổi Uy Danh khi đang đúng 1 500 → Đạo Tổ Thiêu Đốt được |

## Hai việc chờ quyết, nay đã có

- **Tối ưu (CHANGES_v4 mục 25–27)**: **đi tiếp**. Thứ tự chủ dự án chốt: mạng (27) → server
  (26, pprof) → client phần rẻ (25.1 vòng kiếm thành texture, BitmapText, cull). Đo lại
  `loadtest` + 10 tab Chrome headless (thường và 4G yếu), ghi số trước/sau.
- **Thư viện `pixi-filters` và `revolt-fx`**: **đã duyệt**. Quy tắc chung đã thêm vào
  `CLAUDE.md`: thư viện spec ghi đích danh coi như đã duyệt, thư viện khác vẫn hỏi.
