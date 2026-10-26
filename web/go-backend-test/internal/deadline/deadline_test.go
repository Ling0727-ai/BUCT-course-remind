package deadline

import (
	"testing"
	"time"
)

func TestDeadlineTimezoneAndFormats(t *testing.T) {
	want := time.Date(2026, 10, 26, 4, 0, 0, 0, time.UTC)
	for _, value := range []string{"2026-10-26 12:00:00", "2026年10月26日 12:00", "2026-10-26T12:00:00+08:00", "2026-10-26T04:00:00Z"} {
		got, err := Parse(value)
		if err != nil || !got.Equal(want) {
			t.Fatalf("%q: got %v (%v)", value, got, err)
		}
	}
	if _, err := Parse("没有截止日期"); err == nil {
		t.Fatal("invalid deadlines must not be scheduled")
	}
}
