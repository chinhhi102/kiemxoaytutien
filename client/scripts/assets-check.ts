// npm run check:assets — chặn build nếu còn thiếu asset BẮT BUỘC.
//
// NETCODE_SPEC §5 Hiển thị mục 1: "build thật cấm placeholder … CI chặn build nếu
// `assets:report` còn thiếu asset bắt buộc".
//
// Khác `assets:report` ở chỗ: report LIỆT KÊ cho người đọc, còn cái này TRẢ MÃ LỖI cho máy.
// Một danh sách thiếu asset in ra rồi build vẫn xanh thì chẳng chặn được gì — ngày 30/10 đội
// test đã chơi một bản có 12 khung placeholder mà không ai biết.
import { existsSync, readdirSync, readFileSync } from 'node:fs';
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

/**
 * mapObstacleKinds đọc MỌI loại vật cản mà bản đồ thật sự dùng.
 *
 * Vì sao phải đọc bản đồ thay vì tin vào danh mục art: ngày 30/10 chủ dự án nhìn thấy cả
 * một rừng khung chữ nhật trắng trong "Trúc Lâm Mê Cung". Nguyên nhân là `bamboo_wall` —
 * bản đồ dùng 300 cái mà KHÔNG CÓ art nào. Bài kiểm cũ không bắt được, vì nó kiểm "mọi mục
 * trong danh mục đều có tệp", còn `bamboo_wall` thì chưa bao giờ nằm trong danh mục.
 *
 * Thiếu một thứ KHÔNG CÓ TÊN thì không có gì để mà thiếu. Nên phép kiểm phải xuất phát từ
 * thứ game THẬT SỰ DÙNG, không phải từ danh sách ta tự khai.
 */
function mapObstacleKinds(): Map<string, number> {
  const out = new Map<string, number>();
  const dir = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'maps');
  if (!existsSync(dir)) {
    return out;
  }
  for (const f of readdirSync(dir).filter((n) => n.endsWith('.tmj'))) {
    const doc = JSON.parse(readFileSync(join(dir, f), 'utf8')) as {
      layers?: { name?: string; objects?: { name?: string; type?: string; properties?: { name: string; value: unknown }[] }[] }[];
    };
    for (const layer of doc.layers ?? []) {
      if (layer.name !== 'obstacles') {
        continue;
      }
      for (const o of layer.objects ?? []) {
        const props = Object.fromEntries((o.properties ?? []).map((p) => [p.name, p.value]));
        const kind = String(props.art ?? props.kind ?? o.type ?? o.name ?? '');
        if (kind) {
          out.set(kind, (out.get(kind) ?? 0) + 1);
        }
      }
    }
  }
  return out;
}

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

// Loại vật cản bản đồ dùng mà không có art → khung chữ nhật trắng giữa màn hình.
// `intact.png` là trạng thái luôn phải có; các trạng thái vỡ có thể thiếu mà vẫn chơi được.
const kinds = mapObstacleKinds();
for (const [kind, n] of kinds) {
  if (!existsSync(join(ASSETS, 'obstacles', kind, 'intact.png'))) {
    blocking.push({ group: 'Vật cản (bản đồ dùng)', label: `${kind} — ${n} chỗ trên bản đồ`, files: [`obstacles/${kind}/intact.png`] });
  }
}

console.log(`Đã kiểm ${checked} tệp bắt buộc trong ${catalog().length} mục, và ${kinds.size} loại vật cản bản đồ dùng.`);
if (warn.length > 0) {
  console.log(`\nThiếu ${warn.length} mục art KHÔNG chặn build (game bỏ qua, không vẽ placeholder):`);
  // In ĐÍCH DANH tệp, không chỉ đếm theo nhóm. Đếm thì biết "thiếu 7 mục" nhưng muốn sinh
  // bù lại phải đi dò tay — mà dò tay là chỗ `bamboo_wall` nằm im suốt nhiều tuần.
  for (const w of warn) {
    console.log(`  [${w.group}] ${w.label}: ${w.files.join(', ')}`);
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
