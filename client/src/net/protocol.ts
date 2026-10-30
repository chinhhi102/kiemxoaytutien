// Wire types mirroring server/internal/netproto (MessagePack maps with short keys).

export interface PlayerState {
  id: number;
  x: number;
  y: number;
  f: number; // facing
  L: number; // level index
  n: number; // sword count
  ph: number; // ring phase at snapshot time
  d: number; // ring direction ±1
 // ω rad/s
  tl: number; // blade tilt (rad); ring sizes are derived from sc and n alone (spec v4.3)
  sc: number; // scale
  /** md: chỉ số tâm trạng trong MOODS (spec mục 27) — gửi chuỗi thì tốn 13 byte mỗi gói. */
  md: number; // mood = animation name
  bf: number; // buff bits
  fl: number; // flags
  ck?: number; // crack
  xp: number;
  /** vx/vy: vận tốc, chỉ có ở gói gửi cho CHÍNH MÌNH và chỉ khi đang trượt trên băng (§8.0). */
  vx?: number;
  vy?: number;
  /** skc: số lượt kỹ năng đang giữ, chỉ gửi cho chính mình (spec §6.3). */
  skc?: number;
  /** st: bit trạng thái do kỹ năng gây ra (spec §6.3) — xem STATUS. */
  st?: number;
  ti?: number; // room title badge 1..3 (spec §11.3)
  th?: number; // Tiên khí đang có (spec §3.4), chỉ gửi cho chính mình
  tn?: number; // Tiên khí cần cho cảnh giới sau; 0 = đã Đạo Tổ
  nm?: string;
  ch?: string;
}

/**
 * Toạ độ quái và vật phẩm là **số nguyên đơn vị thế giới** (uint16), không phải float —
 * spec mục 27 "lượng tử hoá". Sai số tối đa nửa đơn vị, bằng 1/60 bán kính thân người.
 *
 * Toạ độ NGƯỜI CHƠI vẫn giữ float: client dùng nó để đối chiếu phần mình tự dự đoán
 * (§8.2a), thêm sai số vào đó là bộ đếm "hiệu chỉnh/giây" vừa kéo về 0 sẽ chạy lại.
 */
export interface MonsterState {
  id: number;
  x: number;
  y: number;
  /** hp CHỈ có khi máu đổi (delta snapshot, spec mục 27); thiếu = giữ nguyên máu cũ. */
  hp?: number;
  /** k, mx, r chỉ có ở gói ĐẦU TIÊN thấy con quái này (spec mục 27); sau đó client giữ lại. */
  k: string;
  mx: number;
  r: number;
}

/**
 * ItemState là dạng **client dùng bên trong**: `k` là tên loại.
 *
 * Trên dây `k` là chỉ số (`WireItemState`), và việc đổi chỉ số → tên làm đúng MỘT lần ngay
 * khi nhận gói (`state.ts`). Nếu để kiểu số lan tới tận chỗ vẽ thì mọi phép so `k === 'orb'`
 * rải khắp `entities.ts`, `orblayer.ts`, `scene.ts` đều phải sửa — và chỉ cần sót một chỗ là
 * vật phẩm vẽ sai loại mà không có gì báo.
 */
export interface ItemState {
  id: number;
  x: number;
  y: number;
  k: string;
  v?: number;
}

/** WireItemState là đúng những gì server gửi: `k` là chỉ số trong ITEM_KINDS. */
export interface WireItemState {
  id: number;
  x: number;
  y: number;
  k: number;
  v?: number;
}

/**
 * ITEM_KINDS khớp **từng dòng** với `netproto.ItemKinds` bên server: loại vật phẩm gửi qua
 * mạng bằng chỉ số, vì "skill_scroll" tốn 13 byte mỗi viên mỗi gói mà một gói có thể có 60
 * vật phẩm mới vào tầm nhìn. Thêm loại mới thì thêm vào CUỐI, đừng chèn giữa.
 */
