import { type GameConfig } from '../config';
import { lerp, type Vec } from './geom';
import { moveStep } from './movestep.ts';
import {
  BUFFS,
  type GameEvent,
  type GameMap,
  type ItemState,
  itemKind,
  type MonsterState,
  type PlayerState,
  type Snapshot,
  statusOn,
  type Welcome,
} from './protocol';

interface Timed<T> {
  t: number;
  s: T;
}

export interface PlayerView {
  id: number;
  name: string;
  char: string;
  snaps: Timed<PlayerState>[];
}

interface PendingInput {
  seq: number;
  angle: number;
  moving: boolean;
  burn: boolean;
}

/** What the renderer needs for one player this frame. */
export interface PlayerFrame {
  view: PlayerView;
  s: PlayerState;
  x: number;
  y: number;
  ringAngle: number;
}

/** Lệch dưới ngần này là sai số dấu phẩy động, không phải hiệu chỉnh thật. */
const CORRECTION_EPS = 0.5;

const SNAP_KEEP = 30;
/**
 * Giữ entity thêm ngần này giây sau khi vắng mặt khỏi snapshot (NETCODE_SPEC §4.3).
 *
 * 2 giây là con số spec chọn, và nó phải LỚN HƠN HẲN nhịp 10 Hz của người ở xa (0,1 giây)
 * — nếu không thì vẫn nhấp nháy, chỉ chậm hơn.
 */
const KEEP_GONE_SECS = 2;
const OFFSET_DECAY = 0.0005; // lets the clock estimate adapt if latency grows

/** ClientWorld holds the replicated world: interpolation for others, prediction for me. */
export class ClientWorld {
  myId = 0;
  map: GameMap | null = null;
  readonly players = new Map<number, PlayerView>();
  readonly monsters = new Map<number, Timed<MonsterState>[]>();
  readonly items = new Map<number, ItemState>();
  readonly brokenObstacles = new Map<number, boolean>();
  readonly obstacleHp = new Map<number, number>();
  /** goneAt: lần đầu một entity vắng mặt khỏi snapshot — xem `expire`. */
  private readonly goneAt = new Map<number, number>();
  /** deletedAt: lúc một entity bị xoá hẳn, để nhận ra nó quay lại ngay sau đó (nhấp nháy). */
  private readonly deletedAt = new Map<number, number>();
  /** respawn2s: số lần nhấp nháy chưa báo cáo — xem takeRespawn2s. */
  private respawn2s = 0;
  /** missingDesc: entity chưa biết vẽ nhân vật nào — xem takeMissingDesc. */
  private readonly missingDesc = new Set<number>();
  private offset: number | null = null;
  private pending: PendingInput[] = [];
  private seq = 0;
  private predicted: Vec = { x: 0, y: 0 };
  private correction: Vec = { x: 0, y: 0 };
  /** Vận tốc dự đoán — chỉ khác 0 khi đang trượt trên băng (spec §8.0). */
  private predictedVel: Vec = { x: 0, y: 0 };
  /** Đo "giật khi chạm vật cản" (spec §8.2a): số lần và độ lệch khi server sửa vị trí. */
  fixPerSec = 0;
  fixMax = 0;
  private fixCount = 0;
  private fixWorst = 0;
  private fixAt = 0;
  /** Số cú nhảy thẳng (dịch chuyển, hồi sinh) — tách khỏi hiệu chỉnh để không làm sai số đo. */
  teleports = 0;
  private lastSendAt = 0;
  private lastInput: PendingInput = { seq: 0, angle: 0, moving: false, burn: false };
  lastSnapAt = 0;

  constructor(private readonly cfg: GameConfig) {}

  reset(w: Welcome, map: GameMap): void {
    this.myId = w.id;
    this.map = map;
    this.players.clear();
    this.monsters.clear();
    this.goneAt.clear();
    this.deletedAt.clear();
    this.items.clear();
    this.brokenObstacles.clear();
    this.obstacleHp.clear();
    for (const o of w.obs ?? []) {
      this.obstacleHp.set(o.id, o.hp);
      this.brokenObstacles.set(o.id, o.b);
    }
    this.pending = [];
    this.offset = null;
  }

  /** Estimated current server time in seconds. */
  serverNow(): number {
    return performance.now() / 1000 + (this.offset ?? 0);
  }

  renderTime(): number {
    return this.serverNow() - this.cfg.net.interp_ms / 1000;
  }

