package service

import (
	"context"
	"io"
	"net/http"
	"strings"

	"interchange/internal/core"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
)

type ForwardParams struct {
	Provider    core.ProviderName // e.g. openai, gemini
	ProviderKey string            // 從你們的使用者 API-Key 解析到的供應商金鑰
	Method      string            // GET/POST/PUT/DELETE...
	Base        string            // 供應商 Base URL（不含最後的 /）
	Path        string            // 供應商相對路徑（可含或不含前導 /）
	Version     string            //供應商提供的版本
	RawQuery    string            // 原始 query string（不含 ?）
	Header      http.Header       // 來自上游請求的 header（會做 hop-by-hop 過濾）
	Body        io.Reader         // 請求 body（可直接塞 c.Request.Body）
}
type ProxyService struct {
	httpClient *http.Client
	trace      *telemetry.Trace
}

func NewProxyService(trace *telemetry.Trace, client *http.Client) *ProxyService {
	// client 建議在 DI 時就統一設好 Transport／Timeout
	return &ProxyService{
		httpClient: client,
		trace:      trace,
	}
}

func (service *ProxyService) Forward(ctx context.Context, req ForwardParams) (*http.Response, error) {
	ctx, span, end := service.trace.WithSpan(ctx, "mcp.forward")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", string(req.Provider)),
		attribute.String("http.method", req.Method),
		attribute.String("mcp.raw_path", req.Path),
		attribute.String("mcp.version", req.Version),
		attribute.String("mcp.raw_query", req.RawQuery),
	)

	// 組合目標 URL
	base := strings.TrimRight(req.Base, "/")
	path := "/" + strings.TrimLeft(req.Path, "/")
	version := "/" + strings.TrimLeft(req.Version, "/")
	target := base + version + path
	if req.RawQuery != "" {
		target = target + "?" + req.RawQuery
	}
	span.SetAttributes(attribute.String("http.url", target))

	request, err := http.NewRequestWithContext(ctx, req.Method, target, req.Body)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create provider request failed")
	}

	// 1) 複製上游 header（去除 hop-by-hop 與由我們管理的授權）
	copySafeHeaders(req.Header, request.Header)

	// 2) 依 provider 注入授權
	switch req.Provider {
	case core.ProviderOpenAI:
		// OpenAI: Authorization: Bearer <key>
		request.Header.Set("Authorization", "Bearer "+req.ProviderKey)
	case core.ProviderGemini:
		// Gemini/Google AI: 走 header（亦可改走 query key）
		request.Header.Set("x-goog-api-key", req.ProviderKey)
	default:
		return nil, cErr.Forbidden("unsupported provider")
	}

	// 預設補上 Accept
	if request.Header.Get("Accept") == "" {
		request.Header.Set("Accept", "application/json")
	}

	resp, err := service.httpClient.Do(request)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("provider request failed")
	}

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	return resp, nil
}

// ---- helpers ----

var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func copySafeHeaders(src http.Header, dst http.Header) {
	// 先複製
	for k, vv := range src {
		if _, banned := hopByHopHeaders[http.CanonicalHeaderKey(k)]; banned {
			continue
		}
		// 來自上游的 Authorization 不往下游傳（避免把使用者帶來的 token 洩漏到供應商）
		if strings.EqualFold(k, "Authorization") {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
	// RFC7230: 若 Connection 有列出其他 header，也必須移除
	if cval := src.Get("Connection"); cval != "" {
		tokens := strings.Split(cval, ",")
		for _, t := range tokens {
			if h := http.CanonicalHeaderKey(strings.TrimSpace(t)); h != "" {
				dst.Del(h)
			}
		}
	}
}
