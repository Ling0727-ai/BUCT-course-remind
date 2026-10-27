package handlers

import (
	"fmt"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Todo"
	"github.com/Ling0727-ai/go-buct-course-backend/models/User"
	"github.com/Ling0727-ai/go-buct-course-backend/utils"
	"github.com/gin-gonic/gin"
)

// GetTodos 获取用户待办列表
// 对应 Python GET /api/todos/
func GetTodos(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	includeCompleted := c.Query("include_completed") == "true"
	todos, err := Todo.Repository.GetTodosByUserID(userID, includeCompleted)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Todo.GetFailed})
		return
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "todos": todos, "count": len(todos)})
}

// CreateTodo 创建待办事项
// 对应 Python POST /api/todos/
func CreateTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var body struct {
		Title       string   `json:"title"        binding:"required"`
		Description string   `json:"description"`
		Priority    string   `json:"priority"`
		Hours       *float64 `json:"hours"`
		DueDate     string   `json:"due_date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Todo.TitleEmpty})
		return
	}

	priority := body.Priority
	if priority != "low" && priority != "medium" && priority != "high" {
		priority = "medium"
	}

	now := time.Now()
	var dueDate *time.Time
	var estimatedHours *float64

	switch {
	case body.Hours != nil:
		h := *body.Hours
		if h <= 0 {
			c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Todo.HoursInvalid})
			return
		}
		t := now.Add(time.Duration(h * float64(time.Hour)))
		dueDate = &t
		estimatedHours = body.Hours
	case body.DueDate != "":
		t, err := time.Parse(time.RFC3339, body.DueDate)
		if err != nil {
			c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Todo.DueDateInvalid})
			return
		}
		dueDate = &t
	default:
		h := 24.0
		t := now.Add(24 * time.Hour)
		dueDate = &t
		estimatedHours = &h
	}

	todo := &Todo.Todo{
		UserID:         userID,
		Title:          body.Title,
		Description:    body.Description,
		Priority:       priority,
		DueDate:        dueDate,
		EstimatedHours: estimatedHours,
		Completed:      false,
		IsDeleted:      false,
		Forever:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := Todo.Repository.CreateTodo(todo); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.CreateFailed})
		return
	}
	c.JSON(utils.Defaults.Status.Created, gin.H{"success": true, "message": utils.Defaults.Todo.CreateSuccess, "todo_id": todo.ID})
}

// UpdateTodo 更新待办事项
// 对应 Python PUT /api/todos/:id
func UpdateTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}

	var body map[string]interface{}
	if err = c.ShouldBindJSON(&body); err != nil {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Auth.RequestFormatError})
		return
	}

	if v, ok := body["title"].(string); ok && v != "" {
		existing.Title = v
	}
	if v, ok := body["description"].(string); ok {
		existing.Description = v
	}
	if v, ok := body["priority"].(string); ok {
		if v == "low" || v == "medium" || v == "high" {
			existing.Priority = v
		}
	}
	if hours, ok := body["hours"].(float64); ok && hours > 0 {
		t := time.Now().Add(time.Duration(hours * float64(time.Hour)))
		existing.DueDate = &t
	} else if dueDateStr, ok := body["due_date"].(string); ok && dueDateStr != "" {
		t, err := time.Parse(time.RFC3339, dueDateStr)
		if err == nil {
			existing.DueDate = &t
		}
	}
	existing.UpdatedAt = time.Now()

	if err = Todo.Repository.UpdateTodo(existing); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.UpdateFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.UpdateSuccess})
}

// CompleteTodo 标记待办已完成，直到用户删除或撤销
// 对应 Python POST /api/todos/:id/complete
func CompleteTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}

	if err = Todo.Repository.MarkCompleted(todoID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.UpdateFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.CompleteSuccess + utils.Defaults.Todo.RetentionHint})
}

// UncompleteTodo 撤销待办完成状态
// 对应 Python POST /api/todos/:id/uncomplete
func UncompleteTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}

	if err = Todo.Repository.MarkUncompleted(todoID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.UpdateFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.UncompleteSuccess})
}

// RemindTodo 为待办发送提醒邮件
// 对应 Python POST /api/todos/:id/remind
func RemindTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}
	if existing.Completed {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Todo.CompleteSuccess + utils.Defaults.Todo.AlreadyCompletedHint})
		return
	}
	if existing.IsDeleted {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Todo.AlreadyDeletedHint})
		return
	}

	user, _ := User.Repository.GetUserByID(userID)
	if user == nil || user.Email == "" {
		c.JSON(utils.Defaults.Status.BadRequest, gin.H{"success": false, "error": utils.Defaults.Todo.NoEmailHint})
		return
	}

	priorityText := map[string]string{"high": "高", "medium": "中", "low": "低"}[existing.Priority]
	dueDateStr := ""
	if existing.DueDate != nil {
		dueDateStr = existing.DueDate.Format("2006年01月02日 15:04:05")
	}

	msg := fmt.Sprintf("📝 待办提醒\n\n标题: %s\n优先级: %s", existing.Title, priorityText)
	if existing.Description != "" {
		msg += "\n描述: " + existing.Description
	}
	if dueDateStr != "" {
		msg += "\n截止时间: " + dueDateStr
	}

	if err = config.Mail.SendMail(user.Email, "北化课程提醒 - 待办提醒", msg); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Email.SendFailed})
		return
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.RemindSuccess + existing.Title})
}

// DeleteTodo 软删除待办
// 对应 Python POST /api/todos/:id/delete
func DeleteTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}

	if err = Todo.Repository.DeleteTodo(todoID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.DeleteFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.DeleteSuccess})
}

// RestoreTodo 恢复已删除待办
// 对应 Python POST /api/todos/:id/restore
func RestoreTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}

	if err = Todo.Repository.RestoreTodo(todoID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.RestoreFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.RestoreSuccess})
}

// PermanentDeleteTodo 永久删除待办
// 对应 Python DELETE /api/todos/:id/permanent-delete
func PermanentDeleteTodo(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	todoID := c.Param("id")

	existing, err := Todo.Repository.GetTodoByID(todoID)
	if err != nil || existing == nil || existing.UserID != userID {
		c.JSON(utils.Defaults.Status.NotFound, gin.H{"success": false, "error": utils.Defaults.Todo.NotFound})
		return
	}

	if err = Todo.Repository.PermanentlyDeleteTodo(todoID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.DeleteFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.PermDeleteSuccess})
}

// GetDeletedTodos 获取已删除的待办列表
// 对应 Python GET /api/todos/deleted
func GetDeletedTodos(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	todos, err := Todo.Repository.GetDeletedTodosByUserID(userID)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Todo.GetDeletedFailed})
		return
	}

	// 与 Python 保持一致：在每条记录上附加 todo_id 字段（前端回收站依赖此字段）
	type deletedTodoItem struct {
		*Todo.Todo
		TodoID string `json:"todo_id"`
	}
	items := make([]deletedTodoItem, 0, len(todos))
	for _, t := range todos {
		items = append(items, deletedTodoItem{Todo: t, TodoID: t.ID})
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "deleted_todos": items, "count": len(items)})
}

// ClearDeletedTodos 清空已删除的待办
// 对应 Python DELETE /api/todos/clear-deleted
func ClearDeletedTodos(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	if err := Todo.Repository.ClearDeletedTodos(userID); err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Data.ClearFailed})
		return
	}
	c.JSON(utils.Defaults.Status.OK, gin.H{"success": true, "message": utils.Defaults.Todo.ClearSuccess})
}

// GetTodoStats 获取待办统计
// 对应 Python GET /api/todos/stats
func GetTodoStats(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	todos, err := Todo.Repository.GetTodosByUserID(userID, true)
	if err != nil {
		c.JSON(utils.Defaults.Status.InternalServerError, gin.H{"success": false, "error": utils.Defaults.Todo.GetStatsFailed})
		return
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	total, completed, pending, dueToday := 0, 0, 0, 0
	priorityStats := map[string]int{"low": 0, "medium": 0, "high": 0}

	for _, t := range todos {
		total++
		if t.Completed {
			completed++
		} else {
			pending++
			priorityStats[t.Priority]++
			if t.DueDate != nil && !t.DueDate.Before(todayStart) && t.DueDate.Before(todayEnd) {
				dueToday++
			}
		}
	}

	c.JSON(utils.Defaults.Status.OK, gin.H{
		"success": true,
		"stats": gin.H{
			"total":          total,
			"completed":      completed,
			"pending":        pending,
			"due_today":      dueToday,
			"priority_stats": priorityStats,
		},
	})
}
