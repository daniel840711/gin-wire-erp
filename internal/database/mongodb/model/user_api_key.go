package model

import (
	"interchange/internal/core"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserAPIKey struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`                              // 使用者 API Key 唯一識別碼
	UserID         primitive.ObjectID `json:"userID" bson:"userID"`                       // 所屬使用者 ID
	KeyName        string             `json:"keyName,omitempty" bson:"keyName,omitempty"` // API Key 名稱
	KeyValue       string             `json:"keyValue" bson:"keyValue"`                   // API Key 值
	ProviderAccess []ProviderAccess   `json:"providerAccess" bson:"providerAccess"`       // 各個 provider 的存取權限
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`                 // 建立時間
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`                 // 更新時間
}
type ProviderAccess struct {
	Provider    core.ProviderName `json:"provider" bson:"provider"`                           // openai, gemini...
	ProviderKey string            `json:"providerKey" bson:"providerKey"`                     // 該 provider 綁定的真實 key（可選，安全考慮可 hash）
	Status      core.Status       `json:"status" bson:"status"`                               // active, blocked...
	LimitPeriod *core.LimitPeriod `json:"limitPeriod,omitempty" bson:"limitPeriod,omitempty"` // daily, weekly, monthly...
	LimitCount  *int              `json:"limitCount,omitempty" bson:"limitCount,omitempty"`   // 該 period 的使用次數限制
	UsedCount   int               `json:"usedCount" bson:"usedCount"`                         // 該 period 的已使用次數
	LastResetAt *time.Time        `json:"lastResetAt,omitempty" bson:"lastResetAt,omitempty"` // 該 period 的最後重置時間
	ApiScopes   []core.ApiScope   `json:"apiScopes" bson:"apiScopes"`                         // 可用的 endpoint 清單
	ExpireTime  *time.Time        `json:"expireTime,omitempty" bson:"expireTime,omitempty"`   // API Key 過期時間
	LastSeen    *time.Time        `json:"lastSeen,omitempty" bson:"lastSeen,omitempty"`       // 最後使用時間
}
