package handler

import (
	"interchange/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	healthStatus *service.HealthService
}

func NewHealthHandler(status *service.HealthService) *HealthHandler {
	return &HealthHandler{healthStatus: status}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	if h.healthStatus.IsLive() {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
		return
	}
	c.Status(http.StatusServiceUnavailable)
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	if h.healthStatus.IsReady() {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
		return
	}
	c.Status(http.StatusServiceUnavailable)
}
