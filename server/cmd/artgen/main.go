// Command artgen generates game art with the minigen /jobs API and post-processes it
// into assets/ following docs/ART_PROMPTS.md.
//
//	go run ./cmd/artgen plan                 # list jobs
//	go run ./cmd/artgen run -p 2             # send each missing prompt once, one at a time (-gap 30s)
//	go run ./cmd/artgen prep                 # remove backgrounds, slice sheets, build atlases + manifest
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"kiemxoay/server/internal/config"
)

const pollEvery = 6 * time.Second

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: artgen plan|run|prep|jitter|prompts|status [flags]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	conc := fs.Int("c", 1, "concurrent minigen jobs (keep 1: minigen answers \"too many requests\")")
	gap := fs.Duration("gap", 30*time.Second, "pause between prompts")
	maxPri := fs.Int("p", 9, "only jobs with priority <= p")
	only := fs.String("only", "", "only jobs whose name contains this text (comma separated)")
	redo := fs.Bool("redo", false, "regenerate even if the raw image exists")
	limit := fs.Int("n", 60, "max images to generate this run (minigen allows ~60-100 per 5 hours)")
	_ = fs.Parse(os.Args[2:])

	assets := config.FindRepoFile("assets")
	// Sheet bị cắt sai thì phải sinh lại. Danh sách do chính `prep` ghi ra (xem
	// writeMiscut) chứ không gõ tay vào mã nguồn nữa — gõ tay thì cứ có sheet hỏng mới là
	// lại quên. Bộ cắt tìm khung bằng vùng pixel liền nhau, nên nhân vật có tay/chân rời bị
	// đếm thành nhiều khung; phát hiện 28/10/2026 khi `dai_ca_dep` idle ra 22 khung thay vì
	// 4 (trong game thành một cái áo tím nhấp nháy).
	for _, name := range readMiscut(assets) {
		redoReasons[name] = "sheet bị cắt sai số khung (xem cảnh báo của `artgen prep`)"
	}
	jobs := filterJobs(allJobs(), *maxPri, *only)
	switch os.Args[1] {
	case "plan":
		plan(jobs, assets)
	case "run":
		secrets := loadSecrets(config.FindRepoFile("docs/private/secret"))
		mg, err := newMinigen(secrets)
		if err != nil {
			log.Fatal(err)
		}
		run(mg, jobs, assets, runOpts{conc: *conc, gap: *gap, redo: *redo, limit: *limit})
	case "prep":
		prepAll(jobs, assets)
	case "jitter":
		// Chấm điểm độ giật cho mọi animation (GAME_SPEC §12.5 B.5). Tách thành lệnh riêng
		// chứ không gộp vào `prep`: `prep` phải chạy nhanh sau mỗi lần thả ảnh mới, còn
		// chấm điểm thì đọc lại toàn bộ ~900 frame.
		rep, err := buildJitter(assets)
		if err != nil {
			log.Fatal(err)
		}
		printJitter(rep, 0)
	case "prompts":
		writePrompts(jobs, assets, filepath.Join(filepath.Dir(assets), "docs", "ART_JOBS.md"))
	case "status":
		writeStatus(jobs, assets, filepath.Join(filepath.Dir(assets), "docs", "ASSETS_STATUS.md"))
	default:
		log.Fatalf("unknown command %q", os.Args[1])
	}
}

func filterJobs(jobs []Job, maxPri int, only string) []Job {
	var out []Job
	for _, j := range jobs {
		if j.Priority > maxPri || (only != "" && !matchesAny(j.Name, only)) {
			continue
		}
		out = append(out, j)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Priority < out[b].Priority })
	return out
}

func matchesAny(name, list string) bool {
	for _, s := range strings.Split(list, ",") {
		if s != "" && strings.Contains(name, s) {
			return true
		}
	}
	return false
}

func rawPath(assets string, j Job) string { return filepath.Join(assets, "_raw", j.Name+".png") }

