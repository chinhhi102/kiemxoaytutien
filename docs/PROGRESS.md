# Tiến độ — Kiếm Xoay Tu Tiên

_Cập nhật: 2026-10-22_

## Cách chạy (máy dev)

> Server nghe `:9999` trên **mọi card mạng**, nên máy khác cùng Wi-Fi vào được bằng
> `http://<IP-máy-này>:9999`. Muốn đóng lại, chỉ cho máy này chơi:
> `go run ./cmd/server -addr 127.0.0.1:9999`.
```bash
cd server && go run ./cmd/server          # http://localhost:9999 (đã build client) — đổi cổng: -addr :8090
cd client && npm run dev                  # http://localhost:5173 (tự cập nhật khi sửa code)
cd server && go test ./... && go vet ./...
cd client && npm run build
cd server && go run ./cmd/sim --bots 40 --minutes 10     # giả lập cân bằng
cd server && go run ./cmd/mapgen                         # tạo lại 3 bản đồ (ghi đè!)
cd client && npm run assets:report                       # asset nào còn thiếu (thêm --all để xem cả cái đã có)
cd server && go run ./cmd/artgen status                  # tình trạng theo từng ảnh cần sinh
```
Phím test khi chơi (server chạy mặc định `-debug=true`): **E** +100 EXP · **R** lên cảnh giới · **K** +10 kiếm · **1–7** buff · **X** tự chết · **H** bật/tắt hitbox kiếm của server.
Đo nhịp farm: `cd server && go run ./cmd/sim --bots 20 --minutes 3 --farm` (không ai bị chém; thêm `--orb-cell N`, `--monsters N` để thử).
Giả lập ping: thêm `?lag=150` vào địa chỉ (ví dụ `http://localhost:5173/?lag=150`). Giới hạn FPS: `?fps=30`.

## Ưu tiên hiện tại (2026-10-22)
**Gameplay trước, art sau.** Game chơi đầy đủ khi `assets/` trống (đã test: chạy server với thư mục asset rỗng → vào chơi, lên cảnh giới, buff, chết → Tổng kết → Chơi tiếp, không lỗi JS).
Placeholder khi thiếu ảnh: nhân vật = hình tròn màu + chữ viết tắt (A Hớn → `AH`) + tên anim; vật cản phá được = hình chữ nhật đổi màu theo HP (xanh → vàng → đỏ, đống đổ nát màu xám); tường không phá = xám; kiếm = thanh dài màu theo cảnh giới; quái = hình tròn + số HP; vật phẩm = bong bóng màu.
Danh mục asset theo đúng tên ART_PROMPTS: `client/src/render/catalog.ts` (dùng chung cho loader và `assets:report`).

## Spec v3 / v3.1 / v3.2 (2026-10-22) — ✅ xong
- **Vòng kiếm**: 20 kiếm/vòng, vòng lồng nhau so le `k·18°/R`, lưỡi nghiêng 35° ngả về sau và lật trong 0.1s khi đổi chiều, bán kính chỉ theo vòng + scale (không theo số kiếm), vòng sau lùi xa hơn; kiếm tối đa 60→400 theo `max_swords`; quay 1 vòng/300ms; va chạm theo **góc quét** giữa 2 tick (vòng dày coi cả vành khăn là vùng chém). Camera lùi theo `2.6 × reach`.
- **Vẽ (client)**: `ParticleContainer`, mỗi vòng một màu `ring_tints` (vòng 1 giữ màu skin cảnh giới), vòng ngoài vẽ dưới, vệt mờ (vành khăn mờ + 2 bóng kiếm kéo sau). Thân kiếm vòng 2+ là placeholder trắng/xám cho tới khi sinh lại skin. Phím **H** vẽ capsule hitbox server chồng lên để so.
- **Chiến đấu**: bị chém rớt nguyên 1 cảnh giới, giữ tầng và % tiến độ; Luyện Khí bị chém là chết; kiếm thừa / kiếm rơi khi chết thành **Bó Kiếm** (≤10/bó); bể kiếm ×5 (`clash_unit`) + hai người bị đẩy văng 150 trong 0.2s.
- **EXP & vật phẩm**: viên EXP rải khắp bản đồ (`exp_orbs`), nhặt xong 5s mọc lại trong cùng ô 1000×1000; 400 vật phẩm/phòng mọc lại sau 3s; item Kiếm +3, Bó Kiếm +15; Rương mỗi 60s nổ 8 món; hút đồ 90×scale. Lưới không gian cho item (nhặt + AOI nhanh).
- **Nhịp**: anh chốt Luyện Khí → Trúc Cơ 30–60s → `exp_orbs.cell = 160` (sim đo 40–56s).
- **Vật cản**: chặn thân bằng **chân đế** (90% rộng × 30% cao ở đáy, Tiled đặt tay được bằng thuộc tính `footprint = "w,h"`), kiếm chém theo thân hình; y-sort với nhân vật; vật cản che mình thì mờ 50%. Nền + decor qua `ColorMatrixFilter` (giảm bão hòa/sáng/tương phản); viên EXP/item/quái có bóng đổ + quầng sáng.
- Test mới: rớt cảnh giới giữ tầng, Luyện Khí tầng bất kỳ chết, bó kiếm, bảng bể kiếm ×5, đẩy văng, góc so le R=1/2/3, vòng thiếu kiếm rải đều khe, reach tăng theo vòng, bán kính không đổi theo số kiếm (v3.2), tilt đổi dấu, va chạm góc quét, 1 vòng/300ms, chân đế, viên EXP mọc lại đúng vùng, vật phẩm mọc lại sau 3s. Tổng 64 test Go.
- **Còn lại / lỗi đã biết**:
  - Hàng rào/tường trong 3 bản đồ cũ được vẽ cao hơn hình trong Tiled (kiểu 3/4), nên vùng kiếm chém (theo hình Tiled) hơi thấp hơn phần nhìn thấy. Cần chỉnh bản đồ (`cmd/mapgen`) cho hình Tiled = hình vẽ.
  - Art cần sinh lại: 9 skin kiếm (lưỡi trắng/xám) và 3 nền đất (nhạt) — đã đánh dấu "Sinh lại" trong `docs/ART_JOBS.md`. Bó Kiếm chưa có tên art trong ART_PROMPTS → đang vẽ placeholder.
  - (Q8 đã xong: spec gốc v3.5 đặt `clash_unit` = 1 và sửa checklist §17 cho khớp.)

## Đã làm

- **Thiên Môn (29/10/2026, spec §8.4a, CHANGES_v5 mục 16–18)** — lối đi **chắc chắn** tới
  Thiên Đình, ngược hẳn với 4 cổng thường vốn cố tình giấu đích đến.
  - Mỗi khu trừ Thiên Đình có **1 Thiên Môn ở tâm khu**, mapgen chừa quảng trường trống bán
    kính 500 quanh nó. Thiên Đình có **4 Bến Thiên Môn** cách tâm 1 400 ở đầu 4 đường chính.
  - **Một chiều, đích cố định**: mỗi khu nối cố định với một bến (Linh Sơn → Bắc, Trúc Lâm →
    Đông, Phàm Trấn → Nam, Băng Nguyệt → Tây). Cố định chứ không ngẫu nhiên là có lý do:
    người chơi quen đường sẽ biết ai vừa từ khu nào lên, và ở khu trung tâm đó là thông tin
    đáng giá.
  - Thiên Môn và bến **không nằm trong bể ngẫu nhiên** của cổng thường; **đứng trên bến
    không bị hút đi tiếp** — thiếu luật này thì người vừa lên Thiên Đình bị bắn ngược về khu
    khác ngay lập tức. Cả ba điều có test riêng.
  - Minimap: icon ⛩ vàng **luôn hiện**, cộng mũi tên ở góc chỉ hướng khi còn cách trên 1 200.
    Kill feed khi Top 3 bước qua. Bot chỉ dùng Thiên Môn khi đã Đại Thừa trở lên — dưới cảnh
    giới đó Thiên Môn chỉ ném chúng vào giữa ổ kiến lửa.

- **Kỹ năng nhân vật qua Bí Kíp (29/10/2026, spec §6.3, CHANGES_v5 mục 11–14)** — đủ 10 kỹ
  năng, vật phẩm Bí Kíp (`skill_scroll`, weight 10), khung trạng thái chung, nút bấm, lớp
  trân châu, bot biết dùng, và `cmd/sim --duel`.
  - **Khung trạng thái chung** (`internal/game/status.go`): chậm · trói · bịt mắt · trượt ·
    gồng · xuyên vật cản · mai rùa · lướt. Mười kỹ năng dùng chung một bảng thời hạn, nên
    thêm kỹ năng sau chỉ là thêm một hằng số. Trạng thái **không cộng dồn** — trúng Ru Ngủ
    hai lần vẫn chỉ 3 giây, nếu không hai người cùng kỹ năng khoá cứng một người vô hạn.
  - Phím **Space** / nút tròn trên điện thoại, hiện icon + số lượt, xám khi hết. Chết là mất
    hết lượt; rớt cảnh giới hay qua cổng thì không.
  - **Trân châu dính màn hình**: lớp Pixi riêng, nằm dưới HUD, dồn về giữa, trượt xuống để
    lại vệt. Bot trúng thì **tầm nhìn còn 40%** (§6.3) — không có luật này thì kỹ năng đó
    hoàn toàn vô dụng với bot.
  - **`cmd/sim --duel`** đấu chéo 10×10, in bảng tỉ lệ thắng + **phân tích khắc chế** (ai
    khắc ai) + **số lần thật sự bấm được kỹ năng**.

  **Ba lỗi bài đấu tìm ra, đều là lỗi thật:**
  1. **Đàn em mài kiếm chính đại ca** — không có phép kiểm "cùng phe", nên Gọi Đàn Em là kỹ
     năng tự sát. Đại Ca Dép từ **1,4% → 98,6%** tỉ lệ thắng sau khi sửa.
  2. **Thiếu hẳn luật "bot bị bịt mắt còn 40% tầm nhìn"** của §6.3; `tran_chau_bot_vision`
     có trong config mà không chỗ nào đọc.
  3. **Ba nhân vật chưa từng bấm kỹ năng lần nào** (0/144 lượt): luật bot đòi "đang bị dí",
     mà bot trong bài đấu thì không sợ ai. Nhìn bảng tỉ lệ thắng thì "kỹ năng yếu" và "bot
     không bao giờ bấm" giống hệt nhau — phải đếm số lần bấm mới phân biệt được.

  **Chưa cân bằng xong, và đó là chủ ý**: chủ dự án chốt 29/10 là cứ thả ra cho người chơi
  phản hồi rồi chỉnh sau. Hiện `be_ti` và `dai_ca_dep` chưa ai khắc được; `ba_tam_loa`,
  `co_ba_ganh`, `map_tra_sua`, `shipper` chưa khắc được ai. Chạy lại bảng bất cứ lúc nào:
  `cd server && go run ./cmd/sim --duel`.

