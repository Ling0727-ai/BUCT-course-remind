package scheduling

import "time"

const AutoReminderVersion = 3
const AutoReminderSendAhead = 5 * time.Second
const AutoReminderPrepareAhead = 30 * time.Second

func AutoReminderTime(deadline time.Time, hours int) time.Time {
	return deadline.Add(-time.Duration(hours)*time.Hour - AutoReminderSendAhead)
}

func AutoReminderPreparationTime(deadline time.Time) time.Time {
	return AutoReminderTime(deadline, 24).Add(-AutoReminderPrepareAhead)
}

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
			until = AutoReminderTime(deadline, offsets[i+1])
		}
		if now.Before(until) {
			stages = append(stages, AutoReminderStage{Hours: hours, At: AutoReminderTime(deadline, hours), Until: until})
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
