package service

import (
	"context"
	"fmt"
	"time"

	"interchange/config"
	"interchange/internal/core"
	"interchange/internal/database/mongodb/model"
	mongoDb "interchange/internal/database/mongodb/repository"
	redisDb "interchange/internal/database/redis/repository"
	"interchange/internal/dto"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/telemetry"
	"interchange/utils/apikey"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type UserAPIKeyService struct {
	trace           *telemetry.Trace
	userRepo        *mongoDb.UserRepository
	userApiKeyRepo  *mongoDb.UserAPIKeyRepository
	rateLimiterRepo *redisDb.RateLimiterRepository
	config          *config.Configuration
	logger          *zap.Logger
}

func NewUserAPIKeyService(
	trace *telemetry.Trace,
	userRepo *mongoDb.UserRepository,
	userApiKeyRepo *mongoDb.UserAPIKeyRepository,
	rateLimiterRepo *redisDb.RateLimiterRepository,
	config *config.Configuration,
	logger *zap.Logger,
) *UserAPIKeyService {
	return &UserAPIKeyService{
		trace:           trace,
		userRepo:        userRepo,
		userApiKeyRepo:  userApiKeyRepo,
		rateLimiterRepo: rateLimiterRepo,
		config:          config,
		logger:          logger,
	}
}

// Create 為指定使用者建立 API Key；若未提供 KeyValue 會自動依照 secret 產生。
// 成功回傳遮蔽後的 API Key 回應 DTO。
func (s *UserAPIKeyService) Create(
	ctx context.Context,
	userID primitive.ObjectID,
	dto *dto.CreateUserAPIKeyDto,
) (*dto.UserAPIKeyResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	apiKeyID := primitive.NewObjectID()
	apiKeyVal := dto.KeyValue
	if apiKeyVal == "" {
		gen, err := apikey.GenerateAPIKey(userID.Hex(), apiKeyID.Hex(), s.config.App.SecretKey)
		if err != nil {
			end(err)
			return nil, cErr.InternalServer("failed to generate api key")
		}
		apiKeyVal = gen
	}

	// 構建尚未入庫的 Key 實體
	key := &model.UserAPIKey{
		ID:             apiKeyID,
		UserID:         userID,
		KeyName:        dto.KeyName,
		KeyValue:       apiKeyVal,
		ProviderAccess: providerAccessDtosToModels(dto.ProviderAccess),
		CreatedAt:      time.Now(),
	}

	// 初始化每個 provider 的限流視窗與計數（Redis）
	apiKeyHex := key.ID.Hex()
	for _, pa := range key.ProviderAccess {
		if pa.LimitPeriod != nil && pa.LimitCount != nil {
			windowSec := periodToDuration(*pa.LimitPeriod)
			if err := s.rateLimiterRepo.Reset(ctx, apiKeyHex, pa.Provider, *pa.LimitPeriod, windowSec, pa.LimitCount); err != nil {
				return nil, cErr.DatabaseError("redis reset rate limit failed")
			}
		}
	}

	created, err := s.userApiKeyRepo.Create(ctx, key)
	if err != nil {
		return nil, cErr.DatabaseError("mongodb create api key failed")
	}
	return modelToUserAPIKeyResponseDto(created), nil
}

// ValidateKey 驗證外部送入的 API Key 字串是否合法並取回對應的 UserAPIKey 模型。
// 會驗證 payload、使用者存在性、API Key 存在性與字串簽章。
func (s *UserAPIKeyService) ValidateKey(ctx context.Context, apiKey string) (*model.UserAPIKey, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	// 簽章解析與驗證失敗：屬於授權失敗
	payload, err := apikey.ParseAndVerifyAPIKey(apiKey, s.config.App.SecretKey)
	if err != nil {
		end(err)
		return nil, cErr.UnauthorizedApiKey("invalid api key: signature verification failed")
	}

	// 驗證 payload.UserID
	if payload.UserID == "" {
		return nil, cErr.UnauthorizedApiKey("invalid api key: missing user ID")
	}
	userID, err := primitive.ObjectIDFromHex(payload.UserID)
	if err != nil {
		end(err)
		return nil, cErr.UnauthorizedApiKey("invalid api key: user ID is not a valid ObjectID")
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, cErr.DatabaseError("mongodb get user failed")
	}
	if user == nil {
		return nil, cErr.UnauthorizedApiKey("invalid api key: user not found")
	}

	// 驗證 payload.ApiKeyID
	if payload.ApiKeyID == "" {
		return nil, cErr.UnauthorizedApiKey("invalid api key: missing API key ID")
	}
	apiKeyID, err := primitive.ObjectIDFromHex(payload.ApiKeyID)
	if err != nil {
		end(err)
		return nil, cErr.UnauthorizedApiKey("invalid api key: key ID is not a valid ObjectID")
	}

	apiKeyModel, err := s.userApiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, cErr.DatabaseError("mongodb get api key failed")
	}
	if apiKeyModel == nil || apiKeyModel.KeyValue == "" {
		return nil, cErr.UnauthorizedApiKey("invalid api key: key not found")
	}

	return apiKeyModel, nil
}