- **Thế giới 5 khu xong phần còn lại (29/10/2026, CHANGES_v5 mục 4 goroutine, 7, 8, 9, 10)**
  — và lần đầu tiên **cả 5 ngân sách §13.6 đều ĐẠT**:

  | Chỉ số | Đo được | Ngân sách |
  |---|---|---|
  | Tick server (TB) | **1,2 ms** | < 4 ms ✅ |
  | Tick server (p95) | **1,69 ms** | < 4 ms ✅ |
  | Snapshot lớn nhất | **4,2 KB** | < 8 KB ✅ |
  | Băng thông / client | **18,7 KB/s** | < 40 KB/s ✅ |
  | Ping p95 | 0,00 ms | < 20 ms ✅ |

  Kịch bản là của spec v5: 20 người + bot rải 5 khu, 10 Đạo Tổ. `cmd/loadtest` giờ in thêm
  **tick từng khu** (0,24–0,86 ms) để khi một khu chậm hẳn thì thấy ngay là khu nào.

  **Mỗi khu một goroutine.** An toàn vì trong một tick không khu nào đọc dữ liệu khu khác.
  Để đạt được điều đó phải dọn **bốn chỗ dùng chung** mà `-race` chỉ ra:
  1. `rng` của phòng → mỗi khu một bộ sinh số riêng.
  2. Hàng chờ chuyển khu → hàng riêng mỗi khu, gom lại sau khi mọi khu xong.
  3. Mốc gửi lại tên người chơi (`namesAt/namesDue`) → chuyển vào Zone.
  4. Kill feed liên khu → gom vào `Zone.feeds`, gửi sau `wg.Wait()`; gửi thẳng sang client
     của khu khác là ghi vào kết nối người ta đúng lúc khu kia đang ghi.
  5. `handleDeaths` gọi xếp hạng toàn thế giới giữa lúc khu khác đang sửa dữ liệu → chụp
     hạng **một lần đầu mỗi tick**. Hạng trên bảng Tổng kết chậm một tick thì không ai nhận
     ra; đua dữ liệu thì có.

- **5 bản đồ cho 5 khu (29/10/2026, spec §8.0, CHANGES_v5 mục 6)** — `cd server && go run
  ./cmd/mapgen` sinh cả 5; xem riêng từng khu bằng `-zone X -seed N -png ra.png`.
  Ảnh duyệt: `docs/perf/zone0.png` … `zone4.png`.

  | # | Khu | Thuật toán | Vật cản |
  |---|---|---|---|
  | 0 | Thiên Đình | giữ nguyên §8.1 | 296 |
  | 1 | Linh Sơn Cốc | 5 dãy vách đá **so le** (dãy lẻ chạm mép trái, dãy chẵn mép phải) chia hẻm núi, mỗi dãy chừa 1–2 đèo rộng 200–300 có cầu treo; 2 cao nguyên quanh đài, mỗi cao nguyên 2 lối lên | 158 |
  | 2 | Trúc Lâm Mê Cung | mê cung **đào đường lùi dần** trên lưới 11×11 — bảo đảm mọi ô đi tới được nhau **theo cách nó dựng**, không phải rải tường rồi cầu trời; bụi rậm gấp 3; vài ao | 322 |
  | 3 | Phàm Trấn | lưới phố **cố tình lệch** (khoảng cách 1 500–2 100, mỗi phố xê dịch ±140) + một đại lộ chéo; mỗi ô phố nhồi 3–5 công trình, thỉnh thoảng bỏ trống một ô làm sân | 109 |
  | 4 | Băng Nguyệt Hồ | hồ băng đa giác ở giữa + 4 đảo tuyết, bờ tuyết phủ quanh (`snow_mult` 0,8) | 29 |

  Mỗi khu **4 cổng** thuộc mạng chung cả thế giới và **2 Đài Tiên Khí**; Thiên Đình giữ 4 đài
  nhỏ + Thiên Tiên Đài. Khu 0 trước đây đặt 8 cổng, nay đúng 4 theo §8.4.

  **Ba lỗi bộ kiểm bắt được trong lúc làm** (đều là lỗi thật, không phải chỉnh số cho qua):
  1. `ringWall` nhận `gapArc` là **độ dài cung tính bằng đơn vị**, tôi truyền 0.5 tưởng là
     radian → cao nguyên bị bịt kín, 3,3% ô trống bị nhốt.
  2. `addRoad` vẽ vùng đường bằng **hộp bao** của đoạn, nên một con phố chéo biến gần hết
     Phàm Trấn thành mặt đường. Đã sửa: đoạn chéo cắt thành nhiều khúc ngắn ôm sát đường.
  3. `house()` nhận **góc trên-trái** chứ không phải tâm; truyền tâm vào thì nhà tràn ra
     đường và bị từ chối — cả thị trấn chỉ còn 50 vật cản.

  Ba bản đồ cũ (`lang_tre`, `cho_dem`, `nui_da`) đã xoá, 5 khu thay thế.

- **Thế giới 5 khu — xong phần server (29/10/2026, spec v5 §8.0 §8.4, CHANGES_v5 mục 4+5)**:
  một phòng giờ là **5 khu**, mỗi khu một bản đồ 12 000² riêng, mô phỏng riêng, danh sách
  người riêng, AOI riêng (`internal/room/zone.go`). Số khu lấy từ `room.zones` trong config;
  để 1 là quay về thế giới một bản đồ như cũ.
  - Người mới vào **khu ít người thật nhất**; bot sinh và hồi sinh vào **khu ít nhân vật
    nhất**; trần người + bot vẫn tính cho cả phòng (20), không nhân năm.
  - **Bảng xếp hạng, danh hiệu, Uy Danh tính chung cả thế giới**; riêng **dấu trên minimap**
    (Đài Tiên Khí, rương, ngôi sao Top 1) chỉ vẽ của khu đang đứng — vẽ dấu khu khác lên thì
    người chơi đuổi theo một chấm không bao giờ tới được.
  - **Cổng nối cả thế giới**: 20 cổng chung một mạng, bước vào một cổng ra ở một cổng bất kỳ
    khác, không bao giờ ra lại cổng vừa vào. Khu không biết gì về khu khác — phòng làm người
    chỉ đường qua `GateRouter`/`MoveToZone`. Chuyển người làm ở **ranh giới tick**, lúc không
    khu nào đang chạy, nên không cần khoá.
  - Client nhận `welcome` mới giữa trận = vừa sang khu: tải lại bản đồ, **không** phát lại
    âm thanh hồi sinh hay hướng dẫn, hiện chữ lớn tên khu 1,5s và tên khu cạnh minimap.
  - Thử thật: đi vào cổng ở Thiên Đình → sang **Trúc Lâm Mê Cung**, giữ nguyên kiếm và Uy
    Danh. Test: 6 bài trong `internal/room/zone_test.go`, trong đó bài cổng bắn 2 000 lần
    kiểm đủ 5 khu đều tới được và ~80% lần sang khu khác đúng như spec.

  **Chưa làm trong mục 2**: mỗi khu 1 goroutine (hiện 5 khu chạy tuần tự trong một goroutine
  — đủ nhanh ở quy mô 20 người, chưa cần song song); 4 thuật toán mapgen mới (mục 6) nên 5
  khu hiện dùng lại 3 bản đồ cũ; băng trơn (mục 7); bảng 5 khu khi bấm M (mục 9); loadtest
  ghi tick từng khu (mục 10).
| Milestone | Nội dung | Trạng thái |
|---|---|---|
| M0 | Repo theo mục 13.2, server Go + WebSocket + MessagePack, client Vite + PixiJS, `config/game.json` | ✅ |
| M1 | Vòng kiếm (capsule), quái 4 loại, item kiếm, EXP + 9 cảnh giới, scale 1→5, camera lùi theo scale, phím debug | ✅ |
| M2 | Chém thân → rớt tầng / chết, va kiếm đổi chiều + bể kiếm theo bảng 5.2 (có vết nứt), màn hình Tổng kết → Chơi tiếp | ✅ |
| M3 | Phòng + matchmaking (phòng đông nhất, hòa → cũ nhất), bot fill 9.3, bot AI (Farm/Hunt/Flee/Ambush/Wander, A*), trang chủ, leaderboard SQLite + `GET /api/leaderboard`, `cmd/sim` | ✅ |
| M4 | Đọc Tiled `.tmj`, 3 bản đồ tạo bằng code (`cmd/mapgen`), vật cản phá được + mọc lại 30s (cảnh báo 3s, không mọc đè người), bụi, bùn, gió (theo góc), hẻm 150/250/350, cổng cặp + mạng 4 cổng, bot né vào bụi/cổng. Hướng dẫn: `docs/MAP_GUIDE.md` | ✅ |
| M5 | 7 buff + làm mới thời gian, Rương Báu (báo trước 10s), nội suy 100ms, dự đoán + hiệu chỉnh vị trí mình, AOI, không gửi người ẩn thân/trong bụi, giả lập ping `?lag=` | ✅ |
| M6 | `artgen`: sinh ảnh minigen `/jobs` (gộp nhiều anim/1 ảnh), xóa nền, cắt sheet, căn pivot, ghép atlas + `manifest.json`; anim theo mood, VFX, 23 SFX + nhạc nền tự tổng hợp, UI đầy đủ. Danh sách art thiếu: `docs/ASSETS_STATUS.md` | ✅ (art còn thiếu, xem dưới) |
| M7 | Joystick ảo, giao diện điện thoại (dọc/ngang + gợi ý xoay), giới hạn FPS (PC 60 / ĐT 45), hướng dẫn VPS + Caddy HTTPS: `docs/DEPLOY.md` | ✅ (chưa deploy thật — cần bạn thuê VPS) |

Bổ sung gameplay 2026-10-22: đột phá cảnh giới → người xung quanh hiện emote kinh ngạc (§3.2); bị chém lộ khỏi bụi → emote "Lộ rồi!" (§12); vòng kiếm có vết nứt khi `crack > 0` (§5.2).

Câu hỏi Q1–Q5 đã áp dụng theo `docs/ANSWERS.md` (thưởng hạ sát +30, gió theo góc 60°/120°, viên EXP thuộc người chém 1.5s, chết rơi 50% kiếm).

Test: 50 test Go (EXP, scale, bảng bể kiếm, rớt cảnh giới + rơi kiếm thừa, khiên, buff hết hạn/làm mới, Song Bội Đan, vật cản vỡ/mọc lại, hẻm hẹp, cổng không bất tử, ẩn thân/bụi không lộ trong gói tin, bot fill 5 người → 15 bot, người thứ 21 sang phòng mới, leaderboard không ghi bot, lọc tên tục…).
Hiệu năng: 40 bot, bản đồ thật → ~0.16 ms/tick (mục tiêu < 3 ms).

## Art (minigen)

- **`teleport_out` của MỌI nhân vật cắt được 3/4 khung (29/10/2026)** — không phải lỗi ảnh:
  khung cuối là lúc nhân vật đã biến mất nên không còn pixel nào để bộ cắt tìm ra. Sinh lại
  bao nhiêu lần cũng vậy, chỉ tốn quota. Client vẫn chạy bình thường với 3 khung. Vì thế
  `artgen prep` giờ **chỉ đánh dấu sinh lại khi sheet thật sự hỏng**, dùng đúng ngưỡng của
  client (`render/assets.ts`: bỏ animation khi lệch quá 1,5× hoặc dưới 0,5×). Danh sách sinh
  lại nhờ vậy từ 15 xuống 4.
