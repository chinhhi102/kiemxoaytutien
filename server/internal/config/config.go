// Package config loads config/game.json. Every balance number lives there.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Cultivation struct {
	// TienOverflowScore: đã Đạo Tổ thì không còn cảnh giới để lên, nên mỗi điểm Tiên khí
	// nhặt thêm đổi thành ngần này điểm Uy Danh thay vì bỏ phí. 0 = tắt.
	// Hiện là 0,1 — tỉ lệ 10 Tiên khí ăn 1 Uy Danh (chủ dự án quyết 29/10/2026).
	TienOverflowScore float64  `json:"tien_overflow_score"`
	Realms            []string `json:"realms"`
	TiersPerRealm     int      `json:"tiers_per_realm"`
	ExpBase           float64  `json:"exp_base"`
	RealmMult         float64  `json:"realm_mult"`
	TierStep          float64  `json:"tier_step"`
	ScaleMin          float64  `json:"scale_min"`
	ScaleMax          float64  `json:"scale_max"`

	// Tiên giới (spec §3.4): 5 cảnh giới sau Đại Thừa, không chia tầng, chỉ lên được
	// bằng sát mạng chứ không dùng EXP.
	TienRealms []string `json:"tien_realms"`
	// TienKhiCeilings: TRẦN Tiên khí cộng dồn của từng cảnh giới Tiên (spec v5.7 §3.4).
	// [100,300,600,1000,1500] nghĩa là 1–100 Chân Tiên, 101–300 Kim Tiên, … 1001–1500 Đạo Tổ.
	//
	// Cảnh giới Tiên suy ra THẲNG từ tổng Tiên khí đang có, không cộng dồn từng chặng như
	// bản v5.6 (`tien_khi_per_step`). Khác biệt thật: bị chém thì Tiên khí về đáy cảnh giới
	// mới, và cảnh giới tự tụt theo — không còn hai nguồn sự thật để lệch nhau.
	TienKhiCeilings []float64 `json:"tien_khi_ceilings"`
	// TienEntryRequiresFullExp: phải ĐẦY thanh Đại Thừa tầng 10 mới nhặt được Tiên khí.
	TienEntryRequiresFullExp bool    `json:"tien_entry_requires_full_exp"`
	TienRobShare             float64 `json:"tien_rob_share"`
	TienDropShare            float64 `json:"tien_drop_share"`
	// TienDropScatterMin/Max: viên Tiên khí rơi ra khi bị chém văng xa ngần này (spec v5.9).
	// Phải xa hơn hẳn thân người Tiên giới: họ to gấp 5–6 lần, và vật phẩm vẽ DƯỚI nhân vật,
	// nên viên rơi gần là nằm khuất hẳn dưới xác — đó chính là lỗi "chém mà không thấy rơi".
	TienDropScatterMin float64  `json:"tien_drop_scatter_min"`
	TienDropScatterMax float64  `json:"tien_drop_scatter_max"`
	TienScaleStep      float64  `json:"tien_scale_step"`
	TienAuraColors     []string `json:"tien_aura_colors"`
}

