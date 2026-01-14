package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	ID                 primitive.ObjectID  `json:"id" bson:"_id"`
	TenantID           primitive.ObjectID  `json:"tenantId" bson:"tenantId"`
	Code               string              `json:"code" bson:"code"`
	Name               string              `json:"name" bson:"name"`
	OrganizationNodeID primitive.ObjectID  `json:"organizationNodeId" bson:"organizationNodeId"`
	Region             string              `json:"region,omitempty" bson:"region,omitempty"`
	Status             string              `json:"status" bson:"status"`
	CreatedAt          time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt" bson:"updatedAt"`
	ParentStoreID      *primitive.ObjectID `json:"parentStoreId,omitempty" bson:"parentStoreId,omitempty"`
	Path               string              `json:"path,omitempty" bson:"path,omitempty"`
}

var StoreIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "code", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_code").SetUnique(true),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "organizationNodeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_organizationNodeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "status", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_status"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "parentStoreId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_parentStoreId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "path", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_path"),
	},
}
