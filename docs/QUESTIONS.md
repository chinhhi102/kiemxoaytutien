# Câu hỏi đang chờ anh trả lời

Mỗi câu: bối cảnh → phương án → **tạm dùng** (đã code theo phương án này, đổi lại dễ).
Câu đã xong ở `docs/QUESTIONS_DONE.md`.

---

*(Không còn câu nào đang chờ — tính tới 30/10/2026.)*

Q20, Q22, Q25 và Q26 anh đã trả lời trong `docs/_raw/ANSWERS.md` ngày 30/10; code đã làm
theo và cả bốn đã chuyển sang `QUESTIONS_DONE.md`. Hai việc "chờ anh quyết" — tối ưu
CHANGES_v4 mục 25–27, và hai thư viện `pixi-filters` / `revolt-fx` — cũng đã được duyệt,
xem cuối `QUESTIONS_DONE.md`.

**Việc lớn tiếp theo** (không phải câu hỏi), cập nhật 30/10 chiều:
**NETCODE bước 3 — Protocol v2** (NETCODE_SPEC v1.2): schema chung + generator + bài đối
chiếu golden/fuzz TRƯỚC, rồi vòng đời `Spawn/Desc/Update/Despawn` tường minh, delta theo ack,
keyframe 2s, lượng tử hoá §4.6. Mục tiêu: đóng nốt respawn (về 0) và gói < 6 KB.

Tối ưu CHANGES_v4 mục 25–27 **đã xong** (mốc chuẩn #4 trong PERF_LOG). Hướng A (một thân
nhiều mặt) để sau cùng — art thử đã sinh cho 3 nhân vật, chưa code.
