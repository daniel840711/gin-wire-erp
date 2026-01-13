package middleware

import (
	"encoding/json"
	"fmt"
	"interchange/config"
	"interchange/internal/core"
	"interchange/internal/database/fluentd/model"
	"interchange/internal/database/fluentd/repository"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/telemetry"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Response struct {
	logger            *zap.Logger
	trace             *telemetry.Trace
	metric            *telemetry.Metric
	config            *config.Configuration
	fluentdRepository *repository.LogRepository
}

func NewResponse(
	logger *zap.Logger,
	trace *telemetry.Trace,
	metric *telemetry.Metric,
	config *config.Configuration,
	fluentdRepository *repository.LogRepository,
) *Response {
	return &Response{
		logger:            logger,
		trace:             trace,
		metric:            metric,
		config:            config,
		fluentdRepository: fluentdRepository,
	}
}

func (middleware *Response) FormatHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		endpoint := c.FullPath()
		if strings.HasPrefix(endpoint, "/swagger") ||
			strings.HasPrefix(endpoint, "/metrics") ||
			strings.HasPrefix(endpoint, "/version") ||
			strings.HasPrefix(endpoint, "/health-check") {
			c.Next()
			return
		}

		requestTime := time.Now()
		if startTime, exists := c.Get("requestDuration"); exists {
			if t, ok := startTime.(time.Time); ok {
				requestTime = t
			}
		} else {
			c.Set("requestDuration", requestTime)
		}

		// 執行下游
		c.Next()

		skipWrap := false
		if raw, ok := c.Get("passthrough_raw"); ok {
			if b, _ := raw.(bool); b {
				skipWrap = true
			}
		}
		// 若已經有錯誤交由 Recovery 處理，或已經寫出回應，就不要再動了
		if len(c.Errors) > 0 || (!skipWrap && c.Writer.Written()) {
			return
		}

		// 以「下游結束後」的狀態碼為準
		statusCode := c.Writer.Status()

		// 若 status >= 400：轉為應用錯誤交給 Recovery 統一輸出
		if statusCode >= http.StatusBadRequest {
			response.AbortWithError(c, cErr.MapHttpStatusToError(statusCode, "request error"))
			return
		}

		// ---- 成功回應路徑 ----
		ctx, span, end := middleware.trace.WithSpan(c.Request.Context(), string(core.SpanResponseMiddleware))
		defer end(nil)

		// 組裝回應資料（由 handler 透過 c.Set 設定）
		data, _ := c.Get("data")
		if data == nil {
			data = map[string]any{}
		}
		msg, _ := c.Get("message")
		message := "Request Success"
		if s, ok := msg.(string); ok && s != "" {
			message = s
		}

		if len(c.Errors) > 0 || (!skipWrap && c.Writer.Written()) {
			return
		}
		duration := time.Since(requestTime)
		traceID := span.SpanContext().TraceID()
		spanID := span.SpanContext().SpanID()

		// Trace Meta
		middleware.trace.ApplyTraceAttributes(span, core.TraceResponseMeta{
			Path:       c.Request.URL.Path,
			Method:     c.Request.Method,
			Status:     statusCode,
			Message:    message,
			Code:       0,
			DurationMs: float64(duration.Milliseconds()),
			Data:       safePreviewJSON(data, 2000),
		})

		// Log
		middleware.logger.Info("[Response] "+message,
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("spanId", fmt.Sprintf("%x", spanID[:])),
			zap.String("traceId", fmt.Sprintf("%x", traceID[:])),
		)

		//fluentd
		respBody, _ := json.Marshal(data)
		responseMeta := model.ResponseLog{
			RequestID:   fmt.Sprintf("%x", traceID[:]),
			ProjectName: middleware.config.App.Name,
			Code:        0,
			StatusCode:  statusCode,
			Body:        string(respBody),
			ResponseTS:  time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
			Version:     middleware.config.App.Version,
		}
		middleware.fluentdRepository.LogResponse(ctx, responseMeta)
		// Metrics
		if middleware.metric.ProxySuccessTotal != nil && middleware.metric.HttpRequestDuration != nil {
			middleware.metric.ProxySuccessTotal.
				WithLabelValues(endpoint, strconv.Itoa(statusCode)).
				Inc()
			middleware.metric.HttpRequestDuration.
				WithLabelValues(endpoint).
				Observe(duration.Seconds())
		}
		if c.Writer.Written() {
			return
		}

		// 封裝統一回應
		res := response.Response{
			RequestID:   fmt.Sprintf("%x", traceID[:]),
			Code:        0,
			Data:        data,
			Message:     "OK",
			Description: message,
		}

		// JSON encode（UTF-8 安全）
		jsonBytes, err := json.Marshal(res)
		if err != nil {
			// Marshal 失敗視為 500，交給 Recovery 處理
			response.AbortWithError(c, cErr.InternalServer("marshal response failed"))
			return
		}

		// 輸出 JSON
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(statusCode) // 例如 handler 可能設了 201
		if _, werr := c.Writer.Write(jsonBytes); werr != nil {
			// write 失敗也交 Recovery
			response.AbortWithError(c, cErr.InternalServer("write response failed"))
			return
		}

	}
}

// safePreviewJSON 會把資料序列化為 JSON 字串（UTF-8），並限制長度。
func safePreviewJSON(data any, max int) string {
	switch v := data.(type) {
	case string:
		// 原邏輯：試圖解析 JSON 字串
		var js any
		if err := json.Unmarshal([]byte(v), &js); err != nil {
			if len(v) > max {
				return v[:max] + "…"
			}
			return v
		}
		b, _ := json.Marshal(js)
		out := string(b)
		if len(out) > max {
			return out[:max] + "…"
		}
		return out
	default:
		// 對 struct/map/其他型別，直接 Marshal
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("[marshal error: %v]", err)
		}
		out := string(b)
		if len(out) > max {
			return out[:max] + "…"
		}
		return out
	}
}
