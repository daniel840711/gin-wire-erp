package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PermissionPolicy struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
}

var PermissionPolicyIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_name").SetUnique(true),
	},
}
