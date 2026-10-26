package handlers

import (
	"fmt"
	"regexp"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/deadline"
	"github.com/Ling0727-ai/go-buct-course-backend/models/AssignmentStatus"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Blacklist"
	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
	"github.com/Ling0727-ai/go-buct-course-backend/models/User"
	"github.com/Ling0727-ai/go-buct-course-backend/services"
	"github.com/Ling0727-ai/go-buct-course-backend/utils"
	"github.com/gin-gonic/gin"
)

// reCourseID 从课程 URL 提取 courseId 参数
var reCourseID = regexp.MustCompile(`courseId=(\d+)`)

// extractCourseID 从 URL 字符串中解析 courseId，提取不到则返回空字符串
func extractCourseID(url string) string {
	m := reCourseID.FindStringSubmatch(url)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// GetAssignments 获取用户作业列表（过滤已删除、过滤黑名单）
// 对应 Python GET /api/assignments/standard 和 GET /api/assignments/enhanced
func GetAssignments(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	view, err := services.GetAssignmentSnapshot(c.Request.Context(), userID)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Assignment.GetFailed})
		return
	}

	tasks := view.Tasks
	if c.Query("summary") != "true" {
		tasks, err = CourseData.Repository.GetCourseDataByUserID(userID)
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": "获取数据失败"})
			return
		}
	}
	completedIDs, deletedIDs, blacklistSet := view.Completed, view.Deleted, view.Blacklisted

	var formatted []gin.H
	homeworkCount, testCount := 0, 0
	for _, t := range tasks {
		if deletedIDs[t.TaskID] {
			continue
		}
		// 黑名单过滤：通过 URL 提取 courseId
		if courseID := extractCourseID(t.Url); courseID != "" && blacklistSet[courseID] {
			continue
		}
		entry := gin.H{
			"id":      t.TaskID,
			"subject": t.Subject,
			"type":    t.Type,
			"details": gin.H{
				"task":            t.Title,
				"deadline":        t.Deadline,
				"url":             t.Url,
				"can_submit":      true,
				"is_group":        false,
				"details_content": t.Details,
			},
			"completed":   completedIDs[t.TaskID],
			"has_tasks":   true,
			"tasks_count": 1,
		}
		formatted = append(formatted, entry)
		if t.Type == "homework" {
			homeworkCount++
		} else {
			testCount++
		}
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{
		"success": true,
		"tasks":   formatted,
		"total":   len(formatted),
		"stats": gin.H{
			"homework_count": homeworkCount,
			"tests_count":    testCount,
			"total_count":    len(formatted),
		},
		"source": "database",
	})
}

// GetAssignmentStats 获取作业统计信息
// 对应 Python GET /api/assignments/stats
func GetAssignmentStats(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	view, err := services.GetAssignmentSnapshot(c.Request.Context(), userID)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Assignment.GetStatsFailed})
		return
	}

	tasks, completedIDs, deletedIDs := view.Tasks, view.Completed, view.Deleted

	now := time.Now()
	total, completed, urgent, soon := 0, 0, 0, 0

	for _, t := range tasks {
		if deletedIDs[t.TaskID] {
			continue
		}
		if courseID := extractCourseID(t.Url); courseID != "" && view.Blacklisted[courseID] {
			continue
		}
		total++
		if completedIDs[t.TaskID] {
			completed++
			continue
		}
		if t.Deadline != "" {
			dt := time.Time{}
			var parseErr error
			if t.DeadlineAt != nil {
				dt = *t.DeadlineAt
			} else {
				dt, parseErr = parseDeadlineStr(t.Deadline)
			}
			if parseErr == nil {
				days := int(dt.Sub(now).Hours() / 24)
				if days <= 2 {
					urgent++
				} else if days <= 7 {
					soon++
				}
			}
		}
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{
		"success": true,
		"stats": gin.H{
			"total":     total,
			"completed": completed,
			"urgent":    urgent,
			"soon":      soon,
			"remaining": total - completed,
		},
		"source": "database",
	})
}