  applySnapshot(s: Snapshot): GameEvent[] {
    const est = s.tm - performance.now() / 1000;
    this.offset = this.offset === null || est > this.offset ? est : this.offset - OFFSET_DECAY;
    this.lastSnapAt = performance.now();
    this.updatePlayers(s);
    this.updateMonsters(s);
    // Đổi chỉ số loại → tên ngay tại đây, một lần duy nhất. Phần còn lại của client vẫn
    // so `k === 'orb'` như cũ (xem ghi chú ở `ItemState`).
    for (const it of s.ia ?? []) {
      this.items.set(it.id, { id: it.id, x: it.x, y: it.y, k: itemKind(it.k), v: it.v });
    }
    for (const id of s.ir ?? []) {
      this.items.delete(id);
    }
    const events = s.ev ?? [];
    this.applyObstacleEvents(events);
    this.reconcile(s.ack);
    return events;
  }

  private updatePlayers(s: Snapshot): void {
    const seen = new Set<number>();
    for (const p of s.p ?? []) {
      seen.add(p.id);
      let v = this.players.get(p.id);
      if (!v) {
        v = { id: p.id, name: p.nm ?? '?', char: p.ch ?? '', snaps: [] };
        this.players.set(p.id, v);
      }
      if (p.nm) {
        v.name = p.nm;
        v.char = p.ch ?? v.char;
      }
      // Chưa biết nhân vật nào = chưa vẽ được. Ghi lại để HỎI server thay vì vẽ placeholder
      // (NETCODE_SPEC §5 Hiển thị mục 2). Server chỉ gửi tên+nhân vật LẦN ĐẦU và nhớ là đã
      // gửi; nếu client mất mô tả đó thì không hỏi là kẹt vĩnh viễn.
      if (v.char === '') {
        this.missingDesc.add(p.id);
      } else {
        this.missingDesc.delete(p.id);
      }
      v.snaps.push({ t: s.tm, s: p });
      if (v.snaps.length > SNAP_KEEP) {
        v.snaps.shift();
      }
    }
    this.expire(this.players, seen, s.tm);
  }

  /**
   * expire xoá entity **sau khi vắng mặt liên tục `KEEP_GONE_SECS`**, không xoá ngay khi
   * vắng một gói (NETCODE_SPEC §4.3, §4.4).
   *
   * Đây là gốc của lỗi nhấp nháy. Server CỐ Ý bỏ qua người/quái ở xa trên các tick lẻ để
   * họ chỉ tốn 10 Hz thay vì 20 Hz (tối ưu mục 27) — nhưng client lại hiểu "vắng mặt một
   * gói" là "đã rời đi", xoá khỏi bảng, và `scene.ts` huỷ luôn sprite. Kết quả: **mọi
   * entity xa hơn nửa khung nhìn bị huỷ và dựng lại 10 lần mỗi giây.**
   *
   * Nó còn kéo theo lỗi placeholder: tên và nhân vật (`nm`, `ch`) chỉ gửi lần đầu client
   * thấy một người, server nhớ bằng `knownPlayers` và không hề biết client đã tự vứt view.
   * View dựng lại không có tên nhân vật → không tra được atlas → vẽ vòng tròn placeholder
   * cho tới nhịp gửi lại tên (2 giây).
   *
   * Đo được trước khi sửa: **8/8 lần biến mất rồi quay lại trong 2 giây**, 1/3 nhân vật
   * đang vẽ placeholder (`tools/browsertest/flicker.mjs`). Ngân sách NETCODE §1: cả hai = 0.
   */
  private expire<T>(m: Map<number, T>, seen: Set<number>, now: number): void {
    for (const id of m.keys()) {
      if (seen.has(id)) {
        // Quay lại sau khi đã bị xoá hẳn = một lần nhấp nháy người chơi NHÌN THẤY.
        // Đếm ở đây để telemetry báo được con số đó từ máy thật (TELEMETRY_SPEC §2.2
        // nhóm "Đồng bộ", ngân sách §4.3 là 0) — trước giờ chỉ đo được bằng flicker.mjs
        // trên máy dev, nơi lỗi này gần như không bao giờ xảy ra.
        const left = this.deletedAt.get(id);
        if (left !== undefined) {
          if (now - left < KEEP_GONE_SECS) {
            this.respawn2s++;
          }
          this.deletedAt.delete(id);
        }
        this.goneAt.delete(id);
        continue;
      }
      const since = this.goneAt.get(id);
      if (since === undefined) {
        this.goneAt.set(id, now);
      } else if (now - since >= KEEP_GONE_SECS) {
        m.delete(id);
        this.goneAt.delete(id);
        this.deletedAt.set(id, now);
      }
    }
    // Dọn `deletedAt`: entity rời đi quá `KEEP_GONE_SECS` mà chưa quay lại thì có quay lại
    // cũng không còn tính là nhấp nháy nữa. Không dọn thì map này phình mãi — một trận dài
    // đi qua hàng nghìn entity, và đó là rò rỉ bộ nhớ thật chứ không phải lý thuyết.
    for (const [id, at] of this.deletedAt) {
      if (now - at >= KEEP_GONE_SECS) {
        this.deletedAt.delete(id);
      }
    }
  }

