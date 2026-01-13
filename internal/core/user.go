package core

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role string

const (
	RoleAdmin    Role = "admin"    // 管理員：可編輯所有人
	RoleEditor   Role = "editor"   // 可編輯內容，但不能做高權限操作
	RoleUser     Role = "user"     // 一般使用者
	RoleReadOnly Role = "readonly" // 只能查詢，不能改資料
	RoleBanned   Role = "banned"   // 被禁用，無法登入或操作
)

type Status string

const (
	StatusActive      Status = "active"      // 正常可用
	StatusBlocked     Status = "blocked"     // 被封鎖（例如濫用）
	StatusSuspended   Status = "suspended"   // 暫停（違規調查中）
	StatusExpired     Status = "expired"     // 已過期
	StatusRevoked     Status = "revoked"     // 被手動撤銷
	StatusMaintenance Status = "maintenance" // 系統維護中（暫時停用）
	StatusPending     Status = "pending"     // 尚未啟用（等待審核/激活）
	StatusDeleted     Status = "deleted"     // 已刪除（軟刪除）
)

type UserAPIKeyQuery struct {
	UserID      primitive.ObjectID
	Provider    string
	Status      string
	KeyName     string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}