// GetAssignmentDetail is scoped to the authenticated user and current visibility state.
func GetAssignmentDetail(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	view, err := services.GetAssignmentSnapshot(c.Request.Context(), userID)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": "获取详情失败"})
		return
	}
	taskID := c.Param("id")
	task, err := CourseData.Repository.GetByTaskID(userID, taskID)
	if err != nil || view.Deleted[taskID] || view.Blacklisted[extractCourseID(task.Url)] {
		c.JSON(404, gin.H{"success": false, "error": "作业不存在"})
		return
	}
	c.JSON(200, gin.H{"success": true, "description": task.Details})
}

// GetCompletedAssignments 获取已完成的作业 ID 列表
// 对应 Python GET /api/assignments/completed
func GetCompletedAssignments(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	ids, err := AssignmentStatus.Repository.GetCompletedIDs(userID)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Assignment.GetCompletedFailed})
		return
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "completed_assignments": ids})
}

// GetDeletedAssignments 获取已删除的作业列表（含完整信息）
// 对应 Python GET /api/assignments/deleted
func GetDeletedAssignments(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	statuses, err := AssignmentStatus.Repository.GetUserStatuses(userID, AssignmentStatus.StatusDeleted, false)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Assignment.GetDeletedFailed})
		return
	}

	tasks, _ := CourseData.Repository.GetCourseDataByUserID(userID)
	taskMap := make(map[string]*CourseData.CourseData, len(tasks))
	for _, t := range tasks {
		taskMap[t.TaskID] = t
	}

	// 以「删除记录」为主遍历，而不是以课表为主。
	//
	// 原实现只遍历 course_data，若某作业已从课表同步中消失
	// （学期结束、科目被拉黑、任务被上游移除），
	// 它的删除记录就会变成永远看不到、也无法恢复的孤儿，
	// 尽管软删数据仍在库里。这里对缺失的课表信息回退到
	// 删除时快照的标题/科目。
	result := make([]gin.H, 0, len(statuses))
	for _, s := range statuses {
		entry := gin.H{
			"task_id":     s.AssignmentID,
			"title":       s.AssignmentTitle,
			"subject":     s.AssignmentSubject,
			"type":        s.AssignmentType,
			"delete_time": s.StatusTime.Format(time.RFC3339),
		}

		// 课表里还在的话，补上更完整的字段
		if t, ok := taskMap[s.AssignmentID]; ok {
			entry["title"] = t.Title
			entry["subject"] = t.Subject
			entry["deadline"] = t.Deadline
			entry["details"] = t.Details
			entry["url"] = t.Url
			entry["type"] = t.Type
		}

		result = append(result, entry)
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "deleted_assignments": result, "count": len(result)})
}

// MarkAssignmentComplete 标记作业已完成
// 对应 Python POST /api/assignments/:id/complete
func MarkAssignmentComplete(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	assignmentID := c.Param("id")

	var body struct {
		Title   string `json:"title"`
		Subject string `json:"subject"`
	}
	c.ShouldBindJSON(&body)

	if err := AssignmentStatus.Repository.MarkCompleted(userID, assignmentID, body.Title, body.Subject); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.UpdateFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Assignment.CompleteSuccess})
}

// MarkAssignmentUncomplete 撤销作业完成状态
// 对应 Python POST /api/assignments/:id/uncomplete
func MarkAssignmentUncomplete(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	assignmentID := c.Param("id")

	if err := AssignmentStatus.Repository.RestoreAssignment(userID, assignmentID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.UpdateFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Assignment.UncompleteSuccess})
}

// DeleteAssignment 软删除作业
// 对应 Python POST /api/assignments/:id/delete
func DeleteAssignment(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	assignmentID := c.Param("id")

	var body struct {
		Title   string `json:"title"`
		Subject string `json:"subject"`
		Type    string `json:"type"`
	}
	c.ShouldBindJSON(&body)

	if err := AssignmentStatus.Repository.MarkDeleted(userID, assignmentID, body.Title, body.Subject, body.Type); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.DeleteFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Assignment.DeleteSuccess})
}

// RestoreAssignment 恢复已删除作业
// 对应 Python POST /api/assignments/:id/restore
func RestoreAssignment(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	assignmentID := c.Param("id")

	if err := AssignmentStatus.Repository.RestoreAssignment(userID, assignmentID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.RestoreFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Assignment.RestoreSuccess})
}

// PermanentDeleteAssignment 永久删除作业
// 对应 Python DELETE /api/assignments/:id/permanent-delete
func PermanentDeleteAssignment(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	assignmentID := c.Param("id")

	if err := AssignmentStatus.Repository.PermanentDelete(userID, assignmentID, "", ""); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.DeleteFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Assignment.PermDeleteSuccess})
}

