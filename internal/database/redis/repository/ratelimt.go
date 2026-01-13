package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"interchange/internal/core"
	client "interchange/internal/database/client"
	"interchange/internal/telemetry"

	"github.com/redis/go-redis/v9"
)

type RateLimiterRepository struct {
	trace  *telemetry.Trace
	client *redis.Client
}

func NewRateLimiterRepository(trace *telemetry.Trace, client *client.RedisClient) *RateLimiterRepository {
	return &RateLimiterRepository{trace: trace, client: client.Client()}
}

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// Consume 消耗一次配額；自動處理新週期初始化與剩餘 TTL。
// 回傳：remaining（剩餘次數）、ttlSec（剩餘秒數）、err（若超限為 ErrRateLimitExceeded）
func (repository *RateLimiterRepository) Consume(
	contextValue context.Context,
	apiKeyID string,
	provider core.ProviderName,
	period core.LimitPeriod,
	windowSeconds int64,
	limitCount int,
) (remainingCount int, timeToLiveSeconds int64, returnedError error) {

	contextValue, span, endSpan := repository.trace.WithSpan(contextValue)
	defer func() {
		endSpan(returnedError)
	}()

	traceMetadata := core.TraceRateLimitMeta{
		APIKeyID:  apiKeyID,
		Provider:  string(provider),
		Period:    string(period),
		Limit:     limitCount,
		WindowSec: windowSeconds,
		Op:        "consume",
	}
	repository.trace.ApplyTraceAttributes(span, traceMetadata)

	redisKey := repository.buildKey(apiKeyID, provider, period)
	expirationDuration := time.Duration(windowSeconds) * time.Second

	// 嘗試初始化：SETNX key value EX expiration
	wasSet, setError := repository.client.SetNX(
		contextValue,
		redisKey,
		limitCount-1, // 本次消耗一次，所以初始值 = 總額-1
		expirationDuration,
	).Result()
	if setError != nil {
		returnedError = setError
		return 0, 0, returnedError
	}
	if wasSet {
		// 初始化成功，代表這是第一次消耗
		remainingCount = limitCount - 1
		if remainingCount < 0 {
			remainingCount = 0
			returnedError = ErrRateLimitExceeded
		}
		timeToLiveSeconds = windowSeconds
		traceMetadata.Remaining, traceMetadata.TTL = remainingCount, timeToLiveSeconds
		repository.trace.ApplyTraceAttributes(span, traceMetadata)
		return remainingCount, timeToLiveSeconds, returnedError
	}

	// Key 已存在 → 執行 DECR 扣一次
	newValue, decrError := repository.client.Decr(contextValue, redisKey).Result()
	if decrError != nil {
		returnedError = decrError
		return 0, 0, returnedError
	}

	// 查 TTL
	ttlDuration, _ := repository.client.TTL(contextValue, redisKey).Result()
	if ttlDuration > 0 {
		timeToLiveSeconds = int64(ttlDuration.Seconds())
	}

	if newValue < 0 {
		remainingCount = 0
		traceMetadata.Remaining, traceMetadata.TTL = remainingCount, timeToLiveSeconds
		repository.trace.ApplyTraceAttributes(span, traceMetadata)
		returnedError = ErrRateLimitExceeded
		return remainingCount, timeToLiveSeconds, returnedError
	}

	remainingCount = int(newValue)
	traceMetadata.Remaining, traceMetadata.TTL = remainingCount, timeToLiveSeconds
	repository.trace.ApplyTraceAttributes(span, traceMetadata)
	return remainingCount, timeToLiveSeconds, nil
}

