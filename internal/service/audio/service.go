package audio

import (
	"context"
	"mime/multipart"
)

type AudioTranscriptionRequestBody struct {
	Model                  string                `form:"model" binding:"required"` // whisper-1, gpt-4o-transcribe, etc.
	File                   *multipart.FileHeader `form:"file" binding:"required"`
	Language               string                `form:"language,omitempty"` // ISO-639-1 e.g. "en"
	Prompt                 string                `form:"prompt,omitempty"`
	ResponseFormat         string                `form:"response_format,omitempty"` // json, text, srt, etc.
	Temperature            float64               `form:"temperature,omitempty"`
	ChunkingStrategy       string                `form:"chunking_strategy,omitempty"`   // "auto" or blank
	Include                []string              `form:"include[]" binding:"omitempty"` // optional logprobs
	TimestampGranularities []string              `form:"timestamp_granularities[]" binding:"omitempty"`
	Stream                 *bool                 `form:"stream,omitempty"` // gpt-4o only
}

type AudioSpeechRequestBody struct {
	Input          string  `json:"input" binding:"required"`
	Model          string  `json:"model" binding:"required"`
	Voice          string  `json:"voice" binding:"required"`
	Instructions   string  `json:"instructions,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"` // mp3, wav, etc.
	Speed          float32 `json:"speed,omitempty"`           // 0.25 - 4.0
	StreamFormat   string  `json:"stream_format,omitempty"`   // audio or sse
}
type AudioTranslationRequestBody struct {
	File           *multipart.FileHeader `form:"file" binding:"required"`  // 使用 multipart/form-data 上傳檔案
	Model          string                `form:"model" binding:"required"` // ex: whisper-1
	Prompt         string                `form:"prompt,omitempty"`
	ResponseFormat string                `form:"response_format,omitempty"` // json, text, srt, verbose_json, vtt
	Temperature    float32               `form:"temperature,omitempty"`     // 0 ~ 1
}

type AudioTranscriptionResponse struct {
	Text  string                   `json:"text"`
	Usage *AudioTranscriptionUsage `json:"usage"`
}
type AudioSpeechResponse struct {
	Data        []byte
	ContentType string
	Stream      bool
}
type AudioTranscriptionUsage struct {
	InputTokens        int                                  `json:"input_tokens"`
	InputTokensDetails *AudioTranscriptionInputTokensDetail `json:"input_tokens_details,omitempty"`
	OutputTokens       int                                  `json:"output_tokens"`
	TotalTokens        int                                  `json:"total_tokens"`
}

type AudioTranscriptionInputTokensDetail struct {
	TextTokens  int `json:"text_tokens"`
	AudioTokens int `json:"audio_tokens"`
}
type Service interface {
	AudioSpeechV1(ctx context.Context, req *AudioSpeechRequestBody, apiKey string) (*AudioSpeechResponse, error)
	AudioTranscriptionsV1(ctx context.Context, req *AudioTranscriptionRequestBody, apiKey string) (*AudioTranscriptionResponse, error)
	AudioTranslationsV1(ctx context.Context, req *AudioTranslationRequestBody, apiKey string) (string, error)
}
