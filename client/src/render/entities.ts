import { AnimatedSprite, Container, Graphics, GraphicsContext, Sprite, Text, type Texture } from 'pixi.js';

import type { ItemState, MonsterState } from '../net/protocol';
import { assets } from './assets';
import { BUFF_COLORS, MONSTER_COLORS, MONSTER_VARIANTS } from './palette';

const MONSTER_HEIGHT_PER_R = 2.4;
const ITEM_SIZE_PER_R = 2.6;
// ART_PROMPTS §9.0: EXP uses colours the calm ground never has — glowing cyan, then gold.
export const ORB_COLORS = { small: 0x3ee6ff, medium: 0x00b4ff, large: 0xffd23f };
const GLOW_SCALE = 1.9; // halo radius relative to the item
const BUNDLE_COLOR = 0xe0e6ee;
/**
 * BUNDLE_SCALE: Bó Kiếm vẽ to hơn bong bóng kiếm đơn 20% (ART_PROMPTS §8, chủ dự án chốt
 * 30/10). Hai thứ dùng cùng một kiểu bong bóng nên nếu cùng cỡ thì lúc chạy ngang qua không
 * phân biệt được cái nào cho 1 kiếm, cái nào cho cả bó.
 */
const BUNDLE_SCALE = 1.2;
/** COUNT_PX: chiều cao nhãn số kiếm trên MÀN HÌNH, pixel — không đổi theo thu phóng. */
const COUNT_PX = 15;

// Shared drawing contexts: thousands of orbs reuse a handful of geometries.
const contexts = new Map<string, GraphicsContext>();
function shared(key: string, draw: (g: GraphicsContext) => void): GraphicsContext {
  let c = contexts.get(key);
  if (!c) {
    c = new GraphicsContext();
    draw(c);
    contexts.set(key, c);
  }
  return c;
}

/** under draws the drop shadow and soft halo that lift gameplay objects off the ground. */
function under(r: number, glow: number): Graphics {
  return new Graphics(
    shared(`under:${r}:${glow}`, (g) => {
      g.ellipse(0, r * 0.9, r * 0.9, r * 0.32).fill({ color: 0x000000, alpha: 0.28 });
      g.circle(0, 0, r * GLOW_SCALE).fill({ color: glow, alpha: 0.18 });
      g.circle(0, 0, r * 1.35).fill({ color: glow, alpha: 0.18 });
    }),
  );
}

function variantOf(kind: string, id: number): string {
  const list = MONSTER_VARIANTS[kind] ?? ['banh_bao_cau'];
  return list[id % list.length];
}

/** MonsterSprite: animated monster (or placeholder blob) with its HP printed on the belly. */
export class MonsterSprite {
  readonly root = new Container();
  private readonly body: Container;
  private readonly hpBar = new Graphics();
  private readonly r: number;
  private hurtUntil = 0;
  private lastHp = -1;

  constructor(s: MonsterState) {
    const anims = assets.atlas(`monsters/${variantOf(s.k, s.id)}`);
    const idle = anims?.anims.get('idle');
    if (anims && idle) {
      const sp = new AnimatedSprite(idle.textures);
      sp.anchor.set(anims.pivotX, anims.pivotY);
      sp.animationSpeed = idle.fps / 60;
      sp.play();
      const k = (s.r * MONSTER_HEIGHT_PER_R) / 0.72 / sp.texture.height;
      sp.scale.set(k);
      sp.position.set(0, s.r * 0.8);
      this.body = sp;
    } else {
      this.body = new Graphics().circle(0, 0, s.r).fill(MONSTER_COLORS[s.k] ?? 0xffffff).stroke({ color: 0, width: s.r * 0.12 });
    }
    this.r = s.r;
    const halo = new Graphics(shared(`mon:${s.r}`, (g) => g.circle(0, 0, s.r * 1.3).fill({ color: 0xffffff, alpha: 0.14 })));
    this.root.addChild(new Graphics().ellipse(0, s.r * 0.8, s.r, s.r * 0.35).fill({ color: 0, alpha: 0.3 }), halo, this.body, this.hpBar);
  }

