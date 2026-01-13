package core

const ContextTraceKey = "telemetry_trace_ctx"

// ==== 型別安全 span name ====
// 專案全域建議都寫這裡，方便集中管理
type TraceSpanName string

const (
	SpanHttpRequest         TraceSpanName = "http_request"
	SpanLoggerMiddleware    TraceSpanName = "logger_middleware"
	SpanRecoveryMiddleware  TraceSpanName = "recovery_middleware"
	SpanCorsMiddleware      TraceSpanName = "cors_middleware"
	SpanResponseMiddleware  TraceSpanName = "response_middleware"
	SpanAPIKeyMiddleware    TraceSpanName = "api_key_middleware"
	SpanRateLimitMiddleware TraceSpanName = "ratelimit_middleware"
	SpanUserMiddleware      TraceSpanName = "user_middleware"
)

// 指標名稱常數
type MetricName string

const (
	MetricHttpRequestsTotal   MetricName = "requests_total"
	MetricHttpRequestDuration MetricName = "request_duration_seconds"
	MetricProxySuccessTotal   MetricName = "proxy_success_total"
	MetricProxyFailTotal      MetricName = "proxy_fail_total"
	MetricKeyBlockedTotal     MetricName = "key_blocked_total"
	MetricKeyUsageGauge       MetricName = "key_usage_gauge"
	MetricRateLimitTotal      MetricName = "rate_limited_total"
)

// label name 常數
type MetricLabelName string

const (
	MetricLabelEndpoint MetricLabelName = "endpoint"
	MetricLabelStatus   MetricLabelName = "status"
	MetricLabelReason   MetricLabelName = "reason"
)

type LoggerRequestMeta struct {
	Method     string            `trace:"request.method"`
	Path       string            `trace:"request.path"`
	FullPath   string            `trace:"request.full_path"`
	Query      string            `trace:"request.query"`
	Body       string            `trace:"request.body"`
	Scheme     string            `trace:"http.scheme"`
	Host       string            `trace:"http.host"`
	UserAgent  string            `trace:"http.user_agent"`
	ContentLen int64             `trace:"http.request_content_length"`
	Proto      string            `trace:"http.flavor"`
	ClientIP   string            `trace:"net.peer.ip"`
	Headers    map[string]string `trace:"http.request.header"`
	Params     map[string]string `trace:"http.request.param"`
}
type TraceAPIKeyAuthMeta struct {
	UserID   string  `trace:"auth.user_id"`
	APIKeyID string  `trace:"auth.api_key_id"`
	Provider string  `trace:"auth.provider"`
	Error    *string `trace:"error"`
}
type TraceRequestMeta struct {
	Method     string         `trace:"http.method"`
	Path       string         `trace:"http.path"`
	Host       string         `trace:"http.host"`
	ClientIP   string         `trace:"net.peer.ip"`
	ContentLen int64          `trace:"http.content_length"`
	FromMobile bool           `trace:"device.mobile"`
	Latency    float64        `trace:"response.latency_sec"`
	Tags       []string       `trace:"request.tags"`
	Headers    map[string]any `trace:"http.request.header"`
}
type TraceAdminUserListMeta struct {
	Page        int64          `trace:"list.page"`
	Size        int64          `trace:"list.size"`
	Role        string         `trace:"list.role,omitempty"`
	Status      string         `trace:"list.status,omitempty"`
	Filter      map[string]any `trace:"filter,omitempty"`
	ResultCount int            `trace:"result.count,omitempty"`
	Error       *string        `trace:"error,omitempty"`
}

// 供 Redis 限流 Consume / Reset 使用
type TraceRateLimitMeta struct {
	APIKeyID  string `trace:"rl.api_key_id"`
	Provider  string `trace:"rl.provider"`
	Period    string `trace:"rl.period"`
	Limit     int    `trace:"rl.limit_count"`
	WindowSec int64  `trace:"rl.window_sec"`
	Remaining int    `trace:"rl.remaining,omitempty"`
	TTL       int64  `trace:"rl.ttl_sec,omitempty"`
	Op        string `trace:"rl.op"` // "consume" / "reset" / "get"
}

// 供 Mongo 統計寫入（usedCount 累加/覆寫）使用
type TraceUsageWriteMeta struct {
	APIKeyID      string  `trace:"usage.api_key_id"`
	Provider      string  `trace:"usage.provider"`
	Increment     int     `trace:"usage.increment,omitempty"`
	MatchedCount  int64   `trace:"usage.matched_count,omitempty"`
	ModifiedCount int64   `trace:"usage.modified_count,omitempty"`
	Error         *string `trace:"error,omitempty"`
}
type TraceUserAPIKeyMeta struct {
	Op            string         `trace:"op"`
	APIKeyID      string         `trace:"api_key.id,omitempty"`
	UserID        string         `trace:"user.id,omitempty"`
	Provider      string         `trace:"provider,omitempty"`
	Filter        map[string]any `trace:"filter,omitempty"`
	Count         int            `trace:"result.count,omitempty"`
	MatchedCount  int64          `trace:"mongo.matched_count,omitempty"`
	ModifiedCount int64          `trace:"mongo.modified_count,omitempty"`
	Increment     int            `trace:"usage.increment,omitempty"`
	UsedCount     int            `trace:"usage.used_count,omitempty"`
}
type TraceRateLimitMiddlewareMeta struct {
	APIKeyID      string `trace:"ratelimit.api_key_id"`
	Provider      string `trace:"ratelimit.provider"`
	Period        string `trace:"ratelimit.period"`
	ConfigLimit   int    `trace:"ratelimit.config.limit"`
	Remaining     int    `trace:"ratelimit.remaining"`
	TTLSeconds    int64  `trace:"ratelimit.ttl_sec"`
	Blocked       bool   `trace:"ratelimit.blocked"`
	Uninitialized bool   `trace:"ratelimit.uninitialized"`
}
type TracePanicMeta struct {
	Path       string  `trace:"http.path"`
	Method     string  `trace:"http.method"`
	ClientIP   string  `trace:"net.peer.ip"`
	UserAgent  string  `trace:"http.user_agent"`
	DurationMs float64 `trace:"response.latency_ms"`
	Status     int     `trace:"http.status_code"`
	Message    string  `trace:"error.message"`
	Stack      string  `trace:"error.stack"`
}

