# Bảng điểm "độ giật" hoạt ảnh — gửi chủ dự án duyệt

> Sinh bằng `cd server && go run ./cmd/artgen jitter`. Đây là **việc số 5 của CHANGES_v6_1**:
> đo trước, chờ anh duyệt, chưa sửa hoạt ảnh gì cả.
> Ngày đo: 2026-10-30 · 149 animation của 10 nhân vật + 6 quái.

## Cách đọc bảng này

Spec (§12.5 B.5) định nghĩa điểm là: so từng cặp frame liên tiếp theo **lệch tâm · lệch chiều
cao · khác màu**, quy về 0–100, cao là tệ. Tôi làm đúng vậy, rồi phát hiện một vấn đề và phải
tách bảng làm hai:

**Phép đo của spec chỉ dùng được cho anim LẶP.** Với anim một lần nó báo đỏ **94/102** — gần
như tất cả — nên bảng mất sạch giá trị. Lý do không phải là art tệ: `death` vốn PHẢI biến hình
mạnh (nhân vật bẹp xuống rồi hồn trắng bay lên), nên mọi cặp frame của nó đều vượt mọi ngưỡng.
Phép đo không phân biệt được "biến hình đúng ý đồ" với "frame sinh hỏng".

Nên với nhóm một lần tôi thêm một cột: **lạc lõng** — bước nhảy của một frame lớn gấp mấy lần
bước nhảy *thông thường của chính animation đó*. Thang này tự co giãn: animation biến hình đều
thì không frame nào nổi lên; chỉ frame sinh hỏng mới nhảy vọt. Đó đúng là thứ spec cần để
"đề xuất sinh lại RIÊNG frame đó".

⚠ **Ngưỡng quy đổi (đỏ từ 45) là do tôi chọn, spec không quy định.** Xin anh đọc **thứ
hạng** chứ đừng đọc con số tuyệt đối. Nếu anh thấy một anim bị chấm sai so với mắt mình, nói
tôi biết và tôi chỉnh ngưỡng.

## 1. Anim LẶP — đọc cột "TB" (3/47 vượt ngưỡng)

Đây là nhóm **đáng sửa trước**: chúng chạy liên tục suốt trận nên giật là thấy ngay.

| Nhân vật | Anim | Khung | TB | Tệ nhất (cặp) |
|---|---|---|---|---|
| co_ba_ganh | `idle` | 4 | **77.8** ❌ | 100.0 |
| dep_lao | `idle` | 4 | **57.9** ❌ | 80.7 |
| su_phu_mi | `run` | 8 | **48.9** ❌ | 70.5 |
| noi_com | `idle` | 4 | **42.9** | 57.8 |
| dai_ca_dep | `run` | 8 | **38.8** | 61.7 |
| dan_em | `run` | 8 | **36.6** | 77.8 |
| su_phu_mi | `idle` | 4 | **33.9** | 35.6 |
| ba_tam_loa | `run` | 8 | **32.8** | 58.5 |
| cu_rua | `flee` | 8 | **31.5** | 85.1 |
| vua_ghe | `idle` | 4 | **29.2** | 71.7 |
| muoi | `idle` | 4 | **29.0** | 67.7 |
| be_ti | `idle` | 4 | **28.4** | 56.0 |
| be_ti | `run` | 8 | **28.1** | 41.3 |
| cu_rua | `run` | 8 | **27.6** | 37.4 |
| co_ba_ganh | `chase` | 8 | **25.2** | 56.1 |
| co_ba_ganh | `flee` | 8 | **25.2** | 50.0 |
| ong_ba_vong | `run` | 8 | **23.6** | 38.3 |
| a_hon | `idle` | 4 | **22.7** | 27.2 |
| cu_rua | `idle` | 4 | **20.5** | 30.2 |
| a_hon | `flee` | 8 | **20.0** | 28.1 |
| map_tra_sua | `idle` | 4 | **19.2** | 28.4 |
| shipper | `run` | 8 | **18.9** | 37.4 |
| ong_ba_vong | `idle` | 4 | **18.4** | 26.1 |
| a_hon | `run` | 8 | **17.8** | 32.4 |
| ba_tam_loa | `idle` | 4 | **17.7** | 21.8 |
| shipper | `idle` | 4 | **17.2** | 29.1 |
| be_ti | `chase` | 8 | **16.1** | 35.2 |
| ba_tam_loa | `flee` | 8 | **16.0** | 50.9 |
| ba_tam_loa | `chase` | 8 | **14.8** | 26.9 |
| ga_hoang | `idle` | 4 | **14.1** | 18.7 |
| cu_rua | `chase` | 8 | **13.8** | 21.9 |
| dai_ca_dep | `idle` | 4 | **13.5** | 22.1 |
| shipper | `flee` | 8 | **13.2** | 30.2 |
| map_tra_sua | `run` | 8 | **10.3** | 22.4 |
| dai_ca_dep | `chase` | 8 | **9.0** | 28.6 |
| banh_bao_cau | `idle` | 4 | **7.5** | 21.3 |
| co_ba_ganh | `run` | 8 | **7.3** | 12.5 |
| a_hon | `chase` | 8 | **6.2** | 14.7 |
| ong_ba_vong | `chase` | 8 | **5.9** | 18.1 |
| be_ti | `flee` | 8 | **5.4** | 14.0 |
| su_phu_mi | `chase` | 8 | **3.8** | 12.6 |
| su_phu_mi | `flee` | 8 | **3.6** | 16.2 |
| shipper | `chase` | 8 | **3.5** | 18.0 |
| dai_ca_dep | `flee` | 8 | **3.1** | 14.0 |
| ong_ba_vong | `flee` | 8 | **2.8** | 7.1 |
| map_tra_sua | `chase` | 8 | **2.1** | 6.1 |
| map_tra_sua | `flee` | 8 | **1.5** | 5.2 |

