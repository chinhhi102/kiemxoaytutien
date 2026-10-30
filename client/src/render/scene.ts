import { Application, Container } from 'pixi.js';
import { AnimViewer } from './animviewer';
import { FADE_OUT_SECS, ViewPool } from './viewpool';
import { BobaOverlay } from './boba';

import { sound } from '../audio/sound';
import { absorbable, banksTienKhi, cameraViewHeight, type GameConfig, realmName, realmOf, tienIndex } from '../config';
import type { ClientWorld } from '../net/state';
import type { ItemState } from '../net/protocol';
import { FLAG, type GameEvent, moodName } from '../net/protocol';
import { pick, STR } from '../ui/strings';
import { Beast, type BeastMode, TIEN_BEASTS } from './beast';
import { ItemSprite, MonsterSprite } from './entities';
import { MapView, type Viewer } from './mapview';
import { OrbLayer } from './orblayer';
import { Ambient } from './ambient';
import { Breakthrough, HITSTOP_SECS } from './breakthrough';
import { autoQuality } from './quality';
import { AURA_FAR_PX, TienAura } from './tienaura';
import { VfxLayer } from './vfxlayer';
import { PlayerSprite } from './player';
import { ringGeom } from './ringgeom';
import { Vfx } from './vfx';

const SUCK_SECS = 0.22; // vật phẩm nhặt được bay hút vào người trong 0.22s
// Ngân sách số thanh kiếm vẽ mỗi khung hình. Vượt thì các VÒNG DÀY bị giãn bớt — vòng dày
// vốn là một vành sát thương liền (§4.1b) nên bỏ bớt thanh trong vành nhìn vẫn y hệt, còn
// vòng thưa giữ nguyên vì ở đó từng thanh là một lưỡi kiếm riêng người chơi phải né.
// Không có trần này thì 5 người Đại Thừa 500 kiếm gặp nhau là 2 500 thanh × 60 khung/giây.
// Spec §3.5 v4.7: thần thú LUÔN bay quanh người Tiên giới, không hẹn giờ, không tắt được.
// Giới hạn duy nhất là vệt hạt — chỉ bật cho vài con gần mình nhất.
const BEAST_TRAIL_NEAR = 6;
// Chủ nhỏ hơn ngần này pixel trên màn hình thì thần thú rút gọn còn đầu + 4 đốt.
const BEAST_FAR_PX = 40;

const SWORD_BUDGET = 1600;

/**
 * drawCalls đếm số lệnh vẽ WebGL mỗi khung. Pixi không công bố con số này, mà nó lại là chỉ
 * số quan trọng nhất để biết nghẽn ở GPU hay ở CPU, nên bọc thẳng hàm vẽ của ngữ cảnh WebGL.
 * Chỉ bật ở chế độ debug (spec §13.6).
 */
export const drawCalls = { perFrame: 0, acc: 0, frames: 0 };

export function countDrawCalls(gl: WebGL2RenderingContext): void {
  for (const name of ['drawElements', 'drawArrays', 'drawElementsInstanced', 'drawArraysInstanced'] as const) {
    const fn = gl[name] as (...a: unknown[]) => void;
    (gl as unknown as Record<string, unknown>)[name] = (...a: unknown[]): void => {
      drawCalls.acc += 1;
      fn.apply(gl, a);
    };
  }
}

/** countNodes đếm số đối tượng hiển thị đang nằm trong cây vẽ. */
function countNodes(c: { children?: unknown[] }): number {
  let n = 1;
  for (const k of (c.children ?? []) as { children?: unknown[] }[]) {
    n += countNodes(k);
  }
  return n;
}
const MOOD_EMOTE_CHANCE = 0.12; // chance per mood change that a character shouts (run/chase/flee)

/** GameScene renders the ClientWorld every frame with a camera that zooms out as you grow (spec §4.1). */
export class GameScene {
  readonly app = new Application();
  private readonly camera = new Container();
  private readonly entities = new Container();
  /** Viên Linh khí vẽ riêng bằng particle: vài trăm viên cùng lúc, xem orblayer.ts. */
  private readonly orbs = new OrbLayer();
  /** Lớp phát sáng dùng chung, ăn một lượt bloom cho cả màn hình (spec §3.6). */
  private readonly glow = new VfxLayer();
  /** Sự sống của bản đồ: mây trôi, lá rơi, đom đóm… chỉ ở client (spec §8.6 mục 5). */
  private readonly ambient = new Ambient();
  /** Màn đột phá Tiên giới: sóng xung kích + khựng hình + chớp trắng + chữ (spec §3.6). */
  /** Lớp trân châu dính màn hình khi trúng kỹ năng Thằng Mập (spec §6.3). */
  readonly boba = new BobaOverlay();
  private readonly breakthrough = new Breakthrough();
  private hitstopUntil = 0;
  private readonly auras = new Map<number, TienAura>();
  private readonly rings = new Container();
  private readonly vfx = new Vfx();
  private mapView: MapView | null = null;
  private readonly players = new Map<number, PlayerSprite>();
  private readonly monsters = new Map<number, MonsterSprite>();
  private readonly items = new Map<number, ItemSprite>();
  // Items picked up this snapshot: itemId → who grabbed it, so we can fly them in.
  private readonly pickedBy = new Map<number, number>();
  private readonly flying: { sp: ItemSprite; by: number; from: { x: number; y: number }; at: number }[] = [];
  /** Bản sao trạng thái viên EXP, chỉ để dựng hiệu ứng bay hút lúc nó vừa bị nhặt. */
  private readonly lastOrb = new Map<number, ItemState>();
  private readonly lastMood = new Map<number, string>();
  private zoom = 1;
  private viewH = 900;
  private world: ClientWorld | null = null;
  private lastFrames: ReturnType<ClientWorld['playerFrames']> = [];
  private fps = 0;
  private fpsLast = 0;
  private fpsAcc = 0;
  private fpsFrames = 0;
  private lastRenderMs = 0;
  private detail = 1;
  private lastSpark = { t: 0, x: 0, y: 0 };
  private showHitbox = false;
  // Thần thú Tiên giới (spec §3.5): mỗi con có 2 lớp, một dưới nhân vật một trên.
  private beasts: { b: Beast; owner: number }[] = [];
  /** Cảnh giới Tiên của từng người ở khung hình trước — RealmDrop cần biết họ VỪA là gì. */
  private lastTien = new Map<number, number>();

