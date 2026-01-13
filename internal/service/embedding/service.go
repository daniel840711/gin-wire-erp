package embedding

import "context"

type EmbeddingRequestBody struct {
	Input          interface{} `json:"input"`                     // string or []string or [][]int
	Model          string      `json:"model"`                     // required
	Dimensions     *int        `json:"dimensions,omitempty"`      // optional
	EncodingFormat string      `json:"encoding_format,omitempty"` // optional: "float" or "base64"
	User           string      `json:"user,omitempty"`            // optional
}
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"` // e.g. 1536 floats for ada-002
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
type Service interface {
	GenerateEmbedding(ctx context.Context, req *EmbeddingRequestBody, apiKey string) (*EmbeddingResponse, error)
}