- **Danh sách sheet cắt sai nằm ở `assets/_raw/miscut.json`**, do chính `artgen prep` ghi ra
  chứ không gõ tay vào mã nguồn nữa. Ghi theo kiểu hợp nhất: chỉ gỡ tên nào vừa được cắt lại
  sạch sẽ — ảnh thô của job cũ có thể đã bị dọn đi nên prep bỏ qua chúng, mà coi "không cảnh
  báo" là "đã lành" thì sheet hỏng rơi khỏi ART_JOBS.md trong khi atlas vẫn giữ 22 khung sai.
- **2026-10-22 (phiên sau): 90/211 asset đủ** (artgen: 66 job xong, 130 còn thiếu). Mới có: khung thoại, rương, hit/death Ông Ba Vòng, death Đại Ca, hit Thằng Mập; run Cô Ba Gánh cắt lại được (không cần sinh lại).
  - `artgen run` giờ gửi **mỗi prompt đúng 1 lần**, tuần tự (`-c 1`), nghỉ 30s giữa prompt (`-gap`); hết quota thì chờ tới `reset_at` rồi gửi lại (lúc đó Gemini chưa vẽ gì).
  - `artgen prep`: sheet nhiều hàng mà cắt theo hàng ra sai (hàng dính nhau, ma bay tách khỏi xác) → tự cắt theo lưới cột × hàng, mỗi mảnh vào ô chứa tâm của nó.
  - **Lỗi đã biết (minigen):** sau khi sửa lỗi `metadata`, minigen dùng **chung 1 cuộc chat Gemini** cho mọi job (`chat_id` giống nhau) → Gemini lặp lại hình của prompt trước (5/11 ảnh sai: 4 ảnh vẽ rương, 1 ảnh chép sheet chết Ông Ba). Đã xóa ảnh sai. Cần minigen mở chat mới cho mỗi job trước khi sinh tiếp.
- **2026-10-22: theo ART_PROMPTS mới** — style "bựa", mô tả nhân vật + gag riêng từng anim, **1 animation / ảnh** cho nhân vật (lưới đúng cột × hàng), nền **magenta #FF00FF** (artgen tự nhận nền magenta hay trắng và xóa sạch viền hồng). Nhân vật cũ (style "cute") giữ tạm tới khi ảnh mới về. Tổng 174 ảnh game, đã có 20. Câu hỏi Q6 đã trả lời (xem `docs/QUESTIONS_DONE.md`).
- Q6 (2026-10-22): nhân vật 1 anim/ảnh; quái/cổng/rương gộp các anim của cùng 1 đối tượng, mỗi anim 1 hàng; VFX mỗi hiệu ứng 1 ảnh; vật cản `break`/`regrow` tách riêng; ảnh tĩnh gộp thoải mái. Bộ cắt frame giờ tìm từng hình theo vùng pixel liền nhau (sheet lệch lưới vẫn dùng được). Tổng 186 ảnh game.
- **Danh sách prompt để sinh bằng công cụ khác: `docs/ART_JOBS.md`** (tạo lại: `go run ./cmd/artgen prompts`). Lưu ảnh thành `assets/_raw/<tên job>.png` rồi chạy `go run ./cmd/artgen prep`.
- Tự động: `bash tools/art_watch.sh` — chờ tài khoản minigen sẵn sàng, mỗi đợt 30 ảnh, nghỉ 2.5 giờ giữa các đợt.
- Đã có: 9 skin kiếm, bong bóng kiếm, 3 viên EXP, 7 icon buff, 3 nhân vật (A Hớn, Bà Tám Loa, Ông Ba Võng) anim idle + run.
- Còn thiếu ~91 ảnh → game đang dùng placeholder (hình tròn + chữ / hình khối). Xem `docs/ASSETS_STATUS.md`.
- Quota minigen ~60–100 ảnh / 5 giờ → sinh theo đợt ưu tiên:
  ```bash
  cd server
  go run ./cmd/artgen run -p 1 -n 30 && go run ./cmd/artgen prep   # đợt 1: 7 nhân vật còn lại, quái, nền, vật cản, cổng
  go run ./cmd/artgen run -p 2 -n 30 && go run ./cmd/artgen prep   # đợt 2: anim bị chém/chết, VFX, huy hiệu, rương
  go run ./cmd/artgen run -p 4 -n 40 && go run ./cmd/artgen prep   # đợt 3: còn lại
  ```
- Nếu Gemini trả lời bằng chữ nhưng ảnh vẫn có trong lịch sử Gemini: tải ảnh về, đặt tên `assets/_raw/<tên job>.png` (tên job xem `artgen plan`), chạy `artgen prep`.

## Lỗi / hạn chế đã biết

- **Spec v5.7 — Tiên giới tính theo TỔNG Tiên khí (30/10/2026, §3.4)**. Trước đây cảnh giới
  Tiên nằm trong `Level` còn Tiên khí cộng dồn từng chặng — **hai nguồn sự thật**, và chúng
  lệch nhau đúng lúc bị chém. Giờ Tiên khí là nguồn duy nhất, cảnh giới suy ra từ nó:
  1–100 Chân Tiên · 101–300 Kim Tiên · 301–600 Thái Ất · 601–1000 Đại La · 1001–1500 Đạo Tổ
  (`tien_khi_ceilings`). Viên Tiên khí **đầu tiên** là vào Chân Tiên ngay.
  - **Vào Tiên giới cần Đại Thừa tầng 10 ĐẦY THANH** (694 960), không phải chỉ cần đứng ở
    tầng 10. Trước đây vừa chạm tầng 10 là Linh khí nhặt được cũng bị bỏ phí trong khi Tiên
    khí rất hiếm, nên người chơi kẹt cứng ở đó.
  - **Đạo Tổ nạp thanh 1 001 → 1 500**, chỉ khi đang đúng 1 500 thì Tiên khí dư mới đổi Uy
    Danh (10:1). Nhờ vậy Đạo Tổ **Thiêu Đốt được** — đó là câu trả lời cho Q26.
  - Thiêu Đốt ở Tiên giới = **8% độ rộng khoảng** của cảnh giới đang đứng: 8 · 16 · 24 · 32 ·
    40 mỗi giây. Theo độ rộng khoảng chứ không một con số chung, nên người ở đỉnh không được
    chạy rẻ hơn người mới vào.
- **Spec v5.8 — bị chém: thanh về MỨC SÀN cảnh giới mới (30/10/2026, §5.1)**. Trước giữ
  nguyên tầng và % tiến độ (Trúc Cơ tầng 2 → Luyện Khí tầng 2); giờ Trúc Cơ tầng 2 → **Luyện
  Khí tầng 1, 0%**, Kim Đan tầng 7 → Trúc Cơ tầng 1 0% (1 360). Chân Tiên bị chém → **sàn
  Đại Thừa** (346 800), Tiên khí 0. `TestRealmFloors` khoá cả 9 con số sàn vì `realm_mult`
  đổi một chữ số là cả bảng lệch mà không có gì báo.
- **Spec v5.9 — chém người Tiên giới không thấy Tiên khí rơi: ĐÃ TÌM RA NGUYÊN NHÂN
  (30/10/2026)**. Server **vẫn rơi đúng** — test tái hiện cho ra 3 viên tổng 24 điểm, khớp
  chính xác con số spec (30% × 80). Lỗi nằm ở chỗ **nhìn thấy**: viên văng trong bán kính
  `items.scatter_r` = 90, trong khi người Tiên giới có scale 5–6 nên thân rộng hơn 150 và
  cao hơn 400 pixel, **mà vật phẩm vẽ DƯỚI nhân vật** (`zIndex = y − 1000`). Viên rơi đúng,
  gửi đúng, nhưng nằm khuất hẳn dưới cái xác to đùng. Sửa theo spec: văng **200–500**
  (`tien_drop_scatter_min/max`), chia viên **10 → 2 → 1**, kèm event `tien_burst`. Phần rơi
  ra đất giờ **luôn** xảy ra kể cả người chém là phàm hay bot.
- **Bỏ `cmd/sim --duel` (30/10/2026, Q25)**. Chủ dự án chốt: kỹ năng có loại tăng sức mạnh,
  có loại khắc chế, nên **tỉ lệ thắng tay đôi không phản ánh giá trị** — cân bằng theo phản
  hồi người chơi và số liệu trận thật. Đã xoá hẳn công cụ thay vì giữ một thứ "có nhưng
  không được dùng".
- **Q20 đã chốt: giữ luật va kiếm đúng.** Mốc mới để so về sau: **84 lượt chết / 5 phút /
  40 bot**. Con số 608 lượt/10 phút ghi ở các mục cũ bên dưới là **đo trên luật va kiếm sai**
  (lỗi dấu `SegSegDist2`), không dùng để so nữa.

- **Minimap bấm mới mở, kill feed còn 1 tin (30/10/2026)**. Chủ dự án: *"khung chat điện
  thoại nhiều quá… mini map đang che mất phần trung tâm, làm nhỏ lại, hoặc cho phép bấm mới
  mở ra"*.
  - Trên **màn hẹp** minimap mặc định thu thành **nút 36px**, bấm mới mở. Ba nấc quay vòng:
    nút nhỏ → bản đồ góc (86px) → bản đồ to (252px) → nút nhỏ. Nấc giữa quan trọng: trước
    đây một cú chạm nhầm là nhảy thẳng sang bản đồ to phủ 70% màn hình, đúng cái "che mất
    phần trung tâm". Trên PC giữ nguyên hai nấc (góc ↔ to) — ở đó bản đồ góc không vướng gì
    mà bắt bấm thêm một lần chỉ làm chậm tay.
  - Kill feed **1 tin** (trước 2), chữ 10px, hộp mỏng. Tin loan toàn server cắt tối đa 2 dòng.
  - Mốc hẹp/rộng gom về `client/src/ui/layout.ts` để `hud.ts`, `minimap.ts` và CSS dùng chung
    một con số — ba nơi ba số thì sẽ có khổ màn hình rơi vào khe giữa chúng (bảng đã co lại
    nhưng minimap vẫn mở to).
  - Đo lại: HUD phủ **18–25%** màn hình (320×568 là 36%), không khối nào chồng nhau.
  - `uioverlap.mjs` sửa lại cho đúng: trước nó **tự nhét 2 tin** rồi đếm 2 — tức là kiểm
    chính cái mình vừa nhét. Giờ nhét dư 5 tin rồi bắn sự kiện `resize` để HUD tự cắt theo
    luật thật, nên con số "feed N tin" mới kiểm được luật.

