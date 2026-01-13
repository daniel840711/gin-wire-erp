package router

import (
	"interchange/internal/handler"
	"interchange/internal/middleware"

	"github.com/gin-gonic/gin"
)

type MCPRouter struct {
	proxyHandler        *handler.ProxyHandler
	apiKeyMiddleware    *middleware.APIKey
	ratelimitMiddleware *middleware.RateLimit
	userMiddleware      *middleware.User
}

func NewMCPRouter(
	proxyHandler *handler.ProxyHandler,
	apiKeyMiddleware *middleware.APIKey,
	ratelimitMiddleware *middleware.RateLimit,
	userMiddleware *middleware.User,
) *MCPRouter {
	return &MCPRouter{
		proxyHandler:        proxyHandler,
		apiKeyMiddleware:    apiKeyMiddleware,
		ratelimitMiddleware: ratelimitMiddleware,
		userMiddleware:      userMiddleware,
	}
}

func (MCPRouter *MCPRouter) RegisterRoutes(engine *gin.Engine) {
	router := engine.Group("/mcp-server/:version/:provider")
	router.Use(MCPRouter.apiKeyMiddleware.Handler())
	router.Use(MCPRouter.userMiddleware.Handler())
	router.Use(MCPRouter.ratelimitMiddleware.Guard())
	{
		router.Any("/*action", MCPRouter.proxyHandler.Passthrough)
	}
}