  /**
   * drawHp draws a thin bar above the monster — and only once it has been hurt. The old
   * version stamped a big number across the monster's face, which covered the art and left a
   * bare giant digit floating where art was missing.
   */
  private drawHp(hp: number, maxHp: number): void {
    const g = this.hpBar.clear();
    if (hp >= maxHp || hp <= 0 || maxHp <= 1) {
      return; // chưa bị thương (hoặc quái 1 máu) thì không cần thanh máu
    }
    const w = this.r * 1.5;
    const th = Math.max(2, this.r * 0.14);
    const y = -this.r * 1.25;
    const frac = Math.max(0, Math.min(1, hp / maxHp));
    const color = frac > 0.5 ? 0x5dff7a : frac > 0.25 ? 0xffd23f : 0xff5252;
    g.roundRect(-w / 2, y, w, th, th / 2).fill({ color: 0x000000, alpha: 0.55 });
    g.roundRect(-w / 2, y, w * frac, th, th / 2).fill(color);
  }

  update(x: number, y: number, s: MonsterState, now: number): void {
    this.root.position.set(x, y);
    this.root.zIndex = y;
    // `state.ts` đã điền lại máu từ gói trước khi server bỏ trường `hp` (delta snapshot),
    // nên tới đây luôn có số; `?? this.lastHp` chỉ là lưới an toàn cho gói đầu tiên.
    const hp = s.hp ?? this.lastHp;
    if (hp !== this.lastHp) {
      if (this.lastHp > hp) {
        this.hurtUntil = now + 0.2;
      }
      this.lastHp = hp;
      this.drawHp(hp, s.mx);
    }
    const hurt = now < this.hurtUntil;
    // Chỉ nhuộm đỏ khi dính đòn. KHÔNG bóp chiều cao: dòng cũ nhân `scale.y × 0.97` mỗi
    // KHUNG HÌNH nên mỗi lần mất máu là quái lại dẹt thêm một chút, dính vài đòn liên tiếp
    // là bẹp thành bánh tráng và không bao giờ phồng lại.
    this.body.tint = hurt ? 0xff8080 : 0xffffff;
  }

  destroy(): void {
    this.root.destroy({ children: true });
  }
}

/** tienKhiRadius: viên Tiên khí to theo số điểm (2 → nhỏ, 10 → vừa, 150 → khổng lồ). */
export function tienKhiRadius(v: number): number {
  return v >= 100 ? 46 : v >= 10 ? 26 : 17;
}

function itemTexture(it: ItemState): Texture | null {
  if (it.k === 'orb') {
    const size = (it.v ?? 0) >= 60 ? 'large' : (it.v ?? 0) >= 20 ? 'medium' : 'small';
    return assets.texture(`items/orb_${size}.png`);
  }
  if (it.k === 'bundle') {
    return assets.texture('items/bundle.png');
  }
  if (it.k === 'sword') {
    // `sword` có nhánh riêng trong constructor (dùng `items/sword.png`). Chú thích cũ ở đây
    // nói "sword art is drawn inside the placeholder below" là SAI: placeholder của kiếm chỉ
    // vẽ vòng tròn trắng + thanh nâu, không dán ảnh nào.
    return null;
  }
  if (it.k === 'chest') {
    return assets.frames('items/chest', 'idle')[0] ?? null;
  }
  // Bí Kíp nằm trong `items/`, KHÔNG phải `buffs/`.
  //
  // Dòng cuối hàm này đoán đường dẫn từ tên loại (`buffs/<loại>.png`), đúng cho 7 loại buff
  // nhưng sai cho Bí Kíp — và sai trong im lặng: không tìm thấy ảnh thì rơi về placeholder,
  // y như lúc chưa vẽ xong art. Ảnh đã có từ lâu ở `items/skill_scroll.png`, chỉ là không ai
  // nhìn tới. Chủ dự án phát hiện bằng mắt trong game ngày 30/10.
  if (it.k === 'skill_scroll') {
    return assets.texture('items/skill_scroll.png');
  }
  return assets.texture(`buffs/${it.k}.png`);
}

