import { AnimatedSprite, Container, Graphics, Sprite, Text, type Texture } from 'pixi.js';

import type { GameConfig } from '../config';
import type { PlayerFrame } from '../net/state';
import { BUFFS, FLAG, statusOn } from '../net/protocol';
import { assets, type AnimSet } from './assets';
import { animJitter, JITTER_RED } from './jitter';
import { animShort, initials } from './catalog';
import { BUFF_COLORS, charColor } from './palette';
import { moodName } from '../net/protocol';
import { Input } from '../input';
import { Nameplate, PLATE_FAR_PX } from './nameplate';
import { SwordRing } from './swordring';

// Visual proportions relative to the body hitbox radius (not balance numbers).
const CHAR_HEIGHT_PER_R = 2.6; // drawn character height = 2.6 × body radius
const FEET_BELOW_CENTER = 0.75; // feet sit 0.75 × radius below the hitbox centre
const ART_FILL = 0.72; // characters fill 72% of their 512 canvas (ART_PROMPTS §1)
// Largest text box inside the oval of ui/speech_bubble.png, as fractions of the canvas.
const BUBBLE_INNER = { x0: 0.21, y0: 0.15, x1: 0.65, y1: 0.59 };
const BUBBLE_MAX_TEXT_W = 220;
/** Mờ dần vào khi nhân vật lọt vào khung hình (NETCODE_SPEC §5 mục 4). */
const FADE_IN_SECS = 0.12;

// Animations missing from an atlas fall back along this chain.
const FALLBACK: Record<string, string[]> = {
  // idle có thể bị bỏ nếu sheet cắt sai (xem assets.ts/miscut) — vẫn phải có gì để vẽ.
  idle: ['run', 'chase'],
  chase: ['run'],
  flee: ['run'],
  hit: ['idle'],
  levelup: ['idle'],
  breakthrough: ['levelup', 'idle'],
  realm_drop: ['hit', 'idle'],
  teleport_out: ['idle'],
  teleport_in: ['idle'],
  death: ['hit', 'idle'],
  respawn: ['idle'],
};

/** PlayerSprite draws one character: body, sword ring, aura, name tag, emote bubble. */
export class PlayerSprite {
  readonly root = new Container();
  private readonly swordRing: SwordRing;
  private readonly shadow = new Graphics();
  private readonly aura = new Graphics();
  private readonly bodyBox = new Container();
  private body: AnimatedSprite | Container | null = null;
  /** Bảng tên trên đầu: danh hiệu · cảnh giới · tên, gộp thành một cụm (spec §12.3). */
  private readonly plate: Nameplate;
  private atlas: AnimSet | null;
  private mood = '';
  private bubble: Container | null = null;
  private readonly animLabel = new Text({ text: '', style: { fontSize: 40, fontWeight: '800', fill: 0xffffff, stroke: { color: 0x000000, width: 6 } } });
  private bubbleUntil = 0;
  /** Vòng icon trạng thái quanh nhân vật (🚫 khi không nhặt được đồ — spec v5.5 §6.3). */
  private readonly statusIcons = new Container();
  private statusKey = '';
  /** bornAt: lúc view bắt đầu được vẽ, để mờ dần vào. */
  private bornAt = 0;
  /** animInfo: dòng chữ của phím debug `J` (GAME_SPEC §12.5 C, dòng cuối). */
  private readonly animInfo = new Text({ text: '', style: { fontSize: 32, fontWeight: '700', fill: 0x8ef58e, stroke: { color: 0x000000, width: 6 } } });