// ValidateProviderAccess 驗證 API Key 是否有指定 provider 的存取權限，且該權限為啟用中且未過期，更新狀態為過期的會自動標記。
func (s *UserAPIKeyService) ValidateProviderAccess(
	ctx context.Context,
	data *model.UserAPIKey,
	provider core.ProviderName,
) (*model.ProviderAccess, error) {
	ctx, span, end := s.trace.WithSpan(ctx)
	defer end(nil)
	now := time.Now().UTC()
	providerAccess, err := getActiveProvider(data.ProviderAccess, provider)
	if err != nil || providerAccess == nil {
		s.trace.ApplyTraceAttributes(span, struct {
			Status   string `trace:"auth.status"`
			Provider string `trace:"auth.provider"`
			APIKeyID string `trace:"auth.api_key_id,omitempty"`
			UserID   string `trace:"auth.user_id,omitempty"`
		}{
			Status:   "no_active_provider_found",
			Provider: string(provider),
			APIKeyID: data.ID.Hex(),
			UserID:   data.UserID.Hex(),
		})
		end(err)
		return nil, cErr.UnauthorizedApiKey("No active provider access found")
	}

	if providerAccess.ExpireTime != nil && providerAccess.ExpireTime.Before(now) {
		err := cErr.UnauthorizedApiKey("Provider access has expired")
		updateErr := s.UpdateProviderStatus(ctx, data.ID, providerAccess.Provider, string(core.StatusExpired))
		if updateErr != nil {
			s.trace.ApplyTraceAttributes(span, struct {
				Status   string `trace:"auth.status"`
				Provider string `trace:"auth.provider"`
				Error    string `trace:"error"`
			}{
				Status:   "expire_mark_failed",
				Provider: string(providerAccess.Provider),
				Error:    updateErr.Error(),
			})
			end(updateErr)
			return nil, updateErr
		}

		s.trace.ApplyTraceAttributes(span, struct {
			Status   string `trace:"auth.status"`
			Provider string `trace:"auth.provider"`
		}{
			Status:   "provider_access_expired",
			Provider: string(providerAccess.Provider),
		})
		end(err)
		return nil, err
	}

	// 成功：畫點
	s.trace.ApplyTraceAttributes(span, struct {
		Status       string `trace:"auth.status"`
		Provider     string `trace:"auth.provider"`
		ProviderStat string `trace:"auth.provider_status,omitempty"`
	}{
		Status:       "success",
		Provider:     string(providerAccess.Provider),
		ProviderStat: string(providerAccess.Status),
	})

	return providerAccess, nil
}

// GetByID 依 API Key ID 取得單一 API Key 的回應 DTO。
func (s *UserAPIKeyService) GetByID(ctx context.Context, id primitive.ObjectID) (*dto.UserAPIKeyResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	key, err := s.userApiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, cErr.DatabaseError("mongodb get api key failed")
	}
	return modelToUserAPIKeyResponseDto(key), nil
}

