package repository

import (
	"context"
	"fmt"

	"interchange/internal/core"
	client "interchange/internal/database/client"
	"interchange/internal/database/mongodb/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PermissionRepository struct {
	collection *mongo.Collection
}

func NewPermissionRepository(mongoClient *client.MongoClient) *PermissionRepository {
	repository := &PermissionRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionPermissions)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *PermissionRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.PermissionIndexes)
	return nil
}

func (repository *PermissionRepository) Create(contextValue context.Context, permission *model.Permission) (_ *model.Permission, returnedError error) {
	if permission.ID.IsZero() {
		permission.ID = primitive.NewObjectID()
	}

	insertResult, insertError := repository.collection.InsertOne(contextValue, permission)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	permission.ID = objectID
	return permission, nil
}

func (repository *PermissionRepository) GetByID(contextValue context.Context, permissionIdentifier primitive.ObjectID) (_ *model.Permission, returnedError error) {
	var permission model.Permission
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": permissionIdentifier}).Decode(&permission); returnedError != nil {
		return nil, returnedError
	}
	return &permission, nil
}

func (repository *PermissionRepository) List(contextValue context.Context, filter bson.M) (_ []*model.Permission, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.Permission
	for cursor.Next(contextValue) {
		var permission model.Permission
		if decodeError := cursor.Decode(&permission); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &permission)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *PermissionRepository) UpdateByID(contextValue context.Context, permissionIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": permissionIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *PermissionRepository) DeleteByID(contextValue context.Context, permissionIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": permissionIdentifier})
	return returnedError
}
