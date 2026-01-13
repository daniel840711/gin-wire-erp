package chat

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

// NewOpenAIService 建立 OpenAIService
func NewOpenAIService(
	trace *telemetry.Trace,
	client *http.Client,
) Service {
	return &OpenAIService{HTTPClient: client, trace: trace}
}

// ChatCompletionsV1 呼叫 OpenAI Chat Completions v1。
// 失敗時依錯誤類型回傳：
//   - 請求送出/對方非 2xx：ExternalRequestError
//   - 回應解碼失敗：ExternalResponseFormatError
//   - 本地序列化/建請失敗：InternalServer
func (s *OpenAIService) ChatCompletionsV1(ctx context.Context, req *ChatPayload, apiKey string) (*ChatResult, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1" + string(core.OpenAiChatEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.chat.completions")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) 序列化 payload
	payload, err := json.Marshal(req)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("marshal chat payload failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create http request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("openai api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		cause := fmt.Errorf("openai non-2xx: %s (%d) %s", resp.Status, resp.StatusCode, strings.TrimSpace(string(b)))
		end(cause)
		return nil, cErr.ExternalRequestError("openai api error: " + strings.TrimSpace(string(b)))
	}

	var result ChatResult
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber() // 精度安全
	if err := dec.Decode(&result); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode openai response failed")
	}

	return &result, nil
}