  constructor(
    private readonly cfg: GameConfig,
    readonly id: number,
    name: string,
    public char: string,
    private readonly isMe: boolean,
  ) {
    this.atlas = assets.atlas(`characters/${char}`);
    this.swordRing = new SwordRing(cfg);
    this.plate = new Nameplate(cfg, name, isMe, Input.isTouchDevice());
    // Hào quang Tiên giới KHÔNG còn nằm ở đây: nó chuyển sang `vfxLayer` dùng chung để ăn
    // một lượt bloom cho cả màn hình thay vì mỗi nhân vật một filter (spec §3.6).
    // `animInfo` gắn vào ROOT chứ không vào thân placeholder: thân placeholder chỉ tồn tại
    // khi THIẾU art, mà phím J lại chủ yếu dùng để soi art ĐÃ CÓ. Gắn nhầm chỗ là bấm J
    // không thấy gì — đúng lỗi tôi vừa mắc.
    this.animInfo.anchor.set(0.5, 0);
    this.animInfo.scale.set(0.013);
    // Đặt TRÊN bảng tên chứ không dưới chân: vòng kiếm vẽ ở lớp riêng nằm trên nhân vật,
    // nên mọi chỗ quanh chân đều bị nó che — chụp thử ở Luyện Hư thì chữ mất hẳn sau vòng
    // kiếm tím. Phía trên đầu là chỗ duy nhất luôn trống.
    this.animInfo.y = -2.6;
    this.animInfo.visible = false;
    this.root.addChild(this.shadow, this.aura, this.bodyBox, this.statusIcons, this.plate.root, this.animInfo);
    if (!this.atlas) {
      this.body = this.placeholderBody();
      this.bodyBox.addChild(this.body);
    }
  }

  /**
   * setCharacter đổi nhân vật **tại chỗ**, không dựng lại view (NETCODE_SPEC §5 mục 5).
   *
   * Vì sao cần: gói snapshot chỉ gửi tên nhân vật (`ch`) LẦN ĐẦU client thấy một người. Nếu
   * client từng mất view (xem `state.ts expire`) thì view dựng lại có `char` rỗng → vẽ
   * placeholder, và tới nhịp gửi lại tên mới có `char` thật. Bản cũ gặp `char` đổi thì
   * **huỷ cả sprite rồi dựng lại** — thêm một lần nhấp nháy nữa, đúng lúc người chơi đang
   * nhìn nhân vật hiện ra.
   *
   * Trả về `false` nếu không đổi được (chưa có atlas) để nơi gọi biết mà thử lại sau.
   */
  /**
   * showAnimInfo bật/tắt dòng chữ debug của phím `J` (GAME_SPEC §12.5 C).
   *
   * Hiện NGAY TRÊN từng nhân vật chứ không gom vào một bảng góc màn hình: cái cần nhìn là
   * "con này đang giật" — phải thấy chữ cạnh đúng con đang giật thì mới đối chiếu được với
   * mắt mình.
   */
  showAnimInfo(on: boolean): void {
    this.animInfo.visible = on;
    if (!on) {
      this.animInfo.text = '';
    }
  }

  /**
   * updateAnimInfo cập nhật dòng debug; chỉ chạy khi phím `J` đang bật.
   *
   * Vị trí tính theo `headY` và `zoom` — ĐÚNG hai thứ bảng tên dùng — chứ không đặt một
   * hằng số: toạ độ trong `root` là ĐƠN VỊ THẾ GIỚI, mà nhân vật rộng vài chục đơn vị và
   * camera lùi xa dần theo cảnh giới. Đặt "y = -2.6" như tôi làm lúc đầu thì ở Luyện Hư nó
   * dịch đúng 0.7 pixel, tức là chữ nằm đè lên giữa nhân vật và không ai đọc được.
   */
  private updateAnimInfo(headY: number, zoom: number): void {
    if (!this.animInfo.visible) {
      return;
    }
    // Chia cho zoom để chữ giữ nguyên cỡ trên màn hình dù camera xa hay gần — giống bong
    // bóng thoại. Đặt phía TRÊN bảng tên (bảng tên cao khoảng 26 px màn hình).
    this.animInfo.scale.set(0.34 / zoom);
    this.animInfo.y = headY - 46 / zoom;
    const sp = this.body as AnimatedSprite | null;
    const frames = sp?.textures?.length ?? 0;
    const cur = sp?.currentFrame ?? -1;
    const phase = frames > 0 ? (cur / frames) : 0;
    const jit = animJitter(this.char, this.mood);
    this.animInfo.text = `${this.mood || '?'} ${cur + 1}/${frames} pha ${phase.toFixed(2)}`
      + (jit === null ? '' : ` giật ${jit.toFixed(0)}`);
    this.animInfo.style.fill = jit !== null && jit >= JITTER_RED ? 0xff6b6b : 0x8ef58e;
  }

