package User

import (
	"context"
	"errors"
	"time"

	"github.com/Ling0727-ai/go-buct-course-backend/internal/scheduling"
	"github.com/Ling0727-ai/go-buct-course-backend/internal/usercache"
	"github.com/Ling0727-ai/go-buct-course-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// decodeUser 从 mongo.SingleResult 解码用户，兼容 ObjectID 和 string 两种 _id 类型
func decodeUser(sr *mongo.SingleResult) (*User, error) {
	var raw bson.M
	if err := sr.Decode(&raw); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	user := &User{}
	switch v := raw["_id"].(type) {
	case primitive.ObjectID:
		user.ID = v.Hex()
	case string:
		user.ID = v
	}
	if v, ok := raw["username"].(string); ok {
		user.Username = v
	}
	if v, ok := raw["email"].(string); ok {
		user.Email = v
	}
	if v, ok := raw["student_id"].(string); ok {
		user.StudentID = v
	}
	if v, ok := raw["password_hash"].(string); ok {
		user.PasswordHash = v
	}
	if v, ok := raw["s_password"].(string); ok {
		user.SPassword = v
	}
	if v, ok := raw["is_admin"].(bool); ok {
		user.IsAdmin = v
	}
	if v, ok := raw["created_at"].(primitive.DateTime); ok {
		user.CreatedAt = v.Time()
	}
	if v, ok := raw["updated_at"].(primitive.DateTime); ok {
		user.UpdatedAt = v.Time()
	}
	if v, ok := raw["next_refresh_at"].(primitive.DateTime); ok {
		nextRefresh := v.Time()
		user.NextRefreshAt = &nextRefresh
	}
	return user, nil
}

// idFilter 根据 ID 字符串返回正确的 bson filter，兼容 ObjectID 和 string
func idFilter(id string) bson.M {
	if objID, err := primitive.ObjectIDFromHex(id); err == nil {
		return bson.M{"_id": objID}
	}
	return bson.M{"_id": id}
}

// NewMongoUserRepository 创建一个新的 Repository 实例
func NewMongoUserRepository() (UserRepository, error) {
	client, err := models.ConnectToDB()
	if err != nil {
		return nil, err
	}
	db := client.Database("buct-course")
	return &MongoUserRepository{collection: db.Collection("users")}, nil
}

// CreateUser 创建用户
func (r *MongoUserRepository) CreateUser(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, user)
	if mongo.IsDuplicateKeyError(err) {
		return errors.New("username or email already exists")
	}
	return err
}

// GetUserByID 根据 ID 查询用户（兼容 ObjectID 和 Snowflake string）
func (r *MongoUserRepository) GetUserByID(id string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 先尝试 ObjectID（Python 老数据）
	if objID, err := primitive.ObjectIDFromHex(id); err == nil {
		user, err2 := decodeUser(r.collection.FindOne(ctx, bson.M{"_id": objID}))
		if err2 != nil {
			return nil, err2
		}
		if user != nil {
			return user, nil
		}
	}
	// 再按字符串匹配（Go Snowflake ID）
	return decodeUser(r.collection.FindOne(ctx, bson.M{"_id": id}))
}

// GetUserByUsername 根据用户名查询用户
func (r *MongoUserRepository) GetUserByUsername(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return decodeUser(r.collection.FindOne(ctx, bson.M{"username": username}))
}

// GetUserByEmail 根据邮箱查询用户
func (r *MongoUserRepository) GetUserByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return decodeUser(r.collection.FindOne(ctx, bson.M{"email": email}))
}

// UpdateUser 更新用户信息
func (r *MongoUserRepository) UpdateUser(user *User) error {
	defer usercache.Invalidate(user.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"email":         user.Email,
			"student_id":    user.StudentID,
			"password_hash": user.PasswordHash,
			"s_password":    user.SPassword,
			"is_admin":      user.IsAdmin,
			"updated_at":    user.UpdatedAt,
		},
	}
	if user.RefreshScheduleChanged && user.NextRefreshAt != nil {
		update["$set"].(bson.M)["next_refresh_at"] = *user.NextRefreshAt
	}
	_, err := r.collection.UpdateOne(ctx, idFilter(user.ID), update)
	if err == nil && user.RefreshScheduleChanged {
		scheduling.NotifyRefresh()
	}
	if mongo.IsDuplicateKeyError(err) {
		return errors.New("username or email already exists")
	}
	return err
}

// DeleteUser 删除用户
func (r *MongoUserRepository) DeleteUser(id string) error {
	defer usercache.Invalidate(id)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.DeleteOne(ctx, idFilter(id))
	return err
}

func (r *MongoUserRepository) ScheduleRefresh(id string, at time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.UpdateOne(ctx, idFilter(id), bson.M{"$set": bson.M{"next_refresh_at": at}, "$unset": bson.M{"refresh_lease_until": ""}})
	return err
}
