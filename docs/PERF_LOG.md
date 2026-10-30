# PERF_LOG — Sổ số liệu hiệu năng qua từng lượt tối ưu

> Đặt tại `docs/PERF_LOG.md`. **Agent cập nhật sau MỖI lượt tối ưu** (và sau mọi thay đổi có thể ảnh hưởng hiệu năng: VFX, bản đồ, kỹ năng...). Dữ liệu thô dạng máy đọc được ghi song song vào `docs/perf_log.csv`.
> Quy tắc vàng: **chỉ so sánh các lần đo cùng kịch bản** (cột `Kịch bản`). Số từ kịch bản khác nhau không được đặt cạnh nhau để kết luận.

---

## 1. Kịch bản đo chuẩn (không được đổi; muốn đổi thì tạo kịch bản mới)

| Mã | Lệnh | Mô tả |
|---|---|---|
| **S1** | `go run ./cmd/loadtest --scenario S1 --seed 7` | 40 nhân vật, 10 Đạo Tổ đủ kiếm đứng chụm 1 khu, 120s. Đo server + mạng. |
| **S2** | `node tools/browsertest --scenario S2 --net normal` | 10 tab Chrome headless, rải 5 khu, 5 tab lên Đạo Tổ, 60s chụm + 60s tản. Đo mạng + ổn định hình. |
| **S3** | `node tools/browsertest --scenario S2 --net slow4g` | Như S2 nhưng giả lập 4G yếu (100 ms, 1.6 Mbps / 750 Kbps). |
| **S4** | Chơi tay 1 tab có giao diện, `?debug=true`, máy chuẩn (ghi cấu hình ở §4) | Đứng giữa trận đông nhất 60s, đọc F3. Đo FPS thật. |
| **S5** | `go run ./cmd/sim --bots 40 --minutes 5 --seed 7` | Nhịp chơi (không phải hiệu năng): lượt chết, thời gian lên cảnh giới. |

Mỗi kịch bản chạy **3 lần**, ghi trung vị. Cùng máy, cùng commit build, đóng ứng dụng nặng khác.

## 2. Bộ chỉ số (cột cố định)

| Nhóm | Chỉ số | Đơn vị | Ngân sách | Nguồn |
|---|---|---|---|---|
| Server | Tick TB / p95 (khu nặng nhất) | ms | < 3 / < 5 | loadtest, F3 |
| Server | CPU process | % 1 lõi | – | loadtest |
| Mạng | Băng thông xuống TB / đỉnh mỗi client | KB/s | < 25 / < 40 | browsertest, F3 |
| Mạng | Gói lớn nhất | KB | < 6 | browsertest, F3 |
| Mạng | Ping p95 (S3) | ms | tăng < 20 so với phòng trống | browsertest |
| Client | FPS TB / 1% thấp nhất | fps | ≥ 60 / ≥ 45 | F3 (S4) |
| Client | Thời gian vẽ / frame | ms | < 4 | F3 |
| Client | Draw call / Sprite | số | < 200 / – | F3 |
| Client | Bộ nhớ texture | MB | < 300 | F3 |
| Hình ảnh | Khung có placeholder | số | **0** | F3 |
| Hình ảnh | Entity respawn trong 2s | số / phút | **0** | F3 |
| Hình ảnh | Đổi khu (cache / chưa cache) | ms | < 300 / < 1 500 | browsertest |
| Nhịp chơi | Lượt chết / bot / 5 phút | số | (theo dõi) | sim (S5) |

## 3. Bảng tổng hợp theo lượt

Mỗi lượt 1 dòng **cho mỗi kịch bản đã chạy**. Ô đạt ngân sách ghi ✅, vượt ghi ❌, không đo ghi `–`. Cột `Δ` so với **lượt trước cùng kịch bản**.

