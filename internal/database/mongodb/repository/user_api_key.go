package repository

import (
	"context"
	"errors"
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

type UserAPIKeyRepository struct {
	collection *mongo.Collection
}

func NewUserAPIKeyRepository(mongoClient *client.MongoClient) *UserAPIKeyRepository {
	repository := &UserAPIKeyRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionUserAPIKeys)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

// 建索引：
// 1) userID+keyName 唯一（避免同用戶同名重覆建立）
// 2) 常用查詢加速：userID、createdAt、providerAccess.provider
func (repository *UserAPIKeyRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	models := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "userID", Value: 1},
				{Key: "keyName", Value: 1},
			},
			Options: options.Index().SetName("uniq_userID_keyName").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "userID", Value: 1}},
			Options: options.Index().SetName("idx_userID"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_createdAt_desc"),
		},
		{
			Keys:    bson.D{{Key: "providerAccess.provider", Value: 1}},
			Options: options.Index().SetName("idx_providerAccess_provider"),
		},
	}

	_, returnedError := repository.collection.Indexes().CreateMany(ctx, models)
	// 索引已存在時 Mongo 會回覆 error；在此可忽略重覆錯誤
	if returnedError != nil && !errors.Is(returnedError, mongo.ErrNilDocument) {
		// 許多驅動在索引存在時會回 DuplicateKey 或類似錯；不視為致命
		return nil
	}
	return nil
}

// Create 新增一筆 API Key
func (repository *UserAPIKeyRepository) Create(contextValue context.Context, apiKey *model.UserAPIKey) (_ *model.UserAPIKey, returnedError error) {
	nowUTC := time.Now().UTC()
	apiKey.CreatedAt = nowUTC
	apiKey.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, apiKey)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	apiKey.ID = objectID
	return apiKey, nil
}

// List 依條件查詢 API Key
func (repository *UserAPIKeyRepository) List(contextValue context.Context, filter bson.M) (_ []*model.UserAPIKey, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.UserAPIKey
	for cursor.Next(contextValue) {
		var apiKey model.UserAPIKey
		if decodeError := cursor.Decode(&apiKey); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &apiKey)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

// GetByID 依 ID 取得單一 API Key
func (repository *UserAPIKeyRepository) GetByID(contextValue context.Context, apiKeyIdentifier primitive.ObjectID) (_ *model.UserAPIKey, returnedError error) {
	var apiKey model.UserAPIKey
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": apiKeyIdentifier}).Decode(&apiKey); returnedError != nil {
		return nil, returnedError
	}
	return &apiKey, nil
}

// ListByUserID 取得使用者底下的 API Key
func (repository *UserAPIKeyRepository) ListByUserID(contextValue context.Context, userIdentifier primitive.ObjectID) (_ []*model.UserAPIKey, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, bson.M{"userID": userIdentifier})
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.UserAPIKey
	for cursor.Next(contextValue) {
		var apiKey model.UserAPIKey
		if decodeError := cursor.Decode(&apiKey); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &apiKey)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}
	return results, nil
}

// DeleteByID 依 ID 刪除
func (repository *UserAPIKeyRepository) DeleteByID(contextValue context.Context, apiKeyIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": apiKeyIdentifier})
	return returnedError
}

// DeleteAllByUserID 刪除使用者底下所有 API Key
func (repository *UserAPIKeyRepository) DeleteAllByUserID(contextValue context.Context, userIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteMany(contextValue, bson.M{"userID": userIdentifier})
	return returnedError
}