  constructor(private readonly cfg: GameConfig) {}

  /** viewSize is the world-space rectangle currently on screen (used by the minimap, §12.1). */
  viewSize(): { w: number; h: number } {
    return { w: this.viewH * (this.app.screen.width / Math.max(this.app.screen.height, 1)), h: this.viewH };
  }

  async init(parent: HTMLElement, maxFps: number): Promise<void> {
    await this.app.init({
      resizeTo: window,
      background: 0x2b2b2b,
      antialias: true,
      resolution: Math.min(window.devicePixelRatio || 1, 2),
      autoDensity: true,
    });
    this.app.ticker.maxFPS = maxFps;
    const gl = (this.app.renderer as unknown as { gl?: WebGL2RenderingContext }).gl;
    if (gl && new URLSearchParams(location.search).get('debug') === 'true') {
      countDrawCalls(gl); // chỉ đếm khi đang đo, tránh phí ở bản chơi thật
    }
    parent.appendChild(this.app.canvas);
    this.entities.sortableChildren = true;
    this.rings.sortableChildren = true;
    // Lớp trân châu nằm TRÊN cảnh nhưng vẫn DƯỚI HUD (HUD là DOM, không phải Pixi), đúng
    // yêu cầu "không che HUD" của spec §6.3.
    // Bộ xem animation nằm TRÊN CÙNG và ngoài camera: nó là khung xem cố định giữa màn
    // hình, không phải thứ nằm trong thế giới (ANSWERS mục 1).
    this.app.stage.addChild(this.camera, this.breakthrough.overlay, this.boba.root, this.animViewer.root);
    this.boba.resize(this.app.screen.width, this.app.screen.height);
    this.app.renderer.on('resize', (w: number, h: number) => this.boba.resize(w, h));
  }

  setWorld(w: ClientWorld): void {
    this.world = w;
    this.clear();
    if (!w.map) {
      return;
    }
    this.mapView = new MapView(w.map, this.cfg.gates.radius);
    for (const node of this.mapView.obstacleLayer()) {
      this.entities.addChild(node);
    }
    // Rings sit under every character so a blade never covers a body.
    // Viên EXP luôn nằm dưới mọi thứ khác nên xếp ngay trên mặt đất, không cần y-sort.
    // Lớp phát sáng nằm dưới nhân vật (hào quang ở sau lưng) nhưng trên vòng kiếm, để
    // hàng trăm thanh kiếm không che mất hào quang.
    this.camera.addChild(this.mapView.ground, this.mapView.decor, this.mapView.under, this.orbs.root, this.rings, this.glow.root, this.entities, this.mapView.over, this.ambient.root, this.vfx.layer);
    this.ambient.setBiome(w.map.id);
  }

  clear(): void {
    this.camera.removeChildren();
    // Pool phải dọn theo: view trong pool thuộc về khu CŨ, giữ lại là giữ texture của một
    // bản đồ không còn dùng — đúng kiểu rò rỉ bộ nhớ mang tên "tối ưu".
    this.pool.clear();
    this.leaving.clear();
    for (const p of this.players.values()) {
      p.destroy();
    }
    for (const m of this.monsters.values()) {
      m.destroy();
    }
    for (const it of this.items.values()) {
      it.destroy();
    }
    for (const f of this.flying) {
      f.sp.destroy();
    }
    for (const e of this.beasts) {
      e.b.destroy();
    }
    this.beasts.length = 0;
    this.lastTien.clear();
    this.flying.length = 0;
    this.pickedBy.clear();
    this.orbs.clear();
    this.lastOrb.clear();
    this.ambient.clear();
    for (const a of this.auras.values()) {
      a.destroy();
    }
    this.auras.clear();
    this.players.clear();
    this.monsters.clear();
    this.items.clear();
    this.entities.removeChildren();
    this.rings.removeChildren();
  }

