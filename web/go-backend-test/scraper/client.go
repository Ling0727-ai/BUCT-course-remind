package scraper

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/crypto"
	"github.com/Ling0727-ai/go-buct-course-backend/models/User"
	buct "github.com/Ling0727-ai/go-buct-course/client"
)

const (
	clientCacheExpiry = 30 * 60 // 30 分钟（秒），对应 Python CLIENT_CACHE_EXPIRY
	cleanupInterval   = 5 * 60  // 每 5 分钟清理一次
)

// ──────────────────────────────────────────────
//  客户端缓存管理
// ──────────────────────────────────────────────

// cleanupExpiredClients 清理过期的客户端缓存，防止内存泄漏
// 对应 Python: _cleanup_expired_clients
func (s *BUCTScraper) cleanupExpiredClients() {
	now := time.Now().Unix()
	s.mu.Lock()
	if now-s.lastClean < cleanupInterval {
		s.mu.Unlock()
		return
	}
	s.lastClean = now

	expired := make([]*buct.BUCTClient, 0)
	for userID, ts := range s.timestamps {
		if now-ts > clientCacheExpiry && s.userLocks[userID] == nil {
			if c, ok := s.clients[userID]; ok {
				expired = append(expired, c)
				delete(s.clients, userID)
			}
			delete(s.timestamps, userID)
			log.Printf("[scraper] 清理过期客户端: 用户 %s", userID)
		}
	}
	s.mu.Unlock()
	for _, c := range expired {
		c.Logout()
	}
}

// getClient 获取或创建用户的 BUCTClient，对应 Python: _get_client
func (s *BUCTScraper) getClient(userID string) *buct.BUCTClient {
	s.cleanupExpiredClients()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[userID]; !ok {
		s.clients[userID] = buct.New("", "")
		log.Printf("[scraper] 为用户 %s 创建新 BUCTClient", userID)
	}
	s.timestamps[userID] = time.Now().Unix()
	return s.clients[userID]
}

// Logout 登出并清理指定用户的客户端，对应 Python: logout
func (s *BUCTScraper) Logout(userID string) bool {
	unlock := s.lockUser(userID)
	defer unlock()
	return s.logoutClient(userID)
}

func (s *BUCTScraper) logoutClient(userID string) bool {
	s.mu.Lock()

	c, ok := s.clients[userID]
	if !ok {
		s.mu.Unlock()
		log.Printf("[scraper] 尝试登出未找到客户端的用户: %s", userID)
		return false
	}

	delete(s.clients, userID)
	delete(s.timestamps, userID)
	s.mu.Unlock()
	if c != nil {
		c.Logout()
	}
	log.Printf("[scraper] 用户 %s 客户端已清理", userID)
	return true
}

// ──────────────────────────────────────────────
//  凭证获取
// ──────────────────────────────────────────────

// getUserCredentials 从数据库获取并解密用户的学号和外部密码
// 对应 Python: _get_user_credentials
func (s *BUCTScraper) getUserCredentials(userID string) (studentID, sPassword string, err error) {
	repo := User.Repository
	user, err := repo.GetUserByID(userID)
	if err != nil || user == nil {
		return "", "", fmt.Errorf("未找到用户: %s", userID)
	}

	if user.StudentID == "" || user.SPassword == "" {
		return "", "", fmt.Errorf("用户 %s 未设置学号或密码", userID)
	}

	plainPwd, err := crypto.Crypto.DecryptECC(user.SPassword)
	if err != nil {
		return "", "", fmt.Errorf("用户 %s 密码解密失败: %w", userID, err)
	}

	return user.StudentID, plainPwd, nil
}

// ──────────────────────────────────────────────
//  登录
// ──────────────────────────────────────────────

// autoLogin 用数据库凭证为用户登录，对应 Python: auto_login
func (s *BUCTScraper) autoLogin(userID string) error {
	c := s.getClient(userID)

	studentID, sPassword, err := s.getUserCredentials(userID)
	if err != nil {
		return err
	}

	log.Printf("[scraper] 为用户 %s (学号: %s) 执行登录...", userID, studentID)

	// 先清理旧会话
	c.Logout()

	if err = c.Login(studentID, sPassword); err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	log.Printf("[scraper] 用户 %s 登录成功", userID)
	return nil
}