function orbRadius(v: number): number {
  return v >= 60 ? 22 : v >= 20 ? 17 : 12;
}

export function orbSize(v: number): 'small' | 'medium' | 'large' {
  return v >= 60 ? 'large' : v >= 20 ? 'medium' : 'small';
}

/**
 * bundleCount vẽ số kiếm lên bong bóng Bó Kiếm (ART_PROMPTS §8).
 *
 * Gắn vào `root` (đơn vị THẾ GIỚI), KHÔNG gắn làm con của sprite ảnh. Sprite đặt kích thước
 * bằng `width`/`height`, mà PixiJS cài hai thứ đó bằng cách đổi `scale` — nên chữ nằm trong
 * nó bị nhân tỉ lệ LẦN THỨ HAI. Lần đầu tôi làm vậy và con số ra cao đúng 1 pixel.
 */
function bundleCount(n: number, r: number): Text {
  const t = new Text({
    text: String(n),
    style: { fontSize: 44, fontWeight: '900', fill: 0xffffff, stroke: { color: 0x000000, width: 7 } },
  });
  t.anchor.set(0.5);
  // Tỉ lệ thật do `update()` đặt lại mỗi khung theo thu phóng; đây chỉ là giá trị khởi đầu
  // để khung hình đầu tiên không nhảy cỡ.
  t.scale.set(COUNT_PX / 44);
  t.y = r * 0.15;
  return t;
}

/** ItemSprite: bobbing bubble/orb/chest with a shadow and halo. */
export class ItemSprite {
  readonly root = new Container();
  private readonly phase = Math.random() * Math.PI * 2;
  /** count: nhãn số kiếm của Bó Kiếm (null với mọi loại khác). */
  private count: Text | null = null;

  constructor(readonly it: ItemState, itemR: number) {
    const r = it.k === 'orb' ? orbRadius(it.v ?? 0)
      : it.k === 'tien_khi' ? tienKhiRadius(it.v ?? 0)
      : it.k === 'chest' ? itemR * 2.2 : itemR;
    const chestFrames = it.k === 'chest' ? assets.frames('items/chest', 'idle') : [];
    const tex = itemTexture(it);
    let node: Container;
    if (chestFrames.length > 1) {
      const sp = new AnimatedSprite(chestFrames);
      sp.anchor.set(0.5);
      sp.animationSpeed = 0.1;
      sp.play();
      sp.width = sp.height = r * ITEM_SIZE_PER_R;
      node = sp;
    } else if (tex) {
      const sp = new Sprite(tex);
      sp.anchor.set(0.5);
      const k = it.k === 'bundle' ? BUNDLE_SCALE : 1;
      sp.width = sp.height = r * ITEM_SIZE_PER_R * k;
      node = sp;
    } else if (it.k === 'sword' && assets.texture('items/sword.png')) {
      const sp = new Sprite(assets.texture('items/sword.png') as Texture);
      sp.anchor.set(0.5);
      sp.width = sp.height = r * ITEM_SIZE_PER_R;
      node = sp;
    } else {
      node = this.placeholder(it, r);
    }
    const glow = it.k === 'orb' ? ORB_COLORS[orbSize(it.v ?? 0)]
      : it.k === 'tien_khi' ? 0xffd60a
      : it.k === 'chest' ? 0xffd23f : 0xffffff;
    this.root.addChild(under(r, glow), node);
    // Bó rơi từ xác in SỐ KIẾM lên bong bóng (ART_PROMPTS §8): một bó 3 kiếm và một bó 40
    // kiếm nhìn y hệt nhau, mà quyết định có chạy tới nhặt hay không phụ thuộc đúng con số đó.
    if (it.k === 'bundle' && (it.v ?? 0) > 1) {
      this.count = bundleCount(it.v ?? 0, r);
      this.root.addChild(this.count);
    }
    this.root.position.set(it.x, it.y);
    this.root.zIndex = it.y - 1000;
  }

