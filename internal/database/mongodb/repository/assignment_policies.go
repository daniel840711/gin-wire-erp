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

type AssignmentPolicyRepository struct {
	collection *mongo.Collection
}

func NewAssignmentPolicyRepository(mongoClient *client.MongoClient) *AssignmentPolicyRepository {
	repository := &AssignmentPolicyRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionAssignmentPolicies)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *AssignmentPolicyRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.AssignmentPolicyIndexes)
	return nil
}

func (repository *AssignmentPolicyRepository) Create(contextValue context.Context, assignment *model.AssignmentPolicy) (_ *model.AssignmentPolicy, returnedError error) {
	if assignment.ID.IsZero() {
		assignment.ID = primitive.NewObjectID()
	}

	insertResult, insertError := repository.collection.InsertOne(contextValue, assignment)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	assignment.ID = objectID
	return assignment, nil
}

func (repository *AssignmentPolicyRepository) GetByID(contextValue context.Context, assignmentIdentifier primitive.ObjectID) (_ *model.AssignmentPolicy, returnedError error) {
	var assignment model.AssignmentPolicy
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": assignmentIdentifier}).Decode(&assignment); returnedError != nil {
		return nil, returnedError
	}
	return &assignment, nil
}

func (repository *AssignmentPolicyRepository) List(contextValue context.Context, filter bson.M) (_ []*model.AssignmentPolicy, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.AssignmentPolicy
	for cursor.Next(contextValue) {
		var assignment model.AssignmentPolicy
		if decodeError := cursor.Decode(&assignment); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &assignment)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *AssignmentPolicyRepository) UpdateByID(contextValue context.Context, assignmentIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": assignmentIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *AssignmentPolicyRepository) DeleteByID(contextValue context.Context, assignmentIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": assignmentIdentifier})
	return returnedError
}
