// Bộ điều khiển hoạt ảnh lúc chạy — GAME_SPEC §12.5 C.
//
// Chuẩn hoá frame (B) chữa được chuyện frame lệch nhau, nhưng còn một lớp giật nữa mà art
// không chữa nổi: **cách game chuyển giữa các animation**. Cắt cứng từ `run` sang `chase`
// là một khung hình nhảy hẳn sang dáng khác; reset về frame 0 mỗi lần đổi mood là chân đang
// giữa bước bỗng về tư thế đứng; chân bước đều 12 fps trong khi người lướt nhanh gấp đôi là
// "trượt chân".
//
// Mọi con số trong file này lấy thẳng từ spec §12.5 C. Biên độ cố ý NHỎ và giống nhau cho
// mọi nhân vật — mục đích là tạo cảm giác cùng một "chất", không phải để ai cũng nhận ra có
// hiệu ứng.

/** Một chu kỳ chạy đi được bao nhiêu đơn vị thế giới (độ dài bước × số bước trong chu kỳ). */
const STEP_LEN = 54;
/** Chu kỳ chạy không bao giờ chậm hơn / nhanh hơn ngần này so với fps gốc. */
const CYCLE_MIN = 0.45;
const CYCLE_MAX = 2.2;
/** Crossfade khi đổi animation (spec: 80–120 ms). */
const FADE_SECS = 0.1;
/** Mood giữ tối thiểu trước khi được đổi sang mood khác. */
const MOOD_HOLD = 0.5;
/** Lật trái/phải: co ngang trong ngần này giây rồi giãn lại. */
const FLIP_SECS = 0.08;
/** Chỉ lật khi hướng mới giữ liên tục ngần này giây (chống lật qua lại khi chạy dọc). */
const FLIP_CONFIRM = 0.12;
/** Co ngang tối thiểu giữa chừng cú lật. */
const FLIP_SQUEEZE = 0.2;
/** Nghiêng người tối đa khi rẽ (radian ≈ 6°). */
const LEAN_MAX = 0.105;
/** Squash & stretch khi dừng/đáp đất. */
const SQUASH = 0.06;
/** Biên độ nhún theo bước chạy, tính theo bán kính nhân vật. */
const BOB = 0.035;
/** Thở khi đứng yên: biên độ và nhịp. */
const BREATHE = 0.012;
const BREATHE_HZ = 0.55;

/**
 * cycleAnims dùng CHUNG một pha chu kỳ thân.
 *
 * Spec §12.5 C: "run ↔ chase ↔ flee dùng chung chu kỳ thân nên không reset về frame 0 khi
 * đổi mood". Ba cái này là cùng một chu kỳ chạy với biểu cảm khác nhau; reset pha khi đổi
 * giữa chúng là cho chân đang giữa bước nhảy về tư thế đứng.
 */
const cycleAnims = new Set(['run', 'chase', 'flee']);

export interface AnimOut {
  /** Pha trong chu kỳ, 0..1 — dùng để chọn frame và để đồng bộ nhịp nhún. */
  phase: number;
  /** Hệ số lật ngang: +1 nhìn phải, −1 nhìn trái, giá trị nhỏ = đang giữa cú lật. */
  flipX: number;
  /** Nghiêng người (radian). */
  lean: number;
  /** Nhân vào chiều cao (squash & stretch + thở). */
  scaleY: number;
  /** Dịch đứng theo nhịp nhún, tính theo bán kính nhân vật. */
  bobY: number;
  /** Độ mờ của lớp animation CŨ trong lúc crossfade; 0 = đã chuyển xong. */
  fadeOut: number;
  /** Animation cũ đang mờ dần (rỗng khi không crossfade). */
  prevAnim: string;
}

/**
 * AnimController giữ trạng thái hoạt ảnh của MỘT nhân vật.
 *
 * Tách khỏi `PlayerSprite` để test được bằng số: toàn bộ file này là hàm thuần trên thời
 * gian và vận tốc, không đụng tới PixiJS.
 */
export class AnimController {
  private anim = '';
  private prev = '';
  private fadeLeft = 0;
  private phase = 0;
  private moodAt = -Infinity;
  /** facing: hướng ĐANG vẽ (+1/−1). wantFacing: hướng server báo, chờ xác nhận. */
  private facing = 1;
  private wantFacing = 1;
  private wantSince = -Infinity;
  private flipLeft = 0;
  private lean = 0;
  private squash = 0;
  private lastSpeed = 0;

  /**
   * reset xoá sạch trạng thái, cho View Pool dùng lại controller với một entity khác.
   *
   * Quên gọi là bug im lặng và nặng: controller tưởng mood không đổi nên không báo đổi, thân
   * không được dựng lại, rồi vòng vẽ ném lỗi và cả game đứng.
   */
  reset(): void {
    this.anim = '';
    this.prev = '';
    this.fadeLeft = 0;
    this.phase = 0;
    this.moodAt = -Infinity;
    this.flipLeft = 0;
    this.lean = 0;
    this.squash = 0;
    this.lastSpeed = 0;
  }

  /** current là animation đang vẽ. */
  get current(): string {
    return this.anim;
  }

