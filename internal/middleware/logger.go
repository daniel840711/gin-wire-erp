package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"interchange/config"
	"interchange/internal/core"
	"interchange/internal/database/fluentd/model"
	"interchange/internal/database/fluentd/repository"
	"interchange/internal/pkg/response"
	"interchange/internal/telemetry"
	"io"
	"mime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChatRequest struct {
	Messages []json.RawMessage `json:"messages"`
}

// 只定義需要的結構
type ContentItem struct {
	Text string `json:"text"`
	Type string `json:"type"`
}
type MessageItem struct {
	Content []ContentItem `json:"content"`
	Role    string        `json:"role"`
}
type Logger struct {
	logger            *zap.Logger
	trace             *telemetry.Trace
	config            *config.Configuration
	fluentdRepository *repository.LogRepository
}

func NewLogger(
	logger *zap.Logger,
	trace *telemetry.Trace,
	config *config.Configuration,
	fluentdRepository *repository.LogRepository,
) *Logger {
	return &Logger{
		logger:            logger,
		trace:             trace,
		config:            config,
		fluentdRepository: fluentdRepository,
	}
}

// LoggerHandler 記錄每個請求的詳細資訊（避免讀取二進位 body；文字 body 做安全截斷與 UTF-8 處理）
func (m *Logger) LoggerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		endpoint := c.FullPath()
		if strings.HasPrefix(endpoint, "/swagger") ||
			strings.HasPrefix(endpoint, "/metrics") ||
			strings.HasPrefix(endpoint, "/version") ||
			strings.HasPrefix(endpoint, "/health-check") {
			c.Next()
			return
		}

		ctx, span, end := m.trace.WithSpan(c.Request.Context(), string(core.SpanLoggerMiddleware))

		// ===== 判斷 content-type，二進位不讀 body =====
		ct := c.GetHeader("Content-Type")

		requestTime := time.Now().UTC()
		if startTime, exists := c.Get("requestDuration"); exists {
			if t, ok := startTime.(time.Time); ok {
				requestTime = t
			}
		}

		mediaType, _, _ := mime.ParseMediaType(ct)
		isBinary := isBinaryContent(mediaType)

		var bodyRaw string
		var bodyJSON map[string]any

		if !isBinary && c.Request.Body != nil && c.Request.ContentLength != 0 {
			// 讀完整 body 後回填，確保下游仍可讀取
			data, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(data))

			bodyRaw = toSafePreview(data, 2000)
			// 只在 JSON 時嘗試 decode，避免錯誤/污染 log
			if strings.HasPrefix(mediaType, "application/json") && len(data) > 0 {
				_ = json.Unmarshal(data, &bodyJSON)
			}
		} else if isBinary {
			// 二進位內容不讀 body，提供簡短標記
			if c.Request.ContentLength > 0 {
				bodyRaw = fmt.Sprintf("(binary %s, %d bytes)", mediaType, c.Request.ContentLength)
			} else {
				bodyRaw = fmt.Sprintf("(binary %s)", mediaType)
			}
		}

		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		traceID := span.SpanContext().TraceID()
		spanID := span.SpanContext().SpanID()

		if strings.HasPrefix(mediaType, "application/json") || bodyJSON != nil {
			// 先序列化成 bytes，供 Cline / Roo 判斷
			reqBodyBytes, err := json.Marshal(bodyJSON)
			if err != nil {
				response.AbortWithError(c, err)
				return
			}

			// /mcp-server 才做特殊處理
			if strings.HasPrefix(endpoint, "/mcp-server") {
				if bytes.Contains(reqBodyBytes, []byte("You are Cline,")) {
					bodyRaw, err = getClineText(reqBodyBytes)
				}
				if bytes.Contains(reqBodyBytes, []byte("You are Roo,")) {
					bodyRaw, err = ExtractRooFeedback(reqBodyBytes)
				}
			}
			if err != nil {
				response.AbortWithError(c, err)
				return
			}
		}

		// headers → map[string]string（lowercase key）
		headerMap := make(map[string]string, len(c.Request.Header))
		for k, v := range c.Request.Header {
			lk := strings.ToLower(k)
			headerMap[lk] = strings.Join(v, ",")
		}

		// path params
		paramsMap := make(map[string]string, len(c.Params))
		for _, p := range c.Params {
			paramsMap[p.Key] = p.Value
		}

		// Trace Meta（Body 使用 effectiveBody）
		meta := core.LoggerRequestMeta{
			Method:     method,
			Path:       path,
			FullPath:   endpoint,
			Query:      query,
			Body:       bodyRaw,
			Scheme:     c.Request.URL.Scheme,
			Host:       c.Request.Host,
			UserAgent:  c.Request.UserAgent(),
			ContentLen: c.Request.ContentLength,
			Proto:      c.Request.Proto,
			ClientIP:   c.ClientIP(),
			Headers:    headerMap,
			Params:     paramsMap,
		}
		m.trace.ApplyTraceAttributes(span, meta)

		logFields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Any("headers", headerMap),
		}
		if query != "" {
			logFields = append(logFields, zap.String("query", query))
		}
		if len(paramsMap) > 0 {
			logFields = append(logFields, zap.Any("params", paramsMap))
		}
		if bodyRaw != "" {
			logFields = append(logFields, zap.String("body", bodyRaw)) // ← 統一：處理後的 body
		}
		logFields = append(logFields, zap.String("spanId", fmt.Sprintf("%x", spanID[:])))
		logFields = append(logFields, zap.String("traceId", fmt.Sprintf("%x", traceID[:])))

		m.logger.Info("[Request] logging middleware message", logFields...)

		// Fluentd（Body 使用 effectiveBody）
		responseMeta := model.RequestLog{
			RequestID:   fmt.Sprintf("%x", traceID[:]),
			Method:      method,
			Path:        path,
			ProjectName: m.config.App.Name,
			RequestTS:   requestTime.UTC().Format("2006-01-02 15:04:05.999999 UTC"),
			Body:        bodyRaw, // ← 統一：處理後的 body
			IPHash:      base64.RawStdEncoding.EncodeToString([]byte(c.ClientIP())),
			UserAgent:   c.Request.UserAgent(),
			Version:     m.config.App.Version,
		}
		m.fluentdRepository.LogRequest(ctx, responseMeta)
		end(nil)
		c.Next()
	}
}

