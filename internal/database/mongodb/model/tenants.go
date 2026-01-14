package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Tenant struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Name      string             `json:"name" bson:"name"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

var TenantIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetName("uniq_name").SetUnique(true),
	},
	{
		Keys:    bson.D{{Key: "status", Value: 1}},
		Options: options.Index().SetName("idx_status"),
	},
}
