package apikey

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"interchange/internal/core"
	"interchange/internal/database/mongodb/model"
	cErr "interchange/internal/pkg/error"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIKeyPayload 是 API Key 的內容
type APIKeyPayload struct {
	UserID   string `json:"userID"`
	ApiKeyID string `json:"apiKeyID"`
	IssuedAt int64  `json:"issuedAt"`
}

// 產生 API Key
func GenerateAPIKey(userID, apiKeyID, secret string) (string, error) {
	payload := APIKeyPayload{
		UserID:   userID,
		ApiKeyID: apiKeyID,
		IssuedAt: time.Now().Unix(),
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	pb64 := base64.RawURLEncoding.EncodeToString(pb)
	sig := signShort(pb64, secret)
	return pb64 + "." + sig, nil
}

// 驗證並解析 API Key
func ParseAndVerifyAPIKey(apiKey, secret string) (*APIKeyPayload, error) {
	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid api key format")
	}
	pb64, sig := parts[0], parts[1]
	if signShort(pb64, secret) != sig {
		return nil, errors.New("invalid api key signature")
	}
	pb, err := base64.RawURLEncoding.DecodeString(pb64)
	if err != nil {
		return nil, err
	}
	var pl APIKeyPayload
	if err := json.Unmarshal(pb, &pl); err != nil {
		return nil, err
	}
	return &pl, nil
}

// 只解析 payload，不驗證（用於管理工具，不建議用於認證）
func DecodeAPIKeyPayload(apiKey string) (*APIKeyPayload, error) {
	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid api key format")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var pl APIKeyPayload
	if err := json.Unmarshal(pb, &pl); err != nil {
		return nil, err
	}
	return &pl, nil
}

// HMAC-SHA256 簽章，僅取前16字元
func signShort(pb64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(pb64))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:16]
}

// GetActiveProvider 從 gin.Context 中取得指定的活躍 ProviderAccess
func GetActiveProvider(c *gin.Context, provider core.ProviderName) (*model.ProviderAccess, *cErr.Error) {
	raw, exists := c.Get("providerAccess")
	if !exists {
		return nil, cErr.UnauthorizedApiKey("missing provider access data")
	}
	accessList, ok := raw.([]model.ProviderAccess)
	if !ok {
		return nil, cErr.InternalServer("invalid provider access data")
	}
	for _, acc := range accessList {
		if acc.Provider == provider && acc.Status == core.StatusActive {
			return &acc, nil
		}
	}
	return nil, cErr.UnauthorizedApiKey("no active provider key found")
}
