package middleware

import (
	"interchange/internal/core"
	"interchange/internal/telemetry"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Cors struct {
	trace *telemetry.Trace
}

func NewCors(trace *telemetry.Trace) *Cors {
	return &Cors{trace: trace}
}

// CorsHandler 設定 CORS，並以 WithSpan 紀錄設定（跳過特定路徑的 tracing，但仍套用 CORS）
func (m *Cors) CorsHandler() gin.HandlerFunc {
	cfg := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-API-Key", "X-Api-Key"},
		AllowCredentials: true,
	}
	corsHandler := cors.New(cfg)

	type corsMeta struct {
		AllowOrigins []string `trace:"http.cors.allow_origins"`
		AllowMethods []string `trace:"http.cors.allow_methods"`
		AllowHeaders []string `trace:"http.cors.allow_headers"`
		AllowCreds   bool     `trace:"http.cors.allow_credentials"`
	}

	return func(c *gin.Context) {
		endpoint := c.FullPath()

		// 這些路徑：不做 tracing，但仍需套用 CORS（避免 preflight 失敗）
		if strings.HasPrefix(endpoint, "/swagger") ||
			strings.HasPrefix(endpoint, "/metrics") ||
			strings.HasPrefix(endpoint, "/version") ||
			strings.HasPrefix(endpoint, "/health-check") {
			corsHandler(c)
			return
		}

		_, span, end := m.trace.WithSpan(c.Request.Context(), string(core.SpanCorsMiddleware))
		defer end(nil)

		// 記錄 CORS 設定到 trace（以 struct tag）
		m.trace.ApplyTraceAttributes(span, corsMeta{
			AllowOrigins: cfg.AllowOrigins,
			AllowMethods: cfg.AllowMethods,
			AllowHeaders: cfg.AllowHeaders,
			AllowCreds:   cfg.AllowCredentials,
		})

		// 執行實際的 CORS middleware（其內部會呼叫 c.Next()）
		corsHandler(c)
	}
}
