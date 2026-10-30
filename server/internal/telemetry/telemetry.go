// Package telemetry ghi lại số liệu hiệu năng của NGƯỜI CHƠI THẬT trên server LAN
// (TELEMETRY_SPEC.md). Máy thật, mạng thật, trận thật — thứ mà đo bằng bot trên máy dev
// không bao giờ thay thế được.
//
// Ba luật xương sống của gói này, vi phạm cái nào cũng làm hỏng chính thứ đang đo:
//
//  1. **Không bao giờ chạy trong vòng tick.** Mọi thứ vào một channel có đệm rồi một
//     goroutine riêng ghi xuống đĩa. Một lần `INSERT` chậm 20 ms mà nằm trong tick thì
//     telemetry tự tạo ra đúng cái lag nó đi đo.
//  2. **Đầy thì VỨT, không chặn.** Ingest không bao giờ đợi. Mất vài mẫu còn hơn làm nghẽn
//     vòng đọc WebSocket của một người đang chơi.
//  3. **Ẩn danh.** `guest_id` và IP chỉ vào DB dưới dạng băm (§1.3) — IP giữ lại chỉ để
//     phân biệt "hai phiên này cùng một mạng", không bao giờ để truy ra ai.
package telemetry

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver, no CGO
)

const (
	// flushEvery: nhịp ghi xuống đĩa. Spec §3 chốt 5s.
	flushEvery = 5 * time.Second
	// queueSize: số bản ghi đệm trong RAM. 20 người × (1 mẫu/10s + sự kiện) thì 4096 là
	// thừa sức cho 5 giây; đặt rộng để một lần ghi đĩa chậm bất thường không làm rơi mẫu.
	queueSize = 4096
	// retainDays / maxBytes: giữ 30 ngày, trần 500 MB (§3).
	retainDays = 30
	maxBytes   = 500 << 20
	// sweepEvery: nhịp dọn dữ liệu cũ.
	sweepEvery = time.Hour
)

// Session là thông tin một lần vào trận, gửi 1 lần (§2.1).
type Session struct {
	ID        string  `msgpack:"id"`
	GuestHash string  `msgpack:"gh"`
	Build     string  `msgpack:"b"`
	Proto     int     `msgpack:"p"`
	Schema    string  `msgpack:"sh"`
	UA        string  `msgpack:"ua"`
	Browser   string  `msgpack:"br"`
	OS        string  `msgpack:"os"`
	Mobile    bool    `msgpack:"mo"`
	Cores     int     `msgpack:"co"`
	MemGB     float64 `msgpack:"me"`
	ScreenW   int     `msgpack:"sw"`
	ScreenH   int     `msgpack:"shh"`
	DPR       float64 `msgpack:"dpr"`
	GPU       string  `msgpack:"gpu"`
	WebGL     int     `msgpack:"wg"`
	MaxTex    int     `msgpack:"mt"`
	NetType   string  `msgpack:"nt"`
	Downlink  float64 `msgpack:"dl"`
	NetRTT    float64 `msgpack:"nr"`
	Quality   string  `msgpack:"q"`
	QualAuto  bool    `msgpack:"qa"`
	CanvasW   int     `msgpack:"cw"`
	CanvasH   int     `msgpack:"ch"`
}

// Sample là một mẫu 10 giây đã tổng hợp sẵn ở client (§2.2).
//
// Tên trường ngắn trên dây vì mỗi mẫu đi qua WebSocket của người đang chơi; ngân sách §1 là
// dưới 1 KB/s cho toàn bộ telemetry.
type Sample struct {
	Session string  `msgpack:"s"`
	TS      float64 `msgpack:"ts"` // mốc client, ms unix

	FPSAvg   float64 `msgpack:"fa"`
	FPSP1    float64 `msgpack:"f1"`
	FrameP50 float64 `msgpack:"m50"`
	FrameP95 float64 `msgpack:"m95"`
	FrameMax float64 `msgpack:"mmx"`
	Over33   int     `msgpack:"o33"`
	Over100  int     `msgpack:"o100"`

	LongTasks int `msgpack:"lt"`

	DrawCalls   float64 `msgpack:"dc"`
	Sprites     float64 `msgpack:"sp"`
	TexMB       float64 `msgpack:"tx"`
	TexUploads  int     `msgpack:"tu"`
	Placeholder int     `msgpack:"ph"`

	PingP50 float64 `msgpack:"p50"`
	PingP95 float64 `msgpack:"p95"`
	PingMax float64 `msgpack:"pmx"`
	KbpsIn  float64 `msgpack:"ki"`
	KbpsOut float64 `msgpack:"ko"`
	MaxPkt  int     `msgpack:"mp"`
	Late150 int     `msgpack:"l150"`
	Stalls  int     `msgpack:"st"`

	Corrections float64 `msgpack:"cor"`
	Respawn2s   int     `msgpack:"rs"`
	InterpMS    float64 `msgpack:"im"`

	Zone       int  `msgpack:"zn"`
	Realm      int  `msgpack:"rl"`
	Swords     int  `msgpack:"sws"`
	Rings      int  `msgpack:"rg"`
	CharsView  int  `msgpack:"cv"`
	TienView   int  `msgpack:"tv"`
	SwordsDraw int  `msgpack:"sd"`
	VFX        int  `msgpack:"vf"`
	Burning    bool `msgpack:"bn"`
	Zoning     bool `msgpack:"zg"`
}