type Sword struct {
	StartCount int   `json:"start_count"`
	MinCount   int   `json:"min_count"`
	MaxSwords  []int `json:"max_swords"`
	// RingStep: vòng thứ k chứa RingStep × k kiếm (vòng 1 = 10, vòng 2 = 20… spec §4.1a).
	RingStep       int      `json:"ring_capacity_step"`
	Length         float64  `json:"length"`
	Width          float64  `json:"width"`
	HiltGap        float64  `json:"hilt_gap"`
	RingGrowth     float64  `json:"ring_growth"`
	RingGrowthMode string   `json:"ring_growth_mode"`
	HiltStep       float64  `json:"hilt_step"`
	TiltDeg        float64  `json:"tilt_deg"`
	TiltTweenSecs  float64  `json:"tilt_tween_secs"`
	RingTints      []string `json:"ring_tints"`
	RevSecs        float64  `json:"rev_secs"`
	// FlipCooldown: chỉ giới hạn việc ĐỔI CHIỀU quay, để vòng kiếm không giật 30 lần/giây.
	FlipCooldown float64 `json:"flip_cooldown"`
	// ClashCooldown: giãn cách giữa hai lần TÍNH BỂ KIẾM của cùng một cặp (A, B).
	// 0 = mỗi tick một lần, tức mài hết một vòng 30 kiếm trong đúng 1 giây (spec §5.2).
	ClashCooldown      float64 `json:"clash_cooldown"`
	ClashUnit          int     `json:"clash_unit"`
	FlipOnlySameRealm  bool    `json:"flip_only_same_realm"`
	ClashKnockback     float64 `json:"clash_knockback"`
	ClashKnockbackSecs float64 `json:"clash_knockback_secs"`
	DropBundleMax      int     `json:"drop_bundle_max"`
	// Độ bền kiếm (spec v4.1 §5.2): mỗi cảnh giới bền hơn cảnh giới dưới 20%.
	// "compound" → k = 1.2^d; "linear" → k = 1 + 0.2 × d.
	DurabilityPerRealm float64 `json:"durability_per_realm"`
	DurabilityMode     string  `json:"durability_mode"`
	// BodyHitMaxRings: kiếm địch chỉ chém được vào thân khi nạn nhân còn tối đa bấy nhiêu
	// vòng (spec v4.5 §5.2 — thay cho lá chắn chặn đường đi của bản trước).
	BodyHitMaxRings int     `json:"body_hit_max_rings"`
	TweenSecs       float64 `json:"tween_secs"`
}

type Player struct {
	BodyR          float64 `json:"body_r"`
	MoveBase       float64 `json:"move_base"`
	PickupR        float64 `json:"pickup_r"`
	InvulnOnHit    float64 `json:"invuln_on_hit"`
	InvulnOnSpawn  float64 `json:"invuln_on_spawn"`
	AttackerShare  float64 `json:"attacker_share"`
	DropShare      float64 `json:"drop_share"`
	KillBonusExp   float64 `json:"kill_bonus_exp"`
	DeathSwordDrop float64 `json:"death_sword_drop"`
	OrbOwnerSecs   float64 `json:"orb_owner_secs"`
	CameraViewH    float64 `json:"camera_view_h"`
	CameraExp      float64 `json:"camera_exp"`
	CameraReachMul float64 `json:"camera_reach_mult"`
	HitDropsRealm  bool    `json:"hit_drops_realm"`
	Knockback      float64 `json:"knockback"`
	KnockbackSecs  float64 `json:"knockback_secs"`
	DeathAnimSecs  float64 `json:"death_anim_secs"`
	NameMin        int     `json:"name_min"`
	NameMax        int     `json:"name_max"`
}

// Burn is Thiêu Đốt Tinh Huyết (spec §4.1c): hold to run faster, paying EXP every second.
// ExpFracPerSec is a fraction of the CURRENT tier's requirement, so the cost scales with
// cultivation instead of being free at high realms and crippling at low ones.
type Burn struct {
	Mult          float64 `json:"mult"`
	ExpFracPerSec float64 `json:"exp_frac_per_sec"`
	// TienFracPerSec: người Tiên giới đốt **Tiên khí** thay cho EXP (spec v5.6 §4.1c) —
	// mỗi giây ngần này phần của BƯỚC Tiên khí hiện tại. Spec §16 ghi khoá này trong khối
	// `player` tên `burn_tien_frac_per_sec`; ở đây để cạnh `exp_frac_per_sec` cho khớp
	// cách config đang xếp (khối `burn` riêng), đúng như `burn_exp_per_sec` đã làm.
	TienFracPerSec float64 `json:"tien_frac_per_sec"`
	KeepRealm      bool    `json:"keep_realm"` // never burn below the floor of the current realm
}

type Buff struct {
	Mult    float64 `json:"mult"`
	Secs    float64 `json:"secs"`
	RevealR float64 `json:"reveal_r"`
	// Cương Kiếm only (spec v4.4 §6.2): kiếm cứng hơn khi va (DurabilityMult) và chém quái /
	// vật cản mạnh hơn (DamageMult).
	DurabilityMult float64 `json:"durability_mult"`
	DamageMult     float64 `json:"damage_mult"`
	// Ẩn Thân only (spec §6.2): the buff never stacks past MaxSecs, viewers see the player
	// at VisibleAlpha instead of not at all, and bots only notice them within
	// BotDetectMult × their normal sight range.
	MaxSecs       float64 `json:"max_secs"`
	VisibleAlpha  float64 `json:"visible_alpha"`
	BotDetectMult float64 `json:"bot_detect_mult"`
}

