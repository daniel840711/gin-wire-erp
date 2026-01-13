package error

const (
	// 0 ~ 999: 成功類別
	SUCCESS = 0 // 200 OK

	// 40000 ~ 49999: 用戶請求錯誤 (400 系列)
	BAD_REQUEST_BODY                  = 40000 // 400 - 無效的請求體
	BAD_REQUEST_PARAMS                = 40001 // 400 - 無效的請求參數
	BAD_REQUEST_HEADERS               = 40002 // 400 - 無效的請求標頭
	RESERVATION_STATUS_ERROR          = 40003 // 400 - 預約狀態錯誤
	CANNOT_BLOCK_WITHIN_CHECK_IN_TIME = 40004 // 400 - 無法在入住時間內封鎖預約
	CHECK_IN_TIME_EXCEEDED            = 40005 // 400 - 簽到時間已超過

	// 40100 ~ 40399: 驗證與權限錯誤 (401 403 系列)
	UNAUTHORIZED         = 40100 // 401 - 未授權
	INVALID_SESSION      = 40101 // 401 - 會話失效
	UNAUTHORIZED_API_KEY = 40300 // 403 - API Key 無權限
	FORBIDDEN            = 40301 // 403 - 禁止訪問

	// 40400 ~ 40499: 資源錯誤 (404 系列)
	NOT_FOUND = 40400 // 404 - 資源未找到

	// 42900 ~ 42999: 流量限制錯誤 (429 系列)
	RATE_LIMIT_EXCEEDED = 42900 // 429 - 速率限制超過

	// 50000 ~ 50199: 伺服器內部錯誤 (500 系列)
	INTERNAL_ERROR      = 50000 // 500 - 內部錯誤
	DATABASE_ERROR      = 50001 // 500 - 資料庫錯誤
	SERVICE_UNAVAILABLE = 50002 // 503 - 服務暫停 (維護模式)

	// 50200 ~ 50499: 外部請求錯誤 (502 504 系列)
	EXTERNAL_REQUEST_ERROR         = 50200 // 502 - 外部 API 請求錯誤
	EXTERNAL_RESPONSE_FORMAT_ERROR = 50201 // 502 - 外部 API 回應格式錯誤
	GATEWAY_TIMEOUT                = 50400 // 504 - 外部 API 超時
	UNSUPPORTED_VERSION            = 50401 // 505 - 不支援的 API 版本
)