// 僅對文字內容做安全預覽：UTF-8 直接截斷；非 UTF-8 以 Base64 表示
func toSafePreview(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}
	if utf8.Valid(b) {
		if len(b) > max {
			return string(b[:max]) + "…"
		}
		return string(b)
	}
	// 非 UTF-8 -> base64（先截斷，避免輸出過大）
	if len(b) > max {
		b = b[:max]
	}
	return "b64:" + base64.StdEncoding.EncodeToString(b)
}

// 是否為二進位內容（不讀 body）
func isBinaryContent(mediaType string) bool {
	return strings.HasPrefix(mediaType, "multipart/") ||
		strings.HasPrefix(mediaType, "image/") ||
		strings.HasPrefix(mediaType, "audio/") ||
		strings.HasPrefix(mediaType, "video/") ||
		mediaType == "application/octet-stream"
}

// 單一函式版：符合 /mcp-server 且第一則訊息以 "You are Cline," 開頭 → 回傳最後一則訊息的原始 JSON
func getClineText(bodyBytes []byte) (string, error) {
	var req ChatRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil || len(req.Messages) == 0 {
		return "", err
	}

	var messageItem MessageItem

	if err := json.Unmarshal(req.Messages[len(req.Messages)-1], &messageItem); err != nil {
		return "", err
	}
	var result []ContentItem
	for _, item := range messageItem.Content {
		// 將 map 序列化成字串，再檢查是否包含三個關鍵字
		raw, _ := json.Marshal(item)
		text := string(raw)
		if strings.Contains(text, "environment_details") {
			continue
		}
		if strings.Contains(text, "\\u003ctask\\u003e") ||
			strings.Contains(text, "ask_followup_question") ||
			strings.Contains(text, "\\u003canswer\\u003e") {
			result = append(result, item)
		}
	}

	text, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	if string(text) == "null" {
		return messageItem.Content[0].Text, nil
	}
	return string(text), nil
}

func ExtractRooFeedback(body []byte) (string, error) {
	var req struct {
		Input []MessageItem `json:"input"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		return "", err
	}
	if len(req.Input) == 0 {
		return "", errors.New("no input found")
	}
	var result []ContentItem
	for _, item := range req.Input[0].Content {
		// 將 map 序列化成字串，再檢查是否包含三個關鍵字
		raw, _ := json.Marshal(item)
		text := string(raw)
		if strings.Contains(text, "environment_details") {
			continue
		}
		if strings.Contains(text, "\\u003ctask\\u003e") ||
			strings.Contains(text, "\\u003canswer\\u003e") ||
			strings.Contains(text, "ask_followup_question") ||
			strings.Contains(text, "\\u003cfeedback\\u003e") {
			result = append(result, item)
		}
	}
	text, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(text), nil
}
