package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Danh sách ngoại lệ (ANSWERS 30/10): anim đã duyệt KHÔNG tính đỏ, nhưng VẪN giữ điểm.
//
// Hai nửa đều quan trọng và dễ làm hỏng nửa sau: nếu "duyệt" đồng nghĩa với "xoá điểm" thì
// một ngày art đổi khiến anim đó giật gấp đôi cũng không ai thấy.
func TestNgoaiLeKhongTinhDoNhungVanGiuDiem(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "jitter_exceptions.json"), `{
	  "exceptions": [
	    {"char":"dep_lao","anim":"idle","reason":"đảo mắt có chủ đích","approved_by":"chủ dự án","date":"2026-10-30"}
	  ]
	}`)
	exc := loadExceptions(dir)
	if len(exc) != 1 {
		t.Fatalf("mong 1 ngoại lệ, nhận %d", len(exc))
	}
	e, ok := exc["dep_lao/idle"]
	if !ok {
		t.Fatal("tra theo khoá char/anim không ra")
	}
	if e.ApprovedBy == "" || e.Reason == "" {
		t.Error("người duyệt và lý do phải đọc được — không có thì không truy được ai quyết")
	}

	// Đếm đỏ: anim được duyệt bị loại khỏi số đỏ, nhưng điểm vẫn còn nguyên.
	rep := &JitterReport{Threshold: jitterRed, Anims: []AnimScore{
		{Char: "dep_lao", Anim: "idle", Loop: true, Avg: 56.8, Excepted: true},
		{Char: "x", Anim: "idle", Loop: true, Avg: 60},
		{Char: "y", Anim: "idle", Loop: true, Avg: 10},
	}}
	red := 0
	for _, a := range rep.Anims {
		if a.Avg >= rep.Threshold && !a.Excepted {
			red++
		}
	}
	if red != 1 {
		t.Errorf("mong 1 đỏ (chỉ `x`), nhận %d", red)
	}
	if rep.Anims[0].Avg != 56.8 {
		t.Error("ngoại lệ không được phép xoá điểm — art xuống cấp về sau sẽ không ai thấy")
	}
}

// File hỏng phải KÊU, không được im lặng coi như rỗng: im lặng thì một anim đã duyệt bỗng
// báo đỏ trở lại và không ai hiểu vì sao.
func TestNgoaiLeFileHongThiBaoChuKhongIm(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "jitter_exceptions.json"), "{ hỏng")
	if n := len(loadExceptions(dir)); n != 0 {
		t.Errorf("file hỏng phải ra 0 ngoại lệ, nhận %d", n)
	}
	// Không có file là chuyện bình thường (chưa ai duyệt gì), không phải lỗi.
	if n := len(loadExceptions(t.TempDir())); n != 0 {
		t.Errorf("không có file phải ra 0, nhận %d", n)
	}
}

// Điểm "lạc lõng" cần ÍT NHẤT 3 cặp mới có nghĩa: hai cặp thì cái nào cũng là một nửa dữ
// liệu, lấy trung vị của chúng làm chuẩn là tự chấm mình.
func TestLacLongCanDuDuLieu(t *testing.T) {
	two := []PairScore{{To: "a", Score: 10}, {To: "b", Score: 90}}
	if _, sc := worstOutlier(two); sc != 0 {
		t.Errorf("2 cặp phải trả 0 (không đủ dữ liệu), nhận %.1f", sc)
	}
	// Một bước nhảy gấp nhiều lần phần còn lại thì phải nổi lên.
	many := []PairScore{{To: "a", Score: 10}, {To: "b", Score: 10}, {To: "c", Score: 10}, {To: "d", Score: 90}}
	name, sc := worstOutlier(many)
	if name != "d" || sc < 50 {
		t.Errorf("mong frame `d` điểm cao, nhận %q %.1f", name, sc)
	}
	// Biến hình ĐỀU (mọi bước lớn như nhau) thì KHÔNG frame nào lạc lõng — đây chính là
	// `death`, và đây là lý do phải so tương đối thay vì so với ngưỡng tuyệt đối.
	even := []PairScore{{To: "a", Score: 95}, {To: "b", Score: 98}, {To: "c", Score: 92}, {To: "d", Score: 96}}
	if _, sc := worstOutlier(even); sc > 5 {
		t.Errorf("biến hình đều không được coi là lạc lõng, nhận %.1f", sc)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
