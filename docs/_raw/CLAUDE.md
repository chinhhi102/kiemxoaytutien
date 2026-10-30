# CLAUDE.md — Kiếm Xoay Tu Tiên

Game multiplayer .io 2D trên trình duyệt: kiếm bay vòng quanh nhân vật, tu luyện 9 cảnh giới.
Người sở hữu dự án KHÔNG rành setup kỹ thuật → giải thích ngắn gọn bằng tiếng Việt, đưa lệnh cụ thể để chạy.

## Tài liệu (đọc trước khi làm)
- `docs/GAME_SPEC.md` — luật chơi và kiến trúc. Đây là nguồn sự thật.
- `docs/ART_PROMPTS.md` — quy chuẩn asset (canvas 512, pivot chân y=460, x=256; tên file).
- `docs/PROGRESS.md` — tiến độ. Cập nhật cuối mỗi phiên.

## Stack
- Server: Go, `github.com/coder/websocket`, `github.com/vmihailenco/msgpack/v5`, `modernc.org/sqlite`. Thư mục `server/`.
- Client: TypeScript + Vite + PixiJS + `@msgpack/msgpack` + `howler`. Thư mục `client/`.
- Bản đồ: Tiled `.tmj` trong `maps/`.
- Không thêm thư viện/framework khác khi chưa hỏi.

## Lệnh
- Chạy server: `cd server && go run ./cmd/server` (mặc định cổng 8080)
- Chạy client dev: `cd client && npm run dev`
- Test luật chơi: `cd server && go test ./...`
- Giả lập cân bằng: `cd server && go run ./cmd/sim --bots 40 --minutes 10`
- Build client: `cd client && npm run build`

## Quy tắc bắt buộc
1. Server authoritative: mọi luật chơi chạy trên server (`server/internal/game`). Client chỉ gửi input và vẽ.
2. Mọi con số cân bằng lấy từ `config/game.json`. Không hardcode. Client lấy config qua `GET /api/config`.
3. Luật mới trong `internal/game` phải có unit test (đặc biệt: EXP, scale, bể kiếm, rớt cảnh giới, buff, vật cản mọc lại).
4. Không tự ý đổi luật trong GAME_SPEC. Nếu thấy spec mâu thuẫn hoặc thiếu → dừng lại hỏi.
5. Chữ hiển thị cho người chơi bằng tiếng Việt, gom trong `client/src/ui/strings.ts`.
6. Khi chưa có art: dùng hình placeholder (hình tròn + chữ) nhưng giữ đúng tên file/anim theo ART_PROMPTS để sau chỉ cần thả ảnh vào.
7. Trước khi báo "xong": chạy `go test ./...`, `go vet ./...`, `npm run build` đều phải pass.
8. Làm xong 1 bước nhỏ chạy được → git commit với message tiếng Việt rõ ràng.
9. Cuối phiên: cập nhật `docs/PROGRESS.md` (đã làm gì, còn gì, lỗi đã biết).
12. Sau mỗi lượt tối ưu hoặc thay đổi có thể ảnh hưởng hiệu năng: cập nhật `docs/PERF_LOG.md` (bảng §3, chi tiết §5, nút thắt §6) và `docs/perf_log.csv` theo đúng kịch bản chuẩn S1–S5. Đo trước rồi mới sửa; nút thắt phải có số đo làm bằng chứng.
