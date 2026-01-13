package models

import "context"

// 單一 Model
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`   // "model"
	Created int64  `json:"created"`  // unix seconds
	OwnedBy string `json:"owned_by"` // e.g. "openai" / "organization-owner"
}

// 列表回應
type ListResponse struct {
	Object string  `json:"object"` // "list"
	Data   []Model `json:"data"`
}

// 服務介面
type Service interface {
	List(ctx context.Context, apiKey string) (*ListResponse, error)
}