- **HUD trên điện thoại chiếm gần hết màn hình — đã gọn lại (30/10/2026)**. Chủ dự án gửi ảnh
  chụp: *"chơi game mà không thấy gì"*. Đo lại bằng Chrome ở 4 khổ màn hình, worst case có
  đủ mọi loại thông báo cùng bật:
  - Bảng xếp hạng **Top 3** (trước 10 dòng), bỏ dòng danh hiệu — danh hiệu làm mỗi dòng cao
    gấp đôi mà cảnh giới+danh hiệu đã hiện trên đầu nhân vật rồi.
  - Kill feed **2 tin** (trước 5). Tin loan toàn server 1 dòng.
  - Minimap kẹp thêm theo **tỉ lệ màn hình** (≤ 24% cạnh ngắn): `size_mobile` là số px cố
    định, mà 120px trên máy rộng 360px là một phần ba bề ngang.
  - Khối trạng thái 77 → **54px**: bỏ nhãn "Linh khí" (màu thanh đã phân biệt), gộp
    "Kiếm x/y" và "Uy Danh n" về một dòng.
  - **Mốc "màn hẹp" xét CẢ HAI CHIỀU**, không chỉ bề ngang. Điện thoại xoay ngang là 800×360
    — rộng hơn mốc 720 nên bản đầu coi là "màn rộng" và vẫn vẽ 10 dòng + 5 tin: đo được
    **70% màn hình bị HUD phủ** và 6 cặp khối đè nhau. Chỗ chật ở đó là chiều cao.
  - Kill feed và tin toàn server vào **chung một cột flex** — ghim `bottom` riêng cho từng
    cái thì cái nào dài ra là đè cái kia (đo 212×21px trên máy 320). Cùng bài học với cột
    trái hôm 29/10.
  - Gợi ý xoay máy và dải báo Rương Báu dời khỏi chỗ đè minimap.
  Kết quả: **không khối nào chồng nhau** ở 360×800, 800×360, 320×568 và cửa sổ hẹp 420×900.
- **Hai lỗi CSS rò luật — đã sửa (30/10/2026)**. Cả hai cùng một kiểu: luật viết cho `.portrait`
  trần nên áp cho MỌI chỗ dùng class đó.
  1. `.portrait { width: 64px }` của màn chọn rò sang ô đại diện HUD (40px) → ảnh tràn hẳn
     ra ngoài khung tròn và đè lên chữ "Kiếm x/y". `.badge` không có `overflow:hidden` nên
     không có gì chặn.
  2. `.portrait { position: absolute }` của HUD rò ngược sang màn chọn → ảnh bị kéo ra khỏi
     luồng, **mười thẻ nhân vật trông như không có ảnh**.
  Giờ cả hai khoanh theo thẻ cha (`.badge .portrait`, `.char-card .portrait`), và `.badge`
  có `overflow:hidden` làm chốt chặn; huy hiệu tầng + vương miện chuyển ra `.badge-wrap` vì
  chúng cố tình thò ra ngoài khung.
- **Màn chọn nhân vật có ảnh + kỹ năng (30/10/2026)**. Mười nhân vật chỉ khác nhau ở kỹ năng,
  mà lưới cũ chỉ có tên nên người chơi chọn mù. Thêm khung xem trước: ảnh to, tên, tên kỹ
  năng và mô tả một dòng (`STR.skillDesc`, số lấy đúng spec v5.4/v5.5). Rê chuột để so, bấm
  để chọn. Ảnh lấy `portrait_neutral.png` thay vì `frames/idle_00.png` — khung hình toàn
  thân trên canvas 512 thu về ô 40px thì nhân vật bé bằng hạt gạo.

- **Spec v5.6 — Tiên giới đốt Tiên khí (30/10/2026, §4.1c)**. Đây là câu trả lời cho Q24:
  trước đó lên Tiên giới là mất hẳn khả năng chạy nhanh, vì nhánh duy nhất của Thiêu Đốt
  tiêu EXP mà người Tiên giới đứng đúng ở đáy cảnh giới. Giờ hai đường tu có hai loại nhiên
  liệu: người phàm đốt EXP, người Tiên giới đốt **8% Tiên khí của bước hiện tại mỗi giây**
  (`burn.tien_frac_per_sec`), không xuống dưới mốc khởi điểm của cảnh giới đang đứng.
  "Bước hiện tại" là `tien_khi_per_step[idx]` — bước đã ĐƯA họ tới cảnh giới đang đứng, chứ
  không phải bước đi tiếp; chính ví dụ của spec chốt cách đọc này ("Đạo Tổ: 8% × 500"), vì
  Đạo Tổ không còn bước đi tiếp nào. Đại Thừa tầng 10 chưa nhặt Tiên khí thì chưa đốt được.
  Đo trong game thật (Chân Tiên): Tiên khí **60 → 48 trong 1,5 giây = 8/giây** = 8% × 100,
  đúng con số spec; không rớt cảnh giới; nhả tay là tắt.
- **⚠ Đạo Tổ vẫn không Thiêu Đốt được — chờ chốt (Q26)**. Hai luật đều đúng theo câu chữ
  nhưng cộng lại thì chặn nhau: v5.6 nói không xuống dưới mốc khởi điểm của cảnh giới đang
  đứng, còn `overflowToScore` (Q22) ghim Tiên khí của Đạo Tổ **đúng** ở mốc 1 500 và đổi mọi
  phần dư sang Uy Danh. Nên Tiên khí của Đạo Tổ luôn bằng đáy → không còn gì để đốt, và
  người ở đỉnh lại là người duy nhất không chạy nhanh được. Đã khoá hiện trạng bằng
  `TestDaoToHasNoTienKhiHeadroom`; test ghi rõ là khoá hiện trạng, không khẳng định ý đồ.

- **Spec v5.4 + v5.5 — chỉnh 10 kỹ năng (29/10/2026, §6.3)**. Mọi con số vào `config/game.json`,
  có `TestSkillDurationsMatchSpec` khoá lại để lần sau spec đổi mà quên sửa config thì test đỏ.
  - **Bánh Bao Thần đổi hẳn luật**: trước hồi 30% số kiếm bể trong 10 giây, giờ hồi thẳng
    **20% số kiếm tối đa của cảnh giới**. Luật cũ có lỗ không chữa được bằng cách chỉnh số:
    ai chưa bị đánh thì ăn bánh bao chẳng được gì, nên kỹ năng vô dụng đúng lúc muốn dùng
    nhất. Sổ ghi kiếm bể (`lostSwords`) gỡ bỏ hẳn.
  - **Gọi Đàn Em**: 2 đàn em, 10s, mỗi con mang **50% số kiếm hiện tại của đại ca** (tối
    thiểu 1) xếp vòng như người chơi, nhỏ hơn 30%, không nhặt đồ, không qua cổng. Đại ca
    không bị trừ kiếm.
  - **Giao Hỏa Tốc**: ×3 trong 5s, xuyên **mọi** vật cản (trước chỉ xuyên thứ phá được),
    trừ biên khu. Hết giờ mà kẹt trong vật cản → trượt mượt ra chỗ trống trong 0,3s bằng
    chính `resolveBlocking` — không bật ra tức thời, vì nhảy một phát là client thấy đúng
    kiểu giật mà §8.2a vừa sửa xong.
  - **Phun Trân Châu**: 5s, và trong 5s đó **không nhặt được gì** — chặn ở đúng một cửa
    (`CanPickUp`, ngay đầu `resolvePickups`) chứ không rải vào từng loại vật phẩm, vì spec
    nói cả vật phẩm/Linh khí/Tiên khí/Bí Kíp đều đi xuyên qua. Người ngoài thấy icon 🚫 bay
    quanh nạn nhân: lớp trân châu chỉ nạn nhân nhìn thấy, không có icon thì từ ngoài nhìn
    vào không ai hiểu vì sao họ đứng cạnh viên Linh khí mà không nhặt.
  - **Gánh Lướt**: vệt trái cây giờ gây **choáng** (`StStun`) cùng lúc với trượt — không đi,
    không dùng kỹ năng, không Thiêu Đốt, kiếm vẫn quay. Tách khỏi `StSlip` vì trượt là đổi
    *cách* di chuyển còn choáng là *cấm* di chuyển.
  - Lưới Mì 5s · Gồng Cơ 5s · Rụt Cổ 5s · Loa Phường đẩy 500 · Ru Ngủ bán kính 500.
- **Bài đối chiếu va chạm phủ thêm Giao Hỏa Tốc (29/10/2026)**. Kỹ năng này đổi **cả hai**
  thứ quyết định vị trí — tốc độ ×3 và xuyên vật cản — nên nếu client không biết thì lệch
  mỗi tick. 1/4 số ca trong `cmd/collisionfixture` giờ đang xuyên. Đo: **0,0000 đơn vị lệch**
  trên 1 500 bước × 5 bản đồ; và khi cố tình tắt phần xuyên ở client thì bài kiểm báo **46 ca
  quá ngưỡng, nặng nhất 100,6 đơn vị** — nên đây không phải một dòng chữ xanh vô nghĩa.
- **⚠ Bài đấu từng cặp trước đây đo sai (sửa 29/10/2026)**. `--duel` cho mỗi bên **một vòng
  kiếm** (10 kiếm) trong khi trần Kim Đan là **100** — tức 1/10 số kiếm thật của người chơi
  cùng cảnh giới. Mọi kỹ năng tính theo TRẦN cảnh giới bị thổi phồng gấp mười: Bánh Bao Thần
  ra **100% thắng (54/54)** với một vòng, **75,9%** với đủ kiếm. Lý do cũ của việc dùng một
  vòng là "2 vòng trở lên thì 85% trận hoà", nhưng con số ấy đo TRƯỚC khi `clash_cooldown`
  về 0; đo lại thì đủ kiếm chỉ **9% hoà**, còn ít hơn một vòng (18%). Mặc định đã đổi sang
  **đủ trần cảnh giới**; `--duel-swords N` để quay lại kiểu cũ.

- **Data race thứ 6 — `moveToZone` đọc map của khu khác (sửa 29/10/2026)**. Lúc chia năm khu
  ra năm goroutine tôi tìm và sửa 5 race; con thứ 6 lọt vì nó nấp sau một hàm trông vô hại.
  `MoveToZone` (cổng dịch chuyển gọi, chạy TRONG goroutine của một khu) gọi `zoneOf(p.ID)`,
  mà `zoneOf` quét map `Players` của **cả 5 khu** — trong khi bốn khu kia đang tick và có thể
  đang xoá người. Ngoài đời đây không phải lỗi nhỏ: Go gặp đọc-ghi map song song thì
  `fatal error: concurrent map read and map write`, **sập cả server**. Sửa bằng cách gắn cứng
  hàm chuyển khu với khu cài nó (`moveFromZone(z)`) — khu nguồn vốn đã biết, không cần đi tìm.
  Đã ghi cảnh báo lên `zoneOf`. Kiểm: 30 lượt `-race` → 0 race (trước đó 3/20 lượt dính).
