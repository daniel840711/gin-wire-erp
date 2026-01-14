package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AssignmentPolicy struct {
	ID               primitive.ObjectID `json:"id" bson:"_id"`
	TenantID         primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	RoleAssignmentID primitive.ObjectID `json:"roleAssignmentId" bson:"roleAssignmentId"`
	PolicyID         primitive.ObjectID `json:"policyId" bson:"policyId"`
}

var AssignmentPolicyIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "roleAssignmentId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_roleAssignmentId"),
	},
	{
		Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "roleAssignmentId", Value: 1}, {Key: "policyId", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_roleAssignmentId_policyId").
			SetUnique(true),
	},
}
