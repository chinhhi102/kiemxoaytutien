# Câu hỏi đang chờ anh trả lời

Mỗi câu: bối cảnh → phương án → **tạm dùng** (đã code theo phương án này, đổi lại dễ).
Câu đã xong ở `docs/QUESTIONS_DONE.md`.

---

*(Không còn câu nào đang chờ — tính tới 30/10/2026.)*

Q20, Q22, Q25 và Q26 anh đã trả lời trong `docs/_raw/ANSWERS.md` ngày 30/10; code đã làm
theo và cả bốn đã chuyển sang `QUESTIONS_DONE.md`. Hai việc "chờ anh quyết" — tối ưu
CHANGES_v4 mục 25–27, và hai thư viện `pixi-filters` / `revolt-fx` — cũng đã được duyệt,
xem cuối `QUESTIONS_DONE.md`.

**Việc lớn tiếp theo** (không phải câu hỏi): tối ưu theo đúng thứ tự anh chốt — mạng
(mục 27) → server (mục 26, pprof) → client phần rẻ (mục 25.1: gộp vòng kiếm thành texture,
BitmapText, cull). Kèm đo lại `loadtest` và 10 tab Chrome headless ở cả mạng thường lẫn 4G
yếu, ghi số trước/sau.

## Bó Kiếm chưa có job art (30/10/2026)

**Bối cảnh**: đang làm item art sau khi anh báo "item cho skill còn thiếu". Rà hết 13 loại
vật phẩm thì thấy **Bó Kiếm là loại duy nhất không có ảnh** — nó đang vẽ bằng Graphics (ba
lưỡi chéo buộc dây đỏ). `ART_PROMPTS.md` không có job nào cho nó.

**Em đã làm**: thêm job `item_bundle` → `items/bundle.png` vào `cmd/artgen/catalog.go`, prompt
đồng bộ với `item_sword` (bong bóng xà phòng bóng, bên trong là ba thanh kiếm nhỏ phát sáng
bắt chéo buộc dải vải đỏ), cỡ 128 px như kiếm vì vẽ ra cùng kích thước.

**Cần anh chốt**: có đưa `item_bundle` vào `ART_PROMPTS.md` không? Nếu anh muốn Bó Kiếm giữ
kiểu vẽ tay như hiện tại thì nói em bỏ job đi — em chưa sinh ảnh, chỉ mới thêm job.

Hai loại còn lại cố ý vẽ tay và em KHÔNG đụng vào: `sword` (đã có ảnh, dùng nhánh riêng) và
`tien_khi` (viên vàng kim đổi kích thước + vành ngoài theo giá trị, ảnh tĩnh không tả được).