// ──────────────────────────────────────────────
//  时间格式化工具
// ──────────────────────────────────────────────

// formatDeadline 将中文日期格式转为 "2025-09-23 23:59:00"
// 支持: "2025年9月23日 23:59:00" 和 "2025年09月23日 23:59:00"
// 对应 Python: _format_deadline
func formatDeadline(s string) string {
	if s == "" {
		return ""
	}
	// 先尝试双位月日（09月03日）
	if t, err := time.ParseInLocation("2006年01月02日 15:04:05", s, time.Local); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	// 再尝试单位月日（9月3日）- Go 的格式化数字参考月=1，日=2
	if t, err := time.ParseInLocation("2006年1月2日 15:04:05", s, time.Local); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	// 混合：双位月单位日 / 单位月双位日
	for _, layout := range []string{
		"2006年01月2日 15:04:05",
		"2006年1月02日 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	log.Printf("[scraper] 作业时间解析失败，返回原始值: %q", s)
	return s
}

// formatTestDeadline 兼容多种格式，对应 Python: _format_test_deadline
func formatTestDeadline(s string) string {
	if s == "" {
		return ""
	}
	cleaned := strings.TrimSuffix(s, "Z")
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006年01月02日 15:04:05",
		"2006年1月2日 15:04:05",
		"2006年01月2日 15:04:05",
		"2006年1月02日 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.ParseInLocation(layout, cleaned, time.Local); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	log.Printf("[scraper] 测试时间格式无法解析: %q", s)
	return s
}

// ──────────────────────────────────────────────
//  主流程：GetPendingTasks
// ──────────────────────────────────────────────

// GetPendingTasks 完整执行 login->check->logout 流程，返回任务列表
// 对应 Python: get_pending_tasks
// blacklistedIDs 中的 courseId 将在抓取时直接跳过
func (s *BUCTScraper) GetPendingTasks(userID string, blacklistedIDs []string) (*ScrapeResult, error) {
	unlock := s.lockUser(userID)
	defer unlock()
	defer s.logoutClient(userID)
	log.Printf("[scraper] 用户 %s 开始 GetPendingTasks 流程...", userID)

	// 1. LOGIN
	if err := s.autoLogin(userID); err != nil {
		return nil, err
	}

	c := s.getClient(userID)

	// 确保 CourseMgr 和 ExamMgr 已初始化
	if c.CourseMgr == nil || c.ExamMgr == nil {
		return nil, fmt.Errorf("客户端未完全初始化，缺少 CourseMgr 或 ExamMgr")
	}

	// 构建黑名单 set，O(1) 查询
	blacklistSet := make(map[string]bool, len(blacklistedIDs))
	for _, id := range blacklistedIDs {
		blacklistSet[id] = true
	}

	// 2. CHECK - 获取作业和测试（传入黑名单 set）
	hwTasks, hwErr := scrapeHomework(c, userID, blacklistSet)
	if hwErr != nil {
		log.Printf("[scraper] 用户 %s 获取作业异常: %v", userID, hwErr)
	}

	testTasks, testErr := scrapeTests(c, userID, blacklistSet)
	if testErr != nil {
		log.Printf("[scraper] 用户 %s 获取测试异常: %v", userID, testErr)
	}
	if err := errors.Join(hwErr, testErr); err != nil {
		return nil, err
	}

	// 3. 汇总结果
	allTasks := make([]TaskInfo, 0, len(hwTasks)+len(testTasks))
	allTasks = append(allTasks, hwTasks...)
	allTasks = append(allTasks, testTasks...)

	result := &ScrapeResult{Tasks: allTasks}
	result.Stats.HomeworkCount = len(hwTasks)
	result.Stats.TestsCount = len(testTasks)
	result.Stats.TotalCount = len(allTasks)

	log.Printf("[scraper] 用户 %s 完成，作业 %d 条，测试 %d 条",
		userID, result.Stats.HomeworkCount, result.Stats.TestsCount)

	return result, nil
}
