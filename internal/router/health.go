package router

import (
	"interchange/internal/handler"

	"github.com/gin-gonic/gin"
)

type HealthRouter struct {
	healthHandler *handler.HealthHandler
}

func NewHealthRouter(
	healthHandler *handler.HealthHandler,
) *HealthRouter {
	return &HealthRouter{
		healthHandler: healthHandler,
	}
}

func (healthRouter *HealthRouter) RegisterHealthRoutes(r *gin.Engine) {
	g := r.Group("/health")
	{
		g.GET("/liveness", healthRouter.healthHandler.Liveness)
		g.GET("/readiness", healthRouter.healthHandler.Readiness)
	}
}
