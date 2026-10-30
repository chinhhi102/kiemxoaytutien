import { chromium } from 'playwright';
const URL = process.argv[2];
const SECS = Number(process.argv[3] ?? 40);
const b = await chromium.launch({ headless: true, args: ['--use-gl=angle','--use-angle=gl'] });
const p = await (await b.newContext({ viewport: { width: 1280, height: 720 } })).newPage();
const errs = [];
p.on('console', m => { if (m.type() === 'error') errs.push(m.text().slice(0, 140)); });
await p.goto(`${URL}/?debug=true`, { waitUntil: 'domcontentloaded' });
await p.waitForSelector('input', { timeout: 60000 });
await p.fill('input', 'TelTest');
await p.getByText('Vào chơi', { exact: true }).click();
await p.waitForFunction(() => window.kx?.playing?.() === true, null, { timeout: 60000 });
const t0 = Date.now();
while (Date.now() - t0 < SECS * 1000) {
  for (const k of ['KeyD', 'KeyA']) {
    await p.keyboard.down(k);
    await new Promise(r => setTimeout(r, 2000));
    await p.keyboard.up(k);
    if (Date.now() - t0 >= SECS * 1000) break;
  }
}
await p.close();
await new Promise(r => setTimeout(r, 2000));
console.log(JSON.stringify({ errs: errs.slice(0, 5) }));
await b.close();
