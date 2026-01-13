package handler

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"interchange/config"
	"interchange/internal/core"
	fluentdModel "interchange/internal/database/fluentd/model"
	fluentd "interchange/internal/database/fluentd/repository"
	"interchange/internal/database/mongodb/model"
	"interchange/internal/database/redis/repository"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/telemetry"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// ---- Types kept minimal for SSE parsing ----
type SSEParseResult struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Content string `json:"content"`
	Usage   *Usage `json:"usage,omitempty"`
}
type sseChunk struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Choices []SseChoice `json:"choices"`
	Usage   *Usage      `json:"usage,omitempty"`
}
type SseChoice struct {
	Delta SseDelta `json:"delta"`
}
type SseDelta struct {
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	InputTokens      int `json:"input_tokens,omitempty"`
	OutputTokens     int `json:"output_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// response.output_text.delta
type RespOutputTextDelta struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
	// 其他欄位略
}

// response.output_text.done
type RespOutputTextDone struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// response.completed（usage 在 response.usage 裡）
type ResponseChunk struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Created   int64  `json:"created"`
	CreatedAt int64  `json:"created_at,omitempty"` // 有些實作會用 created_at
	Model     string `json:"model"`
	Usage     *Usage `json:"usage,omitempty"`
}
type RespCompleted struct {
	Type     string        `json:"type"`
	Response ResponseChunk `json:"response"`
}

// ---- Handler ----

type ProxyHandler struct {
	trace                 *telemetry.Trace
	proxyService          *service.ProxyService
	userAPIKeyService     *service.UserAPIKeyService
	logger                *zap.Logger
	config                *config.Configuration
	rateLimiterRepository *repository.RateLimiterRepository
	logRepository         *fluentd.LogRepository
}

func NewProxyHandler(
	trace *telemetry.Trace,
	proxyService *service.ProxyService,
	userAPIKeyService *service.UserAPIKeyService,
	logger *zap.Logger,
	config *config.Configuration,
	rateLimiterRepository *repository.RateLimiterRepository,
	logRepository *fluentd.LogRepository,
) *ProxyHandler {
	return &ProxyHandler{
		trace:                 trace,
		proxyService:          proxyService,
		userAPIKeyService:     userAPIKeyService,
		logger:                logger,
		config:                config,
		rateLimiterRepository: rateLimiterRepository,
		logRepository:         logRepository,
	}
}

// Passthrough 透明轉傳（MCP Server）
// @Summary MCP 透明轉傳
// @Description 將上游請求（method / path / query / headers / body）原樣轉傳到指定 provider 的 API，並把下游的狀態碼、標頭、內容原樣回傳。支援 JSON 與 SSE（text/event-stream）。
// @Tags MCP-Server
// @Accept */*
// @Produce application/json
// @Produce text/event-stream
// @Param version  path string true  "API 版本"            Enums(v1)
// @Param provider path string true  "提供者"              Enums(openai)
// @Param action   path string true  "欲轉傳之相對路徑（萬用字元，例：/chat/completions）"
// @Security ApiKeyAuth
// @Success 200 {string} string "下游原始回應（或 SSE 串流）"
// @Failure 400 {object} cErr.Error "Bad Request"
// @Failure 401 {object} cErr.Error "Unauthorized"
// @Failure 403 {object} cErr.Error "Forbidden"
// @Failure 404 {object} cErr.Error "Not Found"
// @Failure 429 {object} cErr.Error "Too Many Requests"
// @Failure 500 {object} cErr.Error "Internal Server Error"
// @Router /mcp-server/{version}/{provider}/{action} [get]
// @Router /mcp-server/{version}/{provider}/{action} [post]
// @Router /mcp-server/{version}/{provider}/{action} [put]
// @Router /mcp-server/{version}/{provider}/{action} [patch]
// @Router /mcp-server/{version}/{provider}/{action} [delete]
func (h *ProxyHandler) Passthrough(c *gin.Context) {
	ctx, span, end := h.trace.WithSpan(c)
	traceID := span.SpanContext().TraceID()
	defer end(nil)

	version := c.Param("version")
	provider := core.ProviderName(c.Param("provider"))
	action := c.Param("action")
	if action == "" {
		action = "/"
	}
	span.SetAttributes(
		attribute.String("proxy.version", version),
		attribute.String("proxy.provider", string(provider)),
		attribute.String("proxy.action", action),
		attribute.String("http.method", c.Request.Method),
	)

	// ---- helpers ----
	fail := func(err error) {
		end(err)
		response.AbortWithError(c, err)
	}
	getMetaData := func(key string) (string, bool) {
		v, ok := c.Get(key)
		if !ok {
			return "", false
		}
		s, ok := v.(string)
		return s, ok && s != ""
	}

	// ---- auth/context ----
	apiKeyID, ok := getMetaData("apiKeyID")
	if !ok {
		fail(cErr.Unauthorized("missing or invalid API Key"))
		return
	}
	userID, ok := getMetaData("userID")
	if !ok {
		fail(cErr.Unauthorized("missing or invalid API Key"))
		return
	}
	displayName, ok := getMetaData("displayName")
	if !ok {
		fail(cErr.Unauthorized("missing or invalid API Key"))
		return
	}
	rawPA, ok := c.Get("providerAccess")
	if !ok {
		fail(cErr.UnauthorizedApiKey("missing provider access data"))
		return
	}
	providerAccess, ok := rawPA.(*model.ProviderAccess)
	if !ok {
		fail(cErr.InternalServer("invalid provider access data"))
		return
	}
	base, ok := providerBase(provider)
	if !ok {
		fail(cErr.Forbidden("provider not supported: " + string(provider)))
		return
	}

	c.Set("passthrough_raw", true)
	c.Writer.Header().Set("X-Proxy-Passthrough", "true")

	streaming := isStream(c.Request)
	span.SetAttributes(attribute.Bool("mcp.stream", streaming))

	resp, fwdErr := h.proxyService.Forward(ctx, service.ForwardParams{
		Provider:    provider,
		ProviderKey: providerAccess.ProviderKey,
		Method:      c.Request.Method,
		Base:        base,
		Path:        action,
		Version:     version,
		RawQuery:    c.Request.URL.RawQuery,
		Header:      c.Request.Header,
		Body:        c.Request.Body,
	})
	if fwdErr != nil {
		fail(fwdErr)
		return
	}
	defer resp.Body.Close()

	// 扣額度（成功轉發才扣）
	if _, err := h.userAPIKeyService.Consume(ctx, apiKeyID, providerAccess); err != nil {
		fail(err)
		return
	}

	// 先回傳 headers/status（串流不設 Content-Length）
	copyDownstreamHeaders(resp.Header, c.Writer.Header(), streaming)
	c.Status(resp.StatusCode)

	if streaming {
		h.streamAndPreviewSSE(c, resp)
		return
	}

	// ---- non-stream body ----
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fail(cErr.InternalServer("read downstream body failed"))
		return
	}

	// 預覽解析（SSE/JSON 自動處理）
	result := bodyPreviewResult(body, resp.Header, 4000)

	if result.ID == "" && result.Model == "" && result.Content == "" && result.Usage == nil {
		decoded, _ := decompressOnly(body, resp.Header)
		var anyJSON any
		if err := json.Unmarshal(decoded, &anyJSON); err == nil {
			c.Set("data", anyJSON)
		} else {
			c.Set("data", SSEParseResult{
				Object:  "text",
				Content: safeTruncateRunes(string(decoded), 4000),
			})
		}
	} else {
		c.Set("data", result)
	}

	// ---- logging（nil-safe）----
	usage := Usage{}
	if result.Usage != nil { // 若你的型別是大寫 Usage，請改成 result.Usage
		usage = *result.Usage
	}
	_ = h.logRepository.LogUsage(ctx, fluentdModel.AIUsageLog{
		RequestID:        fmt.Sprintf("%x", traceID[:]),
		ExternalID:       userID,
		DisplayName:      displayName,
		ProjectName:      "mcp-server",
		Provider:         string(provider),
		Model:            result.Model,
		Endpoint:         c.Request.URL.Path,
		TokensPrompt:     usage.PromptTokens,
		TokensCompletion: usage.CompletionTokens,
		TextToken:        0,
		AudioToken:       0,
		ImageToken:       0,
		InputToken:       usage.InputTokens,
		OutputToken:      usage.OutputTokens,
		TokensTotal:      usage.TotalTokens,
		Version:          h.config.App.Version,
		LoggedAt:         time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
	})

	// 一律把下游的原始 body 寫回（避免某些分支沒寫回）
	if _, werr := c.Writer.Write(body); werr != nil {
		h.logger.Warn("write downstream body failed", zap.Error(werr))
	}
}

// ---- Streaming (SSE) ----

func (h *ProxyHandler) streamAndPreviewSSE(c *gin.Context, resp *http.Response) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		all, _ := io.ReadAll(resp.Body)
		_, _ = c.Writer.Write(all)
		c.Set("data", bodyPreviewResult(all, resp.Header, 4000))
		return
	}

	const capBytes = 128 * 1024 // 側錄上限
	var mirror bytes.Buffer
	mirror.Grow(capBytes)

	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, werr := c.Writer.Write(chunk); werr != nil {
				break
			}
			flusher.Flush()

			// 側錄
			if mirror.Len() < capBytes {
				remain := capBytes - mirror.Len()
				if n > remain {
					n = remain
				}
				_, _ = mirror.Write(chunk[:n])
			}
		}
		if rerr != nil { // EOF or error
			break
		}
	}

	c.Set("data", bodyPreviewResult(mirror.Bytes(), resp.Header, 4000))
}

// ---- Preview helpers ----
func bodyPreviewResult(raw []byte, header http.Header, maxRunes int) SSEParseResult {
	decoded, _ := decompressOnly(raw, header)
	res := ParseSSEUnified(string(decoded))
	res.Content = safeTruncateRunes(res.Content, maxRunes)
	// 非 SSE：只回 Content
	return res
}

// ---- SSE parsing ----
func ParseSSEUnified(raw string) SSEParseResult {
	var (
		res      SSEParseResult
		builder  strings.Builder
		usage    *Usage
		curEvent string
		dataBuf  strings.Builder // 累積一個事件的多行 data:
		seenMeta bool            // 是否已填好 ID/Object/Created/Model
	)

	flushBlock := func() {
		jsonStr := strings.TrimSpace(dataBuf.String())
		dataBuf.Reset()
		if jsonStr == "" {
			return
		}

		// 無 event → 視為 Chat Completions 的 data 區塊
		if curEvent == "" {
			var ch sseChunk
			if json.Unmarshal([]byte(jsonStr), &ch) == nil {
				// metadata（僅第一次填）
				if !seenMeta && (ch.ID != "" || ch.Model != "") {
					res.ID = ch.ID
					res.Object = ch.Object
					res.Created = ch.Created
					res.Model = ch.Model
					seenMeta = true
				}
				// 累積 content
				for _, c := range ch.Choices {
					if c.Delta.Content != "" {
						builder.WriteString(c.Delta.Content)
					}
				}
				// usage（若有）
				if ch.Usage != nil {
					usage = mergeUsage(usage, ch.Usage)
				}
			}
			return
		}

		// 有 event → Responses API
		switch curEvent {
		case "response.output_text.delta":
			var d RespOutputTextDelta
			if json.Unmarshal([]byte(jsonStr), &d) == nil && d.Delta != "" {
				builder.WriteString(d.Delta)
			}
		case "response.output_text.done":
			// 可選：有些實作會在 done 給完整 text；若前面已累積就忽略
			var d RespOutputTextDone
			if json.Unmarshal([]byte(jsonStr), &d) == nil && d.Text != "" && builder.Len() == 0 {
				builder.WriteString(d.Text)
			}
		case "response.completed":
			var comp RespCompleted
			if json.Unmarshal([]byte(jsonStr), &comp) == nil {
				if !seenMeta && (comp.Response.ID != "" || comp.Response.Model != "") {
					res.ID = comp.Response.ID
					res.Object = comp.Response.Object
					if comp.Response.CreatedAt != 0 {
						res.Created = comp.Response.CreatedAt
					} else {
						res.Created = comp.Response.Created
					}
					res.Model = comp.Response.Model
					seenMeta = true
				}
				if comp.Response.Usage != nil {
					usage = mergeUsage(usage, comp.Response.Usage)
				}
			}
		default:
			// 其他 event 不處理
		}
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "event: "):
			// 新事件開始前，flush 前一個事件累積的 data
			flushBlock()
			curEvent = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
		case strings.HasPrefix(line, "data: "):
			data := strings.TrimPrefix(line, "data: ")
			// Chat Completions 的結束符
			if data == "[DONE]" {
				flushBlock()
				goto DONE
			}
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(data)
		case strings.TrimSpace(line) == "":
			// 事件區塊結束
			flushBlock()
			curEvent = ""
		default:
			// 忽略其他行（如 retry:）
		}
	}
	flushBlock()

DONE:
	res.Content = builder.String()
	if usage != nil {
		res.Usage = usage
	}
	return res
}

// mergeUsage：把兩種來源（prompt/completion vs. input/output）合併到同一份 Usage
func mergeUsage(dst, src *Usage) *Usage {
	if src == nil {
		return dst
	}
	if dst == nil {
		cp := *src
		return &cp
	}
	if src.PromptTokens != 0 {
		dst.PromptTokens = src.PromptTokens
	}
	if src.CompletionTokens != 0 {
		dst.CompletionTokens = src.CompletionTokens
	}
	if src.InputTokens != 0 {
		dst.InputTokens = src.InputTokens
	}
	if src.OutputTokens != 0 {
		dst.OutputTokens = src.OutputTokens
	}
	if src.TotalTokens != 0 {
		dst.TotalTokens = src.TotalTokens
	}
	return dst
}

// 只負責解壓，不做更多處理；若 Content-Encoding 缺失則用 magic 猜測
func decompressOnly(raw []byte, h http.Header) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(h.Get("Content-Encoding")))
	switch enc {
	case "gzip":
		return gunzipBytes(raw)
	case "deflate":
		return inflateZlibBytes(raw)
	case "zstd":
		return zstdBytes(raw)
	case "br":
		return brotliBytes(raw)
	default:
		if isGzip(raw) {
			return gunzipBytes(raw)
		}
		if isZlib(raw) {
			return inflateZlibBytes(raw)
		}
		if isZstd(raw) {
			return zstdBytes(raw)
		}
		return raw, nil
	}
}

// ---- Header / stream utilities ----

func copyDownstreamHeaders(src, dst http.Header, isStream bool) {
	for k, vv := range src {
		ck := http.CanonicalHeaderKey(k)
		switch ck {
		case "Connection",
			"Proxy-Connection",
			"Keep-Alive",
			"Proxy-Authenticate",
			"Proxy-Authorization",
			"Te",
			"Trailer",
			"Transfer-Encoding",
			"Upgrade":
			continue
		case "Content-Length":
			if isStream { // 串流不設定 Content-Length，避免分塊衝突
				continue
			}
		}
		dst.Del(ck)
		for _, v := range vv {
			dst.Add(ck, v)
		}
	}
}

func isStream(r *http.Request) bool {
	if stringsContainsFold(r.Header.Get("Accept"), "text/event-stream") {
		return true
	}
	return stringsContainsFold(r.URL.Query().Get("stream"), "true") // ?stream=true
}

func stringsContainsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// ---- Decompressors ----

func gunzipBytes(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func inflateZlibBytes(b []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func zstdBytes(b []byte) ([]byte, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	return dec.DecodeAll(b, nil)
}
func brotliBytes(b []byte) ([]byte, error) {
	r := brotli.NewReader(bytes.NewReader(b))
	return io.ReadAll(r)
}

// ---- Simple magic number checks ----

func isGzip(b []byte) bool { return len(b) > 2 && b[0] == 0x1f && b[1] == 0x8b }

func isZlib(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x78 && (b[1] == 0x01 || b[1] == 0x9C || b[1] == 0xDA)
}

func isZstd(b []byte) bool {
	return len(b) >= 4 && b[0] == 0x28 && b[1] == 0xB5 && b[2] == 0x2F && b[3] == 0xFD
}

// ---- Misc ----

// 安全截斷前 n 個 rune，避免 UTF-8 亂碼
func safeTruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// providerBase 轉成小函式，集中管理
func providerBase(p core.ProviderName) (string, bool) {
	switch p {
	case core.ProviderOpenAI:
		return string(core.OpenAIAPIBaseURL), true
	case core.ProviderGemini:
		return string(core.GeminiAPIBaseURL), true
	default:
		return "", false
	}
}
