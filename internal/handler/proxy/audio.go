package proxy

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"interchange/config"
	"interchange/internal/core"
	fluentdModel "interchange/internal/database/fluentd/model"
	"interchange/internal/database/fluentd/repository"
	"interchange/internal/database/mongodb/model"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/service/audio"
	"interchange/internal/telemetry"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type AudioHandler struct {
	trace             *telemetry.Trace
	registry          *service.Registry
	logger            *zap.Logger
	config            *config.Configuration
	userAPIKeyService *service.UserAPIKeyService
	logRepository     *repository.LogRepository
}

func NewAudioHandler(
	trace *telemetry.Trace,
	registry *service.Registry,
	logger *zap.Logger,
	config *config.Configuration,
	userAPIKeyService *service.UserAPIKeyService,
	logRepository *repository.LogRepository,
) *AudioHandler {
	return &AudioHandler{
		trace:             trace,
		registry:          registry,
		logger:            logger,
		config:            config,
		userAPIKeyService: userAPIKeyService,
		logRepository:     logRepository,
	}
}

// AudioSpeech 語音合成（Text-to-Speech）
// @Summary 語音合成（Text-to-Speech）
// @Description 將文字透過指定 provider 的語音模型轉為音訊檔，回傳 `audio/*` 位元流。
// @Tags Proxy-Audio
// @Accept json
// @Accept multipart/form-data
// @Produce audio/*
// @Produce application/octet-stream
// @Param version  path string true  "API 版本"     Enums(v1)
// @Param provider path string true  "提供者"       Enums(openai)
// @Param payload  body audio.AudioSpeechRequestBody true "語音合成請求內容"
// @Security ApiKeyAuth
// @Success 200 {object} any "音訊位元流（Content-Type 依實際模型回傳，例如 audio/mpeg、audio/wav）"
// @Failure 400 {object} cErr.Error "Bad Request"
// @Failure 401 {object} cErr.Error "Unauthorized"
// @Failure 403 {object} cErr.Error "Forbidden"
// @Failure 404 {object} cErr.Error "Not Found"
// @Failure 429 {object} cErr.Error "Too Many Requests"
// @Failure 500 {object} cErr.Error "Internal Server Error"
// @Router /proxy/{version}/{provider}/audio/speech [post]
func (handler *AudioHandler) AudioSpeech(c *gin.Context) {
	ctx, span, end := handler.trace.WithSpan(c)
	var cause error
	defer end(nil)

	version := c.Param("version")
	provider := core.ProviderName(c.Param("provider"))

	span.SetAttributes(
		attribute.String("proxy.version", version),
		attribute.String("proxy.provider", string(provider)),
	)

	raw, ok := c.Get("apiKeyID")
	if !ok {
		cause = cErr.Unauthorized("missing or invalid API Key")
		end(cause)
		response.AbortWithError(c, cause)
		return
	}
	apiKeyID, ok := raw.(string)
	if !ok || apiKeyID == "" {
		cause = cErr.Unauthorized("invalid API Key format")
		end(cause)
		response.AbortWithError(c, cause)
		return
	}

	service, ok := handler.registry.GetAudio(provider)
	if !ok {
		cause = cErr.NotFound("audio provider not supported: " + string(provider))
		end(cause)
		response.AbortWithError(c, cause)
		return
	}
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
	var payload audio.AudioSpeechRequestBody
	if err := c.ShouldBind(&payload); err != nil {
		cause = cErr.BadRequestBody("invalid audio speech payload")
		end(err)
		response.AbortWithError(c, cause)
		return
	}

	switch version {
	case "v1":
		audioData, err := service.AudioSpeechV1(ctx, &payload, providerAccess.ProviderKey)
		if err != nil {
			cause = cErr.ExternalRequestError(err.Error())
			response.AbortWithError(c, cause)
			return
		}

		// 非阻塞記帳（失敗不影響主流程）
		if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, providerAccess); err != nil {
			cause = cErr.ExternalRequestError(err.Error())
			response.AbortWithError(c, cause)
		}

		ext := strings.TrimPrefix(audioData.ContentType, "audio/")
		if ext == "" {
			ext = "bin"
		}

		c.Header("Content-Type", audioData.ContentType)
		c.Header("Content-Disposition", `inline; filename="speech.`+ext+`"`)
		c.Data(http.StatusOK, audioData.ContentType, audioData.Data)

	default:
		cause = cErr.UnsupportedVersion("unsupported version")
		response.AbortWithError(c, cause)
	}
}

