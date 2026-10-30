package main

import (
	"fmt"
	"strings"
)

// Sinh bộ giải mã/mã hoá TypeScript — nửa client của NETCODE_SPEC §4.9.
//
// Hai phía sinh từ CÙNG một `Schema` trong CÙNG một lần chạy, nên không thể lệch nhau vì
// quên cập nhật một bên. Đó chính là điều spec muốn: "không viết tay `DataView` hai phía".

func genTS(s *Schema) string {
	var b strings.Builder
	b.WriteString(banner)
	b.WriteString("\n/* eslint-disable */\n\n")
	fmt.Fprintf(&b, "/**\n * SCHEMA_HASH là băm của proto/schema.txt.\n"+
		" *\n * Client gửi nó lúc bắt tay; server so với bản của mình và TỪ CHỐI nếu lệch (§4.9).\n"+
		" * Không có bước này thì client cũ gặp server mới sẽ đọc rác và đứng hình — đã dính\n"+
		" * đúng một lần với trường `mood`.\n */\n"+
		"export const SCHEMA_HASH = %q;\n\n", s.Hash)
	fmt.Fprintf(&b, "export const SCHEMA_VERSION = %d;\n\n", s.Version)

	b.WriteString("/** Loại message — byte đầu của gói. */\nexport const MsgType = {\n")
	for _, m := range s.Messages {
		fmt.Fprintf(&b, "  %s: %d,\n", m.Name, m.ID)
	}
	b.WriteString("} as const;\n\n")

	for _, m := range s.Messages {
		b.WriteString("/**\n")
		for _, d := range m.Doc {
			fmt.Fprintf(&b, " * %s\n", d)
		}
		b.WriteString(" */\n")
		fmt.Fprintf(&b, "export interface %s {\n", m.Name)
		for _, f := range m.Fields {
			c := ""
			if f.Comment != "" {
				c = " // " + f.Comment
			}
			opt := ""
			if f.If != "" {
				// Trường có điều kiện là tuỳ chọn: thiếu nó nghĩa là "không đổi", KHÔNG phải
				// "bằng 0". Lẫn hai thứ đó là nhân vật nhảy về gốc toạ độ mỗi lần đứng yên.
				opt = "?"
			}
			fmt.Fprintf(&b, "  %s%s: %s;%s\n", tsName(f.Name), opt, tsType(f.Type), c)
		}
		b.WriteString("}\n\n")
		genTSDecode(&b, m)
		genTSEncode(&b, m)
	}
	b.WriteString(tsHelpers)
	return b.String()
}

func genTSDecode(b *strings.Builder, m Message) {
	fmt.Fprintf(b, "/** decode%s đọc một %s từ `r`. */\n", m.Name, m.Name)
	fmt.Fprintf(b, "export function decode%s(r: Reader): %s {\n", m.Name, m.Name)
	mask := m.maskField()
	b.WriteString("  const v = {} as " + m.Name + ";\n")
	for _, f := range m.Fields {
		ind := "  "
		if f.If != "" {
			fmt.Fprintf(b, "  if ((v.%s & (1 << %d)) !== 0) {\n", tsName(mask), m.bitIndex(f.If))
			ind = "    "
		}
		fmt.Fprintf(b, "%sv.%s = r.%s();\n", ind, tsName(f.Name), tsRead(f.Type))
		if f.If != "" {
			b.WriteString("  }\n")
		}
	}
	b.WriteString("  return v;\n}\n\n")
}

func genTSEncode(b *strings.Builder, m Message) {
	fmt.Fprintf(b, "/** encode%s ghi một %s vào `w`. Dùng cho bài đối chiếu và cho gói client gửi lên. */\n", m.Name, m.Name)
	fmt.Fprintf(b, "export function encode%s(w: Writer, v: %s): void {\n", m.Name, m.Name)
	mask := m.maskField()
	for _, f := range m.Fields {
		ind := "  "
		if f.If != "" {
			fmt.Fprintf(b, "  if ((v.%s & (1 << %d)) !== 0) {\n", tsName(mask), m.bitIndex(f.If))
			ind = "    "
		}
		def := "0"
		if f.Type == "str" {
			def = "''"
		}
		fmt.Fprintf(b, "%sw.%s(v.%s ?? %s);\n", ind, tsWrite(f.Type), tsName(f.Name), def)
		if f.If != "" {
			b.WriteString("  }\n")
		}
	}
	b.WriteString("}\n\n")
}