- **Phòng đủ 60 người mà vẫn có khu 8–9 (sửa 29/10/2026)**. Hai lỗi chồng nhau:
  1. `rebalanceZones` lặp 5 lượt nhưng việc chuyển khu chỉ xảy ra ở **ranh giới tick**, nên
     `z.Count()` không nhúc nhích trong lúc lặp. Cả năm lượt đều thấy tình trạng cũ → xếp
     năm lệnh cùng chiều → khu cho đi bị rút quá tay thành khu thiếu mới → lượt sau kéo
     ngược về, qua lại mãi. Giờ tự cộng sổ phần đã xếp hàng (`delta`) và quyết theo con số
     **dự kiến**, thêm điều kiện cho đi xong không được tụt dưới sàn, và `spareBot()` không
     bốc trúng con đã nằm trong hàng chờ.
  2. Vá sàn trước đây chạy theo **nhịp sinh bot 2 giây**, mà bot đi qua cổng bất cứ lúc nào
     → khu có thể nằm dưới sàn gần 2 giây. Giờ `rebalanceZones` chạy **mỗi tick** (rẻ: 5 khu,
     vài phép so), nên người vừa bước qua cổng không rơi vào khu gần như trống.
  `processRespawns` cũng dính cùng kiểu sai: bot hồi sinh rời khu đang **đúng sàn** thì chỗ
  thiếu chỉ chạy sang khu khác chứ không mất — đã chặn.
  `TestZoneFloorAndRoomCap` sửa lại cho đúng điều server hứa: sàn là *"tụt xuống thì được vá
  lại ngay"*, không phải *"không khoảnh khắc nào dưới sàn"* (bot có thể rời khu ở đúng tick
  cuối, lệnh bù chỉ chạy được ở tick sau). Kiểm: 40 lượt `-race` xanh; và vẫn đỏ khi cố tình
  tắt `rebalanceZones`, nên không phải nới lỏng cho qua.
- **Điều khiển gọn lại (29/10/2026)**. Thiêu Đốt Tinh Huyết thêm cách **nháy đúp rồi giữ**
  (300 ms, lệch dưới 40 px) bên cạnh Shift và nút 🔥 — hai cách cũ đều đòi tay thứ hai, còn
  nháy đúp giữ nguyên ngón/tay đang lái. Luật nhận nháy đúp (`tapDown`) viết một lần dùng
  chung cho chuột và cảm ứng, vì tách làm hai bản là kiểu lỗi chỉ hiện trên đúng một loại
  thiết bị. Thêm phím test **G** (+1 lượt kỹ năng) — không có nó thì muốn thử kỹ năng phải
  chạy quanh bản đồ chờ Bí Kíp rơi.
- **Kỹ năng giữ phím Space, KHÔNG dùng Ctrl (29/10/2026)**. Có thử Ctrl một nhịp rồi bỏ; lý
  do ghi lại ở đây để đừng ai đổi lại: phím đi là **WASD**, nên vừa giữ Ctrl vừa bấm W là
  **Ctrl+W — đóng tab**. Chrome xếp Ctrl+W vào nhóm phím tắt dành riêng cho trình duyệt,
  `preventDefault()` không chặn được. Cách duy nhất chặn thật là Keyboard Lock API, mà nó
  **chỉ chạy ở chế độ toàn màn hình** (ràng buộc cố ý của trình duyệt, để trang web bất kỳ
  không giam được người dùng trong tab) và chỉ có trên Chrome/Edge — tức là người chơi cửa
  sổ thường vẫn mất tab. Space không đụng phím tắt nào. Cùng lý do đó, tránh `Ctrl`, `Alt`
  và `F`-key cho bất kỳ phím game nào sau này.
- **HUD trái tự đè lên nhau — đã sửa (29/10/2026)**. Ba khối bên trái (chỉ số · hàng buff ·
  minimap) trước đây mỗi khối một `top:` cố định (12 / 84 / 136 px). Hễ khối chỉ số cao thêm
  một dòng — thanh Tiên khí ở Tiên giới, rồi nhãn tên khu mới thêm — là nó tràn xuống đè
  khối dưới. Đây là loại lỗi tự sinh lại: mỗi lần thêm một dòng vào HUD lại phải nhớ dời tay
  cả ba con số, và quên là chuyện chắc chắn xảy ra. Giờ gom vào `.hud-left` xếp dọc bằng
  flex nên khối sau tự đẩy xuống, thêm bao nhiêu dòng cũng không hỏng. Bảng hướng dẫn lúc
  mới vào (`.guide`) cũng đang nằm đè minimap → chuyển xuống **giữa-dưới**. Đo ở trường hợp
  xấu nhất (Đạo Tổ, có thanh Tiên khí, 1 050 kiếm, 5 buff): 12 → 119 → 171 → 205, không
  khối nào chồng nhau.
- **Nền bản đồ có vệt hồng — đã sửa (29/10/2026)**. 7 ảnh nền (`soft_grass` 17,3%,
  `bamboo_litter` 15,7%, `mossy_stone` 10,3%, …) vẫn còn magenta nguyên bản. `modeTile` cố ý
  **không** lọc magenta cho ảnh nền vì nền phải đặc — nhưng magenta *trong* ảnh nền nghĩa là
  bên sinh ảnh không vẽ kín khung, chứ không phải nền cần giữ. `patchMagenta` (`cmd/artgen/prep.go`)
  vá các pixel đó bằng màu trung bình vùng lành **và** ghi job vào `miscut.json` để lần sau
  vẽ lại. Quét lại: 0 asset còn magenta, chụp màn hình đo 0,000% magenta.
- **Giật khi chạm vật cản — đã sửa 3 nguyên nhân, còn sót ít (29/10/2026, spec v5 §8.2a)**.
  Thêm vào F3 dòng `hiệu chỉnh N/giây · lệch nhất X` (đếm số lần server kéo nhân vật về).
  Đo trước khi sửa: đi tới rìa bản đồ ra **20,2 lần/giây**, nhân vật ghim cứng ở y = 11 549.
  Ba chỗ client dự đoán khác server, tất cả đều bị bài đối chiếu 1 500 bước bắt được:

  | # | Lệch ở đâu | Hậu quả |
  |---|---|---|
  | 1 | Client bỏ qua `play_margin` (kẹp `[r, size−r]` thay vì `[margin+r, size−margin−r]`) | Cả vành ngoài rộng 420 đơn vị — ~13% bản đồ — bị kéo về mỗi tick |
  | 2 | Thứ tự đẩy ra khỏi vật cản khác nhau (lưới không gian vs mảng) | 4/1 000 ca lệch, nặng nhất **173,7 đơn vị** |
  | 3 | Client thiếu phép xét **đa giác** và thiếu luật vùng **nước** | Đi qua bùn/nước/gió hình méo là lệch tốc độ liên tục |

  Sau khi sửa: bài đối chiếu **0,0000 đơn vị lệch** trên 1 500 bước × 3 bản đồ; đo trong game
  **20,2 → 4,8 lần/giây** lúc cao nhất, nhân vật không còn ghim ở rìa.

  **Còn sót 1–5 lần/giây**: gần như chắc là đòn đẩy văng (`KnockVel`) — client không dự đoán
  được va chạm chiến đấu của server. Chưa xác minh; nếu muốn triệt thì phải gửi kèm vận tốc
  đẩy trong snapshot để client mô phỏng theo.

- **Thỉnh thoảng mất ngữ cảnh WebGL (28/10/2026)**: chạy 3 lượt kiểm tra bằng Chrome headless
  trên AMD RX 570 (RADV) thì **1 lượt** in `PixiJS Error: Could not initialize shader` kèm
  `WebGL context may be lost`. Khung hình không tụt (58 fps) nên có vẻ chỉ là trình điều
  khiển trong môi trường headless. **Nhưng client hiện KHÔNG bắt sự kiện `webglcontextlost`**
  — nếu chuyện này xảy ra với người chơi thật thì màn hình sẽ đen vĩnh viễn, phải tự tải lại
  trang. Chưa làm vì chưa thấy tái hiện trên máy thật; nếu có người báo "màn hình đen" thì
  đây là chỗ nhìn đầu tiên.
- **Tài khoản minigen `pt1` đang báo UNAUTHENTICATED (cookie hết hạn)** → cần đăng nhập lại cookie bên minigen rồi chạy tiếp `artgen run`.
- Gemini đôi khi vẽ sai số frame (5 thay vì 4, 6 thay vì 8). `artgen prep` giữ đúng số frame thực tế vẽ ra; anim vẫn chạy được.
- Ảnh của Bà Tám Loa lấy từ bản JPEG 1024px (người dùng tải từ Gemini) → hơi mờ, có ngôi sao watermark nhỏ; nên sinh lại khi dư quota (`artgen run -only char_ba_tam_loa_core -redo`).
- Cổng 8080 trên máy dev hiện đang bị chương trình khác chiếm → chạy server với `-addr :8090` (khi dùng `npm run dev` thì chạy `KX_SERVER=http://localhost:8090 npm run dev`).
- Cân bằng (sim 10 phút, 40 bot): lên Trúc Cơ trung bình ~7 phút, nhiều bot chết sớm ở Luyện Khí 1 — nếu thấy chậm, chỉnh `config/game.json` (ví dụ tăng EXP quái, giảm `exp_base`). Chưa đổi gì vì đây là quyết định cân bằng của bạn.
- Dự đoán chuyển động phía client không mô phỏng đẩy lùi (knockback) → khi bị chém thấy giật nhẹ 1 nhịp.
- **Spec v4.0/v4.1 — ĐÃ LÀM (22/10/2026)**: số cân bằng mới (`move_base` 195, `cell` 140), độ bền kiếm `durability_per_realm` 1.2 thay `clash_ratio_*`, UI không tràn màn hình, 3 lỗi bản đồ (lề 300 + tường biên + dãy tường nối liền). Test Tốc Kiếm không đụng `reach` — PASS ngay, logic vốn đã đúng.
- **Spec v4.0 — ĐÃ XONG CẢ 3 VIỆC LỚN (22/10/2026)**: lá chắn vòng kế ngoài cùng §5.2 (xem Q15 về mâu thuẫn công thức), Tiên khí + 5 Đài §3.4/§8.5, mapgen viết lại theo 9 bước §8.1 (`cmd/mapgen/gen.go`, có `--seed N --png out.png` xem trước).
- **Bản đồ sinh động (28/10/2026, spec v4.9 §8.6, CHANGES_v4 mục 43–47)**: vùng bùn/ao là **đa giác méo theo nhiễu** (`r(θ) = r0 × (1 + 0.25·nhiễu(θ))`, Chaikin ×2, 56 đỉnh) — `Shape` thêm `Poly`, **server xét đúng đa giác client vẽ**. 3 ao mỗi bản đồ có mặt nước gợn sóng. Decal rải Poisson-disk theo biome, bóng đổ + vụn dưới chân vật cản, vật cản xoay ±6°/lật theo id. Sự sống: mây trôi, lá rơi, đom đóm, sương, bướm, bóng chim — trần 150/60/0 hạt theo chất lượng. 80 job art mới vào `assets:report` (nền biome, decal, viền vùng, ambient, bảng tên, chân dung HUD).
  **Ba lần đo, ba lần sửa hiệu năng** (bản đầu làm 58 → 36 fps, draw call 237 → 611): (1) decal và hạt là `Graphics` riêng từng cái → nướng thành texture dùng chung; (2) ~2 000 decal rải khắp bản đồ bị duyệt hết mỗi khung → gom ô 2048 và tắt `renderable` ô ngoài khung nhìn; (3) bóng đổ vật cản vẫn là Graphics, 286 vật cản = 286 lượt vẽ → nướng nốt. Kết quả **draw call 611 → 118**, vẽ 0,9 ms.