  /** hasAtlas: đã có ảnh thật chưa, hay đang vẽ placeholder (ngân sách §4.3 là 0 khung). */
  hasAtlas(): boolean {
    return this.atlas !== null;
  }

  setCharacter(char: string): boolean {
    if (char === this.char) {
      return true;
    }
    const next = assets.atlas(`characters/${char}`);
    if (!next) {
      return false;
    }
    this.char = char;
    this.atlas = next;
    this.mood = ''; // ép `updateBody` dựng lại thân theo atlas mới
    if (this.body) {
      this.body.destroy({ children: true });
      this.body = null;
    }
    return true;
  }

  // Placeholder (no art yet): coloured circle + character initials + current animation name.
  private placeholderBody(): Container {
    const [main, accent] = charColor(this.char);
    const c = new Container();
    const g = new Graphics().circle(0, 0, 1).fill(main).stroke({ color: 0x000000, width: 0.12 });
    g.circle(0, -0.55, 0.32).fill(accent);
    const t = new Text({ text: initials(this.char || '?'), style: { fontSize: 44, fontWeight: '900', fill: 0xffffff, stroke: { color: 0x000000, width: 6 } } });
    t.anchor.set(0.5);
    t.scale.set(0.02);
    t.y = 0.1;
    this.animLabel.anchor.set(0.5, 0);
    this.animLabel.scale.set(0.016);
    this.animLabel.y = 1.25;
    c.addChild(g, t, this.animLabel);
    return c;
  }

  /** swordCount: số kiếm đang vẽ, cho đồng hồ đo hiệu năng ở chế độ debug. */
  get swordCount(): number {
    return this.swordRing.count;
  }

  /** ring is the sword ring layer; the scene draws all rings above characters and obstacles. */
  get ring(): Container {
    return this.swordRing.root;
  }

  update(f: PlayerFrame, zoom: number, now: number, showHitbox: boolean, swordDetail = 1, myRealm = 0, plateFade = 1): void {
    const s = f.s;
    const r = this.cfg.player.body_r * s.sc;
    this.root.position.set(f.x, f.y);
    this.root.zIndex = f.y + r * FEET_BELOW_CENTER; // y-sort by the feet (spec §8.2)
    this.drawShadow(r);
    this.updateBody(moodName(s.md), r, Math.cos(s.f) < 0);
    this.drawAura(s.bf, r, now);
    if (s.fl & FLAG.burning) {
      this.drawBurn(r, now); // vẽ đè lên aura buff
    }
    this.applyAlpha(s.fl, now);
    // Xếp từ dưới lên: tên → cảnh giới → danh hiệu (spec §11.2, §11.3).
    const headY = -r * (CHAR_HEIGHT_PER_R - FEET_BELOW_CENTER) - 4 / zoom;
    this.plate.update(s.L, s.ti ?? 0, myRealm, headY, zoom, now, r * 2.6 * zoom >= PLATE_FAR_PX, plateFade);
    this.swordRing.update(f, now, showHitbox, swordDetail);
    this.updateAnimInfo(headY, zoom);
    this.updateStatusIcons(s.st ?? 0, r, now);
    this.updateBubble(r, zoom, now);
  }

  /**
   * updateStatusIcons vẽ icon trạng thái bay quanh nhân vật.
   *
   * Hiện chỉ có 🚫 lúc bị Phun Trân Châu, vì spec v5.5 đòi **người xung quanh cũng thấy**
   * nạn nhân đang không nhặt được gì — lớp trân châu thì chỉ nạn nhân nhìn thấy, nên nếu
   * không có icon này thì từ ngoài nhìn vào không ai hiểu vì sao họ đứng cạnh viên Linh khí
   * mà không nhặt.
   *
   * Dựng lại chỉ khi tập icon ĐỔI (so bằng `statusKey`): vẽ lại Text mỗi khung là tạo
   * texture mới 30 lần/giây cho mỗi nhân vật trên màn hình.
   */
  private updateStatusIcons(st: number, r: number, now: number): void {
    const icons: string[] = [];
    if (statusOn(st, 'blind')) {
      icons.push('🚫');
    }
    const key = icons.join('');
    if (key !== this.statusKey) {
      this.statusKey = key;
      this.statusIcons.removeChildren().forEach((c) => c.destroy({ children: true }));
      for (const ch of icons) {
        this.statusIcons.addChild(new Text({ text: ch, style: { fontSize: 36 }, anchor: 0.5 }));
      }
    }
    this.statusIcons.visible = icons.length > 0;
    if (!this.statusIcons.visible) {
      return;
    }
    // Bay vòng quanh người: thấy ngay cả khi nhân vật đứng im.
    const orbit = r * 1.5;
    const n = this.statusIcons.children.length;
    this.statusIcons.children.forEach((c, i) => {
      const a = now * 2 + (i * Math.PI * 2) / n;
      c.position.set(Math.cos(a) * orbit, Math.sin(a) * orbit * 0.5 - r * 0.6);
      c.scale.set(r / 40);
    });
  }

