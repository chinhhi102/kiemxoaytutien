# Việc cần làm — v6.0: Tối ưu giao thức & cache (NETCODE_SPEC.md)

Anh đã đặt `docs/_raw/NETCODE_SPEC.md`. Đây là thay đổi kiến trúc lớn — làm **từng bước theo NETCODE_SPEC §8**, mỗi bước: đo trước/sau (F3 + loadtest + 10 tab headless), test + build pass, commit riêng, ghi PROGRESS.md. Không gộp nhiều bước vào 1 commit.

1. Đọc toàn bộ NETCODE_SPEC. Trước khi code, gửi tôi: (a) nguyên nhân thật của nhấp nháy + placeholder tìm được trong code hiện tại, (b) kế hoạch chia nhỏ bước 1–7 theo code hiện có, (c) chỗ nào trong spec không khớp với code / thấy không hợp lý.
2. Bước 1 → 7 theo NETCODE_SPEC §8. **Bước 2 (chống nhấp nháy & placeholder) có thể làm trước bước 1** nếu nhỏ và độc lập — em tự quyết, báo lý do.
3. Protocol v2 chạy song song v1 sau cờ cho tới khi qua hết bài kiểm tra §7, rồi mới xóa v1.
4. Bài kiểm tra §7 thành test tự động hoặc kịch bản `tools/browsertest` chạy được bằng 1 lệnh.
5. CI/build: chặn build thật nếu thiếu asset bắt buộc (§5 "Hiển thị" mục 1).

## Sổ số liệu hiệu năng
6. Đặt `docs/PERF_LOG.md` + `docs/perf_log.csv` (anh gửi kèm). Điền máy đo chuẩn (§4). Viết `npm run perf:report` (hoặc `go run ./cmd/perfreport`) chạy S1–S3, in sẵn 1 dòng bảng + 1 dòng CSV.
7. **Trước khi bắt đầu NETCODE bước 1**: chạy đủ S1–S5 trên code hiện tại → dòng #4 "Mốc chuẩn" (thay cho các dòng có dấu *). Mọi lượt sau so với mốc này.
8. Mỗi bước NETCODE §8 = 1 lượt trong PERF_LOG, có khối chi tiết §5 (trước/sau, nút thắt có bằng chứng, bị chặn ở đâu, việc tiếp theo).
9. CLAUDE.md đã thêm quy tắc 12 (anh gửi kèm bản mới).
