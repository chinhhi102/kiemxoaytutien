import { BitmapText, Container, Graphics } from 'pixi.js';

import { type GameConfig, realmName, realmOf, tienIndex, tierOf } from '../config';
import { STR } from '../ui/strings';
import { FONT_NAME, FONT_REALM, FONT_TITLE } from './fonts';
import { hslToHex, REALM_BADGE } from './palette';
import { quality } from './quality';

/**
 * Bảng tên trên đầu nhân vật (spec §12.3): danh hiệu · cảnh giới · tên.
 *
 * Bản cũ là 3 dòng chữ trơn nằm đè lên vòng kiếm nhiều màu — không đọc được, không có cấp
 * bậc thị giác. Giờ cả cụm có **một tấm nền chung** bo góc đen 45% viền sáng 1px nên chữ
 * luôn nổi trên mọi nền, và ba thành phần xếp theo thứ bậc rõ ràng.
 *
 * Kích thước **cố định trên màn hình**: không phóng theo scale nhân vật, không co theo
 * camera — Đạo Tổ to gấp 6 lần Luyện Khí mà tên vẫn đọc được như nhau.
 *
 * Hiệu năng: nội dung chỉ dựng lại khi ĐỔI (hạng, cảnh giới, tầng, tên); hiệu ứng động
 * (vệt sáng lướt, viền chảy, đốm lấp lánh) là lớp riêng vẽ đè, không đụng tới phần tĩnh.
 */

// Cỡ chữ cố định trên màn hình (spec §12.3).
const PC = { name: 18, realm: 13, title: 13, titleTop1: 15 };
const MOBILE = { name: 15, realm: 11, title: 11, titleTop1: 13 };
const PAD_X = 7;
const PAD_Y = 4;
const GAP = 3;
const HEAD_GAP = 8; // cụm cách đỉnh đầu 8px
/** Nhân vật nhỏ hơn ngần này pixel trên màn hình: chỉ hiện tên (spec §12.3 declutter). */
export const PLATE_FAR_PX = 40;

const RIBBON = [
  { bg: 0xc1121f, edge: 0xffd60a, text: 0xffe9a3, shineSecs: 3 }, // Top 1
  { bg: 0x1b2a41, edge: 0xd8dde3, text: 0xeef2f6, shineSecs: 5 }, // Top 2
  { bg: 0x5c4326, edge: 0xcd7f32, text: 0xf0d9b5, shineSecs: 0 }, // Top 3
];
const RIBBON_SMALL = { bg: 0x000000, edge: 0x9aa3ad, text: 0xf2ece0, shineSecs: 0 }; // Top 4–10

/**
 * inkOn chọn màu chữ đọc được trên nền đã cho: nền sáng thì chữ đậm, nền tối thì chữ trắng.
 * Tính theo độ sáng cảm nhận (mắt người nhạy với xanh lá hơn đỏ và lam rất nhiều) chứ không
 * lấy trung bình ba kênh — lấy trung bình thì nền xanh lá sẽ bị coi là tối và chữ trắng trên
 * nền xanh lá thì không đọc nổi.
 */
function inkOn(bg: number): number {
  const r = (bg >> 16) & 0xff;
  const g = (bg >> 8) & 0xff;
  const b = bg & 0xff;
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.55 ? 0x14110c : 0xffffff;
}

/**
 * shortTitle cắt danh hiệu dài trên màn hẹp.
 *
 * Dải danh hiệu nở ra theo ĐỘ DÀI CHỮ, mà danh hiệu dài nhất là "Bản tăng chưa ngán ai bao
 * giờ" — 29 ký tự. Trên điện thoại dải ấy rộng gần nửa màn hình và che cả minimap (ảnh chụp
 * 30/10). Nhỏ cỡ chữ không cứu được vì nhỏ nữa thì không đọc nổi; phải cắt bớt chữ.
 *
 * Cắt theo TỪ chứ không cắt giữa từ: "Bản tăng chưa…" đọc vẫn ra nghĩa, còn "Bản tăng chư…"
 * thì chỉ như lỗi hiển thị.
 */
const TITLE_MAX_MOBILE = 14;
function shortTitle(s: string, mobile: boolean): string {
  if (!mobile || s.length <= TITLE_MAX_MOBILE) {
    return s;
  }
  const cut = s.slice(0, TITLE_MAX_MOBILE);
  const sp = cut.lastIndexOf(' ');
  return (sp > 6 ? cut.slice(0, sp) : cut.trimEnd()) + '…';
}

