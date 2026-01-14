package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PolicyRule struct {
	ID         primitive.ObjectID `json:"id" bson:"_id"`
	TenantID   primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	PolicyID   primitive.ObjectID `json:"policyId" bson:"policyId"`
	Resource   string             `json:"resource" bson:"resource"`
	Action     string             `json:"action" bson:"action"`
	RuleType   string             `json:"ruleType" bson:"ruleType"`
	Conditions bson.M             `json:"conditions,omitempty" bson:"conditions,omitempty"`
}

var PolicyRuleIndexes = []mongo.IndexModel{
	{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "policyId", Value: 1}},
		Options: options.Index().SetName("idx_tenantId_policyId"),
	},
	{
		Keys:    bson.D{{Key: "resource", Value: 1}, {Key: "action", Value: 1}},
		Options: options.Index().SetName("idx_resource_action"),
	},
}
