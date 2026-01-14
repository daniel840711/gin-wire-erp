package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RolePermission struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	TenantID     primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	RoleID       primitive.ObjectID `json:"roleId" bson:"roleId"`
	PermissionID primitive.ObjectID `json:"permissionId" bson:"permissionId"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
}

var RolePermissionIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "roleId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_roleId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "permissionId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_permissionId"),
	},
	{
		Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "roleId", Value: 1}, {Key: "permissionId", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_roleId_permissionId").
			SetUnique(true),
	},
}
