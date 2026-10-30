// Đo HAI triệu chứng spec v6 §0 nói tới, bằng số chứ không bằng mắt:
//   1. entity trong khung nhìn bị huỷ rồi dựng lại (nhấp nháy)
//   2. khung hình có nhân vật vẽ bằng placeholder
//
// Cách đo: bọc `Map.prototype.delete` của bảng người chơi thì không chắc; thay vào đó đếm
// thẳng trên `window.kx` — số người client đang giữ, và số lần một id biến mất rồi quay lại.
import { chromium } from 'playwright';
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const URL = process.argv[2] ?? 'http://127.0.0.1:9998';

const b = await chromium.launch({ headless: true, args: ['--use-gl=angle','--use-angle=gl','--enable-gpu','--ignore-gpu-blocklist'] });
const p = await (await b.newContext({ viewport: { width: 1280, height: 720 } })).newPage();
await p.goto(`${URL}/?debug=true`, { waitUntil: 'domcontentloaded' });
await p.waitForSelector('input', { timeout: 60000 });
await p.fill('input', 'FlickTest');
await p.getByText('Vào chơi', { exact: true }).click();
await p.waitForFunction(() => window.kx?.playing?.() === true, null, { timeout: 60000 });
await sleep(2000);
// Thả bot ngay cạnh mình: không có entity nào trong tầm nhìn thì phép đo vô nghĩa (lượt
// đầu chạy ra "1 người, 0 lần nhấp nháy" chỉ vì tôi đứng một mình giữa khu).
for (let i = 0; i < 8; i++) {
  await p.keyboard.press('b');
  await sleep(200);
}
await sleep(2000);

// Theo dõi 15 giây: mỗi 50 ms ghi lại tập id đang có, đếm số lần một id rời đi rồi quay lại.
const res = await p.evaluate(async () => {
  const seenGone = new Map();   // id -> thời điểm biến mất
  let respawn2s = 0, gone = 0, maxPlayers = 0;
  const t0 = performance.now();
  let prev = new Set();
  while (performance.now() - t0 < 15000) {
    const now = performance.now();
    const cur = new Set(window.kx.world.players.keys());
    maxPlayers = Math.max(maxPlayers, cur.size);
    for (const id of prev) {
      if (!cur.has(id)) { seenGone.set(id, now); gone++; }
    }
    for (const id of cur) {
      const at = seenGone.get(id);
      if (at !== undefined) {
        if (now - at < 2000) { respawn2s++; }
        seenGone.delete(id);
      }
    }
    prev = cur;
    await new Promise((r) => setTimeout(r, 50));
  }
  // Nhân vật đang vẽ bằng placeholder: PlayerSprite không có atlas thì body là Container,
  // có atlas thì là AnimatedSprite.
  let placeholder = 0, total = 0; const info = [];
  for (const sp of window.kx.scene.players.values()) {
    total++;
    if (!sp.atlas) { placeholder++; info.push({ id: sp.id, char: JSON.stringify(sp.char), viewChar: JSON.stringify(window.kx.world.players.get(sp.id)?.char) }); }
  }
  return { gone, respawn2s, maxPlayers, placeholder, total, info, giay: 15 };
});
console.log(JSON.stringify(res, null, 1));
await b.close(); process.exit(0);
