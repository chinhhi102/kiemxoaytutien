import { sound } from '../audio/sound';
import type { GameConfig } from '../config';
import { maxSwords, realmName, realmOf, tienIndex, tierOf } from '../config';
import type { FeedMsg, GameEvent, LeaderboardMsg, PlayerState } from '../net/protocol';
import { BUFFS, moodName } from '../net/protocol';
import { assets } from '../render/assets';
import { REALM_BADGE } from '../render/palette';
import { BUFF_COLORS } from '../render/palette';
import { button, h } from './dom';
import { narrowScreen } from './layout';
import { Minimap } from './minimap';
import { STR } from './strings';

const FEED_SECS = 6;
const SERVER_FEED_SECS = 9;

/**
 * Màn hình điện thoại dọc: HUD phải CO LẠI, không chỉ nhỏ chữ.
 *
 * Trên điện thoại, bảng xếp hạng 10 dòng + kill feed 5 tin + minimap cỡ PC ăn gần hết màn
 * hình — người chơi nhìn vào thấy toàn bảng biểu, không thấy trận đấu. Nhỏ chữ lại không
 * cứu được: vấn đề là SỐ DÒNG, nên phải cắt bớt dòng.
 *
 * Lấy mốc theo KÍCH THƯỚC THẬT chứ không theo "có cảm ứng không": máy tính bảng ngang vẫn
 * rộng rãi, còn cửa sổ trình duyệt hẹp trên PC thì chật y như điện thoại (xem ảnh chụp
 * 30/10 — cửa sổ hẹp bị đè y hệt máy điện thoại).
 *
 * Mốc hẹp/rộng nằm ở `layout.ts` để `minimap.ts` và CSS dùng chung một con số.
 */
const narrow = narrowScreen;
/** Số tin kill feed giữ lại: màn hẹp **1 tin** (chủ dự án chốt 30/10), màn rộng 5. */
function feedKeep(): number {
  return narrow() ? 1 : 5;
}
/** Số dòng bảng xếp hạng phòng: màn hẹp Top 3, màn rộng Top 10. */
function boardRows(): number {
  return narrow() ? 3 : 10;
}
/** Tin loan toàn server: ít dòng hơn nhưng ở lại lâu hơn vì nó là chuyện lớn (spec §11.4). */
function serverFeedKeep(): number {
  return narrow() ? 1 : 3;
}

/**
 * colorizeFeed splits a kill-feed sentence into coloured pieces so it can be read at a glance:
 * who did it (vàng), what happened (trắng), and to whom (đỏ). The sentences come from the joke
 * pools in strings.ts, so the split is done by finding the two names inside the finished text
 * rather than by rebuilding every variant.
 */
function colorizeFeed(text: string, actor: string, target: string): Node[] {
  const marks: { at: number; len: number; cls: string }[] = [];
  const findAll = (name: string, cls: string): void => {
    if (!name) {
      return;
    }
    for (let i = text.indexOf(name); i >= 0; i = text.indexOf(name, i + name.length)) {
      marks.push({ at: i, len: name.length, cls });
    }
  };
  // Tên dài tìm trước: tên này có thể nằm lọt trong tên kia.
  const [first, second] = actor.length >= target.length ? [actor, target] : [target, actor];
  findAll(first, first === actor ? 'f-actor' : 'f-target');
  findAll(second, second === actor ? 'f-actor' : 'f-target');
  marks.sort((a, b) => a.at - b.at);
  const out: Node[] = [];
  let i = 0;
  for (const m of marks) {
    if (m.at < i) {
      continue; // chồng lên đoạn đã tô
    }
    if (m.at > i) {
      out.push(h('span', 'f-act', text.slice(i, m.at)));
    }
    out.push(h('span', m.cls, text.slice(m.at, m.at + m.len)));
    i = m.at + m.len;
  }
  if (i < text.length) {
    out.push(h('span', 'f-act', text.slice(i)));
  }
  return out;
}

/** Hud is the in-match DOM overlay: EXP bar, swords, buffs, room top 10, kill feed (spec §6, §11). */
/** inkOnHex: chữ đen hay trắng thì đọc được trên nền này (xem nameplate.ts). */
function inkOnHex(bg: number): string {
  const r = (bg >> 16) & 0xff;
  const g = (bg >> 8) & 0xff;
  const b = bg & 0xff;
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.55 ? '#14110c' : '#ffffff';
}

