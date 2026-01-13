package middleware

import (
	"fmt"
	"interchange/internal/core"
	"interchange/internal/database/mongodb/model"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/telemetry"
	"interchange/utils/validate"
	"path"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type APIKey struct {
	logger            *zap.Logger
	trace             *telemetry.Trace
	metric            *telemetry.Metric
	userAPIKeyService *service.UserAPIKeyService
}

func NewAPIKey(
	logger *zap.Logger,
	trace *telemetry.Trace,
	metric *telemetry.Metric,
	userAPIKeyService *service.UserAPIKeyService,
) *APIKey {
	return &APIKey{
		logger:            logger,
		trace:             trace,
		metric:            metric,
		userAPIKeyService: userAPIKeyService,
	}
}

func (middleware *APIKey) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span, end := middleware.trace.WithSpan(c.Request.Context(), string(core.SpanAPIKeyMiddleware))
		var cause error = nil
		apiKey, from := middleware.readPlatformKey(c)
		provider := c.Param("provider")
		meta := core.TraceAPIKeyMiddlewareMeta{
			Provider: provider,
			Where:    from,
			ClientIP: c.ClientIP(),
		}

		if provider == "" || !validate.IsValidProviderName(provider) {
			meta.Status = "invalid_provider_in_path"
			middleware.trace.ApplyTraceAttributes(span, meta)
			cause = cErr.BadRequestParams("Invalid provider in path")
			response.AbortWithError(c, cause)
			end(cause)
			return
		}

		if apiKey == "" {
			meta.Status = "missing_api_key"
			middleware.trace.ApplyTraceAttributes(span, meta)
			cause = cErr.UnauthorizedApiKey("Missing API Key")
			response.AbortWithError(c, cause)
			end(cause)
			return
		}

		// 驗證平台 key
		payload, err := middleware.userAPIKeyService.ValidateKey(ctx, apiKey)
		if err != nil {
			errStr := "invalid_api_key"
			meta.Status = errStr
			middleware.trace.ApplyTraceAttributes(span, meta)
			response.AbortWithError(c, cErr.UnauthorizedApiKey("Invalid API Key"))
			end(err)
			return
		}
		providerAccess, cause := middleware.userAPIKeyService.ValidateProviderAccess(ctx, payload, core.ProviderName(provider))
		if cause != nil {
			meta.Status = "no_active_provider_found"
			middleware.trace.ApplyTraceAttributes(span, meta)
			response.AbortWithError(c, cause)
			end(cause)
			return
		}
		reqScope := middleware.requiredScopeFromPath(c.Request.URL.Path)
		if !middleware.isScopeAllowed(providerAccess.ApiScopes, reqScope) {
			meta.Status = "forbidden_scope"
			cause = cErr.Forbidden("forbidden: api scope not allowed")
			middleware.trace.ApplyTraceAttributes(span, meta)
			response.AbortWithError(c, cause)
			end(cause)
			return
		}
		userID := payload.UserID.Hex()
		apiKeyID := payload.ID.Hex()

		meta.UserID = userID
		meta.APIKeyID = apiKeyID
		meta.KeyName = payload.KeyName
		meta.Scopes = middleware.flattenScopes(payload.ProviderAccess)
		meta.Status = "success"
		middleware.trace.ApplyTraceAttributes(span, meta)
		// 記錄授權成功的日誌
		traceID := span.SpanContext().TraceID()
		spanID := span.SpanContext().SpanID()
		middleware.logger.Info("[APIKey Authenticated]",
			zap.String("userID", userID),
			zap.String("apiKeyID", apiKeyID),
			zap.String("provider", provider),
			zap.String("from", from),
			zap.String("spanId", fmt.Sprintf("%x", spanID[:])),
			zap.String("traceId", fmt.Sprintf("%x", traceID[:])),
		)
		end(cause)

		// 設定給下游（ratelimit、handler 會用到）
		c.Set("userID", userID)
		c.Set("apiKeyID", apiKeyID)
		c.Set("keyName", payload.KeyName)
		c.Set("providerAccess", providerAccess)
		c.Next()
	}
}