  /**
   * setMood xin đổi animation. Trả về true nếu đổi thật.
   *
   * Giữ tối thiểu `MOOD_HOLD` giây: server có thể đổi mood liên tục khi nhân vật ở ranh giới
   * (vừa đủ gần để `chase`, vừa đủ xa để `run`), và không giữ thì nhân vật rung giữa hai dáng.
   * Mood `death`/`hit` được đổi ngay — chúng là phản hồi tức thời, trễ nửa giây là sai.
   */
  setMood(mood: string, now: number, urgent = false): boolean {
    if (mood === this.anim) {
      return false;
    }
    if (!urgent && now - this.moodAt < MOOD_HOLD) {
      return false;
    }
    this.prev = this.anim;
    this.fadeLeft = this.anim === '' ? 0 : FADE_SECS;
    this.anim = mood;
    this.moodAt = now;
    // Giữ pha khi đi giữa ba chu kỳ chạy; mọi chuyển khác bắt đầu lại từ đầu.
    if (!(cycleAnims.has(this.prev) && cycleAnims.has(mood))) {
      this.phase = 0;
    }
    return true;
  }

  /** setFacing nhận hướng server báo; cú lật chỉ bắt đầu sau khi hướng giữ đủ lâu. */
  setFacing(faceRight: boolean, now: number): void {
    const want = faceRight ? 1 : -1;
    if (want !== this.wantFacing) {
      this.wantFacing = want;
      this.wantSince = now;
      return;
    }
    if (want !== this.facing && now - this.wantSince >= FLIP_CONFIRM && this.flipLeft <= 0) {
      this.flipLeft = FLIP_SECS;
    }
  }

  /**
   * step chạy một khung hình.
   *
   * `speed` là tốc độ THẬT lấy từ vị trí đã nội suy, không phải từ gói tin thô — spec §12.5 C
   * ("không giật theo mạng"). Lấy từ gói thô thì mỗi lần mạng nấc một cái là chân nhân vật
   * giật theo.
   */
  step(dt: number, speed: number, turnRate: number, baseFps: number, now: number): AnimOut {
    void now;
    dt = Math.min(dt, 0.1); // một khung hình treo lâu không được làm nhân vật nhảy cóc

    // Nhịp chu kỳ theo tốc độ thật: chân bước nhanh đúng bằng mức người di chuyển nhanh.
    // Kẹp hai đầu vì tốc độ có thể về 0 (đứng yên) hoặc vọt lên khi Thiêu Đốt / trượt băng.
    let rate = 1;
    if (cycleAnims.has(this.anim) && STEP_LEN > 0) {
      rate = Math.min(CYCLE_MAX, Math.max(CYCLE_MIN, speed / STEP_LEN));
    }
    this.phase = (this.phase + dt * (baseFps / 8) * rate) % 1;

    if (this.fadeLeft > 0) {
      this.fadeLeft = Math.max(0, this.fadeLeft - dt);
    }

    // Lật: co ngang về FLIP_SQUEEZE rồi giãn lại, đổi hướng đúng giữa cú lật.
    let flipX = this.facing;
    if (this.flipLeft > 0) {
      this.flipLeft = Math.max(0, this.flipLeft - dt);
      const t = 1 - this.flipLeft / FLIP_SECS; // 0 → 1
      if (t >= 0.5) {
        this.facing = this.wantFacing;
      }
      // Hình chữ V: 1 → FLIP_SQUEEZE → 1.
      const k = FLIP_SQUEEZE + (1 - FLIP_SQUEEZE) * Math.abs(t * 2 - 1);
      flipX = this.facing * k;
    }

    // Nghiêng người theo hướng rẽ, tiến dần chứ không nhảy — nhảy thì thành giật mới.
    const wantLean = Math.max(-LEAN_MAX, Math.min(LEAN_MAX, -turnRate * 0.18)) * this.facing;
    this.lean += (wantLean - this.lean) * Math.min(1, dt * 8);

    // Squash & stretch khi tốc độ đổi đột ngột (dừng lại, đáp đất).
    const dv = speed - this.lastSpeed;
    this.lastSpeed = speed;
    if (dv < -40) {
      this.squash = SQUASH;
    }
    this.squash += (0 - this.squash) * Math.min(1, dt * 6);

    // Nhún theo bước chạy, đồng bộ pha chân: hai bước mỗi chu kỳ nên tần số gấp đôi.
    let bobY = 0;
    let scaleY = 1 - this.squash;
    if (cycleAnims.has(this.anim)) {
      bobY = -Math.abs(Math.sin(this.phase * Math.PI * 2)) * BOB;
    } else if (this.anim === 'idle') {
      // Thở nhẹ khi đứng: dùng pha chu kỳ làm đồng hồ nên không cần giữ thêm biến thời gian.
      scaleY += Math.sin(this.phase * Math.PI * 2 * BREATHE_HZ * 8) * BREATHE;
    }

    return {
      phase: this.phase,
      flipX,
      lean: this.lean,
      scaleY,
      bobY,
      fadeOut: this.fadeLeft > 0 ? this.fadeLeft / FADE_SECS : 0,
      prevAnim: this.fadeLeft > 0 ? this.prev : '',
    };
  }
}
