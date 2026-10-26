package CourseData

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

const AutoRefreshInterval = 3 * time.Hour

type CourseData struct {
	ID                  string     `bson:"_id,omitempty" json:"id,omitempty"`
	UserID              string     `bson:"user_id"       json:"user_id"`
	TaskID              string     `bson:"task_id"       json:"task_id"`
	Subject             string     `bson:"subject"       json:"subject"`
	Title               string     `bson:"title"         json:"title"`
	Deadline            string     `bson:"deadline"      json:"deadline"`
	Details             string     `bson:"details"       json:"details"`
	Url                 string     `bson:"url"           json:"url"`
	Type                string     `bson:"type"          json:"type"`
	CreatedAt           time.Time  `bson:"created_at"    json:"created_at"`
	UpdatedAt           time.Time  `bson:"updated_at"    json:"updated_at"`
	DeadlineAt          *time.Time `bson:"deadline_at,omitempty" json:"-"`
	AutoReminderAt      *time.Time `bson:"auto_reminder_at,omitempty" json:"-"`
	AutoReminderPending bool       `bson:"auto_reminder_pending" json:"-"`
	DeadlineNormalized  bool       `bson:"deadline_normalized" json:"-"`
	AutoReminderVersion int        `bson:"auto_reminder_version" json:"-"`
}

// TaskInput 用于批量保存时的输入格式
type TaskInput struct {
	Subject  string `json:"subject"`
	Title    string `json:"title"`
	Deadline string `json:"deadline"`
	Details  string `json:"details"`
	Url      string `json:"url"`
	Type     string `json:"type"` // "homework" | "test" | "todo"
}

type CourseDataRepository interface {
	CreateCourseData(courseData *CourseData) error

	GetCourseDataByID(id string) (*CourseData, error)
	GetCourseDataByUserID(userID string) ([]*CourseData, error)

	UpdateCourseData(courseData *CourseData) error
	DeleteCourseData(id string) error

	GetLastUpdateTimeByUserID(userID string) (time.Time, error)

	ClearCourseDataByUserID(userID string) error

	// SaveUserCourseData 批量保存用户课程数据（先删旧数据再插入新数据）
	SaveUserCourseData(userID string, tasks []TaskInput) (int, error)
}

type MongoCourseDataRepository struct {
	collection *mongo.Collection
}

var Repository = &MongoCourseDataRepository{}

type CourseDataService interface {
}

// Init 在服务启动时绑定 MongoDB collection，同时建立索引
func Init(db *mongo.Database) {
	col := db.Collection("course_data")
	Repository = NewMongoCourseDataRepository(col)
}
