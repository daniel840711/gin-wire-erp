package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"interchange/internal/core"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/telemetry"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

type OpenAIService struct {
	HTTPClient *http.Client
	trace      *telemetry.Trace
}

func NewOpenAIService(trace *telemetry.Trace, client *http.Client) Service {
	return &OpenAIService{HTTPClient: client, trace: trace}
}

func (s *OpenAIService) GenerateEmbedding(ctx context.Context, req *EmbeddingRequestBody, apiKey string) (*EmbeddingResponse, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1" + string(core.OpenAIEmbeddingEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.embeddings.generate")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) 序列化 payload
	payload, err := json.Marshal(req)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("marshal embedding payload failed")
	}

	// 2) 建立 HTTP 請求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create http request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// 3) 發送請求
	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("openai api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// 4) 狀態碼處理
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		end(fmt.Errorf("openai non-2xx: %s (%d) %s", resp.Status, resp.StatusCode, trimBody(b)))

		return nil, cErr.ExternalRequestError("openai api error: " + trimBody(b))
	}

	// 5) 解析回應
	var result EmbeddingResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode openai embedding response failed")
	}

	return &result, nil
}
func trimBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 3000 {
		return s[:3000] + "..."
	}
	return s
}
