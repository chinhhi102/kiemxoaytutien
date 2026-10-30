# Việc cần làm — spec v5.1

Anh đã dán GAME_SPEC v5.0 vào docs/_raw/. Làm theo thứ tự, mỗi phần xong: test + build pass, ghi PROGRESS.md, đo F3/loadtest nếu ảnh hưởng hiệu năng.

## Trả lời Q21
- **Cổng: đúng là đích ngẫu nhiên** (không phải cổng tự đổi chỗ). Spec §8.4 đã viết lại — giờ là mạng chung 20 cổng của cả thế giới 5 khu.
- **Đạo Tổ Tiên khí dư → Uy Danh**: giữ đúng như em làm (thanh đứng ở 1 500, `tien_overflow_score` 1.0). Đã ghi §3.4.

## 1. Sửa giật khi chạm vật cản (§8.2a) — LÀM TRƯỚC
1. Tái hiện + đo: thêm vào F3 "hiệu chỉnh vị trí/giây"; debug H vẽ footprint client (xanh) và server (đỏ) chồng nhau. Gửi tôi nguyên nhân tìm được.
2. Dùng chung 1 thuật toán va chạm cho server và client dự đoán, footprint tính sẵn sau xoay/lật. Test 1 000 bước ngẫu nhiên sát vật cản: lệch < 0.5.
3. Trượt dọc (bỏ thành phần đâm vào, lặp ≤ 3 lần/tick), không bật ngược. Gộp các đoạn rào/tường liền nhau thành 1 collider. Hiệu chỉnh mượt (< 30 đơn vị kéo dần 100ms).

## 2. Thế giới 5 khu (§8.0, §8.4, §9)
4. Room = 5 khu, mỗi khu 1 goroutine mô phỏng + AOI riêng; chuyển nhân vật giữa khu ở ranh giới tick, giữ nguyên mọi trạng thái.
5. Cổng: 4/khu, đích ngẫu nhiên toàn mạng (`exit_pool: "all"`), không ra lại cổng vào. Màu cổng theo khu.
6. mapgen nhận `zone_type`: khu 1 giữ thuật toán hiện tại; viết 4 thuật toán mới (Linh Sơn Cốc: vách đá chia hẻm núi + đèo + cầu treo; Trúc Lâm Mê Cung: mê cung rặng tre + bụi gấp 3; Phàm Trấn: lưới phố, nhà, chợ; Băng Nguyệt Hồ: hồ băng trơn + đảo + bờ tuyết). Mỗi khu 2 Đài Tiên Khí nhỏ, khu 1 giữ 4 + Thiên Tiên Đài. `cmd/mapgen --zone X --seed N --png` để tôi duyệt từng khu.
7. Băng trơn: vận tốc có quán tính (`ice.friction`, `ice.accel`), bờ tuyết ×0.8 — dùng chung mã với client dự đoán.
8. **Giữ 20 người thật/phòng, tổng người + bot vẫn 20** (như cũ). Người mới spawn vào khu ít người nhất; bot sinh/hồi sinh vào khu ít nhân vật nhất; bot biết dùng cổng.
9. HUD tên khu + chữ lớn khi vào khu mới; minimap theo khu; M hiện danh sách 5 khu + số người; kill feed có tiền tố khu.
10. loadtest: 20 người + bot rải 5 khu, 10 Đạo Tổ — ghi tick ms từng khu.