  /** render one frame; dt in seconds. */
  render(dt: number): void {
    const w = this.world;
    if (!w || !this.mapView) {
      return;
    }
    const t0 = performance.now();
    const now = t0 / 1000;
    this.breakthrough.update(now, this.app.screen.width, this.app.screen.height);
    this.boba.update(now, dt);
    if (now < this.hitstopUntil) {
      return; // khựng hình: đóng băng cảnh một nhịp rất ngắn cho cú đột phá "nặng" hơn
    }
    const frames = w.playerFrames(dt);
    this.lastFrames = frames;
    const me = frames.find((f) => f.view.id === w.myId);
    if (me) {
      this.updateCamera(me.x, me.y, me.s.sc, ringGeom(this.cfg, me.s.sc, me.s.n).reach, now);
    }
    autoQuality(this.fps, now); // máy yếu thì tự hạ chất lượng (spec §13.6 mục 8)
    this.glow.apply(); // mức chất lượng vừa đổi thì gắn lại bloom (§3.6)
    this.syncPlayers(frames, now);
    this.syncMonsters(now);
    this.syncItems(now, me ? banksTienKhi(this.cfg, me.s.L) : false);
    this.updateFlyingItems(now);
    this.updateBeasts(frames, now);
    const myBush = me && me.s.fl & 2 ? this.bushAt(me.x, me.y) : -1;
    if (me) {
      const v = this.viewSize();
      this.ambient.update(me.x, me.y, v.w / 2, v.h / 2, now);
    }
    const vs = this.viewSize();
    this.mapView.update(w, now, myBush, me ? this.viewer(me.x, me.y, me.s.sc) : null, this.cfg.obstacles.occlusion_alpha,
      me ? { x: me.x, y: me.y, w: vs.w, h: vs.h } : undefined);
    this.vfx.update(now);
    // Trung bình trượt để số không nhảy loạn; gồm cả lượt vẽ của Pixi ở khung trước.
    this.lastRenderMs = this.lastRenderMs * 0.9 + (performance.now() - t0) * 0.1;
    drawCalls.frames += 1;
    if (drawCalls.frames >= 30) {
      drawCalls.perFrame = drawCalls.acc / drawCalls.frames;
      drawCalls.acc = 0;
      drawCalls.frames = 0;
    }
  }

  /** toggleHitbox shows the server's sword capsules over the drawn swords (debug key H). */
  toggleHitbox(): void {
    this.showHitbox = !this.showHitbox;
  }

  private viewer(x: number, y: number, scale: number): Viewer {
    const r = this.cfg.player.body_r * scale;
    return { x, top: y - r * 1.9, feet: y + r * 0.75, halfW: r * 0.8 };
  }

  private bushAt(x: number, y: number): number {
    for (const z of this.world?.map?.zones ?? []) {
      if (z.k !== 'bush') {
        continue;
      }
      const s = z.s;
      const inside = s.c ? (x - s.x) ** 2 + (y - s.y) ** 2 <= (s.r ?? 0) ** 2 : x >= s.x && x <= s.x + (s.w ?? 0) && y >= s.y && y <= s.y + (s.h ?? 0);
      if (inside) {
        return z.id;
      }
    }
    return -1;
  }

  private updateCamera(x: number, y: number, scale: number, reach: number, now: number): void {
    // Smoothly follow the camera height so growing does not pop.
    const targetH = cameraViewHeight(this.cfg, scale, reach);
    this.viewH += (targetH - this.viewH) * 0.08;
    this.zoom = this.app.screen.height / this.viewH;
    const shake = this.vfx.shakeOffset(now);
    this.camera.scale.set(this.zoom);
    this.camera.position.set(this.app.screen.width / 2 - (x + shake.x) * this.zoom, this.app.screen.height / 2 - (y + shake.y) * this.zoom);
  }

  private syncPlayers(frames: ReturnType<ClientWorld['playerFrames']>, now: number): void {
    const seen = new Set<number>();
    const detail = this.swordDetail(frames);
    const me = frames.find((f) => f.view.id === this.world?.myId);
    const myRealm = me ? realmOf(this.cfg, me.s.L) : 0;
    for (const f of frames) {
      seen.add(f.view.id);
      let sp = this.players.get(f.view.id);
      // Đổi nhân vật thì đổi TẠI CHỖ; chỉ dựng view mới khi thật sự chưa có (xem
      // `PlayerSprite.setCharacter`). Bản cũ huỷ rồi dựng lại, thêm một lần nhấp nháy.
      if (sp && sp.char !== f.view.char && f.view.char) {
        sp.setCharacter(f.view.char);
      }
      if (!sp) {
        // Lấy từ pool trước khi dựng mới: dựng một `PlayerSprite` là tạo cả chục đối tượng
        // PixiJS (thân, vòng kiếm, bảng tên, bóng, hào quang). Làm việc đó giữa trận, mỗi
        // lần có người vào khung hình, là nguồn giật rác bộ nhớ — §5 nói rõ "không tạo/hủy
        // object trong lúc chơi".
        sp = this.pool.take(this.cfg, f.view.id, f.view.name, f.view.char, f.view.id === this.world?.myId);
        this.players.set(f.view.id, sp);
        this.entities.addChild(sp.root);
        this.rings.addChild(sp.ring);
        // Nhân vật vừa xuất hiện cũng phải bật sẵn lớp debug nếu phím J đang bật — không thì
        // đúng những con mới vào khung (chỗ hay giật nhất) lại là chỗ không có số.
        if (this.animInfoOn) {
          sp.showAnimInfo(true);
        }
      }
      sp.update(f, this.zoom, now, this.showHitbox, detail, myRealm, this.plateFade(f, frames));
      this.syncAura(f, now);
      this.moodChatter(f.view.id, moodName(f.s.md), now);
    }
    // Rời khung hình: MỜ DẦN RA 150 ms rồi mới về pool, không biến mất đột ngột
    // (NETCODE_SPEC §5 Hiển thị mục 4). Huỷ hẳn sprite cũng bị bỏ: view vào pool, giữ
    // texture, lần sau lấy ra dùng lại (§5 "View Pool").
    for (const [id, sp] of this.players) {
      if (seen.has(id)) {
        this.leaving.delete(id);
        continue;
      }
      const since = this.leaving.get(id);
      if (since === undefined) {
        this.leaving.set(id, now);
        continue;
      }
      const t = (now - since) / FADE_OUT_SECS;
      if (t < 1) {
        sp.setLeaveFade(1 - t);
        continue;
      }
      this.leaving.delete(id);
      this.players.delete(id);
      this.pool.put(sp);
    }
    for (const [id, a] of this.auras) {
      if (!seen.has(id)) {
        a.destroy();
        this.auras.delete(id);
      }
    }
  }

