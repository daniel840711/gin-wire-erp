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
)

type TenantRepository struct {
	collection *mongo.Collection
}

func NewTenantRepository(mongoClient *client.MongoClient) *TenantRepository {
	repository := &TenantRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionTenants)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *TenantRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.TenantIndexes)
	return nil
}

func (repository *TenantRepository) Create(contextValue context.Context, tenant *model.Tenant) (_ *model.Tenant, returnedError error) {
	nowUTC := time.Now().UTC()
	if tenant.ID.IsZero() {
		tenant.ID = primitive.NewObjectID()
	}
	tenant.CreatedAt = nowUTC
	tenant.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, tenant)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	tenant.ID = objectID
	return tenant, nil
}

func (repository *TenantRepository) GetByID(contextValue context.Context, tenantIdentifier primitive.ObjectID) (_ *model.Tenant, returnedError error) {
	var tenant model.Tenant
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": tenantIdentifier}).Decode(&tenant); returnedError != nil {
		return nil, returnedError
	}
	return &tenant, nil
}

func (repository *TenantRepository) List(contextValue context.Context, filter bson.M) (_ []*model.Tenant, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.Tenant
	for cursor.Next(contextValue) {
		var tenant model.Tenant
		if decodeError := cursor.Decode(&tenant); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &tenant)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *TenantRepository) UpdateByID(contextValue context.Context, tenantIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": tenantIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *TenantRepository) DeleteByID(contextValue context.Context, tenantIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": tenantIdentifier})
	return returnedError
}
