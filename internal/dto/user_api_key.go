package dto

import (
	"interchange/internal/core"
	"time"
)

type ProviderAccessDto struct {
	Provider    core.ProviderName `json:"provider" binding:"required"`     // openai, gemini...
	ProviderKey string            `json:"providerKey" binding:"omitempty"` // 可選，對應 Provider 的 API Key
	Status      core.Status       `json:"status" binding:"required"`       // active, blocked...
	LimitPeriod *core.LimitPeriod `json:"limitPeriod" binding:"omitempty"` // daily, weekly, monthly...
	LimitCount  *int              `json:"limitCount" binding:"omitempty"`  // 限制次數
	UsedCount   int               `json:"usedCount" binding:"omitempty"`   // 已使用次數
	LastResetAt *time.Time        `json:"lastResetAt" binding:"omitempty"` // 最後重置時間
	ApiScopes   []core.ApiScope   `json:"apiScopes" binding:"omitempty"`   // 可選，API 權限範圍
	ExpireTime  *time.Time        `json:"expireTime" binding:"omitempty"`  // API Key 過期時間
	LastSeen    *time.Time        `json:"lastSeen" binding:"omitempty"`    // 最後使用時間
}

// 建立 API Key
type CreateUserAPIKeyDto struct {
	KeyName        string              `json:"keyName" binding:"required"`
	KeyValue       string              `json:"keyValue" binding:"omitempty"`
	ProviderAccess []ProviderAccessDto `json:"providerAccess" binding:"required"`
}

// 更新 API Key（允許部分欄位可選）
type UpdateUserAPIKeyDto struct {
	KeyName        *string              `json:"keyName,omitempty"`
	ProviderAccess *[]ProviderAccessDto `json:"providerAccess,omitempty"`
}

type UserAPIKeyResponseDto struct {
	ID             string              `json:"id"`
	UserID         string              `json:"userId"`
	KeyName        string              `json:"keyName"`
	KeyValue       string              `json:"keyValue"`
	ProviderAccess []ProviderAccessDto `json:"providerAccess"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
}

type UpdateProviderAccessAllDto struct {
	ProviderAccess []ProviderAccessDto `json:"providerAccess" binding:"required"`
}