type Items struct {
	Weights        map[string]int     `json:"weights"`
	CountPerRoom   int                `json:"count_per_room"`
	RespawnSecs    float64            `json:"respawn_secs"`
	SwordAmount    int                `json:"sword_item_amount"`
	BundleAmount   int                `json:"bundle_amount"`
	OvercapExp     float64            `json:"overcap_exp"`
	ChestEverySecs float64            `json:"chest_every_secs"`
	ChestItems     int                `json:"chest_items"`
	SpawnMinDist   float64            `json:"spawn_min_dist"`
	ChestWarnSecs  float64            `json:"chest_warn_secs"`
	ChestHP        int                `json:"chest_hp"`
	ItemR          float64            `json:"item_r"`
	OrbValues      map[string]float64 `json:"orb_values"`
	OrbR           float64            `json:"orb_r"`
	ScatterR       float64            `json:"scatter_r"`
	MaxOrbs        int                `json:"max_orbs"`
}

// ExpOrbs is the ambient EXP spread over the whole map (spec §6.1).
type ExpOrbs struct {
	Cell          float64   `json:"cell"`
	Values        []float64 `json:"values"`
	Weights       []int     `json:"weights"`
	RespawnSecs   float64   `json:"respawn_secs"`
	RespawnRegion float64   `json:"respawn_region"`
}

type MonsterKind struct {
	HP        int     `json:"hp"`
	Exp       float64 `json:"exp"`
	Drop      float64 `json:"drop"`
	DropItems int     `json:"drop_items"`
	Weight    int     `json:"weight"`
	Radius    float64 `json:"radius"`
	Max       int     `json:"max"`
}

type Monsters struct {
	HitCooldown float64                `json:"hit_cooldown"`
	WanderSpeed float64                `json:"wander_speed"`
	FleeSpeed   float64                `json:"flee_speed"`
	FleeSecs    float64                `json:"flee_secs"`
	Kinds       map[string]MonsterKind `json:"kinds"`
}

type Obstacles struct {
	FootprintW     float64        `json:"footprint_w"`
	FootprintH     float64        `json:"footprint_h"`
	OcclusionAlpha float64        `json:"occlusion_alpha"`
	RegrowSecs     float64        `json:"regrow_secs"`
	RegrowWarnSecs float64        `json:"regrow_warn_secs"`
	HP             map[string]int `json:"hp"`
	StallItemDrop  float64        `json:"stall_item_drop"`
	HitCooldown    float64        `json:"hit_cooldown"`
	// Spec §5.3 (v3.6): damage is per player, not per sword — one hit per DmgCooldown
	// per (player, obstacle), scaled by how many rings they have.
	DmgCooldown    float64 `json:"dmg_cooldown_secs"`
	DmgPerExtraRng float64 `json:"dmg_per_extra_ring"`
}

// Ice: mặt băng ở Băng Nguyệt Hồ (spec §8.0). Di chuyển có quán tính:
//
//	vận tốc mới = vận tốc cũ × friction + hướng input × accel × dt
//
// Tốc độ tới hạn = accel × dt / (1 − friction). Với friction 0,92, dt 1/30 và accel 468 thì
// tới hạn ≈ 195 = đúng `player.move_base`, nên chạy thẳng trên băng vẫn nhanh bằng trên
// đất — cái khó là **dừng và đổi hướng**, đúng ý spec.
type Ice struct {
	Friction float64 `json:"friction"`
	Accel    float64 `json:"accel"`
}

