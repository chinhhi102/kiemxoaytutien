// npm run check:wire -- <file.json> — bài đối chiếu mã hoá hai chiều (NETCODE_SPEC §4.9).
//
//   cd server && go run ./cmd/protofixture -o /tmp/wire.json
//   cd client && npm run check:wire -- /tmp/wire.json
//
// Kiểm HAI chiều, và cần cả hai:
//   1. **Giải mã**: client đọc byte server tạo ra → từng trường phải khớp.
//   2. **Mã hoá ngược**: client ghi lại từ chính giá trị đó → byte phải GIỐNG HỆT.
//
// Chiều 2 không thừa. Một lỗi đối xứng — ví dụ cả hai phía cùng đọc/ghi big-endian ở một
// trường — sẽ qua được chiều 1 mà vẫn sai so với schema. Và quan trọng hơn: client CÓ gửi
// gói lên (Input), nên bộ mã hoá phía client là mã chạy thật, không phải mã chỉ để test.
import { readFileSync } from 'node:fs';

import {
  decodeDesc, decodeDespawn, decodeInput, decodeSelfState, decodeSpawn, decodeUpdate,
  encodeDesc, encodeDespawn, encodeInput, encodeSelfState, encodeSpawn, encodeUpdate,
  Reader, SCHEMA_HASH, SCHEMA_VERSION, Writer,
} from '../src/net/wire.gen.ts';

interface Case {
  msg: string;
  fields: Record<string, unknown>;
  bytes: string;
}

interface Fixture {
  schema_hash: string;
  schema_version: number;
  cases: Case[];
}

const path = process.argv[2];
if (!path) {
  console.error('Thiếu đường dẫn fixture.\n  cd server && go run ./cmd/protofixture -o /tmp/wire.json');
  process.exit(2);
}

const fx = JSON.parse(readFileSync(path, 'utf8')) as Fixture;

// Hash phải khớp TRƯỚC khi so gói. Lệch hash nghĩa là fixture sinh từ một schema khác, và
// mọi khác biệt sau đó chỉ là hệ quả — báo "100 gói sai" thay vì "bạn quên chạy protogen"
// là bắt người đọc đi tìm nhầm chỗ.
if (fx.schema_hash !== SCHEMA_HASH) {
  console.error(`LỖI: schema lệch.\n  fixture: ${fx.schema_hash}\n  client:  ${SCHEMA_HASH}`);
  console.error('Chạy `cd server && go run ./cmd/protogen` rồi sinh lại fixture.');
  process.exit(1);
}
if (fx.schema_version !== SCHEMA_VERSION) {
  console.error(`LỖI: version lệch (${fx.schema_version} vs ${SCHEMA_VERSION}).`);
  process.exit(1);
}

type Decoder = (r: Reader) => Record<string, unknown>;
type Encoder = (w: Writer, v: never) => void;

const decoders: Record<string, Decoder> = {
  Spawn: decodeSpawn as Decoder,
  Desc: decodeDesc as Decoder,
  Update: decodeUpdate as Decoder,
  Despawn: decodeDespawn as Decoder,
  SelfState: decodeSelfState as Decoder,
  Input: decodeInput as Decoder,
};
const encoders: Record<string, Encoder> = {
  Spawn: encodeSpawn as Encoder,
  Desc: encodeDesc as Encoder,
  Update: encodeUpdate as Encoder,
  Despawn: encodeDespawn as Encoder,
  SelfState: encodeSelfState as Encoder,
  Input: encodeInput as Encoder,
};

function hex(b: Uint8Array): string {
  return [...b].map((x) => x.toString(16).padStart(2, '0')).join(' ');
}

let bad = 0;
const shown: string[] = [];
const seen = new Map<string, number>();

for (let i = 0; i < fx.cases.length; i++) {
  const c = fx.cases[i];
  seen.set(c.msg, (seen.get(c.msg) ?? 0) + 1);
  const dec = decoders[c.msg];
  const enc = encoders[c.msg];
  if (!dec || !enc) {
    shown.push(`[${i}] ${c.msg}: client không có bộ giải mã — schema có message mà client chưa sinh lại?`);
    bad++;
    continue;
  }
  const raw = Buffer.from(c.bytes, 'base64');
  const bytes = new Uint8Array(raw);

  // Chiều 1 — giải mã.
  let got: Record<string, unknown>;
  try {
    const r = new Reader(bytes);
    got = dec(r);
    if (r.left !== 0) {
      shown.push(`[${i}] ${c.msg}: đọc xong còn thừa ${r.left} byte — hai phía không cùng số trường`);
      bad++;
      continue;
    }
  } catch (e) {
    shown.push(`[${i}] ${c.msg}: giải mã ném lỗi — ${String(e)}`);
    bad++;
    continue;
  }

  let ok = true;
  for (const [k, want] of Object.entries(c.fields)) {
    const have = got[k];
    if (have !== want) {
      shown.push(`[${i}] ${c.msg}.${k}: client đọc ${JSON.stringify(have)}, server ghi ${JSON.stringify(want)}`);
      ok = false;
    }
  }
  // Trường client đọc ra mà server không gửi: với `Update` đó là bug thật (bit tắt mà vẫn
  // đọc), với message khác thì không thể xảy ra.
  for (const k of Object.keys(got)) {
    if (!(k in c.fields)) {
      shown.push(`[${i}] ${c.msg}.${k}: client đọc ra trường server KHÔNG gửi (bit tắt mà vẫn đọc?)`);
      ok = false;
    }
  }

  // Chiều 2 — mã hoá ngược.
  const w = new Writer();
  enc(w, got as never);
  const back = w.bytes();
  if (back.length !== bytes.length || back.some((v, j) => v !== bytes[j])) {
    shown.push(`[${i}] ${c.msg}: mã hoá ngược ra byte khác\n    server: ${hex(bytes)}\n    client: ${hex(back)}`);
    ok = false;
  }

  if (!ok) {
    bad++;
  }
}

const summary = [...seen].map(([k, n]) => `${k} ${n}`).join(' · ');
console.log(`schema ${SCHEMA_HASH} v${SCHEMA_VERSION} · ${fx.cases.length} gói (${summary})`);
if (bad > 0) {
  console.error(`\nLỖI: ${bad}/${fx.cases.length} gói không khớp.\n`);
  for (const line of shown.slice(0, 25)) {
    console.error('  ' + line);
  }
  if (shown.length > 25) {
    console.error(`  … còn ${shown.length - 25} khác biệt nữa.`);
  }
  process.exit(1);
}
console.log('OK: client và server mã hoá giống hệt nhau, cả hai chiều.');
