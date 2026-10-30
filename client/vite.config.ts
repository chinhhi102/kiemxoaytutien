import { defineConfig } from 'vite';

// The Go server (port 9999) owns the API, WebSocket and assets; Vite proxies to it in dev.
// host: true để điện thoại cùng Wi-Fi mở được http://<IP-PC>:5173.
// Override with KX_SERVER=http://localhost:8090 npm run dev when port 9999 is busy.
const target = process.env.KX_SERVER ?? 'http://localhost:9999';

// Telemetry chỉ có trong build test (`TELEMETRY=on npm run build`). Bản release build không
// có biến này nên `__TELEMETRY__` là `false` theo nghĩa đen, Rollup cắt nhánh `import()` và
// module telemetry không nằm trong gói — TELEMETRY_SPEC §1.6 yêu cầu đúng điều đó.
const telemetryOn = process.env.TELEMETRY === 'on';
// BUILD_ID để mọi số liệu truy được về đúng bản build đã deploy (§5).
const buildId = process.env.BUILD_ID ?? new Date().toISOString().slice(0, 16).replace(/[-:T]/g, '').replace(/(\d{8})(\d{4})/, '$1_$2');

export default defineConfig({
  define: {
    __TELEMETRY__: JSON.stringify(telemetryOn),
    BUILD_ID: JSON.stringify(buildId),
  },
  server: {
    host: true,
    proxy: {
      '/api': target,
      '/assets': target,
      '/ws': { target: target.replace(/^http/, 'ws'), ws: true },
    },
  },
  // '/assets/' is the game art served by the Go server, so built JS/CSS go to /static/.
  build: { chunkSizeWarningLimit: 1200, assetsDir: 'static' },
});
