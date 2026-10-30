// Command perfreport chạy kịch bản chuẩn S1 (PERF_LOG §1) và in sẵn **một dòng bảng §3 +
// một dòng CSV**, để việc ghi sổ hiệu năng không phụ thuộc chép tay.
//
//	cd server && go run ./cmd/perfreport                    # 3 lần, lấy trung vị
//	cd server && go run ./cmd/perfreport -runs 1 -label "v6 bước 2"
//
// Vì sao chỉ S1: nó là kịch bản DUY NHẤT chạy hoàn toàn trong tiến trình, không cần trình
// duyệt, nên cho ra con số lặp lại được. S2/S3 cần 10 tab Chrome thật — chạy bằng
// `node tools/browsertest/run.mjs --scenario S2 --net normal|slow4g` rồi điền tay các cột
// FPS / draw call / băng thông client. Lệnh này in sẵn khung dòng để điền vào.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type s1 struct {
	tickAvg, tickP95, pktKB, kbps, pingP95, cpuPct float64
}

var numRe = regexp.MustCompile(`([0-9]+\.[0-9]+|[0-9]+)`)

// pick lấy con số ĐẦU TIÊN trên dòng chứa `label`. Bảng của loadtest có dạng
// "Tick server (trung bình)   1.49 ms   < 4 ms ĐẠT" — số đầu là số đo, các số sau là ngân sách.
func pick(out, label string) float64 {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, label) {
			continue
		}
		rest := line[strings.Index(line, label)+len(label):]
		if m := numRe.FindString(rest); m != "" {
			v, _ := strconv.ParseFloat(m, 64)
			return v
		}
	}
	return 0
}

func runS1(seed int) (s1, error) {
	cmd := exec.Command("go", "run", "./cmd/loadtest", "--scenario", "S1", "--seed", strconv.Itoa(seed))
	b, err := cmd.CombinedOutput()
	out := string(b)
	if err != nil {
		return s1{}, fmt.Errorf("loadtest lỗi: %v\n%s", err, out)
	}
	return s1{
		tickAvg: pick(out, "Tick server (trung bình)"),
		tickP95: pick(out, "Tick server (p95)"),
		pktKB:   pick(out, "Snapshot lớn nhất / client"),
		kbps:    pick(out, "Băng thông xuống / client"),
		pingP95: pick(out, "Ping p95"),
		cpuPct:  pick(out, "CPU dùng"),
	}, nil
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[len(s)/2]
}

// mark đánh dấu đạt/vượt ngân sách PERF_LOG §2.
func mark(v, budget float64) string {
	if v <= budget {
		return fmt.Sprintf("%.2f ✅", v)
	}
	return fmt.Sprintf("%.2f ❌", v)
}

func main() {
	runs := flag.Int("runs", 3, "số lần chạy, lấy trung vị (PERF_LOG §1 yêu cầu 3)")
	seed := flag.Int("seed", 7, "seed của kịch bản chuẩn")
	label := flag.String("label", "", "tên lượt, điền vào cột 'Lượt / thay đổi'")
	num := flag.String("n", "?", "số thứ tự lượt trong bảng §3")
	flag.Parse()

	var tickAvg, tickP95, pkt, kbps, ping, cpu []float64
	for i := 0; i < *runs; i++ {
		fmt.Fprintf(os.Stderr, "S1 lần %d/%d...\n", i+1, *runs)
		r, err := runS1(*seed)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		tickAvg = append(tickAvg, r.tickAvg)
		tickP95 = append(tickP95, r.tickP95)
		pkt = append(pkt, r.pktKB)
		kbps = append(kbps, r.kbps)
		ping = append(ping, r.pingP95)
		cpu = append(cpu, r.cpuPct)
	}

	commit := "?"
	if b, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		commit = strings.TrimSpace(string(b))
	}
	date := time.Now().Format("2006-01-02")
	ta, tp := median(tickAvg), median(tickP95)
	pk, kb, pg, cp := median(pkt), median(kbps), median(ping), median(cpu)

	fmt.Printf("\nTrung vị %d lần · S1 · commit %s\n\n", *runs, commit)
	fmt.Println("Dòng bảng §3 (dán vào docs/PERF_LOG.md):")
	fmt.Printf("| %s | %s | %s | %s | S1 | %s / %s | %.1f / – | %s | – | – | – | – | – | – | – |\n",
		*num, date, commit, *label,
		mark(ta, 3), mark(tp, 5), kb, mark(pk, 6))

	fmt.Println("\nDòng CSV (dán vào docs/perf_log.csv):")
	fmt.Printf("%s,%s,%s,%q,S1,%.2f,%.2f,%.0f,%.1f,,%.1f,%.2f,,,,,,,,,,,,\n",
		*num, date, commit, *label, ta, tp, cp, kb, pk, pg)

	fmt.Fprintln(os.Stderr, "\nS2/S3 chạy riêng (cần Chrome thật):")
	fmt.Fprintln(os.Stderr, "  node tools/browsertest/run.mjs --scenario S2 --net normal")
	fmt.Fprintln(os.Stderr, "  node tools/browsertest/run.mjs --scenario S2 --net slow4g")
}