// ListByUserID 取得指定使用者的 API Key 清單（回應 DTO）。
func (s *UserAPIKeyService) ListByUserID(ctx context.Context, userID primitive.ObjectID) ([]*dto.UserAPIKeyResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	keys, err := s.userApiKeyRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, cErr.DatabaseError("mongodb list api keys failed")
	}
	resp := make([]*dto.UserAPIKeyResponseDto, len(keys))
	for i, k := range keys {
		resp[i] = modelToUserAPIKeyResponseDto(k)
	}
	return resp, nil
}

// DeleteByID 刪除指定 API Key。
func (s *UserAPIKeyService) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.DeleteByID(ctx, id); err != nil {
		return cErr.DatabaseError("mongodb delete api key failed")
	}
	return nil
}

// DeleteAllByUserID 刪除使用者底下的所有 API Key。
func (s *UserAPIKeyService) DeleteAllByUserID(ctx context.Context, userID primitive.ObjectID) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.DeleteAllByUserID(ctx, userID); err != nil {
		return cErr.DatabaseError("mongodb delete user api keys failed")
	}
	return nil
}

// UpdateKeyName 更新 API Key 的名稱。
func (s *UserAPIKeyService) UpdateKeyName(ctx context.Context, id primitive.ObjectID, keyName string) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateKeyName(ctx, id, keyName); err != nil {
		return cErr.DatabaseError("mongodb update key name failed")
	}
	return nil
}

// UpdateKeyValue 更新 API Key 的值。
func (s *UserAPIKeyService) UpdateKeyValue(ctx context.Context, id primitive.ObjectID, keyValue string) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateKeyValue(ctx, id, keyValue); err != nil {
		return cErr.DatabaseError("mongodb update key value failed")
	}
	return nil
}

// UpdateProviderAccessAll 以全量覆寫方式更新 provider_access 欄位。
func (s *UserAPIKeyService) UpdateProviderAccessAll(ctx context.Context, id primitive.ObjectID, access []dto.ProviderAccessDto) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	models := providerAccessDtosToModels(access)
	if err := s.userApiKeyRepo.UpdateProviderAccessAll(ctx, id, models); err != nil {
		return cErr.DatabaseError("mongodb update providerAccess failed")
	}
	return nil
}

// UpdateProviderStatus 更新指定 provider 的狀態。
func (s *UserAPIKeyService) UpdateProviderStatus(ctx context.Context, id primitive.ObjectID, provider core.ProviderName, status string) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateProviderStatus(ctx, id, provider, status); err != nil {
		return cErr.DatabaseError("mongodb update key provider status failed")
	}
	return nil
}

// UpdateProviderLimitCount 更新指定 provider 的限額次數。
func (s *UserAPIKeyService) UpdateProviderLimitCount(ctx context.Context, id primitive.ObjectID, provider core.ProviderName, limitCount int) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateProviderLimitCount(ctx, id, provider, limitCount); err != nil {
		return cErr.DatabaseError("mongodb update limitCount failed")
	}
	return nil
}

// UpdateProviderUsedCount 設定（覆寫）指定 provider 的已用次數。
func (s *UserAPIKeyService) UpdateProviderUsedCount(ctx context.Context, id primitive.ObjectID, provider core.ProviderName, usedCount int) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateProviderUsedCount(ctx, id, provider, usedCount); err != nil {
		return cErr.DatabaseError("mongodb update usedCount failed")
	}
	return nil
}

// UpdateProviderLastResetAt 更新指定 provider 的上次重置時間。
func (s *UserAPIKeyService) UpdateProviderLastResetAt(ctx context.Context, id primitive.ObjectID, provider core.ProviderName, t time.Time) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateProviderLastResetAt(ctx, id, provider, t); err != nil {
		return cErr.DatabaseError("mongodb update key lastResetAt failed")
	}
	return nil
}
func (s *UserAPIKeyService) UpdateProviderExpireTime(ctx context.Context, id primitive.ObjectID, t time.Time) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateExpireTime(ctx, id, t); err != nil {
		return cErr.DatabaseError("mongodb update key expireTime failed")
	}

	return nil
}

