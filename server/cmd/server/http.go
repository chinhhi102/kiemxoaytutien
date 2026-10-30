package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/room"
	"kiemxoay/server/internal/store"
	"kiemxoay/server/internal/telemetry"
	"kiemxoay/server/internal/world"
)

type server struct {
	cfg     *config.Config
	rawCfg  []byte
	mgr     *room.Manager
	store   *store.Store
	maps    []*world.Map
	debug   bool
	names   *room.NameFilter
	limiter *ipLimiter
	// telemetry là nil khi chạy với -telemetry=false; mọi chỗ dùng phải kiểm nil.
	telemetry *telemetry.Store
}

func (s *server) routes(assetsDir, clientDir string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/leaderboard", s.handleLeaderboard)
	mux.HandleFunc("GET /api/rooms", s.handleRooms)
	mux.HandleFunc("GET /api/map/{id}", s.handleMap)
	mux.HandleFunc("GET /ws", s.handleWS)
	// Tóm tắt phiên lúc đóng tab: sendBeacon chỉ gửi được POST, và lúc đó WebSocket đã đứt.
	mux.HandleFunc("POST /telemetry/end", s.handleTelemetryEnd)
	mux.HandleFunc("GET /admin/perf", s.handleAdminPerf)
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", noListing(http.FileServer(http.Dir(assetsDir)))))
	mux.Handle("GET /", spa(clientDir))
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func (s *server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(s.rawCfg)
}

// handleLeaderboard: GET /api/leaderboard?period=day|week|all&board=score|hits|speed (spec §11.2).
func (s *server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	period, board := r.URL.Query().Get("period"), r.URL.Query().Get("board")
	if period == "" {
		period = "all"
	}
	if board == "" {
		board = "score" // Uy Danh là bảng chính
	}
	rows, err := s.store.Top(period, board, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"period": period, "board": board, "rows": rows})
}

func (s *server) handleRooms(w http.ResponseWriter, _ *http.Request) { writeJSON(w, s.mgr.Rooms()) }

// handleMap serves the parsed map so the client does not need its own Tiled parser.
func (s *server) handleMap(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for _, m := range s.maps {
		if m.ID == id {
			w.Header().Set("Cache-Control", "public, max-age=60")
			writeJSON(w, m)
			return
		}
	}
	if id == "empty" {
		writeJSON(w, world.EmptyMap(s.cfg.Room.MapSize))
		return
	}
	http.NotFound(w, r)
}

func noListing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") || strings.HasPrefix(r.URL.Path, "_raw") {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// spa serves the built client and falls back to index.html.
//
// index.html is never cached: its <script> points at a content-hashed bundle, so a plain F5
// after a new build always picks the new game up. The hashed files under /static/ carry a
// new name whenever they change, so they are safe to cache for a long time.
func spa(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
				http.Error(w, "Client chưa được build. Chạy: cd client && npm run build (hoặc dùng npm run dev)", http.StatusNotFound)
				return
			}
			w.Header().Set("Cache-Control", "no-store, must-revalidate")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fs.ServeHTTP(w, r)
	})
}

// ipLimiter caps simultaneous WebSocket connections per IP (spec §13.5).
type ipLimiter struct {
	mu    sync.Mutex
	max   int
	count map[string]int
}

func newIPLimiter(max int) *ipLimiter { return &ipLimiter{max: max, count: map[string]int{}} }

func (l *ipLimiter) acquire(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.count[ip] >= l.max {
		return false
	}
	l.count[ip]++
	return true
}

func (l *ipLimiter) release(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.count[ip]--; l.count[ip] <= 0 {
		delete(l.count, ip)
	}
}

// clientIP honours X-Forwarded-For only from a reverse proxy on this host or its docker
// network (Caddy); the game port is never published publicly.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	isProxy := ip != nil && (ip.IsLoopback() || ip.IsPrivate())
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" && isProxy {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return host
}