/**
 * Icon tạm cho 10 kỹ năng (spec §6.3). Dùng emoji cho tới khi có art thật — thà có thứ
 * nhìn ra được ngay còn hơn một ô vuông trống chờ ảnh.
 */
const SKILL_ICONS: Record<string, string> = {
  a_hon: '🥟',
  ba_tam_loa: '📢',
  ong_ba_vong: '💤',
  dai_ca_dep: '🩴',
  map_tra_sua: '🧋',
  co_ba_ganh: '🧺',
  su_phu_mi: '🍜',
  be_ti: '💪',
  shipper: '⚡',
  cu_rua: '🐢',
};

export class Hud {
  readonly root = h('div', 'hud');
  private readonly badgeWrap = h('div', 'badge-wrap');
  private readonly badge = h('div', 'badge');
  private readonly portrait = h('div', 'portrait');
  private readonly badgeTag = h('div', 'badge-tag');
  private readonly crown = h('div', 'badge-crown');
  private badgeKey = '';
  private readonly realm = h('div', 'realm');
  private readonly expFill = h('div', 'exp-fill');
  private readonly expBar: HTMLElement;
  private readonly expLabel = h('div', 'exp-label');
  private readonly swords = h('div', 'swords');
  private readonly scoreBox = h('div', 'score');
  private readonly pops = h('div', 'score-pops');
  private readonly buffs = h('div', 'buffs');
  private readonly board = h('div', 'room-board');
  private readonly feed = h('div', 'feed');
  /** serverFeed is the top-centre strip for server-wide announcements (spec §11.4). */
  private readonly serverFeed = h('div', 'server-feed');
  private readonly banner = h('div', 'banner');
  /** Tên khu đang đứng, hiện cạnh minimap (spec §8.0). */
  private readonly zoneBox = h('div', 'zone-name');
  /** Khu người chơi đang đứng — để biết tin kill feed nào đến từ khu khác (spec §8.0). */
  private myZone = 0;
  private readonly ping = h('div', 'ping');
  private readonly debugHelp = h('div', 'debug-help', STR.debugHelp);
  /** Đồng hồ đo hiệu năng, chỉ hiện khi ?debug=true — để biết lag nằm ở đâu (xem setPerf). */
  private readonly perf = h('div', 'perf');
  private perfOn = false;
  private buffEnds = new Map<string, { end: number; total: number }>();
  private bannerUntil = 0;

  readonly minimap: Minimap;
  /** burnBtn is the touch-only hold button for Thiêu Đốt Tinh Huyết (spec §4.1c). */
  readonly burnBtn = h('button', 'burn-btn', '🔥');
  /**
   * Nút kỹ năng (spec §6.3): icon kỹ năng của nhân vật + số lượt đang có, xám khi 0 lượt.
   *
   * Luôn hiện cả trên PC — ở đó nó vừa là nút bấm được, vừa là chỗ đọc còn mấy lượt. Spec
   * chỉ nói phím bấm cho PC, nhưng không hiện số lượt ở đâu thì người chơi phải đoán.
   */
  readonly skillBtn = h('button', 'skill-btn');
  private readonly skillCount = h('span', 'skill-count', '0');
  private readonly skillIcon = h('span', 'skill-icon', '✦');
  private readonly guide = h('div', 'guide');
  private readonly isTouch: boolean;

  /** char: id nhân vật đang chơi, để lấy chân dung (spec §12.4). Đặt khi vào phòng. */
  private char = '';

  setChar(char: string): void {
    this.char = char;
    this.badgeKey = ''; // đổi nhân vật thì dựng lại ô đại diện
  }

