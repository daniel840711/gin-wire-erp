package middleware

import (
	"interchange/internal/core"
	"interchange/internal/database/mongodb/model"
	"interchange/internal/database/redis/repository"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/telemetry"
	"interchange/utils/validate"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/attribute"
)

type RateLimit struct {
	trace                 *telemetry.Trace
	rateLimiterRepository *repository.RateLimiterRepository
	userAPIKeyService     *service.UserAPIKeyService
}

func NewRateLimit(
	trace *telemetry.Trace,
	rateLimiterRepository *repository.RateLimiterRepository,
	userAPIKeyService *service.UserAPIKeyService,
) *RateLimit {
	return &RateLimit{
		trace:                 trace,
		rateLimiterRepository: rateLimiterRepository,
		userAPIKeyService:     userAPIKeyService,
	}
}

func (middleware *RateLimit) Guard() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span, end := middleware.trace.WithSpan(c.Request.Context(), string(core.SpanRateLimitMiddleware))
		// 從 APIKey middleware 放進 gin.Context 的資訊
		rawID, ok := c.Get("apiKeyID")
		if !ok {
			err := cErr.Unauthorized("missing or invalid API Key")
			end(err)
			response.AbortWithError(c, err)
			return
		}
		apiKeyID, _ := rawID.(string)
		apiKeyObjectID, err := primitive.ObjectIDFromHex(apiKeyID)
		if err != nil {
			response.AbortWithError(c, cErr.UnauthorizedApiKey("invalid api key: key ID is not a valid ObjectID"))
			end(err)
			return
		}
		provider := c.Param("provider")
		if provider == "" || !validate.IsValidProviderName(provider) {
			span.SetAttributes(
				attribute.String("api_key.id", apiKeyID),
				attribute.String("provider", provider),
				attribute.String("status", "invalid_provider_in_path"),
			)
			err := cErr.BadRequestParams("Invalid provider in path")
			response.AbortWithError(c, err)
			end(err)
			return
		}

		// 從 context 取出已驗證且啟用中的 ProviderAccess
		raw, exist := c.Get("providerAccess")
		if !exist {
			err := cErr.UnauthorizedApiKey("missing provider access data")
			response.AbortWithError(c, err)
			end(err)
			return
		}
		providerAccess, ok := raw.(*model.ProviderAccess)
		if !ok {
			err := cErr.InternalServer("invalid provider access data")
			response.AbortWithError(c, err)
			end(err)
			return
		}
		// 若未設定 period 或 limit，next()
		if providerAccess.LimitPeriod == nil || providerAccess.LimitCount == nil || *providerAccess.LimitCount <= 0 {
			end(nil)
			c.Next()
			return
		}

		// 讀取目前剩餘與 TTL
		remaining, ttlSec, e := middleware.rateLimiterRepository.GetCurrent(
			ctx,
			apiKeyID,
			core.ProviderName(provider),
			*providerAccess.LimitPeriod,
			*providerAccess.LimitCount,
		)
		if e != nil {
			end(nil)
			// 風險控制：讀取錯誤不阻斷主流程（可改為嚴格阻斷）
			c.Next()
			return
		}

		// 尚未初始化 key（新視窗第一次） => remaining=0, ttl=0
		uninitialized := (remaining == 0 && ttlSec == 0)
		effectiveRemaining := remaining
		if uninitialized {
			effectiveRemaining = *providerAccess.LimitCount // 預視剩餘（實際扣款在尾端 Consume）
			// 更新最後重置時間
			err := middleware.userAPIKeyService.UpdateProviderLastResetAt(
				ctx,
				apiKeyObjectID,
				core.ProviderName(provider),
				time.Now().UTC(),
			)
			if err != nil {
				response.AbortWithError(c, err)
				end(err)
				return
			}
			// 重製使用量
			err = middleware.userAPIKeyService.UpdateProviderUsedCount(
				ctx,
				apiKeyObjectID,
				core.ProviderName(provider),
				0, // 重設為 0
			)
			if err != nil {
				response.AbortWithError(c, err)
				end(err)
				return
			}
		}

		// 寫入回應標頭，方便呼叫端與排錯
		c.Header("X-RateLimit-Limit", strconv.Itoa(*providerAccess.LimitCount))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(effectiveRemaining))
		if ttlSec > 0 {
			c.Header("X-RateLimit-Reset", strconv.FormatInt(ttlSec, 10))
		}

		// 超過額度：remaining<=0 且仍在視窗內（ttlSec>0）
		block := (effectiveRemaining <= 0 && ttlSec > 0)

		// Trace 記錄
		middleware.trace.ApplyTraceAttributes(span, core.TraceRateLimitMiddlewareMeta{
			APIKeyID:      apiKeyID,
			Provider:      string(provider),
			Period:        string(*providerAccess.LimitPeriod),
			ConfigLimit:   *providerAccess.LimitCount,
			Remaining:     effectiveRemaining,
			TTLSeconds:    ttlSec,
			Blocked:       block,
			Uninitialized: uninitialized,
		})

		if block {
			if ttlSec > 0 {
				c.Header("Retry-After", strconv.FormatInt(ttlSec, 10))
			}
			err := cErr.RateLimitExceeded("rate limit exceeded")
			response.AbortWithError(c, err)
			end(err)
			return
		}
		end(nil)
		c.Next()
	}
}
