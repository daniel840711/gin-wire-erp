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

type EmployeeOrganizationMembershipRepository struct {
	collection *mongo.Collection
}

func NewEmployeeOrganizationMembershipRepository(mongoClient *client.MongoClient) *EmployeeOrganizationMembershipRepository {
	repository := &EmployeeOrganizationMembershipRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionEmployeeOrganizationMemberships)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *EmployeeOrganizationMembershipRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.EmployeeOrganizationMembershipIndexes)
	return nil
}

func (repository *EmployeeOrganizationMembershipRepository) Create(contextValue context.Context, membership *model.EmployeeOrganizationMembership) (_ *model.EmployeeOrganizationMembership, returnedError error) {
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

func (repository *EmployeeOrganizationMembershipRepository) GetByID(contextValue context.Context, membershipIdentifier primitive.ObjectID) (_ *model.EmployeeOrganizationMembership, returnedError error) {
	var membership model.EmployeeOrganizationMembership
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": membershipIdentifier}).Decode(&membership); returnedError != nil {
		return nil, returnedError
	}
	return &membership, nil
}

func (repository *EmployeeOrganizationMembershipRepository) List(contextValue context.Context, filter bson.M) (_ []*model.EmployeeOrganizationMembership, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.EmployeeOrganizationMembership
	for cursor.Next(contextValue) {
		var membership model.EmployeeOrganizationMembership
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

func (repository *EmployeeOrganizationMembershipRepository) UpdateByID(contextValue context.Context, membershipIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": membershipIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *EmployeeOrganizationMembershipRepository) DeleteByID(contextValue context.Context, membershipIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": membershipIdentifier})
	return returnedError
}
