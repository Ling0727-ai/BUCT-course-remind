package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	buct "github.com/Ling0727-ai/go-buct-course/client"
	"github.com/Ling0727-ai/go-buct-course/exam"
)

// scrapeTests 获取用户所有待进行测试，对应 Python get_pending_tasks 中 2.2 部分
// blacklistSet: courseId → true，在 lid 层面直接跳过整个课程
func scrapeTests(c *buct.BUCTClient, userID string, blacklistSet map[string]bool) ([]TaskInfo, error) {
	log.Printf("[scraper] 用户 %s 开始获取测试...", userID)

	testLids, err := c.ExamMgr.GetPendingTests()
	if err != nil {
		log.Printf("[scraper] 用户 %s 获取测试课程列表失败: %v", userID, err)
		return nil, err
	}
	if len(testLids) == 0 {
		log.Printf("[scraper] 用户 %s 没有获取到任何测试课程", userID)
		return nil, nil
	}

	now := time.Now()
	var tasks []TaskInfo

	for _, courseInfo := range testLids {
		lid := courseInfo.LID
		courseName := courseInfo.CourseName

		if lid == "" {
			log.Printf("[scraper] 课程 %s 没有 LID，跳过", courseName)
			continue
		}

		// 黑名单过滤
		if blacklistSet[lid] {
			log.Printf("[scraper] 跳过黑名单课程(测试): %s (lid=%s)", courseName, lid)
			continue
		}

		testList, err := c.ExamMgr.GetTestList(lid)
		if err != nil {
			log.Printf("[scraper] 获取课程 %s 测试列表失败: %v", courseName, err)
			return nil, err
		}

		for i, test := range testList.List {
			if !shouldShowTest(test, now) {
				continue
			}

			// 生成测试链接，对应 Python test_url
			testURL := fmt.Sprintf(
				"https://course.buct.edu.cn/meol/common/question/test/student/list.jsp"+
					"?sortColumn=createTime&status=1&tagbug=client"+
					"&sortDirection=-1&strStyle=new03&cateId=%s"+
					"&pagingPage=1&pagingNumberPer=30",
				lid,
			)

			// 生成测试标题，对应 Python test_title 逻辑
			title := test.Title
			if title != "" {
				title = fmt.Sprintf("%s - %s", courseName, title)
			} else {
				title = fmt.Sprintf("%s测试%d", courseName, i+1)
			}

			// 格式化截止时间
			deadline := formatTestDeadline(test.EndTime)

			tasks = append(tasks, TaskInfo{
				Subject:  courseName,
				Title:    title,
				Deadline: deadline,
				Details:  "", // 测试不需要 details
				URL:      testURL,
				Type:     "test",
			})

			log.Printf("[scraper] ✅ 测试: %s - %s (截止: %s)", courseName, title, deadline)
		}
	}

	return tasks, nil
}

// shouldShowTest 判断一个测试是否应该展示给用户
// 对应 Python get_pending_tasks 中对 should_show 的判断逻辑：
//  1. can_start == true → 直接展示
//  2. 在有效时间范围内 && 未完成（无分数且 status != 已完成）→ 展示
func shouldShowTest(test *exam.TestInfo, now time.Time) bool {
	if test.CanStart {
		return true
	}

	// 已有结果（已完成）直接跳过
	if test.HasResult || test.Status == "已完成" {
		return false
	}

	startStr := strings.TrimSuffix(test.StartTime, "Z")
	endStr := strings.TrimSuffix(test.EndTime, "Z")
	if startStr == "" || endStr == "" {
		return false
	}

	// 兼容两种时间格式
	parseTime := func(s string) (time.Time, error) {
		if strings.Contains(s, "T") {
			return time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
		}
		return time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	}

	startTime, err1 := parseTime(startStr)
	endTime, err2 := parseTime(endStr)
	if err1 != nil || err2 != nil {
		log.Printf("[scraper] 解析测试时间失败: start=%s end=%s", startStr, endStr)
		return false
	}

	// 在时间窗口内且未完成（无分数）
	return !now.Before(startTime) && now.Before(endTime)
}
