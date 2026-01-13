package proxy

import (
	"interchange/internal/core"
	"interchange/internal/database/mongodb/model"
	"interchange/internal/database/redis/repository"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	models "interchange/internal/service/models"
	"interchange/internal/telemetry"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type ModelsHandler struct {
	trace                 *telemetry.Trace
	registry              *service.Registry
	rateLimiterRepository *repository.RateLimiterRepository
	userAPIKeyService     *service.UserAPIKeyService
	logger                *zap.Logger
}

func NewModelsHandler(
	trace *telemetry.Trace,
	registry *service.Registry,
	rateLimiterRepository *repository.RateLimiterRepository,
	userAPIKeyService *service.UserAPIKeyService,
	logger *zap.Logger,
) *ModelsHandler {
	return &ModelsHandler{
		trace:                 trace,
		registry:              registry,
		rateLimiterRepository: rateLimiterRepository,
		userAPIKeyService:     userAPIKeyService,
		logger:                logger,
	}
}

// ListModels 列出可用模型（OpenAI 形狀）
// @Summary 模型列表
// @Description 依 provider 代理列出模型清單（回應維持 OpenAI 形狀）
// @Tags Proxy-Models
// @Produce json
// @Param version path string true "API 版本（例如 v1、v2）"
// @Param provider path string true "模型提供者（例如 openai、gemini）"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=models.ListResponse}
// @Failure 400 {object} cErr.Error "Bad Request"
// @Failure 403 {object} cErr.Error "Forbidden"
// @Failure 404 {object} cErr.Error "Not Found"
// @Failure 500 {object} cErr.Error "Internal Server Error"
// @Router /proxy/{version}/{provider}/models [get]
func (handler *ModelsHandler) ListModels(c *gin.Context) {
	ctx, span, end := handler.trace.WithSpan(c)
	defer end(nil)

	version := c.Param("version")
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
	span.SetAttributes(attribute.String("auth.api_key_id", apiKeyID))

	// 取得對應 provider 的 models service
	modelsService, ok := handler.registry.GetModels(provider)
	if !ok {
		err := cErr.Forbidden("provider not supported: " + string(provider))
		end(err)
		response.AbortWithError(c, err)
		return
	}

	// 取得啟用中的 provider access（從 middleware 放進 gin.Context）
	raw, exist := c.Get("providerAccess")
	if !exist {
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

	// 依版本路由（目前支援 v1）
	switch version {
	case "v1":
		var result *models.ListResponse
		result, err := modelsService.List(ctx, providerAccess.ProviderKey)
		if err != nil {
			end(err)
			response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
			return
		}

		response.Success(c, result)

	default:
		err := cErr.UnsupportedVersion("unsupported version")
		end(err)
		response.AbortWithError(c, err)
	}
}
