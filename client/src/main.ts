import './ui/style.css';

import { sound } from './audio/sound';
import { type GameConfig, levelThresholds, loadConfig } from './config';
import { Input } from './input';
import { Conn } from './net/conn';
/** Chữ tên khu hiện bao lâu khi vào khu mới (spec §8.0). */
const ZONE_BANNER_SECS = 1.5;

import { FLAG, PROTOCOL_VERSION } from './net/protocol';
import { pendingRejoin, watchContextLoss } from './render/glrecover';
import type { DeadMsg, GameMap, ServerMsg, Snapshot, Welcome } from './net/protocol';
import { ClientWorld } from './net/state';
import { assets } from './render/assets';
import { loadFonts } from './render/fonts';
import { GameScene } from './render/scene';
import { loadJitter } from './render/jitter';
import { cycleQuality, quality, QUALITY_NAME } from './render/quality';
import { ringGeom } from './render/ringgeom';
import type { Telemetry, TelemetryContext } from './telemetry';
import { h, Screens } from './ui/dom';
import { guestId, homeScreen, type Profile } from './ui/home';
import { Hud } from './ui/hud';
import { leaderboardScreen } from './ui/leaderboard';
import { STR } from './ui/strings';
import { summaryScreen } from './ui/summary';

/**
 * Phím dùng kỹ năng (spec §6.3): **Space**.
 *
 * Từng thử **Ctrl** nhưng bỏ, và lý do đáng ghi lại để đừng ai đổi lại: phím đi là **WASD**,
 * nên vừa giữ Ctrl vừa bấm W là **Ctrl+W — đóng tab**. Chrome xếp Ctrl+W vào nhóm phím tắt
 * dành riêng cho trình duyệt, `preventDefault()` KHÔNG chặn được; cách duy nhất chặn thật là
 * Keyboard Lock API, mà nó chỉ chạy ở chế độ toàn màn hình. Space không đụng phím tắt nào.
 */
const SKILL_KEYS = ['Space'];

const DEBUG_KEYS: Record<string, Record<string, unknown>> = {
  KeyE: { cmd: 'exp', n: 100 },
  KeyR: { cmd: 'realm', n: 0 },
  KeyK: { cmd: 'sword', n: 10 },
  KeyX: { cmd: 'die', n: 0 },
  KeyB: { cmd: 'bot', n: 0 }, // thả 1 bot cùng cảnh giới ngay cạnh mình
  KeyG: { cmd: 'skill', n: 1 }, // +1 lượt kỹ năng, khỏi phải đi nhặt Bí Kíp
};

/** fpsCap: 60 on PC, 45 on phones to save battery; override with ?fps=30 (M7). */
function fpsCap(): number {
  const q = Number(new URLSearchParams(location.search).get('fps'));
  if (q > 0) {
    return q;
  }
  return Input.isTouchDevice() ? 45 : 60;
}

class App {
  private readonly screens: Screens;
  private readonly scene: GameScene;
  private readonly world: ClientWorld;
  private readonly hud: Hud;
  private readonly input: Input;
  private readonly conn: Conn;
  private readonly thresholds: number[];
  private profile: Profile | null = null;
  private playing = false;
  // Snapshots that arrive while the map is still loading: the first ones carry names and
  // characters (sent only once per player), so they must not be dropped.
  private joining = false;
  private early: Snapshot[] = [];
  private debug = false;
  /** telemetry: null khi build không bật cờ TELEMETRY (bản release). */
  private telemetry: Telemetry | null = null;
  /** zoneIndex: khu đang đứng, lấy từ Welcome (spec §8.0). Telemetry chia nhóm theo nó. */
  private zoneIndex = 0;
  /** animInfo: phím J đang bật hay tắt (GAME_SPEC §12.5 C). */
  private animInfo = false;

  constructor(private readonly cfg: GameConfig, private readonly overlay: HTMLElement, stage: HTMLElement) {
    this.screens = new Screens(overlay);
    this.scene = new GameScene(cfg);
    this.world = new ClientWorld(cfg);
    this.hud = new Hud(cfg, Input.isTouchDevice());
    this.input = new Input(stage);
    this.bindBurnButton();
    this.bindSkillButton();
    this.conn = new Conn((m) => this.onMessage(m));
    this.thresholds = levelThresholds(cfg);
  }