// UpdateProviderFields 以 map 方式局部更新指定 provider 欄位。
func (s *UserAPIKeyService) UpdateProviderFields(ctx context.Context, id primitive.ObjectID, provider core.ProviderName, fields map[string]interface{}) error {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	if err := s.userApiKeyRepo.UpdateProviderFields(ctx, id, provider, fields); err != nil {
		return cErr.DatabaseError("mongodb partial update provider fields failed")
	}
	return nil
}

// List 依條件查詢 API Key 清單（回應 DTO）。
// 支援 user_id / provider / status / key_name 與建立時間區間。
func (s *UserAPIKeyService) List(ctx context.Context, query core.UserAPIKeyQuery) ([]*dto.UserAPIKeyResponseDto, error) {
	ctx, _, end := s.trace.WithSpan(ctx)
	defer end(nil)

	filter := bson.M{}
	if !query.UserID.IsZero() {
		filter["userID"] = query.UserID
	}
	if query.Provider != "" {
		filter["providerAccess.provider"] = query.Provider
	}
	if query.Status != "" {
		filter["providerAccess.status"] = query.Status
	}
	if query.KeyName != "" {
		filter["keyName"] = query.KeyName
	}
	if query.CreatedFrom != nil || query.CreatedTo != nil {
		created := bson.M{}
		if query.CreatedFrom != nil {
			created["$gte"] = *query.CreatedFrom
		}
		if query.CreatedTo != nil {
			created["$lte"] = *query.CreatedTo
		}
		filter["createdAt"] = created
	}

	keys, err := s.userApiKeyRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	resp := make([]*dto.UserAPIKeyResponseDto, len(keys))
	for i, k := range keys {
		resp[i] = modelToUserAPIKeyResponseDto(k)
	}
	return resp, nil
}

// Consume 消耗一次指定 provider 的限流配額（Redis + MongoDB 同步）。
// 回傳：remaining（剩餘次數），或錯誤。
func (s *UserAPIKeyService) Consume(
	ctx context.Context,
	apiKeyID string,
	providerAccess *model.ProviderAccess,
) (int, error) {
	ctx, span, end := s.trace.WithSpan(ctx)
	traceID := span.SpanContext().TraceID()
	spanID := span.SpanContext().SpanID()

	logError := func(err error) {
		end(err)
		s.logger.Warn(err.Error(),
			zap.String("spanId", fmt.Sprintf("%x", spanID[:])),
			zap.String("traceId", fmt.Sprintf("%x", traceID[:])),
		)
	}

	// 預設成功路徑結束 span
	defer end(nil)

	// 參數檢核 — 使用自家錯誤
	if providerAccess == nil {
		err := cErr.BadRequestParams("providerAccess is nil")
		logError(err)
		return 0, err
	}
	if providerAccess.LimitPeriod == nil {
		// 無限流量就直接返回（維持你的原本語意）
		return 0, nil
	}
	if providerAccess.LimitCount == nil || *providerAccess.LimitCount <= 0 {
		return 0, nil
	}
	if providerAccess.Provider == "" {
		err := cErr.DatabaseError("provider is required for redis rate limiting")
		logError(err)
		return 0, err
	}

	// Rate limiter
	window := periodToDuration(*providerAccess.LimitPeriod)
	remaining, _, rlErr := s.rateLimiterRepo.Consume(
		ctx,
		apiKeyID,
		providerAccess.Provider,
		*providerAccess.LimitPeriod,
		window,
		*providerAccess.LimitCount,
	)
	if rlErr != nil {
		err := cErr.RateLimiterUnavailable("rate limiter consume failed: " + rlErr.Error())
		logError(err)
		return 0, err
	}

	span.SetAttributes(
		attribute.String("api_key.id", apiKeyID),
		attribute.String("provider", string(providerAccess.Provider)),
		attribute.Int("remaining", remaining),
	)

	// 累計 MongoDB 使用次數
	objectID, oidErr := primitive.ObjectIDFromHex(apiKeyID)
	if oidErr != nil {
		err := cErr.BadRequestParams("invalid api key ID: not a valid ObjectID")
		logError(err)
		return 0, err
	}

	if incrErr := s.userApiKeyRepo.IncrProviderUsedCount(ctx, objectID, providerAccess.Provider, 1); incrErr != nil {
		err := cErr.DatabaseError(incrErr.Error())
		return 0, err
	}

	return remaining, nil
}
func getActiveProvider(accessList []model.ProviderAccess, provider core.ProviderName) (*model.ProviderAccess, error) {
	for _, acc := range accessList {
		if acc.Provider == provider && acc.Status == core.StatusActive {
			return &acc, nil
		}
	}
	return nil, cErr.UnauthorizedApiKey("no active provider key found")
}

