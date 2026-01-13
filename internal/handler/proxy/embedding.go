package proxy

import (
	"fmt"
	"interchange/config"
	"interchange/internal/core"
	fluentdModel "interchange/internal/database/fluentd/model"
	fluentd "interchange/internal/database/fluentd/repository"
	"interchange/internal/database/mongodb/model"

	"interchange/internal/database/redis/repository"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/service/embedding"
	"interchange/internal/telemetry"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type EmbeddingHandler struct {
	trace                 *telemetry.Trace
	registry              *service.Registry
	rateLimiterRepository *repository.RateLimiterRepository
	userAPIKeyService     *service.UserAPIKeyService
	logger                *zap.Logger
	config                *config.Configuration
	logRepository         *fluentd.LogRepository
}

func NewEmbeddingHandler(
	trace *telemetry.Trace,
	registry *service.Registry,
	rateLimiterRepository *repository.RateLimiterRepository,
	userAPIKeyService *service.UserAPIKeyService,
	logger *zap.Logger,
	config *config.Configuration,
	logRepository *fluentd.LogRepository,
) *EmbeddingHandler {
	return &EmbeddingHandler{
		trace:                 trace,
		registry:              registry,
		rateLimiterRepository: rateLimiterRepository,
		userAPIKeyService:     userAPIKeyService,
		logger:                logger,
		config:                config,
		logRepository:         logRepository,
	}
}

// GenerateEmbedding 處理嵌入生成請求
// @Summary 嵌入生成
// @Description 處理嵌入生成請求
// @Tags Proxy-Embedding
// @Accept json
// @Produce json
// @Param version path string true "API 版本（例如 v1、v2）"
// @Param provider path string true "模型提供者（例如 openai、azure、gemini）"
// @Param payload body embedding.EmbeddingRequestBody true "嵌入生成請求內容"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=embedding.EmbeddingResponse}
// @Failure 400 {object} cErr.Error "Bad Request"
// @Failure 404 {object} cErr.Error "Not Found"
// @Failure 500 {object} cErr.Error "Internal Server Error"
// @Router /proxy/{version}/{provider}/embeddings/generations [post]
func (handler *EmbeddingHandler) GenerateEmbedding(c *gin.Context) {
	version := c.Param("version")
	userID := c.Query("userID")
	displayName := c.Query("displayName")
	ctx, span, end := handler.trace.WithSpan(c)
	traceID := span.SpanContext().TraceID()
	defer end(nil)

	provider := core.ProviderName(c.Param("provider"))
	span.SetAttributes(
		attribute.String("proxy.version", version),
		attribute.String("proxy.provider", string(provider)),
	)

	// 從 middleware 設置的 context 中取得 apiKeyID
	raw, ok := c.Get("apiKeyID")
	if !ok {
		err := cErr.Unauthorized("missing or invalid API Key")
		end(err)
		response.AbortWithError(c, err)
		return
	}
	apiKeyID, ok := raw.(string)
	if !ok || apiKeyID == "" {
		err := cErr.Unauthorized("invalid API Key format")
		end(err)
		response.AbortWithError(c, err)
		return
	}
	// 取得第三方專案名稱
	keyName, _ := c.Get("keyName")
	projectName, _ := keyName.(string)

	span.SetAttributes(attribute.String("auth.api_key_id", apiKeyID))

	// 取得對應 provider 的 embedding service
	embService, ok := handler.registry.GetEmbedding(provider)
	if !ok {
		err := cErr.Forbidden("provider not supported: " + string(provider))
		end(err)
		response.AbortWithError(c, err)
		return
	}

	// 取得啟用中的 provider access（從 middleware 放進 gin.Context）
	raw, exists := c.Get("providerAccess")
	if !exists {
		err := cErr.UnauthorizedApiKey("missing provider access data")
		end(err)
		response.AbortWithError(c, err)
		return
	}
	providerAccess, ok := raw.(*model.ProviderAccess)
	if !ok {
		err := cErr.InternalServer("invalid provider access data")
		end(err)
		response.AbortWithError(c, err)
		return
	}
	// 綁定 payload
	var payload embedding.EmbeddingRequestBody
	if err := c.ShouldBindJSON(&payload); err != nil {
		end(err)
		response.AbortWithError(c, cErr.BadRequestBody("invalid embedding payload"))
		return
	}

	// 依版本路由（目前支援 v1）
	switch version {
	case "v1":
		result, err := embService.GenerateEmbedding(ctx, &payload, providerAccess.ProviderKey)
		if err != nil {
			response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
			return
		}
		// 同步記帳（失敗不影響主流程）
		if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, providerAccess); err != nil {
			response.AbortWithError(c, err)
			return
		}
		//fluentd 紀錄
		log := fluentdModel.AIUsageLog{
			RequestID:        fmt.Sprintf("%x", traceID[:]),
			ExternalID:       userID,
			DisplayName:      displayName,
			ProjectName:      projectName,
			Provider:         string(provider),
			Model:            payload.Model,
			Endpoint:         c.Request.URL.Path,
			TokensPrompt:     result.Usage.PromptTokens,
			TokensCompletion: 0,
			TextToken:        0,
			AudioToken:       0,
			ImageToken:       0,
			InputToken:       0,
			OutputToken:      0,
			TokensTotal:      result.Usage.TotalTokens,
			Version:          handler.config.App.Version,
			LoggedAt:         time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
		}
		handler.logRepository.LogUsage(ctx, log)
		response.Success(c, result)

	default:
		err := cErr.UnsupportedVersion("unsupported version")
		end(err)
		response.AbortWithError(c, err)
	}
}