  async start(stage: HTMLElement): Promise<void> {
    await this.scene.init(stage, fpsCap());
    this.scene.app.ticker.add((t) => this.frame(t.deltaMS / 1000));
    window.setInterval(() => this.sendInput(), 1000 / this.cfg.net.tick_hz);
    window.addEventListener('keydown', (e) => {
      if (this.playing && SKILL_KEYS.includes(e.code) && !e.repeat) {
        this.skillPressed = true; // kỹ năng nhân vật (spec §6.3)
        e.preventDefault();
        return;
      }
      if (this.playing && e.code === 'KeyM' && !e.repeat) {
        this.hud.minimap.toggle(); // spec §12.1
        return;
      }
      if (this.playing && e.code === 'KeyJ' && !e.repeat && this.debug) {
        // GAME_SPEC §12.5 C: hiện tên anim, frame, pha và điểm độ giật trên từng nhân vật.
        this.animInfo = !this.animInfo;
        void loadJitter().then(() => this.scene.showAnimInfo(this.animInfo));
        this.hud.showBanner(this.animInfo ? STR.animInfoOn : STR.animInfoOff, 1.5);
        return;
      }
      if (this.playing && this.animInfo && (e.code === 'BracketLeft' || e.code === 'BracketRight') && !e.repeat) {
        // ANSWERS mục 1: đi lần lượt qua từng animation theo thứ tự bảng điểm để chủ dự án
        // chốt ngưỡng đỏ bằng mắt.
        this.scene.stepAnimViewer(e.code === 'BracketRight' ? 1 : -1);
        e.preventDefault();
        return;
      }
      if (this.playing && e.code === 'F3' && !e.repeat) {
        this.hud.togglePerf(); // spec §13.6: bảng đo bật/tắt bằng F3
        e.preventDefault();
        return;
      }
      if (this.playing && e.code === 'KeyV' && !e.repeat) {
        // Spec §3.6: đổi mức chất lượng Cao / Vừa / Thấp. Thần thú không còn tắt được nữa
        // (v4.7), nhưng số đốt và vệt hạt co theo mức này.
        this.hud.showBanner(STR.qualitySet(QUALITY_NAME[cycleQuality()]), 2.5);
        return;
      }
      this.debugKey(e);
    });
    this.conn.connect(
      () => undefined,
      () => {
        if (this.playing) {
          this.playing = false;
          this.input.enabled = false;
          this.hud.root.remove();
          this.screens.show(h('div', 'screen center', h('div', 'panel', STR.disconnected)));
          window.setTimeout(() => this.showHome(), 2500);
        }
      },
    );
    // Mất ngữ cảnh WebGL (trình điều khiển card reset, máy ngủ dậy): canvas sẽ đen vĩnh
    // viễn nếu không làm gì. Nhớ lại tên + nhân vật rồi tải lại trang và tự vào lại phòng.
    watchContextLoss(
      this.scene.app.canvas as HTMLCanvasElement,
      () => this.profile,
      (canRecover) => {
        this.playing = false;
        this.input.enabled = false;
        this.hud.root.remove();
        this.screens.show(h('div', 'screen center', h('div', 'panel', canRecover ? STR.glLost : STR.glLostManual)));
      },
    );
    // Vừa tải lại vì mất đồ hoạ thì vào thẳng phòng, đừng bắt bấm lại "Vào chơi".
    const again = pendingRejoin();
    if (again) {
      this.join(again);
    } else {
      this.showHome();
    }
  }

  private showHome(): void {
    this.screens.show(homeScreen(this.cfg, (p) => this.join(p), () => this.showLeaderboard(() => this.showHome())));
  }

  private showLeaderboard(back: () => void): void {
    this.screens.show(leaderboardScreen(this.cfg, back));
  }

  private join(p: Profile): void {
    this.profile = p;
    sound.startMusic();
    this.screens.show(h('div', 'screen center', h('div', 'panel', this.conn.isOpen ? STR.findingRoom : STR.connecting)));
    const send = (): void => {
      if (!this.conn.isOpen) {
        window.setTimeout(send, 300);
        return;
      }
      // Mở game bằng ?debug=true để bật bảng phím test cho riêng mình.
      const wantDebug = new URLSearchParams(location.search).get('debug') === 'true';
      this.conn.send({ t: 'join', name: p.name, char: p.char, guest_id: guestId(), debug: wantDebug });
    };
    send();
  }

