package router

import (
	"interchange/internal/handler"

	"github.com/gin-gonic/gin"
)

type AdminRouter struct {
	userHandler           *handler.AdminUserHandler
	adminUserAPIKeyRouter *AdminUserAPIKeyRouter
}

func NewAdminRouter(
	userHandler *handler.AdminUserHandler,
	adminUserAPIKeyRouter *AdminUserAPIKeyRouter,
) *AdminRouter {
	return &AdminRouter{
		userHandler:           userHandler,
		adminUserAPIKeyRouter: adminUserAPIKeyRouter,
	}
}

func (ar *AdminRouter) RegisterRoutes(r *gin.Engine) {
	admin := r.Group("/admin/users")
	{
		admin.GET("", ar.userHandler.List)
		admin.GET("/:userID", ar.userHandler.Get)
		admin.POST("", ar.userHandler.Create)
		admin.PUT("/:userID", ar.userHandler.Update)
		admin.PATCH("/:userID/status", ar.userHandler.UpdateStatus)
		admin.PATCH("/:userID/role", ar.userHandler.UpdateRole)
		admin.DELETE("/:userID", ar.userHandler.Delete)

		// user_api_key 子路由直接獨立管理
		ar.adminUserAPIKeyRouter.Register(admin)
	}
}
