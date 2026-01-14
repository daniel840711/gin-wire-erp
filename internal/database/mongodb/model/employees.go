package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Employee struct {
	ID                        primitive.ObjectID  `json:"id" bson:"_id"`
	TenantID                  primitive.ObjectID  `json:"tenantId" bson:"tenantId"`
	Account                   string              `json:"account" bson:"account"`
	PasswordHash              string              `json:"passwordHash" bson:"passwordHash"`
	DisplayName               string              `json:"displayName" bson:"displayName"`
	Status                    string              `json:"status" bson:"status"`
	PrimaryOrganizationNodeID primitive.ObjectID  `json:"primaryOrganizationNodeId" bson:"primaryOrganizationNodeId"`
	PrimaryStoreID            *primitive.ObjectID `json:"primaryStoreId,omitempty" bson:"primaryStoreId,omitempty"`
	JobTitle                  string              `json:"jobTitle,omitempty" bson:"jobTitle,omitempty"`
	ReportToEmployeeID        *primitive.ObjectID `json:"reportToEmployeeId,omitempty" bson:"reportToEmployeeId,omitempty"`
	CreatedAt                 time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt                 time.Time           `json:"updatedAt" bson:"updatedAt"`
}

var EmployeeIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "account", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_account").SetUnique(true),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "primaryOrganizationNodeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_primaryOrganizationNodeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "primaryStoreId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_primaryStoreId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "reportToEmployeeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_reportToEmployeeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "status", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_status"),
	},
}