| # | Ngày | Commit | Lượt / thay đổi | Kịch bản | Tick TB / p95 (ms) | KB/s TB / đỉnh | Gói max (KB) | FPS TB / 1% | Vẽ (ms) | Draw call | Placeholder | Respawn/phút | Δ chính | Nút thắt còn lại |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 2026-10-2x | ? | **Mốc trước tối ưu** | S1 | 11.45 / 19.89 ❌ | 43.1 / – ❌ | 38.3 ❌ | – | – | – | – | – | – | ItemsAdded dồn 1 gói khi camera Đạo Tổ lùi xa; server tick vượt 3–5× |
| 0b | 2026-10-2x | ? | Mốc trước tối ưu (game thật 9 người, 1 912 entity) | S4\* | 3.40 / – | 61.7 / – ❌ | 3.0 | 60 / – ✅ | 1.0 ✅ | 245 ❌ | ? | ? | – | Băng thông; draw call |
| 1 | 2026-10-2x | ? | v4.9 bản đồ sinh động — bản đầu | S4\* | – | – | – | 36 / – ❌ | – | 611 ❌ | – | – | FPS 58 → 36 | Decal/hạt là Graphics riêng; duyệt 2 000 decal/frame; bóng vật cản Graphics |
| 2 | 2026-10-2x | ? | v4.9 — bake decal, gom chunk 2048, bake bóng | S4\* | – | – | – | ~58 / – | 0.9 ✅ | 118 ✅ | – | – | Draw call 611 → 118 | – |
| 3 | 2026-10-2x | ? | Sửa lỗi `geom.SegSegDist2` (sai dấu) | S5 | – | – | – | – | – | – | – | – | Lượt chết 304 → 84 / 5 phút (luật đúng) | Số đo nhịp cũ không còn giá trị |
| **4** | 2026-10-30 | `f5720c0` | **Mốc chuẩn** — sau tối ưu mạng (mục 27) + server (mục 26) | S1 | 1.56 ✅ / 3.33 ✅ | 8.6 / – ✅ | 6.8 ❌ | – | – | – | – | – | Tick 11.45 → 1.56 · gói 38.3 → 6.8 KB | Gói còn 6.8 KB vs ngân sách mới 6 KB; cần Protocol v2 nhị phân |
| 4 | 2026-10-30 | `f5720c0` | Mốc chuẩn (client, mạng thường) | S2 | 0.7 / 1.6 ✅ | 19.5 / 45.5 ❌ | 2.5 ✅ | 7.6 / 3.6 ⚠ | 2.5 ✅ | 0 ⚠ | chưa đo | chưa đo | — | Băng thông đỉnh 45.5 KB/s ở tab Đạo Tổ |
| 4 | 2026-10-30 | `f5720c0` | Mốc chuẩn (client, 4G yếu) | S3 | 0.3 / 0.5 ✅ | 7.7 / 13.6 ✅ | 0.6 ✅ | – | – | – | chưa đo | chưa đo | — | **Cột Đạo Tổ rỗng** — xem ghi chú dưới |
| 4 | 2026-10-30 | `f5720c0` | Mốc chuẩn (nhịp chơi) | S5 | – | – | – | – | – | – | – | – | Lượt chết 2.10 → **4.70** / bot / 5 phút | Do luật v5.8 (bị chém về sàn cảnh giới) |

