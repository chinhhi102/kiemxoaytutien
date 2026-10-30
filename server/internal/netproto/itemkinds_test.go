package netproto

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"kiemxoay/server/internal/config"
)

// Mọi loại vật phẩm game sinh ra đều phải có trong `ItemKinds`.
//
// Thiếu một loại thì `ItemKindID` trả 0 và client vẽ nó thành viên Linh khí — **không có
// lỗi, không có log**, chỉ là Bí Kíp bỗng trông như viên EXP. Đúng kiểu hỏng mà chỉ phát
// hiện được khi có người chơi kêu.
func TestItemKindsCoverConfig(t *testing.T) {
	cfg := config.MustLoadDefault()
	for kind := range cfg.Items.Weights {
		if _, ok := itemKindIndex[kind]; !ok {
			t.Errorf("loại vật phẩm %q có trong config nhưng thiếu trong ItemKinds", kind)
		}
	}
	for name := range cfg.Buffs {
		if _, ok := itemKindIndex[name]; !ok {
			t.Errorf("buff %q rơi ra thành vật phẩm nhưng thiếu trong ItemKinds", name)
		}
	}
	// Mấy loại không nằm trong bảng trọng số nhưng vẫn rơi ra đất.
	for _, kind := range []string{"orb", "bundle", "tien_khi"} {
		if _, ok := itemKindIndex[kind]; !ok {
			t.Errorf("loại %q thiếu trong ItemKinds", kind)
		}
	}
}

// Bảng bên client phải khớp TỪNG DÒNG với bảng bên server: loại gửi qua mạng bằng chỉ số,
// nên lệch một dòng là mọi vật phẩm vẽ sai loại. Đọc thẳng file TypeScript để so — chép tay
// hai bảng rồi tin là chúng giống nhau thì đúng một lần quên là hỏng.
func TestItemKindsMatchClient(t *testing.T) {
	b, err := os.ReadFile(config.FindRepoFile("client/src/net/protocol.ts"))
	if err != nil {
		t.Skipf("không đọc được protocol.ts: %v", err)
	}
	m := regexp.MustCompile(`(?s)ITEM_KINDS = \[(.*?)\]`).FindSubmatch(b)
	if m == nil {
		t.Fatal("không tìm thấy ITEM_KINDS trong client/src/net/protocol.ts")
	}
	var got []string
	for _, part := range strings.Split(string(m[1]), ",") {
		if s := strings.Trim(strings.TrimSpace(part), "'\"\n\t "); s != "" {
			got = append(got, s)
		}
	}
	if len(got) != len(ItemKinds) {
		t.Fatalf("client có %d loại, server có %d: %v vs %v", len(got), len(ItemKinds), got, ItemKinds)
	}
	for i := range ItemKinds {
		if got[i] != ItemKinds[i] {
			t.Errorf("dòng %d: client %q, server %q", i, got[i], ItemKinds[i])
		}
	}
}
