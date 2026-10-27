package app

import (
	"log"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/config"
	"github.com/Ling0727-ai/go-buct-course-backend/crypto"
	"github.com/Ling0727-ai/go-buct-course-backend/middleware"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"github.com/Ling0727-ai/go-buct-course-backend/models/AssignmentStatus"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Blacklist"
	"github.com/Ling0727-ai/go-buct-course-backend/models/CourseData"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Reminder"
	"github.com/Ling0727-ai/go-buct-course-backend/models/Todo"
	"github.com/Ling0727-ai/go-buct-course-backend/models/User"
	"github.com/Ling0727-ai/go-buct-course-backend/router"
	"github.com/Ling0727-ai/go-buct-course-backend/scheduler"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config) *gin.Engine {
	config.LoadMailConfig()

	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// 默认信任所有代理会导致 c.ClientIP() 可被 X-Forwarded-For 伪造，
	// 限流形同虚设。这里只信任 Docker 网络与回环地址。
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "172.16.0.0/12", "10.0.0.0/8", "192.168.0.0/16"})
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.SessionMiddleware())

	// 全局兜底限流：单 IP 每分钟 300 次，防止接口被扫
	r.Use(middleware.RateLimit(300, time.Minute))

	router.SetupRoutes(r)
	return r
}

func Start(cfg *config.Config) {
	// 1. 初始化 Snowflake ID 生成器
	sfGen, err := config.NewSnowflakeGenerator()
	if err != nil {
		log.Fatalf("[snowflake] 初始化失败: %v", err)
	}
	config.Snowflake = sfGen
	log.Println("[snowflake] ID 生成器已就绪")

	// 2. 预热 MongoDB 连接池
	client, err := models.ConnectToDB()
	if err != nil {
		log.Fatalf("[db] 无法连接 MongoDB: %v", err)
	}
	defer models.DisconnectDB()

	// 3. 初始化所有 model 包（绑定 collection，避免 nil pointer panic）
	db := client.Database("buct-course")
	User.Init(db)
	CourseData.Init(db)
	if err := AssignmentStatus.Init(db); err != nil {
		log.Fatalf("[app] 迁移已完成作业状态失败: %v", err)
	}
	Blacklist.Init(db)
	if err := Todo.Init(db); err != nil {
		log.Fatalf("[app] 迁移已完成待办失败: %v", err)
	}
	Reminder.Init(db)
	log.Println("[app] 所有 Repository 已初始化")

	// 4. 预生成 RSA 密钥对（服务启动时完成，避免第一次请求时阻塞）
	rsaSvc := crypto.GetRSAService()
	if rsaSvc.IsEnabled() {
		log.Println("[rsa] 正在预生成 RSA 密钥对...")
		if err := rsaSvc.GenerateKeyPair(); err != nil {
			log.Printf("[rsa] 密钥生成失败（RSA 功能将不可用）: %v", err)
		} else {
			log.Println("[rsa] RSA 密钥对已就绪")
		}
	}

	// 5. 启动定时调度器
	s := scheduler.GetScheduler()
	s.Start()
	defer s.Stop()

	// 6. 启动 HTTP 服务
	r := NewRouter(cfg)
	log.Printf("服务器启动在 http://localhost%s (环境: %s)\n", cfg.Port, cfg.Env)
	if err := r.Run(cfg.Port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