## 2. Anim MỘT LẦN — đọc cột "lạc lõng" (2/102 có frame hỏng)

Cột "TB" ở nhóm này **không có ý nghĩa** (xem phần "cách đọc"). Chỉ xem cột lạc lõng, và chỉ
những dòng trên cùng.

| Nhân vật | Anim | Khung | Lạc lõng | Frame đáng sinh lại | TB (bỏ qua) |
|---|---|---|---|---|---|
| a_hon | `teleport_in` | 4 | **68.5** ❌ | `teleport_in_01` | 43.1 |
| ong_ba_vong | `realm_drop` | 6 | **45.2** ❌ | `realm_drop_05` | 53.3 |
| be_ti | `levelup` | 6 | **39.2** | `levelup_04` | 50.4 |
| map_tra_sua | `realm_drop` | 6 | **39.0** | `realm_drop_02` | 36.2 |
| a_hon | `skill` | 6 | **35.3** | `skill_04` | 37.7 |
| ba_tam_loa | `realm_drop` | 6 | **33.2** | `realm_drop_05` | 49.2 |
| ong_ba_vong | `skill` | 6 | **31.0** | `skill_01` | 45.3 |
| map_tra_sua | `levelup` | 6 | **28.7** | `levelup_04` | 48.8 |
| be_ti | `teleport_in` | 8 | **27.7** | `teleport_in_06` | 29.3 |
| shipper | `levelup` | 6 | **20.1** | `levelup_01` | 43.2 |
| dai_ca_dep | `skill` | 6 | **19.5** | `skill_04` | 26.6 |
| ba_tam_loa | `levelup` | 6 | **19.3** | `levelup_01` | 55.1 |
| shipper | `breakthrough` | 10 | **16.8** | `breakthrough_06` | 54.4 |
| co_ba_ganh | `skill` | 6 | **13.0** | `skill_05` | 23.9 |
| be_ti | `skill` | 8 | **11.6** | `skill_05` | 24.7 |
| muoi | `death` | 4 | **10.4** | `death_01` | 65.5 |
| be_ti | `teleport_out` | 6 | **8.9** | `teleport_out_04` | 43.3 |
| dai_ca_dep | `hit` | 8 | **4.7** | `hit_02` | 35.6 |
| shipper | `respawn` | 6 | **4.6** | `respawn_01` | 44.5 |
| banh_bao_cau | `hurt` | 4 | **3.5** | `hurt_02` | 51.0 |

_(còn 82 dòng nữa, điểm thấp hơn — xem `assets/jitter.json`)_

## 3. Hai nhân vật tệ nhất (spec mục 9 yêu cầu GIF trước/sau của hai con này)

Theo nhóm anim lặp:

1. **co_ba_ganh** — `idle` điểm 77.8
2. **dep_lao** — `idle` điểm 57.9

## 4. Tôi cần anh quyết gì

1. **Ngưỡng đỏ 45 có hợp lý không?** Anh mở game bằng `?debug=true` rồi bấm **`J`** —
   chữ xanh trên đầu mỗi nhân vật hiện `tên anim · frame · pha · điểm giật`, đỏ khi vượt
   ngưỡng. Nhìn vài con rồi nói tôi "chấm nặng tay quá" hay "nhẹ tay quá".
2. **Đồng ý tách hai nhóm như trên không?** Nếu anh muốn giữ đúng một con số duy nhất như
   spec viết, tôi làm lại — nhưng khi đó bảng sẽ báo đỏ 94/102 anim một lần.
3. **Sửa theo thứ tự nào?** Spec xếp C (bộ điều khiển, mục 6) trước B (chuẩn hoá frame, mục 7).
   Bảng này cho thấy phần lớn điểm cao đến từ frame lệch nhau — tức là **B** mới chữa gốc, còn
   **C** chỉ làm mượt phần nhìn. Anh muốn giữ thứ tự C → B, hay đảo lại B → C?

**Tôi dừng ở đây chờ anh**, đúng như CHANGES_v6_1 mục 5 yêu cầu — chưa đụng vào hoạt ảnh.