// alreadyHave reports whether a job needs no quota: either its raw download is still around,
// or `prep` has already turned it into the finished asset. assets/_raw is a scratch folder
// that gets cleaned, so checking only there would re-generate art we already shipped.
//
// Jobs in redoReasons are the exception: their finished asset exists but was drawn under
// older rules, so they count as missing until a fresh raw image lands.
func alreadyHave(assets string, j Job) bool {
	if _, err := os.Stat(rawPath(assets, j)); err == nil {
		return true
	}
	if _, redo := redoReasons[j.Name]; redo {
		return false
	}
	return haveOutputs(assets, j)
}

// haveOutputs reports whether every file this job is supposed to produce is on disk.
//
// Việc sinh ảnh dạng sheet (nhân vật, quái, vật cản, cổng, VFX) KHÔNG dùng `j.Out` — đường
// dẫn nằm trong từng `Seg`. Trước đây chỗ này stat `filepath.Join(assets, "")`, tức chính
// thư mục assets, nên luôn có → **mọi job sheet đều bị tính là đã xong**. Đó là lý do
// `artgen plan` khoe "186/196 generated" trong khi `assets:report` vẫn thiếu 89 file: 9
// nhân vật chưa có levelup / đột phá / rớt cảnh giới / hồi sinh / vào cổng / ra cổng.
func haveOutputs(assets string, j Job) bool {
	if j.Mode != modeSheet {
		return j.Out != "" && exists(filepath.Join(assets, filepath.FromSlash(j.Out)))
	}
	for _, s := range j.Segs {
		if s.Out == "" {
			return false
		}
		// Cùng thước đo với `npm run assets:report`: đoạn hoạt ảnh `{anim}_{NN}` chỉ cần
		// khung đầu là coi như đã có art (thiếu vài khung cuối thì hoạt ảnh ngắn đi chứ
		// không phải chưa vẽ); đoạn đặt tên rõ (vật cản intact/cracked/...) phải đủ cả.
		for _, name := range segFileNames(s) {
			if !exists(filepath.Join(assets, filepath.FromSlash(s.Out), name)) {
				return false
			}
		}
	}
	return len(j.Segs) > 0
}