  /**
   * takeMissingDesc trả về danh sách entity chưa có mô tả, để gửi `DescRequest`.
   *
   * KHÔNG xoá khỏi tập: chỉ khi mô tả thật sự về tới (nhánh trên) mới xoá. Xoá ngay lúc hỏi
   * thì gói `dreq` rơi mất là không bao giờ hỏi lại nữa — đúng cái bẫy mà tính năng này sinh
   * ra để tránh. Người gọi tự giới hạn nhịp hỏi.
   */
  takeMissingDesc(): number[] {
    return [...this.missingDesc];
  }

  /**
   * takeRespawn2s trả về số lần nhấp nháy kể từ lần gọi trước, rồi đặt lại bộ đếm.
   *
   * "Lấy rồi xoá" chứ không phải "đọc": telemetry gộp theo cửa sổ 10 giây, nên mỗi lần
   * nhấp nháy phải được đếm đúng MỘT lần vào đúng một cửa sổ.
   */
  takeRespawn2s(): number {
    const n = this.respawn2s;
    this.respawn2s = 0;
    return n;
  }

  private updateMonsters(s: Snapshot): void {
    const seen = new Set<number>();
    for (const m of s.m ?? []) {
      seen.add(m.id);
      const buf = this.monsters.get(m.id) ?? [];
      // Loại, máu tối đa và bán kính chỉ được gửi LẦN ĐẦU (spec mục 27) — chúng không bao
      // giờ đổi. Các gói sau thiếu ba trường đó, phải lấy lại từ gói trước.
      const prev = buf[buf.length - 1]?.s;
      // `hp` cũng chỉ gửi khi ĐỔI (delta snapshot, spec mục 27) — quái đứng yên gặm cỏ thì
      // gói nào cũng thiếu trường đó, phải giữ lại máu cũ chứ không được coi là 0.
      const full: MonsterState = prev
        ? { ...m, k: m.k || prev.k, mx: m.mx || prev.mx, r: m.r || prev.r, hp: m.hp ?? prev.hp }
        : m;
      buf.push({ t: s.tm, s: full });
      if (buf.length > SNAP_KEEP) {
        buf.shift();
      }
      this.monsters.set(m.id, buf);
    }
    this.expire(this.monsters, seen, s.tm);
  }

  private applyObstacleEvents(events: GameEvent[]): void {
    for (const e of events) {
      const id = e.a ?? -1;
      if (e.k === 'ObstacleDamaged') {
        this.obstacleHp.set(id, e.n ?? 0);
      } else if (e.k === 'ObstacleBroken') {
        this.brokenObstacles.set(id, true);
        this.obstacleHp.set(id, 0);
      } else if (e.k === 'ObstacleRegrow' && e.s !== 'warn') {
        this.brokenObstacles.set(id, false);
        this.obstacleHp.delete(id);
      }
    }
  }

  me(): PlayerState | null {
    const v = this.players.get(this.myId);
    return v ? v.snaps[v.snaps.length - 1].s : null;
  }

  // ---- input & prediction ------------------------------------------------

  /** Records and returns the input message to send this network tick. */
  makeInput(angle: number, moving: boolean, burn = false, skill = false): Record<string, unknown> {
    const inp = { seq: ++this.seq, angle: quantize(angle), moving, burn };
    this.pending.push(inp);
    const me = this.me();
    if (me && !(me.fl & 16)) {
      const r = this.step(this.predicted, this.predictedVel, inp, me, 1 / this.cfg.net.tick_hz);
      this.predicted = r.pos;
      this.predictedVel = r.vel;
    }
    this.lastInput = inp;
    this.lastSendAt = performance.now();
    const msg: Record<string, unknown> = { t: 'in', s: inp.seq, a: Math.round((inp.angle / (2 * Math.PI)) * 256) & 255, m: moving, b: burn };
    if (skill) {
      // Chỉ gửi khi thật sự bấm: đây là sự kiện một lần, không phải trạng thái giữ phím.
      msg.sk = true;
    }
    return msg;
  }