export const ITEM_KINDS = ['orb', 'sword', 'bundle', 'chest', 'tien_khi', 'skill_scroll',
  'spin', 'haste', 'magnet', 'hard', 'shield', 'exp2', 'stealth'] as const;

/** itemKind đổi chỉ số trên dây thành tên loại; chỉ số lạ trả 'orb' để không làm sập client. */
export function itemKind(k: number): string {
  return ITEM_KINDS[k] ?? 'orb';
}

export interface GameEvent {
  k: string;
  a?: number;
  b?: number;
  x: number;
  y: number;
  n?: number;
  s?: string;
  /** f: một số thực đi kèm — ví dụ số giây của trạng thái (spec §6.3). */
  f?: number;
}

export interface Snapshot {
  t: 'snap';
  tk: number;
  tm: number;
  ack: number;
  p: PlayerState[] | null;
  m: MonsterState[] | null;
  ia?: WireItemState[];
  ir?: number[];
  ev?: GameEvent[];
  /**
   * ds: id những entity server báo đã RỜI ĐI THẬT (NETCODE_SPEC §4.3).
   *
   * Trước đây client tự đoán bằng "vắng mặt quá 2 giây". Đoán sai cả HAI chiều: vắng vì
   * server bỏ qua ở tick lẻ thì xoá oan (nhấp nháy), còn người đi thật thì giữ thừa 2 giây.
   */
  ds?: number[];
  /** sv: thời gian server xử lý một tick của phòng (ms) — chỉ để overlay đo (spec §13.6). */
  sv?: number;
}

/**
 * Phải khớp với `netproto.ProtocolVersion` bên server. Tăng cả hai MỖI KHI đổi hình dạng
 * gói tin.
 *
 * Không có nó thì client mới gặp server cũ sẽ đọc rác rồi **đứng hình không báo gì** — đã
 * dính đúng một lần: client nói `mood` là chỉ số, server cũ vẫn gửi chuỗi.
 */
/** Thứ tự phải khớp game.StatusNames bên server (spec §6.3). */
export const STATUS = ['slow', 'root', 'blind', 'slip', 'brace', 'ghost', 'shell', 'dash', 'stun'] as const;

/** statusOn: trạng thái này đang bật không (bit theo đúng thứ tự `STATUS` = `game.St*`). */
export function statusOn(st: number | undefined, name: (typeof STATUS)[number]): boolean {
  return ((st ?? 0) & (1 << STATUS.indexOf(name))) !== 0;
}

export const PROTOCOL_VERSION = 4;

export interface Welcome {
  t: 'welcome';
  id: number;
  room: string;
  map: string;
  tick_hz: number;
  time: number;
  obs: { id: number; hp: number; b: boolean }[] | null;
  debug: boolean;
  /** pv: phiên bản giao thức của server; thiếu = server quá cũ, chưa có trường này. */
  pv?: number;
  /** zn / znm: khu đang ở và tên khu (spec §8.0). Gửi lại mỗi lần sang khu khác. */
  zn?: number;
  znm?: string;
}

export interface LeaderboardMsg {
  /** zc: số nhân vật mỗi khu, cho bảng phóng to bằng phím M (spec §8.0). */
  zc?: number[];
  t: 'lb';
  top: { id: number; nm: string; xp: number; L: number; sc: number; ti?: number }[];
  rank: number;
  total: number;
  ms: number; // my Uy Danh (spec §11.1)
  // Minimap markers, the only off-screen positions the client may know (spec §12.1).
  t1?: boolean;
  t1x?: number;
  t1y?: number;
  ch?: boolean;
  chx?: number;
  chy?: number;
  chw?: boolean; // still in the 10s warning
  al?: { x: number; y: number; k: string; w?: boolean }[]; // Đài Tiên Khí (spec §8.5)
}