// GetCurrent 查詢目前「剩餘次數」與剩餘 TTL（秒）。若無紀錄回傳 0,0。
func (repository *RateLimiterRepository) GetCurrent(
	contextValue context.Context,
	apiKeyIdentifier string,
	provider core.ProviderName,
	period core.LimitPeriod,
	limitCount int, // ★ 新增這個參數，才能在 key 不存在時回正確的 remaining
) (remainingCount int, timeToLiveSeconds int64, returnedError error) {

	contextValue, span, endSpan := repository.trace.WithSpan(contextValue)
	defer func() { endSpan(returnedError) }()

	traceMetadata := core.TraceRateLimitMeta{
		APIKeyID: apiKeyIdentifier,
		Provider: string(provider),
		Period:   string(period),
		Op:       "get",
	}
	repository.trace.ApplyTraceAttributes(span, traceMetadata)

	redisKey := repository.buildKey(apiKeyIdentifier, provider, period)

	// 用 pipeline 併發 GET + TTL 減少往返
	pipeline := repository.client.Pipeline()
	getCommand := pipeline.Get(contextValue, redisKey)
	ttlCommand := pipeline.TTL(contextValue, redisKey)
	if _, execError := pipeline.Exec(contextValue); execError != nil && execError != redis.Nil {
		returnedError = execError
		return 0, 0, returnedError
	}

	value, getError := getCommand.Int()
	if getError == redis.Nil {
		// 尚未初始化：remaining = limitCount, ttl = 0
		remainingCount = limitCount
		timeToLiveSeconds = 0
		traceMetadata.Remaining, traceMetadata.TTL = remainingCount, timeToLiveSeconds
		repository.trace.ApplyTraceAttributes(span, traceMetadata)
		return remainingCount, timeToLiveSeconds, nil
	}
	if getError != nil {
		returnedError = getError
		return 0, 0, returnedError
	}

	ttlDuration := ttlCommand.Val()
	if ttlDuration > 0 {
		timeToLiveSeconds = int64(ttlDuration.Seconds())
	} else {
		timeToLiveSeconds = 0
	}

	remainingCount = value // value 就是剩餘（倒數語意）
	if remainingCount < 0 {
		remainingCount = 0
	}

	traceMetadata.Remaining, traceMetadata.TTL = remainingCount, timeToLiveSeconds
	repository.trace.ApplyTraceAttributes(span, traceMetadata)
	return remainingCount, timeToLiveSeconds, nil
}

// Reset 強制重置剩餘次數與 TTL（管理用）。
// limitCount == nil 則重設為 0；否則重設為 *limitCount。TTL 設為 windowSec 秒（0 代表永不過期）。
func (repository *RateLimiterRepository) Reset(
	contextValue context.Context,
	apiKeyIdentifier string,
	provider core.ProviderName,
	period core.LimitPeriod,
	windowSeconds int64,
	limitCount *int,
) (returnedError error) {

	contextValue, span, endSpan := repository.trace.WithSpan(contextValue)
	defer func() { endSpan(returnedError) }()

	valueToSet := 0
	if limitCount != nil {
		valueToSet = *limitCount
		if valueToSet < 0 {
			valueToSet = 0
		}
	}

	traceMetadata := core.TraceRateLimitMeta{
		APIKeyID:  apiKeyIdentifier,
		Provider:  string(provider),
		Period:    string(period),
		Limit:     valueToSet,
		WindowSec: windowSeconds,
		Remaining: valueToSet,
		Op:        "reset",
	}
	repository.trace.ApplyTraceAttributes(span, traceMetadata)

	redisKey := repository.buildKey(apiKeyIdentifier, provider, period)
	expiration := time.Duration(windowSeconds) * time.Second

	returnedError = repository.client.Set(contextValue, redisKey, valueToSet, expiration).Err()
	return returnedError
}

// Delete 刪除配額 key（徹底移除）
func (repository *RateLimiterRepository) Delete(
	contextValue context.Context,
	apiKeyIdentifier string,
	provider core.ProviderName,
	period core.LimitPeriod,
) (returnedError error) {

	contextValue, span, endSpan := repository.trace.WithSpan(contextValue)
	defer func() { endSpan(returnedError) }()

	traceMetadata := core.TraceRateLimitMeta{
		APIKeyID: apiKeyIdentifier,
		Provider: string(provider),
		Period:   string(period),
		Op:       "delete",
	}
	repository.trace.ApplyTraceAttributes(span, traceMetadata)

	redisKey := repository.buildKey(apiKeyIdentifier, provider, period)
	returnedError = repository.client.Del(contextValue, redisKey).Err()
	return returnedError
}

// buildKey 建構 RateLimiter 用的 Redis key
func (r *RateLimiterRepository) buildKey(apiKeyID string, provider core.ProviderName, period core.LimitPeriod) string {
	return fmt.Sprintf("%s:%s:%s:%s", core.RedisKeyServerName, apiKeyID, provider, period)
}