/** fit co chữ về đúng cỡ pixel mong muốn (BitmapFont sinh ở cỡ lớn cho nét). */
function fit(t: BitmapText, px: number): void {
  const base = t.style.fontSize as number;
  t.scale.set(px / base);
}

export class Nameplate {
  // (xem `rename` ở cuối lớp — tên không còn readonly vì view được dùng lại từ pool)
  readonly root = new Container();
  private readonly panel = new Graphics();
  private readonly ribbon = new Graphics();
  private readonly badge = new Graphics();
  private readonly fx = new Graphics(); // vệt sáng + đốm lấp lánh + viền chảy
  private readonly selfMark = new Graphics(); // ▼ chỉ vào cụm của chính mình
  private readonly titleText: BitmapText;
  private readonly realmText: BitmapText;
  private readonly dangerText: BitmapText;
  private readonly nameText: BitmapText;
  private key = '';
  private rank = 0;
  private tienIdx = -1;
  private width = 0;
  private ribbonBox = { x: 0, y: 0, w: 0, h: 0 };
  private badgeBox = { x: 0, y: 0, w: 0, h: 0 };
  /** Lúc hạng đổi, để làm hoạt ảnh bật ra / rách rơi (spec §12.3). */
  private rankFrom = -1;
  private rankWas = 0;

  constructor(
    private readonly cfg: GameConfig,
    private name: string,
    private readonly isMe: boolean,
    private readonly mobile: boolean,
  ) {
    const mk = (font: string): BitmapText => {
      const t = new BitmapText({ text: '', style: { fontFamily: font, fontSize: font === FONT_NAME ? 36 : 26 } });
      t.anchor.set(0.5);
      return t;
    };
    this.titleText = mk(FONT_TITLE);
    this.realmText = mk(FONT_REALM);
    this.dangerText = mk(FONT_REALM);
    this.nameText = mk(FONT_NAME);
    this.nameText.tint = isMe ? 0xffd77a : 0xffffff; // tên mình vàng ấm (spec §12.3)
    this.root.addChild(this.panel, this.ribbon, this.badge, this.titleText, this.realmText, this.dangerText, this.nameText, this.selfMark, this.fx);
    if (isMe) {
      // Mũi tên nhỏ ngay trên cụm: giữa đám đông thì tìm mình trong nửa giây (spec §12.3).
      this.selfMark.poly([-5, -5, 5, -5, 0, 3]).fill({ color: 0xffd77a, alpha: 0.95 });
    }
  }

  private get size(): typeof PC {
    return this.mobile ? MOBILE : PC;
  }

  /**
   * update dựng lại nội dung khi cần rồi đặt cụm lên trên đầu nhân vật.
   * `headY` là đỉnh đầu trong toạ độ cục bộ của nhân vật, `zoom` để giữ cỡ cố định.
   */
  update(level: number, rank: number, myRealm: number, headY: number, zoom: number, now: number, near: boolean, fade: number): void {
    const realm = realmOf(this.cfg, level);
    const k = `${level}|${rank}|${Math.sign(realm - myRealm)}|${near}`;
    if (k !== this.key) {
      if (rank !== this.rank) {
        this.rankWas = this.rank;
        this.rankFrom = now;
      }
      this.key = k;
      this.rank = rank;
      this.tienIdx = tienIndex(this.cfg, realm);
      this.rebuild(level, realm, rank, myRealm, near);
    }
    this.root.position.set(0, headY - HEAD_GAP / zoom);
    this.root.scale.set(1 / zoom);
    this.root.alpha = fade;
    this.animate(now);
  }

