# KIẾM XOAY TU TIÊN — Spec tối ưu: Giao thức client–server & Cache nhiều lớp (v1)

> Đặt tại `docs/_raw/NETCODE_SPEC.md`. Spec này **thay thế** GAME_SPEC §13.3 (mạng) và bổ sung §13.6 (hiệu năng). Mọi luật chơi vẫn theo GAME_SPEC.
> Mục tiêu kép: **nhẹ hơn** (băng thông, CPU server, CPU/GPU client) và **đẹp hơn** (không nhấp nháy, không bao giờ thấy hình placeholder khi chơi thật).

---

## 0. Vấn đề hiện tại

| Triệu chứng | Nguyên nhân khả dĩ |
|---|---|
| Client nhận dữ liệu cả room dù chỉ đứng 1 khu | Room được xử lý như 1 khối; snapshot/sự kiện không tách theo khu |
| Nhân vật đã vào khung hình vẫn **nhấp nháy** | Entity bị despawn/spawn liên tục khi đứng ở mép vùng AOI; view bị hủy rồi tạo lại |
| Hiện **hình placeholder** của nhân vật | Atlas nhân vật tải lười (khi entity đầu tiên xuất hiện); texture bị giải phóng rồi tải lại; build thật vẫn bật placeholder |
| Gói tin đỉnh 38 KB | Gửi lại toàn bộ dữ liệu (tên, danh hiệu, nhân vật...) mỗi lần; item dồn 1 gói |
| Mỗi client tốn CPU server riêng | Mỗi client tự mã hóa lại cùng một entity mỗi tick |

## 1. Ngân sách mục tiêu

| Chỉ số | Mục tiêu |
|---|---|
| Băng thông xuống / client (đông người) | < 25 KB/s trung bình, < 40 KB/s đỉnh |
| Gói snapshot lớn nhất | < 6 KB |
| Server tick / khu | < 3 ms (p95 < 5 ms) |
| Khung hình có placeholder (build thật) | **0** |
| Entity trong khung nhìn bị despawn rồi spawn lại trong 2s | **0** |
| Đổi khu khi bản đồ đã cache | < 300 ms tới khi thấy khu mới |
| Đổi khu lần đầu (chưa cache) | < 1.5 s, có màn chuyển cảnh che |
| Vào trận lần 2 trở đi (asset đã cache) | < 2 s từ bấm "Vào chơi" |

---

## 2. Kiến trúc tổng quan

```
                         ┌───────────────────────── SERVER (1 process) ─────────────────────────┐
                         │                                                                        │
                         │  Room (thế giới)                                                       │
                         │  ┌──────────────── L2: Room Shared Cache ─────────────────┐            │
                         │  │ hồ sơ người chơi · bảng phòng · danh hiệu · kill feed │            │
                         │  │ dân số từng khu · sự kiện toàn room · version số       │            │
                         │  └───────▲───────────────▲──────────────▲────────────────┘            │
                         │          │ delta 2 Hz    │              │                              │
                         │   ┌──────┴─────┐  ┌──────┴─────┐  ┌─────┴──────┐   (5 khu)             │
                         │   │ Zone Shard │  │ Zone Shard │  │ Zone Shard │ ...                    │
                         │   │ L0: sim    │  │ L0: sim    │  │ L0: sim    │                        │
                         │   │ L1: encode │  │ L1: encode │  │ L1: encode │                        │
                         │   └──────┬─────┘  └────────────┘  └────────────┘                        │
                         │          │ chỉ gửi cho client đang ở khu này                          │
                         │  L3: SQLite (ghi trễ theo lô: leaderboard, mạng đã chết)              │
                         └──────────┼─────────────────────────────────────────────────────────────┘
                                    │ WebSocket (2 kênh logic: ROOM + ZONE)
                         ┌──────────▼───────────────── CLIENT ────────────────────────────────────┐
                         │ Net worker (giải mã) → Entity Store → View Pool → Renderer             │
                         │ Cache: GPU textures · RAM (views, nameplate, chunk nền) · IndexedDB/SW │
                         └────────────────────────────────────────────────────────────────────────┘
```

