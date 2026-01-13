package error

import "net/http"

type Error struct {
	httpCode  int
	errorCode int
	errorMsg  string
	errorDesc string
}

func New(httpCode, errorCode int, errorMsg string, errorDesc string) *Error {
	return &Error{
		httpCode:  httpCode,
		errorCode: errorCode,
		errorMsg:  errorMsg,
		errorDesc: errorDesc,
	}

}
func From(err error) *Error {
	if appErr, ok := err.(*Error); ok {
		return appErr
	}
	return InternalServer(err.Error())
}

// ✅ 用戶端錯誤 (400 系列)
func ValidateErr(errorDesc string) *Error {
	errCode := BAD_REQUEST_BODY
	return New(http.StatusBadRequest, errCode, "bad-request/body", errorDesc)
}
func ValidatePathParamsErr(errorDesc string) *Error {
	errCode := BAD_REQUEST_PARAMS
	return New(http.StatusBadRequest, errCode, "bad-request/params", errorDesc)
}

// 更語義化的建構器（Consume 會用到）
func RateLimiterUnavailable(desc string) *Error {
	return New(http.StatusServiceUnavailable, SERVICE_UNAVAILABLE, "rate-limiter-unavailable", desc)
}

// ✅ 伺服器內部錯誤 (500 系列)
func InternalServer(errorDesc string) *Error {
	return New(http.StatusInternalServerError, INTERNAL_ERROR, "internal-server-error", errorDesc)
}

func DatabaseError(errorDesc string) *Error {
	return New(http.StatusInternalServerError, DATABASE_ERROR, "database-error", errorDesc)
}

func ServiceUnavailable(errorDesc string) *Error {
	return New(http.StatusServiceUnavailable, SERVICE_UNAVAILABLE, "service-unavailable", errorDesc)
}

// ✅ 外部 API 錯誤 (502, 504)
func ExternalRequestError(errorDesc string) *Error {
	return New(http.StatusBadGateway, EXTERNAL_REQUEST_ERROR, "external-request-failed", errorDesc)
}

func ExternalResponseFormatError(errorDesc string) *Error {
	return New(http.StatusBadGateway, EXTERNAL_RESPONSE_FORMAT_ERROR, "external-response-invalid", errorDesc)
}

func GatewayTimeout(errorDesc string) *Error {
	return New(http.StatusGatewayTimeout, GATEWAY_TIMEOUT, "gateway-timeout", errorDesc)
}

func UnsupportedVersion(errorDesc string) *Error {
	return New(http.StatusHTTPVersionNotSupported, UNSUPPORTED_VERSION, "unsupported-version", errorDesc)
}

// ✅ 用戶請求錯誤 (400 系列)
func BadRequest(errorDesc string, errorCode ...int) *Error {
	errCode := BAD_REQUEST_BODY
	if len(errorCode) > 0 {
		errCode = errorCode[0]
	}
	return New(http.StatusBadRequest, errCode, "bad-request", errorDesc)
}
func BadRequestBody(errorDesc string) *Error {
	return New(http.StatusBadRequest, BAD_REQUEST_BODY, "bad-request-body", errorDesc)
}

func BadRequestParams(errorDesc string) *Error {
	return New(http.StatusBadRequest, BAD_REQUEST_PARAMS, "bad-request-params", errorDesc)
}

func BadRequestHeaders(errorDesc string) *Error {
	return New(http.StatusBadRequest, BAD_REQUEST_HEADERS, "bad-request-headers", errorDesc)
}

func ReservationStatusError(errorDesc string) *Error {
	return New(http.StatusBadRequest, RESERVATION_STATUS_ERROR, "reservation-status-error", errorDesc)
}

func CannotBlockWithinCheckInTime(errorDesc string) *Error {
	return New(http.StatusBadRequest, CANNOT_BLOCK_WITHIN_CHECK_IN_TIME, "cannot-block-within-check-in-time", errorDesc)
}

func CheckInTimeExceeded(errorDesc string) *Error {
	return New(http.StatusBadRequest, CHECK_IN_TIME_EXCEEDED, "check-in-time-exceeded", errorDesc)
}

// ✅ 權限錯誤 (401, 403)
func Unauthorized(errorDesc string, errorCode ...int) *Error {
	errCode := UNAUTHORIZED
	if len(errorCode) > 0 {
		errCode = errorCode[0]
	}
	return New(http.StatusUnauthorized, errCode, "unauthorized", errorDesc)
}

func InvalidSession(errorDesc string) *Error {
	return New(http.StatusUnauthorized, INVALID_SESSION, "invalid-session", errorDesc)
}

func UnauthorizedApiKey(errorDesc string) *Error {
	return New(http.StatusForbidden, UNAUTHORIZED_API_KEY, "unauthorized-api-key", errorDesc)
}

func RateLimitExceeded(errorDesc string) *Error {
	return New(http.StatusTooManyRequests, RATE_LIMIT_EXCEEDED, "rate-limit-exceeded", errorDesc)
}

func Forbidden(errorDesc string, errorCode ...int) *Error {
	errCode := FORBIDDEN
	if len(errorCode) > 0 {
		errCode = errorCode[0]
	}
	return New(http.StatusForbidden, errCode, "forbidden", errorDesc)
}

// ✅ 資源找不到 (404)
func NotFound(errorDesc string, errorCode ...int) *Error {
	errCode := NOT_FOUND
	if len(errorCode) > 0 {
		errCode = errorCode[0]
	}
	return New(http.StatusNotFound, errCode, "not-found", errorDesc)
}
func (e *Error) HttpCode() int {
	return e.httpCode
}

func (e *Error) ErrorCode() int {
	return e.errorCode
}
func (e *Error) ErrorDesc() string {
	return e.errorDesc
}
func (e *Error) Error() string {
	return e.errorMsg
}
func MapHttpStatusToError(status int, desc string) *Error {
	switch status {
	case http.StatusBadRequest:
		return BadRequest(desc)
	case http.StatusUnauthorized:
		return Unauthorized(desc)
	case http.StatusForbidden:
		return Forbidden(desc)
	case http.StatusNotFound:
		return NotFound(desc)
	case http.StatusInternalServerError:
		return InternalServer(desc)
	case http.StatusServiceUnavailable:
		return ServiceUnavailable(desc)
	case http.StatusGatewayTimeout:
		return GatewayTimeout(desc)
	default:
		return InternalServer(desc)
	}
}
