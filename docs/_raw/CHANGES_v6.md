# Việc cần làm — v6.0: Tối ưu giao thức & cache (NETCODE_SPEC.md)

Anh đã đặt `docs/_raw/NETCODE_SPEC.md`. Đây là thay đổi kiến trúc lớn — làm **từng bước theo NETCODE_SPEC §8**, mỗi bước: đo trước/sau (F3 + loadtest + 10 tab headless), test + build pass, commit riêng, ghi PROGRESS.md. Không gộp nhiều bước vào 1 commit.

1. Đọc toàn bộ NETCODE_SPEC. Trước khi code, gửi tôi: (a) nguyên nhân thật của nhấp nháy + placeholder tìm được trong code hiện tại, (b) kế hoạch chia nhỏ bước 1–7 theo code hiện có, (c) chỗ nào trong spec không khớp với code / thấy không hợp lý.
2. Bước 1 → 7 theo NETCODE_SPEC §8. **Bước 2 (chống nhấp nháy & placeholder) có thể làm trước bước 1** nếu nhỏ và độc lập — em tự quyết, báo lý do.
3. Protocol v2 chạy song song v1 sau cờ cho tới khi qua hết bài kiểm tra §7, rồi mới xóa v1.
4. Bài kiểm tra §7 thành test tự động hoặc kịch bản `tools/browsertest` chạy được bằng 1 lệnh.
5. CI/build: chặn build thật nếu thiếu asset bắt buộc (§5 "Hiển thị" mục 1).