  constructor(private readonly cfg: GameConfig, isTouch = false) {
    this.isTouch = isTouch;
    // Ảnh nằm TRONG khung tròn (khung cắt mọi thứ tràn ra), còn huy hiệu tầng và vương miện
    // nằm NGOÀI khung — chúng cố tình thò ra mép nên không được để khung cắt mất.
    this.badge.append(this.portrait);
    this.badgeWrap.append(this.badge, this.badgeTag, this.crown);
    this.expBar = h('div', 'exp-bar', this.expFill);
    this.minimap = new Minimap(cfg, isTouch);
    this.burnBtn.title = STR.burnName;
    this.burnBtn.style.display = isTouch ? 'flex' : 'none';
    this.skillBtn.title = STR.skillBtnHint(isTouch);
    const mute = button('🔊', 'mute', () => {
      mute.textContent = sound.toggleMute() ? '🔇' : '🔊';
    });
    // Cột trái xếp theo LUỒNG DỌC, không ghim toạ độ cứng cho từng khối.
    //
    // Trước đây mỗi khối có `top` riêng (status 12, buffs 84, minimap 136). Chỉ cần khối
    // trạng thái cao thêm một dòng — lên Tiên giới có thêm thanh Tiên khí, hoặc tên khu
    // mới thêm vào — là nó đè lên hàng buff. Đã dính thật: ảnh chụp của chủ dự án cho thấy
    // chữ "Kiếm 25/30" nằm sau ba cái icon buff.
    this.root.append(
      h('div', 'hud-left',
        h('div', 'status', this.badgeWrap, h('div', 'status-text', this.realm, this.expBar, this.expLabel, this.swords, this.scoreBox)),
        this.buffs,
        this.zoneBox,
        this.minimap.root),
      this.pops,
      this.board,
      // Kill feed và tin loan toàn server vào CHUNG một cột dọc. Trên màn hẹp cả hai đều
      // nằm góc dưới trái, và ghim `bottom` riêng cho từng cái thì cái nào dài ra là đè
      // cái kia — đã đo được đúng vậy (212×21px trên máy 320, 312×56px trên cửa sổ hẹp).
      // Xếp dọc thì không thể chồng nhau, giống hệt cách đã chữa cột trái.
      h('div', 'hud-bottom-left', this.feed, this.serverFeed),
      this.banner,
      this.burnBtn,
      this.skillBtn,
      this.guide,
      h('div', 'corner', this.ping, mute),
      this.debugHelp,
      this.perf,
      h('div', 'rotate-hint', STR.rotateHint),
    );
    this.debugHelp.style.display = 'none';
    this.perf.style.display = 'none';
    // Xoay ngang điện thoại là đổi hẳn mốc hẹp/rộng. Bảng xếp hạng chờ gói sau mới vẽ lại
    // (1 giây), còn kill feed thì chỉ đổi khi có tin mới — xoay xong mà vẫn thấy 5 tin cũ
    // chiếm nửa màn hình thì người chơi tưởng cài đặt không ăn.
    window.addEventListener('resize', () => this.trimLists());
  }

  /** trimLists cắt lại kill feed cho khớp bề ngang màn hình hiện tại. */
  private trimLists(): void {
    while (this.feed.children.length > feedKeep()) {
      this.feed.lastElementChild?.remove();
    }
    while (this.serverFeed.children.length > serverFeedKeep()) {
      this.serverFeed.lastElementChild?.remove();
    }
  }

  /**
   * showGuide fades a short "how to play" card over the first seconds of a match so a new
   * player learns the rules while they are still invulnerable (spec §18.1).
   */
  showGuide(): void {
    this.guide.replaceChildren(
      h('div', 'guide-title', STR.guideTitle),
      ...STR.guideKeys(this.isTouch).map((k) => h('div', 'guide-line', h('span', 'guide-key', k.key), h('span', '', k.what))),
    );
    this.guide.classList.remove('fade');
    // Khởi động lại animation: trình duyệt bỏ qua nếu chỉ gỡ rồi gắn class ngay.
    void this.guide.offsetWidth;
    this.guide.classList.add('fade');
  }

  setDebug(on: boolean): void {
    this.debugHelp.style.display = on ? 'block' : 'none';
    this.perfOn = on;
    this.perf.style.display = on ? 'block' : 'none';
  }

  /** togglePerf bật/tắt bảng đo bằng phím F3 (spec §13.6). */
  togglePerf(): void {
    this.perfOn = !this.perfOn;
    this.perf.style.display = this.perfOn ? 'block' : 'none';
  }

  /** setPerf ghi một dòng số liệu hiệu năng lên góc màn hình (chỉ ở chế độ debug). */
  setPerf(text: string): void {
    this.perf.textContent = text;
  }