  /**
   * plateFade làm mờ bảng tên của người Ở XA khi hai bảng chồng lên nhau, để cụm gần mình
   * luôn đọc được (spec §12.3 declutter).
   */
  private plateFade(f: ReturnType<ClientWorld['playerFrames']>[number], all: ReturnType<ClientWorld['playerFrames']>): number {
    const me = all.find((p) => p.view.id === this.world?.myId);
    if (!me || f.view.id === me.view.id) {
      return 1;
    }
    const mine = Math.hypot(f.x - me.x, f.y - me.y);
    for (const o of all) {
      if (o.view.id === f.view.id || o.view.id === me.view.id) {
        continue;
      }
      // Hai bảng coi là chồng nhau khi gần nhau theo chiều ngang và sàn sàn chiều cao.
      if (Math.abs(o.x - f.x) < 90 && Math.abs(o.y - f.y) < 50 && Math.hypot(o.x - me.x, o.y - me.y) < mine) {
        return 0.5;
      }
    }
    return 1;
  }

  /** syncAura dựng / cập nhật hào quang Tiên giới trong lớp phát sáng dùng chung (§3.6). */
  private syncAura(f: ReturnType<ClientWorld['playerFrames']>[number], now: number): void {
    const idx = tienIndex(this.cfg, realmOf(this.cfg, f.s.L));
    let a = this.auras.get(f.view.id);
    if (idx < 0) {
      if (a) {
        a.destroy();
        this.auras.delete(f.view.id);
      }
      return;
    }
    if (!a) {
      a = new TienAura(this.cfg);
      this.auras.set(f.view.id, a);
      this.glow.root.addChild(a.root);
    }
    a.setRealm(idx);
    const r = this.cfg.player.body_r * f.s.sc;
    // Ẩn Thân / trong bụi: mờ theo chủ (spec §3.4).
    const hidden = (f.s.fl & FLAG.stealth) !== 0 || (f.s.fl & FLAG.bush) !== 0;
    a.update(f.x, f.y + r * 0.75, r, now, hidden ? 0.2 : 1, r * 2.6 * this.zoom >= AURA_FAR_PX);
  }

  private moodChatter(id: number, mood: string, now: number): void {
    if (this.lastMood.get(id) === mood) {
      return;
    }
    this.lastMood.set(id, mood);
    const lines = STR.emotes[mood];
    if (lines && Math.random() < MOOD_EMOTE_CHANCE) {
      this.players.get(id)?.showBubble(pick(lines), now);
    }
  }

  private syncMonsters(now: number): void {
    const seen = new Set<number>();
    for (const m of this.world?.monsterFrames() ?? []) {
      seen.add(m.s.id);
      let sp = this.monsters.get(m.s.id);
      if (!sp) {
        sp = new MonsterSprite(m.s);
        this.monsters.set(m.s.id, sp);
        this.entities.addChild(sp.root);
      }
      sp.update(m.x, m.y, m.s, now);
    }
    for (const [id, sp] of this.monsters) {
      if (!seen.has(id)) {
        sp.destroy();
        this.monsters.delete(id);
      }
    }
  }

  private syncItems(now: number, meBanksTienKhi: boolean): void {
    const items = this.world?.items;
    if (!items) {
      return;
    }
    for (const [id, it] of items) {
      if (it.k === 'orb') {
        this.orbs.add(it);
        continue;
      }
      if (!this.items.has(id)) {
        const sp = new ItemSprite(it, this.cfg.items.item_r);
        this.items.set(id, sp);
        this.entities.addChild(sp.root);
      }
    }
    for (const [id, sp] of this.items) {
      if (!items.has(id)) {
        this.items.delete(id);
        const by = this.pickedBy.get(id);
        this.pickedBy.delete(id);
        if (by === undefined) {
          sp.destroy(); // hết hạn / ra khỏi tầm nhìn: biến mất luôn
        } else {
          // Bị nhặt → bay hút về phía người nhặt thay vì biến mất đột ngột.
          this.flying.push({ sp, by, from: { x: sp.root.x, y: sp.root.y }, at: now });
        }
        continue;
      }
      sp.update(now, absorbable(sp.it.k, meBanksTienKhi), this.zoom);
    }
    this.syncOrbs(items, now, meBanksTienKhi);
  }

  /** syncOrbs dọn viên EXP đã biến mất và cho viên bị nhặt bay hút vào người nhặt. */
  private syncOrbs(items: Map<number, ItemState>, now: number, meBanksTienKhi: boolean): void {
    for (const id of this.orbs.ids()) {
      if (items.has(id)) {
        continue;
      }
      const by = this.pickedBy.get(id);
      const at = this.orbs.at(id);
      this.pickedBy.delete(id);
      if (by !== undefined && at) {
        const ghost = new ItemSprite({ ...(this.lastOrb.get(id) as ItemState), x: at.x, y: at.y }, this.cfg.items.item_r);
        this.entities.addChild(ghost.root);
        this.flying.push({ sp: ghost, by, from: at, at: now });
      }
      this.lastOrb.delete(id);
      this.orbs.remove(id);
    }
    for (const [id, it] of items) {
      if (it.k === 'orb' && !this.lastOrb.has(id)) {
        this.lastOrb.set(id, it);
      }
    }
    this.orbs.update(now, absorbable('orb', meBanksTienKhi));
  }

