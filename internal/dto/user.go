package dto

import (
	"interchange/internal/core"
	"time"
)

// 建立用戶
type CreateUserDto struct {
	ExternalID  string      `json:"externalID,omitempty"`                      // 外部平台 ID（如 Google/Lark）
	DisplayName string      `json:"displayName" binding:"required"`            // 顯示名稱
	Email       string      `json:"email,omitempty" binding:"omitempty,email"` // 信箱可選且格式驗證
	Lang        string      `json:"lang,omitempty"`                            // 語系（如 zh-TW, en）
	Role        core.Role   `json:"role" binding:"required"`                   // 角色
	Status      core.Status `json:"status" binding:"required"`                 // 狀態
}

// 更新用戶
type UpdateUserDto struct {
	ExternalID  *string      `json:"externalID,omitempty"`
	DisplayName *string      `json:"displayName,omitempty"`
	Email       *string      `json:"email,omitempty"`
	Lang        *string      `json:"lang,omitempty"`
	Role        *core.Role   `json:"role,omitempty"`
	Status      *core.Status `json:"status,omitempty"`
}

// 修改用戶狀態
type UpdateUserStatusDto struct {
	Status core.Status `json:"status" binding:"required"`
}

// 修改用戶角色
type UpdateUserRoleDto struct {
	Role core.Role `json:"role" binding:"required"`
}
type UserResponseDto struct {
	ID          string      `json:"id"`
	ExternalID  string      `json:"externalID,omitempty"`
	DisplayName string      `json:"displayName"`
	Email       string      `json:"email,omitempty"`
	Lang        string      `json:"lang,omitempty"`
	Role        core.Role   `json:"role"`
	Status      core.Status `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updatedAt" `
	LastSeen    *time.Time  `json:"last_seen,omitempty"`
}
