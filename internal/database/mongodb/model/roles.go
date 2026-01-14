package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Role struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	Name      string             `json:"name" bson:"name"`
	Code      string             `json:"code,omitempty" bson:"code,omitempty"`
	IsSystem  bool               `json:"isSystem" bson:"isSystem"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

var RoleIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_name").SetUnique(true),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "code", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_code").SetUnique(true),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "status", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_status"),
	},
}