**Nguyên tắc**
1. **Khu (zone) là đơn vị đồng bộ.** Client chỉ nhận dữ liệu thế giới của **đúng khu mình đang đứng**. Không bao giờ nhận entity, item, viên khí của khu khác.
2. **Room chỉ đồng bộ dữ liệu chung**, nhỏ và chậm (bảng phòng, danh hiệu, kill feed, dân số khu...) qua **Room Shared Cache**.
3. **Mã hóa 1 lần, phát cho nhiều người** (encode once, fan-out).
4. **Thứ tĩnh gửi 1 lần, thứ động gửi delta.**
5. **Client không bao giờ hủy hình ảnh vội vàng** — mọi thứ đi qua pool và có thời gian chờ.

---

## 3. Server: các lớp cache

### L0 — Zone Shard (trạng thái mô phỏng, nguồn sự thật)
- Mỗi khu 1 goroutine, tick 30 Hz, giữ toàn bộ entity của khu: nhân vật, quái, item, viên khí, vật cản, cổng, đài.
- Chuyển khu (qua cổng/Thiên Môn): shard nguồn gửi bản ghi nhân vật đầy đủ qua channel sang shard đích ở ranh giới tick. Không khóa chung giữa các shard.
- Shard không biết gì về client ở khu khác.

### L1 — Bộ mã hóa trong shard (Encode Cache)
- **Descriptor cache** (dữ liệu tĩnh của entity): `{id, kind, character, name, title_id, realm, tầng, skin, color, ring_layout_version}` → mã hóa sẵn thành bytes, **có `desc_ver`**. Chỉ mã hóa lại khi một trường thay đổi (đổi hạng, đổi cảnh giới...).
- **State cache theo tick** (dữ liệu động): mỗi tick, mỗi entity "bẩn" được mã hóa **1 lần** thành khối bytes trạng thái (vị trí lượng tử, hướng, mood, ring phase/dir/count, cờ trạng thái, buff mask). Mọi client dùng chung khối này.
- **Gói theo ô lưới AOI**: khu chia ô 1 024 × 1 024; mỗi tick ghép sẵn "bản tin ô" (danh sách khối của entity trong ô). Gói cho từng client = nối các bản tin ô nằm trong vùng nhìn của họ + phần riêng (của chính họ). → chi phí theo client gần như chỉ là nối mảng bytes.
- **Tài nguyên tĩnh của khu** (bản đồ: vật cản, vùng đa giác, cổng, đài, decal seed) mã hóa 1 lần lúc tạo khu thành **map blob** nén (zstd/gzip), có `map_hash`.

### L2 — Room Shared Cache
- Một struct dùng chung cho cả room, **cập nhật bằng sự kiện từ các shard** (channel), mỗi mục có `version` tăng dần:

| Mục | Nội dung | Cập nhật xuống client |
|---|---|---|
| `profiles` | id → tên, nhân vật, guest_id, khu đang ở, cảnh giới, Uy Danh | Khi đổi (delta theo version) |
| `room_board` | Top 10 theo Uy Danh + danh hiệu Top 1–10 | 1 Hz, chỉ khi version đổi |
| `zone_pop` | Số người thật / bot mỗi khu | 0.5 Hz |
| `kill_feed` | Vòng đệm 20 tin gần nhất (kèm tên khu) | Ngay khi có |
| `room_events` | Thiên Tiên Đài sắp phun, Rương Báu, Top 3 qua Thiên Môn, tin loan toàn server | Ngay khi có |
| `top_markers` | Vị trí Top 1 / Đạo Tổ **chỉ cho client cùng khu** (minimap) | 1 Hz, theo khu |

- Client khi kết nối nhận **1 bản đầy đủ** của Room Shared Cache, sau đó chỉ nhận delta theo version. Mất kết nối → gửi lại version đang có, server trả phần thiếu.
- Shard **không gọi trực tiếp** lẫn nhau qua Room Cache; chỉ đẩy sự kiện vào, Room Cache gom và phát.

### L3 — Lưu trữ bền
- SQLite chỉ ghi **trễ theo lô** (mỗi 5s hoặc 100 bản ghi): mạng đã kết thúc, leaderboard. Không đụng DB trong vòng tick.
- Leaderboard toàn cục: cache trong RAM, làm mới từ DB mỗi 10s.