// segFileNames is what must exist for a segment to count as drawn.
func segFileNames(s Seg) []string {
	if len(s.Names) > 0 {
		out := make([]string, 0, len(s.Names))
		for _, n := range s.Names {
			out = append(out, n+".png")
		}
		return out
	}
	return []string{fmt.Sprintf("%s_00.png", s.Anim)}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// writeStatus lists which art exists and which is still a placeholder (M6 deliverable).
func writeStatus(jobs []Job, assets, out string) {
	var done, missing []string
	for _, j := range jobs {
		line := fmt.Sprintf("| `%s` | p%d | %s |", j.Name, j.Priority, jobOutputs(j))
		if alreadyHave(assets, j) {
			done = append(done, line)
		} else {
			missing = append(missing, line)
		}
	}
	var b strings.Builder
	b.WriteString("# Tình trạng art\n\nTạo lại file này: `cd server && go run ./cmd/artgen status`.\n")
	b.WriteString("Asset thiếu → game tự vẽ placeholder (hình tròn + chữ). Sinh thêm: `go run ./cmd/artgen run -p 2 -n 30` rồi `go run ./cmd/artgen prep`.\n")
	b.WriteString("Nếu tự tải ảnh từ Gemini: lưu thành `assets/_raw/<tên job>.png` rồi chạy `prep`.\n\n")
	fmt.Fprintf(&b, "## Còn thiếu (%d ảnh)\n\n| Job | Ưu tiên | Ra file |\n|---|---|---|\n%s\n\n", len(missing), strings.Join(missing, "\n"))
	fmt.Fprintf(&b, "## Đã có (%d ảnh)\n\n| Job | Ưu tiên | Ra file |\n|---|---|---|\n%s\n", len(done), strings.Join(done, "\n"))
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d done, %d missing → %s\n", len(done), len(missing), out)
}

// writePrompts exports every missing job's prompt so images can be made with any tool and
// dropped into assets/_raw/<job>.png before running `artgen prep`.
// redoReasons lists art that exists but must be regenerated because the rules changed.
var redoReasons = map[string]string{
	"tile_grass":    "spec v3: nền nhạt, ít chi tiết (ART_PROMPTS §9.0)",
	"tile_concrete": "spec v3: nền nhạt, ít chi tiết (ART_PROMPTS §9.0)",
	"tile_dirt":     "spec v3: nền nhạt, ít chi tiết (ART_PROMPTS §9.0)",
}

func init() {
	for i := range swords {
		redoReasons[fmt.Sprintf("sword_realm_%d", i)] = "spec v3.1: lưỡi kiếm trắng/xám để game nhuộm màu theo vòng (ART_PROMPTS §7)"
	}
}

func writePrompts(jobs []Job, assets, out string) {
	var b strings.Builder
	b.WriteString("# Danh sách prompt cần sinh ảnh\n\n")
	b.WriteString("Tạo lại file này: `cd server && go run ./cmd/artgen prompts` (chỉ liệt kê ảnh còn thiếu, xếp theo ưu tiên).\n\n")
	b.WriteString("**Cách dùng**\n")
	b.WriteString("1. Dán prompt vào công cụ sinh ảnh bất kỳ (Gemini, ChatGPT, Midjourney…). Nếu công cụ trả lời bằng chữ, thêm vào đầu: `Generate an image:`.\n")
	b.WriteString("2. Ảnh nên khổ ngang 16:9 (hoặc vuông với ảnh 1 món), độ phân giải càng cao càng tốt.\n")
	b.WriteString("3. Lưu đúng tên ở dòng **Lưu thành** (đuôi `.png`; ảnh JPEG đổi tên thành .png vẫn được).\n")
	b.WriteString("4. Chạy `cd server && go run ./cmd/artgen prep` → tool tự xóa nền magenta/trắng, cắt frame, căn pivot, ghép atlas.\n")
	b.WriteString("5. Sheet lệch nhẹ vẫn dùng được, tool cắt theo **vùng pixel liền nhau**.\n\n")
	b.WriteString("**Quan trọng với sheet nhân vật** — tool đếm khung bằng cách tìm các mảng pixel rời nhau, nên:\n")
	b.WriteString("- Mỗi khung phải là MỘT khối liền: tay, chân, vũ khí, nón phải DÍNH vào thân. Một cánh tay bay rời là tool đếm thành một khung nữa.\n")
	b.WriteString("- Giữa hai khung phải có khoảng trống nền rõ ràng, các khung không được chạm nhau.\n")
	b.WriteString("- Không vẽ thêm bóng đổ rời, hạt bụi, vệt tốc độ tách khỏi thân.\n")
	b.WriteString("- Sai số khung quá nhiều thì client **bỏ hẳn animation đó** và dùng tạm `idle`/`run`, nên các mục ghi \"sheet bị cắt sai số khung\" bên dưới cần sinh lại thật.\n\n")
	n := 0
	for _, j := range jobs {
		why, redo := redoReasons[j.Name]
		// Dùng CHUNG một phép thử với `plan`/`run`: còn raw trong scratch, hoặc đã prep ra
		// file asset thật. Trước đây chỗ này chỉ nhìn `assets/_raw` — mà đó là thư mục tạm
		// hay bị dọn — nên ART_JOBS.md liệt kê lại cả những ảnh đã vẽ xong từ lâu.
		if alreadyHave(assets, j) && !redo {
			continue
		}
		n++
		fmt.Fprintf(&b, "## %d. `%s` · ưu tiên %d\n\n", n, j.Name, j.Priority)
		if redo {
			fmt.Fprintf(&b, "- **Sinh lại**: %s\n", why)
		}
		fmt.Fprintf(&b, "- **Lưu thành**: `assets/_raw/%s.png`\n", j.Name)
		fmt.Fprintf(&b, "- **Bố cục**: %s\n", jobLayout(j))
		fmt.Fprintf(&b, "- **Ra file game**: %s\n\n", jobOutputs(j))
		fmt.Fprintf(&b, "```text\n%s\n```\n\n", j.Prompt)
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d prompts → %s\n", n, out)
}

func jobLayout(j Job) string {
	switch j.Mode {
	case modeSheet:
		parts := make([]string, len(j.Rows))
		for i, r := range j.Rows {
			parts[i] = fmt.Sprintf("hàng %d: %d hình", i+1, r)
		}
		return "sprite sheet, " + strings.Join(parts, " · ") + ", nền magenta #FF00FF"
	case modeTile:
		return "texture vuông lặp liền mạch, phủ kín ảnh"
	case modeWide:
		return "tranh ngang 16:9, có nền"
	default:
		return "1 hình ở giữa, nền magenta #FF00FF"
	}
}

func jobOutputs(j Job) string {
	if j.Mode != modeSheet {
		return j.Out
	}
	var parts []string
	for _, s := range j.Segs {
		parts = append(parts, fmt.Sprintf("%s (%s ×%d)", s.Out, s.Anim, s.Frames))
	}
	return strings.Join(parts, ", ")
}

func plan(jobs []Job, assets string) {
	done := 0
	for _, j := range jobs {
		mark := " "
		if alreadyHave(assets, j) {
			mark, done = "x", done+1
		}
		fmt.Printf("[%s] p%d %-32s → %s\n", mark, j.Priority, j.Name, j.Out)
	}
	fmt.Printf("%d jobs, %d generated\n", len(jobs), done)
}

// state remembers submitted job ids so a restarted run resumes polling instead of resubmitting.
type state struct {
	mu   sync.Mutex
	path string
	IDs  map[string]string `json:"ids"`
}

func loadState(path string) *state {
	s := &state{path: path, IDs: map[string]string{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, s)
	}
	return s
}

func (s *state) set(name, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		delete(s.IDs, name)
	} else {
		s.IDs[name] = id
	}
	b, _ := json.MarshalIndent(s, "", " ")
	_ = os.WriteFile(s.path, b, 0o644)
}

func (s *state) get(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.IDs[name]
}

type runOpts struct {
	conc, limit int
	redo        bool
	gap         time.Duration
}

func run(mg *minigen, jobs []Job, assets string, o runOpts) {
	if err := os.MkdirAll(filepath.Join(assets, "_raw"), 0o755); err != nil {
		log.Fatal(err)
	}
	st := loadState(filepath.Join(assets, "_raw", "state.json"))
	queue := make(chan Job)
	var wg sync.WaitGroup
	var okN, failN int
	var mu sync.Mutex
	for w := 0; w < o.conc; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range queue {
				err := generate(mg, st, j, rawPath(assets, j))
				mu.Lock()
				if err != nil {
					failN++
					log.Printf("FAIL %s: %v", j.Name, err)
				} else {
					okN++
					log.Printf("ok   %s (%d ok, %d fail)", j.Name, okN, failN)
				}
				mu.Unlock()
			}
		}()
	}
	sent := 0
	for _, j := range jobs {
		if alreadyHave(assets, j) && !o.redo {
			continue
		}
		if sent >= o.limit {
			log.Printf("reached -n %d; run again later for the rest", o.limit)
			break
		}
		if sent > 0 {
			time.Sleep(o.gap)
		}
		queue <- j
		sent++
	}
	close(queue)
	wg.Wait()
	log.Printf("done: %d ok, %d failed", okN, failN)
}

