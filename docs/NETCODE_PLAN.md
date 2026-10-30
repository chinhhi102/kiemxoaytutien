# Báo cáo trước khi làm v6.0 (NETCODE_SPEC)

CHANGES_v6 mục 1 yêu cầu gửi trước ba thứ. Đây là ba thứ đó.
Viết ngày 30/10/2026, đối chiếu với mã tại commit `d132a60`.

---

## (a) Nguyên nhân THẬT của nhấp nháy và placeholder

Spec §0 liệt kê "nguyên nhân khả dĩ". Tôi đo và đọc mã để tìm nguyên nhân thật, và cả hai
triệu chứng hoá ra là **một lỗi duy nhất với hai hệ quả**.

### Chuỗi nhân quả

1. `server/internal/room/broadcast.go` **cố ý bỏ qua** người/quái ở xa trên các tick lẻ, để
   họ chỉ được gửi 10 Hz thay vì 20 Hz (tối ưu mục 27 — "20/10 Hz theo khoảng cách"):

   ```go
   slow := z.tick%farRateDivisor != 0
   if known && slow && p.ID != viewer.ID && p.Pos.Dist(viewer.Pos) > half {
       continue
   }
   ```

2. `client/src/net/state.ts` coi **vắng mặt trong một gói = đã rời đi**:

   ```ts
   for (const id of this.players.keys()) {
     if (!seen.has(id)) { this.players.delete(id); }
   }
   ```

3. `client/src/render/scene.ts` **huỷ hẳn** sprite khi id biến mất một khung:

   ```ts
   if (!seen.has(id)) { sp.destroy(); this.players.delete(id); }
   ```

Ghép lại: **mọi người chơi/quái ở xa hơn nửa khung nhìn bị huỷ và dựng lại 10 lần mỗi
giây.** Đó là nhấp nháy, và cũng là một khoản CPU/GC đáng kể mà không ai ngờ tới.

### Placeholder là hệ quả của chính lỗi trên, không phải lỗi riêng

Tên và nhân vật (`nm`, `ch`) chỉ gửi **lần đầu client thấy một người** — server nhớ bằng
`c.knownPlayers[p.ID]`. Nhưng khi client tự huỷ view, server **không biết**: nó vẫn tin là
client đã biết người đó, nên không gửi lại `ch`.

View mới dựng lại với `char: ''` → `assets.atlas('characters/')` trả `null` →
`PlayerSprite` rơi vào `placeholderBody()` (vòng tròn + chữ cái đầu). Phải chờ tới nhịp gửi
lại tên định kỳ (`nameResendSecs = 2.0`) mới có hình thật.

→ **Không phải "atlas tải lười" hay "build thật bật placeholder"** như §0 phỏng đoán. Tôi đã
kiểm: cả 10 atlas đều có trên đĩa, nằm trong `manifest.json`, và `assets.preload()` tải hết
**trước** khi vào trận. Placeholder xuất hiện vì **thiếu tên nhân vật**, không vì thiếu ảnh.

### Số đo (10 tab bỏ qua, đo 1 tab + 8 bot cạnh mình, 15 giây)

| Chỉ số | Đo được | Ngân sách §1 |
|---|---|---|
| Entity biến mất rồi quay lại trong 2 s | **8 / 8 lần** | 0 |
| Nhân vật đang vẽ bằng placeholder | **1 / 3** | 0 |

Công cụ: `tools/browsertest/flicker.mjs` (chạy một lệnh, in ra bảng trên).

⚠ Phép đo này **chập chờn theo cách bố trí**: lượt chạy đầu tôi đứng một mình giữa khu, ra
"0 nhấp nháy" và suýt kết luận là không có lỗi. Phải có entity đứng gần mép khung nhìn thì
mới đo được. Khi làm bước 2 tôi sẽ cho bot đi qua lại đúng mép, như bài kiểm tra §7 đòi.

### Hệ quả cho thứ tự làm

Spec §8 xếp bước 2 (chống nhấp nháy) sau bước 1 (tách khu). Nhưng gốc của nhấp nháy **không
nằm ở việc tách khu** — nó nằm ở chỗ client suy ra "đã rời đi" từ "vắng mặt". Có một bản vá
**nhỏ và độc lập hẳn**: client giữ entity thêm 2 giây thay vì xoá ngay khi vắng mặt một gói.
Không cần Protocol v2, không cần đổi server, vài chục dòng.

**Đề nghị: làm bước 2 trước**, đúng như CHANGES_v6 mục 2 cho phép — nó sửa đúng thứ anh
nhìn thấy, rẻ, và không chặn bước nào khác.

---

## (b) Kế hoạch chia nhỏ bước 1–7 theo mã hiện có

Mỗi bước một commit riêng, đo trước/sau, ghi PROGRESS.md — như CHANGES_v6 yêu cầu.

### Bước 2a (làm trước) — chống nhấp nháy, phần rẻ
- `state.ts`: entity vắng mặt thì đánh dấu `lastSeen`, chỉ xoá sau 2 s. Giữ keyframe cuối.
- `scene.ts`: `destroy()` → trả view vào **pool**; `Spawn` lấy lại từ pool.
- Mờ vào 120 ms / mờ ra 150 ms (§5 Hiển thị 4).
- Server: khi gửi lại một người sau nhịp 10 Hz, **gửi kèm `ch`** nếu client có thể đã quên —
  hoặc đơn giản hơn: bỏ hẳn việc suy "đã rời đi" ở client thì vấn đề tự hết.
- Đo lại bằng `flicker.mjs` với bot đi qua lại mép khung nhìn 30 s → mục tiêu 0.