  private placeholder(it: ItemState, r: number): Graphics {
    if (it.k === 'tien_khi') {
      // Viên Tiên khí: vàng kim, nhiều lớp sáng, viên khổng lồ có vành ngoài.
      const g = new Graphics();
      g.circle(0, 0, r * 1.5).fill({ color: 0xffd60a, alpha: 0.18 });
      g.circle(0, 0, r).fill(0xfff3a0).stroke({ color: 0xffb020, width: r * 0.3 });
      g.circle(0, -r * 0.3, r * 0.4).fill({ color: 0xffffff, alpha: 0.85 });
      if ((it.v ?? 0) >= 100) {
        g.circle(0, 0, r * 1.9).stroke({ color: 0xffd60a, width: r * 0.14, alpha: 0.9 });
      }
      return g;
    }
    if (it.k === 'orb') {
      const size = orbSize(it.v ?? 0);
      return new Graphics(shared(`orb:${size}`, (g) => g.circle(0, 0, r).fill(ORB_COLORS[size]).stroke({ color: 0xffffff, width: 3 })));
    }
    const g = new Graphics();
    if (it.k === 'bundle') {
      // Bó Kiếm: a few crossed blades tied with a red band.
      for (const a of [-0.35, 0, 0.35]) {
        const c = Math.cos(a);
        const sn = Math.sin(a);
        const pt = (x: number, y: number): number[] => [x * c - y * sn, x * sn + y * c];
        g.poly([...pt(-3, -r), ...pt(3, -r), ...pt(3, r), ...pt(-3, r)]).fill(BUNDLE_COLOR).stroke({ color: 0x333333, width: 1.5 });
      }
      g.rect(-r * 0.6, -3, r * 1.2, 6).fill(0xe63946);
      return g;
    }
    if (it.k === 'chest') {
      return g.roundRect(-r, -r * 0.7, r * 2, r * 1.4, 8).fill(0x9c6644).stroke({ color: 0xffd23f, width: 6 });
    }
    const color = it.k === 'sword' ? 0xffffff : BUFF_COLORS[it.k] ?? 0xcccccc;
    g.circle(0, 0, r).fill({ color, alpha: 0.35 }).stroke({ color, width: 4 });
    if (it.k === 'sword') {
      g.rect(-3, -r * 0.7, 6, r * 1.4).fill(0xc8a165);
    }
    return g;
  }

  /**
   * update bobs the item and dims it when the local player cannot absorb it — Linh khí và
   * Tiên khí là hai đường tu tách hẳn (spec §3.4), nên viên không dùng được phải nhìn ra
   * ngay chứ không để người chơi chạy theo vô ích.
   */
  update(now: number, absorbable = true, zoom = 1): void {
    this.root.pivot.y = Math.sin(now * 3 + this.phase) * 4;
    this.root.alpha = absorbable ? 1 : 0.3;
    if (this.count) {
      // Giữ cỡ CỐ ĐỊNH trên màn hình, giống bảng tên: camera lùi ra theo cảnh giới nên nhãn
      // tính bằng đơn vị thế giới sẽ teo dần — đo được 7 px ở Luyện Hư, không đọc nổi. Mà
      // con số này tồn tại đúng để người chơi quyết định có chạy tới nhặt hay không.
      this.count.scale.set(COUNT_PX / 44 / Math.max(0.05, zoom));
    }
  }

  destroy(): void {
    this.root.destroy({ children: true });
  }
}
