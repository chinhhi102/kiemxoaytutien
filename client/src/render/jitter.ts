// Điểm "độ giật" đọc từ `assets/_raw/jitter.json` để phím debug `J` hiện được
// (GAME_SPEC §12.5 B.5 và C dòng cuối).
//
// Điểm do `cd server && go run ./cmd/artgen jitter` tính sẵn lúc build — client KHÔNG tự
// tính. Tính ở client nghĩa là mỗi máy người chơi đọc lại ~900 ảnh mỗi lần vào game, và con
// số thì giống hệt nhau ở mọi máy vì nó chỉ phụ thuộc vào art.

/** JITTER_RED: từ điểm này trở lên hiện chữ đỏ. Khớp `jitterRed` bên artgen. */
export const JITTER_RED = 45;

interface AnimScore {
  char: string;
  anim: string;
  loop: boolean;
  avg: number;
  worst: number;
  outlier: number;
}

/** JitterEntry là một dòng trong bảng điểm, dùng cho bộ xem `[` / `]`. */
export interface JitterEntry {
  char: string;
  anim: string;
  loop: boolean;
  score: number;
}

/** scores[char][anim] = điểm. Rỗng cho tới khi `loadJitter()` chạy xong. */
const scores = new Map<string, number>();
/** ordered: cùng dữ liệu nhưng XẾP THEO ĐIỂM giảm dần — thứ tự đi của phím `[` / `]`. */
let ordered: JitterEntry[] = [];
let loaded = false;

function key(char: string, anim: string): string {
  return `${char}/${anim}`;
}

/**
 * loadJitter tải bảng điểm. Chỉ gọi khi người chơi bấm `J` lần đầu.
 *
 * Tải theo yêu cầu chứ không tải lúc khởi động: đây là dữ liệu dành cho người đang sửa art,
 * người chơi bình thường không bao giờ cần — và mọi byte tải lúc vào game đều làm chậm đúng
 * cái lúc nhạy cảm nhất.
 */
export async function loadJitter(): Promise<void> {
  if (loaded) {
    return;
  }
  loaded = true; // đặt trước: hỏng thì cũng đừng thử lại mỗi lần bấm phím
  try {
    const res = await fetch('/assets/jitter.json', { cache: 'no-cache' });
    if (!res.ok) {
      return;
    }
    const data = (await res.json()) as { anims: AnimScore[] };
    for (const a of data.anims ?? []) {
      // Anim lặp đọc theo điểm trung bình, anim một lần đọc theo điểm "lạc lõng" — đúng như
      // bảng của `artgen jitter`, vì hai nhóm không so được với nhau bằng cùng một con số.
      const score = a.loop ? a.avg : a.outlier;
      scores.set(key(a.char, a.anim), score);
      ordered.push({ char: a.char, anim: a.anim, loop: a.loop, score });
    }
    // Tệ nhất lên đầu: bấm `]` lần đầu là thấy ngay con đáng xem nhất, không phải lướt qua
    // ba chục animation ổn trước khi tới cái cần xem.
    ordered.sort((x, y) => y.score - x.score);
  } catch {
    // Không có file (chưa chạy `artgen jitter`) thì phím J vẫn chạy, chỉ không có cột điểm.
  }
}

/** animList trả về toàn bộ bảng điểm đã xếp hạng, cho bộ xem `[` / `]`. */
export function animList(): JitterEntry[] {
  return ordered;
}

/** animJitter trả về điểm của một animation, hoặc null nếu chưa có số liệu. */
export function animJitter(char: string, anim: string): number | null {
  return scores.get(key(char, anim)) ?? null;
}
