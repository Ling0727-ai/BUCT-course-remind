package scheduling

import "time"

const AutoReminderVersion = 2

type AutoReminderStage struct {
	Hours int
	At    time.Time
	Until time.Time
}

// AutoReminderStages retains future stages and at most one overdue stage.
// Each stage expires when the next stage becomes due, preventing restart bursts.
func AutoReminderStages(deadline, now time.Time) []AutoReminderStage {
	offsets := [...]int{24, 12, 1}
	stages := make([]AutoReminderStage, 0, len(offsets))
	for i, hours := range offsets {
		until := deadline
		if i+1 < len(offsets) {
			until = deadline.Add(-time.Duration(offsets[i+1]) * time.Hour)
		}
		if now.Before(until) {
			stages = append(stages, AutoReminderStage{Hours: hours, At: deadline.Add(-time.Duration(hours) * time.Hour), Until: until})
		}
	}
	return stages
}

func AutoReminderStageExpired(hours int, deadline, now time.Time) bool {
	if hours == 0 {
		hours = 24
	} // Legacy reminders represent the original 24-hour stage.
	for _, stage := range AutoReminderStages(deadline, now) {
		if stage.Hours == hours {
			return false
		}
	}
	return true
}