  private onMessage(m: ServerMsg): void {
    switch (m.t) {
      case 'welcome':
        void this.onWelcome(m);
        break;
      case 'snap':
        if (this.joining) {
          this.early.push(m);
        } else if (this.playing) {
          this.applySnapshot(m);
        }
        break;
      case 'lb':
        this.hud.setLeaderboard(m, this.world.myId);
        break;
      case 'feed':
        this.hud.addFeed(m);
        break;
      case 'dead':
        this.onDead(m);
        break;
      case 'error':
        if (m.msg === 'bad_name') {
          this.showHome();
        }
        break;
      default:
    }
  }

  private srvMs = 0;

  private applySnapshot(m: Snapshot): void {
    this.srvMs = m.sv ?? this.srvMs;
    const events = this.world.applySnapshot(m);
    this.scene.handleEvents(events);
    this.hud.onEvents(events, this.world.myId);
    // Nút kỹ năng đọc số lượt từ chính snapshot của mình (spec §6.3).
    const me = this.world.me();
    if (me) {
      this.hud.setSkill(this.profile?.char ?? '', me.skc ?? 0);
    }
  }

  private async onWelcome(w: Welcome): Promise<void> {
    if ((w.pv ?? 0) !== PROTOCOL_VERSION) {
      // Server và client nói hai thứ tiếng khác nhau. Dừng hẳn ở đây và nói rõ, thay vì
      // đọc rác rồi treo máy — chính là chuyện đã xảy ra một lần.
      this.conn.stop();
      this.screens.show(h('div', 'screen center', h('div', 'panel', STR.versionMismatch)));
      return;
    }
    // Welcome đến GIỮA TRẬN nghĩa là vừa qua cổng sang khu khác (spec §8.0, §8.4): sang
    // khu là đổi hẳn bản đồ nên phải tải lại từ đầu, nhưng KHÔNG phải vào trận lại — giữ
    // nguyên hồ sơ, không phát lại âm thanh hồi sinh, không hiện lại hướng dẫn.
    const changedZone = this.playing;
    this.joining = true;
    this.early = [];
    const res = await fetch(`/api/map/${w.map}`);
    const map = (await res.json()) as GameMap;
    this.world.reset(w, map);
    this.scene.setWorld(this.world);
    this.hud.minimap.setMap(map);
    this.debug = w.debug;
    this.hud.setDebug(w.debug);
    this.hud.setChar(this.profile?.char ?? '');
    this.hud.setSkill(this.profile?.char ?? '', 0);
    if (w.debug) {
      // ?debug=true: mở cửa sau cho console của trình duyệt (xem VFX, đếm thần thú...).
      const g = window as unknown as { kx?: unknown; __stats?: unknown };
      // `playing` phải hỏi qua hàm: `world.me()` vẫn trả về dữ liệu CŨ sau khi chết và rời
      // phòng, nên công cụ đo tưởng nhân vật còn sống và không bấm "Chơi tiếp" (mất mấy
      // lượt đo vì chuyện này).
      g.kx = { scene: this.scene, world: this.world, conn: this.conn, playing: () => this.playing };
      // __stats là cùng bộ số của overlay F3 nhưng ở dạng object, cho tools/browsertest
      // đọc mỗi giây. Giữ tên cố định để script không phải bám vào nội bộ renderer.
      g.__stats = (): Record<string, number> => ({
        ...this.scene.perfStats(),
        ping: this.conn.rttMs,
        kbPerSec: this.conn.kbPerSec,
        snapKB: this.conn.lastSnapBytes / 1024,
        srvMs: this.srvMs,
      });
    }
    this.screens.show(null);
    this.overlay.appendChild(this.hud.root);
    this.playing = true;
    this.joining = false;
    for (const snap of this.early) {
      this.applySnapshot(snap);
    }
    this.early = [];
    this.input.enabled = true;
    this.zoneIndex = w.zn ?? 0;
    this.hud.setZone(w.znm ?? '', w.zn ?? 0);
    // Bắt đầu thu số liệu từ lần vào trận đầu tiên; sang khu khác cũng gửi `welcome` nhưng
    // `startTelemetry` tự bỏ qua lần hai, vì một PHIÊN là một lần chơi chứ không phải một khu.
    void this.startTelemetry();
    if (this.telemetry && changedZone) {
      this.telemetry.event('zone_switch', String(w.zn ?? 0));
    }
    if (changedZone) {
      // Chữ lớn mờ dần giữa màn hình khi vào khu mới (spec §8.0).
      this.hud.showBanner(STR.zoneEntered(w.znm ?? ''), ZONE_BANNER_SECS);
      return;
    }
    this.hud.showGuide(); // hướng dẫn mờ trong lúc còn bất tử (spec §18.1)
    sound.play('respawn');
  }

