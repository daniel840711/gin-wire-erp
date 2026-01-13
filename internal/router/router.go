package router

import (
	docs "interchange/cmd/docs"
	"interchange/config"
	"interchange/internal/middleware"
	"interchange/internal/pkg/response"
	"net/http"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var ProviderSet = wire.NewSet(
	NewRouter,
	NewAdminRouter,
	NewAdminUserAPIKeyRouter,
	NewProxyRouter,
	NewMCPRouter,
)

// 透過依賴注入將
func NewRouter(
	config *config.Configuration,
	traceEntry *middleware.TraceEntry,
	recovery *middleware.Recovery,
	cors *middleware.Cors,
	logger *middleware.Logger,
	responseMiddleware *middleware.Response,
	adminRouter *AdminRouter,
	proxyRouter *ProxyRouter,
	mcpRouter *MCPRouter,
) *gin.Engine {

	switch config.App.Env {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}
	router := gin.New()
	router.Use(traceEntry.Handler())
	router.Use(logger.LoggerHandler())
	router.Use(cors.CorsHandler())
	router.Use(recovery.ErrorHandler())
	router.Use(responseMiddleware.FormatHandler())
	router.GET("/health-check", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.Response{
			Code:        0,
			Data:        "ok",
			Message:     "success",
			Description: "service is alive",
		})
		c.Abort()
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if config.App.SwaggerEnabled {
		router.GET("/swagger/*any", func(c *gin.Context) {
			docs.SwaggerInfo.Host = c.Request.Host

			if config.App.Env == "production" {
				docs.SwaggerInfo.Schemes = []string{"https"}
				docs.SwaggerInfo.BasePath = "/interchange/api"
			}
		}, ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	// 註冊 AI Proxy 入口路由
	proxyRouter.RegisterRoutes(router)
	mcpRouter.RegisterRoutes(router)
	adminRouter.RegisterRoutes(router)
	pprof.Register(router)
	return router
}