  /** updateFlyingItems tweens picked-up items into their owner, then drops them. */
  private updateFlyingItems(now: number): void {
    for (let i = this.flying.length - 1; i >= 0; i--) {
      const f = this.flying[i];
      const t = Math.min(1, (now - f.at) / SUCK_SECS);
      const owner = this.players.get(f.by)?.root;
      const tx = owner?.x ?? f.from.x;
      const ty = owner?.y ?? f.from.y;
      // Ease-in: lề mề lúc đầu rồi lao vút vào người, giống bị hút.
      const k = t * t;
      f.sp.root.position.set(f.from.x + (tx - f.from.x) * k, f.from.y + (ty - f.from.y) * k);
      f.sp.root.scale.set(1 - 0.7 * t);
      f.sp.root.alpha = 1 - t * t;
      if (t >= 1) {
        f.sp.destroy();
        this.flying.splice(i, 1);
      }
    }
  }

  // ---- events → VFX, sound, bubbles ---------------------------------------

  handleEvents(events: GameEvent[]): void {
    const now = performance.now() / 1000;
    for (const e of events) {
      this.handleEvent(e, now);
    }
  }

  private playerScale(id: number | undefined): number {
    const v = id !== undefined ? this.world?.players.get(id) : undefined;
    return v ? v.snaps[v.snaps.length - 1].s.sc : 1;
  }

  /**
   * swordDetail chọn mức giãn chung cho cả màn hình: 1 = vẽ đủ, 2 = vẽ cách một thanh…
   * Dùng bậc thang 1/2/3/4 kèm mép trễ để số kiếm dao động quanh ngưỡng không làm dựng lại
   * vòng kiếm liên tục.
   */
  private swordDetail(frames: ReturnType<ClientWorld['playerFrames']>): number {
    let total = 0;
    for (const f of frames) {
      total += f.s.n;
    }
    const want = Math.ceil(total / SWORD_BUDGET);
    if (want > this.detail) {
      this.detail = want; // quá tải thì giãn ngay
    } else if (want <= this.detail - 1 && total < SWORD_BUDGET * (this.detail - 1) * 0.85) {
      this.detail -= 1; // nhẹ hẳn rồi mới dày lại, tránh dựng lại liên tục
    }
    return Math.max(1, Math.min(4, this.detail));
  }

  /**
   * perfLine là dòng số liệu cho đồng hồ debug: khung hình, thời gian vẽ, và số thứ đang vẽ.
   * Có nó thì lúc ai đó kêu lag là biết ngay nghẽn ở đâu — khung hình thấp mà `vẽ` nhỏ là
   * nghẽn mạng chứ không phải máy yếu, còn `vẽ` cao là do quá nhiều thứ trên màn hình.
   */
  /**
   * drawStats là những con số chỉ đếm được bằng cách duyệt các nhân vật ĐANG VẼ
   * (TELEMETRY_SPEC §2.2). Để ở đây chứ không để `players` lộ ra ngoài: vòng lặp nằm cạnh
   * dữ liệu, và không ai ngoài scene chạm được vào bảng sprite.
   */
  /** showAnimInfo bật/tắt lớp chữ debug của phím `J` trên MỌI nhân vật đang vẽ. */
  showAnimInfo(on: boolean): void {
    this.animInfoOn = on;
    if (on) {
      this.animViewer.show(this.app.screen.width, this.app.screen.height);
    } else {
      this.animViewer.hide();
    }
    for (const p of this.players.values()) {
      p.showAnimInfo(on);
    }
  }

  /** animInfoOn: giữ trạng thái để nhân vật MỚI xuất hiện cũng bật sẵn, không phải bấm lại. */
  private animInfoOn = false;
  /** leaving[id] = lúc entity bắt đầu mờ ra. Xem vòng dọn ở cuối `drawPlayers`. */
  private readonly leaving = new Map<number, number>();
  /** pool: view đã rời khung hình, giữ nguyên texture để dùng lại (§5 "View Pool"). */
  private readonly pool = new ViewPool();
  /** animViewer: khung xem từng animation của chế độ `J` (phím `[` / `]`). */
  readonly animViewer = new AnimViewer();

  /** stepAnimViewer đi tới / lùi một animation trong bảng điểm. */
  stepAnimViewer(d: number): void {
    this.animViewer.step(d);
  }

  drawStats(): { charsInView: number; swordsDrawn: number; placeholder: boolean } {
    let swordsDrawn = 0;
    let placeholder = false;
    for (const p of this.players.values()) {
      swordsDrawn += p.swordCount;
      if (!p.hasAtlas()) {
        placeholder = true;
      }
    }
    return { charsInView: this.players.size, swordsDrawn, placeholder };
  }