  private onDead(d: DeadMsg): void {
    this.playing = false;
    this.input.enabled = false;
    this.hud.root.remove();
    const p = this.profile;
    this.screens.show(
      summaryScreen(this.cfg, d, p?.char ?? '', {
        playAgain: () => p && this.join(p),
        leaderboard: () => this.showLeaderboard(() => this.onDead(d)),
        changeCharacter: () => this.showHome(),
      }),
    );
  }

  /** skillPressed: nút kỹ năng vừa được bấm, chờ gói input kế tiếp mang đi (spec §6.3). */
  private skillPressed = false;

  /** bindSkillButton: nút kỹ năng trên điện thoại — bấm một phát, không phải giữ. */
  private bindSkillButton(): void {
    const b = this.hud.skillBtn;
    const press = (e: Event): void => {
      this.skillPressed = true;
      e.preventDefault();
    };
    b.addEventListener('touchstart', press, { passive: false });
    b.addEventListener('mousedown', press);
  }

  /** bindBurnButton makes the on-screen 🔥 a hold button (spec §4.1c). */
  private bindBurnButton(): void {
    const b = this.hud.burnBtn;
    const set = (on: boolean) => (e: Event): void => {
      this.input.burnButton = on;
      e.preventDefault();
    };
    b.addEventListener('touchstart', set(true), { passive: false });
    b.addEventListener('touchend', set(false));
    b.addEventListener('touchcancel', set(false));
    b.addEventListener('mousedown', set(true));
    window.addEventListener('mouseup', () => (this.input.burnButton = false));
  }

  private sendInput(): void {
    if (!this.playing) {
      return;
    }
    const { angle, moving, burn } = this.input.read();
    // Nút kỹ năng gom lại từ lúc gửi gói trước: bấm giữa hai nhịp mạng cũng không mất.
    const skill = this.skillPressed;
    this.skillPressed = false;
    this.conn.send(this.world.makeInput(angle, moving, burn, skill));
  }

  private debugKey(e: KeyboardEvent): void {
    if (!this.playing || !this.debug || e.repeat) {
      return;
    }
    if (e.code === 'KeyH') {
      this.scene.toggleHitbox();
      return;
    }
    const cmd = DEBUG_KEYS[e.code];
    if (cmd) {
      this.conn.send({ t: 'dbg', ...cmd });
      return;
    }
    const n = Number(e.key);
    if (n >= 1 && n <= 7) {
      this.conn.send({ t: 'dbg', cmd: 'buff', n: n - 1 });
    }
  }

  private frame(dt: number): void {
    if (!this.playing) {
      return;
    }
    this.scene.render(dt);
    const me = this.world.me();
    this.hud.update(me, this.conn.rttMs, (L) => ({ from: this.thresholds[L], to: this.thresholds[L + 1] }));
    if (this.debug) {
      this.hud.setPerf(this.scene.perfLine(dt, {
        rttMs: this.conn.rttMs,
        kbPerSec: this.conn.kbPerSec,
        snapBytes: this.conn.lastSnapBytes,
        srvMs: this.srvMs,
      }));
    }
    if (me) {
      const view = this.scene.viewSize();
      this.hud.minimap.render(me.x, me.y, me.f, view.w, view.h, performance.now() / 1000);
    }
    // Telemetry ĐỨNG CUỐI vòng vẽ và chỉ ghi một con số: mọi thứ đắt hơn nó tự dồn về 4 Hz.
    this.telemetry?.frame();
  }

