package deadline

import (
	"fmt"
	"strings"
	"time"
)

// Campus timestamps without an offset are always China Standard Time, even in UTC containers.
var Campus = time.FixedZone("Asia/Shanghai", 8*60*60)

func Parse(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05",
		"2006-1-2 15:04:05", "2006-1-2 15:04", "2006年1月2日 15:04:05", "2006年1月2日 15:04",
	} {
		if t, err := time.ParseInLocation(layout, value, Campus); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析截止时间: %q", value)
}
