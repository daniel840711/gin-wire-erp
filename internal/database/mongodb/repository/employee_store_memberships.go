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

type EmployeeStoreMembershipRepository struct {
	collection *mongo.Collection
}

func NewEmployeeStoreMembershipRepository(mongoClient *client.MongoClient) *EmployeeStoreMembershipRepository {
	repository := &EmployeeStoreMembershipRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionEmployeeStoreMemberships)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *EmployeeStoreMembershipRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.EmployeeStoreMembershipIndexes)
	return nil
}

func (repository *EmployeeStoreMembershipRepository) Create(contextValue context.Context, membership *model.EmployeeStoreMembership) (_ *model.EmployeeStoreMembership, returnedError error) {
	if membership.ID.IsZero() {
		membership.ID = primitive.NewObjectID()
	}
	membership.CreatedAt = time.Now().UTC()

	insertResult, insertError := repository.collection.InsertOne(contextValue, membership)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	membership.ID = objectID
	return membership, nil
}

func (repository *EmployeeStoreMembershipRepository) GetByID(contextValue context.Context, membershipIdentifier primitive.ObjectID) (_ *model.EmployeeStoreMembership, returnedError error) {
	var membership model.EmployeeStoreMembership
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": membershipIdentifier}).Decode(&membership); returnedError != nil {
		return nil, returnedError
	}
	return &membership, nil
}

func (repository *EmployeeStoreMembershipRepository) List(contextValue context.Context, filter bson.M) (_ []*model.EmployeeStoreMembership, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.EmployeeStoreMembership
	for cursor.Next(contextValue) {
		var membership model.EmployeeStoreMembership
		if decodeError := cursor.Decode(&membership); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &membership)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *EmployeeStoreMembershipRepository) UpdateByID(contextValue context.Context, membershipIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": membershipIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *EmployeeStoreMembershipRepository) DeleteByID(contextValue context.Context, membershipIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": membershipIdentifier})
	return returnedError
}
