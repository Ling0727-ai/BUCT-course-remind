package scraper

import (
	"sync"

	"github.com/Ling0727-ai/go-buct-course/client"
)

// TaskInfo 统一的任务信息结构，对应 Python 中的 task_info 字典
type TaskInfo struct {
	Subject  string `json:"subject"`
	Title    string `json:"title"`
	Deadline string `json:"deadline"`
	Details  string `json:"details"` // 仅 homework 有值，test 为空
	URL      string `json:"url"`
	Type     string `json:"type"` // "homework" | "test"
}

// ScrapeResult 对应 Python get_pending_tasks 的返回值
type ScrapeResult struct {
	Tasks []TaskInfo `json:"tasks"`
	Stats struct {
		HomeworkCount int `json:"homework_count"`
		TestsCount    int `json:"tests_count"`
		TotalCount    int `json:"total_count"`
	} `json:"stats"`
}

// Scraper 爬虫接口
type Scraper interface {
	// GetPendingTasks 完整执行 login->check->logout 流程，返回任务列表
	// blacklistedIDs 为用户黑名单中的 subjectID（courseId），scraper 内部直接跳过这些课程
	GetPendingTasks(userID string, blacklistedIDs []string) (*ScrapeResult, error)
	// Logout 登出并清理指定用户的客户端缓存
	Logout(userID string) bool
}

// BUCTScraper 爬虫实现，对应 Python BUCTScraperEnhanced
type BUCTScraper struct {
	clients    map[string]*client.BUCTClient
	timestamps map[string]int64 // Unix 秒，记录每个客户端最后使用时间
	mu         sync.Mutex
	lastClean  int64 // 上次清理时间（Unix 秒）
	userLocks  map[string]*userLock
}

// 全局单例
var (
	instance *BUCTScraper
	once     sync.Once
)

// GetScraper 获取全局单例，对应 Python get_scraper()
func GetScraper() *BUCTScraper {
	once.Do(func() {
		instance = &BUCTScraper{
			clients:    make(map[string]*client.BUCTClient),
			timestamps: make(map[string]int64),
		}
	})
	return instance
}
