// Thu số liệu hiệu năng của NGƯỜI CHƠI THẬT (TELEMETRY_SPEC).
//
// Ba ràng buộc của §1, và cả ba đều là ràng buộc về việc KHÔNG được làm hỏng thứ đang đo:
//
//  1. Dưới 0.3 ms mỗi khung. Trong vòng vẽ chỉ ghi đúng một con số (thời gian khung); mọi
//     thứ đắt hơn — đếm sprite, quét nhân vật — chạy ở 4 Hz chứ không phải 60 Hz.
//  2. Dưới 1 KB/s. Gộp sẵn ở client rồi gửi MỘT gói mỗi 10 giây, không gửi từng khung.
//  3. Không chặn vòng vẽ. Gửi là `send` không đợi; mất kết nối thì xếp hàng trong RAM.
//
// Module này nằm sau cờ build `TELEMETRY=on` và được nạp bằng `import()` động, nên bản
// release (`TELEMETRY` không bật) KHÔNG đóng gói một dòng nào của nó.

import { collectSession, randomId, type SessionInfo } from './session';

/** SAMPLE_MS: nhịp gộp và gửi một mẫu (§2.2). */
const SAMPLE_MS = 10_000;
/** POLL_MS: nhịp đọc những số đắt tiền (đếm sprite, quét nhân vật trong khung). */
const POLL_MS = 250;
/** MAX_QUEUE: mất mạng thì giữ tối đa ngần này mẫu rồi gửi bù (§3). */
const MAX_QUEUE = 30;
/** MAX_EVENTS_PER_MIN: trần sự kiện (§2.3). */
const MAX_EVENTS_PER_MIN = 20;

interface Sample {
  s: string;
  ts: number;
  fa: number; f1: number; m50: number; m95: number; mmx: number; o33: number; o100: number;
  lt: number;
  dc: number; sp: number; tx: number; tu: number; ph: number;
  p50: number; p95: number; pmx: number; ki: number; ko: number; mp: number; l150: number; st: number;
  cor: number; rs: number; im: number;
  zn: number; rl: number; sws: number; rg: number; cv: number; tv: number; sd: number; vf: number;
  bn: boolean; zg: boolean;
}

interface Event {
  s: string;
  ts: number;
  k: string;
  d: string;
}

/** TelemetryContext là bối cảnh trận ở một thời điểm (§2.2 nhóm cuối). */
export interface TelemetryContext {
  zone: number;
  realm: number;
  swords: number;
  rings: number;
  charsInView: number;
  tienInView: number;
  swordsDrawn: number;
  vfx: number;
  burning: boolean;
  zoning: boolean;
  placeholderNow: boolean;
  respawn2s: number;
  quality: string;
  qualityAuto: boolean;
}

/** Host là những thứ telemetry ĐỌC từ game. Chỉ đọc — không gọi hàm nào làm đổi trạng thái. */
export interface Host {
  /** perfStats: bộ số sẵn có của overlay F3 (scene.perfStats). */
  perfStats(): Record<string, number>;
  /** net: số liệu kết nối hiện tại. */
  net(): { rttMs: number; kbPerSec: number; snapBytes: number };
  /** context: bối cảnh trận (§2.2 nhóm cuối). */
  context(): TelemetryContext;
  /** send đẩy một gói qua WebSocket sẵn có. Trả false khi chưa nối được. */
  send(msg: Record<string, unknown>): boolean;
  /** canvasSize cho thông tin phiên. */
  canvasSize(): { w: number; h: number };
}

function pct(v: number[], p: number): number {
  if (v.length === 0) {
    return 0;
  }
  const s = [...v].sort((a, b) => a - b);
  return s[Math.min(s.length - 1, Math.floor(s.length * p))];
}

export class Telemetry {
  private readonly id = randomId();
  private readonly startedAt = performance.now();
  private session: SessionInfo | null = null;
  private sessionSent = false;

  /** frames: thời gian từng khung trong cửa sổ 10 giây hiện tại. */
  private frames: number[] = [];
  private pings: number[] = [];
  private kbps: number[] = [];
  private draws: number[] = [];
  private sprites: number[] = [];
  private maxPkt = 0;
  private longTasks = 0;
  private placeholderFrames = 0;
  private respawn2s = 0;
  private corrections: number[] = [];
  private polls = 0;

  private lastFrameAt = 0;
  private lastPollAt = 0;
  private windowStart = 0;
  private nSamples = 0;
  private fpsHistory: number[] = [];

  private queue: Sample[] = [];
  private events: Event[] = [];
  private eventWindow = 0;
  private eventCount = 0;

  private observer: PerformanceObserver | null = null;
  private stopped = false;

  constructor(private readonly host: Host, private readonly build: string, private readonly proto: number) {}

