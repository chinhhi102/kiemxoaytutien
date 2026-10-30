# Việc cần làm — v6.1: Tên bot + Hoạt ảnh liền mạch

Anh gửi kèm `config/bot_names.txt` (1 000 tên). Spec: GAME_SPEC §9.4, §12.5; ART_PROMPTS §4.3b.

## Tên bot (§9.4)
1. Thay nguồn tên bot bằng `config/bot_names.txt`. Rút không lặp trong room, không trùng tên người/bot hiện có hoặc vừa rời 10 phút; 15% thêm biến thể nhỏ.
2. Bot chết: 70% quay lại cùng tên sau 4–12s, 30% rời phòng và bot mới (tên mới) vào sau 5–20s; 5% đứng yên 3–8s sau khi vào. Phân bố nhân vật bot theo thống kê người chơi.
3. Bảng tên: tên có ký tự ngoài BitmapFont → font dự phòng, không hiện ô vuông. Kiểm tra cả 1 000 tên hiển thị đủ dấu.
4. Rà mọi chỗ có thể lộ bot (tiền tố id, màu, hành vi hồi sinh tức thì, kill feed...) và báo tôi danh sách đã sửa.

## Hoạt ảnh (§12.5) — làm theo thứ tự
5. **Trước khi sửa**: thêm phím `J` (debug) + điểm "độ giật" vào `assets:report` cho mọi animation hiện có; gửi tôi bảng điểm để biết nhân vật/anim nào tệ nhất.
6. **C — bộ điều khiển anim** (không cần art mới, làm trước): crossfade 80–120 ms, giữ pha khi đổi mood, fps chu kỳ chạy theo tốc độ thật, lật trái/phải có trễ + co giãn, chuyển động phụ thủ tục (nhún, nghiêng, squash & stretch, thở), anim lấy từ vị trí đã nội suy.
7. **B — chuẩn hóa frame** trong `tools/atlas-pack`: neo chân theo trung điểm 2 chân, cùng tỉ lệ theo trung vị cả animation, neo đầu, palette + viền, điểm độ giật, GIF + onion-skin. Chạy lại cho toàn bộ art hiện có.
8. **A — một thân nhiều mặt**: hỗ trợ lớp mặt chồng lên thân (điểm neo mặt mỗi frame thân); run/chase/flee dùng chung chu kỳ thân + tư thế thủ tục. Trong lúc chưa có art mới: dùng tạm frame `run` hiện có làm thân cho cả 3 mood. Thêm job art mới vào assets:report theo ART_PROMPTS §4.3b.
9. Đo lại điểm độ giật sau mỗi bước, ghi vào PERF_LOG (mục mới "Chất lượng hoạt ảnh"). Gửi tôi GIF trước/sau của 2 nhân vật tệ nhất.
