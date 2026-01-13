package middleware

import (
	"interchange/internal/core"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/pkg/response"
	"interchange/internal/service"
	"interchange/internal/telemetry"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type User struct {
	logger      *zap.Logger
	trace       *telemetry.Trace
	metric      *telemetry.Metric
	userService *service.UserService
}

func NewUser(
	logger *zap.Logger,
	trace *telemetry.Trace,
	metric *telemetry.Metric,
	userService *service.UserService,
) *User {
	return &User{
		logger:      logger,
		trace:       trace,
		metric:      metric,
		userService: userService,
	}
}

func (m *User) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span, end := m.trace.WithSpan(c.Request.Context(), string(core.SpanUserMiddleware))
		var cause error = nil
		rawUID, ok := c.Get("userID")
		if !ok {

			m.trace.ApplyTraceAttributes(span, core.TraceUserMiddlewareMeta{
				Status: "missing_user_context",
			})
			cause = cErr.UnauthorizedApiKey("missing user context")
			response.AbortWithError(c, cause)
			end(cause)
			return
		}

		uidStr, _ := rawUID.(string)
		oid, err := primitive.ObjectIDFromHex(uidStr)
		if err != nil {
			m.trace.ApplyTraceAttributes(span, core.TraceUserMiddlewareMeta{
				Status: "invalid_user_id",
			})

			cause = cErr.UnauthorizedApiKey("invalid userID format")
			response.AbortWithError(c, cause)
			end(cause)
			return
		}

		userDTO, err := m.userService.GetUserByID(ctx, oid)
		if err != nil {
			m.trace.ApplyTraceAttributes(span, core.TraceUserMiddlewareMeta{
				UserID: uidStr,
				Status: "user_check_failed",
			})
			response.AbortWithError(c, err)
			end(err)
			return
		}
		if userDTO.Status != core.StatusActive {
			m.trace.ApplyTraceAttributes(span, core.TraceUserMiddlewareMeta{
				UserID:     uidStr,
				Status:     "invalid_user_status",
				UserStatus: string(userDTO.Status),
			})
			err := cErr.Unauthorized("invalid_user_status")
			response.AbortWithError(c, err)
			end(err)
			return
		}
		updateErr := m.userService.UpdateUserLastSeen(ctx, oid)
		if updateErr != nil {
			m.trace.ApplyTraceAttributes(span, core.TraceUserMiddlewareMeta{
				UserID:     uidStr,
				Status:     "update_last_seen_failed",
				UserStatus: string(userDTO.Status),
			})
			response.AbortWithError(c, updateErr)
			end(updateErr)
			return
		}
		// 成功
		m.trace.ApplyTraceAttributes(span, core.TraceUserMiddlewareMeta{
			UserID:          userDTO.ID,
			UserStatus:      string(userDTO.Status),
			UpdatedLastSeen: true,
			Status:          "success",
		})
		c.Set("displayName", userDTO.DisplayName)
		end(nil)
		c.Next()
	}
}
