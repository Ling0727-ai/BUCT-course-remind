package router

import (
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/api/handlers"
	"github.com/Ling0727-ai/go-buct-course-backend/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// 健康检查：前端和 Docker 都用 /api/health，同时保留 /health
	r.GET("/health", handlers.HealthCheck)
	r.GET("/api/health", handlers.HealthCheck)

	// 敏感接口的限流。
	//
	// 注意：这里刻意不做「单 IP 10 次/5 分钟」这类严格限制。
	// 校园网里大量用户共享同一个 NAT 出口 IP，严格按 IP 计数会把
	// 整栋楼的正常用户一起挡在门外，而攻击者换 IP 成本极低。
	// 真正的抗爆破防线是 middleware 里的「账号级失败锁定」，
	// IP 限流只作为兜底，阈值必须留足共享出口的余量。
	authLimit := middleware.RateLimit(120, 5*time.Minute) // 登录/注册/重置
	codeLimit := middleware.RateLimit(60, 5*time.Minute)  // 验证码收发

	api := r.Group("/api")
	{
		// ── 认证，对应 Python /api/auth ──────────────────────────────
		auth := api.Group("/auth")
		{
			auth.POST("/login", authLimit, handlers.Login)
			auth.POST("/logout", handlers.Logout)
			auth.POST("/register", authLimit, handlers.Register)
			auth.GET("/status", handlers.AuthStatus)
			auth.GET("/user-info", handlers.GetUserInfo)
			auth.POST("/update-email", handlers.UpdateEmail)
			auth.POST("/update-student-info", handlers.UpdateStudentInfo)
			auth.POST("/check-email", handlers.CheckEmail)
			auth.POST("/reset-password", authLimit, handlers.ResetPassword)
			auth.POST("/send-verification-code", codeLimit, handlers.SendVerificationCode)
			auth.POST("/verify-code", codeLimit, handlers.VerifyCode)
		}

		// ── 作业，对应 Python /api/assignments ───────────────────────
		assign := api.Group("/assignments")
		{
			// 前端用 /api/assignments/standard 和 /api/assignments/enhanced，都返回作业列表
			assign.GET("/standard", handlers.GetAssignments)
			assign.GET("/enhanced", handlers.GetAssignments)
			assign.GET("/stats", handlers.GetAssignmentStats)
			assign.GET("/completed", handlers.GetCompletedAssignments)
			assign.GET("/deleted", handlers.GetDeletedAssignments)
			assign.GET("/:id/detail", handlers.GetAssignmentDetail)
			assign.DELETE("/clear-deleted", handlers.ClearDeletedAssignments)
			assign.POST("/:id/complete", handlers.MarkAssignmentComplete)
			assign.POST("/:id/uncomplete", handlers.MarkAssignmentUncomplete)
			assign.POST("/:id/delete", handlers.DeleteAssignment)
			assign.POST("/:id/restore", handlers.RestoreAssignment)
			assign.DELETE("/:id/permanent-delete", handlers.PermanentDeleteAssignment)
			assign.POST("/:id/remind", handlers.RemindAssignment)
			// 同步刷新（前端 settings 页）
			assign.POST("/refresh-sync", handlers.RefreshAssignmentsSync)
			// 独立 API：获取未完成作业（不走 JWT，通过账号密码验证）
			assign.POST("/uncompleted", handlers.GetUncompletedAssignments)
		}

		// ── 课程数据，对应 Python /api/course-data ───────────────────
		course := api.Group("/course-data")
		{
			course.GET("/list", handlers.GetCourseDataList)
			course.GET("/status", handlers.GetCourseDataStatus)
			course.POST("/refresh", handlers.RefreshCourseData)
		}

		// ── 待办，对应 Python /api/todos ─────────────────────────────
		todos := api.Group("/todos")
		{
			todos.GET("/", handlers.GetTodos)
			todos.POST("/", handlers.CreateTodo)
			todos.GET("/stats", handlers.GetTodoStats)
			todos.GET("/deleted", handlers.GetDeletedTodos)
			todos.DELETE("/clear-deleted", handlers.ClearDeletedTodos)
			todos.PUT("/:id", handlers.UpdateTodo)
			todos.POST("/:id/complete", handlers.CompleteTodo)
			todos.POST("/:id/uncomplete", handlers.UncompleteTodo)
			todos.POST("/:id/remind", handlers.RemindTodo)
			todos.POST("/:id/delete", handlers.DeleteTodo)
			todos.POST("/:id/restore", handlers.RestoreTodo)
			todos.DELETE("/:id/permanent-delete", handlers.PermanentDeleteTodo)
		}

		// ── 提醒，对应 Python /api/webhooks ──────────────────────────
		reminders := api.Group("/reminders")
		{
			reminders.GET("", handlers.GetReminders)
			reminders.POST("", handlers.CreateReminder)
			reminders.DELETE("/:id", handlers.DeleteReminder)
			reminders.POST("/manual", handlers.ManualReminder)
			reminders.POST("/test", handlers.TestReminder)
		}
		// webhooks 路由别名，和 Python 蓝图前缀一致
		webhooks := api.Group("/webhooks")
		{
			webhooks.POST("/scan_due_soon", handlers.ManualReminder)
			webhooks.POST("/manual", handlers.ManualReminder)
			webhooks.POST("/test", handlers.TestReminder)
		}

		// ── 设置，对应 Python /api/settings ──────────────────────────
		api.GET("/settings", handlers.GetSettings)
		api.POST("/settings", handlers.SaveSettings)
		api.GET("/settings/email", handlers.GetEmailSettings)
		api.POST("/settings/email", handlers.SaveEmailSettings)

		// ── 统计，对应 Python /api/stats ─────────────────────────────
		api.GET("/stats", handlers.GetStats)

		// ── 加密，对应 Python /api/crypto ────────────────────────────
		crypto := api.Group("/crypto")
		{
			crypto.GET("/public-key", handlers.GetPublicKey)
			crypto.GET("/challenge", handlers.GetChallenge)
			crypto.GET("/status", handlers.GetCryptoStatus)
		}

		// ── 黑名单，对应 GET/POST/DELETE /api/blacklist ──────────────
		blacklist := api.Group("/blacklist")
		{
			blacklist.GET("", handlers.GetBlacklist)
			blacklist.POST("", handlers.AddBlacklist)
			blacklist.DELETE("", handlers.ClearBlacklist)
			blacklist.DELETE("/:subject_id", handlers.RemoveBlacklist)
		}

		// ── 测试调试，对应 Python /api/test ──────────────────────────
		// 需要登录：send-test-email 会真实发信，config 曾泄露数据库连接串
		test := api.Group("/test", middleware.AuthRequired())
		{
			test.POST("/send-test-email", handlers.SendTestEmail)
			test.POST("/verify-test-code", handlers.VerifyTestCode)
			test.GET("/config", handlers.GetTestConfig)
		}

		// ── 管理员，对应 Python /api/admin ───────────────────────────
		// AuthRequired 只保证已登录，管理员身份由各 handler 内的
		// adminRequired 校验（会回 403）。
		admin := api.Group("/admin", middleware.AuthRequired())
		{
			admin.POST("/cleanup/trigger", handlers.AdminTriggerCleanup)
			admin.GET("/completed-assignments", handlers.AdminGetCompletedAssignments)
			admin.GET("/system/status", handlers.AdminGetSystemStatus)
		}
	}
}