\* Dòng có dấu `*`: đo trước khi chuẩn hóa kịch bản, chỉ để tham khảo, **không dùng làm mốc so sánh**. Lượt tối ưu kế tiếp phải chạy lại đủ S1–S4 để tạo **mốc chuẩn mới (#4)**.

## 4. Máy đo chuẩn

| Vai trò | Cấu hình |
|---|---|
| Server / loadtest | Intel Core i5-12400F (6 nhân / 12 luồng), 32 GB RAM, Linux Mint 22.3 (nhân 7.0.0) |
| Client S2–S3 (headless) | Cùng máy trên · Chromium 156.0.8078.4 (Playwright) · 10 tab 960×540 |
| Client S4 (có giao diện) | Cùng máy trên · AMD Radeon RX 570 (RADV) · Chromium 156 · 1 tab 1280×720 |
| Điện thoại (khi có) | _(chưa đo)_ |

**Hai điều phải nhớ khi đọc số S2/S3**, nếu không sẽ kết luận sai:

1. **Chrome headless vẽ WebGL bằng CPU (SwiftShader) nếu không ép dùng GPU.** Mười tab
   SwiftShader tranh nhau 12 luồng thì chính việc vẽ trở thành nút thắt, và FPS trong S2/S3
   **không nói gì về máy người chơi**. Các kịch bản ở đây chạy với
   `--use-gl=angle --use-angle=gl` để dùng card thật; đo được `--use-angle=vulkan` → RADV,
   `--use-gl=swiftshader` và `--use-gl=egl` đều rơi về CPU.
2. **Server phải chạy tách lõi khỏi trình duyệt.** Không tách thì tick 2,55 ms bị thổi lên
   **269 ms** chỉ vì goroutine bị hệ điều hành gạt ra liên tục — số đo thành vô nghĩa.
   S1 chạy một mình nên không dính; S2/S3 dùng `taskset`.

FPS thật của máy người chơi chỉ lấy từ **S4** (một tab, có giao diện, GPU thật).

## 5. Chi tiết từng lượt

### Lượt #4 — Mốc chuẩn sau tối ưu mục 26+27 (2026-10-30, commit `f5720c0`)

Mục tiêu: tạo **mốc chuẩn** để mọi lượt NETCODE v6 so vào (CHANGES_v6 mục 7). Các dòng #0–#3
đo trước khi chuẩn hoá kịch bản nên không dùng để so nữa.

Thay đổi so với #0: tối ưu mạng (CHANGES_v4 mục 27) + server (mục 26) — xem commit `d132a60`.

Kết quả S1 (trung vị 3 lần, `go run ./cmd/perfreport`):

| Chỉ số | Trước (#0) | Sau (#4) | Δ | Ngân sách §2 |
|---|---|---|---|---|
| Tick TB | 11.45 ms | **1.56 ms** | −86% | < 3 ✅ |
| Tick p95 | 19.89 ms | **3.33 ms** | −83% | < 5 ✅ |
| Gói lớn nhất | 38.3 KB | **6.8 KB** | −82% | < 6 ❌ |
| Băng thông xuống | 43.1 KB/s | **8.6 KB/s** | −80% | < 25 ✅ |
| CPU process | – | 5% một lõi | – | – |

Nút thắt tìm được (có bằng chứng, xem commit `d132a60`):
1. `ItemPicked` — **173 sự kiện / 8.1 KB / 74% gói** (mổ gói bằng `cmd/loadtest`) — đã sửa.
2. `sparseRingsTouch` — **57% CPU** (pprof) — đã sửa bằng cách chỉ xét vòng ngoài cùng.
3. `expireItems` — **21% CPU** (pprof) — đã sửa bằng danh sách viên có hạn riêng.

**Chưa đo được, và vì sao** (không ghi bừa vào bảng):
- **S4 (FPS thật)**: cần một tab có giao diện, người ngồi xem 60 giây. Tôi không chạy được
  kịch bản này; các cột FPS phải do chủ dự án chạy tay rồi điền.
- **S3 cột Đạo Tổ rỗng**: dưới 4G yếu + 10 tab, cả 5 tab cần nâng lên Đạo Tổ đều mất ngữ
  cảnh WebGL (`PixiJS Error: Could not initialize shader`), `glrecover` tải lại trang, nên
  lúc nâng cấp thì `window.kx` chưa có → 0/5 tab lên được Đạo Tổ. **Số S3 hiện chỉ phản ánh
  tab thường.** Đây là lỗi của BÀI ĐO, không phải của game.
- **Draw call ở S2 ra 0.0**: bộ đếm draw call không hoạt động ở cấu hình headless đang dùng.
  Không được đọc "0 draw call" là tốt — nó là **số không đo được**.

Tác dụng phụ đã biết:
- **Nhịp chơi đổi**: lượt chết 2.10 → **4.70 / bot / 5 phút** (S5). Không phải hồi quy hiệu
  năng — là hệ quả của luật v5.8 (bị chém thì thanh về sàn cảnh giới, nên nhiều người rơi
  về Luyện Khí, nơi bị chém là chết). Ghi lại vì quy tắc §7.4 đòi ghi cả chỉ số xấu đi.

Việc tiếp theo đề xuất:
1. Sửa bài đo trước khi làm tiếp: S3 phải nâng được Đạo Tổ, và bộ đếm draw call phải chạy.
   Đo bằng một thước hỏng thì mọi kết luận sau đó đều không tin được.
2. NETCODE bước 2 (chống nhấp nháy) — xem `docs/NETCODE_PLAN.md`.
3. Gói 6.8 KB vs ngân sách mới 6 KB: cần Protocol v2 nhị phân (bước 3), msgpack còn tên
   trường nên vẫn tốn ~35% mỗi entity.

---

### Mẫu — copy cho mỗi lượt

```
### Lượt #N — <tên ngắn> (YYYY-MM-DD, commit abc1234)

Mục tiêu: <chỉ số muốn giảm, vì sao>
Thay đổi: <liệt kê ngắn, link mục spec / CHANGES>

Kết quả (trung vị 3 lần):
| Chỉ số | Trước (#M) | Sau | Δ | Ngân sách |
|---|---|---|---|---|
| ... | | | | |

Nút thắt tìm được (top 3, có bằng chứng):
1. <hàm / hệ thống> — <số đo: % CPU từ pprof, ms, KB, draw call> — <file:dòng>
2. ...

Bị chặn / chưa làm được:
- <việc> — <lý do: thiếu asset, rủi ro, cần anh quyết...>

Tác dụng phụ / rủi ro: <hình ảnh, gameplay, test nào bị ảnh hưởng>
Việc tiếp theo đề xuất: <1–3 việc, xếp theo lợi ích / công sức>
```

## 6. Nút thắt đang mở (cập nhật liên tục)

| # | Nút thắt | Ảnh hưởng | Bằng chứng | Trạng thái | Lượt xử lý |
|---|---|---|---|---|---|
| B1 | ItemsAdded dồn 1 gói khi camera lùi xa | Gói 38 KB, nghẽn mạng yếu | Lượt #0 | **Đóng** — gói 38.3 → 6.8 KB | #4 |
| B2 | Server tick vượt ngân sách khi 10 Đạo Tổ chụm | Lag server | Lượt #0 | **Đóng** — tick 11.45 → 1.56 ms | #4 |
| B3 | Nhấp nháy + placeholder khi entity vào khung | Hình ảnh | Đo: **8/8 lần respawn trong 2s**, 1/3 nhân vật placeholder (`tools/browsertest/flicker.mjs`) | Mở — đã tìm ra nguyên nhân, xem `docs/NETCODE_PLAN.md` | – |
| B4 | Băng thông game thật 61.7 KB/s | Mạng yếu | Lượt #0b | Mở — S2 còn **45.5 KB/s đỉnh** ở tab Đạo Tổ (ngân sách 40) | – |
| B5 | Gói snapshot 6.8 KB vs ngân sách mới 6 KB | Mạng yếu | Lượt #4 | Mở — cần Protocol v2 nhị phân (NETCODE bước 3) | – |
| **B6** | **Bài đo S3 không nâng được Đạo Tổ**; draw call đọc ra 0 | Không đánh giá được kịch bản nặng trên mạng yếu | Lượt #4 | Mở — **phải sửa trước khi tin số S3** | – |

## 7. Quy tắc cho agent
1. **Đo trước, rồi mới sửa.** Mỗi lượt tối ưu bắt đầu bằng 1 dòng "Trước" (hoặc dùng dòng mốc gần nhất cùng kịch bản nếu chưa đổi code).
2. Sau mỗi lượt: thêm dòng vào §3, thêm khối chi tiết §5, cập nhật §6, ghi dòng CSV vào `docs/perf_log.csv`.
3. Nút thắt phải có **bằng chứng số** (pprof top 10, F3, kích thước gói) — không ghi phỏng đoán.
4. Lượt làm **xấu đi** chỉ số nào (dù vì lý do đúng, như thêm VFX) vẫn phải ghi rõ, kèm lý do chấp nhận.
5. Viết lệnh `npm run perf:report` (hoặc `go run ./cmd/perfreport`) chạy S1–S3 và in sẵn 1 dòng bảng §3 + 1 dòng CSV, để việc ghi sổ không phụ thuộc chép tay.

## 8. Định dạng CSV (`docs/perf_log.csv`)
```
run,date,commit,label,scenario,tick_avg_ms,tick_p95_ms,cpu_pct,kbps_avg,kbps_peak,pkt_max_kb,ping_p95_ms,fps_avg,fps_low1,draw_ms,draw_calls,sprites,tex_mb,placeholder_frames,respawn_per_min,zone_switch_cached_ms,zone_switch_cold_ms,deaths_per_bot_5min,notes
```
