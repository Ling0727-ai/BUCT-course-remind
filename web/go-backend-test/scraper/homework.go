package scraper

import (
	"fmt"
	"log"
	"strings"

	buct "github.com/Ling0727-ai/go-buct-course/client"
)

// scrapeHomework 获取用户所有待提交作业，对应 Python get_pending_tasks 中 2.1 部分
// blacklistSet: courseId → true，在 lid 层面直接跳过整个课程
func scrapeHomework(c *buct.BUCTClient, userID string, blacklistSet map[string]bool) ([]TaskInfo, error) {
	log.Printf("[scraper] 用户 %s 开始获取作业...", userID)

	hwDetails, err := c.CourseMgr.GetAllPendingHomeworkDetails()
	if err != nil {
		log.Printf("[scraper] 用户 %s 获取作业数据失败: %v", userID, err)
		return nil, err
	}

	var tasks []TaskInfo

	for _, courseDetail := range hwDetails {
		lid := courseDetail.LID
		courseName := courseDetail.CourseName

		// 黑名单过滤：lid 即为 courseId
		if blacklistSet[lid] {
			log.Printf("[scraper] 跳过黑名单课程(作业): %s (lid=%s)", courseName, lid)
			continue
		}

		for _, hw := range courseDetail.HomeworkList {
			if !hw.CanSubmit {
				continue
			}

			// 格式化截止时间，对应 Python: _format_deadline
			deadline := formatDeadline(hw.Deadline)

			// 生成作业链接，对应 Python 中 homework_url
			hwURL := fmt.Sprintf(
				"https://course.buct.edu.cn/meol/jpk/course/layout/newpage/index.jsp?courseId=%s",
				lid,
			)

			// 存入DetailHref，方便前端直接跳转
			if hw.DetailHref == "" {
				hw.DetailHref = hwURL
				log.Printf("[scraper] 作业 %s - %s 没有 DetailHref，使用课程链接替代: %s", courseName, hw.Title, hwURL)
			}

			// 获取作业详情文本
			var detailsText string
			if hw.HwTID != "" {
				tasksList, err2 := c.CourseMgr.GetHomeworkTasks(hw.DetailHref)
				if err2 != nil {
					log.Printf("[scraper] 获取作业任务列表失败 (href=%s): %v", hw.DetailHref, err2)
					return nil, err2
				} else {
					detailsText = strings.Join(tasksList, "\n")
				}
			}

			tasks = append(tasks, TaskInfo{
				Subject:  courseName,
				Title:    hw.Title,
				Deadline: deadline,
				Details:  detailsText,
				URL:      hwURL,
				Type:     "homework",
			})

			log.Printf("[scraper] ✅ 作业: %s - %s (截止: %s)", courseName, hw.Title, deadline)
		}
	}

	return tasks, nil
}
