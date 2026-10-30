import { Assets, Rectangle, Texture } from 'pixi.js';

import { CHARACTER_ANIMS } from './catalog';

// Asset paths follow docs/ART_PROMPTS.md §13. Anything missing falls back to placeholders,
// so dropping a PNG into assets/ and re-running `artgen prep` is enough to use real art.

interface AtlasJson {
  cell: number;
  pivot_x: number;
  pivot_y: number;
  frames: Record<string, { x: number; y: number; w: number; h: number }>;
  anims: Record<string, { frames: string[]; fps: number; loop: boolean }>;
}

/**
 * miscut nhận ra hoạt ảnh bị cắt sai số khung.
 *
 * Bộ cắt của artgen tìm khung bằng vùng pixel LIỀN NHAU, nên nhân vật nào có tay/chân/đồ vật
 * tách rời thân là bị đếm thành nhiều khung. Đã xảy ra thật: `dai_ca_dep` idle ra 22 khung
 * thay vì 4 và trong game thành một cái áo tím nhấp nháy. Thà bỏ hoạt ảnh hỏng và rơi về
 * hoạt ảnh thay thế còn hơn vẽ ra thứ đó.
 */
function miscut(name: string, got: number): boolean {
  const want = CHARACTER_ANIMS.find((a) => a.name === name)?.frames;
  return want !== undefined && (got > want * 1.5 || got * 2 < want);
}

export interface AnimSet {
  anims: Map<string, { textures: Texture[]; fps: number; loop: boolean }>;
  pivotX: number;
  pivotY: number;
}

class AssetStore {
  private files = new Set<string>();
  private textures = new Map<string, Texture>();
  private atlases = new Map<string, AnimSet>();

  async init(): Promise<void> {
    try {
      const res = await fetch('/assets/manifest.json', { cache: 'no-cache' });
      if (res.ok) {
        const m = (await res.json()) as { files: string[] };
        this.files = new Set(m.files);
      }
    } catch (err) {
      console.warn('no asset manifest, using placeholders', err);
    }
  }

  has(path: string): boolean {
    return this.files.has(path);
  }

  /**
   * charAtlas lấy atlas nhân vật, có NHÂN VẬT DỰ PHÒNG (NETCODE_SPEC §5 Hiển thị mục 1).
   *
   * Build thật CẤM placeholder. Thiếu atlas của một nhân vật thì vẽ bằng nhân vật dự phòng
   * — art thật, nhìn vẫn ra một người — chứ không vẽ vòng tròn màu kèm chữ viết tắt.
   *
   * Lý do cấm: vòng tròn placeholder là công cụ của người làm game, không phải thứ người
   * chơi nên thấy. Ngày 30/10 telemetry đo được 12 khung placeholder trên server thật, và
   * người chơi không có cách nào biết đó là "art chưa xong" hay "game hỏng".
   *
   * Trả về null chỉ khi KHÔNG CÓ nhân vật nào có atlas — lúc đó mới rơi về placeholder, và
   * khi đó cả game đang thiếu art nên vòng tròn là đúng thứ cần hiện.
   */
  charAtlas(char: string): { set: AnimSet; fallback: boolean } | null {
    const want = this.atlas(`characters/${char}`);
    if (want) {
      return { set: want, fallback: false };
    }
    for (const alt of this.fallbackChars) {
      const set = this.atlas(`characters/${alt}`);
      if (set) {
        if (!this.warnedChars.has(char)) {
          this.warnedChars.add(char);
          console.warn(`thiếu atlas nhân vật "${char}" — dùng tạm "${alt}" (NETCODE_SPEC §5.1)`);
        }
        return { set, fallback: true };
      }
    }
    return null;
  }

  /** fallbackChars: thứ tự thử nhân vật dự phòng. Đặt bởi `setFallbackChars` lúc có config. */
  private fallbackChars: string[] = [];
  private readonly warnedChars = new Set<string>();

  /** setFallbackChars nhận danh sách nhân vật từ config (thứ tự trong config là thứ tự thử). */
  setFallbackChars(chars: string[]): void {
    this.fallbackChars = chars;
  }

  /** Loads every listed single image plus atlases. Missing files are simply skipped. */
  async preload(onProgress?: (p: number) => void): Promise<void> {
    const singles = [...this.files].filter((f) => f.endsWith('.png') && !f.endsWith('atlas.png') && !f.includes('ref_sheet'));
    const atlases = [...this.files].filter((f) => f.endsWith('atlas.json'));
    const total = singles.length + atlases.length;
    let done = 0;
    const tick = (): void => {
      done++;
      onProgress?.(total ? done / total : 1);
    };
    await Promise.all([
      ...singles.map(async (f) => {
        await this.loadTexture(f);
        tick();
      }),
      ...atlases.map(async (f) => {
        await this.loadAtlas(f);
        tick();
      }),
    ]);
  }

  private async loadTexture(path: string): Promise<Texture | null> {
    try {
      const tex = (await Assets.load(`/assets/${path}`)) as Texture;
      this.textures.set(path, tex);
      return tex;
    } catch (err) {
      console.warn('texture failed', path, err);
      return null;
    }
  }

  private async loadAtlas(jsonPath: string): Promise<void> {
    const dir = jsonPath.slice(0, jsonPath.lastIndexOf('/'));
    try {
      const res = await fetch(`/assets/${jsonPath}`);
      const atlas = (await res.json()) as AtlasJson;
      const base = await this.loadTexture(`${dir}/atlas.png`);
      if (!base) {
        return;
      }
      const set: AnimSet = { anims: new Map(), pivotX: atlas.pivot_x, pivotY: atlas.pivot_y };
      for (const [name, a] of Object.entries(atlas.anims)) {
        if (miscut(name, a.frames.length)) {
          // Sheet bị cắt sai (xem artgen prep): bỏ hẳn hoạt ảnh này để nhân vật rơi về
          // hoạt ảnh thay thế, thay vì nhấp nháy mấy mảnh vụn của chính nó.
          console.warn(`atlas ${dir}: bỏ "${name}" vì cắt sai (${a.frames.length} khung)`);
          continue;
        }
        const textures = a.frames.map((f) => {
          const r = atlas.frames[f];
          return new Texture({ source: base.source, frame: new Rectangle(r.x, r.y, r.w, r.h) });
        });
        set.anims.set(name, { textures, fps: a.fps, loop: a.loop });
      }
      this.atlases.set(dir, set);
    } catch (err) {
      console.warn('atlas failed', jsonPath, err);
    }
  }

  texture(path: string): Texture | null {
    return this.textures.get(path) ?? null;
  }

  /** Frames like vfx/clash_spark/clash_spark_00.png … in order, or [] if missing. */
  frames(dir: string, anim: string): Texture[] {
    const out: Texture[] = [];
    for (let i = 0; i < 99; i++) {
      const t = this.textures.get(`${dir}/${anim}_${String(i).padStart(2, '0')}.png`);
      if (!t) {
        break;
      }
      out.push(t);
    }
    return out;
  }

  atlas(dir: string): AnimSet | null {
    return this.atlases.get(dir) ?? null;
  }
}

export const assets = new AssetStore();
