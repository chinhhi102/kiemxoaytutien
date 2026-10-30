// Every piece of text shown to players lives here (CLAUDE.md rule 5).

export const STR = {
  title: 'Kiếm Xoay Tu Tiên',
  subtitle: 'Dân phố tấu hài tu tiên bằng kiếm bay',
  namePlaceholder: 'Nhập tên (2–16 ký tự)',
  pickCharacter: 'Chọn nhân vật',
  play: 'Vào chơi',
  leaderboard: 'Bảng xếp hạng',
  findingRoom: 'Đang tìm phòng...',
  connecting: 'Đang kết nối máy chủ...',
  /** Tên 5 khu, khớp với room.ZoneNames bên server (spec §8.0). */
  /** Tên 10 kỹ năng (spec §6.3), khớp key nhân vật bên server. */
  skillNames: {
    a_hon: 'Bánh Bao Thần',
    ba_tam_loa: 'Loa Phường',
    ong_ba_vong: 'Ru Ngủ',
    dai_ca_dep: 'Gọi Đàn Em',
    map_tra_sua: 'Phun Trân Châu',
    co_ba_ganh: 'Gánh Lướt',
    su_phu_mi: 'Lưới Mì',
    be_ti: 'Gồng Cơ',
    shipper: 'Giao Hỏa Tốc',
    cu_rua: 'Rụt Cổ',
  } as Record<string, string>,
  /**
   * Mô tả kỹ năng ở màn chọn nhân vật — người chơi phải biết mình chọn gì trước khi vào trận.
   *
   * Viết theo LỢI ÍCH chứ không theo cơ chế: "chạy xuyên tường" dễ hiểu hơn "StGhost 5s".
   * Con số lấy đúng spec v5.4/v5.5 §6.3; đổi spec thì sửa ở đây luôn, nếu không người chơi
   * chọn theo một bảng mô tả đã cũ.
   */
  skillDesc: {
    a_hon: 'Ăn bánh bao, hồi ngay 20% số kiếm tối đa của cảnh giới.',
    ba_tam_loa: 'Hét một tiếng: đẩy văng địch phía trước 500 và làm vòng kiếm của họ quay ngược.',
    ong_ba_vong: 'Hát ru: mọi địch quanh 500 chậm 40%, vòng kiếm quay chậm một nửa trong 3 giây.',
    dai_ca_dep: 'Gọi 2 đàn em trong 10 giây, mỗi đứa mang nửa số kiếm của mình — mình không mất kiếm nào.',
    map_tra_sua: 'Phun trân châu bịt mắt địch 5 giây; trong 5 giây đó họ không nhặt được gì.',
    co_ba_ganh: 'Lướt 600 và rơi trái cây: ai giẫm phải thì trượt và choáng 1 giây.',
    su_phu_mi: 'Ném lưới mì trói chặt địch 5 giây — chúng đứng yên, chỉ còn kiếm quay.',
    be_ti: 'Gồng 5 giây: kiếm cứng gấp đôi, miễn mọi đòn đẩy, chậm, trói và choáng.',
    shipper: 'Chạy nhanh gấp 3 trong 5 giây và đi xuyên qua mọi vật cản.',
    cu_rua: 'Rụt vào mai 5 giây: không bể kiếm, không bị chém; ra khỏi mai thì bật sóng đẩy.',
  } as Record<string, string>,
  skillLabel: 'Kỹ năng',
  /** Câu thoại tấu hài khi dùng kỹ năng (spec §6.3). */
  skillLines: {
    a_hon: 'Ăn miếng bánh bao, kiếm về như mới!',
    ba_tam_loa: 'A lô a lô, tránh ra cho bà đi!',
    ong_ba_vong: 'À ơi… ngủ đi con…',
    dai_ca_dep: 'Tụi bay đâu, lên đây!',
    map_tra_sua: 'Nếm thử trân châu nhà tao!',
    co_ba_ganh: 'Tránh đường, hàng nóng!',
    su_phu_mi: 'Mì gói ba phút, trói mày ba giây!',
    be_ti: 'Gồng! Không đứa nào đẩy nổi!',
    shipper: 'Đơn gấp, xin phép đi xuyên!',
    cu_rua: 'Thiền… các ngươi cứ chém đi.',
  } as Record<string, string>,
  zoneNames: ['Thiên Đình', 'Linh Sơn Cốc', 'Trúc Lâm Mê Cung', 'Phàm Trấn', 'Băng Nguyệt Hồ'],
  zoneListTitle: 'Khu vực',
  zoneEntered: (name: string): string => `— ${name} —`,
  glLost: 'Card màn hình vừa bị gián đoạn. Đang khôi phục, bạn sẽ vào lại phòng ngay…',
  glLostManual: 'Đồ hoạ lỗi liên tục. Hãy hạ chất lượng bằng phím V rồi tải lại trang (Ctrl+F5).',
  versionMismatch: 'Máy chủ vừa được cập nhật. Hãy tải lại trang (Ctrl+F5) để chơi tiếp.',
  disconnected: 'Mất kết nối với máy chủ. Đang thử lại...',
  badName: 'Tên phải dài 2–16 ký tự.',
  controlsPc: 'Di chuyển: WASD / phím mũi tên / giữ chuột. Thiêu Đốt Tinh Huyết: giữ Shift, hoặc nháy đúp chuột rồi giữ. Kỹ năng: Space. Không có nút đánh — kiếm tự xoay!',
  animInfoOn: 'Hiện số liệu hoạt ảnh (J): tên anim · frame · pha · điểm độ giật',
  animInfoOff: 'Tắt số liệu hoạt ảnh',
  qualitySet: (name: string) => `Chất lượng đồ họa: ${name} (phím V để đổi)`,
  controlsMobile: 'Kéo ngón tay để di chuyển. Thiêu Đốt Tinh Huyết: chạm 2 lần rồi giữ, hoặc giữ nút 🔥. Kiếm tự xoay!',

  // Thiêu Đốt Tinh Huyết (spec §4.1c)
  burnName: 'Thiêu Đốt Tinh Huyết',
  burnHint: (mult: number) => `Giữ Shift hoặc nháy đúp rồi giữ: chạy nhanh ×${mult}, đốt tu vi mỗi giây`,
  skillBtnHint: (isTouch: boolean) => (isTouch ? 'Kỹ năng — chạm để dùng' : 'Kỹ năng — bấm Space hoặc nhấn vào đây'),
  burnSpent: 'Tu vi đã thiêu đốt',
  burnEmpty: 'Cạn tinh huyết rồi!',
  debugHelp: 'Phím test: E +100 EXP · R lên cảnh giới · K +10 kiếm · G +1 lượt kỹ năng · B thả 1 bot cạnh mình · 1–7 buff · X tự chết · H xem hitbox kiếm',

  // HUD
  realmTier: (realm: string, tier: number) => `${realm} tầng ${tier}`,
  // Thanh tiến độ đổi tên theo cảnh giới (spec §3.4): phàm = Linh khí, Tiên giới = Tiên khí.
  linhKhi: 'Linh khí',
  tienKhi: (have: number, need: number) => `Tiên khí ${Math.floor(have)}/${Math.round(need)}`,
  tienMax: 'Đỉnh cao Tiên giới',
  /** Đạo Tổ đã nạp đầy 1 500: từ đây Tiên khí nhặt thêm mới đổi Uy Danh (spec v5.7 §3.4). */
  tienOverflow: 'Tiên khí dư → Uy Danh',
  swords: (n: number, max: number) => `Kiếm ${n}/${max}`,
  roomTop: 'Top phòng',
  yourRank: (rank: number, total: number) => `Hạng của bạn: ${rank}/${total}`,
  ping: (ms: number) => `Ping ${ms}ms`,
  chestIncoming: (secs: number) => `Rương Báu sắp xuất hiện trong ${secs}s!`,
  maxLevel: 'Đỉnh cao tu vi',

  // Death / summary screen (spec §7)
  summaryTitle: 'Tổng kết',
  killedBy: (name: string) => `Bị ${name} chém`,
  killedByUnknown: 'Bị chém bởi một cao thủ vô danh',
  peakRealm: 'Cảnh giới cao nhất',
  peakExp: 'Tổng EXP đỉnh',
  kills: 'Số người đã chém',
  survive: 'Thời gian sống',
  roomRank: 'Hạng trong phòng',
  playAgain: 'Chơi tiếp',
  changeCharacter: 'Đổi nhân vật',
  back: 'Quay lại',
  deathJokes: [
    'Bạn đã được tiễn về quê nuôi cá.',
    'Kiếm gãy, người bay, mộng tu tiên tạm hoãn.',
    'Tu tiên không thành, về bán bánh bao tiếp.',
    'Ăn một nhát kiếm, ngộ ra chân lý: chạy nhanh hơn.',
    'Đạo tâm vỡ vụn như bánh tráng.',
    'Thôi thì kiếp sau làm quái cho đỡ khổ.',
    'Tu được ba hôm, nằm viện ba tháng.',
    'Cảnh giới cao nhất bạn đạt được: cảnh giới nằm.',
    'Người ta phi thăng, bạn phi ra khỏi bản đồ.',
    'Sư phụ dặn né, bạn đứng lại ngắm.',
    'Độ kiếp thất bại ngay tại tầng gửi xe.',
    'Tu vi của bạn: đang trong quá trình tích lũy lại từ đầu.',
    'Kiếm người ta xoay, kiếm bạn xoay xuống đất.',
    'Linh khí trời đất không cứu nổi tốc độ phản xạ của bạn.',
    'Chốt đơn: một vé về quê, không hoàn không huỷ.',
    'Bạn đi rồi, bản đồ thoáng hẳn ra.',
    'Thọ nguyên còn dài, nhưng kiếm người ta dài hơn.',
    'Có cái nịt, không có cái mạng.',
    'Đạo hữu nằm xuống, nhạc nền nổi lên.',
    'Hẹn gặp lại ở quán trà đá cuối thôn.',
    'Thiên đạo nhắn: "Thử lại sau nhé."',
    'Bế quan mãi mãi, không ai làm phiền nữa.',
  ],
  secs: (s: number) => {
    const m = Math.floor(s / 60);
    const r = Math.floor(s % 60);
    return m > 0 ? `${m} phút ${r} giây` : `${r} giây`;
  },

  // Global leaderboard (spec §11.2)
  lbTitle: 'Bảng xếp hạng',
  lbToday: 'Hôm nay',
  lbWeek: 'Tuần này',
  lbAll: 'Mọi thời đại',
  lbScore: 'Uy Danh',
  lbKills: 'Sát Thần',
  lbSpeed: 'Thần Tốc',
  lbEmpty: 'Chưa có ai ghi danh. Vào chém đi!',
  lbEmptySpeed: 'Chưa ai lên tới Đại Thừa. Cơ hội của bạn đây!',
  lbName: 'Tên',
  lbRealm: 'Cảnh giới',
  lbLoading: 'Đang tải...',
  lbError: 'Không tải được bảng xếp hạng.',

  // Uy Danh (spec §11.1)
  score: 'Uy Danh',
  scoreHud: (n: number) => `Uy Danh ${Math.round(n).toLocaleString('vi-VN')}`,
  scorePop: (n: number) => `+${Math.round(n).toLocaleString('vi-VN')}`,
  scoreUpset: 'LẬT KÈO!',
  scoreSources: {
    hit: 'Chém trúng người',
    kill: 'Chém chết',
    breakthrough: 'Đột phá cảnh giới',
    top1: 'Giữ ngôi Top 1',
  } as Record<string, string>,
  hits: 'Số lần chém trúng',
  toDaiThua: 'Thời gian lên Đại Thừa',
  top1Held: 'Thời gian giữ Top 1',
  notReached: 'Chưa đạt',

  // Danh hiệu Top 1–10 trong phòng (spec §11.3). Thứ tự khớp config `titles`.
  titles: [
    'Bần tăng chưa ngán ai bao giờ',
    'Ngon thì nhào vô',
    'Chỗ này nhiều máu nè',
    'Chưa gặp đối thủ xứng tầm',
    'Đi nhẹ nói khẽ, chém hơi mạnh',
    'Cũng có số má đấy chứ',
    'Tạm gọi là biết múa kiếm',
    'Đang cố gắng từng ngày',
    'Thấy ai cũng chào trước',
    'Nấp giỏi nên còn sống',
  ],

  // Kill feed (spec §7) — mỗi dòng bốc ngẫu nhiên một cách nói cho đỡ nhàm
  feed: {
    /** Top 3 bước qua Thiên Môn (spec §8.4a). */
    heaven_gate: (a: string, title: string): string => `${a} (${title}) vừa bước qua Thiên Môn — Thiên Đình dậy sóng!`,
    kill: (a: string, b: string) => pick([
      `${a} vừa tiễn ${b} về quê nuôi cá`,
      `${a} tặng ${b} một vé về quê, không hoàn không huỷ`,
      `${b} bị ${a} chém bay cả giấc mộng tu tiên`,
      `${a} dọn dẹp ${b} gọn gàng sạch sẽ`,
      `${b} nằm xuống, ${a} phủi tay đi tiếp`,
      `${a} vừa cho ${b} bế quan vĩnh viễn`,
    ]),
    realm_drop: (a: string, b: string, realm: string) => pick([
      `${b} bị ${a} chém rớt xuống ${realm}`,
      `${a} đạp ${b} về lại ${realm}`,
      `${b} tu bao ngày, về ${realm} trong một nhát của ${a}`,
      `${a} dạy ${b} một bài: ${realm} thôi nhé`,
      `${b} rớt xuống ${realm}, công sức bay theo gió`,
    ]),
    breakthrough: (a: string, realm: string) => pick([
      `${a} đột phá ${realm}, cả phòng run rẩy`,
      `${a} lên ${realm} rồi, ai nhanh chân thì chạy`,
      `Linh khí dồn về một chỗ: ${a} chạm ${realm}`,
      `${a} bước vào ${realm}, trời đất đổi màu`,
      `${a} vừa lên ${realm} — tuyệt đối điện ảnh`,
    ]),
    // Tin loan toàn server khi hạ một người trong Top 5 của phòng họ (spec §11.4).
    serverTag: (rank: number) => `📢 TOÀN CÕI TU CHÂN · hạ Top ${rank}: `,
    tien_ascend: (a: string, realm: string) => `${a} vượt Thiên Kiếp, phi thăng ${realm}!`,
    title_gain: (a: string) => `${a} đã lên ngôi "Bần tăng chưa ngán ai bao giờ"!`,
    title_lose: (a: string) => `${a} bị hạ bệ, bần tăng giờ đã ngán`,
    gate_kill: (a: string, b: string) => pick([
      `${b} chui cổng ra liền bị ${a} chém, đen thôi đỏ quên đi`,
      `${b} vừa ló đầu khỏi cổng đã ăn kiếm của ${a}`,
      `${a} ngồi canh cổng, ${b} xui xẻo bước ra`,
      `${b} dịch chuyển hết một vòng để tới chỗ ${a} chết`,
    ]),
  },

  // Emote bubbles (ART_PROMPTS §10). Khoá = tên sự kiện emote của server hoặc tên mood (§12).
  // Giữ câu ngắn: bong bóng chỉ hiện 1.8s và xuống dòng ở 220px.
  // Cố tình KHÔNG có khoá 'idle': người chơi dừng/đi liên tục nên sẽ lải nhải quá nhiều.
  emotes: {
    sword_lost: ['Kiếm của taaa!', 'Trả kiếm đây!', 'Kiếm mới mua mà!', 'Rụng mất một cây rồi!', 'Ai nhặt là mất dạy đó!'],
    shield_break: ['Khiên ơi!', 'Vỡ rồi!', 'Khiên giấy à trời!', 'Mới đeo được 3 giây!', 'Bảo hành đâu?!'],
    chest: ['Trúng mánh!', 'Giàu rồi!', 'Đổi đời rồi bà con!', 'Hôm nay số đỏ!', 'Toàn đồ xịn!', 'Ét ô ét, giàu quá!'],
    revealed: ['Lộ rồi!', 'Ối!', 'Sao thấy tui hay vậy?', 'Chưa núp xong mà!', 'Thôi xong, lộ hàng!'],
    run: ['AAAAA!', 'Tránh ra!', 'Xe tui không phanh!', 'Dô dô dô!', 'Chân ơi cố lên!', 'Nhanh nhanh nhanh!'],
    chase: ['Hehehe...', 'Đứng lại đó!', 'Chạy đâu cho thoát!', 'Lại đây cưng!', 'Đứng im một nhát thôi!', 'Ngon, con mồi kìa!', 'Đạo hữu, dừng bước!'],
    flee: ['Cứu tôi với!', 'Mẹ ơi!', 'Tui còn nhỏ lắm!', 'Nhầm người rồi!', 'Tha cho đệ!', 'Đệ xin hàng!', 'Không đánh nhau, hoà bình!'],
    hit: ['Ối giời!', 'Đau quá!', 'Á đù!', 'Chơi bẩn!', 'Hự!', 'Ai đánh lén đó?!', 'Nhẹ tay coi!'],
    breakthrough: ['Ta đã ngộ rồi!!!', 'Thiên địa chứng giám!', 'Lên cảnh giới rồi nha!', 'Nhìn ta đây này!', 'Cảm tạ trời đất!'],
    levelup: ['Lên tầng!', 'Thơm quá!', 'Tiến bộ rồi nha!', 'Thêm tí tu vi!', 'Mượt!'],
    realm_drop: ['Mất cảnh giới rồi!', 'Công sức bao ngày...', 'Tụt hạng đau quá!', 'Làm lại từ đầu vậy!', 'Trời không thương tui!'],
    teleport_in: ['Đây là đâu???', 'Ủa? Lạc rồi!', 'Cổng gì chóng mặt vậy!', 'Ai đưa tui tới đây?', 'Về nhầm nhà rồi!'],
    teleport_out: ['Tui biến đây!', 'Bái bai nha!', 'Hẹn gặp bên kia!', 'Chui cổng cái đã!'],
    death: ['Thôi xong...', 'Tui đi trước nha!', 'Hẹn kiếp sau!', 'Tin được không trời!'],
    respawn: ['Tui lại về rồi!', 'Chưa hết đâu nha!', 'Hồi sinh, chém tiếp!', 'Nghỉ tí rồi tu lại!'],
    no_swords: ['Hết kiếm rồi!', 'Tui tay không nè!', 'Ai cho mượn cây kiếm...', 'Thôi chạy đã!'],
    bush: ['Suỵt...', 'Không có ai ở đây đâu.', 'Tui là cái bụi cỏ.', 'Đừng nhìn, đừng nhìn...'],
    shocked: ['Hả???', 'Trời đất ơi!', 'Ghê vậy!', 'Gì cơ?!', 'Ông này ghê quá!', 'Tuyệt đối điện ảnh!'],
  } as Record<string, string[]>,

  characters: {
    a_hon: 'A Hớn Bánh Bao',
    ba_tam_loa: 'Bà Tám Loa',
    ong_ba_vong: 'Ông Ba Võng',
    dai_ca_dep: 'Đại Ca Dép Tổ Ong',
    map_tra_sua: 'Thằng Mập Trà Sữa',
    co_ba_ganh: 'Cô Ba Gánh Hàng',
    su_phu_mi: 'Sư Phụ Mì Gói',
    be_ti: 'Bé Tí Cơ Bắp',
    shipper: 'Anh Shipper Tia Chớp',
    cu_rua: 'Cụ Rùa Thiền',
  } as Record<string, string>,

  buffs: {
    spin: 'Tốc Kiếm',
    haste: 'Thần Hành',
    magnet: 'Nam Châm',
    hard: 'Cương Kiếm',
    shield: 'Kim Cang',
    exp2: 'Song Bội Đan',
    stealth: 'Ẩn Thân Phù',
  } as Record<string, string>,

  rotateHint: 'Xoay ngang điện thoại để nhìn rộng hơn',
  minimapToggle: 'Phím M: phóng to bản đồ',

  // Bảng điều khiển mờ hiện vài giây đầu khi vào trận (spec §18.1). Chỉ các nút, không
  // giải thích luật — người chơi tự khám phá nhanh hơn đọc.
  guideTitle: 'Điều khiển',
  guideKeys: (isTouch: boolean): { key: string; what: string }[] =>
    isTouch
      ? [
          { key: 'Kéo', what: 'Di chuyển' },
          { key: 'Chạm 2 lần rồi giữ · 🔥', what: 'Thiêu Đốt' },
          { key: '✦', what: 'Kỹ năng (cần Bí Kíp)' },
        ]
      : [
          { key: 'W A S D', what: 'Di chuyển' },
          { key: 'Shift · nháy đúp giữ', what: 'Thiêu Đốt' },
          { key: 'Space', what: 'Kỹ năng (cần Bí Kíp)' },
          { key: 'M', what: 'Bản đồ' },
        ],
};

export function pick<T>(list: T[]): T {
  return list[Math.floor(Math.random() * list.length)];
}