export interface FeedMsg {
  t: 'feed';
  k: string;
  a: string;
  b?: string;
  r: number;
  tp?: number; // hạng của nạn nhân; > 0 = tin loan toàn server (spec §11.4)
  rm?: string; // phòng xảy ra, chỉ có khi loan toàn server
  /** zn/znm: khu xảy ra chuyện (spec §8.0); tin từ khu khác hiện kèm tên khu. */
  zn?: number;
  znm?: string;
}

export interface DeadMsg {
  t: 'dead';
  killer: string;
  killer_char: string;
  peak_level: number;
  peak_exp: number;
  kills: number;
  survive: number;
  rank: number;
  score: number;
  hits: number;
  burned_exp: number;
  score_by: Record<string, number> | null;
  dai_thua_secs: number;
  top1_secs: number;
}

export interface PongMsg {
  t: 'pong';
  c: number;
  s: number;
}

export interface ErrorMsg {
  t: 'error';
  msg: string;
}

export type ServerMsg = Snapshot | Welcome | LeaderboardMsg | FeedMsg | DeadMsg | PongMsg | ErrorMsg;

// Player flags (netproto.Flag*).
export const FLAG = {
  invuln: 1,
  bush: 2,
  stealth: 4,
  transit: 8,
  dead: 16,
  bot: 32,
  revealed: 64,
  burning: 128, // Thiêu Đốt Tinh Huyết (spec §4.1c)
} as const;

// Buff bit order = server game.BuffNames.
export const BUFFS = ['spin', 'haste', 'magnet', 'hard', 'shield', 'exp2', 'stealth'] as const;

/**
 * MOODS phải khớp **đúng thứ tự** với `game.MoodList` ở server: tâm trạng gửi qua mạng bằng
 * chỉ số. Thêm thì thêm vào CUỐI, đừng chèn giữa.
 */
export const MOODS = [
  'idle', 'run', 'chase', 'flee', 'hit', 'levelup',
  'breakthrough', 'realm_drop', 'teleport_out', 'teleport_in', 'death', 'respawn',
] as const;

/** moodName đổi chỉ số về tên hoạt ảnh; ngoài danh sách thì coi như idle. */
export function moodName(i: number): string {
  return MOODS[i] ?? 'idle';
}

// Map JSON from GET /api/map/{id} (server/internal/world.Map).
export interface Shape {
  c?: boolean;
  x: number;
  y: number;
  w?: number;
  h?: number;
  r?: number;
  /**
   * p: đa giác hữu cơ, toạ độ TUYỆT ĐỐI xen kẽ x,y (spec §8.6). Khi có `p` thì x,y,w,h chỉ
   * là hộp bao. Server xét va chạm trên chính đa giác này nên client phải vẽ y hệt — vẽ
   * khác là người chơi thấy một đằng, luật chạy một nẻo.
   */
  p?: number[];
}

export interface MapObstacle {
  id: number;
  k: string;
  s: Shape; // what you see; swords hit it
  f: Shape; // footprint at the bottom; blocks bodies (spec §8.2)
  hp: number;
}

export interface MapZone {
  id: number;
  k: string;
  s: Shape;
  wd?: number;
}

export interface MapGate {
  id: number;
  x: number;
  y: number;
  n: string;
  c: string;
  /** hv: đây là Thiên Môn — một chiều, luôn ra ở Thiên Đình (spec §8.4a). */
  hv?: boolean;
  /** dk: đây là Bến Thiên Môn ở Thiên Đình — chỗ đặt chân, không đi được. */
  dk?: boolean;
}

export interface GameMap {
  id: string;
  name: string;
  size: number;
  biome: string;
  obstacles: MapObstacle[];
  zones: MapZone[];
  gates: MapGate[];
  decor: { sp: string; x: number; y: number }[];
  /**
   * play_margin: lề tường biên. Vùng chơi được là `[play_margin, size − play_margin]`.
   * Server luôn gửi trường này; client PHẢI dùng nó khi dự đoán va chạm, nếu không sẽ dự
   * đoán ra ngoài vùng cho phép và bị server kéo về mỗi tick (spec §8.2a).
   */
  play_margin: number;
}
