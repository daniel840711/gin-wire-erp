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

type RoleAssignmentRepository struct {
	collection *mongo.Collection
}

func NewRoleAssignmentRepository(mongoClient *client.MongoClient) *RoleAssignmentRepository {
	repository := &RoleAssignmentRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionRoleAssignments)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *RoleAssignmentRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.RoleAssignmentIndexes)
	return nil
}

func (repository *RoleAssignmentRepository) Create(contextValue context.Context, assignment *model.RoleAssignment) (_ *model.RoleAssignment, returnedError error) {
	if assignment.ID.IsZero() {
		assignment.ID = primitive.NewObjectID()
	}
	assignment.CreatedAt = time.Now().UTC()

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

func (repository *RoleAssignmentRepository) GetByID(contextValue context.Context, assignmentIdentifier primitive.ObjectID) (_ *model.RoleAssignment, returnedError error) {
	var assignment model.RoleAssignment
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": assignmentIdentifier}).Decode(&assignment); returnedError != nil {
		return nil, returnedError
	}
	return &assignment, nil
}

func (repository *RoleAssignmentRepository) List(contextValue context.Context, filter bson.M) (_ []*model.RoleAssignment, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.RoleAssignment
	for cursor.Next(contextValue) {
		var assignment model.RoleAssignment
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

func (repository *RoleAssignmentRepository) UpdateByID(contextValue context.Context, assignmentIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": assignmentIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *RoleAssignmentRepository) DeleteByID(contextValue context.Context, assignmentIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": assignmentIdentifier})
	return returnedError
}