  update(me: PlayerState | null, rttMs: number, prog: (L: number) => { from: number; to: number }): void {
    const now = performance.now() / 1000;
    this.ping.textContent = STR.ping(rttMs);
    if (now > this.bannerUntil) {
      this.banner.style.display = 'none';
    }
    this.updateBuffs(me, now);
    if (!me) {
      return;
    }
    const r = realmOf(this.cfg, me.L);
    const tienHere = tienIndex(this.cfg, r) >= 0;
    this.setBadge(r, tienHere ? 0 : tierOf(this.cfg, me.L), me.ti ?? 0, moodName(me.md));
    // Tiên giới: không chia tầng, thanh tiến độ đầy bằng sát mạng (spec §3.4).
    const tien = tienIndex(this.cfg, r) >= 0;
    this.realm.textContent = tien ? realmName(this.cfg, r) : STR.realmTier(realmName(this.cfg, r), tierOf(this.cfg, me.L));
    this.expBar.classList.toggle('tien', tien);
    let frac: number;
    if (tien) {
      const need = me.tn ?? 0;
      const have = me.th ?? 0;
      frac = need > 0 ? Math.min(1, have / need) : 1;
      this.expLabel.textContent = need > 0 ? STR.tienKhi(have, need) : STR.tienMax;
      // Đạo Tổ nạp đầy trần (spec v5.7 §3.4): từ lúc này Tiên khí nhặt thêm mới đổi sang
      // Uy Danh. Phải báo rõ, vì trước mốc đó nhặt bao nhiêu cũng không thấy Uy Danh tăng
      // và người chơi sẽ tưởng luật hỏng.
      const full = need > 0 && have >= need - 1e-9 && tienIndex(this.cfg, r) === this.cfg.cultivation.tien_realms.length - 1;
      this.expBar.classList.toggle('tien-full', full);
      if (full) {
        this.expLabel.textContent = STR.tienOverflow;
      }
    } else {
      const { from, to } = prog(me.L);
      frac = to > from ? Math.min(1, (me.xp - from) / (to - from)) : 1;
      this.expLabel.textContent = STR.linhKhi;
    }
    this.expFill.style.width = `${(frac * 100).toFixed(1)}%`;
    this.swords.textContent = STR.swords(me.n, maxSwords(this.cfg, r));
  }

  /**
   * setBadge vẽ ô đại diện ở góc trên trái (spec §12.4): chân dung nhân vật trong khung
   * tròn, viền màu cảnh giới, huy hiệu cảnh giới + tầng ở góc dưới phải, vương miện nếu
   * đang Top 1. Thay cho vòng tròn ghi mỗi con số cấp — số thì ai cũng phải dịch trong đầu,
   * còn mặt nhân vật thì nhận ra ngay.
   *
   * Chưa có art chân dung riêng thì cắt phần đầu (45% trên) của khung `idle_00`.
   */
  private setBadge(realm: number, tier: number, rank: number, mood: string): void {
    const key = `${realm}|${tier}|${rank}|${mood}`;
    if (key === this.badgeKey) {
      return; // chỉ dựng lại khi có gì đổi, không chạm DOM mỗi khung hình
    }
    this.badgeKey = key;
    this.setPortrait(this.portraitUrl(mood));
    const col = `#${(REALM_BADGE[realm] ?? 0x8d99ae).toString(16).padStart(6, '0')}`;
    this.badge.style.borderColor = col;
    this.badge.classList.toggle('tien', realm >= this.cfg.cultivation.realms.length);
    this.badgeTag.textContent = tier > 0 ? String(tier) : '仙';
    this.badgeTag.style.background = col;
    this.badgeTag.style.color = inkOnHex(REALM_BADGE[realm] ?? 0x8d99ae);
    this.crown.style.display = rank === 1 ? 'block' : 'none';
    this.crown.textContent = rank === 1 ? '♚' : '';
  }

  /**
   * portraitUrl chọn ảnh chân dung theo tâm trạng (spec §12.4). Chưa có bộ chân dung riêng
   * thì dùng khung `idle_00` và để CSS cắt lấy phần đầu.
   */
  private portraitUrl(mood: string): string {
    const want = mood === 'chase' ? 'evil' : mood === 'hit' || mood === 'no_swords' ? 'panic' : mood === 'realm_drop' ? 'cry' : 'neutral';
    if (!this.char) {
      return '';
    }
    const real = `characters/${this.char}/portrait_${want}.png`;
    if (assets.has(real)) {
      return `/assets/${real}`;
    }
    // Dự phòng là chân dung TRUNG TÍNH, không phải `frames/idle_00.png`: `frames/` là ảnh
    // nguồn, không có trong manifest và không được deploy (xem tools/deploy.sh).
    const neutral = `characters/${this.char}/portrait_neutral.png`;
    return assets.has(neutral) ? `/assets/${neutral}` : '';
  }