  private reconcile(ack: number): void {
    const me = this.me();
    if (!me) {
      return;
    }
    this.pending = this.pending.filter((p) => p.seq > ack);
    let pos: Vec = { x: me.x, y: me.y };
    // Phát lại từ đúng VẬN TỐC server đang giữ, không phải từ 0: trên mặt băng vận tốc là
    // trạng thái, bỏ nó đi là mỗi snapshot lại dự đoán sai và nhân vật giật cả mặt hồ.
    let vel: Vec = { x: me.vx ?? 0, y: me.vy ?? 0 };
    for (const inp of this.pending) {
      const r = this.step(pos, vel, inp, me, 1 / this.cfg.net.tick_hz);
      pos = r.pos;
      vel = r.vel;
    }
    // Đếm số lần server kéo nhân vật về chỗ khác (spec §8.2a): đây là thước đo trực tiếp
    // của cảm giác "giật khi đi sát vật cản". Dự đoán đúng thì con số này ~0; lệch thì nó
    // chạy lên bằng đúng tần số snapshot (20/giây) và người chơi thấy rung liên tục.
    const off = Math.hypot(this.predicted.x - pos.x, this.predicted.y - pos.y);
    // Bỏ qua cú nhảy lớn: đó là dịch chuyển qua cổng hoặc hồi sinh, không phải lỗi dự đoán.
    // Không lọc thì một lần qua cổng làm "lệch nhất" nhảy lên 12 000 và che hết số thật.
    if (off > CORRECTION_EPS && off <= this.cfg.net.correction_snap) {
      this.fixCount++;
      this.fixWorst = Math.max(this.fixWorst, off);
    }
    const nowMs = performance.now();
    if (nowMs - this.fixAt >= 1000) {
      this.fixPerSec = (this.fixCount * 1000) / (nowMs - this.fixAt);
      this.fixMax = this.fixWorst;
      this.fixCount = 0;
      this.fixWorst = 0;
      this.fixAt = nowMs;
    }
    // Keep the displayed position continuous; the correction fades out in renderFrame.
    this.correction.x += this.predicted.x - pos.x;
    this.correction.y += this.predicted.y - pos.y;
    // Spec §8.2a: lệch nhỏ thì kéo dần về trong `correction_tau_ms`; lệch lớn (dịch chuyển
    // qua cổng, hồi sinh) thì nhảy thẳng — kéo dần một cú nhảy nửa bản đồ chỉ làm nhân vật
    // trôi dài qua màn hình. Ngưỡng cũ 400 đơn vị quá rộng: một cú lệch 200 đơn vị vẫn bị
    // kéo dần và người chơi thấy nhân vật "bơi" về chỗ cũ.
    if (Math.hypot(this.correction.x, this.correction.y) > this.cfg.net.correction_snap) {
      this.correction = { x: 0, y: 0 };
      this.teleports++;
    }
    this.predicted = pos;
    this.predictedVel = vel;
  }

  /** One movement step, mirroring game.movePlayer on the server. */
  private step(from: Vec, vel: Vec, inp: PendingInput, me: PlayerState, dt: number): { pos: Vec; vel: Vec } {
    const map = this.map;
    if (!map) {
      return { pos: from, vel };
    }
    // fl & 8 = đang dịch chuyển qua cổng: không điều khiển được, cũng không trôi.
    if (me.fl & 8) {
      return { pos: from, vel: { x: 0, y: 0 } };
    }
    // Trói / rụt mai / choáng: server đặt tốc độ 0, nên client cũng phải đứng yên — dự đoán
    // là mình đang chạy trong lúc server giữ tại chỗ thì mỗi tick bị kéo về một lần.
    const frozen = statusOn(me.st, 'root') || statusOn(me.st, 'shell') || statusOn(me.st, 'stun');
    let mult = 1;
    if (me.bf & (1 << BUFFS.indexOf('haste'))) {
      mult *= this.cfg.buffs.haste.mult;
    }
    // Thiêu Đốt Tinh Huyết (spec §4.1c). Server dừng cộng tốc khi EXP chạm đáy cảnh giới
    // và sẽ sửa lại vị trí; ở đây chỉ cần dự đoán đúng trường hợp thường gặp.
    if (inp.burn && !frozen) {
      mult *= this.cfg.burn.mult;
    }
    if (statusOn(me.st, 'slow')) {
      mult *= this.cfg.skills.ru_ngu_move;
    }
    if (statusOn(me.st, 'ghost')) {
      mult *= this.cfg.skills.ghost_speed;
    }
    if (frozen) {
      mult = 0;
    }
    return moveStep(this.cfg, map, from, inp.angle, me.sc, dt,
      (id: number) => this.brokenObstacles.get(id) === true, mult, vel, inp.moving,
      statusOn(me.st, 'ghost'));
  }