  private drawShadow(r: number): void {
    this.shadow.clear().ellipse(0, r * FEET_BELOW_CENTER, r * 0.9, r * 0.35).fill({ color: 0x000000, alpha: 0.25 });
  }

  private updateBody(mood: string, r: number, flip: boolean): void {
    if (this.atlas) {
      this.setMood(mood);
      const h = (r * CHAR_HEIGHT_PER_R) / ART_FILL; // world height of the whole 512 canvas
      const sp = this.body as AnimatedSprite;
      const k = h / sp.texture.height;
      sp.scale.set(flip ? -k : k, k);
      sp.position.set(0, r * FEET_BELOW_CENTER);
      return;
    }
    const body = this.body as Container;
    if (this.mood !== mood) {
      this.mood = mood;
      this.animLabel.text = animShort(mood);
    }
    // The placeholder circle is symmetric: never mirror it, so the initials stay readable.
    body.scale.set(r, r);
    const squash = mood === 'hit' || mood === 'death' ? 0.7 : 1;
    body.scale.y = r * squash;
  }

  private setMood(mood: string): void {
    if (mood === this.mood || !this.atlas) {
      return;
    }
    this.mood = mood;
    const anim = this.findAnim(mood);
    if (!anim) {
      return;
    }
    if (!this.body) {
      const sp = new AnimatedSprite(anim.textures);
      sp.anchor.set(this.atlas.pivotX, this.atlas.pivotY);
      this.body = sp;
      this.bodyBox.addChild(sp);
    }
    const sp = this.body as AnimatedSprite;
    sp.textures = anim.textures;
    sp.animationSpeed = anim.fps / 60;
    sp.loop = anim.loop;
    sp.gotoAndPlay(0);
  }

  private findAnim(mood: string): { textures: Texture[]; fps: number; loop: boolean } | null {
    const set = this.atlas?.anims;
    if (!set) {
      return null;
    }
    for (const name of [mood, ...(FALLBACK[mood] ?? []), 'idle']) {
      const a = set.get(name);
      if (a) {
        return a;
      }
    }
    return set.values().next().value ?? null;
  }





  /** drawBurn is the blood-red flame ring of Thiêu Đốt Tinh Huyết (spec §4.1c). */
  private drawBurn(r: number, now: number): void {
    const flicker = 1 + 0.12 * Math.sin(now * 22);
    this.aura
      .circle(0, r * 0.2, r * 1.3 * flicker)
      .fill({ color: 0xff3b1f, alpha: 0.16 })
      .stroke({ color: 0xff6b2c, width: r * 0.14, alpha: 0.9 });
    // Lưỡi lửa liếm lên quanh chân, lệch pha nhau cho đỡ đều đặn.
    for (let i = 0; i < 6; i++) {
      const a = (i / 6) * Math.PI * 2 + now * 3;
      const h = r * (0.45 + 0.3 * Math.sin(now * 14 + i * 1.7));
      this.aura
        .circle(Math.cos(a) * r * 1.1, r * 0.2 + Math.sin(a) * r * 0.45 - h * 0.4, h * 0.35)
        .fill({ color: i % 2 ? 0xffb020 : 0xff4a1a, alpha: 0.5 });
    }
  }

