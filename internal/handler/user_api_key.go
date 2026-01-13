package handler

import (
	"fmt"
	"interchange/internal/core"
	"interchange/internal/dto"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/telemetry"
	"interchange/utils/validate"

	"github.com/gin-gonic/gin"
)

type AdminUserAPIKeyHandler struct {
	trace             *telemetry.Trace
	userAPIKeyService *service.UserAPIKeyService
	userService       *service.UserService
}

func NewAdminUserAPIKeyHandler(
	trace *telemetry.Trace,
	userAPIKeyService *service.UserAPIKeyService,
	userService *service.UserService,
) *AdminUserAPIKeyHandler {
	return &AdminUserAPIKeyHandler{trace: trace, userAPIKeyService: userAPIKeyService, userService: userService}
}

// ListByUserID 列出用戶的所有 API Key
// @Summary 取得用戶的 API Key 列表
// @Tags Admin-APIKey
// @Security BearerAuth
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {array} dto.UserAPIKeyResponseDto
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/api-keys [get]
func (h *AdminUserAPIKeyHandler) ListByUserID(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)

	userID, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if _, err := h.userService.GetUserByID(ctx, userID); err != nil {
		response.AbortWithError(c, err)
		return
	}

	keys, err := h.userAPIKeyService.ListByUserID(ctx, userID)
	if err != nil {
		response.AbortWithError(c, cErr.InternalServer(err.Error()))
		return
	}
	response.Success(c, keys)
}

// CreateForUser 新增 API Key
// @Summary 為指定用戶新增 API Key
// @Tags Admin-APIKey
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param body body dto.CreateUserAPIKeyDto true "API Key 資訊"
// @Success 201 {object} dto.UserAPIKeyResponseDto
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/api-keys [post]
func (h *AdminUserAPIKeyHandler) CreateForUser(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)

	userID, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	// 確認使用者存在
	if _, err := h.userService.GetUserByID(ctx, userID); err != nil {
		response.AbortWithError(c, err)
		return
	}

	var req dto.CreateUserAPIKeyDto
	if cause, respErr := validate.BindAndValidate(c, &req); cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	created, err := h.userAPIKeyService.Create(ctx, userID, &req)
	if err != nil {
		response.AbortWithError(c, err)
		return
	}
	response.Create(c, created)
}

// UpdateProviderAccessAll 全量更新 provider_access
// @Summary 全量覆蓋 provider_access 欄位
// @Tags Admin-APIKey
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param apiKeyID path string true "API Key ID"
// @Param body body dto.UpdateProviderAccessAllDto true "provider_access"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/api-keys/{apiKeyID} [put]
func (h *AdminUserAPIKeyHandler) UpdateProviderAccessAll(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)

	userID, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if _, err := h.userService.GetUserByID(ctx, userID); err != nil {
		response.AbortWithError(c, cErr.NotFound(fmt.Sprintf("user with id %s not found", userID.Hex())))
		return
	}

	apiKeyID, cause, respErr := validate.ParseObjectID(c, "apiKeyID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	var req dto.UpdateProviderAccessAllDto
	if cause, respErr := validate.BindAndValidate(c, &req); cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if err := h.userAPIKeyService.UpdateProviderAccessAll(ctx, apiKeyID, req.ProviderAccess); err != nil {
		response.AbortWithError(c, cErr.InternalServer(err.Error()))
		return
	}
	response.Success(c, "provider_access updated successfully")
}

// PatchProviderField 局部更新 provider 欄位
// @Summary 局部更新 provider 欄位
// @Tags Admin-APIKey
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param apiKeyID path string true "API Key ID"
// @Param provider path string true "Provider"
// @Param body body object true "要更新的欄位map"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/api-keys/{apiKeyID}/provider/{provider} [patch]
func (h *AdminUserAPIKeyHandler) PatchProviderField(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)

	userID, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if _, err := h.userService.GetUserByID(ctx, userID); err != nil {
		response.AbortWithError(c, cErr.NotFound(fmt.Sprintf("user with id %s not found", userID.Hex())))
		return
	}

	apiKeyID, cause, respErr := validate.ParseObjectID(c, "apiKeyID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	provider := core.ProviderName(c.Param("provider"))

	var fields map[string]any
	if cause, respErr := validate.BindAndValidate(c, &fields); cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if err := h.userAPIKeyService.UpdateProviderFields(ctx, apiKeyID, provider, fields); err != nil {
		response.AbortWithError(c, cErr.InternalServer(err.Error()))
		return
	}
	response.Success(c, "provider field updated successfully")
}

// DeleteByID 刪除 API Key
// @Summary 刪除 API Key
// @Tags Admin-APIKey
// @Security BearerAuth
// @Produce json
// @Param userID path string true "User ID"
// @Param apiKeyID path string true "API Key ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/users/{userID}/api-keys/{apiKeyID} [delete]
func (h *AdminUserAPIKeyHandler) DeleteByID(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)

	userID, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if _, err := h.userService.GetUserByID(ctx, userID); err != nil {
		response.AbortWithError(c, cErr.NotFound(fmt.Sprintf("user with id %s not found", userID.Hex())))
		return
	}

	apiKeyID, cause, respErr := validate.ParseObjectID(c, "apiKeyID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if err := h.userAPIKeyService.DeleteByID(ctx, apiKeyID); err != nil {

		response.AbortWithError(c, cErr.InternalServer(err.Error()))
		return
	}
	response.Success(c, "api key deleted successfully")
}

// GetByID 取得單一 API Key
// @Summary 取得單一 API Key
// @Tags Admin-APIKey
// @Security BearerAuth
// @Produce json
// @Param userID path string true "User ID"
// @Param apiKeyID path string true "API Key ID"
// @Success 200 {object} dto.UserAPIKeyResponseDto
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /admin/users/{userID}/api-keys/{apiKeyID} [get]
func (h *AdminUserAPIKeyHandler) GetByID(c *gin.Context) {
	ctx, _, end := h.trace.WithSpan(c)
	defer end(nil)

	userID, cause, respErr := validate.ParseObjectID(c, "userID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	if _, err := h.userService.GetUserByID(ctx, userID); err != nil {
		response.AbortWithError(c, cErr.NotFound(fmt.Sprintf("user with id %s not found", userID.Hex())))
		return
	}

	apiKeyID, cause, respErr := validate.ParseObjectID(c, "apiKeyID")
	if cause != nil {
		end(cause)
		response.AbortWithError(c, respErr)
		return
	}

	key, err := h.userAPIKeyService.GetByID(ctx, apiKeyID)
	if err != nil {
		response.AbortWithError(c, cErr.NotFound("api key not found"))
		return
	}
	response.Success(c, key)
}