---

## 4. Giao thức mới (Protocol v2)

### 4.1 Khung gói
- WebSocket nhị phân. Mỗi gói: `[u8 type][u16 seq][u16 ack_snapshot_seq?][payload]`.
- **Đường nóng** (snapshot, input, sự kiện chiến đấu): mã hóa nhị phân tự viết bằng `DataView` (không msgpack, không tên trường).
- **Đường lạnh** (lobby, leaderboard toàn cục, map blob, Room Cache đầy đủ): msgpack dạng **mảng** (không dùng map/khóa chuỗi).
- Có `protocol_version`; client lệch version → yêu cầu tải lại trang.

### 4.2 Hai kênh logic
| Kênh | Nội dung | Nhận khi |
|---|---|---|
| `ROOM` | Room Shared Cache (full + delta), sự kiện toàn room | Luôn luôn |
| `ZONE` | Map ref, descriptor, snapshot, sự kiện chiến đấu, item, viên khí | Chỉ khu đang đứng |

### 4.3 Vòng đời entity (thay cho gửi lại toàn bộ mỗi snapshot)
| Message | Khi nào | Nội dung |
|---|---|---|
| `Spawn` | Entity vào vùng quan tâm của client lần đầu (hoặc sau khi đã despawn) | `id, desc_ver, descriptor đầy đủ, state đầy đủ` |
| `Desc` | `desc_ver` đổi (đổi hạng, cảnh giới, danh hiệu, skin) | descriptor mới |
| `Update` | Mỗi snapshot | `id + bitmask trường đổi + các trường đổi` (delta so với bản client đã **ack**) |
| `Despawn` | Rời vùng quan tâm **quá 2s**, chết, qua cổng | `id, reason` (`out_of_view`, `died`, `teleported`, `left`) |

- Client lưu **Entity Store** theo id; descriptor cache theo `(id, desc_ver)`.
- Mỗi 2s gửi **keyframe** (state đầy đủ của entity trong tầm) để tự sửa sai lệch.

### 4.4 Vùng quan tâm (AOI) có trễ — chống nhấp nháy
- **Bán kính vào** = khung nhìn + lề 400. **Bán kính ra** = khung nhìn + lề 900 (lớn hơn vào) → entity lảng vảng ở mép không bị vào/ra liên tục.
- Ra khỏi bán kính ra → **giữ thêm 2s** mới `Despawn`; vào lại trong 2s → chỉ tiếp tục `Update`, không `Spawn` lại.
- Camera đổi zoom (đột phá, lên tầng) → mở rộng AOI **trước** khi zoom chạy, thu hẹp **sau** khi zoom xong.

### 4.5 Ưu tiên & tần suất
- Mỗi entity có điểm ưu tiên: gần + to + đang giao tranh + là Top/Đạo Tổ → cao.
- Gần (½ khung nhìn) 20 Hz, xa 10 Hz, rất xa (chỉ thấy chấm nhỏ) 5 Hz.
- Ngân sách mỗi gói ~5 KB: entity ưu tiên thấp bị dời sang tick sau (không bỏ hẳn).
- Item/viên khí đứng yên: chỉ `Spawn`/`Despawn` theo sự kiện, **chia nhỏ ≤ 2 KB/gói**, gần trước. Viên Tiên khí **không bao giờ bị LOD loại**.
- Client nghẽn (bộ đệm gửi đầy) → bỏ snapshot cũ chưa gửi, chỉ giữ bản mới nhất + mọi sự kiện quan trọng (Spawn/Despawn/Desc/chiến đấu không bao giờ bị bỏ).

### 4.6 Lượng tử hóa
- Vị trí `u16` theo khu (12 000 / 65 535 ≈ 0.18), góc `u8`, scale `u8` (bước 0.02), ring phase `u16`, mood `u4`, buff `u16` bitmask, cờ trạng thái `u16` bitmask.
- Vòng kiếm: chỉ gửi `count, phase, dir, flip_tick` — client tự tính hình học (GAME_SPEC §4.1a).