// tsRead ánh xạ kiểu schema sang tên phương thức của `Reader`.
//
// `u8`/`i8` thành `u8v`/`i8v`: `Uint8Array` đã chiếm tên `u8` trong lớp Reader, và một
// phương thức trùng tên trường nội bộ là lỗi chỉ lộ ra lúc CHẠY, không phải lúc biên dịch.
func tsRead(t string) string {
	switch t {
	case "str":
		return "str"
	case "u8":
		return "u8v"
	case "i8":
		return "i8v"
	default:
		return t
	}
}

func tsWrite(t string) string {
	switch t {
	case "str":
		return "putStr"
	default:
		return "put" + strings.ToUpper(t[:1]) + t[1:]
	}
}

// tsName đổi `ring_phase` thành `ringPhase` (kiểu TypeScript).
func tsName(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

const tsHelpers = `
// ── đọc/ghi nguyên thuỷ ──────────────────────────────────────────────────────────────────
//
// MỌI số đều little-endian. ` + "`DataView`" + ` mặc định là BIG-endian, nên mọi lời gọi phải
// truyền ` + "`true`" + `. Quên một chỗ là đọc đúng kiểu nhưng sai giá trị — và sai im lặng,
// không có lỗi nào được ném ra. Đó là lý do hai phía phải sinh từ cùng một nguồn.

const TEXT_DECODER = new TextDecoder();
const TEXT_ENCODER = new TextEncoder();

/** Reader đọc tuần tự từ một ArrayBuffer. */
export class Reader {
  private readonly dv: DataView;
  private readonly u8: Uint8Array;
  p = 0;

  constructor(buf: ArrayBuffer | Uint8Array) {
    const u8 = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
    this.u8 = u8;
    this.dv = new DataView(u8.buffer, u8.byteOffset, u8.byteLength);
  }

  get left(): number {
    return this.u8.length - this.p;
  }

  private need(n: number): void {
    if (this.p + n > this.u8.length) {
      throw new RangeError(` + "`wire: gói ngắn (cần ${n} byte ở vị trí ${this.p}/${this.u8.length})`" + `);
    }
  }

  u8v(): number { this.need(1); return this.dv.getUint8(this.p++); }
  i8v(): number { this.need(1); return this.dv.getInt8(this.p++); }

  u16(): number { this.need(2); const v = this.dv.getUint16(this.p, true); this.p += 2; return v; }
  i16(): number { this.need(2); const v = this.dv.getInt16(this.p, true); this.p += 2; return v; }
  u32(): number { this.need(4); const v = this.dv.getUint32(this.p, true); this.p += 4; return v; }
  i32(): number { this.need(4); const v = this.dv.getInt32(this.p, true); this.p += 4; return v; }
  f32(): number { this.need(4); const v = this.dv.getFloat32(this.p, true); this.p += 4; return v; }

  str(): string {
    const n = this.u8v();
    this.need(n);
    const s = TEXT_DECODER.decode(this.u8.subarray(this.p, this.p + n));
    this.p += n;
    return s;
  }
}

/** Writer ghi tuần tự, tự nới mảng. */
export class Writer {
  private buf = new Uint8Array(256);
  private len = 0;

  private need(n: number): void {
    if (this.len + n <= this.buf.length) { return; }
    let cap = this.buf.length * 2;
    while (cap < this.len + n) { cap *= 2; }
    const next = new Uint8Array(cap);
    next.set(this.buf.subarray(0, this.len));
    this.buf = next;
  }

  private dv(n: number): DataView {
    this.need(n);
    return new DataView(this.buf.buffer, this.buf.byteOffset + this.len, n);
  }

  putU8(v: number): void { this.need(1); this.buf[this.len++] = v & 0xff; }
  putI8(v: number): void { this.need(1); this.dv(1).setInt8(0, v); this.len += 1; }
  putU16(v: number): void { this.dv(2).setUint16(0, v, true); this.len += 2; }
  putI16(v: number): void { this.dv(2).setInt16(0, v, true); this.len += 2; }
  putU32(v: number): void { this.dv(4).setUint32(0, v, true); this.len += 4; }
  putI32(v: number): void { this.dv(4).setInt32(0, v, true); this.len += 4; }
  putF32(v: number): void { this.dv(4).setFloat32(0, v, true); this.len += 4; }

  putStr(v: string): void {
    const b = TEXT_ENCODER.encode(v);
    // Cắt ở 255 byte giống hệt phía Go: hai bên phải xử lý chuỗi dài y như nhau, nếu không
    // bài đối chiếu sẽ đỏ vì một lý do không ai ngờ tới.
    const n = Math.min(b.length, 255);
    this.putU8(n);
    this.need(n);
    this.buf.set(b.subarray(0, n), this.len);
    this.len += n;
  }

  bytes(): Uint8Array {
    return this.buf.subarray(0, this.len);
  }
}
`