- **Spec v4.6 VFX (28/10/2026)**: `vfxLayer` riêng gắn `AdvancedBloomFilter` một lần cho cả lớp; hào quang Tiên 4 lớp blend cộng; đột phá có `ShockwaveFilter` + khựng hình 80ms + chớp trắng + chữ cảnh giới bật ra. Cài `pixi-filters`, `revolt-fx`.
- **Spec v4.7 thần thú (28/10/2026)**: bay quanh chủ liên tục, động tác ngẫu nhiên mỗi 8–15s, Đạo Tổ 2 con lệch pha 180°, phản ứng lóe sáng khi nhặt Tiên khí / khiên đỡ đòn, bay sát + vệt đỏ khi Thiêu Đốt. Phím **V** đổi chất lượng Cao/Vừa/Thấp; máy yếu thì **tự hạ** (FPS < 45 suốt 3s hạ một bậc, > 55 suốt 10s nâng lại), nhưng người chơi đã tự chọn thì khoá.
- **Spec v4.8 bảng tên + v4.9 ảnh đại diện (28/10/2026)**: xem commit `c840446` và `eca03d2`.
- **Thần thú luôn lượn lờ (28/10/2026, spec v4.7 §3.5, CHANGES_v4 mục 34 + 36)**: bỏ kiểu "mỗi 20s bay 1 vòng" và bỏ luôn nút tắt. Sau đột phá, thần thú thu nhỏ 60% và **ở lại bay quanh chủ suốt thời gian còn ở Tiên giới**: quỹ đạo elip 1.3–1.8 × chiều cao nhân vật, 1 vòng/4s, ép dẹt 0.35, độ cao nhấp nhô chu kỳ 3s, tâm bám chủ lerp 0.1 nên lúc chạy bị kéo dài ra sau. Mỗi 8–15s làm một động tác 1.5–3s (xoắn ốc vút lên · số 8 · đổi chiều · lao ra rồi về · cuộn quanh đầu), vào/ra bằng bao hình sin nên không giật. Đạo Tổ có **2 con lệch pha 180°**, một trắng một đen. Đốt sau đi theo vị trí đầu các khung trước, lưu trong `Float64Array` cấp phát một lần (spec cấm tạo object mỗi frame). Chất lượng Cao/Vừa/Thấp = 16/10/6 đốt, vệt hạt 100/50/0%, chủ nhỏ hơn 40 px chỉ vẽ đầu + 4 đốt, vệt hạt chỉ bật cho 6 con gần nhất. Ẩn Thân/trong bụi thì mờ 20% theo chủ. **Phím V giờ đổi mức chất lượng** (§3.6) thay cho bật/tắt thần thú. Đo tại chỗ: Đạo Tổ song long, 9 người, 1 912 entity trong AOI → **60 fps, vẽ 1,0 ms, 245 draw call**.
- **Đo hiệu năng TRƯỚC tối ưu (28/10/2026, CHANGES_v4 mục 24)** — `cd server && go run ./cmd/loadtest -secs 15`, đúng kịch bản §13.6 (40 nhân vật, 10 người Đạo Tổ đủ 1 050 kiếm đứng chụm giữa bản đồ):

  | Chỉ số | Đo được | Ngân sách §13.6 | |
  |---|---|---|---|
  | Tick server (trung bình) | **11,45 ms** | < 4 ms | VƯỢT ~3× |
  | Tick server (p95) | **19,89 ms** | < 4 ms | VƯỢT ~5× |
  | Snapshot lớn nhất / client | **38,3 KB** | < 8 KB | VƯỢT ~5× |
  | Băng thông xuống / client | **43,1 KB/s** | < 40 KB/s | VƯỢT nhẹ |
  | Ping p95 | 0,00 ms | < 20 ms | ĐẠT |

  Ping 0 ms vì loadtest chạy trong cùng tiến trình, không qua mạng thật — con số này chỉ
  chứng minh đường trả Pong **không phải chờ vòng lặp game**, không nói gì về mạng thật.
  Snapshot 38,3 KB là đỉnh lúc một loạt vật phẩm cùng lọt vào tầm nhìn khổng lồ của người
  Đạo Tổ (`ItemsAdded` dồn vào một gói) — đúng chỗ mục 27 (delta + chỉ gửi sự kiện) nhắm tới.
- **Mục 27 — gói snapshot gọn lại (28/10/2026)**: cùng lệnh `go run ./cmd/loadtest`, cùng kịch bản:

  | Chỉ số | TRƯỚC | SAU | Ngân sách §13.6 | |
  |---|---|---|---|---|
  | Tick server (trung bình) | 11,45 ms | **14,04 ms** | < 4 ms | VƯỢT ~3,5× |
  | Tick server (p95) | 19,89 ms | **25,09 ms** | < 4 ms | VƯỢT ~6× |
  | Snapshot lớn nhất / client | 38,3 KB | **8,3 KB** (−78%) | < 8 KB | sát mép |
  | Băng thông xuống / client | 43,1 KB/s | **21,4 KB/s** (−50%) | < 40 KB/s | ĐẠT |
  | Ping p95 | 0,00 ms | 0,00 ms | < 20 ms | ĐẠT |

  Làm những gì: `ItemsAdded` cắt theo ngân sách 2 KB/gói ưu tiên vật phẩm gần người chơi ·
  bỏ viên EXP nhỏ ngoài 1 500 khi camera đã lùi xa · người/quái ngoài nửa tầm nhìn chỉ gửi
  10 Hz thay vì 20 Hz · trần 150 entity mỗi gói · client nghẽn thì **bỏ gói cũ** chứ không
  xếp hàng (`SendLossy`) · quái chỉ gửi loại / máu tối đa / bán kính **lần đầu** client thấy
  nó (46 → 27 byte/con) · người chơi bỏ `omega` (client tự tính từ config), `mood` thành chỉ
  số thay cho chuỗi, `exp` chỉ gửi cho chính chủ (102 → 88 byte/người).

  Tick server **xấu đi** chứ không tốt lên — việc xếp vật phẩm theo khoảng cách tốn thêm CPU.
  Đó là mục 26, xem pprof bên dưới.
- **pprof: 88% CPU server nằm trong `diffItems` (28/10/2026, mục 26)** — `go run ./cmd/loadtest -secs 30 -cpuprofile cpu.prof`, 10 hàm tốn nhất:

  | | Hàm | flat | cum |
  |---|---|---|---|
  | 1 | `math.archExp` | 19,2% | 19,2% |
  | 2 | `math.archHypot` | 8,1% | 8,1% |
  | 3 | `maps.(*Iter).Next` (duyệt map) | 7,0% | 8,6% |
  | 4 | `room.(*Room).diffItems` | 6,0% | **87,9%** |
  | 5 | `game.(*Game).ViewHalfExtents` | 6,0% | **44,0%** |
  | 6 | `math.archLog` | 4,9% | 4,9% |
  | 7 | `maps.ctrlGroup.matchH2` | 3,9% | 3,9% |
  | 8 | `runtime.mapaccess1_fast32` | 3,9% | 15,5% |
  | 9 | `math.pow` | 3,5% | 35,1% |
  | 10 | `math.ldexp` | 3,3% | 3,7% |
  |  | `game.(*Game).InViewRadius` | 3,0% | 48,1% |
  |  | `room.(*Room).wantItem` | 1,4% | 47,9% |

  Đọc bảng: gần như toàn bộ CPU đi vào vòng lặp lọc vật phẩm theo tầm nhìn, và bên trong nó
  `ViewHalfExtents` gọi `math.Pow(Scale, CameraExp)` — tức `exp(log())` — **cho từng vật phẩm,
  từng client, từng tick**, trong khi giá trị đó chỉ phụ thuộc người xem. `archHypot` là do
  xếp vật phẩm theo khoảng cách dùng `math.Hypot` thay vì so bình phương. Nhóm `maps.*`
  (≈15%) là tra `knownItems` và duyệt toàn bộ vật phẩm của bản đồ cho mỗi client.
- **Mục 26 — sửa xong, tick 14,04 → 2,55 ms (28/10/2026)**:

  | Chỉ số | TRƯỚC mục 27 | SAU mục 27 | SAU mục 26 | Ngân sách |
  |---|---|---|---|---|
  | Tick server (trung bình) | 11,45 ms | 14,04 ms | **2,55 ms** | < 4 ms ✅ |
  | Tick server (p95) | 19,89 ms | 25,09 ms | **5,00 ms** | < 4 ms ❌ |
  | Snapshot lớn nhất / client | 38,3 KB | 8,3 KB | **8,0 KB** | < 8 KB ✅ |
  | Băng thông xuống / client | 43,1 KB/s | 21,4 KB/s | **20,0 KB/s** | < 40 KB/s ✅ |
  | CPU server | 42% | 42% | **8%** | |

  Bốn việc, theo đúng thứ tự pprof chỉ ra:
  1. `ViewHalfExtents` gọi `math.Pow(Scale, CameraExp)` — tức `exp(log())` — cho **từng vật
     phẩm, từng client, từng tick**, trong khi nó chỉ phụ thuộc người xem. Giờ tính sẵn
     trong `refreshDerived` (`Player.ViewHW/ViewHH`) và lấy một lần trước vòng lặp.
  2. Xếp vật phẩm theo khoảng cách dùng `math.Hypot` → so **bình phương** khoảng cách.
  3. `diffItems` duyệt **toàn bộ** `g.Items` cho mỗi client → dùng `itemGrid` vốn đã có sẵn,
     chỉ quét những ô chạm khung nhìn. Ô lưới đổi từ `map[uint32]*Item` sang slice, xoá O(1)
     bằng hoán đổi với phần tử cuối (có `itemgrid_test.go` riêng vì kiểu xoá này dễ làm mất
     phần tử khác).
  4. `knownItems` của client đổi từ `map[uint32]bool` sang **slice theo chỉ số dày đặc**
     (`Item.Slot()`), lưu id để nhận ra slot đã được viên khác dùng lại. Hết hẳn băm trong
     vòng nóng.

  Rồi phần tối ưu đó **phơi ra một lỗi hình học thật** (xem mục dưới); sửa lỗi xong thì CPU
  tăng lại, nên phải tối ưu tiếp vòng so kiếm. Số cuối cùng, ba lần chạy liên tiếp:
  **tick 4,88 / 5,01 / 5,04 ms · p95 6,2–6,7 ms · snapshot 8,1–8,5 KB · 22 KB/s**.