  /** perfStats là cùng bộ số của overlay F3 nhưng ở dạng object (tools/browsertest đọc). */
  perfStats(): Record<string, number> {
    let swords = 0;
    for (const p of this.players.values()) {
      swords += p.swordCount;
    }
    const me = this.world?.me();
    return {
      myLevel: me?.L ?? -1,
      mySwords: me?.n ?? 0,
      fps: this.fps,
      frameMs: 1000 / Math.max(this.fps, 1),
      drawMs: this.lastRenderMs,
      sprites: countNodes(this.app.stage),
      drawCalls: drawCalls.perFrame,
      fixPerSec: this.world?.fixPerSec ?? 0,
      fixMax: this.world?.fixMax ?? 0,
      players: this.players.size,
      swords,
      monsters: this.monsters.size,
      orbs: this.orbs.ids().length,
      items: this.items.size,
      aoi: this.players.size + this.monsters.size + this.items.size + this.orbs.ids().length,
      detail: this.detail,
    };
  }

  perfLine(dt: number, net: { rttMs: number; kbPerSec: number; snapBytes: number; srvMs: number }): string {
    // Đo FPS bằng ĐỒNG HỒ THẬT, không dùng `dt` của ticker: Pixi kẹp `deltaMS` ở minFPS=10,
    // nên một máy chạy 3 khung/giây vẫn báo đúng 10.0. Bài test 10 tab headless từng cho ra
    // cột FPS toàn 10.0 / 10.0 / 10.0 vì đúng lỗi này.
    const t = performance.now() / 1000;
    this.fpsAcc += this.fpsLast > 0 ? t - this.fpsLast : dt;
    this.fpsLast = t;
    this.fpsFrames += 1;
    if (this.fpsAcc >= 0.5) {
      this.fps = this.fpsFrames / this.fpsAcc;
      this.fpsAcc = 0;
      this.fpsFrames = 0;
    }
    let swords = 0;
    for (const p of this.players.values()) {
      swords += p.swordCount;
    }
    const aoi = this.players.size + this.monsters.size + this.items.size + this.orbs.ids().length;
    return [
      `${this.fps.toFixed(0)} fps · ${(1000 / Math.max(this.fps, 1)).toFixed(1)} ms/khung · vẽ ${this.lastRenderMs.toFixed(1)} ms`,
      `server ${net.srvMs.toFixed(2)} ms/tick · ping ${net.rttMs.toFixed(0)} ms`,
      `mạng ${net.kbPerSec.toFixed(1)} KB/s · snapshot ${(net.snapBytes / 1024).toFixed(1)} KB`,
      `sprite ${countNodes(this.app.stage)} · draw call ${drawCalls.perFrame.toFixed(0)}`,
      `AOI ${aoi} · người ${this.players.size} · kiếm ${swords} · quái ${this.monsters.size}`,
      `EXP ${this.orbs.ids().length} · vật phẩm ${this.items.size} · giãn kiếm 1/${this.detail}`,
      // Spec §8.2a: số lần server kéo nhân vật về chỗ khác. Dự đoán đúng thì ~0; lệch thì
      // nó chạy lên bằng tần số snapshot và người chơi thấy rung khi đi sát vật cản.
      `hiệu chỉnh ${(this.world?.fixPerSec ?? 0).toFixed(1)}/giây · lệch nhất ${(this.world?.fixMax ?? 0).toFixed(1)}`,
    ].join('\n');
  }

  /** beastCount: số thần thú đang chạy, theo kiểu — dùng để kiểm tra bằng console debug. */
  beastCount(): Record<string, number> {
    const n: Record<string, number> = { breakthrough: 0, ambient: 0, shatter: 0 };
    for (const e of this.beasts) {
      n[e.b.mode] += 1;
    }
    return n;
  }

  /**
   * updateBeasts chạy toàn bộ thần thú Tiên giới mỗi khung hình: hẹn giờ bản thường trực,
   * đặt lại vị trí theo chủ, và xếp hai lớp ôm lấy nhân vật đúng thứ tự chiều sâu.
   */
  private updateBeasts(frames: ReturnType<ClientWorld['playerFrames']>, now: number): void {
    const byId = new Map(frames.map((f) => [f.view.id, f]));
    this.lastTien.clear();
    const me = frames.find((f) => f.view.id === this.world?.myId);
    for (const f of frames) {
      const idx = tienIndex(this.cfg, realmOf(this.cfg, f.s.L));
      this.lastTien.set(f.view.id, idx);
      if (idx < 0) {
        continue;
      }
      // Mỗi người Tiên giới luôn có thần thú đi kèm; Đạo Tổ có hai con lệch pha 180°.
      const want = TIEN_BEASTS[idx]?.twin ? 2 : 1;
      const have = this.beasts.filter((e) => e.owner === f.view.id && e.b.mode === 'companion').length;
      for (let k = have; k < want; k++) {
        this.spawnBeast(f.view.id, 'companion', now, idx, k * Math.PI);
      }
    }
    // Vệt hạt chỉ cho vài con gần mình nhất (spec §3.5): xếp theo khoảng cách rồi bật dần.
    const dist = (id: number): number => {
      const f = byId.get(id);
      return f && me ? Math.hypot(f.x - me.x, f.y - me.y) : Infinity;
    };
    const order = [...this.beasts].sort((a, b) => dist(a.owner) - dist(b.owner));
    order.forEach((e, i) => {
      e.b.trailOn = i < BEAST_TRAIL_NEAR;
    });

    this.beasts = this.beasts.filter((e) => {
      const f = byId.get(e.owner);
      const stillTien = f ? tienIndex(this.cfg, realmOf(this.cfg, f.s.L)) >= 0 : false;
      // Bản companion sống tới khi chủ rời Tiên giới hoặc ra khỏi tầm nhìn.
      if (!f || e.b.done(now) || (e.b.mode === 'companion' && !stillTien)) {
        this.removeBeast(e.b);
        return false;
      }
      const r = this.cfg.player.body_r * f.s.sc;
      const feet = f.y + r * 0.75; // chân nhân vật, khớp FEET_BELOW_CENTER của PlayerSprite
      e.b.resize(r, r * 2.6);
      // Ẩn Thân / trong bụi: mờ theo chủ (spec §3.5), không ẩn hẳn.
      const hidden = (f.s.fl & FLAG.stealth) !== 0 || (f.s.fl & FLAG.bush) !== 0;
      e.b.ownerAlpha = hidden ? 0.2 : 1;
      e.b.burning = (f.s.fl & FLAG.burning) !== 0;
      e.b.update(f.x, feet, now, r * 2.6 * this.zoom >= BEAST_FAR_PX);
      // Lớp sau nằm ngay dưới nhân vật, lớp trước ngay trên — nhân vật zIndex = y + r×0.75.
      e.b.back.zIndex = feet - 0.5;
      e.b.front.zIndex = feet + 0.5;
      return true;
    });
  }

