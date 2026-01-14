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

type PolicyRuleRepository struct {
	collection *mongo.Collection
}

func NewPolicyRuleRepository(mongoClient *client.MongoClient) *PolicyRuleRepository {
	repository := &PolicyRuleRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionPolicyRules)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *PolicyRuleRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.PolicyRuleIndexes)
	return nil
}

func (repository *PolicyRuleRepository) Create(contextValue context.Context, rule *model.PolicyRule) (_ *model.PolicyRule, returnedError error) {
	if rule.ID.IsZero() {
		rule.ID = primitive.NewObjectID()
	}

	insertResult, insertError := repository.collection.InsertOne(contextValue, rule)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	rule.ID = objectID
	return rule, nil
}

func (repository *PolicyRuleRepository) GetByID(contextValue context.Context, ruleIdentifier primitive.ObjectID) (_ *model.PolicyRule, returnedError error) {
	var rule model.PolicyRule
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": ruleIdentifier}).Decode(&rule); returnedError != nil {
		return nil, returnedError
	}
	return &rule, nil
}

func (repository *PolicyRuleRepository) List(contextValue context.Context, filter bson.M) (_ []*model.PolicyRule, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.PolicyRule
	for cursor.Next(contextValue) {
		var rule model.PolicyRule
		if decodeError := cursor.Decode(&rule); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &rule)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *PolicyRuleRepository) UpdateByID(contextValue context.Context, ruleIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": ruleIdentifier}, update)
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *PolicyRuleRepository) DeleteByID(contextValue context.Context, ruleIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": ruleIdentifier})
	return returnedError
}
