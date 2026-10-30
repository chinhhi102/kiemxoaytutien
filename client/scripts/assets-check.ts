// npm run check:assets — chặn build nếu còn thiếu asset BẮT BUỘC.
//
// NETCODE_SPEC §5 Hiển thị mục 1: "build thật cấm placeholder … CI chặn build nếu
// `assets:report` còn thiếu asset bắt buộc".
//
// Khác `assets:report` ở chỗ: report LIỆT KÊ cho người đọc, còn cái này TRẢ MÃ LỖI cho máy.
// Một danh sách thiếu asset in ra rồi build vẫn xanh thì chẳng chặn được gì — ngày 30/10 đội
// test đã chơi một bản có 12 khung placeholder mà không ai biết.
import { existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { catalog, type CatalogEntry } from '../src/render/catalog.ts';

const ASSETS = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'assets');
const AUDIO_EXTS = ['.ogg', '.mp3', '.wav'];

function exists(rel: string): boolean {
  if (rel.startsWith('audio/')) {
    const base = rel.replace(/\.[a-z0-9]+$/, '');
    return AUDIO_EXTS.some((e) => existsSync(join(ASSETS, base + e)));
  }
  return existsSync(join(ASSETS, rel));
}

/** requiredOf: danh sách file BẮT BUỘC của một mục; `required` không khai báo thì là tất cả. */
function requiredOf(e: CatalogEntry): string[] {
  return e.required ?? e.files;
}

/**
 * BLOCKING: nhóm mà THIẾU art là người chơi NHÌN THẤY PLACEHOLDER.
 *
 * Chỉ chặn build vì những nhóm này, không chặn vì mọi thứ. Thiếu `ambient/firefly.png` thì
 * lớp sinh động bỏ qua con đom đóm và không ai biết; thiếu atlas nhân vật thì người chơi
 * thấy một vòng tròn màu có chữ viết tắt — hai chuyện khác hẳn nhau.
 *
 * Chặn bừa cũng nguy hiểm theo cách riêng: một lệnh kiểm lúc nào cũng đỏ sẽ bị bỏ qua, rồi
 * tới lúc nó đỏ vì lý do thật thì không ai nhìn nữa.
 */
const BLOCKING = new Set(['Nhân vật', 'Quái', 'Vật phẩm', 'Vật cản', 'Kiếm']);

const blocking: { group: string; label: string; files: string[] }[] = [];
const warn: { group: string; label: string; files: string[] }[] = [];
let checked = 0;
for (const e of catalog()) {
  const need = requiredOf(e);
  checked += need.length;
  const gone = need.filter((f) => !exists(f));
  if (gone.length === 0) {
    continue;
  }
  (BLOCKING.has(e.group) ? blocking : warn).push({ group: e.group, label: e.label, files: gone });
}

console.log(`Đã kiểm ${checked} tệp bắt buộc trong ${catalog().length} mục.`);
if (warn.length > 0) {
  console.log(`\nThiếu ${warn.length} mục art KHÔNG chặn build (game bỏ qua, không vẽ placeholder):`);
  const byGroup = new Map<string, number>();
  for (const w of warn) {
    byGroup.set(w.group, (byGroup.get(w.group) ?? 0) + 1);
  }
  for (const [g, n] of [...byGroup].sort((a, b) => b[1] - a[1])) {
    console.log(`  ${g}: ${n} mục`);
  }
}
if (blocking.length > 0) {
  console.error(`\nLỖI: thiếu art ở ${blocking.length} mục mà người chơi SẼ THẤY placeholder.\n`);
  for (const m of blocking.slice(0, 40)) {
    console.error(`  [${m.group}] ${m.label}: ${m.files.join(', ')}`);
  }
  if (blocking.length > 40) {
    console.error(`  … và ${blocking.length - 40} mục nữa.`);
  }
  console.error('\nSinh ảnh: cd server && go run ./cmd/artgen run   (rồi `go run ./cmd/artgen prep`)');
  process.exit(1);
}
console.log('\nOK: không thiếu art nào gây placeholder.');