  /** rebuild vẽ lại toàn bộ phần TĨNH. Chỉ chạy khi nội dung đổi, không chạy mỗi khung. */
  private rebuild(level: number, realm: number, rank: number, myRealm: number, near: boolean): void {
    const s = this.size;
    const showExtras = near; // nhỏ quá thì chỉ còn tên (spec §12.3)

    // --- Danh hiệu ---
    const titlePx = rank === 1 ? s.titleTop1 : s.title;
    const hasRibbon = showExtras && rank >= 1 && rank <= STR.titles.length;
    this.titleText.visible = hasRibbon;
    this.ribbon.visible = hasRibbon;
    if (hasRibbon) {
      this.titleText.text = (rank === 1 ? '♚ ' : '') + shortTitle(STR.titles[rank - 1] ?? '', this.mobile);
      fit(this.titleText, titlePx);
    }

    // --- Huy hiệu cảnh giới ---
    this.badge.visible = showExtras;
    this.realmText.visible = showExtras;
    this.dangerText.visible = showExtras && !this.isMe;
    if (showExtras) {
      const tier = this.tienIdx >= 0 ? '' : ` ${tierOf(this.cfg, level)}`;
      this.realmText.text = realmName(this.cfg, realm).toUpperCase() + tier;
      fit(this.realmText, s.realm);
      // ▲ đỏ = cao hơn mình · ● vàng = ngang · ▼ xanh = thấp hơn (spec §12.3)
      const d = Math.sign(realm - myRealm);
      this.dangerText.text = d > 0 ? '▲' : d < 0 ? '▼' : '●';
      this.dangerText.tint = d > 0 ? 0xff5d5d : d < 0 ? 0x7ee787 : 0xffd60a;
      fit(this.dangerText, s.realm);
    }

    // --- Tên ---
    this.nameText.text = this.name;
    fit(this.nameText, s.name);

    this.layout(realm, rank, hasRibbon, showExtras);
  }

  /** layout xếp ba tầng rồi vẽ nền chung, ruy băng và huy hiệu vừa khít. */
  private layout(realm: number, rank: number, hasRibbon: boolean, showExtras: boolean): void {
    const badgeW = showExtras ? this.realmText.width + (this.dangerText.visible ? this.dangerText.width + 4 : 0) + PAD_X * 2 : 0;
    const badgeH = showExtras ? this.realmText.height + PAD_Y : 0;
    this.width = Math.max(this.nameText.width, badgeW, hasRibbon ? this.titleText.width + PAD_X * 2 : 0) + PAD_X * 2;
    const ribbonH = hasRibbon ? this.titleText.height + PAD_Y * 1.5 : 0;

    // Vẽ từ dưới lên: tên ở đáy (sát đầu nhân vật), danh hiệu trên cùng.
    let y = -PAD_Y - this.nameText.height / 2;
    this.nameText.position.set(0, y);
    y -= this.nameText.height / 2 + GAP;
    if (showExtras) {
      const by = y - badgeH / 2;
      this.badgeBox = { x: -badgeW / 2, y: by - badgeH / 2, w: badgeW, h: badgeH };
      const textLeft = -badgeW / 2 + PAD_X;
      this.realmText.position.set(textLeft + this.realmText.width / 2, by);
      if (this.dangerText.visible) {
        this.dangerText.position.set(textLeft + this.realmText.width + 4 + this.dangerText.width / 2, by);
      }
      y -= badgeH + GAP;
    }
    if (hasRibbon) {
      const ry = y - ribbonH / 2;
      const rw = this.titleText.width + PAD_X * 2.5;
      this.ribbonBox = { x: -rw / 2, y: ry - ribbonH / 2, w: rw, h: ribbonH };
      this.titleText.position.set(0, ry);
      y -= ribbonH;
    }

    // Nền chung: đen 45%, bo góc, viền mảnh sáng → chữ nổi trên mọi nền.
    const top = y - PAD_Y;
    this.selfMark.position.set(0, top - 5);
    const p = this.panel.clear();
    p.roundRect(-this.width / 2, top, this.width, -top + PAD_Y, 6)
      .fill({ color: 0x000000, alpha: 0.45 })
      .stroke({ color: 0xffffff, width: 1, alpha: 0.18 });

    // Ruy băng danh hiệu
    const rb = this.ribbon.clear();
    if (hasRibbon) {
      const st = rank <= 3 ? RIBBON[rank - 1] : RIBBON_SMALL;
      const b = this.ribbonBox;
      const tail = Math.min(10, b.h * 0.8); // hai đuôi xẻ
      rb.poly([
        b.x, b.y,
        b.x + b.w, b.y,
        b.x + b.w - tail, b.y + b.h / 2,
        b.x + b.w, b.y + b.h,
        b.x, b.y + b.h,
        b.x + tail, b.y + b.h / 2,
      ])
        .fill({ color: st.bg, alpha: rank <= 3 ? 0.92 : 0.5 })
        .stroke({ color: st.edge, width: rank <= 3 ? 1.5 : 1, alpha: 0.9 });
      this.titleText.tint = st.text;
    }

    // Huy hiệu cảnh giới: viên thuốc bo tròn, màu theo cảnh giới
    const bg = this.badge.clear();
    if (showExtras) {
      const b = this.badgeBox;
      const col = REALM_BADGE[realm] ?? 0x8d99ae;
      bg.roundRect(b.x, b.y, b.w, b.h, b.h / 2)
        .fill({ color: col, alpha: 0.92 })
        .stroke({ color: 0x000000, width: 1, alpha: 0.35 });
      // Đại Thừa: nền đen, chữ vàng. Còn lại chọn đen/trắng theo độ sáng của nền.
      this.realmText.tint = realm === this.cfg.cultivation.realms.length - 1 ? 0xffd60a : inkOn(col);
    }
  }