// ProviderAccess DTO to model
func providerAccessDtosToModels(dto []dto.ProviderAccessDto) []model.ProviderAccess {
	result := make([]model.ProviderAccess, len(dto))
	for i, providerAccess := range dto {
		lastResetAt := providerAccess.LastResetAt
		if lastResetAt == nil && providerAccess.LimitPeriod != nil {
			now := time.Now().UTC()
			lastResetAt = &now
		}
		result[i] = model.ProviderAccess{
			Provider:    providerAccess.Provider,
			ProviderKey: providerAccess.ProviderKey,
			Status:      providerAccess.Status,
			LimitPeriod: providerAccess.LimitPeriod,
			LimitCount:  providerAccess.LimitCount,
			UsedCount:   providerAccess.UsedCount,
			LastResetAt: lastResetAt,
			ApiScopes:   providerAccess.ApiScopes,
			LastSeen:    providerAccess.LastSeen,
			ExpireTime:  providerAccess.ExpireTime,
		}
	}
	return result
}

// ProviderAccess model to DTO
func providerAccessModelsToDtos(model []model.ProviderAccess) []dto.ProviderAccessDto {
	result := make([]dto.ProviderAccessDto, len(model))
	for i, providerAccess := range model {
		result[i] = dto.ProviderAccessDto{
			Provider:    providerAccess.Provider,
			ProviderKey: maskKey(providerAccess.ProviderKey),
			Status:      providerAccess.Status,
			LimitPeriod: providerAccess.LimitPeriod,
			LimitCount:  providerAccess.LimitCount,
			UsedCount:   providerAccess.UsedCount,
			LastResetAt: providerAccess.LastResetAt,
			ApiScopes:   providerAccess.ApiScopes,
			LastSeen:    providerAccess.LastSeen,
			ExpireTime:  providerAccess.ExpireTime,
		}
	}
	return result
}

// UserAPIKey model to response DTO
func modelToUserAPIKeyResponseDto(m *model.UserAPIKey) *dto.UserAPIKeyResponseDto {
	return &dto.UserAPIKeyResponseDto{
		ID:             m.ID.Hex(),
		UserID:         m.UserID.Hex(),
		KeyName:        m.KeyName,
		KeyValue:       maskKey(m.KeyValue),
		ProviderAccess: providerAccessModelsToDtos(m.ProviderAccess),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// maskKey 遮蔽 API Key（顯示前4後4）。
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// periodToDuration 限流期間轉秒數。
func periodToDuration(period core.LimitPeriod) int64 {
	switch period {
	case core.LimitPeriodDaily:
		return 24 * 60 * 60
	case core.LimitPeriodWeekly:
		return 7 * 24 * 60 * 60
	case core.LimitPeriodMonthly:
		return 30 * 24 * 60 * 60
	case core.LimitPeriodYearly:
		return 365 * 24 * 60 * 60
	case core.LimitPeriodNone:
		return 0
	default:
		return 24 * 60 * 60
	}
}