// Skills: mọi con số của 10 kỹ năng nhân vật (spec §6.3). Bán kính và tầm xa còn được
// nhân theo scale người dùng lúc thi hành.
type Skills struct {
	MaxCharges  int     `json:"max_charges"`
	LockSecs    float64 `json:"lock_secs"`
	OvercapExp  float64 `json:"overcap_exp"`
	OvercapTien float64 `json:"overcap_tien"`
	KnockSecs   float64 `json:"knock_secs"`

	// BanhBaoMaxShare: hồi ngay ngần này phần của số kiếm TỐI ĐA ở cảnh giới hiện tại
	// (spec v5.5 §6.3). Bản v5.0 hồi theo số kiếm vừa bể trong 10s — đổi rồi, không còn
	// sổ ghi kiếm bể nữa.
	BanhBaoMaxShare float64 `json:"banh_bao_max_share"`

	LoaConeDeg float64 `json:"loa_cone_deg"`
	LoaReach   float64 `json:"loa_reach"`
	LoaPush    float64 `json:"loa_push"`

	RuNguRadius float64 `json:"ru_ngu_radius"`
	RuNguSecs   float64 `json:"ru_ngu_secs"`
	RuNguMove   float64 `json:"ru_ngu_move"`
	RuNguSpin   float64 `json:"ru_ngu_spin"`

	DanEmCount int     `json:"dan_em_count"`
	DanEmSecs  float64 `json:"dan_em_secs"`
	// DanEmSwordShare: mỗi đàn em có ngần này phần số kiếm HIỆN TẠI của đại ca (làm tròn
	// xuống, tối thiểu 1). Đại ca không bị trừ kiếm — đây là bản sao, không phải chia của.
	DanEmSwordShare float64 `json:"dan_em_sword_share"`
	// DanEmScale: đàn em nhỏ hơn đại ca (0.7 = nhỏ hơn 30%).
	DanEmScale float64 `json:"dan_em_scale"`

	TranChauConeDeg float64 `json:"tran_chau_cone_deg"`
	TranChauReach   float64 `json:"tran_chau_reach"`
	TranChauSecs    float64 `json:"tran_chau_secs"`
	// TranChauNoPickup: người đang bị bịt mắt không nhặt được gì (spec v5.5 §6.3).
	TranChauNoPickup  bool    `json:"tran_chau_no_pickup"`
	TranChauBotVision float64 `json:"tran_chau_bot_vision"`

	DashDist    float64 `json:"dash_dist"`
	DashSecs    float64 `json:"dash_secs"`
	SlickCount  int     `json:"slick_count"`
	SlickSecs   float64 `json:"slick_secs"`
	SlickRadius float64 `json:"slick_radius"`
	SlipSecs    float64 `json:"slip_secs"`

	LuoiMiRange  float64 `json:"luoi_mi_range"`
	LuoiMiRadius float64 `json:"luoi_mi_radius"`
	LuoiMiSecs   float64 `json:"luoi_mi_secs"`

	BraceSecs       float64 `json:"brace_secs"`
	BraceDurability float64 `json:"brace_durability"`

	GhostSecs  float64 `json:"ghost_secs"`
	GhostSpeed float64 `json:"ghost_speed"`
	// GhostUnstickSecs: hết Giao Hỏa Tốc mà đang đứng trong vật cản thì được đẩy mượt ra
	// chỗ trống gần nhất trong ngần này giây (spec v5.5 §6.3) — không dịch tức thời, vì
	// nhảy một phát là client thấy giật y như lỗi §8.2a vừa sửa xong.
	GhostUnstickSecs float64 `json:"ghost_unstick_secs"`

	ShellSecs       float64 `json:"shell_secs"`
	ShellPushRadius float64 `json:"shell_push_radius"`
	ShellPush       float64 `json:"shell_push"`
}

type Zones struct {
	// SnowMult: bờ tuyết ở Băng Nguyệt Hồ làm chậm (spec §8.0). Cùng cơ chế với bùn.
	SnowMult            float64 `json:"snow_mult"`
	MudMult             float64 `json:"mud_mult"`
	WindWith            float64 `json:"wind_with"`
	WindAgainst         float64 `json:"wind_against"`
	WindWithMaxDeg      float64 `json:"wind_with_max_deg"`
	WindAgainstMinDeg   float64 `json:"wind_against_min_deg"`
	BushRevealR         float64 `json:"bush_reveal_r"`
	BushRevealOnHitSecs float64 `json:"bush_reveal_on_hit_secs"`
}

