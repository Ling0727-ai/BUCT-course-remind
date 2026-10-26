package scheduling

import (
	"slices"
	"testing"
	"time"
)

func TestThreeReminderStagesUseExactDeadlineOffsets(t *testing.T) {
	deadline := time.Date(2026, 10, 30, 18, 37, 42, 0, time.FixedZone("CST", 8*60*60))
	stages := AutoReminderStages(deadline, deadline.Add(-25*time.Hour))
	if len(stages) != 3 {
		t.Fatalf("got %d stages", len(stages))
	}
	for i, hours := range []int{24, 12, 1} {
		if stages[i].Hours != hours || !stages[i].At.Equal(deadline.Add(-time.Duration(hours)*time.Hour)) || stages[i].At.Second() != 42 {
			t.Fatalf("wrong timestamp for %dh stage: %v", hours, stages[i])
		}
	}
}

func TestLateDiscoveryAndRestartDoNotProduceReminderBursts(t *testing.T) {
	deadline := time.Date(2026, 10, 30, 18, 37, 42, 0, time.UTC)
	for _, test := range []struct {
		left time.Duration
		want []int
	}{
		{25 * time.Hour, []int{24, 12, 1}},
		{24 * time.Hour, []int{24, 12, 1}},
		{12 * time.Hour, []int{12, 1}},
		{2 * time.Hour, []int{12, 1}},
		{time.Hour, []int{1}},
		{30 * time.Minute, []int{1}},
		{0, []int{}},
		{-time.Second, []int{}},
	} {
		stages := AutoReminderStages(deadline, deadline.Add(-test.left))
		got := make([]int, 0, len(stages))
		overdue := 0
		for _, stage := range stages {
			got = append(got, stage.Hours)
			if !stage.At.After(deadline.Add(-test.left)) {
				overdue++
			}
		}
		if !slices.Equal(got, test.want) || overdue > 1 {
			t.Fatalf("remaining %v: stages=%v overdue=%d", test.left, got, overdue)
		}
	}
}
