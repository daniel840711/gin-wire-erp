package model

type AIUsageLog struct {
	// 身份/追蹤
	RequestID        string `bson:"request_id,omitempty" json:"request_id"`
	ExternalID       string `bson:"external_id,omitempty" json:"external_id,omitempty"`
	DisplayName      string `bson:"display_name,omitempty" json:"display_name,omitempty"`
	ProjectName      string `bson:"project_name,omitempty" json:"project_name,omitempty"`
	Provider         string `bson:"provider" json:"provider"`
	Model            string `bson:"model,omitempty" json:"model,omitempty"`
	Endpoint         string `bson:"endpoint" json:"endpoint"`
	TokensPrompt     int    `bson:"tokens_prompt,omitempty" json:"tokens_prompt,omitempty"`
	TokensCompletion int    `bson:"tokens_completion,omitempty" json:"tokens_completion,omitempty"`
	TextToken        int    `bson:"text_tokens,omitempty" json:"text_tokens,omitempty"`
	AudioToken       int    `bson:"audio_tokens,omitempty" json:"audio_tokens,omitempty"`
	ImageToken       int    `bson:"image_tokens,omitempty" json:"image_tokens,omitempty"`
	InputToken       int    `bson:"input_tokens,omitempty" json:"input_tokens,omitempty"`
	OutputToken      int    `bson:"output_tokens,omitempty" json:"output_tokens,omitempty"`
	TokensTotal      int    `bson:"tokens_total,omitempty" json:"tokens_total,omitempty"`
	Version          string `bson:"version" json:"version"`
	LoggedAt         string `bson:"logged_at" json:"logged_at"`
}