  // ---- per-frame queries -------------------------------------------------

  /** Interpolated (others) or predicted (me) state for rendering. */
  playerFrames(dt: number): PlayerFrame[] {
    const decay = Math.exp((-dt * 1000) / Math.max(this.cfg.net.correction_tau_ms, 1));
    this.correction.x *= decay;
    this.correction.y *= decay;
    const rt = this.renderTime();
    const out: PlayerFrame[] = [];
    for (const v of this.players.values()) {
      out.push(v.id === this.myId ? this.myFrame(v) : interpPlayer(v, this.cfg, rt));
    }
    return out;
  }

  private myFrame(v: PlayerView): PlayerFrame {
    const last = v.snaps[v.snaps.length - 1];
    const s = last.s;
    // Advance within the current input step so motion is smooth at high frame rates.
    const since = Math.min((performance.now() - this.lastSendAt) / 1000, 1 / this.cfg.net.tick_hz);
    const ahead = this.step(this.predicted, this.predictedVel, this.lastInput, s, since).pos;
    const ringAngle = s.ph + s.d * omega(this.cfg, s) * (this.serverNow() - last.t);
    return { view: v, s, x: ahead.x + this.correction.x, y: ahead.y + this.correction.y, ringAngle };
  }

  monsterFrames(): { s: MonsterState; x: number; y: number }[] {
    const rt = this.renderTime();
    const out: { s: MonsterState; x: number; y: number }[] = [];
    for (const buf of this.monsters.values()) {
      const { a, b, t } = bracket(buf, rt);
      out.push({ s: b.s, x: lerp(a.s.x, b.s.x, t), y: lerp(a.s.y, b.s.y, t) });
    }
    return out;
  }
}

function quantize(angle: number): number {
  const step = (2 * Math.PI) / 256;
  const a = ((angle % (2 * Math.PI)) + 2 * Math.PI) % (2 * Math.PI);
  return Math.round(a / step) * step;
}

/** bracket finds the two snapshots around time t and the blend factor between them. */
function bracket<T>(buf: Timed<T>[], t: number): { a: Timed<T>; b: Timed<T>; t: number } {
  for (let i = buf.length - 1; i > 0; i--) {
    if (buf[i - 1].t <= t) {
      const a = buf[i - 1];
      const b = buf[i];
      const span = b.t - a.t;
      return { a, b, t: span > 0 ? Math.min(1, (t - a.t) / span) : 1 };
    }
  }
  return { a: buf[0], b: buf[0], t: 1 };
}

/**
 * omega: tốc độ quay vòng kiếm, rad/giây. Server KHÔNG gửi con số này nữa (spec mục 27) —
 * nó suy ra được từ `rev_secs` trong config và buff Tốc Kiếm mà client đã có sẵn, nên gửi
 * lại là lãng phí 7 byte mỗi người mỗi gói, 20 lần mỗi giây.
 */
export function omega(cfg: GameConfig, s: PlayerState): number {
  const spin = s.bf & (1 << BUFFS.indexOf('spin')) ? (cfg.buffs.spin?.mult ?? 1) : 1;
  return ((2 * Math.PI) / cfg.sword.rev_secs) * spin;
}

function interpPlayer(v: PlayerView, cfg: GameConfig, rt: number): PlayerFrame {
  const { a, b, t } = bracket(v.snaps, rt);
  // Large jumps (teleport, respawn) are not interpolated.
  const jump = Math.hypot(b.s.x - a.s.x, b.s.y - a.s.y) > 600;
  const k = jump ? 1 : t;
  const base = t >= 1 ? b : a;
  const ringAngle = base.s.ph + base.s.d * omega(cfg, base.s) * (rt - base.t);
  return { view: v, s: b.s, x: lerp(a.s.x, b.s.x, k), y: lerp(a.s.y, b.s.y, k), ringAngle };
}