- **LỖI: `SegSegDist2` sai dấu — kiếm va nhau ở vị trí ma (28/10/2026)**: công thức giao
  điểm hai đoạn thẳng viết `cross(d2, r.Mul(-1))` thay vì `cross(d2, r)`, tức **đảo dấu cả
  hai tham số** t và u. Hệ quả: hàm báo "hai đoạn cắt nhau" đúng vào lúc phần **kéo dài
  ngược** của chúng cắt nhau. Đo được một trường hợp thật: hai thanh kiếm cách nhau 207 đơn
  vị (xa hơn tổng chiều dài hai lưỡi) vẫn bị tính là va vào nhau. Bắt được nhờ bản tối ưu
  lọc đúng theo hình học lại "lệch" so với bản cũ. Đã sửa, có `internal/geom/segseg_test.go`
  khoá cả hai chiều.

  **Ảnh hưởng cân bằng — cần người chủ dự án xem lại**: `go run ./cmd/sim --bots 40
  --minutes 5` cho **84 lượt chết** (2,10 lượt/bot). Con số cũ ghi trong tài liệu là 608
  lượt trong 10 phút (≈ 304 trong 5 phút) — nhưng con số cũ đo khi va kiếm còn lỗi, tức là
  có rất nhiều cú va ảo. Nay va kiếm chỉ tính khi hai lưỡi thật sự chạm nhau nên bể kiếm
  chậm hơn nhiều. Đây là **đổi nhịp trận đấu**, không phải chỉnh số — nên ghi vào
  `docs/QUESTIONS.md` chờ quyết định, chưa đụng vào `config/game.json`.
- **Vòng so kiếm (`sparseRingsTouch`) — 83% CPU server (28/10/2026)**: pprof trên **server
  thật** (có bot, bản đồ thật, 10 tab trình duyệt thật) cho hình ảnh khác hẳn `cmd/loadtest`:
  ở đó 88% CPU nằm ở lọc vật phẩm, còn ở đây 83% nằm ở so kiếm. Ba việc đã làm:
  1. Kiếm của b tính sẵn một lần mỗi bước quét — bản cũ gọi `swordAt(b…)` đúng
     `a.Count × b.Count` lần thay vì `b.Count` lần.
  2. Bỏ sớm lưỡi nằm ở phía đối diện (cách tâm đối phương quá `bán kính ngoài + nửa lưỡi +
     bề rộng`), và so khoảng cách hai **tâm** lưỡi trước khi gọi `SegSegDist2`.
  3. `swordAtRot`: hướng lưỡi suy từ hướng chuôi bằng một phép quay thay vì cos/sin lần hai;
     `RingOf` tính một lần thay vì hai; trả luôn chiều dài lưỡi để khỏi đo lại bằng `Hypot`.

  **Hai hướng đã thử và BỎ** (đo ra tệ hơn, ghi lại để khỏi thử lại):
  - Nhớ vị trí kiếm theo từng người mỗi tick → **12,01 ms**. Số bước quét `n` phụ thuộc
    *cặp* người chơi nên bộ nhớ đệm bị huỷ liên tục.
  - Dựng sẵn bảng hướng kiếm trong `ComputeRingGeom` → **9,53 ms**; chuyển sang dựng lười
    trên Player → **9,61 ms**. Lý do: bể kiếm xảy ra mỗi tick nên bảng 1 050 phần tử phải
    dựng lại liên tục, tốn hơn phần tiết kiệm được.
- **Bài test 10 tab, TRƯỚC/SAU mục 26 (28/10/2026)** — cùng kịch bản, 5 tab Đạo Tổ 1 050
  kiếm chụm giữa bản đồ, ~5 300 kiếm trên màn hình, ~1 400 entity trong AOI:

  | Chỉ số (nhóm Đạo Tổ, TB) | TRƯỚC | SAU |
  |---|---|---|
  | Tick server | 269 ms | **8,2 ms** |
  | Ping | 101 ms | 339 ms* |
  | Kích thước snapshot | 1,2 KB | 1,7 KB |
  | FPS (tab 1) | 16,2 | 4,2 |
  | Thời gian vẽ (tab 1) | 2,2 ms | 16,1 ms |

  *Ping và FPS xấu đi vì nay **nút thắt chuyển sang client**: trước đây server nghẽn nên
  client nhận ít gói và vẽ ít; giờ server trả đủ 20 Hz, mười tab cùng vẽ 5 000 sprite trên
  một card RX 570 và chính việc vẽ trở thành chỗ tắc (draw call 431). Đây chính là điều
  kiện mà kế hoạch đặt ra cho **mục 25.1 (gộp vòng kiếm thành một RenderTexture)** và
  **mục 25.6 (giải mã msgpack trong Web Worker)** — hai mục này nay có bằng chứng để làm.
- **Bài test 10 tab dùng GPU thật (28/10/2026)**: máy có AMD RX 570; đo được
  `--use-gl=angle --use-angle=vulkan` → RADV, `--use-angle=gl` → radeonsi, còn
  `--use-gl=swiftshader` và `--use-gl=egl` đều rơi về SwiftShader (CPU). Đổi sang Vulkan
  nên cả 10 tab chạy khổ 960×540 với chất lượng mặc định. Server chạy riêng trên lõi CPU
  tách khỏi trình duyệt bằng `taskset`, vì không tách thì tick 2,55 ms bị thổi lên 269 ms
  do goroutine bị hệ điều hành gạt ra liên tục.