func (middleware *APIKey) readPlatformKey(c *gin.Context) (key string, from string) {
	// 1) Authorization: Bearer <platform_key>
	if auth := strings.TrimSpace(c.GetHeader("Authorization")); auth != "" {
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			tok := strings.TrimSpace(auth[len("Bearer "):])
			return tok, "bearer"
		}
	}

	// 2) X-API-Key
	if x := strings.TrimSpace(c.GetHeader("X-API-Key")); x != "" {
		return x, "x-api-key"
	}
	return "", ""
}
func (middleware *APIKey) flattenScopes(pas []model.ProviderAccess) []string {
	if len(pas) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
	for _, pa := range pas {
		for _, s := range pa.ApiScopes {
			k := string(s)
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				out = append(out, k)
			}
		}
	}
	sort.Strings(out)
	return out
}

// 依實際 URL.Path 推導你的 ApiScope（你定義的就是「剩餘路徑」的片段）
func (middleware *APIKey) requiredScopeFromPath(urlPath string) core.ApiScope {
	// 標準化（去除多餘斜線）
	cleanPath := path.Clean(urlPath)

	// 先判斷 MCP 路由：/mcp-server/:version/:provider/*
	if strings.HasPrefix(cleanPath, "/mcp-server/") {
		// 你目前希望 MCP 任意 action 都走 /mcp-server/* 這個大 Scope
		return core.ApiScopeMCPServer
	}

	// 再判斷 Proxy 路由：/proxy/:version/:provider/<scope...>
	if strings.HasPrefix(cleanPath, "/proxy/") {
		// 移除 /proxy/{version}/{provider} 這三段前綴，取出剩餘 path
		// 做法：把 "/proxy/" 之後切三段
		// cleanPath 例：/proxy/v1/openai/chat/completions
		parts := strings.Split(cleanPath, "/")
		// parts[0]="" parts[1]="proxy" parts[2]="v1" parts[3]="openai" parts[4:]是我們要的 scope 片段
		if len(parts) >= 5 {

			sub := "/" + strings.Join(parts[4:], "/") // 變成 "/chat/completions" or "/images/generations"...
			switch sub {
			case string(core.ApiScopeChatCompletions),
				string(core.ApiScopeImagesGenerations),
				string(core.ApiScopeImagesVariations),
				string(core.ApiScopeImagesEdits),
				string(core.ApiScopeAudioTranscriptions),
				string(core.ApiScopeEmbeddingsGenerations),
				string(core.ApiScopeGetModels):
				return core.ApiScope(sub) // 精準對應
			default:
				// 你也可以選擇 default 改成 ApiScopeAll 或直接視為無權限
				return core.ApiScope(sub)
			}
		}
	}

	// 其他路徑（理論上不會進來到這個 middleware），保守：要求 All
	return core.ApiScopeAll
}

// 依路徑判斷所需 Scope，並與 providerAccess.ApiScopes 比對
// 讓 allowed 可以包含： "*", "/mcp-server/*", "/images/*", 以及具體端點 "/chat/completions"
func (middleware *APIKey) isScopeAllowed(allowed []core.ApiScope, required core.ApiScope) bool {
	if len(allowed) == 0 {
		return false
	}
	req := middleware.normalizeScope(string(required))

	for _, a := range allowed {
		allow := middleware.normalizeScope(string(a))
		if middleware.scopeMatch(allow, req) {
			return true
		}
	}
	return false
}

// 規範化：移除多餘斜線、確保前綴 "/"（但保留 "*" 不動）
func (middleware *APIKey) normalizeScope(s string) string {
	if s == "*" {
		return s
	}
	// 標準化 path
	clean := path.Clean(s)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return clean
}

// 規則：
// 1) "*" → 全通過
// 2) 完全相等
// 3) 父類萬用 "/xxx/*"：允許 "/xxx" 與 "/xxx/..."
func (middleware *APIKey) scopeMatch(allow, req string) bool {
	// 全開
	if allow == "*" {
		return true
	}
	// 精準
	if allow == req {
		return true
	}
	// 父類別萬用
	if strings.HasSuffix(allow, "/*") {
		base := strings.TrimSuffix(allow, "/*")
		// 允許 "/xxx" 本身與 "/xxx/..." 都通過
		return req == base || strings.HasPrefix(req, base+"/")
	}
	return false
}