  /** flashBeasts cho thần thú của một người lóe sáng + lượn nhanh (spec §3.5 v4.7). */
  private flashBeasts(owner: number, now: number): void {
    for (const e of this.beasts) {
      if (e.owner === owner && e.b.mode === 'companion') {
        e.b.flash(now);
      }
    }
  }

  /** spawnBeast tạo thần thú cho một người, tôn trọng trần 6 con của spec §3.5. */
  private spawnBeast(owner: number, mode: BeastMode, now: number, realmIdx?: number, phase = 0): void {
    const f = this.lastFrames.find((p) => p.view.id === owner);
    if (!f) {
      return;
    }
    const idx = realmIdx ?? tienIndex(this.cfg, realmOf(this.cfg, f.s.L));
    const spec = TIEN_BEASTS[idx];
    if (!spec) {
      return;
    }
    const r = this.cfg.player.body_r * f.s.sc;
    const b = new Beast(spec, mode, r, r * 2.6, now, phase);
    this.entities.addChild(b.back, b.front);
    this.beasts.push({ b, owner });
  }

  private removeBeast(b: Beast): void {
    this.entities.removeChild(b.back, b.front);
    b.destroy();
  }

  private handleEvent(e: GameEvent, now: number): void {
    const mine = e.a === this.world?.myId;
    const size = 90 * this.playerScale(e.a);
    switch (e.k) {
      case 'RingFlip':
        // Both rings emit RingFlip at the same contact point; show one spark.
        if (now - this.lastSpark.t > 0.05 || Math.hypot(e.x - this.lastSpark.x, e.y - this.lastSpark.y) > 1) {
          this.lastSpark = { t: now, x: e.x, y: e.y };
          this.vfx.play('clash_spark', e.x, e.y, size * 1.2, now);
          sound.play('sword_clash', mine ? 1 : 0.5);
        }
        break;
      case 'SwordBreak': {
        // Server gộp số kiếm gãy lại rồi mới báo (xem flushSwordBreaks), nên một sự kiện có
        // thể là cả chục cây — vẽ một vụ nổ to theo số lượng thay vì chục vụ nổ nhỏ.
        const many = Math.min(3, 1 + Math.log2(Math.max(1, e.n ?? 1)) / 2);
        this.vfx.play('sword_shatter', e.x, e.y, size * 1.5 * many, now);
        sound.play('sword_break', mine ? 1 : 0.5);
        break;
      }
      case 'Hit':
        if (e.s === 'shield') {
          this.flashBeasts(e.b ?? 0, now); // khiên đỡ được một đòn → thần thú lóe sáng
        }
        this.vfx.play(e.s === 'shield' ? 'shield_break' : 'stun_stars', e.x, e.y - size * 0.6, size * 1.6, now);
        sound.play(e.s === 'shield' ? 'shield_break' : 'sword_hit_body', e.b === this.world?.myId || mine ? 1 : 0.5);
        if (e.b === this.world?.myId) {
          this.vfx.shake(18 * this.playerScale(e.b), 0.25, now);
        }
        break;
      case 'Death':
        this.vfx.play('death_smoke', e.x, e.y, size * 2, now);
        sound.play('death');
        break;
      case 'LevelUp':
        this.vfx.play('levelup_aura', e.x, e.y, size * 2, now);
        if (mine) {
          sound.play('levelup');
        }
        break;
      case 'Breakthrough':
        this.onBreakthrough(e, size, mine, now);
        break;
      case 'Blinded':
        // Server chỉ gửi event này cho ĐÚNG người bị trúng (spec §6.3), nên không cần lọc
        // thêm; thời gian lấy thẳng từ server để client không tự đoán.
        // Thời gian lấy THẲNG từ server; chỉ khi gói thiếu mới rơi về số trong config.
        // Tự đoán ở đây là client hết trân châu trong khi server vẫn tính là đang bị bịt
        // mắt — người chơi nhìn rõ nhưng vẫn không nhặt được gì, không hiểu vì sao.
        this.boba.hit(e.f ?? this.cfg.skills.tran_chau_secs, now, this.app.screen.width, this.app.screen.height);
        sound.play('sword_hit_body');
        break;
      case 'SkillCast':
        // Bong bóng thoại tấu hài khi dùng kỹ năng (spec §6.3).
        if (e.s && STR.skillLines[e.s]) {
          this.players.get(e.a ?? 0)?.showBubble(STR.skillLines[e.s], now);
        }
        break;
      case 'RealmDrop':
        {
          const was = this.lastTien.get(e.a ?? 0) ?? -1;
          if (was >= 0) {
            this.spawnBeast(e.a ?? 0, 'shatter', now, was); // thần thú gãy vụn (spec §3.5)
          }
        }
        if (mine) {
          sound.play('realm_drop');
        }
        break;
      default:
        this.handleWorldEvent(e, now, mine, size);
    }
  }

