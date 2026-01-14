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

type RolePermissionRepository struct {
	collection *mongo.Collection
}

func NewRolePermissionRepository(mongoClient *client.MongoClient) *RolePermissionRepository {
	repository := &RolePermissionRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionRolePermissions)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *RolePermissionRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.RolePermissionIndexes)
	return nil
}

func (repository *RolePermissionRepository) Create(contextValue context.Context, rolePermission *model.RolePermission) (_ *model.RolePermission, returnedError error) {
	if rolePermission.ID.IsZero() {
		rolePermission.ID = primitive.NewObjectID()
	}
	rolePermission.CreatedAt = time.Now().UTC()

	insertResult, insertError := repository.collection.InsertOne(contextValue, rolePermission)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	rolePermission.ID = objectID
	return rolePermission, nil
}

func (repository *RolePermissionRepository) GetByID(contextValue context.Context, rolePermissionIdentifier primitive.ObjectID) (_ *model.RolePermission, returnedError error) {
	var rolePermission model.RolePermission
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": rolePermissionIdentifier}).Decode(&rolePermission); returnedError != nil {
		return nil, returnedError
	}
	return &rolePermission, nil
}

func (repository *RolePermissionRepository) List(contextValue context.Context, filter bson.M) (_ []*model.RolePermission, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.RolePermission
	for cursor.Next(contextValue) {
		var rolePermission model.RolePermission
		if decodeError := cursor.Decode(&rolePermission); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &rolePermission)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *RolePermissionRepository) UpdateByID(contextValue context.Context, rolePermissionIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": rolePermissionIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *RolePermissionRepository) DeleteByID(contextValue context.Context, rolePermissionIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": rolePermissionIdentifier})
	return returnedError
}