  start(): void {
    const now = performance.now();
    this.windowStart = now;
    this.lastFrameAt = now;
    this.lastPollAt = now;
    const ctx = this.host.context();
    this.session = collectSession({
      id: this.id, build: this.build, proto: this.proto, schema: '',
      quality: ctx.quality, qualityAuto: ctx.qualityAuto, canvas: this.host.canvasSize(),
    });
    this.watchLongTasks();
    this.watchPageEvents();
  }

  /**
   * frame được gọi MỖI KHUNG. Phải rẻ: chỉ một phép trừ và một lần push.
   *
   * Mọi thứ đắt hơn dồn vào `poll` ở 4 Hz. Đếm sprite trên cây hiển thị là O(số nút) — chạy
   * nó 60 lần/giây thì chính telemetry làm tụt FPS, và con số đo được sẽ là con số của một
   * cái game đang bị telemetry làm chậm.
   */
  frame(): void {
    if (this.stopped) {
      return;
    }
    const now = performance.now();
    const ms = now - this.lastFrameAt;
    this.lastFrameAt = now;
    this.frames.push(ms);
    if (now - this.lastPollAt >= POLL_MS) {
      this.lastPollAt = now;
      this.poll();
    }
    if (now - this.windowStart >= SAMPLE_MS) {
      this.flushSample(now);
    }
  }

  /** poll đọc những số đắt tiền ở 4 Hz. */
  private poll(): void {
    this.polls++;
    const p = this.host.perfStats();
    const n = this.host.net();
    const c = this.host.context();
    this.pings.push(n.rttMs);
    this.kbps.push(n.kbPerSec);
    this.draws.push(p.drawCalls ?? 0);
    this.sprites.push(p.sprites ?? 0);
    this.corrections.push(p.fixPerSec ?? 0);
    this.maxPkt = Math.max(this.maxPkt, n.snapBytes);
    if (c.placeholderNow) {
      this.placeholderFrames++;
    }
    this.respawn2s += c.respawn2s;
  }

  /** flushSample gộp cửa sổ hiện tại thành một mẫu rồi gửi đi. */
  private flushSample(now: number): void {
    const f = this.frames;
    const secs = (now - this.windowStart) / 1000;
    const fps = secs > 0 ? f.length / secs : 0;
    // FPS 1% thấp = trung bình của 1% KHUNG CHẬM NHẤT, không phải phân vị của FPS.
    // Đây là chỉ số nói lên cú khựng; tính bằng FPS trung bình thì cú khựng biến mất.
    const slow = [...f].sort((a, b) => b - a).slice(0, Math.max(1, Math.ceil(f.length / 100)));
    const slowMs = slow.reduce((t, x) => t + x, 0) / slow.length;
    const c = this.host.context();
    const p = this.host.perfStats();
    const s: Sample = {
      s: this.id,
      ts: Date.now(),
      fa: +fps.toFixed(1),
      f1: slowMs > 0 ? +(1000 / slowMs).toFixed(1) : 0,
      m50: +pct(f, 0.5).toFixed(2),
      m95: +pct(f, 0.95).toFixed(2),
      mmx: +Math.max(0, ...f).toFixed(2),
      o33: f.filter((x) => x > 33).length,
      o100: f.filter((x) => x > 100).length,
      lt: this.longTasks,
      dc: Math.round(avg(this.draws)),
      sp: Math.round(avg(this.sprites)),
      tx: 0, // chưa đo được: PixiJS v8 không công bố tổng bộ nhớ texture (xem PERF_LOG §6)
      tu: 0,
      ph: this.placeholderFrames,
      p50: Math.round(pct(this.pings, 0.5)),
      p95: Math.round(pct(this.pings, 0.95)),
      pmx: Math.round(Math.max(0, ...this.pings)),
      ki: +avg(this.kbps).toFixed(2),
      ko: 0, // server đo chiều gửi chính xác hơn client (xem server_samples.kbps_out)
      mp: this.maxPkt,
      l150: this.pings.filter((x) => x > 150).length,
      st: 0,
      cor: +avg(this.corrections).toFixed(2),
      rs: this.respawn2s,
      im: 0,
      zn: c.zone, rl: c.realm, sws: c.swords, rg: c.rings,
      cv: c.charsInView, tv: c.tienInView, sd: c.swordsDrawn, vf: c.vfx,
      bn: c.burning, zg: c.zoning,
    };
    this.nSamples++;
    this.fpsHistory.push(s.fa);
    this.resetWindow(now);
    this.enqueue(s);
    void p;
  }

  private resetWindow(now: number): void {
    this.windowStart = now;
    this.frames = [];
    this.pings = [];
    this.kbps = [];
    this.draws = [];
    this.sprites = [];
    this.corrections = [];
    this.maxPkt = 0;
    this.longTasks = 0;
    this.placeholderFrames = 0;
    this.respawn2s = 0;
    this.polls = 0;
  }

