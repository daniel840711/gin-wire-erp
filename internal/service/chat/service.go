package chat

import "context"

type ChatPayload struct {
	Model               string                 `json:"model"`
	Messages            []ChatMessage          `json:"messages"`
	MaxCompletionTokens *int                   `json:"max_completion_tokens,omitempty"`
	Temperature         *float64               `json:"temperature,omitempty"`
	TopP                *float64               `json:"top_p,omitempty"`
	FrequencyPenalty    *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64               `json:"presence_penalty,omitempty"`
	N                   *int                   `json:"n,omitempty"`
	Stop                interface{}            `json:"stop,omitempty"` // string or []string or null
	Stream              *bool                  `json:"stream,omitempty"`
	Logprobs            *bool                  `json:"logprobs,omitempty"`
	TopLogprobs         *int                   `json:"top_logprobs,omitempty"`
	LogitBias           map[string]float64     `json:"logit_bias,omitempty"`    // token id(string): bias
	Tools               interface{}            `json:"tools,omitempty"`         // array (function/tool desc)
	ToolChoice          interface{}            `json:"tool_choice,omitempty"`   // string or object
	Functions           interface{}            `json:"functions,omitempty"`     // Deprecated (array)
	FunctionCall        interface{}            `json:"function_call,omitempty"` // Deprecated (string or object)
	Metadata            map[string]string      `json:"metadata,omitempty"`
	ParallelToolCalls   *bool                  `json:"parallel_tool_calls,omitempty"`
	Seed                *int64                 `json:"seed,omitempty"`
	Modalities          *[]string              `json:"modalities,omitempty"`      // e.g. ["text", "audio"]
	Audio               map[string]interface{} `json:"audio,omitempty"`           // 支援 audio output
	ResponseFormat      map[string]interface{} `json:"response_format,omitempty"` // e.g. {type:json_schema}
	Prediction          map[string]interface{} `json:"prediction,omitempty"`
	WebSearchOptions    map[string]interface{} `json:"web_search_options,omitempty"`
	ReasoningEffort     *string                `json:"reasoning_effort,omitempty"` // "low"/"medium"/"high"
	ServiceTier         *string                `json:"service_tier,omitempty"`     // "auto"/"default"/...
	PromptCacheKey      *string                `json:"prompt_cache_key,omitempty"`
	SafetyIDentifier    *string                `json:"safety_identifier,omitempty"`
	Store               *bool                  `json:"store,omitempty"`
	StreamOptions       map[string]interface{} `json:"stream_options,omitempty"`
	User                *string                `json:"user,omitempty"` // Deprecated, for compat
	// Custom platform extension:
	Extensions map[string]interface{} `json:"extensions,omitempty"` // any custom keys for RAG, vendor-specific
}

// 建議訊息型態用結構支援多模態
type ChatMessage struct {
	Role         string                 `json:"role"`                    // "system"/"user"/"assistant"/...
	Content      interface{}            `json:"content"`                 // string (text)、map（image/audio等）、支援多模態
	Name         *string                `json:"name,omitempty"`          // optional
	FunctionCall interface{}            `json:"function_call,omitempty"` // 可存 openai function call
	ToolCalls    interface{}            `json:"tool_calls,omitempty"`    // 工具調用紀錄
	Extensions   map[string]interface{} `json:"extensions,omitempty"`
}
type ChatResult struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatChoice           `json:"choices"`
	Usage             ChatUsage              `json:"usage"`
	ServiceTier       string                 `json:"service_tier,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Extensions        map[string]interface{} `json:"extensions,omitempty"` // 供其他平台或自家加欄位
}

type ChatChoice struct {
	Index        int                 `json:"index"`
	Message      ChatResponseMessage `json:"message"`
	Logprobs     interface{}         `json:"logprobs,omitempty"`
	FinishReason string              `json:"finish_reason"`
	// 可再擴充：如 Gemini 會有自家 output/ext，可加 Extensions
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type ChatResponseMessage struct {
	Role        string        `json:"role"`
	Content     interface{}   `json:"content"`
	Refusal     interface{}   `json:"refusal,omitempty"`
	Annotations []interface{} `json:"annotations,omitempty"`
	// 支援未來擴充（如 Gemini 多模態等）
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// usage 欄位
type ChatUsage struct {
	PromptTokens            int                    `json:"prompt_tokens"`
	CompletionTokens        int                    `json:"completion_tokens"`
	TotalTokens             int                    `json:"total_tokens"`
	PromptTokensDetails     map[string]interface{} `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails map[string]interface{} `json:"completion_tokens_details,omitempty"`
	Extensions              map[string]interface{} `json:"extensions,omitempty"`
}
type Service interface {
	ChatCompletionsV1(ctx context.Context, req *ChatPayload, key string) (*ChatResult, error)
}
