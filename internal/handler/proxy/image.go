package proxy

import (
	"context"
	"fmt"
	"interchange/config"
	"interchange/internal/core"
	fluentdModel "interchange/internal/database/fluentd/model"
	"interchange/internal/database/fluentd/repository"
	mongoModel "interchange/internal/database/mongodb/model"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/service/images"
	"interchange/internal/telemetry"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type LogArgs struct {
	traceID      string
	userID       string
	apiKeyID     string
	displayName  string
	projectName  string
	model        string
	endpoint     string
	provider     core.ProviderName
	responseBody *images.ImageGenerationResponse
}
type ImageHandler struct {
	trace             *telemetry.Trace
	registry          *service.Registry
	userAPIKeyService *service.UserAPIKeyService
	logger            *zap.Logger
	config            *config.Configuration
	logRepository     *repository.LogRepository
}

func NewImageHandler(
	trace *telemetry.Trace,
	registry *service.Registry,
	userAPIKeyService *service.UserAPIKeyService,
	logger *zap.Logger,
	config *config.Configuration,
	logRepository *repository.LogRepository,
) *ImageHandler {
	return &ImageHandler{
		trace:             trace,
		registry:          registry,
		userAPIKeyService: userAPIKeyService,
		logger:            logger,
		config:            config,
		logRepository:     logRepository,
	}
}

// ===== 共用前置：Trace / 版本 / Provider / API Key / 取 Service / ProviderAccess
func (h *ImageHandler) getContextPreps(c *gin.Context) (
	ctx context.Context,
	traceID string,
	apiKeyID string,
	userID string,
	version string,
	displayName string,
	projectName string,
	provider core.ProviderName,
	imgSvc images.Service,
	access *mongoModel.ProviderAccess,
	ok bool,
) {
	ctx, span, end := h.trace.WithSpan(c)

	// 把 end 放進 Context，供呼叫端 defer
	c.Set("_endSpan", end)
	version = c.Param("version")
	userID = c.Query("userID")
	displayName = c.Query("displayName")
	provider = core.ProviderName(c.Param("provider"))
	span.SetAttributes(
		attribute.String("proxy.version", version),
		attribute.String("proxy.provider", string(provider)),
	)
	tid := span.SpanContext().TraceID()
	traceID = fmt.Sprintf("%x", tid[:])

	// 1) 驗證 apiKeyID
	raw, exists := c.Get("apiKeyID")
	if !exists {
		response.AbortWithError(c, cErr.Unauthorized("missing or invalid API Key"))
		return
	}
	apiKeyID, _ = raw.(string)
	if apiKeyID == "" {
		response.AbortWithError(c, cErr.Unauthorized("invalid API Key format"))
		return
	}
	span.SetAttributes(attribute.String("auth.api_key_id", apiKeyID))

	// 取得第三方專案名稱
	keyName, _ := c.Get("keyName")
	projectName, _ = keyName.(string)

	// 2) 取得 images service
	svc, ok2 := h.registry.GetImages(provider)
	if !ok2 {
		response.AbortWithError(c, cErr.Forbidden("images provider not supported: "+string(provider)))
		return
	}
	imgSvc = svc

	// 取得啟用中的 provider access（從 middleware 放進 gin.Context）
	raw, exist := c.Get("providerAccess")
	if !exist {
		err := cErr.UnauthorizedApiKey("missing provider access data")
		end(err)
		response.AbortWithError(c, err)
		return
	}
	providerAccess, ok := raw.(*mongoModel.ProviderAccess)
	if !ok {
		err := cErr.InternalServer("invalid provider access data")
		end(err)
		response.AbortWithError(c, err)
		return
	}
	access = providerAccess
	ok = true
	return
}

// ===== 具體處理們：payload 綁定 + 呼叫對應 service 方法 =====

// @Summary 處理圖片生成請求
// @Tags Proxy-Image
// @Accept json
// @Produce json
// @Param version path string true "API 版本（例如 v1、v2）"
// @Param provider path string true "模型提供者（例如 openai、azure、gemini）"
// @Param payload body images.ImageGenerationRequestBody true "圖片生成請求內容"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=images.ImageGenerationResponse}
// @Router /proxy/{version}/{provider}/images/generations [post]
func (handler *ImageHandler) ImagesGenerations(c *gin.Context) {

	ctx,
		traceID,
		apiKeyID,
		userID,
		version,
		displayName,
		projectName,
		provider,
		imgSvc,
		access,
		ok := handler.getContextPreps(c)
	if !ok {
		return
	}
	endAny, _ := c.Get("_endSpan")
	defer endAny.(func(error))(nil)

	if version != "v1" {
		response.AbortWithError(c, cErr.UnsupportedVersion("unsupported version"))
		return
	}

	var payload images.ImageGenerationRequestBody
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.AbortWithError(c, cErr.BadRequestBody("invalid image generation payload"))
		return
	}

	result, err := imgSvc.GenerateV1(ctx, &payload, access.ProviderKey)
	if err != nil {
		response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
		return
	}
	if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, access); err != nil {
		response.AbortWithError(c, err)
		return
	}

	handler.logImageUsage(ctx, LogArgs{
		displayName:  displayName,
		projectName:  projectName,
		traceID:      traceID,
		userID:       userID,
		apiKeyID:     apiKeyID,
		provider:     provider,
		model:        string(payload.Model),
		endpoint:     c.Request.URL.Path,
		responseBody: result,
	})
	response.Success(c, result)
}

