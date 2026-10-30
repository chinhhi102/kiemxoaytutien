# KIẾM XOAY TU TIÊN — Thu thập số liệu từ người chơi thật (Real User Monitoring)

> Đặt tại `docs/_raw/TELEMETRY_SPEC.md`. Bổ sung kịch bản **S6** cho `PERF_LOG.md`.
> Bối cảnh: server LAN `http://192.168.1.74:9999` có người chơi thật vào bất kỳ lúc nào. Mục tiêu: **tự động** ghi lại số liệu hiệu năng thật của họ (máy thật, mạng thật, trận thật) mà không cần ai ngồi đo, rồi phân tích được bằng 1 lệnh.

---

## 1. Nguyên tắc
1. **Thụ động, không ảnh hưởng chơi**: tốn < 0.3 ms/frame ở client, < 1 KB/s mạng, không chặn vòng vẽ.
2. **Tổng hợp tại client, gửi theo lô**: không gửi từng frame; mỗi **10 giây** gửi 1 mẫu đã tổng hợp.
3. **Ẩn danh**: không thu tên thật, IP chỉ dùng để phân biệt mạng (lưu dạng băm), không thu nội dung chat. `guest_id` lưu dạng băm.
4. **Không tin số liệu client cho luật chơi** — chỉ dùng để phân tích.
5. Người chơi trên `:9999` là **đội test** (vào chơi để giúp cải thiện game) → telemetry **luôn bật**, không có thông báo hay công tắc tắt trong game.
6. Telemetry nằm sau cờ build `TELEMETRY=on`. Build test (`:9999`) bật; **bản release là build khác, tắt hẳn** (module không được đóng gói vào).

## 2. Client thu gì

### 2.1 Thông tin phiên (gửi 1 lần khi vào trận)
| Trường | Nguồn |
|---|---|
| `session_id` | Ngẫu nhiên mỗi lần vào trận |
| `guest_hash` | Băm `guest_id` |
| `build`, `proto`, `schema_hash` | Hằng số build |
| Thiết bị | `navigator.userAgent` (rút gọn: trình duyệt + version + OS), `hardwareConcurrency`, `deviceMemory`, màn hình + `devicePixelRatio`, là điện thoại hay PC |
| GPU | `WEBGL_debug_renderer_info` → tên GPU (nếu trình duyệt cho), WebGL1/2, `MAX_TEXTURE_SIZE` |
| Mạng | `navigator.connection.effectiveType` / `downlink` / `rtt` (nếu có) |
| Cài đặt | Mức chất lượng (Cao/Vừa/Thấp, tự hay tay), kích thước khung vẽ thật |

### 2.2 Mẫu 10 giây (trong trận)
| Nhóm | Trường |
|---|---|
| FPS | `fps_avg`, `fps_p1` (1% thấp nhất), `frame_ms_p50/p95/max`, số frame > 33 ms, số frame > 100 ms |
| Luồng chính | Số long task > 50 ms (`PerformanceObserver('longtask')` nếu có), tổng ms GC ước lượng (nếu có) |
| Vẽ | `draw_calls_avg`, `sprites_avg`, `tex_mb`, số texture upload, số khung có placeholder |
| Mạng | `ping_p50/p95/max`, `kbps_in`, `kbps_out`, gói lớn nhất, số gói đến muộn > 150 ms, số lần nghẽn |
| Đồng bộ | Hiệu chỉnh vị trí/giây, entity respawn trong 2s, độ dài bộ đệm nội suy |
| Bối cảnh trận | Khu đang đứng, cảnh giới mình, số kiếm & số vòng của mình, số nhân vật trong khung nhìn, **số người Tiên giới trong khung nhìn**, tổng kiếm đang vẽ, số VFX/thần thú đang chạy, đang Thiêu Đốt/đổi khu không |

### 2.3 Sự kiện (gửi ngay, có giới hạn 20 sự kiện/phút)
`webgl_context_lost` / `restored`, `error` (thông điệp + 5 dòng stack đầu, đã bỏ đường dẫn file máy), `reconnect`, `zone_switch {ms, cached}`, `quality_auto_change {from, to, fps}`, `ws_close {code}`, `tab_hidden/visible` (để loại các đoạn chạy nền khỏi thống kê FPS).

### 2.4 Kết thúc phiên
Khi chết/thoát/đóng tab: gửi tóm tắt phiên (thời lượng, số mẫu, FPS trung vị toàn phiên) bằng `navigator.sendBeacon` tới HTTP `POST /telemetry/end` (để không mất khi đóng tab).

## 3. Truyền & lưu

- Mẫu và sự kiện đi qua **WebSocket hiện có** (message lạnh `Telemetry`, msgpack mảng). Mất kết nối → giữ tối đa 30 mẫu trong RAM, gửi bù khi nối lại.
- Server **không xử lý telemetry trong vòng tick**: đẩy vào channel có đệm → 1 goroutine ghi.
- Lưu: SQLite riêng `data/telemetry.db` (bảng `sessions`, `samples`, `events`) — tách khỏi DB leaderboard. Ghi theo lô mỗi 5s.
- **Server tự ghi kèm** vào cùng DB mỗi 10s: tick ms TB/p95 theo từng khu, số người/bot mỗi khu, KB/s gửi mỗi client, CPU/RAM process → để ghép cùng mốc thời gian với mẫu client.
- Giữ dữ liệu 30 ngày, trần 500 MB (xóa cũ nhất trước).
- Gửi thêm `server_time` vào mỗi mẫu để căn mốc thời gian client–server.

