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

type OrganizationNodeRepository struct {
	collection *mongo.Collection
}

func NewOrganizationNodeRepository(mongoClient *client.MongoClient) *OrganizationNodeRepository {
	repository := &OrganizationNodeRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionOrganizationNodes)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *OrganizationNodeRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.OrganizationNodeIndexes)
	return nil
}

func (repository *OrganizationNodeRepository) Create(contextValue context.Context, node *model.OrganizationNode) (_ *model.OrganizationNode, returnedError error) {
	nowUTC := time.Now().UTC()
	if node.ID.IsZero() {
		node.ID = primitive.NewObjectID()
	}
	node.CreatedAt = nowUTC
	node.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, node)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	node.ID = objectID
	return node, nil
}

func (repository *OrganizationNodeRepository) GetByID(contextValue context.Context, nodeIdentifier primitive.ObjectID) (_ *model.OrganizationNode, returnedError error) {
	var node model.OrganizationNode
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": nodeIdentifier}).Decode(&node); returnedError != nil {
		return nil, returnedError
	}
	return &node, nil
}

func (repository *OrganizationNodeRepository) List(contextValue context.Context, filter bson.M) (_ []*model.OrganizationNode, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.OrganizationNode
	for cursor.Next(contextValue) {
		var node model.OrganizationNode
		if decodeError := cursor.Decode(&node); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &node)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *OrganizationNodeRepository) UpdateByID(contextValue context.Context, nodeIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": nodeIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *OrganizationNodeRepository) DeleteByID(contextValue context.Context, nodeIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": nodeIdentifier})
	return returnedError
}