## 3. Kỹ năng nhân vật qua vật phẩm Bí Kíp (§6.3)
11. Thêm vật phẩm **Bí Kíp** (`skill_scroll`, weight 10): nhặt → +1 lượt kỹ năng của **chính nhân vật mình**, tối đa 2 lượt; dư → +20 Linh khí / +2 Tiên khí. Mất lượt khi chết. **Không có hồi chiêu tự động** — chỉ khóa 3s giữa 2 lần dùng.
12. Khung kỹ năng chung (Space / nút mobile hiện icon + số lượt, xám khi 0), event SkillCast/SkillEffect, server tính hiệu ứng; các hiệu ứng điều khiển (chậm, trói, đẩy, trượt, bịt mắt) dùng chung 1 hệ "trạng thái" có thời hạn. Làm 10 kỹ năng theo bảng.
13. **Thằng Mập — Phun Trân Châu (bịt mắt)**: nón 45° dài 700; người trúng nhận event `Blinded`, client phủ 25–40 viên trân châu + vệt trà sữa lên ~60% màn hình, trượt xuống và tan trong 3.5s; không che HUD/minimap; bot trúng → tầm nhìn 40%. Trúng lại thì làm mới + thêm 10 viên.
14. Bot nhặt Bí Kíp và dùng kỹ năng theo luật đơn giản.
15. Art: placeholder trước; asset (vật phẩm Bí Kíp, anim `skill`, icon, VFX, overlay trân châu, sprite đàn em, art 4 khu mới) vào assets:report theo ART_PROMPTS §11.3, §11.4.

## Còn treo từ trước
- Mục 27 (mạng) + bài test 10 tab Chrome headless trước/sau — vẫn ưu tiên ngay sau phần 1 nếu chưa xong, vì thế giới 5 khu sẽ tăng số người/phòng.

## v5.2 — Thiên Môn (§8.4a)
16. mapgen 4 khu (trừ Thiên Đình): chừa quảng trường bán kính 500 ở tâm, ≥ 2 đường dẫn vào, đặt Thiên Môn. Thiên Đình: 4 Bến Thiên Môn cách tâm ~1 400 ở đầu 4 đường chính, trống bán kính 400.
17. Thiên Môn: một chiều, luôn ra ở bến cố định theo khu (`heaven_gate.map`), cùng luật cổng thường (0.8s, không bất tử khi ra, bến lóe trước 0.5s, cooldown chung 8s).
18. Hiển thị: cổng to + cột sáng thấy từ xa, icon minimap luôn hiện + mũi tên chỉ hướng ở góc minimap. Bot dùng khi ≥ Đại Thừa hoặc Thiên Tiên Đài sắp phun. Kill feed khi Top 3 qua Thiên Môn.

## v5.4 — Chỉnh kỹ năng (§6.3)
19. Đại Ca **Gọi Đàn Em**: 2 đàn em tồn tại 10s; mỗi đàn em có số kiếm = floor(50% số kiếm hiện tại của đại ca), tối thiểu 1, xếp vòng như người chơi, cùng cảnh giới; nhỏ hơn đại ca 30%; không chém thân, không nhặt đồ, không qua cổng; hết kiếm/hết giờ thì biến mất. Đại ca không bị trừ kiếm. Chú ý hiệu năng: đàn em dùng chung đường vẽ vòng kiếm (texture) và va chạm theo vòng như người chơi.
20. Sư Phụ Mì **Lưới Mì**: trói 5s. Bé Tí **Gồng Cơ**: 5s. Cụ Rùa **Rụt Cổ**: 5s.
21. Cập nhật test.

## v5.5 — Chỉnh kỹ năng lần 2 (§6.3)
22. A Hớn: hồi ngay 20% số kiếm tối đa của cảnh giới (không vượt tối đa, lấp vòng trong trước).
23. Bà Tám: đẩy lùi 500. Ông Ba Võng: bán kính 500.
24. Thằng Mập: bịt mắt 5s + không nhặt được gì trong 5s (server bỏ qua nhặt đồ, item đi xuyên), icon 🚫 quanh nhân vật; bot tương tự.
25. Cô Ba: trượt 1s = choáng (không điều khiển, không dùng kỹ năng/Thiêu Đốt, kiếm vẫn quay).
26. Shipper: tốc × 3 trong 5s, xuyên mọi vật cản kể cả không phá được, không xuyên biên khu; hết giờ kẹt trong vật cản → đẩy mượt ra chỗ trống gần nhất 0.3s (dùng chung mã va chạm client/server).
27. Cập nhật test.