## 4. Phân tích

### 4.0 Tổng hợp tự động 2 lần/ngày
- Thu mẫu liên tục khi có người chơi (không có ai thì không có gì để thu, không tốn gì).
- Server tự chạy tổng hợp **2 lần/ngày: 12:00 và 23:30** (giờ Việt Nam, cấu hình được) cho khoảng thời gian kể từ lần tổng hợp trước.
- Kết quả ghi ra file `docs/perf/real/YYYY-MM-DD_HHMM.md` (đủ 7 phần §4.1) + nối 1 dòng S6 vào `docs/perf_log.csv`.
- **Không có phiên nào trong khoảng đó → không tạo file**, chỉ ghi 1 dòng log "không có người chơi".
- Có thể chạy tay bất kỳ lúc nào bằng lệnh ở §4.1.

### 4.1 Lệnh báo cáo
`go run ./cmd/perfreport --real --since 24h` (hoặc `--since 7d`, `--session <id>`) in ra:

1. **Tổng quan**: số phiên, số người (guest_hash khác nhau), tổng giờ chơi, phân bố thiết bị (PC/điện thoại, GPU, trình duyệt).
2. **Bảng S6** theo cùng cột PERF_LOG §3 (trung vị + p10 xấu nhất theo phiên) → 1 dòng chép thẳng vào PERF_LOG + 1 dòng CSV.
3. **Theo nhóm thiết bị**: FPS trung vị / p1 cho từng nhóm (PC GPU rời, PC GPU tích hợp, điện thoại).
4. **Theo tình huống** (chỗ nào làm tụt FPS): FPS theo số nhân vật trong khung (0–5 / 6–15 / 16+), theo số người Tiên giới trong khung (0 / 1–2 / 3+), theo khu, theo mức chất lượng, lúc đổi khu.
5. **Ghép client–server**: những đoạn FPS client tụt hoặc ping tăng → server tick lúc đó bao nhiêu → kết luận nghẽn ở **client / server / mạng**.
6. **Top 5 phiên tệ nhất** (FPS p1 thấp nhất) kèm bối cảnh và sự kiện (context lost, lỗi...).
7. **Lỗi**: gom theo thông điệp, đếm số lần, số phiên bị ảnh hưởng.

### 4.2 Trang xem nhanh
`http://192.168.1.74:9999/admin/perf` — chỉ mở được từ LAN + mật khẩu trong biến môi trường. Hiện các bảng ở §4.1 + biểu đồ FPS/ping theo thời gian của 24h gần nhất. Không cần đẹp, cần đúng.

### 4.3 Ngưỡng cảnh báo (in đỏ trong báo cáo)
- > 10% mẫu có `fps_p1` < 30 ở bất kỳ nhóm thiết bị nào.
- Bất kỳ khung placeholder hay respawn trong 2s nào ở build thật.
- `webgl_context_lost` > 0.
- Ping p95 > 150 ms trên LAN (LAN mà ping cao = nghẽn client hoặc server, không phải mạng).

## 5. Ghi vào PERF_LOG

Thêm kịch bản:

| Mã | Lệnh | Mô tả |
|---|---|---|
| **S6** | `go run ./cmd/perfreport --real --since 7d` | Số liệu **người chơi thật** trên server LAN `:9999`. Ghi cả số phiên N; N < 5 thì đánh dấu "ít mẫu, tham khảo". |

- S4 (đo tay) vẫn giữ để kiểm soát; S6 là số **đáng tin nhất về trải nghiệm thật**.
- Dòng S6 được tạo **tự động 2 lần/ngày** (§4.0), luôn ghi build id đang chạy. Agent chép dòng mới vào bảng PERF_LOG §3 khi bắt đầu mỗi phiên làm việc. **So S6 trước/sau một build** để biết tối ưu có tác dụng với người thật không.

## 6. Kiểm tra chấp nhận
- [ ] Bật telemetry: FPS không đổi quá 1 fps, KB/s gửi lên < 1.
- [ ] Mở game từ 1 máy khác trong LAN, chơi 2 phút, đóng tab → `perfreport --real --since 1h` thấy 1 phiên với ~12 mẫu + bản tóm tắt.
- [ ] Rút mạng 20s rồi cắm lại → mẫu gửi bù đủ.
- [ ] Build release (`TELEMETRY=off`) không chứa module telemetry, không gửi gì.
- [ ] Tổng hợp tự động chạy đúng 12:00 / 23:30; khoảng không có phiên → không tạo file.
- [ ] Giả lập `webgl_context_lost` → báo cáo hiện sự kiện + in đỏ.
- [ ] Server tắt bất ngờ → DB không hỏng (WAL), khởi động lại vẫn đọc được.
