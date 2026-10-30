// Kịch bản đo netcode CÓ KIỂM SOÁT — nút thắt B3b trong PERF_LOG.
//
//   node tools/browsertest/netcode-bench.mjs [số lượt]
//
// ── Vì sao cần ──────────────────────────────────────────────────────────────────────────
// `flicker.mjs` đo trên server thật, nơi 60 bot tự do đi lại. Số entity đi ngang khung nhìn
// mỗi lượt dao động 1–22, và chỉ số respawn theo đó nhảy 0–9 TRÊN CÙNG MỘT BẢN BUILD. Ngày
// 30/10 tôi đã lấy "2/13 lượt" so với "8/15 lượt" rồi kết luận một thay đổi làm tốt lên —
// kết luận đó sai, vì hai con số ấy nằm trong cùng một khoảng dao động.
//
// Bài này bỏ hết yếu tố ngẫu nhiên:
//   - Server RIÊNG, cổng riêng, DB tạm, **đúng MỘT bot** thay vì 60 (`bot_fill_target = 1`).
//   - Một khu duy nhất: bỏ luôn yếu tố chuyển khu.
//   - Một tab quan sát, đi theo đường CỐ ĐỊNH và nhịp CỐ ĐỊNH để liên tục kéo con bot kia
//     qua lại ranh giới AOI.
//
// Bản đầu tôi dùng hai tab người chơi và tắt sạch bot. Nó chạy ra toàn số 0 đẹp đẽ — nhưng
// `thấy=1` lộ ra rằng hai tab KHÔNG BAO GIỜ nhìn thấy nhau (chúng sinh ra ở hai chỗ ngẫu
// nhiên cách nhau cả nghìn đơn vị). Bài đo không chạm tới thứ nó định đo, mà vẫn báo "đạt".
//
// Nhờ vậy hai lần chạy khác nhau chỉ còn khác vì thứ ta vừa sửa, không phải vì hôm nay bot
// đi đâu. Muốn so hai cấu hình thì chạy bài này cho từng cấu hình, cùng số lượt.
import { chromium } from 'playwright';
import { spawn } from 'node:child_process';
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const PORT = 9996; // KHÔNG phải 9999: bài đo không bao giờ được đụng tới server người chơi
const ROUNDS = Number(process.argv[2] ?? 5);
/** Mỗi lượt đo bao lâu. */
const WATCH_MS = 20000;
/** Người đi: mỗi chiều bao lâu rồi quay đầu. Cố định để hai lượt giống hệt nhau. */
const LEG_MS = 4500;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// ── 1. Config riêng: tắt sạch bot ────────────────────────────────────────────────────────
const dir = mkdtempSync(join(tmpdir(), 'kx-bench-'));
const cfg = JSON.parse(readFileSync(join(REPO, 'config', 'game.json'), 'utf8'));
// ĐÚNG MỘT bot. Zero thì không có gì để đo; 60 thì phương sai nuốt mất tín hiệu.
cfg.room.bot_fill_target = 1;
cfg.room.zone_min_chars = 0;
cfg.room.zones = 1; // một khu: bỏ luôn yếu tố chuyển khu
const cfgPath = join(dir, 'game.json');
writeFileSync(cfgPath, JSON.stringify(cfg));

const bin = join(REPO, 'server', 'kxserver');
const srv = spawn(bin, [
  '-addr', `127.0.0.1:${PORT}`,
  '-config', cfgPath,
  '-db', join(dir, 'lb.db'),
  '-telemetry=false',
  '-client', join(REPO, 'server', 'data', 'deploy', 'client'),
  '-assets', join(REPO, 'server', 'data', 'deploy', 'assets'),
], { stdio: ['ignore', 'pipe', 'pipe'] });
srv.stdout.on('data', () => {});
srv.stderr.on('data', () => {});
process.on('exit', () => srv.kill());

await sleep(3000);

