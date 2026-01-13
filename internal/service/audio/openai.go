package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"interchange/internal/core"
	cErr "interchange/internal/pkg/error"
	"interchange/internal/telemetry"
	"io"
	"mime/multipart"
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

// AudioTranscriptionsV1 呼叫 OpenAI /audio/transcriptions。
// 失敗分類：
//   - 本地組裝/建請失敗：InternalServer
//   - 對外請求/非 2xx：ExternalRequestError
//   - 回應解析失敗：ExternalResponseFormatError
func (s *OpenAIService) AudioTranscriptionsV1(ctx context.Context, req *AudioTranscriptionRequestBody, apiKey string) (*AudioTranscriptionResponse, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1" + string(core.OpenAITranscriptionEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.audio.transcriptions")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) multipart 組裝
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	audioFile, err := req.File.Open()
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("open audio file failed")
	}
	defer audioFile.Close()

	part, err := writer.CreateFormFile("file", req.File.Filename)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create form file failed")
	}
	if _, err := io.Copy(part, audioFile); err != nil {
		end(err)
		return nil, cErr.InternalServer("copy audio file failed")
	}

	if err := writer.WriteField("model", req.Model); err != nil {
		end(err)
		return nil, cErr.InternalServer("write model field failed")
	}
	if req.Language != "" {
		if err := writer.WriteField("language", req.Language); err != nil {
			end(err)
			return nil, cErr.InternalServer("write language field failed")
		}
	}
	if req.Prompt != "" {
		if err := writer.WriteField("prompt", req.Prompt); err != nil {
			end(err)
			return nil, cErr.InternalServer("write prompt field failed")
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", req.ResponseFormat); err != nil {
			end(err)
			return nil, cErr.InternalServer("write response_format field failed")
		}
	}
	if req.ChunkingStrategy != "" {
		if err := writer.WriteField("chunking_strategy", req.ChunkingStrategy); err != nil {
			end(err)
			return nil, cErr.InternalServer("write chunking_strategy field failed")
		}
	}
	if req.Temperature > 0 {
		if err := writer.WriteField("temperature", floatToStr(req.Temperature)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write temperature field failed")
		}
	}
	for _, v := range req.Include {
		if err := writer.WriteField("include[]", v); err != nil {
			end(err)
			return nil, cErr.InternalServer("write include field failed")
		}
	}
	for _, v := range req.TimestampGranularities {
		if err := writer.WriteField("timestamp_granularities[]", v); err != nil {
			end(err)
			return nil, cErr.InternalServer("write timestamp_granularities field failed")
		}
	}
	if req.Stream != nil {
		if err := writer.WriteField("stream", boolToStr(*req.Stream)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write stream field failed")
		}
	}
	if err := writer.Close(); err != nil {
		end(err)
		return nil, cErr.InternalServer("close multipart writer failed")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create http request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("openai api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		cause := newHTTPError(resp, b)
		end(cause)
		return nil, cErr.ExternalRequestError("openai api error: " + trimBody(b))
	}

	var result AudioTranscriptionResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode openai transcription response failed")
	}
	return &result, nil
}