  private drawAura(bits: number, r: number, now: number): void {
    this.aura.clear();
    let ring = 0;
    BUFFS.forEach((b, i) => {
      if (!(bits & (1 << i)) || !BUFF_COLORS[b]) {
        return;
      }
      const pulse = 1 + 0.06 * Math.sin(now * 6 + i);
      if (b === 'shield') {
        this.aura.circle(0, -r * 0.4, r * 1.5 * pulse).fill({ color: BUFF_COLORS[b], alpha: 0.18 }).stroke({ color: BUFF_COLORS[b], width: r * 0.08, alpha: 0.8 });
        return;
      }
      this.aura.circle(0, r * 0.2, r * (1.15 + ring * 0.18) * pulse).stroke({ color: BUFF_COLORS[b], width: r * 0.1, alpha: 0.7 });
      ring++;
    });
  }

  private applyAlpha(flags: number, now: number): void {
    let a = 1;
    const stealth = (flags & FLAG.stealth) !== 0;
    if (stealth && !this.isMe) {
      // Spec §6.2: Ẩn Thân = tàng hình 80%, nhìn thấy lờ mờ chứ không biến mất hẳn.
      a = this.cfg.buffs.stealth?.visible_alpha ?? 0.2;
    } else if (flags & FLAG.bush || stealth) {
      a = 0.45; // in a bush, or my own stealth so I can tell the buff is up
    }
    // Ẩn Thân / trong bụi: giấu cả cụm bảng tên của người khác (spec §12.3).
    this.plate.root.visible = (!stealth && !(flags & FLAG.bush)) || this.isMe;
    if (flags & FLAG.invuln && Math.floor(now * 10) % 2 === 0) {
      a *= 0.4; // blink while invulnerable (spec §5.1)
    }
    if (flags & FLAG.transit) {
      a *= 0.5;
    }
    // Mờ dần VÀO trong 120 ms khi nhân vật lọt vào khung hình (NETCODE_SPEC §5 mục 4):
    // bật sáng đột ngột đọc ra y như nhấp nháy, kể cả khi view không hề bị dựng lại.
    if (this.bornAt === 0) {
      this.bornAt = now;
    }
    a *= Math.min(1, (now - this.bornAt) / FADE_IN_SECS);
    this.root.alpha = a;
    this.ring.alpha = (flags & FLAG.stealth ? 0.45 : 1) * Math.min(1, (now - this.bornAt) / FADE_IN_SECS);
    this.swordRing.visible = !(flags & FLAG.dead);
  }

  // ---- emote bubble --------------------------------------------------------

  showBubble(text: string, now: number): void {
    this.bubble?.destroy({ children: true });
    const c = new Container();
    const t = new Text({ text, style: { fontFamily: 'system-ui, sans-serif', fontSize: 16, fontWeight: '800', fill: 0x111111, wordWrap: true, wordWrapWidth: BUBBLE_MAX_TEXT_W, align: 'center' } });
    t.anchor.set(0.5);
    const tex = assets.texture('ui/speech_bubble.png');
    const pad = 10;
    if (tex) {
      // Size the art so its white inner box (not the whole canvas) wraps the text.
      const bg = new Sprite(tex);
      const box = BUBBLE_INNER;
      bg.anchor.set((box.x0 + box.x1) / 2, (box.y0 + box.y1) / 2);
      bg.width = (t.width + pad * 2) / (box.x1 - box.x0);
      bg.height = (t.height + pad * 2) / (box.y1 - box.y0);
      c.addChild(bg);
    } else {
      c.addChild(new Graphics().roundRect(-t.width / 2 - pad, -t.height / 2 - pad / 2, t.width + pad * 2, t.height + pad, 12).fill(0xffffff).stroke({ color: 0, width: 3 }));
    }
    c.addChild(t);
    this.bubble = c;
    this.bubbleUntil = now + 1.8;
    this.root.addChild(c);
  }

  private updateBubble(r: number, zoom: number, now: number): void {
    if (!this.bubble) {
      return;
    }
    if (now > this.bubbleUntil) {
      this.bubble.destroy({ children: true });
      this.bubble = null;
      return;
    }
    this.bubble.scale.set(1 / zoom);
    this.bubble.position.set(r * 0.8, -r * (CHAR_HEIGHT_PER_R - FEET_BELOW_CENTER) - 34 / zoom);
  }

  destroy(): void {
    this.plate.destroy();
    this.root.destroy({ children: true });
    this.swordRing.destroy();
  }
}