### 4.7 Chuyển khu (cổng / Thiên Môn)
```
Client A bước vào cổng
 → Server (shard nguồn): event TeleportOut, A vào trạng thái "đang dịch chuyển" (không chém/không bị chém)
 → Server gửi ZoneEnter {zone_id, zone_type, map_hash, seed, exit_pos}
 → Client:
     map_hash có trong cache?  → dựng khu ngay từ cache (RAM / IndexedDB)
     chưa có                   → tải map blob (kênh ROOM, đường lạnh) + hiển thị màn chuyển cảnh mây
     preload asset riêng của khu đó (nếu chưa)
     gửi ZoneReady
 → Server (shard đích): nhận ZoneReady (hoặc hết 3s) mới đặt A vào khu, cổng đích lóe sáng 0.5s rồi A xuất hiện
 → Shard đích gửi Spawn cho mọi entity trong tầm của A
```
- Thời gian "đang dịch chuyển" kéo dài tới khi client sẵn sàng (tối đa 3s) → không ai chết vì máy tải chậm. Vẫn **không bất tử sau khi đã xuất hiện** (luật GAME_SPEC §8.4 giữ nguyên).
- Client giữ lại cache khu vừa rời (xem §5 L2) → quay lại khu cũ gần như tức thì.

### 4.8 Bảng message (tóm tắt)
C→S: `Hello{proto, build, guest_id}`, `Join`, `Input{seq, angle_u8, flags}` (30 Hz), `Ack{snapshot_seq}` (gộp trong Input), `ZoneReady{zone_id}`, `Skill`, `Ping`.
S→C (ROOM): `RoomFull`, `RoomDelta`, `RoomEvent`, `MapBlob`, `Pong`.
S→C (ZONE): `ZoneEnter`, `Spawn`, `Desc`, `Update`(trong `Snapshot`), `Despawn`, `Combat{Hit, SwordBreak, Death, RingFlip...}`, `ItemSpawn/ItemGone`, `OrbSpawn/OrbGone`, `ObstacleState`, `SelfState` (dữ liệu riêng của mình: EXP, Tiên khí, lượt kỹ năng, buff, hồi chiêu).

---

## 5. Client: các lớp cache

### L0 — Net worker & Entity Store
- **Web Worker** nhận WebSocket, giải mã, ghép delta vào bản sao Entity Store, gửi sang luồng chính dạng gói đã giải mã (transferable `ArrayBuffer`). Luồng chính không bao giờ giải mã.
- Entity Store giữ: descriptor (theo `desc_ver`), bộ đệm nội suy (3–5 snapshot gần nhất), trạng thái hiển thị.

### L1 — GPU / texture
- **Tải trước bắt buộc** ở màn hình "Đang tìm phòng..." (thanh tiến trình): toàn bộ atlas 10 nhân vật, skin kiếm 14 cảnh giới, item/buff, VFX dùng chung, font BitmapFont. Sau khi tải: **"làm ấm GPU"** — vẽ thử mỗi texture 1 lần ngoài màn hình để upload lên GPU trước khi vào trận.
- **Không bao giờ giải phóng** atlas nhân vật/kiếm trong trận.
- **Cache texture vòng kiếm dùng chung**: key `(skin, ring_index, sword_count_in_ring, tilt_dir, scale_bucket, tint)` → 1 `RenderTexture`. Nhiều người cùng cảnh giới/cùng số kiếm dùng chung 1 texture. LRU, trần 96 MB texture vòng kiếm.
- Cache texture nameplate: key `(name, title_id, realm, tầng, rank, is_self)`.

### L2 — RAM
- **View Pool**: mỗi entity có 1 "view" (sprite nhân vật, vòng kiếm, nameplate, aura, thần thú). Khi `Despawn` → view vào pool (ẩn, giữ texture), không hủy. `Spawn` lấy view từ pool. Không tạo/hủy object trong lúc chơi.
- **Cache khu**: chunk nền đã bake + dữ liệu bản đồ của **khu hiện tại và khu vừa rời** (LRU 2 khu). Khu khác giải phóng chunk nền (giữ map blob trong IndexedDB).

### L3 — Bền (trình duyệt)
- **Service Worker + Cache Storage** cho asset: tên file có hash nội dung (`a_hon.3f9c2e.png`), header `Cache-Control: immutable`. Lần chơi sau không tải lại.
- **IndexedDB** cho map blob theo `map_hash` (giữ 20 bản gần nhất) → vào lại khu cũ/room có cùng seed không cần tải.
- `asset-manifest.json` (có hash) tải mỗi lần mở game để biết cái gì đổi.