  /** animate vẽ phần ĐỘNG: vệt sáng lướt, đốm lấp lánh, viền chảy của Tiên giới. */
  private animate(now: number): void {
    const g = this.fx.clear();
    if (quality() === 'low') {
      return; // mức Thấp: giữ màu tĩnh (spec §12.3)
    }
    // Viền chảy quanh huy hiệu Tiên giới; Đạo Tổ đổi màu liên tục.
    if (this.tienIdx >= 0 && this.badge.visible) {
      const b = this.badgeBox;
      const spec = this.cfg.cultivation.tien_aura_colors[this.tienIdx] ?? '#ffffff';
      const col = spec === 'yin_yang' ? hslToHex((now * 70) % 360, 0.9, 0.62) : Number(`0x${spec.replace('#', '')}`);
      g.roundRect(b.x - 1, b.y - 1, b.w + 2, b.h + 2, (b.h + 2) / 2).stroke({ color: col, width: 1.6, alpha: 0.5 + 0.4 * Math.sin(now * 3) });
    }
    if (!this.ribbon.visible || this.rank > 3) {
      return;
    }
    const st = RIBBON[this.rank - 1];
    const b = this.ribbonBox;
    // Vệt sáng lướt qua chữ
    if (st.shineSecs > 0) {
      const t = (now % st.shineSecs) / st.shineSecs;
      if (t < 0.25) {
        const x = b.x + b.w * (t / 0.25) * 1.3 - b.w * 0.15;
        g.poly([x, b.y, x + 7, b.y, x + 1, b.y + b.h, x - 6, b.y + b.h]).fill({ color: 0xffffff, alpha: 0.35 });
      }
    }
    // Đốm lấp lánh quanh ruy băng Top 1
    if (this.rank === 1) {
      for (let i = 0; i < 3; i++) {
        const t = (now * 0.7 + i / 3) % 1;
        const x = b.x + b.w * ((i * 0.37 + 0.2) % 1);
        const y = b.y + (i % 2 === 0 ? -3 : b.h + 3);
        g.circle(x, y, 1.6 * (1 - Math.abs(t - 0.5) * 2)).fill({ color: 0xfff3a0, alpha: 0.9 });
      }
    }
    // Lên hạng: bật ra. Mất hạng: ruy băng rơi xuống.
    if (this.rankFrom >= 0) {
      const t = now - this.rankFrom;
      const up = this.rankWas === 0 || this.rank < this.rankWas;
      const dur = up ? 0.35 : 0.4;
      if (t >= dur) {
        this.rankFrom = -1;
        this.ribbon.scale.set(1);
        this.ribbon.y = 0;
      } else if (up) {
        const k = t / dur;
        this.ribbon.scale.set(k < 0.6 ? (k / 0.6) * 1.2 : 1.2 - ((k - 0.6) / 0.4) * 0.2);
      } else {
        this.ribbon.y = (t / dur) * 14;
        this.ribbon.alpha = 1 - t / dur;
      }
    }
  }


  /**
   * rename đổi tên trên bảng tên tại chỗ, cho View Pool dùng lại view
   * (NETCODE_SPEC §5 "View Pool" + mục Hiển thị 5 "đổi tại chỗ, không tạo lại view").
   *
   * Quên gọi hàm này khi lấy view ra khỏi pool là nhân vật mới hiện lên mang tên người cũ.
   */
  rename(name: string): void {
    this.name = name;
    this.nameText.text = name;
  }

  destroy(): void {
    this.root.destroy({ children: true });
  }
}
