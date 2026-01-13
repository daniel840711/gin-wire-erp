package handler

import (
	"context"
	"strconv"

	"interchange/internal/core"
	"interchange/internal/dto"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/telemetry"
	"interchange/utils/validate"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AdminUserHandler struct {
	trace       *telemetry.Trace
	userService *service.UserService
}

func NewAdminUserHandler(trace *telemetry.Trace, userService *service.UserService) *AdminUserHandler {
	return &AdminUserHandler{trace: trace, userService: userService}
}

// List 用戶列表
// @Summary 取得用戶列表
// @Tags Admin-User
// @Security BearerAuth
// @Produce json
// @Param page query int false "頁碼"
// @Param size query int false "每頁筆數"
// @Param role query string false "角色"
// @Param status query string false "狀態"
// @Success 200 {array} dto.UserResponseDto
// @Failure 500 {object} map[string]string
// @Router /admin/users [get]
func (h *AdminUserHandler) List(c *gin.Context) {
	ctx, span, end := h.trace.WithSpan(c)
	defer end(nil)

	page := getInt64Query(c, "page", 0)
	size := getInt64Query(c, "size", 20)
	role := c.Query("role")
	status := c.Query("status")

	filter := map[string]any{}
	if role != "" {
		filter["role"] = role
	}
	if status != "" {
		filter["status"] = status
	}

	users, err := h.userService.ListUsers(ctx, filter, page, size)
	meta := core.TraceAdminUserListMeta{
		Page:        page,
		Size:        size,
		Role:        role,
		Status:      status,
		Filter:      filter,
		ResultCount: len(users),
	}
	h.trace.ApplyTraceAttributes(span, meta)

	if err != nil {
		end(err)
		response.AbortWithError(c, cErr.InternalServer(err.Error()))
		return
	}
	response.Success(c, users)
}

// Get 取得用戶
// @Summary 取得單一用戶資訊
// @Tags Admin-User
// @Security BearerAuth
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {object} dto.UserResponseDto
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /admin/users/{userID} [get]
func (h *AdminUserHandler) Get(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)
	id, cause, err := validate.ParseObjectID(c, "userID")
	if err != nil {
		end(cause)
		response.AbortWithError(c, err)
		return
	}

	user, err := h.userService.GetUserByID(ctx, id)

	if err != nil {
		response.AbortWithError(c, err)
		return
	}

	response.Success(c, user)
}

// Create 新增用戶
// @Summary 新增用戶
// @Tags Admin-User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateUserDto true "用戶資訊"
// @Success 201 {object} dto.UserResponseDto
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users [post]
func (h *AdminUserHandler) Create(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)
	var req dto.CreateUserDto
	if cause, respErr := validate.BindAndValidate(c, &req); cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	res, err := h.userService.CreateUser(ctx, &req)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.Create(c, res)
}

// Update 更新用戶
// @Summary 更新用戶
// @Tags Admin-User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param body body dto.UpdateUserDto true "用戶更新資訊"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID} [put]
func (h *AdminUserHandler) Update(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)
	id, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	var req dto.UpdateUserDto
	if cause, respErr = validate.BindAndValidate(c, &req); cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	err := h.userService.UpdateUserByID(ctx, id, &req)
	if err != nil {
		response.AbortWithError(c, cErr.InternalServer(err.Error()))
		return
	}
	response.Success(c, "user updated successfully")
}

// UpdateStatus 更新用戶狀態
// @Summary 更新用戶狀態
// @Tags Admin-User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param body body dto.UpdateUserStatusDto true "狀態資訊"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/status [patch]
func (h *AdminUserHandler) UpdateStatus(c *gin.Context) {
	h.updateField(c,
		func(ctx context.Context, id primitive.ObjectID, req any) error {
			return h.userService.UpdateUserStatus(ctx, id, req.(*dto.UpdateUserStatusDto))
		},
		&dto.UpdateUserStatusDto{},
		"user status updated successfully",
	)
}

// UpdateRole 更新用戶角色
// @Summary 更新用戶角色
// @Tags Admin-User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param body body dto.UpdateUserRoleDto true "角色資訊"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/role [patch]
func (h *AdminUserHandler) UpdateRole(c *gin.Context) {
	h.updateField(c,
		func(ctx context.Context, id primitive.ObjectID, req any) error {
			return h.userService.UpdateUserRole(ctx, id, req.(*dto.UpdateUserRoleDto))
		},
		&dto.UpdateUserRoleDto{},
		"user role updated successfully",
	)
}

// Delete 刪除用戶
// @Summary 刪除用戶
// @Tags Admin-User
// @Security BearerAuth
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID} [delete]
func (h *AdminUserHandler) Delete(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)
	id, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	err := h.userService.DeleteUser(ctx, id)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.Success(c, "user deleted successfully")
}

func getInt64Query(c *gin.Context, key string, defaultVal int64) int64 {
	if v := c.Query(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}
func (h *AdminUserHandler) updateField(
	c *gin.Context,
	updateFn func(ctx context.Context, id primitive.ObjectID, req any) error,
	req any,
	successMsg string,
) {
	ctx, _, end := h.trace.WithSpan(c)

	id, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}
	if cause, respErr := validate.BindAndValidate(c, req); cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	err := updateFn(ctx, id, req)
	end(err)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.Success(c, successMsg)
}