// ── 2. Một lượt đo ───────────────────────────────────────────────────────────────────────
async function round(browser) {
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  const watcher = await ctx.newPage();
  const walker = await ctx.newPage();

  const join = async (page, name) => {
    await page.goto(`http://127.0.0.1:${PORT}/?debug=true`, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input', { timeout: 30000 });
    await page.fill('input', name);
    await page.getByText('Vào chơi', { exact: true }).click();
    await page.waitForFunction(() => window.kx?.playing?.() === true, null, { timeout: 30000 });
  };
  await join(watcher, 'QuanSat');
  await join(walker, 'NguoiDi');
  await sleep(1500);

  // KHÔNG lên cảnh giới, KHÔNG thêm kiếm: khung nhìn phải NHỎ để một quãng đi cố định đủ
  // cắt qua ranh giới AOI cả hai chiều, và người quan sát không được giết người đi.
  //
  // Bản trước thả 3 bot cạnh mình. Nó cho `rời = 3` đẹp đẽ nhưng `quay lại = 0`: bot đi mất
  // hoặc chết, không bao giờ trở lại. Mà `respawn = 0` khi KHÔNG AI quay lại là con số hiển
  // nhiên, không phải kết quả đo.

  // Người đi TỰ TÌM TỚI người quan sát. Hai tab sinh ra ở hai chỗ ngẫu nhiên cách nhau cả
  // nghìn đơn vị; không có bước này thì họ không bao giờ thấy nhau.
  const posOf = (page) => page.evaluate(() => {
    const me = window.kx.world.me();
    return me ? { x: me.x, y: me.y } : null;
  });
  const target = await posOf(watcher);
  for (let i = 0; i < 120; i++) {
    const me = await posOf(walker);
    if (!me || !target) { break; }
    const dx = target.x - me.x;
    const dy = target.y - me.y;
    if (Math.hypot(dx, dy) < 220) { break; }
    const keys = [];
    if (dx > 60) { keys.push('KeyD'); } else if (dx < -60) { keys.push('KeyA'); }
    if (dy > 60) { keys.push('KeyS'); } else if (dy < -60) { keys.push('KeyW'); }
    for (const k of keys) { await walker.keyboard.down(k); }
    await sleep(250);
    for (const k of keys) { await walker.keyboard.up(k); }
  }
  // Thả thêm vài bot NGAY CẠNH mình: bot do phòng tự sinh nằm ở chỗ ngẫu nhiên, có thể cả
  // lượt không đi ngang qua. Thả cạnh mình thì chắc chắn chúng ở trong khung nhìn lúc bắt
  // đầu, và đường đi cố định bên dưới sẽ kéo chúng qua ranh giới.
  for (let i = 0; i < 3; i++) { await watcher.keyboard.press('b'); await sleep(200); }
  await sleep(1200);

  // Bắt đầu đếm từ tab quan sát.
  const watch = watcher.evaluate(async (ms) => {
    const w = window.kx.world;
    const base = { ds: w.respawnBy.ds, hethan: w.respawnBy.hethan };
    const goneAt = new Map();
    let gone = 0, respawn2s = 0, placeholder = 0, frames = 0, maxSeen = 0;
    // `quayLai` đếm MỌI lần entity trở lại, không chỉ trong 2 giây.
    //
    // Thiếu con số này thì bài đo tự lừa mình: nếu entity đi rồi không bao giờ quay lại,
    // `respawn2s = 0` là hiển nhiên và chẳng chứng minh được gì. Có quay lại thì số 0 kia
    // mới có nghĩa là "quay lại mà KHÔNG nhấp nháy".
    let quayLai = 0;
    let prev = new Set();
    const t0 = performance.now();
    while (performance.now() - t0 < ms) {
      const now = performance.now();
      const cur = new Set(w.players.keys());
      maxSeen = Math.max(maxSeen, cur.size);
      for (const id of prev) { if (!cur.has(id)) { goneAt.set(id, now); gone++; } }
      for (const id of cur) {
        const at = goneAt.get(id);
        if (at !== undefined) { quayLai++; if (now - at < 2000) { respawn2s++; } goneAt.delete(id); }
      }
      prev = cur;
      frames++;
      for (const sp of window.kx.scene.players.values()) {
        if (!sp.hasAtlas()) { placeholder++; break; }
      }
      await new Promise((r) => setTimeout(r, 50));
    }
    return {
      gone, quayLai, respawn2s, placeholder, frames, maxSeen,
      doDespawn: w.respawnBy.ds - base.ds,
      doHetHan: w.respawnBy.hethan - base.hethan,
    };
  }, WATCH_MS);

  // Người đi: ra xa rồi quay lại, nhịp CỐ ĐỊNH, cắt qua ranh giới AOI cả hai chiều.
  const t0 = Date.now();
  while (Date.now() - t0 < WATCH_MS) {
    for (const key of ['KeyD', 'KeyA']) {
      await walker.keyboard.down(key);
      await sleep(LEG_MS);
      await walker.keyboard.up(key);
      await sleep(300);
      if (Date.now() - t0 >= WATCH_MS) { break; }
    }
  }
  const out = await watch;
  await ctx.close();
  return out;
}

// ── 3. Chạy nhiều lượt rồi tổng hợp ─────────────────────────────────────────────────────
const browser = await chromium.launch({ headless: true, args: ['--use-gl=angle', '--use-angle=gl'] });
const rows = [];
for (let i = 0; i < ROUNDS; i++) {
  rows.push(await round(browser));
  const r = rows[rows.length - 1];
  console.log(`lượt ${i + 1}: respawn=${r.respawn2s} · quay lại=${r.quayLai} · rời=${r.gone} · placeholder=${r.placeholder} · thấy=${r.maxSeen}`);
}
await browser.close();
srv.kill();

const sum = (f) => rows.reduce((t, r) => t + f(r), 0);
const vals = rows.map((r) => r.respawn2s);
console.log('\n── tổng hợp ──');
console.log(`respawn: tổng ${sum((r) => r.respawn2s)} · mỗi lượt ${vals.join(', ')}`);
console.log(`placeholder: tổng ${sum((r) => r.placeholder)} khung / ${sum((r) => r.frames)}`);
console.log(`rời khung ${sum((r) => r.gone)} lần · quay lại ${sum((r) => r.quayLai)} lần`);
console.log(`nhấp nháy do Despawn ${sum((r) => r.doDespawn)} · do luật hết hạn 2s ${sum((r) => r.doHetHan)}`);
if (sum((r) => r.quayLai) === 0) {
  // Không có lần quay lại nào thì `respawn = 0` là hiển nhiên, không phải kết quả.
  console.error('\nCẢNH BÁO: không entity nào QUAY LẠI khung nhìn — bài đo chưa chạm tới thứ nó định đo.');
  process.exit(2);
}
// Thoát khác 0 khi vượt ngân sách §1 — chạy được trong CI.
process.exit(sum((r) => r.respawn2s) === 0 && sum((r) => r.placeholder) === 0 ? 0 : 1);