  /**
   * setPortrait nạp ảnh rồi mới gắn làm nền. Không kiểm tra qua danh sách asset được: ảnh
   * từng khung của nhân vật đã được gộp vào atlas nên không nằm trong danh sách, dù file vẫn
   * có trên đĩa. Tải hụt thì rơi về hai chữ cái đầu của tên nhân vật.
   */
  private setPortrait(url: string): void {
    const initials = this.char.split('_').map((w) => w[0] ?? '').join('').toUpperCase().slice(0, 2);
    if (!url) {
      this.portrait.style.backgroundImage = '';
      this.portrait.textContent = initials;
      return;
    }
    const img = new Image();
    img.onload = () => {
      this.portrait.style.backgroundImage = `url(${url})`;
      this.portrait.textContent = '';
    };
    img.onerror = () => {
      this.portrait.style.backgroundImage = '';
      this.portrait.textContent = initials;
    };
    img.src = url;
  }

  private updateBuffs(me: PlayerState | null, now: number): void {
    const active = BUFFS.filter((_, i) => me && me.bf & (1 << i));
    const nodes = active.map((b) => {
      const t = this.buffEnds.get(b);
      const left = t ? Math.max(0, t.end - now) : 0;
      const frac = t ? left / t.total : 1;
      const icon = h('div', 'buff');
      icon.title = STR.buffs[b] ?? b;
      const color = `#${(BUFF_COLORS[b] ?? 0xaaaaaa).toString(16).padStart(6, '0')}`;
      icon.style.background = `conic-gradient(${color} ${frac * 360}deg, rgba(0,0,0,.45) 0)`;
      const inner = h('div', 'buff-inner');
      if (assets.has(`buffs/${b}.png`)) {
        inner.style.backgroundImage = `url(/assets/buffs/${b}.png)`;
      } else {
        inner.textContent = (STR.buffs[b] ?? b).slice(0, 2);
      }
      icon.append(inner, h('span', 'buff-left', left > 0 ? String(Math.ceil(left)) : ''));
      return icon;
    });
    this.buffs.replaceChildren(...nodes);
  }

  /** onEvents tracks my buff timers and the chest warning. */
  onEvents(events: GameEvent[], myId: number): void {
    const now = performance.now() / 1000;
    for (const e of events) {
      if (e.k === 'Score' && e.a === myId) {
        // Điểm vượt mức cơ bản của một lần chém = có nhân lật kèo (spec §11.1).
        const base = this.cfg.score?.hit_base ?? 100;
        this.popScore(e.n ?? 0, e.s ?? '', e.s === 'hit' && (e.n ?? 0) > base);
      } else if (e.k === 'BuffStart' && e.a === myId && e.s) {
        const total = (e.n ?? 0) / 1000;
        this.buffEnds.set(e.s, { end: now + total, total });
      } else if (e.k === 'ChestIncoming') {
        this.showBanner(STR.chestIncoming(e.n ?? this.cfg.items.chest_warn_secs), 4);
      }
    }
  }

  /**
   * setSkill cập nhật nút kỹ năng: icon theo nhân vật, số lượt, và xám khi hết lượt.
   */
  setSkill(char: string, charges: number): void {
    this.skillIcon.textContent = SKILL_ICONS[char] ?? '✦';
    this.skillCount.textContent = String(charges);
    this.skillBtn.classList.toggle('empty', charges <= 0);
    this.skillBtn.title = STR.skillNames[char] ?? '';
    if (!this.skillBtn.firstChild) {
      this.skillBtn.append(this.skillIcon, this.skillCount);
    }
  }

  /** setZone đổi tên khu cạnh minimap; chuỗi rỗng thì ẩn hẳn (phòng 1 khu như trước). */
  setZone(name: string, index = 0): void {
    this.myZone = index;
    this.minimap.setZone(index);
    this.zoneBox.textContent = name;
    this.zoneBox.style.display = name ? 'block' : 'none';
  }

  showBanner(text: string, secs: number): void {
    this.banner.textContent = text;
    this.banner.style.display = 'block';
    this.bannerUntil = performance.now() / 1000 + secs;
  }

