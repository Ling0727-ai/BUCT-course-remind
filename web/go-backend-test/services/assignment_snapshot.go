package services

import (
	"context"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/usercache"
	"github.com/Ling0727-ai/go-buct-course-backend/models/AssignmentStatus"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Blacklist"
	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
)

type AssignmentSnapshot struct {
	Tasks                           []*CourseData.CourseData
	Completed, Deleted, Blacklisted map[string]bool
}

var assignmentSnapshots = usercache.New[*AssignmentSnapshot](128, 8<<20)

func GetAssignmentSnapshot(ctx context.Context, userID string) (*AssignmentSnapshot, error) {
	return assignmentSnapshots.Get(ctx, userID, func() (*AssignmentSnapshot, int, time.Duration, error) {
		tasks, err := CourseData.Repository.GetUserSummaries(userID)
		if err != nil {
			return nil, 0, 0, err
		}
		statuses, err := AssignmentStatus.Repository.GetUserStatuses(userID, "", true)
		if err != nil {
			return nil, 0, 0, err
		}
		blocked, err := Blacklist.Repository.GetBlacklistedIDs(userID)
		if err != nil {
			return nil, 0, 0, err
		}
		view := &AssignmentSnapshot{Tasks: tasks, Completed: make(map[string]bool), Deleted: make(map[string]bool), Blacklisted: make(map[string]bool)}
		now, ttl, size := time.Now(), 10*time.Second, 512
		for _, status := range statuses {
			expired := false
			for _, expiry := range []*time.Time{status.ExpiresAt, status.TodoExpiresAt} {
				if expiry != nil {
					left := expiry.Sub(now)
					if left <= 0 {
						expired = true
					} else if left < ttl {
						ttl = left
					}
				}
			}
			if expired {
				continue
			}
			switch status.Status {
			case AssignmentStatus.StatusCompleted:
				view.Completed[status.AssignmentID] = status.Forever != 0
			case AssignmentStatus.StatusDeleted, AssignmentStatus.StatusPermanentDeleted:
				view.Deleted[status.AssignmentID] = true
			}
			size += 128 + len(status.AssignmentID)
		}
		for _, id := range blocked {
			view.Blacklisted[id] = true
			size += 64 + len(id)
		}
		for _, task := range tasks {
			size += 512 + len(task.ID) + len(task.TaskID) + len(task.Subject) + len(task.Title) + len(task.Url) + len(task.Deadline)
		}
		return view, size, ttl, nil
	})
}
