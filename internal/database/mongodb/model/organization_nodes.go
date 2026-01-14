package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrganizationNode struct {
	ID                primitive.ObjectID  `json:"id" bson:"_id"`
	TenantID          primitive.ObjectID  `json:"tenantId" bson:"tenantId"`
	Name              string              `json:"name" bson:"name"`
	Type              string              `json:"type" bson:"type"`
	ParentID          *primitive.ObjectID `json:"parentId,omitempty" bson:"parentId,omitempty"`
	Path              string              `json:"path" bson:"path"`
	Depth             int                 `json:"depth" bson:"depth"`
	ManagerEmployeeID *primitive.ObjectID `json:"managerEmployeeId,omitempty" bson:"managerEmployeeId,omitempty"`
	Status            string              `json:"status" bson:"status"`
	CreatedAt         time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt" bson:"updatedAt"`
}

var OrganizationNodeIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "parentId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_parentId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "path", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_path"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "managerEmployeeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_managerEmployeeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "status", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_status"),
	},
}
