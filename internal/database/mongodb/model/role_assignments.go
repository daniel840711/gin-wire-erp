package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoleAssignment struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id"`
	TenantID    primitive.ObjectID  `json:"tenantId" bson:"tenantId"`
	SubjectType string              `json:"subjectType" bson:"subjectType"`
	SubjectID   primitive.ObjectID  `json:"subjectId" bson:"subjectId"`
	RoleID      primitive.ObjectID  `json:"roleId" bson:"roleId"`
	ScopeType   string              `json:"scopeType" bson:"scopeType"`
	ScopeID     *primitive.ObjectID `json:"scopeId,omitempty" bson:"scopeId,omitempty"`
	Effect      string              `json:"effect,omitempty" bson:"effect,omitempty"`
	ExpiresAt   *time.Time          `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"`
	CreatedBy   primitive.ObjectID  `json:"createdBy" bson:"createdBy"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
}

var RoleAssignmentIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "subjectType", Value: 1}, {Key: "subjectId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_subjectType_subjectId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "roleId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_roleId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "scopeType", Value: 1}, {Key: "scopeId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_scopeType_scopeId"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "expiresAt", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_expiresAt"),
	},
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "createdBy", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_createdBy"),
	},
}