  private onBreakthrough(e: GameEvent, size: number, mine: boolean, now: number): void {
    if (e.s === 'tien') {
      // Phi thăng Tiên giới: thần thú to chui lên từ chân, ai trong tầm nhìn đều thấy (§3.5).
      // Lấy cảnh giới từ e.n (cấp MỚI) chứ không từ khung hình đã lưu — khung hình còn là
      // cảnh giới cũ nên sẽ gọi nhầm thần thú của cảnh giới vừa rời đi.
      const realm = realmOf(this.cfg, e.n ?? 0);
      this.spawnBeast(e.a ?? 0, 'breakthrough', now, tienIndex(this.cfg, realm));
      this.vfx.shake(22 * this.playerScale(e.a), 0.8, now);
      if (mine) {
        // Khựng hình 80ms + sóng xung kích + chớp trắng + chữ cảnh giới (spec §3.6 mục 5).
        this.hitstopUntil = now + HITSTOP_SECS;
        this.breakthrough.play(realmName(this.cfg, realm), { x: 0.5, y: 0.5 }, this.camera, now);
      }
    }
    this.vfx.play('lightning', e.x, e.y - size, size * 3, now);
    this.vfx.play('levelup_aura', e.x, e.y, size * 3, now);
    sound.play('breakthrough', mine ? 1 : 0.6);
    if (mine) {
      this.vfx.shake(30 * this.playerScale(e.a), 0.6, now);
    }
    this.players.get(e.a ?? 0)?.showBubble(pick(STR.emotes.breakthrough), now);
  }

  private handleWorldEvent(e: GameEvent, now: number, mine: boolean, size: number): void {
    switch (e.k) {
      case 'ItemPicked':
        // Viên Linh khí / Tiên khí được server GỘP: một sự kiện mang `n` viên, `b` là viên
        // cuối. Quét qua bãi viên dày có thể nhặt mấy trăm viên trong một gói — nếu vẫn một
        // sự kiện một viên thì gói snapshot phồng lên 21 KB và client kêu 400 tiếng chồng
        // lên nhau. Giờ vẽ một vệt bay hút và kêu đúng một tiếng.
        if (e.b !== undefined) {
          this.pickedBy.set(e.b, e.a ?? 0); // syncItems biến nó thành hiệu ứng bay hút
        }
        if (mine) {
          const s = e.s ?? '';
          sound.play(s === 'orb' ? 'pickup_exp' : s === 'sword' ? 'pickup_sword' : s === 'chest' ? 'chest_open' : 'pickup_buff');
          if (s === 'chest') {
            this.vfx.play('levelup_aura', e.x, e.y, size * 2, now);
          }
        }
        break;
      case 'TienKhiPicked':
        // Nhặt Tiên khí: thần thú mừng, lóe sáng và lượn nhanh một vòng (spec §3.5 v4.7).
        this.flashBeasts(e.a ?? 0, now);
        if (mine) {
          sound.play('pickup_exp');
        }
        break;
      case 'BuffEnd':
        if (mine) {
          sound.play('buff_end');
        }
        break;
      case 'MonsterHurt':
        sound.play('monster_hurt', 0.4);
        break;
      case 'MonsterDie':
        this.vfx.play('death_smoke', e.x, e.y, 120, now);
        sound.play('monster_pop', 0.6);
        break;
      case 'ObstacleDamaged':
        sound.play('obstacle_hit', 0.4);
        break;
      case 'ObstacleBroken':
        this.vfx.play('death_smoke', e.x, e.y, 260, now);
        sound.play('obstacle_break', 0.7);
        break;
      case 'ObstacleRegrow':
        if (e.s === 'warn') {
          this.mapView?.warnObstacle(e.a ?? -1, now);
        } else {
          sound.play('obstacle_regrow', 0.6);
        }
        break;
      default:
        this.handleGateEvent(e, now, mine);
    }
  }

  private handleGateEvent(e: GameEvent, now: number, mine: boolean): void {
    switch (e.k) {
      case 'TeleportOut':
        this.vfx.play('levelup_aura', e.x, e.y, 300, now);
        sound.play('gate_in', mine ? 1 : 0.5);
        break;
      case 'GateWarn':
        this.mapView?.warnGate(e.n ?? -1, now);
        sound.play('gate_warn', 0.5);
        break;
      case 'TeleportIn':
        sound.play('gate_out', mine ? 1 : 0.5);
        if (mine) {
          this.players.get(e.a ?? 0)?.showBubble(pick(STR.emotes.teleport_in), now);
        }
        break;
      case 'Emote':
        this.players.get(e.a ?? 0)?.showBubble(pick(STR.emotes[e.s ?? ''] ?? ['!']), now);
        break;
      case 'ChestIncoming':
        sound.play('chest_incoming');
        break;
      default:
    }
  }

  /** realm of the local player for HUD. */
  myRealm(): number {
    const me = this.world?.me();
    return me ? realmOf(this.cfg, me.L) : 0;
  }

  get currentZoom(): number {
    return this.zoom;
  }
}