// AudioTranscriptions 語音轉錄（Speech-to-Text）
// @Summary 語音轉錄（Speech-to-Text）
// @Description 上傳音訊檔轉為文字（支援常見格式，如 OGG/MP3/WAV 等），回傳文字結果。
// @Tags Proxy-Audio
// @Accept multipart/form-data
// @Produce json
// @Param version  path string true  "API 版本"     Enums(v1)
// @Param provider path string true  "提供者"       Enums(openai, gemini)
// @Param payload  body audio.AudioTranscriptionRequestBody true "轉錄請求內容（含檔案與參數）"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=string} "轉錄文字內容"
// @Failure 400 {object} cErr.Error "Bad Request"
// @Failure 401 {object} cErr.Error "Unauthorized"
// @Failure 403 {object} cErr.Error "Forbidden"
// @Failure 404 {object} cErr.Error "Not Found"
// @Failure 429 {object} cErr.Error "Too Many Requests"
// @Failure 500 {object} cErr.Error "Internal Server Error"
// @Router /proxy/{version}/{provider}/audio/transcriptions [post]
func (handler *AudioHandler) AudioTranscriptions(c *gin.Context) {
	ctx, span, end := handler.trace.WithSpan(c)
	traceID := span.SpanContext().TraceID()
	defer end(nil)
	version := c.Param("version")
	userID := c.Query("userID")
	displayName := c.Query("displayName")
	provider := core.ProviderName(c.Param("provider"))
	span.SetAttributes(
		attribute.String("proxy.version", version),
		attribute.String("proxy.provider", string(provider)),
	)

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
	// 取得第三方專案名稱
	keyName, _ := c.Get("keyName")
	projectName, _ := keyName.(string)
	audioService, ok := handler.registry.GetAudio(provider)
	if !ok {
		err := cErr.Forbidden("provider not supported: " + string(provider))
		end(err)
		response.AbortWithError(c, err)
		return
	}
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
	var payload audio.AudioTranscriptionRequestBody
	if err := c.ShouldBind(&payload); err != nil {
		end(err)
		response.AbortWithError(c, cErr.BadRequestBody("invalid audio transcription payload"))
		return
	}

	switch version {
	case "v1":
		result, err := audioService.AudioTranscriptionsV1(ctx, &payload, providerAccess.ProviderKey)
		if err != nil {
			response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
			return
		}
		if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, providerAccess); err != nil {
			response.AbortWithError(c, err)
			return
		}
		var textToken,
			audioToken,
			inputToken,
			outputToken,
			tokensTotal int
		if result.Usage != nil {
			textToken = result.Usage.InputTokensDetails.TextTokens
			audioToken = result.Usage.InputTokensDetails.AudioTokens
			inputToken = result.Usage.InputTokens
			outputToken = result.Usage.OutputTokens
			tokensTotal = result.Usage.TotalTokens
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
			TokensPrompt:     0,
			TokensCompletion: 0,
			ImageToken:       0,
			TextToken:        textToken,
			AudioToken:       audioToken,
			InputToken:       inputToken,
			OutputToken:      outputToken,
			TokensTotal:      tokensTotal,
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

// AudioTranslations 語音翻譯（Speech Translation）
// @Summary 語音翻譯（Speech Translation）
// @Description 上傳音訊檔並翻譯為指定語言的文字，回傳翻譯結果。
// @Tags Proxy-Audio
// @Accept multipart/form-data
// @Produce json
// @Param version  path string true  "API 版本"     Enums(v1)
// @Param provider path string true  "提供者"       Enums(openai, gemini)
// @Param payload  body audio.AudioTranslationRequestBody true "翻譯請求內容（含檔案與參數）"
// @Security ApiKeyAuth
// @Success 200 {object} response.Response{data=string} "翻譯後文字內容"
// @Failure 400 {object} cErr.Error "Bad Request"
// @Failure 401 {object} cErr.Error "Unauthorized"
// @Failure 403 {object} cErr.Error "Forbidden"
// @Failure 404 {object} cErr.Error "Not Found"
// @Failure 429 {object} cErr.Error "Too Many Requests"
// @Failure 500 {object} cErr.Error "Internal Server Error"
// @Router /proxy/{version}/{provider}/audio/translations [post]
func (handler *AudioHandler) AudioTranslations(c *gin.Context) {
	ctx, span, end := handler.trace.WithSpan(c)
	defer end(nil)

	version := c.Param("version")
	provider := core.ProviderName(c.Param("provider"))
	span.SetAttributes(
		attribute.String("proxy.version", version),
		attribute.String("proxy.provider", string(provider)),
	)

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

	audioService, ok := handler.registry.GetAudio(provider)
	if !ok {
		err := cErr.Forbidden("provider not supported: " + string(provider))
		end(err)
		response.AbortWithError(c, err)
		return
	}
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
	var payload audio.AudioTranslationRequestBody
	if err := c.ShouldBind(&payload); err != nil {
		end(err)
		response.AbortWithError(c, cErr.BadRequestBody("invalid audio translation payload"))
		return
	}

	switch version {
	case "v1":
		result, err := audioService.AudioTranslationsV1(ctx, &payload, providerAccess.ProviderKey)
		if err != nil {
			response.AbortWithError(c, cErr.ExternalRequestError(err.Error()))
			return
		}
		if _, err := handler.userAPIKeyService.Consume(ctx, apiKeyID, providerAccess); err != nil {
			response.AbortWithError(c, err)
			return
		}
		response.Success(c, result)

	default:
		err := cErr.UnsupportedVersion("unsupported version")
		end(err)
		response.AbortWithError(c, err)
	}
}