### Hiển thị — không nhấp nháy, không placeholder
1. **Build thật cấm placeholder**: placeholder chỉ bật với `?debug=true`. Asset thiếu trong build thật → dùng **nhân vật dự phòng có art thật** (1 nhân vật mặc định) và ghi log; CI chặn build nếu `assets:report` còn thiếu asset bắt buộc.
2. Entity chỉ được vẽ khi **descriptor + atlas đã sẵn sàng** (vì đã tải trước nên luôn sẵn sàng).
3. **Nội suy thích ứng**: bộ đệm 100 ms, tự tăng tới 200 ms khi jitter cao; thiếu dữ liệu → ngoại suy tối đa 100 ms rồi đứng yên; tuyệt đối không "nhảy" trừ khi là dịch chuyển thật.
4. Vào khung hình: nhân vật **mờ dần vào** 120 ms (không bật đột ngột); rời: mờ dần ra 150 ms rồi mới về pool.
5. Đổi descriptor (đổi danh hiệu, cảnh giới): đổi tại chỗ có hiệu ứng, không tạo lại view.
6. Vòng kiếm đổi số kiếm: texture mới được chuẩn bị **trước** rồi mới thay (double-buffer), không có khung hình trống.

---

## 6. Đo đạc (bổ sung overlay F3)
- Spawn/Despawn mỗi giây, **số entity respawn trong 2s** (phải = 0), **số khung có placeholder** (phải = 0).
- Texture upload mỗi giây, bộ nhớ texture (MB), tỉ lệ trúng cache texture vòng kiếm / nameplate.
- Byte/s theo kênh ROOM và ZONE, kích thước gói lớn nhất 10s gần nhất.
- Thời gian đổi khu (bước vào cổng → thấy khu mới), có/không có cache.
- Server: tick ms **theo từng khu**, ms mã hóa L1, số byte encode/tick, số client mỗi khu.

## 7. Bài kiểm tra chấp nhận
- [ ] Client ở khu A không nhận bất kỳ gói ZONE nào của khu B (kiểm bằng log gói).
- [ ] 10 tab Chrome headless rải 5 khu + `cmd/loadtest`: đạt bảng §1.
- [ ] Bot đi qua lại đúng mép khung nhìn 30s: 0 lần respawn, không nhấp nháy.
- [ ] Build thật, vào trận, đi khắp 5 khu: 0 khung placeholder (F3).
- [ ] Đổi khu đã cache < 300 ms; chưa cache < 1.5 s có màn mây; không ai bị chém trong lúc đang tải khu.
- [ ] Mất mạng 5s rồi nối lại: nhận đúng phần Room Cache thiếu, khu hiện tại đồng bộ lại bằng keyframe, không phải vào trận lại.
- [ ] Vào trận lần 2: không tải lại asset (Service Worker), < 2 s.

## 8. Thứ tự triển khai (mỗi bước đo trước/sau, ghi PROGRESS.md)
1. **Tách khu cho mạng**: kênh ROOM/ZONE, client chỉ nhận khu đang đứng, Room Shared Cache (L2 server).
2. **Chống nhấp nháy & placeholder** (ưu tiên vì ảnh hưởng hình ảnh nhất): tải trước + làm ấm GPU, View Pool, AOI có trễ + giữ 2s, mờ vào/ra, cấm placeholder ở build thật.
3. **Vòng đời entity + descriptor cache + delta theo ack + keyframe** (Protocol v2 đường nóng nhị phân). Giữ Protocol v1 song song sau cờ `proto=1` cho tới khi v2 ổn.
4. **Encode once / gói theo ô lưới** (L1 server).
5. **Chuyển khu có ZoneReady**, cache map blob (IndexedDB), cache 2 khu trong RAM.
6. **Net Web Worker**, texture vòng kiếm dùng chung, nameplate cache.
7. **Service Worker + asset có hash**.
8. Chạy lại toàn bộ bài kiểm tra §7, gửi bảng tổng kết.
