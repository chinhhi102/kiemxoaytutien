// Command server runs the game: HTTP API, WebSocket rooms and the built client.
//
//	go run ./cmd/server              # http://localhost:9999
//	go run ./cmd/server -debug=false # production: disable debug keys
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/room"
	"kiemxoay/server/internal/store"
	"kiemxoay/server/internal/world"
)

func main() {
	// `:9999` nghe trên MỌI card mạng, nên máy khác cùng Wi-Fi vào được bằng
	// `http://<IP-máy-này>:9999`. Muốn đóng lại, chỉ cho máy này chơi:
	// `go run ./cmd/server -addr 127.0.0.1:9999`.
	addr := flag.String("addr", ":9999", "địa chỉ nghe; \":9999\" = mở ra mạng LAN, \"127.0.0.1:9999\" = chỉ máy này")
	cfgPath := flag.String("config", config.FindRepoFile("config/game.json"), "config file")
	mapsDir := flag.String("maps", config.FindRepoFile("maps"), "Tiled maps directory")
	assetsDir := flag.String("assets", config.FindRepoFile("assets"), "assets directory")
	clientDir := flag.String("client", config.FindRepoFile("client/dist"), "built client directory")
	dbPath := flag.String("db", "", "SQLite file (default: server/data/leaderboard.db)")
	debug := flag.Bool("debug", true, "cho phép phím test; người chơi vẫn phải tự mở bằng ?debug=true")
	// Trần kết nối mỗi IP là để chống lạm dụng (§13.5). Chạy tools/browsertest thì 10 tab
	// cùng đến từ 127.0.0.1 nên phải nới, nếu không tab thứ 9 trở đi ăn HTTP 429.
	maxConns := flag.Int("max-conns-per-ip", 0, "ghi đè net.max_conns_per_ip (0 = theo config)")
	// Hồ sơ CPU cho công cụ đo: cmd/loadtest chạy trong ống nghiệm, còn đây là server thật
	// với bot, bản đồ thật và client thật — hai nơi cho ra hình ảnh CPU khác hẳn nhau.
	cpuProf := flag.String("cpuprofile", "", "ghi hồ sơ CPU ra file này cho tới khi tắt server")
	flag.Parse()

	if *cpuProf != "" {
		f, err := os.Create(*cpuProf)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	cfg, raw, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	maps, err := world.LoadDir(*mapsDir, world.LoadOptions{HP: cfg.Obstacles.HP, FootprintW: cfg.Obstacles.FootprintW, FootprintH: cfg.Obstacles.FootprintH})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d maps", len(maps))
	st := openStore(*dbPath, cfg)
	defer st.Close()

	if *maxConns > 0 {
		cfg.Net.MaxConnsPerIP = *maxConns
	}
	mgr := room.NewManager(cfg, maps, st)
	mgr.AllowDebug = *debug // người chơi vẫn phải tự mở bằng ?debug=true mới thấy bảng
	srv := &server{cfg: cfg, rawCfg: raw, mgr: mgr, store: st, maps: maps, debug: *debug,
		names: room.LoadNameFilter(cfg, filepath.Join(filepath.Dir(*cfgPath), "bad_words.txt")), limiter: newIPLimiter(cfg.Net.MaxConnsPerIP)}
	mux := srv.routes(*assetsDir, *clientDir)
	log.Printf("Kiếm Xoay server listening on %s (debug=%v)", *addr, *debug)
	httpSrv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	// Tắt êm khi nhận SIGINT/SIGTERM: cần thế thì hồ sơ CPU mới được ghi xuống đĩa. Bị
	// SIGKILL thì file hồ sơ rỗng.
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		if *cpuProf != "" {
			pprof.StopCPUProfile()
			log.Printf("đã ghi hồ sơ CPU vào %s", *cpuProf)
		}
		_ = httpSrv.Close()
	}()
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func openStore(path string, cfg *config.Config) *store.Store {
	if path == "" {
		repo := filepath.Dir(filepath.Dir(config.FindRepoFile("config/game.json")))
		dir := filepath.Join(repo, "server", "data")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
		path = filepath.Join(dir, "leaderboard.db")
	}
	st, err := store.Open(path, time.Duration(cfg.Net.LeaderboardCacheSc*float64(time.Second)))
	if err != nil {
		log.Fatal(err)
	}
	return st
}