  /** setLeaderboard draws the room top 10: realm+tier order, Uy Danh, Top 1–3 badges (§11.2). */
  setLeaderboard(lb: LeaderboardMsg, myId: number): void {
    this.minimap.setLeaderboard(lb);
    this.scoreBox.textContent = STR.scoreHud(lb.ms ?? 0);
    // Xếp theo Uy Danh; mỗi dòng ghi danh hiệu thay cho cảnh giới (cảnh giới đã hiện
    // trên đầu nhân vật trong trận — spec §11.2).
    const rows = lb.top.slice(0, boardRows()).map((e, i) => {
      const line = h('div', `lb-line${e.id === myId ? ' me' : ''}`);
      const who = h('span', 'lb-who', `${i + 1}. ${e.nm}`);
      if (e.ti === 1) {
        who.prepend(h('span', 'lb-crown', '👑'));
      }
      const titleText = STR.titles[(e.ti || i + 1) - 1] ?? '';
      line.append(
        h('div', 'lb-main', who, h('span', 'lb-score', Math.round(e.sc ?? 0).toLocaleString('vi-VN'))),
        h('div', `lb-title t${Math.min(e.ti || i + 1, 3)}`, titleText),
      );
      return line;
    });
    this.board.replaceChildren(h('div', 'board-title', STR.roomTop), ...rows, h('div', 'my-rank', STR.yourRank(lb.rank, lb.total)));
  }

  /** popScore floats a "+800 LẬT KÈO!" number up when Uy Danh is earned (spec §11.1). */
  private popScore(points: number, source: string, upset: boolean): void {
    const el = h('div', `score-pop ${source}`, upset ? `${STR.scorePop(points)} ${STR.scoreUpset}` : STR.scorePop(points));
    this.pops.prepend(el);
    while (this.pops.children.length > 4) {
      this.pops.lastElementChild?.remove();
    }
    window.setTimeout(() => el.remove(), 1600);
  }

  addFeed(f: FeedMsg): void {
    const realm = this.cfg.cultivation.realms[f.r] ?? '';
    let text =
      f.k === 'kill' ? STR.feed.kill(f.a, f.b ?? '')
      : f.k === 'realm_drop' ? STR.feed.realm_drop(f.a, f.b ?? '', realm)
      : f.k === 'breakthrough' ? STR.feed.breakthrough(f.a, realm)
      : f.k === 'gate_kill' ? STR.feed.gate_kill(f.a, f.b ?? '')
      : f.k === 'title_gain' ? STR.feed.title_gain(f.a)
      : f.k === 'title_lose' ? STR.feed.title_lose(f.a)
      : f.k === 'tien_ascend' ? STR.feed.tien_ascend(f.a, realmName(this.cfg, f.r))
      : f.k === 'heaven_gate' ? STR.feed.heaven_gate(f.a, STR.titles[(f.r || 1) - 1] ?? '')
      : '';
    if (!text) {
      return;
    }
    // Tin loan toàn server đi lên dải giữa-trên màn hình, không lẫn vào feed góc trái.
    if ((f.tp ?? 0) > 0) {
      const big = h('div', 'server-line');
      big.append(h('span', 'feed-tag', STR.feed.serverTag(f.tp ?? 0)), ...colorizeFeed(text, f.a, f.b ?? ''));
      this.serverFeed.prepend(big);
      while (this.serverFeed.children.length > serverFeedKeep()) {
        this.serverFeed.lastElementChild?.remove();
      }
      window.setTimeout(() => big.remove(), SERVER_FEED_SECS * 1000);
      return;
    }
    const line = h('div', 'feed-line');
    // Tin từ KHU KHÁC hiện kèm tên khu (spec §8.0) — không có tiền tố thì người chơi tưởng
    // vừa có người chết ngay cạnh mình, trong khi chuyện xảy ra ở đầu kia thế giới.
    if (f.znm && (f.zn ?? 0) !== this.myZone) {
      line.append(h('span', 'feed-zone', `[${f.znm}]`), document.createTextNode(' '));
    }
    line.append(...colorizeFeed(text, f.a, f.b ?? ''));
    this.feed.prepend(line);
    while (this.feed.children.length > feedKeep()) {
      this.feed.lastElementChild?.remove();
    }
    window.setTimeout(() => line.remove(), FEED_SECS * 1000);
  }
}
