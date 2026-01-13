package model

import (
	"interchange/internal/core"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`                                      // 使用者唯一識別碼
	ExternalID  string             `json:"externalID,omitempty" bson:"externalID,omitempty"`   // 外部登入平台的使用者 ID
	DisplayName string             `json:"displayName,omitempty" bson:"displayName,omitempty"` // 使用者顯示名稱
	Email       string             `json:"email,omitempty" bson:"email,omitempty"`             // 使用者信箱
	Role        core.Role          `json:"role" bson:"role"`                                   // 使用者角色
	Status      core.Status        `json:"status" bson:"status"`                               // 帳號狀態
	LastSeen    *time.Time         `json:"lastSeen,omitempty" bson:"lastSeen,omitempty"`       // 最後使用時間
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`                         // 建立時間
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`                         // 更新時間
}
