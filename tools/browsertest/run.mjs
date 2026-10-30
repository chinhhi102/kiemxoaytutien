// 10 tab Chrome headless tự chơi, đo hiệu năng client (GAME_SPEC §13.6).
//
//   cd tools/browsertest && npm install && npm run install-browser
//   node run.mjs
//   node run.mjs --tabs 10 --warmup 30 --fight 60
//
// Bài test **tự dựng một server riêng** ở cổng 8099 với database tạm, rồi tắt khi xong —
// không đụng tới server 9999 đang có người chơi thật. 10 bot giả sẽ không chen vào phòng
// của ai, và bảng xếp hạng thật cũng không bị bẩn.
// Muốn dùng server có sẵn thì: `node run.mjs --port 9999 --no-spawn` (nhớ server đó phải
// chạy với `-max-conns-per-ip 40`, vì 10 tab cùng đến từ 127.0.0.1).
// Đo server từ xa (staging): `node run.mjs --url https://kiemtien.staging.maveragroups.com --tabs 8`
// — không dựng server riêng, kết quả ghi sang BROWSERTEST-staging.md để không đè số đo trên máy.
// Staging giới hạn 8 kết nối mỗi IP, nên quá 8 tab là tab thứ 9 trở đi bị từ chối.
//
// Mỗi tab là một **browser context riêng** → localStorage riêng, coi như một người chơi
// khác máy. Tab tự đi lại ngẫu nhiên, sau `warmup` giây thì 5 tab lên Đạo Tổ đủ kiếm rồi
// cả 10 tab kéo về chụm một chỗ để ép va chạm kiếm và dồn vật phẩm vào tầm nhìn.
//
// ĐỌC KẾT QUẢ: headless Chrome vẽ WebGL bằng CPU (SwiftShader) và 10 tab tranh nhau CPU
// trên cùng một máy, nên **FPS ở đây thấp hơn thực tế rất nhiều — đừng dùng để kết luận về
// FPS**. Số đáng tin là: ping, KB/s, kích thước snapshot, tick server, số sprite, draw call.

