// internal/middleware/trace_entry.go
package middleware

import (
	"net"
	"strconv"
	"strings"
	"time"

	"interchange/config"
	"interchange/internal/core"
	"interchange/internal/telemetry"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type TraceEntry struct {
	trace  *telemetry.Trace
	metric *telemetry.Metric
	conf   *config.Configuration
}

func NewTraceEntry(trace *telemetry.Trace, metric *telemetry.Metric, conf *config.Configuration) *TraceEntry {
	return &TraceEntry{trace: trace, metric: metric, conf: conf}
}

func (m *TraceEntry) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerTraceID := ""
		// 跳過不追蹤的路徑
		endpoint := c.FullPath()
		if strings.HasPrefix(endpoint, "/swagger") ||
			strings.HasPrefix(endpoint, "/metrics") ||
			strings.HasPrefix(endpoint, "/version") ||
			strings.HasPrefix(endpoint, "/health-check") {
			c.Next()
			return
		}
		carrier := propagation.HeaderCarrier(c.Request.Header)
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), carrier)
		path := c.Request.URL.Path
		spanName := c.Request.Method + " " + c.Request.URL.Path
		ctx, span := m.trace.StartSpanForLayer(ctx, core.TraceSpanName(spanName), trace.WithSpanKind(trace.SpanKindServer))
		c.Request = c.Request.WithContext(ctx)
		c.Set(core.ContextTraceKey, ctx)
		curTraceID := span.SpanContext().TraceID().String()
		if headerTraceID == "" {
			headerTraceID = curTraceID
		}
		// 計時
		start := time.Now().UTC()
		if _, exists := c.Get("requestDuration"); !exists {
			c.Set("requestDuration", start)
		}

		// peer ip:port
		peerAddr, peerPort := "", 0
		if host, port, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
			peerAddr = host
			if p, err2 := strconv.Atoi(port); err2 == nil {
				peerPort = p
			}
		} else {
			peerAddr = c.ClientIP()
		}

		// ---- 準備 meta（request 部分）----
		meta := core.TraceHttpServerMeta{
			ClientAddr:        c.ClientIP(),
			HttpRequestMethod: c.Request.Method,
			HttpRoute:         path,
			UrlPath:           c.Request.URL.Path,
			UrlScheme: func() string {
				if c.Request.TLS != nil {
					return "https"
				}
				return "http"
			}(),
			UserAgent:       c.Request.UserAgent(),
			ServerAddress:   m.conf.App.Name,
			NetworkPeerAddr: peerAddr,
			NetworkPeerPort: peerPort,
			NetworkProtoVer: c.Request.Proto,
			SpanTraceID:     headerTraceID,
		}
		// 一次把 request 面向屬性打進 span
		m.trace.ApplyTraceAttributes(span, &meta)

		// ---- 執行後續 ----
		c.Next()

		// 回應狀態與指標
		statusCode := c.Writer.Status()
		meta.HttpStatusCode = statusCode
		m.trace.ApplyTraceAttributes(span, &meta) // 二次打入：補上 status

		if statusCode >= 400 {
			err := c.Errors
			if err == nil || len(c.Errors) == 0 {
				m.trace.EndSpan(span, nil)
			} else {
				m.trace.EndSpan(span, c.Errors.Last().Err)
			}

		}

		// Prometheus
		if m.metric.HttpRequestsTotal != nil && m.metric.HttpRequestDuration != nil {
			duration := time.Since(start)
			m.metric.HttpRequestsTotal.WithLabelValues(endpoint, strconv.Itoa(statusCode)).Inc()
			m.metric.HttpRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
		}
		span.End()
	}
}