// AudioSpeechV1 呼叫 OpenAI /audio/speech。
// 失敗分類同上；讀取音訊資料失敗 -> ExternalResponseFormatError。
func (s *OpenAIService) AudioSpeechV1(ctx context.Context, req *AudioSpeechRequestBody, apiKey string) (*AudioSpeechResponse, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1" + string(core.OpenAISpeechEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.audio.speech")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) multipart 組裝
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("input", req.Input); err != nil {
		end(err)
		return nil, cErr.InternalServer("write input field failed")
	}
	if err := writer.WriteField("model", req.Model); err != nil {
		end(err)
		return nil, cErr.InternalServer("write model field failed")
	}
	if err := writer.WriteField("voice", req.Voice); err != nil {
		end(err)
		return nil, cErr.InternalServer("write voice field failed")
	}
	if req.Instructions != "" {
		if err := writer.WriteField("instructions", req.Instructions); err != nil {
			end(err)
			return nil, cErr.InternalServer("write instructions field failed")
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", req.ResponseFormat); err != nil {
			end(err)
			return nil, cErr.InternalServer("write response_format field failed")
		}
	}
	if req.Speed > 0 {
		if err := writer.WriteField("speed", fmt.Sprintf("%f", req.Speed)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write speed field failed")
		}
	}
	if req.StreamFormat != "" {
		if err := writer.WriteField("stream_format", req.StreamFormat); err != nil {
			end(err)
			return nil, cErr.InternalServer("write stream_format field failed")
		}
	}
	if err := writer.Close(); err != nil {
		end(err)
		return nil, cErr.InternalServer("close multipart writer failed")
	}

	// 2) 建請
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create http request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// 3) 請求
	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("openai api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// 4) 狀態碼
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		cause := newHTTPError(resp, b)
		end(cause)
		return nil, cErr.ExternalRequestError("openai api error: " + trimBody(b))
	}

	// 5) 讀取音訊資料
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("read openai speech response failed")
	}

	return &AudioSpeechResponse{
		Data:        data,
		ContentType: resp.Header.Get("Content-Type"),
		Stream:      req.StreamFormat == "sse",
	}, nil
}

// AudioTranslationsV1 呼叫 OpenAI /audio/translations。
// 非 text 格式以 JSON 解碼 { "text": "..." }。
func (s *OpenAIService) AudioTranslationsV1(ctx context.Context, req *AudioTranslationRequestBody, apiKey string) (string, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1" + string(core.OpenAITranslationEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.audio.translations")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) multipart 組裝
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	file, err := req.File.Open()
	if err != nil {
		end(err)
		return "", cErr.InternalServer("open uploaded file failed")
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", req.File.Filename)
	if err != nil {
		end(err)
		return "", cErr.InternalServer("create form file failed")
	}
	if _, err := io.Copy(part, file); err != nil {
		end(err)
		return "", cErr.InternalServer("copy file content failed")
	}

	if err := writer.WriteField("model", req.Model); err != nil {
		end(err)
		return "", cErr.InternalServer("write model field failed")
	}
	if req.Prompt != "" {
		if err := writer.WriteField("prompt", req.Prompt); err != nil {
			end(err)
			return "", cErr.InternalServer("write prompt field failed")
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", req.ResponseFormat); err != nil {
			end(err)
			return "", cErr.InternalServer("write response_format field failed")
		}
	}
	if req.Temperature != 0 {
		if err := writer.WriteField("temperature", fmt.Sprintf("%f", req.Temperature)); err != nil {
			end(err)
			return "", cErr.InternalServer("write temperature field failed")
		}
	}
	if err := writer.Close(); err != nil {
		end(err)
		return "", cErr.InternalServer("close multipart writer failed")
	}

	// 2) 建請
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		end(err)
		return "", cErr.InternalServer("create http request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	// 3) 請求
	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		end(err)
		return "", cErr.ExternalRequestError("openai api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	body, _ := io.ReadAll(resp.Body)

	// 4) 狀態碼
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		cause := newHTTPError(resp, body)
		end(cause)
		return "", cErr.ExternalRequestError("openai api error: " + trimBody(body))
	}

	// 5) 根據 response_format 處理
	if strings.EqualFold(req.ResponseFormat, "text") {
		return string(body), nil
	}

	var result struct {
		Text string `json:"text"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		end(err)
		return "", cErr.ExternalResponseFormatError("decode openai translation json failed")
	}
	return result.Text, nil
}

// ===== helpers =====

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func floatToStr(f float64) string {
	// 使用 json 去除多餘 0，避免 fmt 的科學記號等
	b, _ := json.Marshal(f)
	return string(b)
}

func newHTTPError(resp *http.Response, body []byte) error {
	// 詳細錯誤原因寫入 span，但對外回傳統一的 cErr（上層已處理）
	return fmt.Errorf("openai non-2xx: %s (%d) %s", resp.Status, resp.StatusCode, trimBody(body))
}

func trimBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 3000 {
		return s[:3000] + "..."
	}
	return s
}
