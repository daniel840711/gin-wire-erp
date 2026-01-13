package repository

import (
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson"
)

// 統一管理所有 MySQL repository
type MongoDBRepository struct {
	userRepo             *UserRepository
	userAPIKeyRepository *UserAPIKeyRepository
}

// 建立 MySQL repository 物件
func NewMongoDBRepository(
	userRepo *UserRepository,
	userAPIKeyRepository *UserAPIKeyRepository,
) *MongoDBRepository {
	return &MongoDBRepository{
		userRepo:             userRepo,
		userAPIKeyRepository: userAPIKeyRepository,
	}
}

// Wire 依賴提供
var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewUserAPIKeyRepository,
	NewMongoDBRepository)

func withUpdatedAt(update bson.M) bson.M {
	// 確保 $currentDate 存在
	currentDate, ok := update["$currentDate"].(bson.M)
	if !ok || currentDate == nil {
		currentDate = bson.M{}
	}
	currentDate["updatedAt"] = true
	update["$currentDate"] = currentDate
	return update
}