// Event là sự kiện gửi ngay (§2.3).
type Event struct {
	Session string  `msgpack:"s"`
	TS      float64 `msgpack:"ts"`
	Kind    string  `msgpack:"k"`
	Data    string  `msgpack:"d"`
}

// End là tóm tắt gửi lúc đóng tab qua sendBeacon (§2.4).
type End struct {
	Session   string  `msgpack:"id" json:"id"`
	DurSecs   float64 `msgpack:"dur" json:"dur"`
	NSamples  int     `msgpack:"n" json:"n"`
	FPSMedian float64 `msgpack:"fps" json:"fps"`
}

// ServerSample là số liệu SERVER tự ghi mỗi 10s, cùng DB cùng mốc thời gian (§3).
//
// Vì sao phải nằm chung DB: câu hỏi đắt nhất là "client tụt FPS lúc đó thì server đang làm
// gì?" (§4.1 mục 5). Hai file riêng thì phải tự ghép tay theo đồng hồ, và đồng hồ hai máy
// lệch nhau.
type ServerSample struct {
	TS       int64
	Zone     int
	TickAvg  float64
	TickP95  float64
	Humans   int
	Bots     int
	KbpsOut  float64
	CPUPct   float64
	RSSMB    float64
	BuildID  string
	ZoneName string
}

const schema = `
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS sessions (
	session_id TEXT PRIMARY KEY,
	guest_hash TEXT NOT NULL DEFAULT '',
	ip_hash    TEXT NOT NULL DEFAULT '',
	build      TEXT NOT NULL DEFAULT '',
	proto      INTEGER NOT NULL DEFAULT 0,
	schema_hash TEXT NOT NULL DEFAULT '',
	started_at INTEGER NOT NULL,
	ua TEXT NOT NULL DEFAULT '', browser TEXT NOT NULL DEFAULT '', os TEXT NOT NULL DEFAULT '',
	mobile INTEGER NOT NULL DEFAULT 0, cores INTEGER NOT NULL DEFAULT 0, mem_gb REAL NOT NULL DEFAULT 0,
	screen_w INTEGER NOT NULL DEFAULT 0, screen_h INTEGER NOT NULL DEFAULT 0, dpr REAL NOT NULL DEFAULT 0,
	gpu TEXT NOT NULL DEFAULT '', webgl INTEGER NOT NULL DEFAULT 0, max_tex INTEGER NOT NULL DEFAULT 0,
	net_type TEXT NOT NULL DEFAULT '', downlink REAL NOT NULL DEFAULT 0, net_rtt REAL NOT NULL DEFAULT 0,
	quality TEXT NOT NULL DEFAULT '', qual_auto INTEGER NOT NULL DEFAULT 0,
	canvas_w INTEGER NOT NULL DEFAULT 0, canvas_h INTEGER NOT NULL DEFAULT 0,
	ended_at INTEGER NOT NULL DEFAULT 0, dur_secs REAL NOT NULL DEFAULT 0,
	n_samples INTEGER NOT NULL DEFAULT 0, fps_median REAL NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL,
	ts INTEGER NOT NULL,
	server_ts INTEGER NOT NULL,
	fps_avg REAL, fps_p1 REAL, frame_p50 REAL, frame_p95 REAL, frame_max REAL,
	over33 INTEGER, over100 INTEGER, long_tasks INTEGER,
	draw_calls REAL, sprites REAL, tex_mb REAL, tex_uploads INTEGER, placeholder INTEGER,
	ping_p50 REAL, ping_p95 REAL, ping_max REAL, kbps_in REAL, kbps_out REAL,
	max_pkt INTEGER, late150 INTEGER, stalls INTEGER,
	corrections REAL, respawn2s INTEGER, interp_ms REAL,
	zone INTEGER, realm INTEGER, swords INTEGER, rings INTEGER,
	chars_view INTEGER, tien_view INTEGER, swords_draw INTEGER, vfx INTEGER,
	burning INTEGER, zoning INTEGER
);
CREATE INDEX IF NOT EXISTS samples_ts ON samples(server_ts);
CREATE INDEX IF NOT EXISTS samples_sess ON samples(session_id);
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL,
	ts INTEGER NOT NULL,
	server_ts INTEGER NOT NULL,
	kind TEXT NOT NULL,
	data TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS events_ts ON events(server_ts);
CREATE TABLE IF NOT EXISTS server_samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts INTEGER NOT NULL,
	zone INTEGER NOT NULL,
	zone_name TEXT NOT NULL DEFAULT '',
	tick_avg REAL, tick_p95 REAL,
	humans INTEGER, bots INTEGER,
	kbps_out REAL, cpu_pct REAL, rss_mb REAL,
	build_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS srv_ts ON server_samples(ts);
CREATE TABLE IF NOT EXISTS rollups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ran_at INTEGER NOT NULL,
	until_ts INTEGER NOT NULL
);`

