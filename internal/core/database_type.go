package core

import "go.mongodb.org/mongo-driver/bson"

// ─── Database Types ────────────────────────────────────────────────────────────

// DatabaseType defines the type of database
type DatabaseType string

const (
	MySQL DatabaseType = "mysql"
	Mongo DatabaseType = "mongo"
	Redis DatabaseType = "redis"
)

// Databases contains all supported database types
var Databases = []DatabaseType{MySQL, Mongo, Redis}

// MySQLDatabaseName defines the database instance names
type MySQLDatabaseName string
type MongoDatabaseName string
type MongoCollection string
type RedisKey string
type FluentdSubTag string

const (
	MySQLDBMaster MySQLDatabaseName = "master"
	MySQLDBSlave  MySQLDatabaseName = "slave"
	MySQLDBLog    MySQLDatabaseName = "log"
	MySQLDBStats  MySQLDatabaseName = "stats"
)

// ─── MongoDB ───────────────────────────────────────────────────────────────────
const (
	MongoDBInterchange MongoDatabaseName = "uptown"
)

// MongoDB collections
const (
	MongoCollectionUsers                           MongoCollection = "interchange_users"
	MongoCollectionUserAPIKeys                     MongoCollection = "interchange_user_api_keys"
	MongoCollectionTenants                         MongoCollection = "tenants"
	MongoCollectionOrganizationNodes               MongoCollection = "organization_nodes"
	MongoCollectionStores                          MongoCollection = "stores"
	MongoCollectionEmployees                       MongoCollection = "employees"
	MongoCollectionEmployeeOrganizationMemberships MongoCollection = "employee_organization_memberships"
	MongoCollectionEmployeeStoreMemberships        MongoCollection = "employee_store_memberships"
	MongoCollectionPermissions                     MongoCollection = "permissions"
	MongoCollectionRoles                           MongoCollection = "roles"
	MongoCollectionRolePermissions                 MongoCollection = "role_permissions"
	MongoCollectionRoleAssignments                 MongoCollection = "role_assignments"
	MongoCollectionPermissionPolicies              MongoCollection = "permission_policies"
	MongoCollectionPolicyRules                     MongoCollection = "policy_rules"
	MongoCollectionAssignmentPolicies              MongoCollection = "assignment_policies"
)

// ─── Redis Keys ────────────────────────────────────────────────────────────────

const (
	RedisKeySession      RedisKey = "session"         // 使用者 session 資料
	RedisKeyBlacklist    RedisKey = "blacklist_token" // 黑名單 token
	RedisKeyRefreshToken RedisKey = "refresh_token"   // refresh token
	RedisKeyServerName   RedisKey = "interchange"     // 伺服器名稱
)

const (
	FluentdRequest  FluentdSubTag = "request_log"
	FluentdResponse FluentdSubTag = "response_log"
	FluentUsage     FluentdSubTag = "interchange_usage_log"
)

type ListOptions struct {
	Filter bson.M `json:"filter,omitempty" bson:"filter,omitempty"`
	Page   int64  `json:"page,omitempty" bson:"page,omitempty"`
	Size   int64  `json:"size,omitempty" bson:"size,omitempty"`
}