type TraceErrorMeta struct {
	Code       int     `trace:"error.code"`
	Message    string  `trace:"error.message"`
	Detail     string  `trace:"error.detail"`
	Status     int     `trace:"http.status_code"`
	DurationMs float64 `trace:"response.latency_ms"`
}

type TraceResponseMeta struct {
	Path       string  `trace:"http.path"`
	Method     string  `trace:"http.method"`
	Status     int     `trace:"http.status_code"`
	Message    string  `trace:"response.message"`
	Code       int     `trace:"response.code"`
	DurationMs float64 `trace:"response.latency_ms"`
	Data       string  `trace:"response.data_preview"`
}
type TraceHttpServerMeta struct {
	// request side
	ClientAddr        string `trace:"client.address"`
	HttpRequestMethod string `trace:"http.request.method"`
	HttpRoute         string `trace:"http.route"`
	UrlPath           string `trace:"http.request.path"`
	UrlScheme         string `trace:"http.request.url.scheme"`
	UserAgent         string `trace:"user_agent.original"`
	ServerAddress     string `trace:"server.address"`
	NetworkPeerAddr   string `trace:"network.peer.address"`
	NetworkPeerPort   int    `trace:"network.peer.port"`
	NetworkProtoVer   string `trace:"network.protocol.version"`
	SpanKind          string `trace:"span.kind"`
	SpanTraceID       string `trace:"span.trace_id"`
	HttpStatusCode    int    `trace:"http.response.status_code"`
}
type TraceRequestLogMeta struct {
	RequestID   string `trace:"http.request.request_id"`
	Path        string `trace:"http.request.path"`
	Method      string `trace:"http.request.method"`
	ProjectName string `trace:"project.name"`
	Body        string `trace:"http.request.body,omitempty"`
	IPHash      string `trace:"http.request.net.peer.ip_hash"`
	UserAgent   string `trace:"http.request.user_agent"`
	Version     string `trace:"log.version"`
	RequestTS   string `trace:"http.request_ts"`
	LoggedAt    string `trace:"http.logged_at"`
}

type TraceResponseLogMeta struct {
	RequestID   string `trace:"http.request.request_id"`
	ProjectName string `trace:"project.name"`
	Code        int    `trace:"http.response.code"`
	StatusCode  int    `trace:"http.response.status_code"`
	Body        string `trace:"http.response.body,omitempty"`
	Error       string `trace:"http.response.error_message,omitempty"`
	Version     string `trace:"log.version"`
	ResponseTS  string `trace:"http.request_ts"`
	LoggedAt    string `trace:"http.logged_at"`
}

type TraceUsageLogMeta struct {
	RequestID        string `trace:"http.request.request_id"`
	ExternalID       string `trace:"user.external_id,omitempty"`
	DisplayName      string `trace:"user.display_name,omitempty"`
	ProjectName      string `trace:"project.name"`
	Provider         string `trace:"ai.provider"`
	Model            string `trace:"ai.model"`
	Endpoint         string `trace:"ai.endpoint"`
	TokensPrompt     int    `trace:"ai.tokens.prompt"`
	TokensCompletion int    `trace:"ai.tokens.completion"`
	TextToken        int    `trace:"ai.tokens.text"`
	AudioToken       int    `trace:"ai.tokens.audio"`
	ImageToken       int    `trace:"ai.tokens.image"`
	InputToken       int    `trace:"ai.tokens.input"`
	OutputToken      int    `trace:"ai.tokens.output"`
	TokensTotal      int    `trace:"ai.tokens.total"`
	Version          string `trace:"log.version"`
	LoggedAt         string `trace:"http.logged_at"`
}
type TraceAPIKeyMiddlewareMeta struct {
	Provider string   `trace:"auth.provider"`
	Where    string   `trace:"auth.where"`
	ClientIP string   `trace:"net.peer.ip,omitempty"`
	UserID   string   `trace:"auth.user_id,omitempty"`
	APIKeyID string   `trace:"auth.api_key_id,omitempty"`
	KeyName  string   `trace:"auth.key_name,omitempty"`
	Scopes   []string `trace:"auth.scopes,omitempty"`
	Status   string   `trace:"auth.status,omitempty"`
}
type TraceUserMiddlewareMeta struct {
	UserID          string `trace:"auth.user_id,omitempty"`
	UserStatus      string `trace:"auth.user_status,omitempty"`
	UpdatedLastSeen bool   `trace:"user.updated_last_seen"`
	Status          string `trace:"auth.status,omitempty"`
}
