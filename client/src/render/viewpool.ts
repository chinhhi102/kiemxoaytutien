// View Pool — NETCODE_SPEC §5 "View Pool" và mục Hiển thị 4.
//
// Luật của spec: "Khi `Despawn` → view vào pool (ẩn, giữ texture), không hủy. `Spawn` lấy
// view từ pool. Không tạo/hủy object trong lúc chơi."
//
// Vì sao đáng làm: một `PlayerSprite` không phải một sprite — nó là thân, vòng kiếm (tới
// hàng trăm thanh), bảng tên, bóng đổ, hào quang, vòng icon trạng thái. Dựng và huỷ cả cụm
// đó mỗi lần có người đi qua mép khung nhìn là vừa tốn CPU vừa rải rác cho bộ dọn rác —
// và bộ dọn rác chạy giữa trận chính là một cú khựng nhìn thấy được.

import type { GameConfig } from '../config';
import { PlayerSprite } from './player';

/** FADE_OUT_SECS: mờ dần ra trước khi về pool (spec: 150 ms). */
export const FADE_OUT_SECS = 0.15;

/**
 * POOL_MAX: giữ tối đa ngần này view rỗi.
 *
 * Phòng đông nhất là 60 nhân vật, nên 24 view rỗi đã thừa cho lượng người ra vào mép khung
 * nhìn. Không để vô hạn: một trận dài đi qua hàng nghìn entity, giữ hết thì pool biến thành
 * rò rỉ bộ nhớ mang tên khác.
 */
const POOL_MAX = 24;

export class ViewPool {
  private readonly free: PlayerSprite[] = [];

  /** size là số view đang rỗi (overlay F3 đọc). */
  get size(): number {
    return this.free.length;
  }

  /**
   * take lấy một view ra dùng lại, hoặc dựng mới nếu pool rỗng.
   *
   * View dùng lại phải được ĐẶT LẠI hoàn toàn: id, tên, nhân vật, độ mờ. Quên một thứ là
   * nhân vật mới hiện ra mang tên người cũ — lỗi nhìn thấy ngay và khó lần ra.
   */
  take(cfg: GameConfig, id: number, name: string, char: string, isMe: boolean): PlayerSprite {
    const sp = this.free.pop();
    if (!sp || !sp.reuse(id, name, char, isMe)) {
      // `reuse` trả false khi view cũ không đổi sang nhân vật này được (thiếu atlas). Lúc
      // đó dựng mới — thà tốn một lần dựng còn hơn vẽ sai nhân vật.
      if (sp) {
        sp.destroy();
      }
      return new PlayerSprite(cfg, id, name, char, isMe);
    }
    return sp;
  }

  /** put cất một view đã rời khung hình. Quá sức chứa thì huỷ hẳn. */
  put(sp: PlayerSprite): void {
    if (this.free.length >= POOL_MAX) {
      sp.destroy();
      return;
    }
    sp.park();
    this.free.push(sp);
  }

  /** clear huỷ sạch pool — gọi khi đổi phòng/khu, vì nhân vật của khu cũ không còn liên quan. */
  clear(): void {
    for (const sp of this.free) {
      sp.destroy();
    }
    this.free.length = 0;
  }
}
