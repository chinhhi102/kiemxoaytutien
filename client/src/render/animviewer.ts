// Bộ xem animation của chế độ debug `J` (ANSWERS "Trả lời bảng điểm độ giật" mục 1).
//
// Chủ dự án cần chốt ngưỡng đỏ bằng MẮT, mà xem bằng cách chạy quanh bản đồ chờ nhân vật
// tình cờ chơi đúng animation muốn xem thì không bao giờ xem đủ. Bộ này cho đi lần lượt qua
// từng animation theo đúng thứ tự bảng điểm: `]` tiếp, `[` lùi.
//
// Ba điều kiện của yêu cầu, đều có lý do:
//   - **Giữa màn hình**: xem dáng người, không phải xem nhân vật nhỏ xíu lẫn trong trận.
//   - **Chậm 0.5×**: giật là chuyện giữa hai frame liền nhau; ở tốc độ thật mắt không bắt kịp.
//   - **Kèm điểm**: để đối chiếu "máy chấm bao nhiêu" với "mắt tôi thấy thế nào" — đó chính
//     là việc chốt ngưỡng.

import { AnimatedSprite, Container, Graphics, Text } from 'pixi.js';

import { assets } from './assets';
import { animList, animJitter, JITTER_RED, type JitterEntry } from './jitter';

/** PLAY_SPEED: 0.5× tốc độ thật, theo yêu cầu. */
const PLAY_SPEED = 0.5;
/** VIEW_H: chiều cao nhân vật trên màn hình, pixel. To hơn hẳn lúc chơi (87 px) để soi được. */
const VIEW_H = 320;

export class AnimViewer {
  readonly root = new Container();
  private readonly panel = new Graphics();
  private readonly label = new Text({
    text: '',
    style: { fontSize: 20, fontWeight: '700', fill: 0xffffff, stroke: { color: 0x000000, width: 4 }, align: 'center' },
  });
  private readonly hint = new Text({
    text: '[ ] đổi animation · J tắt',
    style: { fontSize: 14, fill: 0xbbbbbb, stroke: { color: 0x000000, width: 3 } },
  });
  private sprite: AnimatedSprite | null = null;
  private list: JitterEntry[] = [];
  private at = 0;

  constructor() {
    this.root.visible = false;
    this.label.anchor.set(0.5, 0);
    this.hint.anchor.set(0.5, 0);
    this.root.addChild(this.panel, this.label, this.hint);
  }

  get open(): boolean {
    return this.root.visible;
  }

  /** show mở bộ xem ở animation đầu bảng (tệ nhất). */
  show(screenW: number, screenH: number): void {
    this.list = animList();
    if (this.list.length === 0) {
      return;
    }
    this.root.visible = true;
    this.at = 0;
    this.layout(screenW, screenH);
    this.render();
  }

  hide(): void {
    this.root.visible = false;
    this.destroySprite();
  }

  /** step đi tới (+1) hoặc lùi (−1) một animation, vòng tròn. */
  step(d: number): void {
    if (!this.root.visible || this.list.length === 0) {
      return;
    }
    this.at = (this.at + d + this.list.length) % this.list.length;
    this.render();
  }

  /** layout đặt khung vào giữa màn hình; gọi lại khi đổi cỡ cửa sổ. */
  layout(screenW: number, screenH: number): void {
    const w = 420;
    const h = VIEW_H + 96;
    const x = (screenW - w) / 2;
    const y = (screenH - h) / 2;
    this.panel.clear().roundRect(x, y, w, h, 12).fill({ color: 0x101010, alpha: 0.85 }).stroke({ color: 0x555555, width: 2 });
    this.label.position.set(screenW / 2, y + 10);
    this.hint.position.set(screenW / 2, y + h - 24);
    this.spriteX = screenW / 2;
    this.spriteY = y + h - 44;
  }

  private spriteX = 0;
  private spriteY = 0;

  private destroySprite(): void {
    if (this.sprite) {
      this.sprite.destroy();
      this.root.removeChild(this.sprite);
      this.sprite = null;
    }
  }

  private render(): void {
    const e = this.list[this.at];
    this.destroySprite();
    const atlas = assets.atlas(`characters/${e.char}`) ?? assets.atlas(`monsters/${e.char}`);
    const anim = atlas?.anims.get(e.anim);
    const score = animJitter(e.char, e.anim);
    const scoreTxt = score === null ? '–' : score.toFixed(1);
    this.label.text = `${this.at + 1}/${this.list.length}  ${e.char} · ${e.anim}\nđiểm ${scoreTxt}${e.loop ? ' (lặp, TB)' : ' (một lần, lạc lõng)'}`;
    this.label.style.fill = score !== null && score >= JITTER_RED ? 0xff6b6b : 0x8ef58e;
    if (!anim) {
      // Atlas bỏ animation này vì cắt sai (xem `miscut` trong assets.ts) — nói thẳng ra,
      // vì một khung trống không phân biệt được với "animation đứng im".
      this.label.text += '\n(atlas đã bỏ animation này: cắt sai số khung)';
      return;
    }
    const sp = new AnimatedSprite(anim.textures);
    sp.anchor.set(atlas!.pivotX, atlas!.pivotY);
    const k = VIEW_H / sp.texture.height;
    sp.scale.set(k);
    sp.position.set(this.spriteX, this.spriteY);
    sp.animationSpeed = (anim.fps / 60) * PLAY_SPEED;
    // Ép LẶP kể cả với animation một lần: xem `death` chạy đúng một lần rồi đứng im thì
    // không soi được chỗ giật, mà soi chỗ giật mới là việc của bộ này.
    sp.loop = true;
    sp.gotoAndPlay(0);
    this.sprite = sp;
    this.root.addChild(sp);
  }
}