// generate sends the prompt to minigen once and downloads its image. A prompt is sent
// again only when minigen refused it for quota (nothing was drawn): we then wait until
// reset_at. A job that timed out keeps its id so the next run resumes it.
func generate(mg *minigen, st *state, j Job, dest string) error {
	for waits := 0; ; waits++ {
		reset, err := generateOnce(mg, st, j, dest)
		if reset == "" || waits >= 3 {
			return err
		}
		waitQuota(reset)
	}
}

// generateOnce returns the quota reset time when the job was refused for quota.
func generateOnce(mg *minigen, st *state, j Job, dest string) (string, error) {
	id := st.get(j.Name)
	if id == "" {
		out, err := submitOnce(mg, j.Prompt)
		if err != nil {
			return "", err
		}
		id = out.ID
		st.set(j.Name, id)
	}
	out, err := waitJob(mg, id)
	if err != nil {
		// Job BIẾN MẤT phía minigen (404 "not found or expired") thì giữ id lại là tự hại:
		// id đó không bao giờ hồi lại được, nên mọi lần chạy sau đều hỏng y hệt và ảnh ấy
		// kẹt vĩnh viễn. `plate_shine_streak` đã kẹt đúng hai đợt vì chuyện này.
		//
		// Khác hẳn trường hợp hết giờ chờ: ở đó job vẫn đang vẽ, giữ id để lần sau lấy về
		// là đúng — và cũng là để không tốn thêm một lượt quota vẽ lại.
		if jobGone(err) {
			st.set(j.Name, "")
			return "", err
		}
		if out == nil || out.Status != "failed" {
			return "", err // id kept: resumed next run
		}
		st.set(j.Name, "")
		if strings.Contains(strings.ToLower(out.Error), "quota") {
			return out.ResetAt, err
		}
		return "", err
	}
	if err := mg.download(out.imageURL(), dest); err != nil {
		return "", err // id kept: the next run downloads it again without a new prompt
	}
	st.set(j.Name, "")
	return "", nil
}

