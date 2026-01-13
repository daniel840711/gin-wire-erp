package middleware

import (
	"encoding/base64"
	"fmt"
	"interchange/config"
	"interchange/internal/core"
	cErr "interchange/internal/pkg/error"
	res "interchange/internal/pkg/response"
	"net/http"
	"runtime/debug"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Recovery struct {
	logger *zap.Logger
	// trace             *telemetry.Trace
	// metric            *telemetry.Metric
	config *config.Configuration
	// fluentdRepository *repository.LogRepository
}

func NewRecovery(
	logger *zap.Logger,
	// trace *telemetry.Trace,
	// metric *telemetry.Metric,
	config *config.Configuration,
	// fluentdRepository *repository.LogRepository,
) *Recovery {
	return &Recovery{
		logger: logger,
		// trace:             trace,
		// metric:            metric,
		config: config,
		// fluentdRepository: fluentdRepository,
	}
}

func (middleware *Recovery) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestTime := time.Now()
		if startTime, exists := c.Get("requestDuration"); exists {
			if t, ok := startTime.(time.Time); ok {
				requestTime = t
			}
		}
		RequestID, err := uuid.NewV7()
		if err != nil {
			RequestID = uuid.New()
		}
		// ---- panic recover 必須在 c.Next() 之前註冊 ----
		defer func() {
			if rec := recover(); rec != nil {
				duration := time.Since(requestTime)

				// ctx, span, end := middleware.trace.WithSpan(c.Request.Context(), string(core.SpanRecoveryMiddleware))
				// traceID := span.SpanContext().TraceID()
				// spanID := span.SpanContext().SpanID()
				// defer end(nil)

				meta := core.TracePanicMeta{
					Path:       c.Request.URL.Path,
					Method:     c.Request.Method,
					ClientIP:   c.ClientIP(),
					UserAgent:  c.Request.UserAgent(),
					DurationMs: float64(duration.Milliseconds()),
					Message:    toSafeString(fmt.Sprint(rec)),
					Stack:      toSafeStack(debug.Stack()),
					Status:     http.StatusInternalServerError,
				}
				// middleware.trace.ApplyTraceAttributes(span, meta)

				middleware.logger.Error("[PANIC] Recovered",
					zap.String("path", meta.Path),
					zap.String("method", meta.Method),
					zap.String("client_ip", meta.ClientIP),
					zap.String("user_agent", meta.UserAgent),
					zap.Duration("duration", duration),
					zap.String("panic", meta.Message),
					zap.String("stacktrace", meta.Stack),
					zap.String("requestId", fmt.Sprintf("%x", RequestID.String())),
					// zap.String("spanId", fmt.Sprintf("%x", spanID[:])),
					// zap.String("traceId", fmt.Sprintf("%x", traceID[:])),
				)

				// 尚未回寫才輸出
				if !c.Writer.Written() {
					err := cErr.InternalServer("unexpected panic")
					// end(err)
					res.FailByErr(c, fmt.Sprintf("%x", RequestID.String()), err)
				}
				//fluentd
				/*
					responseMeta := model.ResponseLog{
						RequestID:   fmt.Sprintf("%x", RequestID.String()),
						ProjectName: middleware.config.App.Name,
						Code:        cErr.INTERNAL_ERROR,
						StatusCode:  http.StatusInternalServerError,
						Error:       toSafeString(fmt.Sprint(rec)),
						ResponseTS:  time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
						Version:     middleware.config.App.Version,
					}
					middleware.fluentdRepository.LogResponse(ctx, responseMeta)
				*/
				// metrics
				// if middleware.metric.ProxyFailTotal != nil && middleware.metric.HttpRequestDuration != nil {
				// 	middleware.metric.ProxyFailTotal.WithLabelValues("panic").Inc()
				// 	middleware.metric.HttpRequestDuration.WithLabelValues(c.FullPath()).Observe(duration.Seconds())
				// }
				// 直接中止
				c.Abort()
			}
		}()

		// 執行下游
		c.Next()

		// ---- 統一處理非 panic 的 gin errors（若尚未回寫）----
		if len(c.Errors) > 0 && !c.Writer.Written() {
			duration := time.Since(requestTime)

			// ctx, span, end := middleware.trace.WithSpan(c.Request.Context(), string(core.SpanRecoveryMiddleware))
			// traceID := span.SpanContext().TraceID()
			// spanID := span.SpanContext().SpanID()
			// defer end(nil)

			// 找第一個 *cErr.Error
			for _, e := range c.Errors {
				if appErr, ok := e.Err.(*cErr.Error); ok {
					/*
						meta := core.TraceErrorMeta{
							Code:       appErr.ErrorCode(),
							Message:    appErr.Error(),
							Detail:     appErr.ErrorDesc(),
							DurationMs: float64(duration.Milliseconds()),
							Status:     appErr.HttpCode(),
						}
						middleware.trace.ApplyTraceAttributes(span, meta)
					*/
					middleware.logger.Warn(appErr.Error(),
						zap.Int("code", appErr.ErrorCode()),
						zap.String("data", appErr.ErrorDesc()),
						zap.Duration("duration", duration),
						zap.String("requestId", fmt.Sprintf("%x", RequestID.String())),
						// zap.String("spanId", fmt.Sprintf("%x", spanID[:])),
						// zap.String("traceId", fmt.Sprintf("%x", traceID[:])),
					)
					//fluentd
					/*
						responseMeta := model.ResponseLog{
							RequestID:   fmt.Sprintf("%x", RequestID.String()),
							ProjectName: middleware.config.App.Name,
							Code:        appErr.ErrorCode(),
							StatusCode:  appErr.HttpCode(),
							Error:       appErr.Error(),
							ResponseTS:  time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
							Version:     middleware.config.App.Version,
						}
							middleware.fluentdRepository.LogResponse(ctx, responseMeta)
							if middleware.metric.ProxyFailTotal != nil && middleware.metric.HttpRequestDuration != nil {
								middleware.metric.ProxyFailTotal.WithLabelValues(appErr.Error()).Inc()
								middleware.metric.HttpRequestDuration.WithLabelValues(c.FullPath()).Observe(duration.Seconds())
							}
					*/
					res.FailByErr(c, fmt.Sprintf("%x", RequestID.String()), appErr)
					c.Abort()
					return
				}
			}

			// 其餘未知錯誤
			unknown := c.Errors.String()
			/*
				meta := core.TraceErrorMeta{
					Code:       cErr.INTERNAL_ERROR,
					Message:    "unknown-error",
					Detail:     toSafeString(unknown),
					DurationMs: float64(duration.Milliseconds()),
					Status:     http.StatusInternalServerError,
				}
				middleware.trace.ApplyTraceAttributes(span, meta)
			*/
			middleware.logger.Warn("[ERROR] unknown",
				zap.String("error", unknown),
				zap.Duration("duration", duration),
				zap.String("requestId", fmt.Sprintf("%x", RequestID.String())),
				// zap.String("spanId", fmt.Sprintf("%x", spanID[:])),
				// zap.String("traceId", fmt.Sprintf("%x", traceID[:])),
			)
			//fluentd
			/*
				responseMeta := model.ResponseLog{
					RequestID:   fmt.Sprintf("%x", RequestID.String()),
					ProjectName: middleware.config.App.Name,
					Code:        cErr.INTERNAL_ERROR,
					StatusCode:  http.StatusInternalServerError,
					Error:       toSafeString(unknown),
					ResponseTS:  time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC"),
					Version:     middleware.config.App.Version,
				}
					middleware.fluentdRepository.LogResponse(ctx, responseMeta)
					if middleware.metric.ProxyFailTotal != nil && middleware.metric.HttpRequestDuration != nil {
						middleware.metric.ProxyFailTotal.WithLabelValues("unknown").Inc()
						middleware.metric.HttpRequestDuration.WithLabelValues(c.FullPath()).Observe(duration.Seconds())
					}
			*/
			res.Fail(c, fmt.Sprintf("%x", RequestID.String()), http.StatusInternalServerError, cErr.INTERNAL_ERROR, "unknown-error", unknown)
			c.Abort()
			return
		}
	}
}

// ---- helpers ----

func toSafeString(s string) string {
	const max = 8000
	if utf8.ValidString(s) {
		if len(s) > max {
			return s[:max] + "…"
		}
		return s
	}
	b := []byte(s)
	if len(b) > max {
		b = b[:max]
	}
	return "b64:" + base64.StdEncoding.EncodeToString(b)
}

func toSafeStack(b []byte) string {
	const max = 16000
	if utf8.Valid(b) {
		if len(b) > max {
			return string(b[:max]) + "…"
		}
		return string(b)
	}
	if len(b) > max {
		b = b[:max]
	}
	return "b64:" + base64.StdEncoding.EncodeToString(b)
}
