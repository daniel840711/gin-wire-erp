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

type EmployeeRepository struct {
	collection *mongo.Collection
}

func NewEmployeeRepository(mongoClient *client.MongoClient) *EmployeeRepository {
	repository := &EmployeeRepository{
		collection: mongoClient.Client().Database(string(core.MongoDBInterchange)).Collection(string(core.MongoCollectionEmployees)),
	}
	_ = repository.ensureIndexes(context.Background())
	return repository
}

func (repository *EmployeeRepository) ensureIndexes(contextValue context.Context) error {
	ctx := contextValue

	_, _ = repository.collection.Indexes().CreateMany(ctx, model.EmployeeIndexes)
	return nil
}

func (repository *EmployeeRepository) Create(contextValue context.Context, employee *model.Employee) (_ *model.Employee, returnedError error) {
	nowUTC := time.Now().UTC()
	if employee.ID.IsZero() {
		employee.ID = primitive.NewObjectID()
	}
	employee.CreatedAt = nowUTC
	employee.UpdatedAt = nowUTC

	insertResult, insertError := repository.collection.InsertOne(contextValue, employee)
	if insertError != nil {
		return nil, insertError
	}
	objectID, ok := insertResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected InsertedID type: %T", insertResult.InsertedID)
	}
	employee.ID = objectID
	return employee, nil
}

func (repository *EmployeeRepository) GetByID(contextValue context.Context, employeeIdentifier primitive.ObjectID) (_ *model.Employee, returnedError error) {
	var employee model.Employee
	if returnedError = repository.collection.FindOne(contextValue, bson.M{"_id": employeeIdentifier}).Decode(&employee); returnedError != nil {
		return nil, returnedError
	}
	return &employee, nil
}

func (repository *EmployeeRepository) List(contextValue context.Context, filter bson.M) (_ []*model.Employee, returnedError error) {
	cursor, findError := repository.collection.Find(contextValue, filter)
	if findError != nil {
		return nil, findError
	}
	defer cursor.Close(contextValue)

	var results []*model.Employee
	for cursor.Next(contextValue) {
		var employee model.Employee
		if decodeError := cursor.Decode(&employee); decodeError != nil {
			return nil, decodeError
		}
		results = append(results, &employee)
	}
	if cursorError := cursor.Err(); cursorError != nil {
		return nil, cursorError
	}

	return results, nil
}

func (repository *EmployeeRepository) UpdateByID(contextValue context.Context, employeeIdentifier primitive.ObjectID, update bson.M) (returnedCount int64, returnedError error) {
	result, updateError := repository.collection.UpdateOne(contextValue, bson.M{"_id": employeeIdentifier}, withUpdatedAt(update))
	if updateError != nil {
		return 0, updateError
	}
	if result.MatchedCount == 0 {
		return 0, mongo.ErrNoDocuments
	}
	return result.MatchedCount, nil
}

func (repository *EmployeeRepository) DeleteByID(contextValue context.Context, employeeIdentifier primitive.ObjectID) (returnedError error) {
	_, returnedError = repository.collection.DeleteOne(contextValue, bson.M{"_id": employeeIdentifier})
	return returnedError
}