func waitQuota(reset string) {
	d := 10 * time.Minute
	if t, err := time.Parse(time.RFC3339, reset); err == nil {
		d = max(time.Until(t)+30*time.Second, time.Minute)
	}
	log.Printf("quota locked until %s, waiting %s", reset, d.Round(time.Second))
	time.Sleep(d)
}

// submitOnce retries only while the service refuses to accept the job.
func submitOnce(mg *minigen, prompt string) (*jobOut, error) {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		out, err := mg.submit(prompt)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !rejected(err) {
			return nil, err
		}
		d := backoff(attempt) * 4
		log.Printf("minigen busy (%v), waiting %s before sending again", truncate(err.Error(), 80), d)
		time.Sleep(d)
	}
	return nil, lastErr
}

func rejected(err error) bool {
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "http 429") || strings.Contains(m, "too many") || strings.Contains(m, "quota")
}

// jobGone: minigen báo job này không còn tồn tại, tức là chờ nữa cũng vô ích.
//
// Nhận theo CẢ mã HTTP lẫn câu chữ: mã 404 là tín hiệu chắc chắn, còn câu "not found or
// expired" phòng trường hợp minigen đổi sang trả 410 hay 400 — thà bắt rộng một chút còn
// hơn để một ảnh kẹt vĩnh viễn mà không ai biết vì sao.
func jobGone(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "http 404") ||
		strings.Contains(s, "not found or expired") ||
		strings.Contains(s, "job not found")
}

func waitJob(mg *minigen, id string) (*jobOut, error) {
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		out, err := mg.get(id)
		if err != nil {
			return nil, err
		}
		switch out.Status {
		case "completed":
			if out.imageURL() == "" {
				return out, fmt.Errorf("completed without image")
			}
			return out, nil
		case "failed":
			return out, fmt.Errorf("job failed: %s", out.Error)
		default:
			time.Sleep(pollEvery)
		}
	}
	return nil, fmt.Errorf("job %s timed out", id)
}

func backoff(attempt int) time.Duration { return time.Duration(15*(attempt+1)) * time.Second }