// ClearDeletedAssignments 清空回收站
// 对应 Python DELETE /api/assignments/clear-deleted
func ClearDeletedAssignments(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	count, err := AssignmentStatus.Repository.ClearUserStatus(userID, AssignmentStatus.StatusDeleted)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.ClearFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Assignment.ClearSuccess, "deleted_count": count})
}

// RemindAssignment 为作业创建定时提醒
// 对应 Python POST /api/assignments/:id/remind
func RemindAssignment(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	assignmentID := c.Param("id")

	// 检查作业是否已完成或已删除
	completed, _ := AssignmentStatus.Repository.IsCompleted(userID, assignmentID)
	if completed {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Assignment.AlreadyCompleted})
		return
	}
	deleted, _ := AssignmentStatus.Repository.IsDeleted(userID, assignmentID)
	if deleted {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Assignment.AlreadyDeleted})
		return
	}

	// 从数据库获取作业详情
	tasks, err := CourseData.Repository.GetCourseDataByUserID(userID)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Assignment.QueryFailed})
		return
	}
	var task *CourseData.CourseData
	for _, t := range tasks {
		if t.TaskID == assignmentID {
			task = t
			break
		}
	}
	if task == nil {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Assignment.NotFound})
		return
	}

	var body struct {
		ReminderConfig map[string]interface{} `json:"reminderConfig"`
	}
	c.ShouldBindJSON(&body)

	// 计算 scheduledTime，对应 Python remind_assignment 中的时间解析逻辑
	now := time.Now()
	scheduledAt := now
	scheduleDesc := "立即"

	cfg := body.ReminderConfig
	if cfg != nil {
		scheduledAt, scheduleDesc = calcScheduledTime(cfg, task.Deadline, now)
	}

	taskType := "作业"
	if task.Type == "test" {
		taskType = "测试"
	}
	msg := buildAssignmentReminderMsg(taskType, task.Subject, task.Title, task.Deadline, scheduleDesc)

	// 立即发送或写入 scheduled_reminders 由 services 层决定
	if !scheduledAt.After(now) {
		if err := services.SendManualReminder(&services.ManualReminderInput{
			UserID:  userID,
			Message: msg,
		}); err != nil {
			c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Email.SendFailed})
			return
		}
		c.JSON(utils.Defaults.Status.OK, gin.H{
			"success": true,
			"message": utils.Defaults.Assignment.RemindSuccess + task.Subject + " - " + task.Title,
		})
		return
	}

	if _, err := services.CreateScheduledReminder(userID, assignmentID, "assignment", msg, scheduledAt); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Assignment.CreateReminderFail})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{
		"success": true,
		"message": "已设置" + taskType + "提醒（" + scheduleDesc + "）：" + task.Subject + " - " + task.Title,
	})
}

// ──────────────────────────────────────────────
//  工具函数
// ──────────────────────────────────────────────

// toStringSet 将字符串切片转为 set，忽略错误
func toStringSet(ids []string, _ error) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// parseDeadlineStr 解析多种格式的截止时间字符串
func parseDeadlineStr(s string) (time.Time, error) {
	return deadline.Parse(s)
}

// calcScheduledTime 根据前端 reminderConfig 计算提醒发送时间
// 对应 Python remind_assignment 中的 rtype 分支逻辑
func calcScheduledTime(cfg map[string]interface{}, deadline string, now time.Time) (time.Time, string) {
	rtype, _ := cfg["type"].(string)
	timing, _ := cfg["timing"].(string)

	var due *time.Time
	if deadline != "" {
		if t, err := parseDeadlineStr(deadline); err == nil {
			due = &t
		}
	}

	getHours := func() float64 {
		switch v := cfg["hours"].(type) {
		case float64:
			return v
		case string:
			presets := map[string]float64{"1h": 1, "3h": 3, "6h": 6, "12h": 12}
			if h, ok := presets[v]; ok {
				return h
			}
		}
		presets := map[string]float64{"1h": 1, "3h": 3, "6h": 6, "12h": 12}
		if h, ok := presets[rtype]; ok {
			return h
		}
		return 1
	}

	switch rtype {
	case "instant", "":
		return now, "立即"
	case "custom-datetime":
		if dtStr, ok := cfg["datetime"].(string); ok {
			for _, f := range []string{"2006-01-02T15:04", "2006-01-02 15:04:05"} {
				if t, err := time.ParseInLocation(f, dtStr, time.Local); err == nil {
					return t, t.Format("2006-01-02 15:04")
				}
			}
		}
		return now, "立即"
	default:
		hours := getHours()
		dur := time.Duration(hours * float64(time.Hour))
		if timing == "after" || rtype == "custom-hours-after" {
			return now.Add(dur), fmtHours(hours) + "后"
		}
		if due != nil {
			return due.Add(-dur), "截止前" + fmtHours(hours)
		}
		return now.Add(dur), fmtHours(hours) + "后"
	}
}

