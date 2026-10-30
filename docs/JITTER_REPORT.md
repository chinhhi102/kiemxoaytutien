# Bảng điểm "độ giật" — KẾT QUẢ SAU BƯỚC B

> Cập nhật 2026-10-30. Anh đã duyệt: giữ ngưỡng 45, tách 2 nhóm,
> thứ tự **B → C → A**. Đây là kết quả B (chuẩn hoá frame) + sinh lại art hỏng.

## Tóm tắt

| Nhóm | n | Trước | Sau | Δ | Đỏ trước | Đỏ sau |
|---|---|---|---|---|---|---|
| Anim lặp (điểm TB) | 47 | 20.5 | **15.7** | -24% | 3 | **1** |
| Anim một lần (lạc lõng) | 102 | 4.8 | **2.0** | -57% | 2 | **0** |

**Không animation nào tệ đi** — xem phần "luật tự kiểm" bên dưới.

## Cải thiện nhiều nhất (anim lặp)

| Nhân vật | Anim | Trước | Sau |
|---|---|---|---|
| co_ba_ganh | `idle` | 77.8 | **4.4** |
| su_phu_mi | `run` | 48.9 | **28.8** |
| dai_ca_dep | `run` | 38.8 | **24.0** |
| su_phu_mi | `idle` | 33.9 | **20.8** |
| be_ti | `run` | 28.1 | **16.3** |
| shipper | `run` | 18.9 | **8.2** |
| a_hon | `flee` | 20.0 | **9.9** |
| dan_em | `run` | 36.6 | **26.9** |

## Còn đỏ: 1

- **dep_lao `idle`** — 56.8

`dep_lao` là quái hình dép. Ảnh onion-skin (`docs/anim_preview/dep_lao_idle_sau_onion.png`)
cho thấy **thân trùng khớp tốt**, chỉ có cặp mắt chạy lung tung giữa các frame. Rất có thể đó
là hoạt ảnh **cố ý** (đảo mắt), không phải lỗi — nhưng máy không phân biệt được "mắt đảo có
chủ ý" với "mắt vẽ lệch". **Cần anh nhìn bằng mắt**: bấm `J` rồi `]` tới `dep_lao idle`.

## Hai việc lớn nhất trong bước B

### 1. `co_ba_ganh idle` 77.8 → sạch — và nó KHÔNG phải lỗi art

Animation tệ nhất toàn bộ art. Ảnh onion-skin lột ra nguyên nhân mà điểm số không nói được:
**một frame chứa CẢ BỐN nhân vật, ba frame còn lại là mảnh vụn cao 5–12 px.**

Bộ dò cắt-sai cũ cho qua vì nó chỉ **ĐẾM** frame: 3 mảnh vụn + 1 khối = "đúng 4 frame, hợp
lệ". Em thêm phép kiểm mới (`warnFrameSizes`): frame nhỏ hơn 20% frame trung vị của cùng
animation là cắt hụt. Nó bắt được ngay, sinh lại sheet, và giờ onion-skin ra **một nhân vật
nét rõ** thay vì bốn bóng ma rời nhau.

So trực tiếp:
- `docs/anim_preview/co_ba_ganh_idle_HONG_truoc_khi_sinh_lai.png` — bốn bóng ma rời nhau
- `docs/anim_preview/co_ba_ganh_idle_sau_onion.png` — một nhân vật nét rõ

Đây là loại lỗi đáng sợ: nó **không báo gì cả**, build vẫn xanh, chỉ có người chơi thấy nhân
vật nhấp nháy.

### 2. Áp spec nguyên văn sẽ phá hỏng art — hai chỗ

**Mục B.2 ("co giãn mọi frame về chiều cao trung vị")** chạy nguyên văn ra **co giãn 896%,
dịch 4 116 px trên canvas 128 px**. Vì `death` cố ý biến hình: nhân vật bẹp xuống rồi hồn nhỏ
bay lên — ép về cùng chiều cao là thổi cái hồn to bằng người. Em đặt trần ±15%: quá trần thì
KHÔNG bóp méo, mà ghi vào danh sách sinh lại.

**Neo chân theo trung điểm hai chân** là phép dành cho nhân vật hình người. Đo ra: `run` của
người cải thiện mạnh, nhưng `idle` của **rùa, cua, gà** TỆ ĐI gấp đôi — chúng không có hai
bàn chân ở đáy để neo.

Thay vì đi liệt kê tay con nào hình người (sẽ sai, và sẽ lỗi thời khi anh thêm nhân vật),
em cho máy **chấm điểm cả hai bản rồi giữ bản tốt hơn**. 25 animation tự bị bỏ qua. Luật này
bảo đảm chuẩn hoá **không bao giờ làm hỏng thêm thứ gì** — đó là lý do cột "tệ đi" bằng 0.

## Lệnh

```
tools/atlas-pack              # cắt → chuẩn hoá → ghép atlas → chấm lại điểm
tools/atlas-pack --score      # chỉ chấm điểm
tools/atlas-pack --preview    # GIF + onion-skin của anim còn đỏ
```

Trong game: `?debug=true` → `J` bật số liệu, `[` / `]` đi qua từng animation theo thứ tự bảng
điểm, phát lặp 0.5× giữa màn hình.

## Còn lại

- 5 tile vẫn còn sót nền magenta sau 2 lượt sinh lại (`tile_damp_earth`, `tile_dusty_earth`,
  `tile_mossy_stone`, `tile_rocky_ground`, `tile_village_grass`). Bộ sinh ảnh cứ để hở viền.
  Hiện đã vá tạm bằng màu trung bình nên trong game không thấy vệt hồng.
- **C (bộ điều khiển hoạt ảnh) đã làm xong** — crossfade 100 ms, giữ pha khi đổi
  run↔chase↔flee, nhịp chân theo tốc độ thật, lật có co ngang, nghiêng khi rẽ, nhún theo
  bước, thở khi đứng. Chưa đo được bằng số vì đây là cảm giác chứ không phải điểm —
  **nhờ anh vào chơi thử và nói thấy thế nào.**

