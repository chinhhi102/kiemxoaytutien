// Đo HAI triệu chứng NETCODE_SPEC §0/§1 nói tới, bằng số chứ không bằng mắt:
//   1. entity trong khung nhìn bị huỷ rồi dựng lại (nhấp nháy) — ngân sách §1: **0**
//   2. nhân vật vẽ bằng placeholder — ngân sách §1: **0**
//
//   node flicker.mjs [url]
//
// Kịch bản đúng như bài kiểm tra §7: **bot đi qua lại ĐÚNG MÉP khung nhìn 30 giây**.
// Đây là chỗ duy nhất lỗi xuất hiện, và cũng là chỗ tôi suýt bỏ sót: lượt đo đầu tiên tôi
// đứng một mình giữa khu, ra "0 nhấp nháy" và gần như kết luận là không có lỗi.
//
// Cách ép bot ra mép: thả bot cạnh mình (phím B), rồi **tự mình chạy ra xa** cho tới khi
// chúng nằm quanh ranh giới AOI — server bỏ qua entity xa hơn NỬA khung nhìn trên các tick
// lẻ (tối ưu 10 Hz), nên mép đó chính là nơi "vắng mặt một gói" xảy ra liên tục.
import { chromium } from 'playwright';
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const URL = process.argv[2] ?? 'http://127.0.0.1:9998';
const WATCH_MS = 30000;

const b = await chromium.launch({ headless: true, args: ['--use-gl=angle','--use-angle=gl','--enable-gpu','--ignore-gpu-blocklist'] });
const p = await (await b.newContext({ viewport: { width: 1280, height: 720 } })).newPage();
await p.goto(`${URL}/?debug=true`, { waitUntil: 'domcontentloaded' });
await p.waitForSelector('input', { timeout: 60000 });
await p.fill('input', 'FlickTest');
await p.getByText('Vào chơi', { exact: true }).click();
await p.waitForFunction(() => window.kx?.playing?.() === true, null, { timeout: 60000 });
await sleep(2000);

// Lên cảnh giới cao: khung nhìn rộng ra, nên "nửa khung nhìn" (ranh giới 10 Hz) nằm xa
// hẳn và bot thả ra dễ rơi đúng vào vùng mép.
for (let i = 0; i < 6; i++) { await p.keyboard.press('r'); await sleep(200); }
for (let i = 0; i < 10; i++) { await p.keyboard.press('b'); await sleep(200); }
await sleep(1500);

// Đi qua lại để bot liên tục cắt qua ranh giới AOI.
const walk = async () => {
  const t0 = Date.now();
  while (Date.now() - t0 < WATCH_MS) {
    for (const key of ['KeyD', 'KeyA']) {
      await p.keyboard.down(key);
      await sleep(2500);
      await p.keyboard.up(key);
      if (Date.now() - t0 >= WATCH_MS) { break; }
    }
  }
};

const watch = p.evaluate(async (ms) => {
  const goneAt = new Map();
  let gone = 0, respawn2s = 0, maxSeen = 0;
  let placeholderFrames = 0, frames = 0;
  const t0 = performance.now();
  let prev = new Set();
  while (performance.now() - t0 < ms) {
    const now = performance.now();
    const cur = new Set(window.kx.world.players.keys());
    maxSeen = Math.max(maxSeen, cur.size);
    for (const id of prev) {
      if (!cur.has(id)) { goneAt.set(id, now); gone++; }
    }
    for (const id of cur) {
      const at = goneAt.get(id);
      if (at !== undefined) {
        if (now - at < 2000) { respawn2s++; }
        goneAt.delete(id);
      }
    }
    prev = cur;
    // Khung có placeholder: PlayerSprite không có atlas thì thân là Container, không phải
    // AnimatedSprite. Đếm theo KHUNG, đúng chỉ số §6 của spec.
    frames++;
    for (const sp of window.kx.scene.players.values()) {
      if (!sp.atlas) { placeholderFrames++; break; }
    }
    await new Promise((r) => setTimeout(r, 50));
  }
  return { gone, respawn2s, maxSeen, placeholderFrames, frames, theo: { ...window.kx.world.respawnBy } };
}, WATCH_MS);

await walk();
const res = await watch;
console.log(JSON.stringify({ ...res, giay: WATCH_MS / 1000 }, null, 1));
await b.close();
// Thoát khác 0 khi vượt ngân sách, để chạy được trong CI mà không phải đọc bằng mắt.
process.exit(res.respawn2s === 0 && res.placeholderFrames === 0 ? 0 : 1);
