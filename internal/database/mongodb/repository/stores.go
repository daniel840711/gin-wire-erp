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

type StoreRepository struct {
	collection *mongo.Collection
}

func NewStoreRepository(mongoClient *client.MongoClient) *StoreRepository {
	repository := &StoreRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionStores)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *StoreRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.StoreIndexes)
	return nil
}

func (repository *StoreRepository) Create(contextValue context.Context, store *model.Store) (_ *model.Store, returnedError error) {
	nowUTC := time.Now().UTC()
	if store.ID.IsZero() {
		store.ID = primitive.NewObjectID()
	}
	store.CreatedAt = nowUTC
	store.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, store)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	store.ID = objectID
	return store, nil
}

func (repository *StoreRepository) GetByID(contextValue context.Context, storeIdentifier primitive.ObjectID) (_ *model.Store, returnedError error) {
	var store model.Store
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": storeIdentifier}).Decode(&store); returnedError != nil {
		return nil, returnedError
	}
	return &store, nil
}

func (repository *StoreRepository) List(contextValue context.Context, filter bson.M) (_ []*model.Store, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.Store
	for cursor.Next(contextValue) {
		var store model.Store
		if decodeError := cursor.Decode(&store); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &store)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *StoreRepository) UpdateByID(contextValue context.Context, storeIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": storeIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *StoreRepository) DeleteByID(contextValue context.Context, storeIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": storeIdentifier})
	return returnedError
}
