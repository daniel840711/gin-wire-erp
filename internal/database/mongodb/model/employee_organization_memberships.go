package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EmployeeOrganizationMembership struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id"`
	TenantID           primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	EmployeeID         primitive.ObjectID `json:"employeeId" bson:"employeeId"`
	OrganizationNodeID primitive.ObjectID `json:"organizationNodeId" bson:"organizationNodeId"`
	RoleInOrganization string             `json:"roleInOrganization,omitempty" bson:"roleInOrganization,omitempty"`
	CreatedAt          time.Time          `json:"createdAt" bson:"createdAt"`
}

var EmployeeOrganizationMembershipIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "employeeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_employeeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "organizationNodeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_organizationNodeId"),
	},
	{
		Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "employeeId", Value: 1}, {Key: "organizationNodeId", Value: 1}},
		Options: options.Index().SetName("uniq_tenantId_employeeId_organizationNodeId").
			SetUnique(true),
	},
}