// Store là cửa duy nhất để ghi telemetry.
type Store struct {
	db      *sql.DB
	path    string
	buildID string

	q        chan any // *Session | *Sample | *Event | *End | *ServerSample
	done     chan struct{}
	wg       sync.WaitGroup
	once     sync.Once
	closeErr error

	dropped atomic.Int64
	written atomic.Int64
}

// Open mở (hoặc tạo) DB telemetry và chạy goroutine ghi.
func Open(path, buildID string) (*Store, error) {
	// `_pragma` trên DSN: modernc/sqlite chỉ nhận PRAGMA theo cách này cho MỌI kết nối.
	// Đặt WAL trong `schema` thôi là chưa đủ — §6 yêu cầu server tắt đột ngột không hỏng DB.
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema telemetry: %w", err)
	}
	s := &Store{db: db, path: path, buildID: buildID, q: make(chan any, queueSize), done: make(chan struct{})}
	s.wg.Add(1)
	go s.run()
	return s, nil
}

// Close ghi nốt những gì còn trong hàng đợi rồi đóng DB.
//
// Gọi được NHIỀU LẦN: main vừa đóng trong bộ xử lý tín hiệu (để tắt bằng Ctrl-C vẫn ghi
// được 5 giây cuối) vừa `defer` phòng khi ListenAndServe lỗi. Không có `Once` thì lần
// `close(s.done)` thứ hai làm panic cả server.
func (s *Store) Close() error {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
		s.closeErr = s.db.Close()
	})
	return s.closeErr
}

// BuildID là mã build đang chạy, ghi kèm mọi dòng server_samples (§5).
func (s *Store) BuildID() string { return s.buildID }

// DB cho phép phần báo cáo đọc trực tiếp. Chỉ ĐỌC.
func (s *Store) DB() *sql.DB { return s.db }

// Stats trả về (đã ghi, đã vứt) để trang /admin/perf nói thật về chính nó.
func (s *Store) Stats() (written, dropped int64) {
	return s.written.Load(), s.dropped.Load()
}

// Sync đợi tới khi mọi thứ đã nằm trong hàng đợi lúc gọi được ghi xong xuống đĩa.
//
// Chỉ dùng cho test và cho trang quản trị khi cần đọc ngay số vừa ghi. KHÔNG gọi từ đường
// chạy của người chơi: nó CHẶN, và đó đúng là thứ luật 2 cấm.
//
// Cách làm: đẩy một dấu mốc vào cùng hàng đợi rồi đợi goroutine ghi chạm tới nó. Vì hàng
// đợi vào trước ra trước, dấu mốc được xử lý nghĩa là mọi bản ghi trước nó đã xong.
func (s *Store) Sync() {
	done := make(chan struct{})
	select {
	case s.q <- syncMark{done}:
	case <-s.done:
		return
	}
	select {
	case <-done:
	case <-s.done:
	}
}

