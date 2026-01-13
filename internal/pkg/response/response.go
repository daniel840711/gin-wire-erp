package response

import (
	cErr "interchange/internal/pkg/error"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	RequestID   string `json:"requestID"`
	Code        int    `json:"code"`
	Data        any    `json:"data"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

func Create(c *gin.Context, data any) {
	message := "Create Success"
	if msg, ok := data.(gin.H); ok && msg["message"] != "" {
		message = msg["message"].(string)
		delete(msg, "message")
	}
	c.Set("data", data)
	c.Set("message", message)
	c.Abort()
}
func Success(c *gin.Context, data any) {
	message := "Request Success"
	if msg, ok := data.(gin.H); ok && msg["message"] != "" {
		message = msg["message"].(string)
		delete(msg, "message")
	}
	c.Set("data", data)
	c.Set("message", message)
	c.Abort()
}
func AbortWithError(c *gin.Context, err error) {
	c.Error(err)
	c.Abort()
}
func Fail(c *gin.Context, RequestID string, httpCode int, errorCode int, msg string, desc string) {
	c.JSON(httpCode, Response{
		RequestID:   RequestID,
		Code:        errorCode,
		Data:        nil,
		Message:     msg,
		Description: desc,
	})
	c.Abort()
}

func FailByErr(c *gin.Context, RequestID string, err error) {
	v, ok := err.(*cErr.Error)
	if ok {
		Fail(c, RequestID, v.HttpCode(), v.ErrorCode(), v.Error(), v.ErrorDesc())
	} else {
		Fail(c, RequestID, http.StatusBadRequest, cErr.INTERNAL_ERROR, err.Error(), "internal error")
	}
}
