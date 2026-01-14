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

type RoleRepository struct {
	collection *mongo.Collection
}

func NewRoleRepository(mongoClient *client.MongoClient) *RoleRepository {
	repository := &RoleRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionRoles)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *RoleRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.RoleIndexes)
	return nil
}

func (repository *RoleRepository) Create(contextValue context.Context, role *model.Role) (_ *model.Role, returnedError error) {
	nowUTC := time.Now().UTC()
	if role.ID.IsZero() {
		role.ID = primitive.NewObjectID()
	}
	role.CreatedAt = nowUTC
	role.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, role)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	role.ID = objectID
	return role, nil
}

func (repository *RoleRepository) GetByID(contextValue context.Context, roleIdentifier primitive.ObjectID) (_ *model.Role, returnedError error) {
	var role model.Role
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": roleIdentifier}).Decode(&role); returnedError != nil {
		return nil, returnedError
	}
	return &role, nil
}

func (repository *RoleRepository) List(contextValue context.Context, filter bson.M) (_ []*model.Role, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.Role
	for cursor.Next(contextValue) {
		var role model.Role
		if decodeError := cursor.Decode(&role); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &role)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *RoleRepository) UpdateByID(contextValue context.Context, roleIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": roleIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *RoleRepository) DeleteByID(contextValue context.Context, roleIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": roleIdentifier})
	return returnedError
}