- **`cmd/loadtest` và `cmd/stress` (28/10/2026)**: `loadtest` đo đúng một kịch bản §13.6 và in bảng đối chiếu ngân sách; `stress` quét 10 → 320 người tìm trần của máy (đo được: **160 người vẫn mượt**, 200 người lag). Cả hai chạy nguyên đường thật (ghép phòng, bot, snapshot theo tầm nhìn, mã hoá msgpack), khác `cmd/sim` vốn chỉ chạy `game.Tick` thuần.
- **Overlay F3 (28/10/2026, §13.6)**: `?debug=true` → phím **F3** bật/tắt bảng đo: fps · ms/khung · thời gian vẽ · tick server (server gửi kèm trong snapshot) · ping · KB/s · cỡ snapshot · số sprite · số draw call (bọc hàm vẽ WebGL) · số entity trong AOI · số kiếm · mức giãn kiếm.
- **Spec v4.3–v4.5 (28/10/2026)**: bỏ hẳn **Trường Kiếm** — tầm chém chỉ tăng theo số vòng và cảnh giới, `ComputeRingGeom`/`ringGeom` bỏ tham số `longMult`. Thêm **Cương Kiếm** (`hard`): độ bền va kiếm × 1.5, sát thương quái/vật cản × 2, 10s, weight 7. Bể kiếm giờ so theo **độ bền D = 1.2^r × hệ số buff** chứ không so cảnh giới, nên Luyện Khí có Cương Kiếm cứng hơn Trúc Cơ tay không; đổi chiều quay vẫn chỉ xét cảnh giới. `shield_ring` → `body_hit_max_rings: 1`; bỏ `tien_khi_to_exp_base`.
- **Thần thú Tiên giới (28/10/2026, spec v4.2 §3.5)**: VFX chuỗi đốt ở client (`render/beast.ts`) — 1 đầu + 14 đốt + 1 đuôi, đốt sau đi theo vị trí đầu lúc trước như con rắn, quỹ đạo xoắn ốc từ chân lên, đốt ở nửa sau vòng xoắn vẽ dưới nhân vật nửa trước vẽ trên. 3 kiểu: đột phá 2.5s · thường trực mỗi 20s 1.5s (phím **V** tắt/bật) · gãy vụn khi rớt cảnh giới 0.8s. Trần 6 con cùng lúc, Ẩn Thân/trong bụi không phát. Placeholder tam giác+tròn; 15 job art đã vào `assets:report` nhóm "Thần thú Tiên giới" theo ART_PROMPTS §10.1.
- **Linh khí và Tiên khí tách hẳn (28/10/2026)**: từ Đại Thừa tầng 10 trở lên chỉ hấp thụ Tiên khí, dưới đó chỉ hấp thụ Linh khí — viên không dùng được đi xuyên qua và client làm mờ 30%. Lệch §3.4 (spec còn ghi đổi 50×2^r) → **Q18**. Phím test **R** giờ cấp thẳng Tiên khí khi đã qua Đại Thừa 10, nếu không sẽ tắc không thử được Tiên giới.
- **Bể kiếm nhanh gấp 9 lần (28/10/2026)**: tách `clash_cooldown` (mới, = 0 → tính mỗi tick) khỏi `flip_cooldown` (giữ 0.3s, chỉ giới hạn đổi chiều quay). Hai vòng ngoài 30 kiếm ngang cảnh giới mài sạch nhau trong đúng 1 giây. Lệch câu "cooldown 0.3s" của §5.2 → **Q19**.
- **Người chơi không chặn đường nhau (28/10/2026)**: lá chắn §5.2 chuyển từ luật DI CHUYỂN sang luật CHÉM — đi xuyên qua nhau thoải mái, nhưng còn ≥ 2 vòng thì kiếm địch không chạm tới thân. Giả lập 40 bot/10 phút: 148 → 608 lượt chết. Hệ quả: Bó Kiếm khi rớt cảnh giới gần như không còn xảy ra → **Q17**.
- **Không ai lọt ra ngoài tường biên (28/10/2026)**: `resolveBlocking` kẹp vào vùng chơi chứ không phải cả bản đồ (trước kẹp `[r, size-r]` tức cả dải lề ngoài tường). Quái vật cũng vậy. `internal/game/bounds_test.go` giữ luật này.
- **Vành mờ vòng kiếm (28/10/2026)**: chỉ vẽ cho vòng thật sự quét kín (≥ 30 kiếm, hoặc ≥ 19 khi có Tốc Kiếm) — vòng thưa không còn tô vành mờ làm phạm vi nhìn rộng hơn thực tế.
- **Cửa sau debug (28/10/2026)**: mở `?debug=true` thì có `window.kx = { scene, world }` để soi bằng console (ví dụ `kx.scene.beastCount()`).
- **⚠ Nhịp farm đang lệch spec (22/10/2026)**: `--farm` đo được Luyện Khí → Trúc Cơ **165s**, §6.1 vẫn ghi 30–60s. Do ba thay đổi cộng lại: tốc độ chạy một nửa + EXP gấp đôi mỗi cảnh giới + mật độ giảm 50%. Đại Thừa giờ cần ≈7.7 giờ farm → bảng Thần Tốc sẽ trống. Phương án: **Q13 trong `docs/QUESTIONS.md`** — chờ chốt, chưa đổi gì.
- **EXP mỗi cảnh giới gấp đôi cảnh giới trước (22/10/2026)**: `realm_mult` 1.2 → **2.0**. Tổng tới Đại Thừa 10: 28 288 → **694 960**.
- **Mật độ giảm 50% (22/10/2026)**: `exp_orbs.cell` 160 → **226** (≈160×√2, số viên giảm một nửa), `items.count_per_room` 400 → **200**.
- **Bot sinh ra tối đa Trúc Cơ (22/10/2026)**: `bots.realm_weights` [60,30,10] → **[60,40]**. Bot vẫn farm lên cao hơn sau khi sinh — đó là bình thường.
- **⚠ Spec gốc chuyển sang `docs/_raw/` (22/10/2026)**: chỉ chủ dự án sửa, Claude không ghi vào đó. `docs/GAME_SPEC.md` cũ đã xoá để khỏi có hai bản. Bản `_raw` hiện **thiếu 13 thứ đã code hôm nay** → danh sách đầy đủ ở **Q14 trong QUESTIONS.md**, cần dán lại vào `_raw`.
- **Tiên giới (22/10/2026, spec §3.4)**: 5 cảnh giới sau Đại Thừa (Chân Tiên → Đạo Tổ), không chia tầng, chỉ lên bằng **sát mạng** (mỗi cú chém trúng thân = 1, bot = 0.5, cùng nạn nhân 60s không tính, chỉ đếm từ Đại Thừa tầng 10). Bị chém rớt 1 cảnh giới và bộ đếm về mức khởi điểm. Level 90/100/110/120/130 nên `Realm(L) = L/10` vẫn đúng; `setTotalExp` không được đụng vào level của người Tiên giới. Hào quang sau lưng mỗi cảnh giới một màu, HUD đổi thanh Linh khí → Tiên khí vàng kim.
- **Kiếm tối đa theo công thức (22/10/2026, §3.1)**: `max_swords[r] = 5(r+1)(r+2)` → 10, 30, 60, 100, 150, 210, 280, 360, 450, 550, 660, 780, 910, 1050. Cảnh giới r có đúng r+1 vòng. `ring_step` đổi tên thành `ring_capacity_step` cho khớp spec gốc.
- **Bảng xếp hạng & danh hiệu (22/10/2026, spec v3.8)**: bảng phòng xếp theo **Uy Danh** (không còn theo cảnh giới), mỗi dòng ghi **danh hiệu** thay cảnh giới; cảnh giới+tầng chuyển lên **đầu nhân vật** (dưới danh hiệu, trên tên). Danh hiệu mở rộng Top 1→10, chữ nằm trong `config.titles` + `STR.titles` (phải sửa cả hai cho khớp).
- **Tin loan toàn server (22/10/2026, §11.4)**: hạ người đang Top 5 của phòng → kill feed gửi tới mọi phòng, hiện nền đỏ viền vàng. Ngưỡng ở `serverWideTopN` trong `room/broadcast.go`.
- **Bot sinh ngang hạng 11 (22/10/2026, §9.3)**: `bots.spawn_rank` = 11. Đủ người thì bot sinh bằng level người hạng 11; chưa đủ thì Luyện Khí. **Không còn trần cảnh giới** — `bots.realm_weights` đã bị xoá khỏi config vì hạng 11 tự nó là cái phanh.
- **artgen cắt sprite lệch (sửa 22/10/2026)**: vài pixel rác sót lại sau khi xoá nền (có ảnh chỉ 3 pixel ở góc) kéo khung nội dung ra cả ảnh → sprite bị thu nhỏ và dồn sang trái. `alphaBBoxDense` giờ bỏ các mảng rời dưới 1% mảng lớn nhất. 9 skin kiếm đã prep lại, căn giữa đúng.
- **artgen đếm sai ảnh đã có (sửa 22/10/2026)**: trước đây chỉ nhìn `assets/_raw/<job>.png`, mà thư mục đó là scratch nên bị dọn → `plan` báo "0 generated" và `run` sẽ vẽ lại toàn bộ 196 ảnh, đốt sạch quota. Giờ `alreadyHave` xét cả file đã prep trong `assets/`, đồng thời tôn trọng `redoReasons` (ảnh cần vẽ lại vì đổi luật vẫn tính là thiếu). Thực tế chỉ còn **22 ảnh** cần sinh: 9 skin kiếm + 3 nền (vẽ lại theo v3/v3.1) và 10 ref_sheet ưu tiên 9.
- **Thiêu Đốt Tinh Huyết (22/10/2026, spec v3.7 §4.1c)**: giữ Shift (PC) hoặc nút 🔥 (điện thoại) khi đang chạy → tốc độ ×1.8, mỗi giây mất 8% EXP của tầng hiện tại. Không rớt cảnh giới (dừng ở đáy cảnh giới), Luyện Khí đốt cạn về 0 vẫn sống. Có vòng lửa cho cả phòng thấy. Số để chỉnh: `burn.mult`, `burn.exp_frac_per_sec`, `burn.keep_realm`.
- **Va kiếm không đẩy văng nữa (22/10/2026)**: `clash_knockback` 150 → **0**. Hai người đứng mài nhau tới khi một bên hết kiếm. Đặt lại > 0 là bật lại kiểu cũ.
- **Tốc độ chạy (22/10/2026)**: `move_base` 260 → **130** và bỏ hệ số `sqrt(scale)`. Spec dán vào sau đó cứ lùi lại về `260 * sqrt(scale)`; đang giữ theo yêu cầu nhắn trực tiếp — xem Q12.
- **Tốc độ xoay kiếm (22/10/2026)**: `rev_secs` → **1.0** (1 vòng/giây theo spec v3.4), Tốc Kiếm trả về ×1.6.
- **Số kiếm tối thiểu (22/10/2026)**: `min_count` 1 → **0** (spec v3.5). Bị tước sạch kiếm thì không chạm được ai, hiện emote `no_swords`, bot chuyển sang chạy/farm. **Không chết** — luật chết vẫn chỉ áp cho Luyện Khí bị chém vào thân.
- **Spec v3.3–v3.6 đã code xong (22/10/2026)**: điểm Uy Danh + 3 bảng xếp hạng (Uy Danh/Sát Thần/Thần Tốc), danh hiệu Top 1–3 phòng, minimap, Ẩn Thân mờ 20%, clash_unit = 1 + chỉ đổi chiều khi cùng cảnh giới, damage vật cản theo người. Cổng dịch chuyển giờ ghép cặp 2 chiều cố định, `transit_secs` = 2s. Vật phẩm nhặt được bay hút vào người 0.22s.
- Schema SQLite thêm cột `score/hits/dai_thua_secs/top1_secs`, tự nâng cấp database cũ khi mở. API đổi `metric=exp|kills` → `board=score|hits|speed`.
- Môi trường dev: `~/go/pkg/mod` có một số thư mục thuộc quyền `root` (dấu vết của lần chạy `sudo go` trước) → `go build/test` báo "permission denied". Đang né bằng `GOMODCACHE=~/.cache/kx-gomod`. Sửa tận gốc: `sudo chown -R chinhdc:chinhdc ~/go/pkg/mod`. Ngoài ra `go` không còn trong PATH, đang dùng bản trong cache: `~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.11.linux-amd64/bin/go`.

## Việc tiếp theo gợi ý
1. Đăng nhập lại minigen → chạy các đợt art ở trên.
2. Chơi thử nhiều tab / điện thoại qua LAN, ghi lại cảm giác cân bằng.
3. Deploy theo `docs/DEPLOY.md`.

## Staging (2026-10-22)
- Chạy tại https://kiemtien.staging.maveragroups.com (container `kiemtien-web` trên máy staging, Caddy chung chuyển tiếp vào).
- Deploy lại: `set -a; . docs/private/deploy; set +a; deploy/staging.sh` (build Go arm64 + client ở máy mình, rsync lên `~/kiemxoaytutien`, `docker compose up -d --build`, cài khối Caddy có kiểm tra trước khi reload).
- Bảng xếp hạng lưu ở volume `kiemtien_kiemtien-data`, không mất khi deploy lại. Server đang để `-debug=true` (người chơi vẫn phải mở `?debug=true`).

### Đo tải staging (2026-10-30)
- Browsertest qua Internet: `cd tools/browsertest && node run.mjs --url https://kiemtien.staging.maveragroups.com --tabs 7`
  → kết quả ghi `docs/perf/BROWSERTEST-staging.md` (không đè số đo trên máy). Mỗi lượt ghi file ngay khi xong.
- Chỉ chạy **7 tab**: staging giới hạn 8 kết nối mỗi IP, chừa 1 suất cho người chơi thật cùng mạng văn phòng
  (lần chạy 8 tab bị người thật chen vào → tab 8 ăn HTTP 429).
- Số đo: ping ~200 ms (tab thường) / ~250 ms (tab Đạo Tổ), xuống 7–25 KB/s mỗi người, tick server 2–3 ms.
  Container trên staging (ARM 4 nhân): CPU TB 28% / p95 40% / max 54% của 1 nhân, RAM tối đa 54 MB
  — cho 1 phòng 7 người thật + bot lấp đầy 60.
- Lỗi đã biết khi đo: nếu máy đo có container docker khởi động lại liên tục (vd. `mavera-sentry-worker` hỏng
  redis) thì Chrome báo `ERR_NETWORK_CHANGED` và tab rớt liên tục → dừng container đó trước khi đo.
  Chạy Chrome trong docker không giải quyết được vì không có GPU (vẽ bằng CPU, 800% CPU, tab 2 không mở nổi).
- Log staging có `tick chậm ~27 ms` do AI bot (25 ms) ở khu Trúc Lâm Mê Cung khi phòng không có người — cần xem.

## Đụng file giữa hai hướng (worktree song song, 30/10/2026)

Hai hướng chạy trên hai nhánh/worktree riêng (`kx-netcode` ← `netcode-2a`, `kx-anim` ←
`anim-v61`). Phân chia sở hữu để không giẫm chân:

| File | Chủ sở hữu | Ghi chú |
|---|---|---|
| `client/src/net/state.ts` | **A** | vòng đời entity, giữ 2 s |
| `client/src/render/scene.ts` — tạo/huỷ/pool/fade view | **A** | |
| `client/src/render/player.ts` — `setCharacter`, mờ vào | **A** | ⚠ **đụng**: hướng B sẽ sửa bộ điều khiển hoạt ảnh trong CÙNG file |
| `server/internal/room/bots.go`, `botnames.go` | **B** | tên bot §9.4 |
| `server/internal/config/config.go` — nạp `bot_names.txt` | **B** | |
| `tools/atlas-pack` | **B** | chưa đụng |

**`player.ts` là file duy nhất cả hai cần.** Hướng A đã merge trước (`1695b15`), hướng B
rebase sau — đúng thứ tự chủ dự án định. Khi hướng B làm bộ điều khiển hoạt ảnh, phần A
đã nằm sẵn trong `main` nên chỉ cần rebase, không phải gộp tay.

Thứ tự merge: **A (2a) ✅ → B1 tên bot ✅ → A (2b) → B hoạt ảnh** (sau khi chủ dự án duyệt
bảng điểm độ giật).

### Một cái bẫy khi chạy song song

Hai lượt `go test ./...` đỏ ở `TestRemovalReportedEvenWhenSlotReused` hoá ra **không phải
lỗi sản phẩm cũng không phải lỗi test**: tôi sửa `config/game.json` bằng `perl -i` trong
lúc test đang build, nên test đọc phải file viết dở (thiếu `pickup_r` → vòng nhặt 0 → đúng
triệu chứng "chưa nhặt được viên"). Đo lại dưới đúng điều kiện cũ: **0/4 lượt đỏ**.
→ Đừng sửa `config/` trong lúc test đang chạy.
