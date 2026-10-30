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