// UpdateKeyName 更新 KeyName
func (repository *UserAPIKeyRepository) UpdateKeyName(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, keyName string) (returnedError error) {
	update := bson.M{"$set": bson.M{"keyName": keyName}}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": apiKeyIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateKeyValue 更新 KeyValue
func (repository *UserAPIKeyRepository) UpdateKeyValue(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, keyValue string) (returnedError error) {
	update := bson.M{"$set": bson.M{"keyValue": keyValue}}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": apiKeyIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateProviderAccessAll 全量覆蓋 providerAccess
func (repository *UserAPIKeyRepository) UpdateProviderAccessAll(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, accessList []model.ProviderAccess) (returnedError error) {
	update := bson.M{"$set": bson.M{"providerAccess": accessList}}
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": apiKeyIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateProviderStatus 更新單一 provider 的狀態
func (repository *UserAPIKeyRepository) UpdateProviderStatus(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, providerName core.ProviderName, status string) (returnedError error) {
	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}
	update := bson.M{
		"$set": bson.M{
			"providerAccess.$[target].status": status,
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateProviderLimitCount 更新限額（arrayFilters 版）
func (repository *UserAPIKeyRepository) UpdateProviderLimitCount(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, providerName core.ProviderName, limitCount int) (returnedError error) {
	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}
	update := bson.M{
		"$set": bson.M{
			"providerAccess.$[target].limitCount": limitCount,
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateProviderUsedCount 覆寫已用次數（arrayFilters 版）
func (repository *UserAPIKeyRepository) UpdateProviderUsedCount(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, providerName core.ProviderName, usedCount int) (returnedError error) {
	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}
	update := bson.M{
		"$set": bson.M{
			"providerAccess.$[target].usedCount": usedCount,
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateExpireTime 更新 API Key 的到期時間
func (repository *UserAPIKeyRepository) UpdateExpireTime(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, expireTime time.Time) (returnedError error) {
	filter := bson.M{"_id": apiKeyIdentifier}
	update := bson.M{"$set": bson.M{"expireTime": expireTime.UTC()}}

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update))
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// IncrProviderUsedCount 累加已用次數
func (repository *UserAPIKeyRepository) IncrProviderUsedCount(
	contextValue context.Context,
	apiKeyIdentifier primitive.ObjectID,
	providerName core.ProviderName,
	increment int,
) (returnedError error) {
	if increment == 0 {
		return nil
	}

	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}
	update := bson.M{
		"$inc": bson.M{
			"providerAccess.$[target].usedCount": increment, // ★ 使用傳入 increment
		},
		"$set": bson.M{
			"providerAccess.$[target].lastSeen": time.Now().UTC(),
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateProviderLastResetAt
func (repository *UserAPIKeyRepository) UpdateProviderLastResetAt(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, providerName core.ProviderName, resetTime time.Time) (returnedError error) {
	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}
	update := bson.M{
		"$set": bson.M{
			"providerAccess.$[target].lastResetAt": resetTime.UTC(),
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateLastSeen 更新最後看見時間
func (repository *UserAPIKeyRepository) UpdateLastSeen(
	contextValue context.Context,
	apiKeyIdentifier primitive.ObjectID,
	providerName core.ProviderName,
	lastSeenTime time.Time,
) (_ int64, returnedError error) {
	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}
	update := bson.M{
		"$set": bson.M{
			"providerAccess.$[target].lastSeen": lastSeenTime.UTC(),
		},
	}
	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return 0, updateError
	}
	return result.ModifiedCount, nil
}

// UpdateProviderFields 通用局部更新
func (repository *UserAPIKeyRepository) UpdateProviderFields(contextValue context.Context, apiKeyIdentifier primitive.ObjectID, providerName core.ProviderName, fields map[string]interface{}) (returnedError error) {
	if len(fields) == 0 {
		return nil
	}

	filter := bson.M{
		"_id":                     apiKeyIdentifier,
		"providerAccess.provider": providerName,
	}

	setData := bson.M{}
	for fieldName, fieldValue := range fields {
		setData["providerAccess.$[target]."+fieldName] = fieldValue
	}
	update := bson.M{"$set": setData}

	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{bson.M{"target.provider": providerName}},
	}
	opts := options.Update().SetArrayFilters(arrayFilters)

	result, updateError := repository.collection.UpdateOne(contextValue, filter, withUpdatedAt(update), opts)
	if updateError != nil {
		return updateError
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
