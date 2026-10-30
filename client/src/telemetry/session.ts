// Thông tin phiên: máy gì, GPU gì, mạng gì (TELEMETRY_SPEC §2.1).
//
// Gửi ĐÚNG MỘT LẦN mỗi trận. Mục đích duy nhất của nó là trả lời câu "người bị tụt FPS đang
// dùng máy nào" — không có nó thì mọi con số FPS gộp chung thành một đống vô nghĩa, vì một
// phiên trên máy bàn GPU rời và một phiên trên laptop GPU tích hợp là hai thế giới khác nhau.

export interface SessionInfo {
  id: string;
  gh: string;
  b: string;
  p: number;
  sh: string;
  ua: string;
  br: string;
  os: string;
  mo: boolean;
  co: number;
  me: number;
  sw: number;
  shh: number;
  dpr: number;
  gpu: string;
  wg: number;
  mt: number;
  nt: string;
  dl: number;
  nr: number;
  q: string;
  qa: boolean;
  cw: number;
  ch: number;
}

/** randomId sinh mã phiên. Mỗi lần vào trận một mã mới (§2.1). */
export function randomId(): string {
  const a = new Uint8Array(8);
  crypto.getRandomValues(a);
  return [...a].map((b) => b.toString(16).padStart(2, '0')).join('');
}

/**
 * gpuInfo hỏi tên GPU qua WEBGL_debug_renderer_info.
 *
 * Trình duyệt được phép từ chối (Firefox mặc định chặn vì đây là một mẩu vân tay thiết bị),
 * nên KHÔNG có GPU là chuyện bình thường, không phải lỗi. Báo cáo xếp những phiên đó vào
 * nhóm "PC (không rõ GPU)" thay vì đoán bừa.
 *
 * Tạo canvas riêng rồi vứt ngay: không đụng gì tới ngữ cảnh WebGL của game — mất ngữ cảnh đó
 * là cả màn hình đen, cái giá quá đắt cho một dòng thông tin.
 */
function gpuInfo(): { gpu: string; webgl: number; maxTex: number } {
  const cv = document.createElement('canvas');
  const gl = (cv.getContext('webgl2') ?? cv.getContext('webgl')) as WebGLRenderingContext | null;
  if (!gl) {
    return { gpu: '', webgl: 0, maxTex: 0 };
  }
  let gpu = '';
  const ext = gl.getExtension('WEBGL_debug_renderer_info');
  if (ext) {
    gpu = String(gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) ?? '');
  }
  const maxTex = Number(gl.getParameter(gl.MAX_TEXTURE_SIZE) ?? 0);
  const webgl = cv.getContext('webgl2') ? 2 : 1;
  gl.getExtension('WEBGL_lose_context')?.loseContext();
  return { gpu, webgl, maxTex };
}

/**
 * parseUA rút gọn user-agent còn "trình duyệt + bản" và "hệ điều hành".
 *
 * Rút gọn thay vì lưu nguyên chuỗi: chuỗi đầy đủ vừa dài vừa là vân tay thiết bị, mà báo cáo
 * chỉ cần gom nhóm. Chuỗi gốc vẫn lưu để còn tra lại khi gặp một máy lạ.
 */
function parseUA(ua: string): { br: string; os: string } {
  const br =
    /Edg\/([\d.]+)/.exec(ua)?.[0] ??
    /OPR\/([\d.]+)/.exec(ua)?.[0] ??
    /Firefox\/([\d.]+)/.exec(ua)?.[0] ??
    /Chrome\/([\d.]+)/.exec(ua)?.[0] ??
    /Version\/([\d.]+).*Safari/.exec(ua)?.[0] ??
    'khác';
  const os = /Windows NT [\d.]+/.exec(ua)?.[0]
    ?? /Android [\d.]+/.exec(ua)?.[0]
    ?? /iPhone OS [\d_]+/.exec(ua)?.[0]
    ?? /Mac OS X [\d_]+/.exec(ua)?.[0]
    ?? (/Linux/.test(ua) ? 'Linux' : 'khác');
  return { br: br.split(' ')[0], os };
}

interface NetworkInfo {
  effectiveType?: string;
  downlink?: number;
  rtt?: number;
}

/** collectSession gom toàn bộ §2.1. Không ném lỗi: thiếu trường nào thì để trống. */
export function collectSession(opts: {
  id: string;
  build: string;
  proto: number;
  schema: string;
  quality: string;
  qualityAuto: boolean;
  canvas: { w: number; h: number };
}): SessionInfo {
  const ua = navigator.userAgent;
  const { br, os } = parseUA(ua);
  const { gpu, webgl, maxTex } = gpuInfo();
  const conn = (navigator as Navigator & { connection?: NetworkInfo }).connection ?? {};
  return {
    id: opts.id,
    // guest_hash do SERVER băm từ guest_id nó đã biết — client không gửi, để hai bản build
    // khác nhau không băm ra hai kiểu rồi không ghép được phiên của cùng một người.
    gh: '',
    b: opts.build,
    p: opts.proto,
    sh: opts.schema,
    ua: ua.slice(0, 300),
    br,
    os,
    mo: /Android|iPhone|iPad|Mobile/.test(ua),
    co: navigator.hardwareConcurrency ?? 0,
    me: (navigator as Navigator & { deviceMemory?: number }).deviceMemory ?? 0,
    sw: screen.width,
    shh: screen.height,
    dpr: window.devicePixelRatio,
    gpu: gpu.slice(0, 120),
    wg: webgl,
    mt: maxTex,
    nt: conn.effectiveType ?? '',
    dl: conn.downlink ?? 0,
    nr: conn.rtt ?? 0,
    q: opts.quality,
    qa: opts.qualityAuto,
    cw: opts.canvas.w,
    ch: opts.canvas.h,
  };
}
