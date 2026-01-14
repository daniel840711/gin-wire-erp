package repository

import (
	"context"
	"fmt"
	"time"

	"interchange/internal/core"
	client "interchange/internal/database/client"
	"interchange/internal/database/mongodb/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(mongoClient *client.MongoClient) *UserRepository {
	repository := &UserRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionUsers)),
	}
	// 建議：啟動時建立常用索引（冪等、存在即跳過）
	_ = repository.ensureIndexes(context.Background())
	return repository
}

// 依實際查詢習慣調整索引；這裡先提供最通用的兩個
func (repository *UserRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	indexModels := []mongo.IndexModel{
		{ // 依建立時間倒序查列表
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_createdAt_desc"),
		},
		{ // 依使用者狀態查詢
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	}
	_, _ = repository.collection.Indexes().CreateMany(ctx, indexModels)
	return nil
}

// Create：單文件插入
func (repository *UserRepository) Create(
	contextValue context.Context,
	user *model.User,
) (_ *model.User, returnedError error) {

	nowUTC := time.Now().UTC()
	// 若上游未指定 _id，可自己先產生；InsertOne 會沿用
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	user.CreatedAt = nowUTC
	user.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, user)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	user.ID = objectID
	return user, nil
}

// GetByID：單文件讀取
func (repository *UserRepository) GetByID(
	contextValue context.Context,
	userIdentifier primitive.ObjectID,
) (_ *model.User, returnedError error) {

	var user model.User
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": userIdentifier}).Decode(&user); returnedError != nil {
		return nil, returnedError
	}
	return &user, nil
}

// UpdateStatus：單文件部分更新
func (repository *UserRepository) UpdateStatus(
	contextValue context.Context,
	userIdentifier primitive.ObjectID,
	status core.Status,
) (_ int64, returnedError error) {

	update := bson.M{"$set": bson.M{"status": status}}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": userIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	return result.MatchedCount, nil
}

// UpdateRole：單文件部分更新
func (repository *UserRepository) UpdateRole(
	contextValue context.Context,
	userIdentifier primitive.ObjectID,
	role core.Role,
) (_ int64, returnedError error) {

	update := bson.M{"$set": bson.M{"role": role}}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": userIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	return result.MatchedCount, nil
}

// UpdateLastSeen：單文件部分更新
func (repository *UserRepository) UpdateLastSeen(
	contextValue context.Context,
	userIdentifier primitive.ObjectID,
	lastSeenTime time.Time,
) (_ int64, returnedError error) {

	update := bson.M{"$set": bson.M{"lastSeen": lastSeenTime.UTC()}}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": userIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	return result.MatchedCount, nil
}

// UpdateByID：將呼叫端給的欄位寫入 $set（請確認呼叫端只傳「欄位值」，不要傳 $inc 之類 operator）
func (repository *UserRepository) UpdateByID(
	contextValue context.Context,
	userIdentifier primitive.ObjectID,
	setFields bson.M,
) (_ int64, returnedError error) {

	update := bson.M{"$set": setFields}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": userIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	return result.MatchedCount, nil
}

// DeleteByID：單文件刪除
func (repository *UserRepository) DeleteByID(
	contextValue context.Context,
	userIdentifier primitive.ObjectID,
) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": userIdentifier})
	return returnedError
}

// List：分頁查詢（注意：這裡預設 page 為「0 起算」）
func (repository *UserRepository) List(
	contextValue context.Context,
	listOptions core.ListOptions,
) (_ []*model.User, returnedError error) {

	findOptions := options.Find().
		SetSkip(int64(listOptions.Page) * int64(listOptions.Size)).
		SetLimit(int64(listOptions.Size)).
		// 修正：你的欄位是 createdAt（小駝峰），不是 created_at
		SetSort(bson.M{"createdAt": -1})

	cursor, findError := repository.collection.Find(contextValue, listOptions.Filter, findOptions)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var users []*model.User
	if returnedError = cursor.All(contextValue, &users); returnedError != nil {
		return nil, returnedError
	}
	return users, nil
}

// ListAll：全量列舉（小量資料可用；大量資料請改用批次或游標迭代）
func (repository *UserRepository) ListAll(
	contextValue context.Context,
) (_ []*model.User, returnedError error) {

	cursor, findError := repository.collection.Find(contextValue, bson.M{})
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var users []*model.User
	for cursor.Next(contextValue) {
		var user model.User
		if decodeError := cursor.Decode(&user); decodeError != nil {
			return nil, decodeError
		}
		users = append(users, &user)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}
	return users, nil
}