type Gates struct {
	// RandomExit: bước vào một cổng thì ra ở một cổng BẤT KỲ khác cùng mạng, thay vì cổng
	// đôi cố định. Đặt false để quay về kiểu cũ.
	RandomExit bool `json:"random_exit"`
	// ExitPool: "all" (mặc định) = ra ở cổng bất kỳ trong cả thế giới; "other_zone" = luôn
	// sang khu khác (spec §8.4).
	ExitPool string  `json:"exit_pool"`
	Radius   float64 `json:"radius"`
	// HeavenRadius: bán kính kích hoạt Thiên Môn — to gấp đôi cổng thường (spec §8.4a).
	HeavenRadius float64 `json:"heaven_radius"`
	TransitSecs  float64 `json:"transit_secs"`
	ExitWarnSecs float64 `json:"exit_warn_secs"`
	CooldownSecs float64 `json:"cooldown_secs"`
	FeedSecs     float64 `json:"feed_secs"`
}

type Room struct {
	// Zones: số khu mỗi phòng (spec §8.0). Để 1 là quay về thế giới một bản đồ như trước.
	Zones int `json:"zones"`
	// ZoneMinChars: sàn nhân vật mỗi khu. Bot được thả thêm cho tới khi khu nào cũng đạt
	// sàn này, nhưng không bao giờ vượt `BotFillTarget` của cả phòng — nếu không thì một
	// khu đông người sẽ kéo tổng số vượt trần.
	ZoneMinChars   int     `json:"zone_min_chars"`
	MaxHumans      int     `json:"max_humans"`
	BotFillTarget  int     `json:"bot_fill_target"`
	MapSize        float64 `json:"map_size"`
	MonsterCount   int     `json:"monster_count"`
	EmptyCloseSecs float64 `json:"empty_close_secs"`
	SafeSpawnR     float64 `json:"safe_spawn_r"`
}

type Bots struct {
	SpawnEverySecs float64 `json:"spawn_every_secs"`
	SpawnBatch     int     `json:"spawn_batch"`
	RespawnSecs    float64 `json:"respawn_secs"`
	// SpawnRank: bot sinh ra ngang với người đang đứng hạng này trong phòng (mặc định 11,
	// tức ngay dưới Top 10). Phòng chưa đủ người để có hạng đó → sinh ở Luyện Khí.
	// Không có trần cảnh giới: phòng toàn Nguyên Anh thì bot mới cũng Nguyên Anh.
	SpawnRank   int      `json:"spawn_rank"`
	DecideMin   float64  `json:"decide_min"`
	DecideMax   float64  `json:"decide_max"`
	ReactionMin float64  `json:"reaction_min"`
	ReactionMax float64  `json:"reaction_max"`
	HuntR       float64  `json:"hunt_r"`
	HuntRatio   float64  `json:"hunt_ratio"`
	FleeR       float64  `json:"flee_r"`
	FleeRatio   float64  `json:"flee_ratio"`
	PathGrid    float64  `json:"path_grid"`
	AmbushR     float64  `json:"ambush_r"`
	AmbushSecs  float64  `json:"ambush_secs"`
	Names       []string `json:"names"`
}

type Mood struct {
	MinSecs          float64 `json:"min_secs"`
	HitSecs          float64 `json:"hit_secs"`
	LevelupSecs      float64 `json:"levelup_secs"`
	BreakthroughSecs float64 `json:"breakthrough_secs"`
	RealmDropSecs    float64 `json:"realm_drop_secs"`
	RespawnSecs      float64 `json:"respawn_secs"`
	NearR            float64 `json:"near_r"`
}

// Minimap is the client-side minimap (spec §12.1); the server only passes it through.
// AltarKind is one Đài Tiên Khí type (spec §8.5).
type AltarKind struct {
	Count          int     `json:"count"`
	EverySecs      float64 `json:"every_secs"`
	Orbs           []int   `json:"orbs"` // [min, max] viên mỗi đợt thường
	OrbValue       float64 `json:"orb_value"`
	BurstEverySecs float64 `json:"burst_every_secs"`
	BurstSmall     int     `json:"burst_small"`
	BurstMedium    int     `json:"burst_medium"`
	MediumValue    float64 `json:"medium_value"`
	WarnSecs       float64 `json:"warn_secs"`
}

// Altars is the Đài Tiên Khí section (spec §8.5).
type Altars struct {
	Small      AltarKind `json:"small"`
	Center     AltarKind `json:"center"`
	SprayR     []float64 `json:"spray_r"`
	OrbLifeSec float64   `json:"orb_lifetime_secs"`
	ClearR     float64   `json:"clear_r"`
}