### Bước 2b — cấm placeholder ở build thật
- Placeholder chỉ bật khi `?debug=true`; thiếu asset → dùng nhân vật dự phòng có art thật.
- Thêm `npm run check:assets` chặn build nếu thiếu asset bắt buộc (CHANGES_v6 mục 5).
  Hiện **328/328 asset đã có**, nên bật cổng chặn này ngay được mà không chặn ai.
- F3 thêm bộ đếm "khung có placeholder" (§6).

### Bước 1 — tách kênh ROOM / ZONE + Room Shared Cache
Phần lớn đã có: mỗi khu đã là một `Game` riêng với goroutine riêng, và `snapshotFor` chỉ
duyệt `z.G` nên **client vốn đã không nhận entity của khu khác**. Việc còn lại:
- Gom bảng phòng, kill feed, dân số khu, danh hiệu vào một `RoomCache` có `version`.
- Tách message thành hai kênh logic; client nhận full một lần rồi chỉ nhận delta.
- Bài kiểm tra §7 dòng 1 (log gói) thành test tự động.

### Bước 3 — vòng đời entity + Protocol v2 (đường nóng nhị phân)
- `Spawn / Desc / Update / Despawn` thay cho "gửi lại cả danh sách mỗi gói".
- Delta theo `ack`, keyframe mỗi 2 s.
- Chạy song song v1 sau cờ `proto=1` (CHANGES_v6 mục 3).
- **Cần một bài đối chiếu mã hoá/giải mã** giống `cmd/collisionfixture`: server sinh gói,
  client giải mã, so từng trường. Viết tay `DataView` hai phía mà không có bài đối chiếu
  thì lệch một byte là đọc rác, và đó đúng là loại lỗi làm game đứng hình không báo gì.

### Bước 4 — encode once / gói theo ô lưới
- `L1` trong mỗi shard: descriptor cache + state cache theo tick + bản tin ô 1024².
- Đo bằng pprof: hiện `diffItems` + `snapshotFor` là phần lớn chi phí mỗi client.

### Bước 5 — ZoneReady + cache map blob
- `ZoneEnter/ZoneReady`, giữ "đang dịch chuyển" tới khi client sẵn sàng (tối đa 3 s).
- IndexedDB theo `map_hash`; RAM giữ 2 khu.

### Bước 6 — Net Web Worker + texture vòng kiếm dùng chung + nameplate cache
### Bước 7 — Service Worker + asset có hash

---

## (c) Chỗ spec không khớp mã, hoặc tôi thấy cần bàn

1. **Bước 1 nhỏ hơn spec tưởng.** §0 ghi "client nhận dữ liệu cả room dù chỉ đứng 1 khu".
   Đọc mã thì không đúng nữa: từ bản 5 khu (29/10), `snapshotFor` chỉ duyệt `z.G` của khu
   người đó. Cái còn thiếu là **Room Shared Cache có version**, không phải việc lọc entity.

2. **§4.6 lượng tử hoá vị trí `u16` theo khu (0,18 đơn vị)** — hôm nay tôi đã lượng tử hoá
   quái và vật phẩm nhưng dùng **đơn vị thế giới thẳng** (sai số 0,5), và **cố ý giữ float
   cho người chơi**: client dùng vị trí server để đối chiếu phần tự dự đoán (§8.2a), thêm
   sai số vào đó là bộ đếm "hiệu chỉnh/giây" vừa kéo về 0 sẽ chạy lại. Với thang 0,18 của
   spec thì sai số đủ nhỏ để áp cho cả người chơi — tôi sẽ đổi sang đúng thang spec ở bước 3
   và đo lại bộ đếm hiệu chỉnh để chắc.

3. **Ngân sách §1 chặt hơn §13.6.** Snapshot < 6 KB (cũ 8 KB) và tick/khu < 3 ms. Sau đợt
   tối ưu hôm nay đang là **6,4–7,3 KB** và tick cả phòng 1,45–1,70 ms. Phần KB còn lại cần
   đúng Protocol v2 nhị phân mới đạt — msgpack có tên trường vẫn tốn ~35% mỗi entity.

4. **"Trần 96 MB texture vòng kiếm" (§5 L1) quá nhiều cho điện thoại.** Máy trong ảnh chụp
   của anh là máy tầm trung; 96 MB texture riêng cho vòng kiếm có thể làm trình duyệt huỷ
   context WebGL (chúng ta đã có cơ chế tự khôi phục, nhưng tốt nhất đừng chạm tới). Đề nghị
   **trần theo thiết bị**: 96 MB trên PC, 24–32 MB trên điện thoại, đưa vào `config`.

5. **Service Worker (bước 7) có một cái bẫy đã cắn ta một lần.** Cache bền dễ phục vụ bản
   build cũ; hôm 29/10 client cũ gặp server mới đã làm game đứng hình, và chúng ta phải thêm
   `ProtocolVersion`. Nếu làm SW thì phải giữ nguyên cổng chặn version và cho SW **luôn** lấy
   `asset-manifest.json` từ mạng trước.

6. **Bước 3 nên có bài đối chiếu hai chiều trước khi xoá v1**, không phải sau. Xem bước 3 ở
   trên — đây là chỗ duy nhất trong kế hoạch mà một lỗi nhỏ gây hỏng im lặng.

7. **Một mâu thuẫn nhỏ trong thứ tự §8**: bước 2 đòi "AOI có trễ + giữ 2 s", mà làm đúng
   nghĩa thì cần `Despawn` tường minh — tức là một phần của bước 3. Bản vá rẻ ở (a) tránh
   được chuyện này: client tự giữ 2 s mà không cần server đổi gì. Tôi sẽ làm bản rẻ trước,
   rồi bước 3 thay bằng `Despawn` tường minh.
