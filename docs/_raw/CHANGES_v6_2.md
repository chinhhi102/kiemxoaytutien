# Việc cần làm — v6.2: Số liệu người chơi thật (TELEMETRY_SPEC.md)

Anh đặt `docs/_raw/TELEMETRY_SPEC.md`. Làm như **hướng C**, song song với A (netcode) và B (anim) — nhánh riêng `telemetry`, worktree `../kx-telemetry`. Phần này độc lập, nên **merge sớm** để bắt đầu thu số liệu từ người thật trên `192.168.1.74:9999` càng sớm càng tốt (mỗi ngày chậm là mất dữ liệu).

1. Client: module `telemetry/` — phiên, mẫu 10s, sự kiện, sendBeacon khi đóng. **Không** có thông báo hay công tắc (đội test). Đặt sau cờ build `TELEMETRY=on` — build release tắt hẳn. Đo: có/không telemetry FPS chênh ≤ 1.
2. Server: message `Telemetry` + `POST /telemetry/end`, ghi `data/telemetry.db` (WAL, ghi theo lô, ngoài vòng tick); server tự ghi số liệu tick/khu, KB/s, CPU/RAM mỗi 10s.
3. `cmd/perfreport --real` (7 phần ở §4.1) + trang `/admin/perf` (chỉ LAN + mật khẩu env) + **tổng hợp tự động 2 lần/ngày 12:00 & 23:30** ra `docs/perf/real/` (§4.0), không có phiên thì bỏ qua.
4. Thêm S6 vào `docs/PERF_LOG.md` §1; thêm cột `n_sessions` vào CSV.
5. Deploy lên `:9999`, tự chơi thử 2 phút từ máy khác trong LAN để kiểm tra (§6). Báo tôi khi đã bắt đầu thu.
6. Sau đó: mỗi lần deploy build mới lên `:9999` → ghi build id. Đầu mỗi phiên làm việc: chép các dòng S6 tự động mới vào bảng PERF_LOG §3.

Đụng file: telemetry cần hook vào vòng vẽ (FPS) và net (ping/KB) — chỉ **đọc** số liệu, không sửa logic; nếu cần sửa file thuộc hướng A/B thì ghi vào PROGRESS "Đụng file".
