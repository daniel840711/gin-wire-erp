package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EmployeeStoreMembership struct {
	ID         primitive.ObjectID `json:"id" bson:"_id"`
	TenantID   primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	EmployeeID primitive.ObjectID `json:"employeeId" bson:"employeeId"`
	StoreID    primitive.ObjectID `json:"storeId" bson:"storeId"`
	Position   string             `json:"position,omitempty" bson:"position,omitempty"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
}

var EmployeeStoreMembershipIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "employeeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_employeeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "storeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_storeId"),
	},
	{
		Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "employeeId", Value: 1}, {Key: "storeId", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_employeeId_storeId").
			SetUnique(true),
	},
}
