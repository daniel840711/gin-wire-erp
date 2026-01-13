package core

// ProviderName
type ProviderName string

const (
	ProviderOpenAI ProviderName = "openai"
	ProviderGemini ProviderName = "gemini"
	ProviderGrok   ProviderName = "grok"
	ProviderCustom ProviderName = "custom"
)

// LimitPeriod
type LimitPeriod string

const (
	LimitPeriodDaily   LimitPeriod = "daily"
	LimitPeriodWeekly  LimitPeriod = "weekly"
	LimitPeriodMonthly LimitPeriod = "monthly"
	LimitPeriodYearly  LimitPeriod = "yearly"
	LimitPeriodNone    LimitPeriod = "none"
)

// ApiScope
type ApiScope string

const (
	ApiScopeChatCompletions       ApiScope = "/chat/completions"
	ApiScopeImagesGenerations     ApiScope = "/images/generations"
	ApiScopeImagesVariations      ApiScope = "/images/variations"
	ApiScopeImagesEdits           ApiScope = "/images/edits"
	ApiScopeAudioTranscriptions   ApiScope = "/audio/transcriptions"
	ApiScopeEmbeddingsGenerations ApiScope = "/embeddings"
	ApiScopeGetModels             ApiScope = "/models"
	ApiScopeMCPServer             ApiScope = "/mcp-server/*"
	ApiScopeAll                   ApiScope = "*"
)

type OpenAIEndpoint string

const (
	OpenAIAPIBaseURL = "https://api.openai.com"
	GeminiAPIBaseURL = "https://generativelanguage.googleapis.com"
)
const (
	OpenAiChatEndpoint           OpenAIEndpoint = "/chat/completions"
	OpenAISpeechEndpoint         OpenAIEndpoint = "/audio/speech"
	OpenAITranscriptionEndpoint  OpenAIEndpoint = "/audio/transcriptions"
	OpenAITranslationEndpoint    OpenAIEndpoint = "/audio/translations"
	OpenAIEmbeddingEndpoint      OpenAIEndpoint = "/embeddings"
	OpenAIImageGenerateEndpoint  OpenAIEndpoint = "/images/generations"
	OpenAIImageEditEndpoint      OpenAIEndpoint = "/images/edits"
	OpenAIImageVariationEndpoint OpenAIEndpoint = "/images/variations"
	OpenAIModelsEndpoint         OpenAIEndpoint = "/models"
)
