package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Permission struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id"`
	TenantID    *primitive.ObjectID `json:"tenantId,omitempty" bson:"tenantId,omitempty"`
	Key         string              `json:"key" bson:"key"`
	Resource    string              `json:"resource" bson:"resource"`
	Action      string              `json:"action" bson:"action"`
	Description string              `json:"description,omitempty" bson:"description,omitempty"`
	Status      string              `json:"status" bson:"status"`
}

var PermissionIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "key", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_key").SetUnique(true),
	},
	{
		Keys:    bson.D{{Key: "resource", Value: 1}, {Key: "action", Value: 1}},
		Options: options.Index().SetName("idx_resource_action"),
	},
}
