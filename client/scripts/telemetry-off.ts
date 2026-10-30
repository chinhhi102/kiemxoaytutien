// Kiểm bản RELEASE không đóng gói module telemetry (TELEMETRY_SPEC §1.6, §6).
//
//   npm run check:telemetry-off
//
// Vì sao cần một lệnh máy chạy thay vì tin vào cờ build: cái bẫy ở đây im lặng. Chỉ cần ai
// đó đổi `import()` động thành `import` tĩnh ở đầu file, Rollup hết cắt được nhánh và module
// lọt vào bản release — build vẫn xanh, test vẫn xanh, không có gì báo. Người chơi thật thì
// bị thu số liệu mà không ai định thế.
import { execFileSync } from 'node:child_process';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const DIST = join(import.meta.dirname, '..', 'dist', 'static');

// Những dấu vết CHỈ có trong module telemetry. Không tìm chữ "telemetry" chung chung: tên
// hàm keo nối (`telemetryContext`) nằm trong main.ts và còn lại là chuyện bình thường —
// nó không chạy gì khi `__TELEMETRY__` là false.
// Chỉ dùng CHUỖI, không dùng tên hàm: minify đổi tên hàm (`collectSession` biến mất khỏi
// bản build dù module vẫn nằm trong đó), nên lấy tên hàm làm dấu vết là một phép kiểm luôn
// báo OK — tệ hơn không kiểm, vì nó cho cảm giác an toàn giả. Bốn chuỗi dưới đây đã kiểm
// chứng là CÓ trong bản `TELEMETRY=on` và KHÔNG có trong bản release.
const MARKERS = [
  'sendBeacon',
  'WEBGL_debug_renderer_info',
  '/telemetry/end',
  'longtask',
];

function bundleFiles(): string[] {
  return readdirSync(DIST).filter((f) => f.endsWith('.js')).map((f) => join(DIST, f));
}

function scan(): Map<string, string[]> {
  const hits = new Map<string, string[]>();
  for (const file of bundleFiles()) {
    const src = readFileSync(file, 'utf8');
    const found = MARKERS.filter((m) => src.includes(m));
    if (found.length > 0) {
      hits.set(file, found);
    }
  }
  return hits;
}

// Build sạch KHÔNG có cờ, rồi soi. Build lại ở đây chứ không soi dist sẵn có: dist trên máy
// có thể là bản `TELEMETRY=on` vừa build xong, và lúc đó lệnh kiểm sẽ báo động nhầm.
const env = { ...process.env };
delete env.TELEMETRY;
execFileSync('npx', ['vite', 'build'], { cwd: join(import.meta.dirname, '..'), env, stdio: 'ignore' });

const hits = scan();
if (hits.size > 0) {
  console.error('LỖI: bản release vẫn chứa module telemetry (TELEMETRY_SPEC §1.6).');
  for (const [file, found] of hits) {
    console.error(`  ${file}: ${found.join(', ')}`);
  }
  console.error('Thường là do `import` tĩnh thay cho `import()` động trong main.ts.');
  process.exit(1);
}
console.log(`OK: ${bundleFiles().length} tệp bản release, không có dấu vết module telemetry.`);