  /**
   * startTelemetry nạp module thu số liệu — chỉ khi build bật cờ `TELEMETRY=on`.
   *
   * `import()` động nằm sau một hằng số biên dịch: bản release có `__TELEMETRY__ === false`
   * nên Rollup cắt hẳn nhánh này và KHÔNG đóng gói module (yêu cầu §1.6 — "module không
   * được đóng gói vào"). Kiểm bằng `npm run check:telemetry-off`.
   */
  private async startTelemetry(): Promise<void> {
    if (!__TELEMETRY__ || this.telemetry) {
      return;
    }
    const { Telemetry } = await import('./telemetry');
    const t = new Telemetry(
      {
        perfStats: () => this.scene.perfStats(),
        net: () => ({ rttMs: this.conn.rttMs, kbPerSec: this.conn.kbPerSec, snapBytes: this.conn.lastSnapBytes }),
        context: () => this.telemetryContext(),
        send: (msg) => {
          // `Conn.send` tự bỏ gói khi chưa nối; telemetry cần BIẾT điều đó để còn giữ mẫu
          // lại gửi bù (§3), nên hỏi trạng thái trước rồi mới gửi.
          if (!this.conn.isOpen) {
            return false;
          }
          this.conn.send(msg);
          return true;
        },
        canvasSize: () => {
          const c = this.scene.app.canvas as HTMLCanvasElement;
          return { w: c.width, h: c.height };
        },
      },
      BUILD_ID,
      PROTOCOL_VERSION,
    );
    t.start();
    this.telemetry = t;
    // Mất ngữ cảnh WebGL là ngưỡng cảnh báo §4.3 — chính là lỗi đội test gặp ngày 30/10.
    const canvas = this.scene.app.canvas as HTMLCanvasElement;
    canvas.addEventListener('webglcontextlost', () => t.event('webgl_context_lost'));
    canvas.addEventListener('webglcontextrestored', () => t.event('webgl_context_restored'));
  }

  /** telemetryContext gom bối cảnh trận (§2.2 nhóm cuối). Chỉ ĐỌC trạng thái game. */
  private telemetryContext(): TelemetryContext {
    const me = this.world.me();
    const draw = this.scene.drawStats();
    // Số người Tiên giới trong khung nhìn: spec §4.1 mục 4 tách riêng nhóm này vì họ mang
    // vòng kiếm nhiều lớp và hào quang — vài người Tiên giới cùng khung là tình huống nặng
    // nhất mà game tạo ra, và cũng là chỗ đáng ngờ nhất khi ai đó báo tụt FPS.
    let tien = 0;
    const tienFrom = this.cfg.cultivation.realms.length; // cảnh giới >= số cảnh giới thường
    for (const p of this.world.players.values()) {
      const last = p.snaps[p.snaps.length - 1];
      if (last && last.s.L >= tienFrom) {
        tien++;
      }
    }
    return {
      zone: this.zoneIndex,
      realm: me?.L ?? 0,
      swords: me?.n ?? 0,
      rings: me ? ringGeom(this.cfg, me.sc, me.n).rings : 0,
      charsInView: draw.charsInView,
      tienInView: tien,
      swordsDrawn: draw.swordsDrawn,
      // vfx: chưa đo được — lớp VFX không giữ bộ đếm số hiệu ứng đang chạy. Để 0 chứ không
      // đoán: một con số bịa trong báo cáo hiệu năng còn tệ hơn một ô trống.
      vfx: 0,
      burning: ((me?.fl ?? 0) & FLAG.burning) !== 0,
      zoning: this.joining,
      placeholderNow: draw.placeholder,
      respawn2s: this.world.takeRespawn2s(),
      quality: QUALITY_NAME[quality()] ?? '',
      qualityAuto: false,
    };
  }
}

async function boot(): Promise<void> {
  const overlay = document.getElementById('overlay') as HTMLElement;
  const stage = document.getElementById('stage') as HTMLElement;
  const loading = h('div', 'screen center', h('div', 'panel', h('div', 'logo', STR.title), h('div', 'muted', STR.connecting)));
  overlay.appendChild(loading);
  try {
    const cfg = await loadConfig();
    await assets.init();
await loadFonts(); // BitmapFont đủ dấu tiếng Việt cho bảng tên (spec §12.3)
    await assets.preload();
    sound.init();
    loading.remove();
    await new App(cfg, overlay, stage).start(stage);
  } catch (err) {
    console.error(err);
    loading.replaceChildren(h('div', 'panel', `${STR.disconnected} (${String(err)})`));
    window.setTimeout(() => location.reload(), 3000);
  }
}

void boot();
