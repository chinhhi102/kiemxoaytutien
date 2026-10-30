# Kết quả đo 10 tab Chrome headless

Chạy lại: `cd tools/browsertest && node run.mjs` — bài test **tự dựng server riêng ở
cổng 8099 với database tạm** rồi tắt khi xong, không đụng tới server đang có người chơi.
10 browser context riêng, mỗi context một người chơi; 60s đi lang thang rồi
5 tab lên Đạo Tổ đủ kiếm và cả 10 tab chụm một chỗ 60s.

> Bài này chạy **GPU thật** (`--use-gl=angle --use-angle=vulkan`), không dùng
> SwiftShader nữa. Nhưng **10 tab vẫn chia nhau MỘT card và một CPU**, nên cột FPS chỉ
> nói lên "10 người chơi trên một máy", không nói lên trải nghiệm của một người chơi thật.
>
> Server chạy riêng trên lõi 0-2, trình duyệt trên lõi 3-11.
> Con số **tick server chuẩn** lấy ở `cd server && go run ./cmd/loadtest`, nơi không có
> trình duyệt tranh CPU.
>
> Số đáng tin nhất ở bảng này: **ping, KB/s, kích thước snapshot, số entity trong AOI**.
>
> Lượt B bóp băng thông **sau khi đã vào game**: ta đo lưu lượng lúc chơi, không đo lúc
> tải trang.

**Lượt B — giả lập 4G yếu (trễ 100 ms, 1.6 Mbps xuống, 750 Kbps lên)** — 67 mẫu trong giai đoạn đánh nhau

| Chỉ số | 5 tab Đạo Tổ (TB / p95 / tệ nhất) | 5 tab thường (TB / p95 / tệ nhất) |
|---|---|---|
| FPS *(TB / p5 / thấp nhất — xem ghi chú)* | — | — |
| Cảnh giới (cấp) | — | 5.9 / 10.0 / 10.0 |
| Kiếm của chính tab | — | 1.0 / 4.0 / 4.0 |
| Thời gian vẽ | — | — |
| Ping | — | 14.0 / 22.0 / 26.0 ms |
| Băng thông xuống | — | 7.7 / 9.4 / 13.6 KB/s |
| Kích thước snapshot | — | 0.3 / 0.4 / 0.6 KB |
| Tick server | — | 0.3 / 0.5 / 0.5 ms |
| Số sprite | — | — |
| Draw call | — | — |
| Entity trong AOI | — | 311.7 / 375.0 / 382.0 |
| Tổng kiếm thấy trên màn hình | — | 2.4 / 14.0 / 14.0 |

Ảnh chụp lúc đông nhất: `docs/perf/browsertest-mang-binh-thuong.png` và
`docs/perf/browsertest-4g-yeu.png`.
