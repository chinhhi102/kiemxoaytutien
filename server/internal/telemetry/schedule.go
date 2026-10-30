package telemetry

import (
	"fmt"
	"strings"
	"time"
)

// Lịch tổng hợp tự động (TELEMETRY_SPEC §4.0) và cách đọc tham số `--since`.
//
// Đặt ở gói này chứ không ở `cmd/server` để TEST ĐƯỢC: `nextRollup` là chỗ dễ sai âm thầm
// nhất trong cả tính năng — trả về một mốc trong quá khứ thì bộ lịch quay vòng liên tục và
// ghi ra hàng loạt file tổng hợp rỗng, mà không có gì báo lỗi.

// rollupHours: giờ chạy tổng hợp, giờ Việt Nam (§4.0).
var rollupHours = []struct{ h, m int }{{12, 0}, {23, 30}}

// vnLocation: múi giờ Việt Nam, đặt cứng.
//
// Không dùng giờ hệ thống: server có thể chạy trên máy đặt UTC, và "12:00" trong spec là 12
// giờ trưa ở Việt Nam — đúng lúc người chơi nghỉ trưa vào chơi — chứ không phải 12 giờ ở
// một múi nào khác.
var vnLocation = time.FixedZone("ICT", 7*3600)

// VNLocation cho phần ngoài gói dùng chung đúng một múi giờ.
func VNLocation() *time.Location { return vnLocation }

// NextRollup trả về mốc tổng hợp kế tiếp SAU `now`. Luôn ở phía trước, không bao giờ trả về
// quá khứ hay đúng bằng `now`.
func NextRollup(now time.Time) time.Time { return nextRollup(now) }

func nextRollup(now time.Time) time.Time {
	now = now.In(vnLocation)
	for d := 0; d < 2; d++ {
		day := now.AddDate(0, 0, d)
		for _, r := range rollupHours {
			at := time.Date(day.Year(), day.Month(), day.Day(), r.h, r.m, 0, 0, vnLocation)
			if at.After(now) {
				return at
			}
		}
	}
	return now.Add(12 * time.Hour) // không bao giờ tới đây, nhưng đừng trả về quá khứ
}

// ParseSince đọc "24h", "7d", "90m".
func ParseSince(v string) (time.Duration, error) { return parseSince(v) }

func parseSince(v string) (time.Duration, error) {
	if strings.HasSuffix(v, "d") {
		var days int
		if _, err := fmt.Sscanf(v, "%dd", &days); err != nil {
			return 0, err
		}
		if days <= 0 {
			return 0, fmt.Errorf("số ngày phải lớn hơn 0")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(v)
}
