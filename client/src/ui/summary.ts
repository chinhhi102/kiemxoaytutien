import type { GameConfig } from '../config';
import { realmOf, tierOf } from '../config';
import type { DeadMsg } from '../net/protocol';
import { assets } from '../render/assets';
import { button, h } from './dom';
import { pick, STR } from './strings';

export interface SummaryActions {
  playAgain: () => void;
  leaderboard: () => void;
  changeCharacter: () => void;
}

/** summaryScreen is the Tổng kết screen after death (spec §7). */
export function summaryScreen(cfg: GameConfig, d: DeadMsg, myChar: string, act: SummaryActions): HTMLElement {
  const root = h('div', 'screen summary');
  const panel = h('div', 'panel scroll-panel');
  if (assets.has('ui/scroll.png')) {
    panel.classList.add('has-scroll');
  }
  // Hồn ma dùng CHÂN DUNG, không dùng `frames/death_07.png`.
  //
  // `frames/` là ảnh NGUỒN để ghép atlas — nó không có trong `manifest.json` và không được
  // deploy (hơn 100 MB mà game không đọc tới). Trước đây đường dẫn này chạy được chỉ vì
  // server phục vụ thẳng thư mục làm việc; từ lúc có `tools/deploy.sh` thì nó 404 và màn
  // Tổng kết mất hình. Chân dung khóc hợp cảnh hơn và luôn có mặt.
  const ghostFile = assets.has(`characters/${myChar}/portrait_cry.png`) ? 'portrait_cry.png'
    : assets.has(`characters/${myChar}/portrait_neutral.png`) ? 'portrait_neutral.png'
    : '';
  const ghost = ghostFile ? h('img', 'ghost') : null;
  if (ghost) {
    ghost.src = `/assets/characters/${myChar}/${ghostFile}`;
    ghost.onerror = () => ghost.remove();
  }
  const realm = cfg.cultivation.realms[realmOf(cfg, d.peak_level)] ?? '';
  const stat = (label: string, value: string): HTMLElement => h('div', 'stat', h('span', '', label), h('b', '', value));
  const killer = d.killer ? `${STR.killedBy(d.killer)} (${STR.characters[d.killer_char] ?? d.killer_char})` : STR.killedByUnknown;
  const num = (n: number): string => Math.round(n).toLocaleString('vi-VN');
  panel.append(
    h('h2', '', STR.summaryTitle),
    ...(ghost ? [ghost] : []),
    h('div', 'killer', killer),
    h('div', 'joke', pick(STR.deathJokes)),
    h('div', 'score-total', h('span', '', STR.score), h('b', '', num(d.score ?? 0))),
    ...scoreBreakdown(d, num),
    stat(STR.peakRealm, STR.realmTier(realm, tierOf(cfg, d.peak_level))),
    stat(STR.peakExp, num(d.peak_exp)),
    stat(STR.hits, String(d.hits ?? 0)),
    ...(d.burned_exp > 0 ? [stat(STR.burnSpent, num(d.burned_exp))] : []),
    stat(STR.kills, String(d.kills)),
    stat(STR.toDaiThua, d.dai_thua_secs > 0 ? STR.secs(d.dai_thua_secs) : STR.notReached),
    stat(STR.top1Held, STR.secs(d.top1_secs ?? 0)),
    stat(STR.survive, STR.secs(d.survive)),
    stat(STR.roomRank, `#${d.rank}`),
    h('div', 'row', button(STR.playAgain, 'btn primary', act.playAgain), button(STR.leaderboard, 'btn', act.leaderboard), button(STR.changeCharacter, 'btn', act.changeCharacter)),
  );
  root.append(panel);
  return root;
}

/** scoreBreakdown lists Uy Danh per source, biggest first (spec §11.1). */
function scoreBreakdown(d: DeadMsg, num: (n: number) => string): HTMLElement[] {
  const by = Object.entries(d.score_by ?? {}).filter(([, v]) => v >= 1);
  if (!by.length) {
    return [];
  }
  by.sort((a, b) => b[1] - a[1]);
  return [h('div', 'score-by', ...by.map(([k, v]) => h('div', 'score-line', h('span', '', STR.scoreSources[k] ?? k), h('b', '', `+${num(v)}`))))];
}