// @Summary 處理圖片變體請求
// @Tags Proxy-Image
// @Accept json
// @Produce json
// @Param version path string true "API 版本（例如 v1、v2）"
// @Param provider path string true "模型提供者（例如 openai、azure、gemini）"
// @Param payload body images.ImageVariantRequestBody true "圖片變體請求內容"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=images.ImageGenerationResponse}
// @Router /proxy/{version}/{provider}/images/variations [post]
func (handler *ImageHandler) ImagesVariations(c *gin.Context) {
	ctx,
		traceID,
		apiKeyID,
		userID,
		version,
		displayName,
		projectName,
		provider,
		imgSvc,
		access,
		ok := handler.getContextPreps(c)
	if !ok {
		return
	}
	endAny, _ := c.Get("_endSpan")
	defer endAny.(func(error))(nil)

	if version != "v1" {
		response.AbortWithError(c, cErr.UnsupportedVersion("unsupported version"))
		return
	}

	var payload images.ImageVariantRequestBody // multipart/form-data
	if err := c.ShouldBind(&payload); err != nil {
		response.AbortWithError(c, cErr.BadRequestBody("invalid image variation payload"))
		return
	}
	result, err := imgSvc.VariationV1(ctx, &payload, access.ProviderKey)
	if err != nil {
		response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
		return
	}
	if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, access); err != nil {
		response.AbortWithError(c, err)
		return
	}

	handler.logImageUsage(ctx, LogArgs{
		displayName:  displayName,
		projectName:  projectName,
		traceID:      traceID,
		userID:       userID,
		apiKeyID:     apiKeyID,
		provider:     provider,
		model:        string(payload.Model),
		endpoint:     c.Request.URL.Path,
		responseBody: result,
	})
	response.Success(c, result)
}

// @Summary 處理圖片編輯請求
// @Tags Proxy-Image
// @Accept json
// @Produce json
// @Param version path string true "API 版本（例如 v1、v2）"
// @Param provider path string true "模型提供者（例如 openai、azure、gemini）"
// @Param payload body images.ImageEditRequestBody true "圖片編輯請求內容"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=images.ImageGenerationResponse}
// @Router /proxy/{version}/{provider}/images/edits [post]
func (handler *ImageHandler) ImagesEdits(c *gin.Context) {
	ctx,
		traceID,
		apiKeyID,
		userID,
		version,
		displayName,
		projectName,
		provider,
		imgSvc,
		access,
		ok := handler.getContextPreps(c)
	if !ok {
		return
	}
	endAny, _ := c.Get("_endSpan")
	defer endAny.(func(error))(nil)

	if version != "v1" {
		response.AbortWithError(c, cErr.UnsupportedVersion("unsupported version"))
		return
	}

	var payload images.ImageEditRequestBody // multipart/form-data
	if err := c.ShouldBind(&payload); err != nil {
		response.AbortWithError(c, cErr.BadRequestBody("invalid image edit payload"))
		return
	}

	result, err := imgSvc.EditV1(ctx, &payload, access.ProviderKey)
	if err != nil {
		response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
		return
	}
	if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, access); err != nil {
		response.AbortWithError(c, err)
		return
	}

	handler.logImageUsage(ctx, LogArgs{
		displayName:  displayName,
		projectName:  projectName,
		traceID:      traceID,
		userID:       userID,
		apiKeyID:     apiKeyID,
		provider:     provider,
		model:        string(payload.Model),
		endpoint:     c.Request.URL.Path,
		responseBody: result,
	})
	response.Success(c, result)
}

// ===== 共用記錄：把 token / req / resp 與基礎欄位打包，統一寫 Fluentd
func (h *ImageHandler) logImageUsage(ctx context.Context, args LogArgs) {
	var (
		textToken, imageToken, inputToken, outputToken, total int
	)
	if args.responseBody != nil && args.responseBody.Usage != nil {
		textToken = args.responseBody.Usage.InputTokensDetails.TextTokens
		imageToken = args.responseBody.Usage.InputTokensDetails.ImageTokens
		inputToken = args.responseBody.Usage.InputTokens
		outputToken = args.responseBody.Usage.OutputTokens
		total = args.responseBody.Usage.TotalTokens
	}

	log := fluentdModel.AIUsageLog{
		RequestID:        args.traceID,
		ExternalID:       args.userID,
		DisplayName:      args.displayName,
		ProjectName:      args.projectName,
		Provider:         string(args.provider),
		Model:            args.model,
		Endpoint:         args.endpoint,
		TokensPrompt:     0,
		TokensCompletion: 0,
		AudioToken:       0,
		TextToken:        textToken,
		ImageToken:       imageToken,
		InputToken:       inputToken,
		OutputToken:      outputToken,
		TokensTotal:      total,
		Version:          h.config.App.Version,
		LoggedAt:         time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
	}
	h.logRepository.LogUsage(ctx, log)
}
