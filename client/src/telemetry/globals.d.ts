// Hằng số do Vite thay lúc build (xem vite.config.ts).
//
// `__TELEMETRY__` phải là HẰNG SỐ biên dịch chứ không phải biến đọc lúc chạy: chỉ khi nó là
// `false` theo nghĩa đen thì Rollup mới cắt được nhánh `import()` và KHÔNG đóng gói module
// telemetry vào bản release — đúng yêu cầu TELEMETRY_SPEC §1.6.
declare const __TELEMETRY__: boolean;
declare const BUILD_ID: string;
