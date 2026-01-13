package router

import (
	"interchange/internal/handler"

	"github.com/gin-gonic/gin"
)

type AdminUserAPIKeyRouter struct {
	handler *handler.AdminUserAPIKeyHandler
}

func NewAdminUserAPIKeyRouter(
	handler *handler.AdminUserAPIKeyHandler,
) *AdminUserAPIKeyRouter {
	return &AdminUserAPIKeyRouter{handler: handler}
}

// 這個方法讓你可以把 route group 掛在任何 group 下
func (ar *AdminUserAPIKeyRouter) Register(group *gin.RouterGroup) {
	apiKeys := group.Group("/:userID/api-keys")
	{
		apiKeys.GET("", ar.handler.ListByUserID)
		apiKeys.POST("", ar.handler.CreateForUser)
		apiKeys.GET("/:apiKeyID", ar.handler.GetByID)
		apiKeys.DELETE("/:apiKeyID", ar.handler.DeleteByID)
	}
}