type syncMark struct{ done chan struct{} }

// push đẩy một bản ghi vào hàng đợi. KHÔNG BAO GIỜ CHẶN (luật 2).
func (s *Store) push(v any) {
	select {
	case s.q <- v:
	default:
		s.dropped.Add(1)
	}
}

func (s *Store) AddSession(v *Session, ipHash string) { s.push(sessionWithIP{v, ipHash}) }
func (s *Store) AddSample(v *Sample)                  { s.push(v) }
func (s *Store) AddEvent(v *Event)                    { s.push(v) }
func (s *Store) AddEnd(v *End)                        { s.push(v) }
func (s *Store) AddServerSample(v *ServerSample)      { s.push(v) }

type sessionWithIP struct {
	s      *Session
	ipHash string
}

// run là goroutine ghi: gom trong 5s rồi ghi một lô trong một giao dịch.
func (s *Store) run() {
	defer s.wg.Done()
	tick := time.NewTicker(flushEvery)
	defer tick.Stop()
	sweep := time.NewTicker(sweepEvery)
	defer sweep.Stop()
	s.sweep()

	batch := make([]any, 0, 256)
	drain := func() {
		// Vét sạch channel trước khi ghi: gộp được bao nhiêu thì một giao dịch ghi bấy nhiêu.
		for {
			select {
			case v := <-s.q:
				batch = append(batch, v)
			default:
				return
			}
		}
	}
	for {
		select {
		case <-s.done:
			drain()
			s.flush(batch)
			return
		case <-tick.C:
			drain()
			if len(batch) > 0 {
				s.flush(batch)
				batch = batch[:0]
			}
		case <-sweep.C:
			s.sweep()
		case v := <-s.q:
			batch = append(batch, v)
			if len(batch) >= queueSize/2 {
				// Hàng đợi đầy nhanh bất thường: ghi ngay, đừng đợi hết 5s rồi vứt mẫu.
				s.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (s *Store) flush(batch []any) {
	if len(batch) == 0 {
		return
	}
	// Dấu mốc Sync phải được đánh thức SAU khi commit, không phải lúc gặp nó trong lô — gọi
	// sớm thì người đợi đọc DB trước khi giao dịch ghi xong và thấy rỗng.
	//
	// `defer` chứ không gọi ở cuối hàm: mọi đường thoát, kể cả lỗi mở giao dịch hay lỗi
	// commit, đều phải đánh thức người đang đợi. Quên một nhánh là Sync treo vĩnh viễn.
	var marks []syncMark
	defer func() {
		for _, m := range marks {
			close(m.done)
		}
	}()
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("telemetry: mở giao dịch: %v", err)
		return
	}
	now := time.Now().Unix()
	ok := 0
	for _, v := range batch {
		if m, isMark := v.(syncMark); isMark {
			marks = append(marks, m)
			continue
		}
		if err := writeOne(tx, v, now); err != nil {
			log.Printf("telemetry: ghi %T: %v", v, err)
			continue
		}
		ok++
	}
	if err := tx.Commit(); err != nil {
		log.Printf("telemetry: commit: %v", err)
		return
	}
	s.written.Add(int64(ok))
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func writeOne(tx *sql.Tx, v any, now int64) error {
	switch x := v.(type) {
	case sessionWithIP:
		d := x.s
		_, err := tx.Exec(`INSERT OR REPLACE INTO sessions
			(session_id,guest_hash,ip_hash,build,proto,schema_hash,started_at,ua,browser,os,mobile,
			 cores,mem_gb,screen_w,screen_h,dpr,gpu,webgl,max_tex,net_type,downlink,net_rtt,
			 quality,qual_auto,canvas_w,canvas_h)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			d.ID, d.GuestHash, x.ipHash, d.Build, d.Proto, d.Schema, now, d.UA, d.Browser, d.OS, b2i(d.Mobile),
			d.Cores, d.MemGB, d.ScreenW, d.ScreenH, d.DPR, d.GPU, d.WebGL, d.MaxTex, d.NetType, d.Downlink, d.NetRTT,
			d.Quality, b2i(d.QualAuto), d.CanvasW, d.CanvasH)
		return err
	case *Sample:
		_, err := tx.Exec(`INSERT INTO samples
			(session_id,ts,server_ts,fps_avg,fps_p1,frame_p50,frame_p95,frame_max,over33,over100,long_tasks,
			 draw_calls,sprites,tex_mb,tex_uploads,placeholder,ping_p50,ping_p95,ping_max,kbps_in,kbps_out,
			 max_pkt,late150,stalls,corrections,respawn2s,interp_ms,zone,realm,swords,rings,
			 chars_view,tien_view,swords_draw,vfx,burning,zoning)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			x.Session, int64(x.TS), now, x.FPSAvg, x.FPSP1, x.FrameP50, x.FrameP95, x.FrameMax, x.Over33, x.Over100, x.LongTasks,
			x.DrawCalls, x.Sprites, x.TexMB, x.TexUploads, x.Placeholder, x.PingP50, x.PingP95, x.PingMax, x.KbpsIn, x.KbpsOut,
			x.MaxPkt, x.Late150, x.Stalls, x.Corrections, x.Respawn2s, x.InterpMS, x.Zone, x.Realm, x.Swords, x.Rings,
			x.CharsView, x.TienView, x.SwordsDraw, x.VFX, b2i(x.Burning), b2i(x.Zoning))
		return err
	case *Event:
		_, err := tx.Exec(`INSERT INTO events (session_id,ts,server_ts,kind,data) VALUES (?,?,?,?,?)`,
			x.Session, int64(x.TS), now, x.Kind, x.Data)
		return err
	case *End:
		_, err := tx.Exec(`UPDATE sessions SET ended_at=?, dur_secs=?, n_samples=?, fps_median=? WHERE session_id=?`,
			now, x.DurSecs, x.NSamples, x.FPSMedian, x.Session)
		return err
	case *ServerSample:
		_, err := tx.Exec(`INSERT INTO server_samples
			(ts,zone,zone_name,tick_avg,tick_p95,humans,bots,kbps_out,cpu_pct,rss_mb,build_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			x.TS, x.Zone, x.ZoneName, x.TickAvg, x.TickP95, x.Humans, x.Bots, x.KbpsOut, x.CPUPct, x.RSSMB, x.BuildID)
		return err
	}
	return fmt.Errorf("kiểu lạ %T", v)
}

// sweep xoá dữ liệu quá 30 ngày, rồi nếu file vẫn quá 500 MB thì xoá tiếp từ cũ nhất (§3).
func (s *Store) sweep() {
	cut := time.Now().AddDate(0, 0, -retainDays).Unix()
	s.deleteOlderThan(cut)
	for i := 0; i < 8 && s.sizeBytes() > maxBytes; i++ {
		// Quá trần dung lượng: cắt thêm từng 3 ngày tính từ bản ghi cũ nhất. Lặp có TRẦN để
		// một DB phình bất thường không biến thành vòng xoá vô tận.
		var oldest sql.NullInt64
		if err := s.db.QueryRow(`SELECT MIN(server_ts) FROM samples`).Scan(&oldest); err != nil || !oldest.Valid {
			return
		}
		s.deleteOlderThan(oldest.Int64 + 3*24*3600)
		if _, err := s.db.Exec(`VACUUM`); err != nil {
			log.Printf("telemetry: vacuum: %v", err)
			return
		}
	}
}

func (s *Store) deleteOlderThan(cut int64) {
	for _, q := range []string{
		`DELETE FROM samples WHERE server_ts < ?`,
		`DELETE FROM events WHERE server_ts < ?`,
		`DELETE FROM server_samples WHERE ts < ?`,
		`DELETE FROM sessions WHERE started_at < ? AND session_id NOT IN (SELECT DISTINCT session_id FROM samples)`,
	} {
		if _, err := s.db.Exec(q, cut); err != nil {
			log.Printf("telemetry: dọn cũ: %v", err)
			return
		}
	}
}

func (s *Store) sizeBytes() int64 {
	var total int64
	for _, suf := range []string{"", "-wal", "-shm"} {
		if fi, err := os.Stat(s.path + suf); err == nil {
			total += fi.Size()
		}
	}
	return total
}

// Hash băm một chuỗi nhận dạng (guest_id, IP) trước khi vào DB (§1.3).
//
// Cắt còn 16 ký tự hex: đủ để phân biệt người/mạng trong một server LAN vài chục người,
// và ngắn thì bảng báo cáo đọc được bằng mắt.
func Hash(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("kx-telemetry\x00" + s))
	return hex.EncodeToString(sum[:])[:16]
}