// fmtHours 格式化小时数为可读字符串
func fmtHours(h float64) string {
	if h >= 24 && float64(int(h)) == h {
		d := int(h) / 24
		r := int(h) % 24
		if r > 0 {
			return fmt.Sprintf("%d天%d小时", d, r)
		}
		return fmt.Sprintf("%d天", d)
	}
	if float64(int(h)) == h {
		return fmt.Sprintf("%d小时", int(h))
	}
	return fmt.Sprintf("%.1f小时", h)
}

// buildAssignmentReminderMsg 构建作业提醒消息
func buildAssignmentReminderMsg(taskType, subject, title, deadline, scheduleDesc string) string {
	dl := deadline
	if dl == "" {
		dl = "未知"
	}
	return fmt.Sprintf("⚠️ %s提醒\n\n科目: %s\n标题: %s\n截止时间: %s\n提醒时间: %s",
		taskType, subject, title, dl, scheduleDesc)
}

// RefreshAssignmentsSync 同步刷新作业数据
// 对应 POST /api/assignments/refresh-sync（前端 settings 页调用）
func RefreshAssignmentsSync(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	count, err := services.RefreshUserDataContext(c.Request.Context(), userID)
	if err != nil {
		respondRefreshFailure(c, err)
		return
	}

	c.JSON(200, gin.H{
		"success":    true,
		"message":    fmt.Sprintf("数据刷新完成，共更新 %d 条记录", count),
		"count":      count,
		"updated_at": time.Now().Format(time.RFC3339),
	})
}

// GetUncompletedAssignments 获取所有未完成作业（独立API，不使用JWT session）
// 排除：软删除、永久删除、黑名单科目、已完成
// 通过账号密码验证身份，密码使用scrypt加密对比
func GetUncompletedAssignments(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": "请求格式错误"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(400, gin.H{"success": false, "error": "账号和密码不能为空"})
		return
	}

	user, err := User.Service.LoginUser(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"success": false, "error": "账号或密码错误"})
		return
	}

	userID := user.ID

	tasks, err := CourseData.Repository.GetCourseDataByUserID(userID)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": "获取作业数据失败"})
		return
	}

	completedIDs := toStringSet(AssignmentStatus.Repository.GetCompletedIDs(userID))
	deletedIDs := toStringSet(AssignmentStatus.Repository.GetDeletedIDs(userID))

	blacklistedIDs, _ := Blacklist.Repository.GetBlacklistedIDs(userID)
	blacklistSet := make(map[string]bool, len(blacklistedIDs))
	for _, id := range blacklistedIDs {
		blacklistSet[id] = true
	}

	result := make([]gin.H, 0)
	homeworkCount, testCount := 0, 0

	for _, t := range tasks {
		if deletedIDs[t.TaskID] {
			continue
		}
		if completedIDs[t.TaskID] {
			continue
		}
		if courseID := extractCourseID(t.Url); courseID != "" && blacklistSet[courseID] {
			continue
		}

		result = append(result, gin.H{
			"id":       t.TaskID,
			"subject":  t.Subject,
			"title":    t.Title,
			"deadline": t.Deadline,
			"details":  t.Details,
			"url":      t.Url,
			"type":     t.Type,
		})

		if t.Type == "homework" {
			homeworkCount++
		} else {
			testCount++
		}
	}

	c.JSON(200, gin.H{
		"success":        true,
		"tasks":          result,
		"total":          len(result),
		"homework_count": homeworkCount,
		"test_count":     testCount,
	})
}
