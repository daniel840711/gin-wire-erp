package models

import (
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

func (s *OpenAIService) List(ctx context.Context, apiKey string) (*ListResponse, error) {

	url := string(core.OpenAIAPIBaseURL) + "/v1" + string(core.OpenAIModelsEndpoint)

	ctx, span, end := s.trace.WithSpan(ctx, "openai.models.list")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 建立 GET 請求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create http request failed")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	// 發送
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("openai api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// 狀態碼處理
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		end(fmt.Errorf("openai non-2xx: %s (%d) %s", resp.Status, resp.StatusCode, trimBody(body)))
		return nil, cErr.ExternalRequestError("openai api error: " + trimBody(body))
	}

	// 解析回應
	var out ListResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode openai models response failed")
	}

	return &out, nil
}

func trimBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 3000 {
		return s[:3000] + "..."
	}
	return s
}
