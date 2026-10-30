/** h creates an element with a class and optional text/children. */
export function h<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  cls = '',
  ...children: (Node | string)[]
): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  if (cls) {
    el.className = cls;
  }
  for (const c of children) {
    el.append(c);
  }
  return el;
}

export function button(label: string, cls: string, onClick: () => void): HTMLButtonElement {
  const b = h('button', cls, label);
  b.addEventListener('click', onClick);
  return b;
}

/** Screen shows exactly one overlay at a time. */
export class Screens {
  private current: HTMLElement | null = null;

  constructor(private readonly root: HTMLElement) {}

  show(el: HTMLElement | null): void {
    this.current?.remove();
    this.current = el;
    if (el) {
      this.root.appendChild(el);
    }
  }
}

/** portrait shows the character's first idle frame, or a coloured circle placeholder. */
/**
 * portrait dựng ảnh nhân vật cho màn chọn.
 *
 * Ưu tiên `portrait_neutral.png` — ảnh CHÂN DUNG vẽ riêng cho việc này, mặt chiếm gần hết
 * khung. Bản trước dùng `frames/idle_00.png`, là một khung hình toàn thân trên canvas 512
 * với nhân vật chỉ chiếm 72% và chân ghim ở y=460: thu về ô 40px thì người bé bằng hạt gạo
 * giữa một vùng trong suốt, nhìn như thẻ trống (ảnh chụp 30/10 — mười thẻ đều trắng trơn).
 *
 * Chưa có chân dung thì chấm tròn màu — để màn chọn không bao giờ trống trơn khi art chưa kịp.
 */
export function portrait(char: string, has: (p: string) => boolean, color: string): HTMLElement {
  // Chỉ dùng chân dung. Mức dự phòng cũ là `frames/idle_00.png`, nhưng `frames/` là ảnh
  // nguồn: không có trong manifest, không được deploy → 404 và ô trống. Thiếu chân dung thì
  // rơi thẳng xuống chấm tròn màu bên dưới, vẫn không bao giờ trống trơn.
  const file = has(`characters/${char}/portrait_neutral.png`) ? `portrait_neutral.png` : '';
  if (file) {
    const img = h('img', 'portrait');
    img.src = `/assets/characters/${char}/${file}`;
    img.alt = '';
    return img;
  }
  const d = h('div', 'portrait placeholder');
  d.style.background = color;
  return d;
}
