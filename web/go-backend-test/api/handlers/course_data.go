package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
	"github.com/Ling0727-ai/go-buct-course-backend/models/User"
	"github.com/Ling0727-ai/go-buct-course-backend/services"
	"github.com/gin-gonic/gin"
)

// RefreshCourseData 手动刷新课程数据（同步），对应 Python POST /api/course-data/refresh
func RefreshCourseData(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	count, err := services.RefreshUserDataContext(c.Request.Context(), userID)
	if err != nil {
		respondRefreshFailure(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    fmt.Sprintf("数据刷新成功，共更新 %d 条记录", count),
		"count":      count,
		"updated_at": time.Now().Format(time.RFC3339),
	})
}

func respondRefreshFailure(c *gin.Context, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	status := http.StatusInternalServerError
	if errors.Is(err, services.ErrRefreshQueueFull) {
		status = http.StatusServiceUnavailable
		c.Header("Retry-After", "5")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
	}
	c.JSON(status, gin.H{"success": false, "error": "刷新数据失败: " + err.Error()})
}

// GetCourseDataList 从数据库获取课程数据列表
func GetCourseDataList(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	tasks, err := CourseData.Repository.GetCourseDataByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取数据失败"})
		return
	}

	lastUpdate, _ := CourseData.Repository.GetLastUpdateTimeByUserID(userID)

	var lastUpdateStr interface{} = nil
	if !lastUpdate.IsZero() {
		lastUpdateStr = lastUpdate.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"tasks":       tasks,
		"total":       len(tasks),
		"last_update": lastUpdateStr,
		"source":      "database",
	})
}

// GetCourseDataStatus 获取数据状态（最后更新时间、下次刷新时间）
func GetCourseDataStatus(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	lastUpdate, _ := CourseData.Repository.GetLastUpdateTimeByUserID(userID)
	hasData := !lastUpdate.IsZero()

	var lastUpdateStr, nextRefreshStr interface{} = nil, nil
	var hoursUntilRefresh interface{} = nil

	if hasData {
		lastUpdateStr = lastUpdate.Format(time.RFC3339)
		nextRefresh := lastUpdate.Add(CourseData.AutoRefreshInterval)
		if user, err := User.Repository.GetUserByID(userID); err == nil && user != nil && user.NextRefreshAt != nil {
			nextRefresh = *user.NextRefreshAt
		}
		nextRefreshStr = nextRefresh.Format(time.RFC3339)
		h := time.Until(nextRefresh).Hours()
		if h < 0 {
			h = 0
		}
		hoursUntilRefresh = h
	}

	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"has_data":            hasData,
		"last_update":         lastUpdateStr,
		"next_auto_refresh":   nextRefreshStr,
		"hours_until_refresh": hoursUntilRefresh,
	})
}