import { chromium } from 'playwright';
import { execSync, spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import { cpus, tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO = resolve(HERE, '../..');

const args = Object.fromEntries(
  process.argv.slice(2).flatMap((a, i, all) => (a.startsWith('--') ? [[a.slice(2), all[i + 1] ?? '1']] : [])),
);
const PORT = Number(args.port ?? 8099);
const REMOTE = args.url !== undefined;
const ORIGIN = REMOTE ? args.url.replace(/\/+$/, '') : `http://127.0.0.1:${PORT}`;
const SPAWN = !REMOTE && args['no-spawn'] === undefined;
const OUT_TAG = REMOTE ? '-staging' : '';
// Kịch bản chuẩn PERF_LOG §1. `--scenario S2` đặt sẵn tham số để hai lần đo cách nhau một
// tuần chắc chắn giống nhau — quy tắc vàng của PERF_LOG là chỉ so các lần đo CÙNG kịch bản,
// mà gõ tay `--tabs 10 --warmup 60 --fight 60` thì sớm muộn cũng lệch một con số.
//
// S2 = mạng thường, S3 = cùng kịch bản nhưng 4G yếu; chọn bằng `--net normal|slow4g`.
const SCENARIOS = { S2: { tabs: 10, tien: 5, warmup: 60, fight: 60 } };
const SC = args.scenario ? SCENARIOS[args.scenario] : null;
if (args.scenario && !SC) {
  console.error(`kịch bản ${args.scenario} không có; browsertest chỉ chạy S2 (xem docs/PERF_LOG.md §1)`);
  process.exit(2);
}
const TABS = Number(args.tabs ?? SC?.tabs ?? 10);
const TIEN_TABS = Number(args.tien ?? SC?.tien ?? 5);
const WARMUP_S = Number(args.warmup ?? SC?.warmup ?? 30);
const FIGHT_S = Number(args.fight ?? SC?.fight ?? 60);
const BASE = `${ORIGIN}/?debug=true`;

// 4G yếu (spec yêu cầu): trễ 100 ms, 1.6 Mbps xuống, 750 Kbps lên.
const SLOW_4G = { offline: false, latency: 100, downloadThroughput: (1.6 * 1024 * 1024) / 8, uploadThroughput: (750 * 1024) / 8 };

const DIRS = { up: 'KeyW', down: 'KeyS', left: 'KeyA', right: 'KeyD' };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** goBin tìm lệnh `go`: trong PATH, hoặc bản nằm trong cache toolchain của máy này. */
function goBin() {
  const cached = join(
    process.env.HOME ?? '',
    'go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.11.linux-amd64/bin/go',
  );
  return existsSync(cached) ? cached : 'go';
}

/**
 * startServer dựng một server RIÊNG cho bài test: cổng khác, database tạm, nới trần kết nối
 * mỗi IP. Người đang chơi ở server thật không bị 10 bot giả chen vào phòng.
 */
/**
 * Chia lõi CPU: server giữ riêng vài lõi, trình duyệt lấy phần còn lại.
 *
 * Không chia thì con số "tick server" trong bảng vô nghĩa: 10 tab Chrome vẽ 5 000 sprite
 * mỗi tab ăn sạch 12 luồng của máy, goroutine của server bị hệ điều hành gạt ra liên tục và
 * một tick tốn 2,5 ms (đo bằng cmd/loadtest) kéo dài thành 269 ms. Đo được như vậy thật.
 */
const CORES = cpus().length;
const SERVER_CORES = `0-${Math.max(1, Math.floor(CORES / 4) - 1)}`;
const BROWSER_CORES = `${Math.max(2, Math.floor(CORES / 4))}-${CORES - 1}`;
const HAS_TASKSET = (() => {
  try {
    execSync('command -v taskset', { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
})();

async function startServer() {
  const db = join(tmpdir(), `kx-browsertest-${process.pid}.db`);
  const srvArgs = ['run', './cmd/server', '-addr', `127.0.0.1:${PORT}`, '-db', db, '-max-conns-per-ip', '40'];
  // --cpuprofile <file>: lấy hồ sơ CPU của SERVER THẬT trong lúc 10 tab đang chơi. Khác
  // cmd/loadtest ở chỗ có bot, bản đồ thật và client thật — hai nơi cho hình ảnh khác nhau.
  if (args.cpuprofile) {
    srvArgs.push('-cpuprofile', resolve(args.cpuprofile));
  }
  const proc = spawn(
    ...(HAS_TASKSET ? ['taskset', ['-c', SERVER_CORES, goBin(), ...srvArgs]] : [goBin(), srvArgs]),
    {
      cwd: resolve(REPO, 'server'),
      env: { ...process.env, GOMODCACHE: process.env.GOMODCACHE ?? join(process.env.HOME ?? '', '.cache/kx-gomod') },
      stdio: ['ignore', 'pipe', 'pipe'],
      // Nhóm tiến trình riêng: `go run` chạy binary trong tiến trình CON, giết mỗi `go run`
      // thì binary sống tiếp và giữ cổng — lần chạy sau chết vì "address already in use"
      // (đã mất ba lượt đo vì chuyện này). Giết cả nhóm bằng `process.kill(-pid)`.
      detached: true,
    },
  );
  let log = '';
  const keep = (b) => {
    log += b;
    // Chuyển tiếp ngay những dòng server từ chối lệnh test: không có nó thì bảng số ra sai
    // mà chẳng biết vì sao (đã mất hai lượt đo vì chuyện này).
    for (const line of String(b).split('\n')) {
      if (line.includes('dbg ') || line.includes('tick chậm')) console.log(`  [server] ${line.trim()}`);
    }
  };
  proc.stdout.on('data', keep);
  proc.stderr.on('data', keep);
  const deadline = Date.now() + 180000;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(`http://127.0.0.1:${PORT}/api/config`);
      if (r.ok) {
        console.log(`Server riêng cho bài test: cổng ${PORT}, database tạm ${db}`);
        return { proc, db };
      }
    } catch {
      /* chưa lên, thử lại */
    }
    await sleep(500);
  }
  await killTree(proc);
  throw new Error(`Không dựng được server test ở cổng ${PORT}:\n${log}`);
}

/** killTree giết cả nhóm tiến trình của server (`go run` + binary con nó đẻ ra). */
async function killTree(proc) {
  // SIGTERM trước để server kịp ghi hồ sơ CPU xuống đĩa; SIGKILL sau nếu nó lì.
  try {
    process.kill(-proc.pid, 'SIGTERM');
  } catch {
    proc.kill('SIGTERM');
  }
  await new Promise((r) => setTimeout(r, 1500));
  try {
    process.kill(-proc.pid, 'SIGKILL');
  } catch {
    /* đã chết rồi */
  }
}

/**
 * chơi một tab: vào game với tên cho trước rồi trả về các hàm điều khiển.
 *
 * Thử lại tối đa 3 lần với context mới: khi đo qua Internet, Chrome trên máy đo hay báo
 * ERR_NETWORK_CHANGED (Tailscale/bridge docker đổi route) và cắt ngang lúc tab đang tải
 * hàng chục MB atlas — tab treo mãi ở màn hình tải dù server đã nhận người vào.
 */
async function openTab(browser, name, slow, full) {
  for (let attempt = 1; ; attempt++) {
    try {
      return await openTabOnce(browser, name, slow, full);
    } catch (e) {
      if (attempt >= 3) throw e;
      console.log(`  ${name} hỏng lần ${attempt}, thử lại với context mới: ${String(e.message).slice(0, 120)}`);
    }
  }
}

async function openTabOnce(browser, name, slow, full) {
  // Từ khi dùng GPU thật thì cả 10 tab chạy được khổ 960×540 với chất lượng mặc định, tức
  // là 10 người chơi thật chứ không phải 9 cái bóng. Chỉ số VẼ vẫn chỉ đọc ở tab 1 vì 10
  // tab dù sao cũng chia nhau một GPU.
  //
  // (Vùng nhìn AOI của server tính từ config, không phụ thuộc kích thước cửa sổ — nên dù
  // có thu nhỏ cửa sổ thì cũng không sai một byte số liệu mạng.)
  const ctx = await browser.newContext({ viewport: { width: 960, height: 540 } });
  const page = await ctx.newPage();
  page.on('pageerror', (e) => console.log(`  [${name}] lỗi trang: ${String(e).slice(0, 160)}`));
  page.on('console', (m) => {
    if (m.type() === 'error') console.log(`  [${name}] console: ${m.text().slice(0, 160)}`);
  });
  try {
    await page.goto(BASE, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('input', { timeout: 120000 });
    await page.fill('input', name);
    await page.getByText('Vào chơi', { exact: true }).click();
    await page.waitForFunction(() => typeof window.__stats === 'function', null, { timeout: 120000 });
  } catch (e) {
    // Đóng context hỏng để server thả kết nối cũ, không thì lần thử lại ăn thêm một suất
    // trong giới hạn kết nối mỗi IP.
    await ctx.close().catch(() => {});
    throw new Error(`${name} không vào được game. Nếu console báo HTTP 429 thì server đang chặn số kết nối mỗi IP — chạy server với -max-conns-per-ip 40. (${e.message})`);
  }
  if (slow) {
    // Bóp băng thông SAU khi đã vào game: ta đo lưu lượng lúc CHƠI, không đo lúc tải trang.
    // Bóp từ đầu thì 10 tab phải tải ~500 KB JS + hàng chục MB ảnh qua 1.6 Mbps, mất hàng
    // phút và chẳng nói lên điều gì về netcode.
    const cdp = await ctx.newCDPSession(page);
    await cdp.send('Network.enable');
    await cdp.send('Network.emulateNetworkConditions', SLOW_4G);
  }
  return { ctx, page, name, held: new Set() };
}

/** hold giữ đúng tập phím mong muốn, nhả những phím không còn cần. */
async function hold(tab, want) {
  for (const k of [...tab.held]) {
    if (!want.has(k)) {
      await tab.page.keyboard.up(k).catch(() => {});
      tab.held.delete(k);
    }
  }
  for (const k of want) {
    if (!tab.held.has(k)) {
      await tab.page.keyboard.down(k).catch(() => {});
      tab.held.add(k);
    }
  }
}

/** wander: đổi hướng ngẫu nhiên, thỉnh thoảng giữ Shift (Thiêu Đốt Tinh Huyết). */
async function wander(tab) {
  const want = new Set();
  const h = ['left', 'right', null][Math.floor(Math.random() * 3)];
  const v = ['up', 'down', null][Math.floor(Math.random() * 3)];
  if (h) want.add(DIRS[h]);
  if (v) want.add(DIRS[v]);
  if (want.size === 0) want.add(DIRS.right);
  if (Math.random() < 0.3) want.add('ShiftLeft');
  await hold(tab, want);
}

/** steer: đi về phía (tx, ty) bằng chính phím di chuyển, như người thật. */
async function steer(tab, tx, ty) {
  const me = await tab.page.evaluate(() => {
    const m = window.kx?.world?.me?.();
    return m ? { x: m.x, y: m.y } : null;
  }).catch(() => null);
  if (!me) {
    return null;
  }
  const dx = tx - me.x;
  const dy = ty - me.y;
  const want = new Set();
  if (Math.abs(dx) > 60) want.add(dx > 0 ? DIRS.right : DIRS.left);
  if (Math.abs(dy) > 60) want.add(dy > 0 ? DIRS.down : DIRS.up);
  await hold(tab, want);
  return Math.hypot(dx, dy);
}

const sample = (tab) => tab.page.evaluate(() => window.__stats()).catch(() => null);

function stats(values) {
  if (!values.length) {
    return { avg: 0, p95: 0, worst: 0 };
  }
  const v = [...values].sort((a, b) => a - b);
  return {
    avg: v.reduce((s, x) => s + x, 0) / v.length,
    p95: v[Math.min(v.length - 1, Math.floor(v.length * 0.95))],
    worst: v[v.length - 1],
  };
}

/**
 * summarise: với hầu hết chỉ số thì "xấu" = lớn (p95 lấy đuôi trên, tệ nhất = max). Riêng
 * FPS thì ngược lại — xấu = nhỏ, nên lấy p5 và min.
 */
function summarise(rows, key, higherIsBetter = false) {
  const v = rows.map((r) => r[key] ?? 0).sort((a, b) => a - b);
  if (!v.length) {
    return { avg: 0, p95: 0, worst: 0 };
  }
  const avg = v.reduce((s, x) => s + x, 0) / v.length;
  if (higherIsBetter) {
    return { avg, p95: v[Math.floor(v.length * 0.05)], worst: v[0] };
  }
  return { avg, p95: v[Math.min(v.length - 1, Math.floor(v.length * 0.95))], worst: v[v.length - 1] };
}

async function runOnce(label, slow) {
  console.log(`\n=== Lượt ${label} ===`);
  if (HAS_TASKSET) {
    // Đổi lõi của CHÍNH tiến trình này; Chrome là con nên thừa hưởng, server đã chạy từ
    // trước trên lõi riêng nên không bị kéo theo.
    try {
      execSync(`taskset -cp ${BROWSER_CORES} ${process.pid}`, { stdio: 'ignore' });
    } catch {
      /* không có quyền thì thôi, chỉ mất độ chính xác của cột tick server */
    }
  }
  const browser = await chromium.launch({
    headless: true,
    // Chrome mặc định chỉ cho ~16 ngữ cảnh WebGL cùng lúc rồi giết bớt cái cũ; 10 tab +
    // ngữ cảnh phụ là chạm trần, nên phải nới ra.
    args: [
      // DÙNG GPU THẬT. Mặc định của Chrome headless là SwiftShader — tô từng điểm ảnh bằng
      // CPU — nên 10 tab ăn sạch CPU máy đo, Chrome bỏ cả việc xử lý gói WebSocket và cả
      // bảng số (ping, KB/s) hoá ra vô nghĩa. Đo được trên máy này:
      //   --use-gl=swiftshader        → SwiftShader (CPU)
      //   --use-gl=angle --use-angle=vulkan → AMD Radeon RX 570 qua RADV  ← dùng cái này
      //   --use-gl=angle --use-angle=gl     → AMD Radeon RX 570 qua radeonsi (dự phòng)
      // Máy không có GPU thì đổi `--use-angle=swiftshader`, số vẽ sẽ không đọc được nhưng
      // số mạng vẫn đúng.
      '--use-gl=angle',
      `--use-angle=${args.angle ?? 'vulkan'}`,
      '--enable-features=Vulkan',
      '--enable-gpu',
      '--ignore-gpu-blocklist',
      '--disable-dev-shm-usage',
      '--max-active-webgl-contexts=32',
      '--disable-frame-rate-limit',
    ],
  });
  const tabs = [];
  for (let i = 0; i < TABS; i++) {
    // Tab 1 là tab "mắt người": khổ thật, chất lượng thật, chỉ nó mới cho số FPS / draw
    // call đáng đọc. Nó cũng nằm trong nhóm Đạo Tổ nên thấy đúng cảnh nặng nhất.
    const full = i === 0;
    const t = await openTab(browser, `Test${String(i + 1).padStart(2, '0')}`, slow, full);
    t.full = full;
    tabs.push(t);
    console.log(`  ${t.name} đã vào`);
  }

  const samples = new Map(tabs.map((t) => [t.name, []]));
  let phase = 'wander';
  const collector = setInterval(async () => {
    for (const t of tabs) {
      const s = await sample(t);
      if (s) {
        samples.get(t.name).push({ ...s, phase });
      }
    }
  }, 1000);

  // Giai đoạn 1: đi lang thang
  console.log(`  đi lang thang ${WARMUP_S}s...`);
  const wanderUntil = Date.now() + WARMUP_S * 1000;
  while (Date.now() < wanderUntil) {
    await Promise.all(tabs.map(wander));
    await sleep(1000 + Math.random() * 1000);
  }

  // Giai đoạn 2: 5 tab lên Đạo Tổ đủ kiếm, rồi tất cả chụm về giữa bản đồ
  const tien = tabs.slice(0, TIEN_TABS);
  // Cấp Đạo Tổ lấy từ config chứ không viết cứng: đổi số cảnh giới trong config/game.json
  // là bài test tự theo, không âm thầm kiểm sai ngưỡng.
  const cult = (await (await fetch(`${ORIGIN}/api/config`)).json()).cultivation;
  const TOP_LEVEL = (cult.realms.length + cult.tien_realms.length - 1) * cult.tiers_per_realm;
  // n: 1 = giữ bất tử. Không có nó thì 5 người Đạo Tổ chụm vào mài nhau chết sạch trong
  // chục giây, cả phòng tụt về Luyện Khí 1 kiếm và số đo thành vô nghĩa (đã dính một lần).
  // Trả về lý do hỏng thay vì nuốt lỗi: lần trước chỉ 1/5 tab lên được mà không biết vì
  // sao, do `.catch(() => {})` giấu hết.
  const boost = () => Promise.all(tien.map((t) =>
    t.page.evaluate(() => {
      if (!window.kx) return 'chưa có window.kx (tab không ở chế độ debug)';
      if (!window.kx.conn?.isOpen) return 'websocket chưa mở';
      window.kx.conn.send({ t: 'dbg', cmd: 'max', n: 1 });
      return '';
    }).catch((e) => `evaluate lỗi: ${e.message.slice(0, 80)}`)));
  // Gửi LẠI mỗi vòng chứ không gửi một lần: lúc 10 tab đang tranh CPU, một lệnh đi lạc là
  // cả bảng số sai kịch bản (đã dính: cột cảnh giới ra 0.0 trong khi lẽ ra phải là 130).
  // Trước khi boost: tab nào chết trong lúc lang thang thì đang đứng ở bảng Tổng kết, đã
  // RỜI PHÒNG — lệnh test gửi vào đó rơi vào hư vô (server báo "chưa ở trong phòng nào").
  // Bấm "Chơi tiếp" cho nó vào lại rồi mới đo.
  const revive = (list = tien) => Promise.all(list.map(async (t) => {
    // Hỏi `playing()` chứ không hỏi `world.me()`: sau khi chết và rời phòng, `me()` vẫn
    // trả về bản ghi cũ nên tab đứng ở bảng Tổng kết mà ta tưởng nó đang chơi.
    const alive = await t.page.evaluate(() => window.kx?.playing?.() === true).catch(() => false);
    if (alive) {
      return;
    }
    await t.page.getByText('Chơi tiếp', { exact: true }).click({ timeout: 2000 }).catch(() => {});
    await t.page.waitForFunction(() => window.kx?.playing?.() === true, null, { timeout: 15000 }).catch(() => {});
  }));

  for (let i = 0; i < 20; i++) {
    await revive();
    const why = (await boost()).filter(Boolean);
    if (why.length && i % 5 === 0) {
      console.log(`  boost chưa tới ${why.length} tab: ${[...new Set(why)].join(' · ')}`);
    }
    await sleep(500);
    const levels = await Promise.all(tien.map((t) =>
      t.page.evaluate(() => window.kx?.world?.me?.()?.L ?? -1).catch(() => -1)));
    if (levels.every((L) => L >= TOP_LEVEL)) {
      console.log(`  xác nhận ${tien.length} tab đã lên Đạo Tổ`);
      break;
    }
    if (i === 19) {
      console.log(`  CẢNH BÁO: chỉ ${levels.filter((L) => L >= TOP_LEVEL).length}/${tien.length} tab lên được Đạo Tổ — bảng số sẽ không đúng kịch bản (cấp đọc được: ${levels.join(', ')})`);
    }
  }
  console.log(`  ${tien.length} tab lên Đạo Tổ, chụm lại đánh nhau ${FIGHT_S}s...`);
  phase = 'fight';
  // Chỗ chụm quân: hỏi tab ĐẦU TIÊN còn đọc được, không cố định tab 0.
  //
  // Một tab có thể mất ngữ cảnh WebGL giữa chừng (10 tab tranh GPU, nhất là lúc bóp 4G);
  // `glrecover` sẽ tải lại trang, và trong lúc đó `window.kx` chưa có. Bản cũ đọc thẳng
  // `tabs[0]` nên chỉ một tab hỏng là **cả bài đo chết**, mất sạch số liệu của chín tab
  // còn lại — đã xảy ra đúng vậy ở lượt S3 đầu tiên.
  let target = null;
  for (const t of tabs) {
    target = await t.page
      .evaluate(() => {
        const m = window.kx?.world?.me?.();
        return m ? { x: m.x, y: m.y } : null;
      })
      .catch(() => null);
    if (target) {
      break;
    }
  }
  if (!target) {
    console.log('  BỎ LƯỢT: không tab nào đọc được vị trí (mất ngữ cảnh WebGL?)');
    clearInterval(collector);
    await browser.close();
    return null;
  }
  const fightUntil = Date.now() + FIGHT_S * 1000;
  let shot = false;
  let beat = 0;
  while (Date.now() < fightUntil) {
    // Năm tab thường vẫn chết thật trong lúc đánh nhau; để chúng nằm ở bảng Tổng kết thì
    // mẫu đo đông cứng ở giá trị cuối và cả cột băng thông ra 0.0 KB/s.
    await revive(tabs);
    await Promise.all(tabs.map((t) => steer(t, target.x, target.y)));
    // Nạp lại kiếm mỗi 2s: dù bất tử, va kiếm vẫn mài rụng ~120 kiếm/giây khi 5 người chụm
    // vào nhau, nên không nạp lại thì chỉ sau mươi giây là hết cảnh nặng cần đo.
    if (++beat % 4 === 0) {
      await boost();
    }
    await sleep(500);
    if (!shot && Date.now() > fightUntil - (FIGHT_S * 1000) / 3) { // ~2/3 pha, lúc đông nhất
      shot = true;
      // Ảnh chụp là thứ phụ, KHÔNG được làm hỏng phép đo. Mười tab đang tranh nhau GPU nên
      // `screenshot` có lúc quá 30 giây rồi ném lỗi — và lỗi đó từng làm cả bài đo chết ở
      // bước cuối, mất sạch số liệu vừa thu được trong hai phút.
      await mkdir(resolve(REPO, 'docs/perf'), { recursive: true });
      await tien[0].page
        .screenshot({ path: resolve(REPO, `docs/perf/browsertest${OUT_TAG}-${label}.png`), timeout: 15000 })
        .catch((e) => console.log(`  (bỏ qua ảnh chụp: ${e.message.split('\n')[0]})`));
    }
  }

  clearInterval(collector);
  await browser.close();

  const rows = [];
  for (const [name, list] of samples) {
    for (const s of list) {
      rows.push({ ...s, name, tien: tien.some((t) => t.name === name), full: tabs.find((t) => t.name === name)?.full === true });
    }
  }
  return { label, rows };
}

function table(rows, title) {
  const fight = rows.filter((r) => r.phase === 'fight');
  const group = (want) => fight.filter((r) => r.tien === want);
  const out = [`**${title}** — ${fight.length} mẫu trong giai đoạn đánh nhau\n`];
  out.push(`| Chỉ số | ${TIEN_TABS} tab Đạo Tổ (TB / p95 / tệ nhất) | ${TABS - TIEN_TABS} tab thường (TB / p95 / tệ nhất) |`);
  out.push('|---|---|---|');
  // Chỉ tab 1 chạy khổ thật; chín tab kia thu nhỏ để không ăn hết CPU máy đo, nên số vẽ
  // của chúng vô nghĩa. Các chỉ số VẼ lấy riêng từ tab 1, các chỉ số MẠNG lấy cả nhóm.
  const eye = fight.filter((r) => r.full);
  const lineEye = (name, key, unit, higherIsBetter = false) => {
    if (!eye.length) {
      out.push(`| ${name} | — | — |`);
      return;
    }
    const s = summarise(eye, key, higherIsBetter);
    out.push(`| ${name} | ${s.avg.toFixed(1)} / ${s.p95.toFixed(1)} / ${s.worst.toFixed(1)}${unit ? ' ' + unit : ''} | *(chỉ đo ở tab 1)* |`);
  };
  const line = (name, key, unit, higherIsBetter = false) => {
    const cell = (g) => {
      if (!g.length) return '—';
      const s = summarise(g, key, higherIsBetter);
      return `${s.avg.toFixed(1)} / ${s.p95.toFixed(1)} / ${s.worst.toFixed(1)}${unit ? ' ' + unit : ''}`;
    };
    out.push(`| ${name} | ${cell(group(true))} | ${cell(group(false))} |`);
  };
  lineEye('FPS *(TB / p5 / thấp nhất — xem ghi chú)*', 'fps', '', true);
  line('Cảnh giới (cấp)', 'myLevel', '');
  line('Kiếm của chính tab', 'mySwords', '');
  lineEye('Thời gian vẽ', 'drawMs', 'ms');
  line('Ping', 'ping', 'ms');
  line('Băng thông xuống', 'kbPerSec', 'KB/s');
  line('Kích thước snapshot', 'snapKB', 'KB');
  line('Tick server', 'srvMs', 'ms');
  lineEye('Số sprite', 'sprites', '');
  lineEye('Draw call', 'drawCalls', '');
  line('Entity trong AOI', 'aoi', '');
  line('Tổng kiếm thấy trên màn hình', 'swords', '');
  return out.join('\n');
}

/**
 * writeReport ghi bảng kết quả sau MỖI lượt: lượt 4G hỏng giữa chừng (vd. người thật chen vào
 * chiếm suất kết nối cùng IP) thì số của lượt mạng thường vẫn còn trên đĩa.
 */
async function writeReport(runs) {
  const doc = [
    `# Kết quả đo ${TABS} tab Chrome headless${REMOTE ? ` — server ${ORIGIN}` : ''}`,
    '',
    ...(REMOTE
      ? [`Chạy lại: \`cd tools/browsertest && node run.mjs --url ${ORIGIN} --tabs ${TABS}\` — tab chạy`,
         'trên máy đo, nối qua Internet tới server thật; ping và KB/s là số qua mạng thật.']
      : [`Chạy lại: \`cd tools/browsertest && node run.mjs\` — bài test **tự dựng server riêng ở`,
         `cổng ${PORT} với database tạm** rồi tắt khi xong, không đụng tới server đang có người chơi.`]),
    `${TABS} browser context riêng, mỗi context một người chơi; ${WARMUP_S}s đi lang thang rồi`,
    `${TIEN_TABS} tab lên Đạo Tổ đủ kiếm và cả ${TABS} tab chụm một chỗ ${FIGHT_S}s.`,
    '',
    `> Bài này chạy **GPU thật** (\`--use-gl=angle --use-angle=${args.angle ?? 'vulkan'}\`), không dùng`,
    `> SwiftShader nữa. Nhưng **${TABS} tab vẫn chia nhau MỘT card và một CPU**, nên cột FPS chỉ`,
    `> nói lên "${TABS} người chơi trên một máy", không nói lên trải nghiệm của một người chơi thật.`,
    '>',
    ...(REMOTE
      ? ['> Server chạy trên máy riêng, không tranh CPU với trình duyệt: cột **tick server** là số thật',
         '> của máy đó (cộng thêm bot lấp phòng).']
      : [`> Server chạy riêng trên lõi ${SERVER_CORES}, trình duyệt trên lõi ${BROWSER_CORES}` +
           `${HAS_TASKSET ? '' : ' *(không có taskset — cột tick server sẽ bị thổi phồng)*'}.`,
         '> Con số **tick server chuẩn** lấy ở `cd server && go run ./cmd/loadtest`, nơi không có',
         '> trình duyệt tranh CPU.']),
    '>',
    '> Số đáng tin nhất ở bảng này: **ping, KB/s, kích thước snapshot, số entity trong AOI**.',
    '>',
    '> Lượt B bóp băng thông **sau khi đã vào game**: ta đo lưu lượng lúc chơi, không đo lúc',
    '> tải trang.',
    '',
    ...runs.flatMap((r) => [
      table(r.rows, r.label === '4g-yeu'
        ? 'Lượt B — giả lập 4G yếu (trễ 100 ms, 1.6 Mbps xuống, 750 Kbps lên)'
        : `Lượt A — mạng bình thường (${REMOTE ? 'Internet tới server thật' : 'LAN nội bộ'})`),
      '',
    ]),
    `Ảnh chụp lúc đông nhất: \`docs/perf/browsertest${OUT_TAG}-mang-binh-thuong.png\` và`,
    `\`docs/perf/browsertest${OUT_TAG}-4g-yeu.png\`.`,
    '',
  ].join('\n');

  await mkdir(resolve(REPO, 'docs/perf'), { recursive: true });
  await writeFile(resolve(REPO, `docs/perf/BROWSERTEST${OUT_TAG}.md`), doc, 'utf8');
  console.log(`\n${doc}`);
  console.log(`\n→ docs/perf/BROWSERTEST${OUT_TAG}.md`);
}

const server = SPAWN ? await startServer() : null;
const runs = [];
try {
  // --only mang-binh-thuong | 4g-yeu: chạy một lượt thôi, để dò lỗi cho nhanh.
  // `--net normal|slow4g` là tên theo PERF_LOG §1 (S2 / S3), ánh xạ lên đúng cơ chế này:
  // `--only X` nghĩa là CHỈ chạy lượt X.
  const only = args.net === 'normal' ? 'mang-binh-thuong'
    : args.net === 'slow4g' ? '4g-yeu'
    : args.only;
  if (only !== '4g-yeu') {
    const rA = await runOnce("mang-binh-thuong", false);
    if (rA) { runs.push(rA); }
    if (runs.length) { await writeReport(runs); }
  }
  if (only !== 'mang-binh-thuong') {
    const rB = await runOnce("4g-yeu", true);
    if (rB) { runs.push(rB); }
    await writeReport(runs);
  }
} finally {
  if (server) {
    await killTree(server.proc);
    await rm(server.db, { force: true }).catch(() => {});
    console.log('Đã tắt server test và xoá database tạm.');
  }
}

