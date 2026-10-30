package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Đọc `proto/schema.txt` — NETCODE_SPEC §4.9.
//
// Tự viết bộ đọc thay vì thêm thư viện YAML: dự án có luật không thêm thư viện khi chưa hỏi,
// và spec chỉ nói "vd proto/schema.yaml" chứ không chỉ đích danh thư viện nào. Định dạng
// dòng đơn giản này còn dễ đọc hơn YAML cho người không rành kỹ thuật, và bộ đọc gọn tới
// mức không cần ai bảo trì.

// Field là một trường trong message.
type Field struct {
	Name string
	Type string
	// BitmaskFor: trường này là bitmask điều khiển các trường có tên trong danh sách.
	BitmaskFor []string
	// If: trường này chỉ xuất hiện khi bit cùng tên bật trong bitmask đứng trước.
	If      string
	Comment string
}

// Message là một loại gói.
type Message struct {
	Name   string
	ID     int
	Doc    []string
	Fields []Field
}

// Schema là toàn bộ file.
type Schema struct {
	Version  int
	Messages []Message
	// Hash: băm của NỘI DUNG schema, nhúng vào cả hai phía và kiểm lúc bắt tay (§4.9).
	// Lệch là từ chối kết nối — thà báo lỗi rõ còn hơn đọc rác trong im lặng.
	Hash string
}

// sizeOf trả về số byte của một kiểu cố định; `str` trả 0 vì nó dài thay đổi.
var sizeOf = map[string]int{"u8": 1, "i8": 1, "u16": 2, "i16": 2, "u32": 4, "i32": 4, "f32": 4, "str": 0}

// goType / tsType ánh xạ kiểu schema sang kiểu ngôn ngữ.
var goType = map[string]string{
	"u8": "uint8", "i8": "int8", "u16": "uint16", "i16": "int16",
	"u32": "uint32", "i32": "int32", "f32": "float32", "str": "string",
}

func tsType(t string) string {
	if t == "str" {
		return "string"
	}
	return "number"
}

// parseSchema đọc file schema.
func parseSchema(path string) (*Schema, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	s := &Schema{Hash: hex.EncodeToString(sum[:])[:16]}

	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var cur *Message
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		// Bỏ chú thích đuôi dòng, nhưng KHÔNG bỏ dòng `doc` (chú thích trong doc là nội dung).
		body := strings.TrimSpace(text)
		if body == "" || strings.HasPrefix(body, "#") {
			continue
		}
		words := strings.Fields(body)
		switch words[0] {
		case "version":
			if len(words) < 2 {
				return nil, fmt.Errorf("dòng %d: `version` thiếu số", line)
			}
			s.Version, _ = strconv.Atoi(words[1])
		case "message":
			if len(words) < 3 {
				return nil, fmt.Errorf("dòng %d: `message` cần tên và id", line)
			}
			id, err := strconv.Atoi(words[2])
			if err != nil {
				return nil, fmt.Errorf("dòng %d: id %q không phải số", line, words[2])
			}
			s.Messages = append(s.Messages, Message{Name: words[1], ID: id})
			cur = &s.Messages[len(s.Messages)-1]
		case "doc":
			if cur == nil {
				return nil, fmt.Errorf("dòng %d: `doc` nằm ngoài message", line)
			}
			cur.Doc = append(cur.Doc, strings.TrimSpace(strings.TrimPrefix(body, "doc")))
		default:
			if cur == nil {
				return nil, fmt.Errorf("dòng %d: trường %q nằm ngoài message", line, words[0])
			}
			f, err := parseField(body, line)
			if err != nil {
				return nil, err
			}
			cur.Fields = append(cur.Fields, f)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return s, validate(s)
}

func parseField(body string, line int) (Field, error) {
	var f Field
	if i := strings.Index(body, "#"); i >= 0 {
		f.Comment = strings.TrimSpace(body[i+1:])
		body = strings.TrimSpace(body[:i])
	}
	w := strings.Fields(body)
	if len(w) < 2 {
		return f, fmt.Errorf("dòng %d: trường cần `<tên> <kiểu>`", line)
	}
	f.Name, f.Type = w[0], w[1]
	if _, ok := sizeOf[f.Type]; !ok {
		return f, fmt.Errorf("dòng %d: kiểu %q không biết", line, f.Type)
	}
	for i := 2; i < len(w); i++ {
		switch w[i] {
		case "bitmask":
			if i+1 >= len(w) {
				return f, fmt.Errorf("dòng %d: `bitmask` thiếu danh sách trường", line)
			}
			f.BitmaskFor = strings.Split(w[i+1], ",")
			i++
		case "if":
			if i+1 >= len(w) {
				return f, fmt.Errorf("dòng %d: `if` thiếu tên bit", line)
			}
			f.If = w[i+1]
			i++
		default:
			return f, fmt.Errorf("dòng %d: không hiểu %q", line, w[i])
		}
	}
	return f, nil
}

// validate bắt những lỗi mà generator không tự phát hiện được nhưng sẽ gây hỏng lúc chạy.
func validate(s *Schema) error {
	if len(s.Messages) == 0 {
		return fmt.Errorf("schema không có message nào")
	}
	ids := map[int]string{}
	for _, m := range s.Messages {
		if old, dup := ids[m.ID]; dup {
			// Hai message cùng id là thảm hoạ im lặng: client đọc gói này tưởng là gói kia.
			return fmt.Errorf("id %d dùng cho cả %s và %s", m.ID, old, m.Name)
		}
		ids[m.ID] = m.Name
		if m.ID <= 0 || m.ID > 255 {
			return fmt.Errorf("%s: id %d nằm ngoài 1..255 (id đi vào 1 byte)", m.Name, m.ID)
		}
		names := map[string]bool{}
		var bits []string
		for _, f := range m.Fields {
			if names[f.Name] {
				return fmt.Errorf("%s: trường %q khai hai lần", m.Name, f.Name)
			}
			names[f.Name] = true
			if len(f.BitmaskFor) > 0 {
				bits = f.BitmaskFor
				if n := sizeOf[f.Type] * 8; len(bits) > n {
					return fmt.Errorf("%s: bitmask %s có %d bit nhưng kiểu %s chỉ chứa %d",
						m.Name, f.Name, len(bits), f.Type, n)
				}
			}
			if f.If != "" && !contains(bits, f.If) {
				return fmt.Errorf("%s: trường %q dùng `if %s` nhưng %q không có trong bitmask nào đứng trước",
					m.Name, f.Name, f.If, f.If)
			}
		}
		for _, b := range bits {
			if !names[b] {
				// Bit điều khiển một trường không tồn tại: generator vẫn sinh được, nhưng
				// trường đó sẽ không bao giờ được ghi — delta mất dữ liệu trong im lặng.
				return fmt.Errorf("%s: bitmask nhắc tới trường %q không có", m.Name, b)
			}
		}
	}
	sort.Slice(s.Messages, func(i, j int) bool { return s.Messages[i].ID < s.Messages[j].ID })
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// bitIndex trả về vị trí bit của một trường trong bitmask của message.
func (m Message) bitIndex(field string) int {
	for _, f := range m.Fields {
		if len(f.BitmaskFor) == 0 {
			continue
		}
		for i, b := range f.BitmaskFor {
			if b == field {
				return i
			}
		}
	}
	return -1
}

// maskField trả về tên trường bitmask của message (rỗng nếu không có).
func (m Message) maskField() string {
	for _, f := range m.Fields {
		if len(f.BitmaskFor) > 0 {
			return f.Name
		}
	}
	return ""
}
