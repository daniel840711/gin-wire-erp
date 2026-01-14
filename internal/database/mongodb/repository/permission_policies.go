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

type PermissionPolicyRepository struct {
	collection *mongo.Collection
}

func NewPermissionPolicyRepository(mongoClient *client.MongoClient) *PermissionPolicyRepository {
	repository := &PermissionPolicyRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionPermissionPolicies)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *PermissionPolicyRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.PermissionPolicyIndexes)
	return nil
}

func (repository *PermissionPolicyRepository) Create(contextValue context.Context, policy *model.PermissionPolicy) (_ *model.PermissionPolicy, returnedError error) {
	if policy.ID.IsZero() {
		policy.ID = primitive.NewObjectID()
	}

	insertResult, insertError := repository.collection.InsertOne(contextValue, policy)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	policy.ID = objectID
	return policy, nil
}

func (repository *PermissionPolicyRepository) GetByID(contextValue context.Context, policyIdentifier primitive.ObjectID) (_ *model.PermissionPolicy, returnedError error) {
	var policy model.PermissionPolicy
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": policyIdentifier}).Decode(&policy); returnedError != nil {
		return nil, returnedError
	}
	return &policy, nil
}

func (repository *PermissionPolicyRepository) List(contextValue context.Context, filter bson.M) (_ []*model.PermissionPolicy, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.PermissionPolicy
	for cursor.Next(contextValue) {
		var policy model.PermissionPolicy
		if decodeError := cursor.Decode(&policy); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &policy)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *PermissionPolicyRepository) UpdateByID(contextValue context.Context, policyIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": policyIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *PermissionPolicyRepository) DeleteByID(contextValue context.Context, policyIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": policyIdentifier})
	return returnedError
}
