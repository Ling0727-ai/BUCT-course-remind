package User

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User 用户表结构
type User struct {
	ID                     string     `bson:"_id,omitempty"    json:"id,omitempty"`
	Username               string     `bson:"username"         json:"username"`
	Email                  string     `bson:"email"            json:"email"`
	PasswordHash           string     `bson:"password_hash"    json:"-"`
	StudentID              string     `bson:"student_id"       json:"student_id"`
	SPassword              string     `bson:"s_password"       json:"-"`
	SPasswordLen           *int       `bson:"s_password_len"   json:"-"`
	IsAdmin                bool       `bson:"is_admin"         json:"is_admin"`
	CreatedAt              time.Time  `bson:"created_at"       json:"created_at"`
	UpdatedAt              time.Time  `bson:"updated_at"       json:"updated_at"`
	NextRefreshAt          *time.Time `bson:"next_refresh_at,omitempty" json:"-"`
	RefreshScheduleChanged bool       `bson:"-" json:"-"`
}

// UserRepository 定义用户数据库操作接口
type UserRepository interface {
	CreateUser(user *User) error
	GetUserByID(id string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id string) error
}

type MongoUserRepository struct {
	collection *mongo.Collection
}

var Repository = &MongoUserRepository{}

// UserService 定义用户业务逻辑接口
type UserService interface {
	RegisterUser(username, email, password, studentId, sPassword string) (*User, error)
	LoginUser(username, password string) (*User, error)

	GetUserProfile(id string) (*User, error)

	UpdateUserEmail(id string, email string) (*User, error)
	UpdateUserPassword(id string, newPassword string) (*User, error)
	UpdateStudentID(id string, studentId string) (*User, error)
	UpdateSPassword(id string, sPassword string) (*User, error)

	DeleteUserAccount(id string) error
}

type UserServiceImpl struct {
	repo UserRepository
}

var Service = &UserServiceImpl{}

// Init 在服务启动时绑定 MongoDB collection，必须在接收任何请求前调用
func Init(db *mongo.Database) {
	Repository.collection = db.Collection("users")
	Service.repo = Repository

	// The scheduler only reads users whose next refresh is due. Keeping this
	// index partial avoids indexing accounts that cannot be refreshed.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, field := range []string{"username", "email"} {
		_, err := Repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{{Key: field, Value: 1}},
			Options: options.Index().SetName(field + "_unique").SetUnique(true).
				SetPartialFilterExpression(bson.M{field: bson.M{"$type": "string", "$gt": ""}}),
		})
		if err != nil {
			// Keep legacy duplicates intact; a lookup index still prevents collection scans.
			log.Printf("[user] %s 唯一索引创建失败（需检查旧数据重复值）: %v", field, err)
			if _, lookupErr := Repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys: bson.D{{Key: field, Value: 1}}, Options: options.Index().SetName(field + "_lookup"),
			}); lookupErr != nil {
				log.Printf("[user] %s 查询索引创建失败: %v", field, lookupErr)
			}
		}
	}
	if _, err := Repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "next_refresh_at", Value: 1}, {Key: "_id", Value: 1}},
		Options: options.Index().SetPartialFilterExpression(bson.M{
			"student_id": bson.M{"$type": "string", "$gt": ""},
			"s_password": bson.M{"$type": "string", "$gt": ""},
		}),
	}); err != nil {
		log.Printf("[user] 创建自动刷新索引失败: %v", err)
	}
}