  /**
   * enqueue xếp mẫu vào hàng rồi thử gửi cả hàng.
   *
   * Gửi cả hàng chứ không chỉ mẫu mới: mất mạng 20 giây thì lúc nối lại phải bù đủ, nếu
   * không thì đúng cái đoạn mạng tậm tịt — đoạn đáng xem nhất — lại là đoạn không có số liệu.
   */
  private enqueue(s: Sample): void {
    this.queue.push(s);
    if (this.queue.length > MAX_QUEUE) {
      // Quá hạn mức thì bỏ mẫu CŨ NHẤT: mẫu mới nói về tình trạng hiện tại, hữu ích hơn.
      this.queue.splice(0, this.queue.length - MAX_QUEUE);
    }
    this.trySend();
  }

  private trySend(): void {
    if (this.queue.length === 0 && this.events.length === 0 && this.sessionSent) {
      return;
    }
    const msg: Record<string, unknown> = { t: 'tm' };
    if (!this.sessionSent && this.session) {
      msg.s = this.session;
    }
    if (this.queue.length > 0) {
      msg.m = this.queue;
    }
    if (this.events.length > 0) {
      msg.e = this.events;
    }
    if (!this.host.send(msg)) {
      return; // chưa nối được: giữ nguyên hàng, lần sau gửi lại
    }
    this.sessionSent = true;
    this.queue = [];
    this.events = [];
  }

  /** event ghi một sự kiện (§2.3), có trần 20/phút để một tab lỗi không làm ngập DB. */
  event(kind: string, data = ''): void {
    if (this.stopped) {
      return;
    }
    const now = Date.now();
    if (now - this.eventWindow >= 60_000) {
      this.eventWindow = now;
      this.eventCount = 0;
    }
    if (++this.eventCount > MAX_EVENTS_PER_MIN) {
      return;
    }
    this.events.push({ s: this.id, ts: now, k: kind, d: data.slice(0, 500) });
    this.trySend();
  }

  private watchLongTasks(): void {
    if (typeof PerformanceObserver === 'undefined') {
      return;
    }
    try {
      this.observer = new PerformanceObserver((list) => {
        this.longTasks += list.getEntries().length;
      });
      this.observer.observe({ entryTypes: ['longtask'] });
    } catch {
      // Firefox/Safari chưa có 'longtask'. Không có thì thôi, cột đó để 0.
      this.observer = null;
    }
  }

  private watchPageEvents(): void {
    // tab_hidden/visible để báo cáo loại các đoạn chạy nền khỏi thống kê FPS: trình duyệt
    // hạ vòng vẽ xuống ~1 Hz khi tab bị ẩn, không lọc ra thì mọi phiên đều "tụt FPS".
    document.addEventListener('visibilitychange', () => {
      this.event(document.hidden ? 'tab_hidden' : 'tab_visible');
    });
    window.addEventListener('error', (e) => {
      this.event('error', cleanError(e.message, e.error));
    });
    window.addEventListener('unhandledrejection', (e) => {
      this.event('error', cleanError(String(e.reason), e.reason));
    });
    // pagehide chứ không phải unload: trên điện thoại `unload` thường không chạy.
    window.addEventListener('pagehide', () => this.end());
  }

  /** end gửi tóm tắt phiên bằng sendBeacon (§2.4) — thứ duy nhất còn gửi được khi đóng tab. */
  end(): void {
    if (this.stopped) {
      return;
    }
    this.stopped = true;
    this.observer?.disconnect();
    const body = JSON.stringify({
      id: this.id,
      dur: (performance.now() - this.startedAt) / 1000,
      n: this.nSamples,
      fps: pct(this.fpsHistory, 0.5),
    });
    try {
      navigator.sendBeacon('/telemetry/end', new Blob([body], { type: 'application/json' }));
    } catch {
      // Đóng tab rồi thì không còn gì để cứu, và cũng không còn ai đọc lỗi.
    }
  }
}

function avg(v: number[]): number {
  return v.length === 0 ? 0 : v.reduce((t, x) => t + x, 0) / v.length;
}

/**
 * cleanError giữ thông điệp + 5 dòng stack đầu, BỎ đường dẫn file máy (§2.3).
 *
 * Đường dẫn trong stack chứa tên người dùng hệ điều hành ("C:\Users\Khoa\..."), nên để
 * nguyên là lưu tên thật — đúng thứ §1.3 cấm.
 */
function cleanError(msg: string, err: unknown): string {
  const stack = err instanceof Error && err.stack ? err.stack.split('\n').slice(1, 6) : [];
  const strip = (s: string): string => s.replace(/(?:https?:\/\/[^\s)]+|[A-Za-z]:\\[^\s)]+|\/[^\s):]+)/g, (m) => {
    const i = Math.max(m.lastIndexOf('/'), m.lastIndexOf('\\'));
    return i >= 0 ? m.slice(i + 1) : m;
  });
  return [msg, ...stack.map(strip)].join('\n').slice(0, 500);
}