// Mapgen is the §8.1 generator settings.
type Mapgen struct {
	CenterPlazaR float64 `json:"center_plaza_r"`
	SectR        float64 `json:"sect_r"`
	SectInset    float64 `json:"sect_inset"`
	RoadW        float64 `json:"road_w"`
	RingRoadR    float64 `json:"ring_road_r"`
	Symmetry     string  `json:"symmetry"`
	Jitter       float64 `json:"jitter"`
	MinOpenRatio float64 `json:"min_open_ratio"`
	MaxRetries   int     `json:"max_retries"`
}

type Minimap struct {
	SizePC     int  `json:"size_pc"`
	SizeMobile int  `json:"size_mobile"`
	ShowTop1   bool `json:"show_top1"`
	ShowChest  bool `json:"show_chest"`
	ShowOthers bool `json:"show_others"`
}

// Score is the Uy Danh scoring table (spec §11.1).
type Score struct {
	HitBase              float64 `json:"hit_base"`
	KillBonus            float64 `json:"kill_bonus"`
	UpsetPerRealm        float64 `json:"upset_per_realm"`
	BreakthroughPerRealm float64 `json:"breakthrough_per_realm"`
	Top1PerSec           float64 `json:"top1_per_sec"`
	BotVictimMult        float64 `json:"bot_victim_mult"`
	SameVictimCooldown   float64 `json:"same_victim_cooldown_secs"`
}

type Net struct {
	TickHz             int     `json:"tick_hz"`
	SnapshotHz         int     `json:"snapshot_hz"`
	InterpMs           int     `json:"interp_ms"`
	AOIMargin          float64 `json:"aoi_margin"`
	AOIAspect          float64 `json:"aoi_aspect"`
	LeaderboardHz      int     `json:"leaderboard_hz"`
	MaxMsgsPerSec      int     `json:"max_msgs_per_sec"`
	MaxConnsPerIP      int     `json:"max_conns_per_ip"`
	LeaderboardCacheSc float64 `json:"leaderboard_cache_secs"`
	// CorrectionSnap / CorrectionTauMs: client dùng để sửa vị trí dự đoán (spec §8.2a).
	// Server không dùng, nhưng phải có ở đây thì `/api/config` mới gửi đi được.
	CorrectionSnap  float64 `json:"correction_snap"`
	CorrectionTauMs float64 `json:"correction_tau_ms"`
}

// Config mirrors config/game.json.
type Config struct {
	Skills      Skills          `json:"skills"`
	Ice         Ice             `json:"ice"`
	Cultivation Cultivation     `json:"cultivation"`
	Sword       Sword           `json:"sword"`
	Player      Player          `json:"player"`
	Burn        Burn            `json:"burn"`
	Buffs       map[string]Buff `json:"buffs"`
	Items       Items           `json:"items"`
	ExpOrbs     ExpOrbs         `json:"exp_orbs"`
	Monsters    Monsters        `json:"monsters"`
	Obstacles   Obstacles       `json:"obstacles"`
	Zones       Zones           `json:"zones"`
	Gates       Gates           `json:"gates"`
	Room        Room            `json:"room"`
	Bots        Bots            `json:"bots"`
	Characters  []string        `json:"characters"`
	Mood        Mood            `json:"mood"`
	Titles      []string        `json:"titles"`
	TitleHold   float64         `json:"title_hold_secs"`
	Minimap     Minimap         `json:"minimap"`
	Altars      Altars          `json:"altars"`
	Mapgen      Mapgen          `json:"mapgen"`
	Score       Score           `json:"score"`
	Net         Net             `json:"net"`
}

// Load reads and parses a config file. It returns the parsed config and raw bytes
// (the raw bytes are served to clients at GET /api/config).
func Load(path string) (*Config, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, raw, nil
}

// FindRepoFile walks up from the working directory to locate a repo-relative path
// such as "config/game.json", so commands work from both repo root and server/.
func FindRepoFile(rel string) string {
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		dir = filepath.Dir(dir)
	}
	return rel
}

// MustLoadDefault loads config/game.json found from the working directory; for tests and tools.
func MustLoadDefault() *Config {
	c, _, err := Load(FindRepoFile("config/game.json"))
	if err != nil {
		panic(err)
	}
	return c
}
