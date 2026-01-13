package images

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
	"net/textproto"
	"os"
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

// GenerateV1 呼叫 OpenAI /images/generations（JSON）。
// 失敗分類：
//   - 本地序列化/建請失敗：InternalServer
//   - 對外請求/非 2xx：ExternalRequestError
//   - 回應解析失敗：ExternalResponseFormatError
func (s *OpenAIService) GenerateV1(ctx context.Context, req *ImageGenerationRequestBody, apiKey string) (*ImageGenerationResponse, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1/" + string(core.OpenAIImageGenerateEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.images.generate")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) 序列化
	payload, err := json.Marshal(req)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("marshal image payload failed")
	}

	// 2) 建請
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create http request failed")
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// 3) 請求
	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		end(err)
		return nil, cErr.ExternalRequestError("openai image api request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// 4) 狀態碼
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		end(newHTTPError(resp, b))
		return nil, cErr.ExternalRequestError("openai image api error: " + trimBody(b))
	}

	// 5) 解析
	var result any
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode openai image response failed")
	}
	file, err := os.Create("text.txt")
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("failed to create file")
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(result); err != nil {
		end(err)
		return nil, cErr.InternalServer("failed to write result to file")
	}
	res, ok := result.(ImageGenerationResponse)
	if !ok {
		end(fmt.Errorf("unexpected response type"))
		return nil, cErr.ExternalResponseFormatError("unexpected response type")
	}
	return &res, nil
}

// EditV1 呼叫 OpenAI /images/edits（multipart）。
// 失敗分類同上；檔案處理/欄位寫入錯誤：InternalServer。
func (s *OpenAIService) EditV1(ctx context.Context, req *ImageEditRequestBody, apiKey string) (*ImageGenerationResponse, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1/" + string(core.OpenAIImageEditEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.images.edits")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// --- 基本檢核 ---
	if len(req.Images) == 0 {
		err := fmt.Errorf("at least one image is required")
		end(err)
		return nil, cErr.BadRequestBody(err.Error())
	}
	switch req.Model {
	case "dall-e-2":
		if len(req.Images) != 1 {
			err := fmt.Errorf("dall-e-2 only supports one image")
			end(err)
			return nil, cErr.BadRequestBody(err.Error())
		}
	case "gpt-image-1":
		if len(req.Images) > 16 {
			err := fmt.Errorf("gpt-image-1 supports up to 16 images")
			end(err)
			return nil, cErr.BadRequestBody(err.Error())
		}
	default:
		err := fmt.Errorf("unsupported model: %s", req.Model)
		end(err)
		return nil, cErr.BadRequestBody(err.Error())
	}

	// 1) multipart 組裝
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 圖片
	for i, fh := range req.Images {
		f, err := fh.Open()
		if err != nil {
			end(err)
			return nil, cErr.InternalServer(fmt.Sprintf("open image #%d failed", i+1))
		}
		defer f.Close()

		if !strings.HasSuffix(strings.ToLower(fh.Filename), ".png") {
			err := fmt.Errorf("only .png files are supported")
			end(err)
			return nil, cErr.BadRequestBody(err.Error())
		}

		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, fh.Filename))
		h.Set("Content-Type", "image/png")
		part, err := writer.CreatePart(h)
		if err != nil {
			end(err)
			return nil, cErr.InternalServer(fmt.Sprintf("create image part #%d failed", i+1))
		}
		if _, err := io.Copy(part, f); err != nil {
			end(err)
			return nil, cErr.InternalServer(fmt.Sprintf("copy image #%d failed", i+1))
		}
	}

	// mask（可選）
	if req.Mask != nil {
		mf, err := req.Mask.Open()
		if err != nil {
			end(err)
			return nil, cErr.InternalServer("open mask failed")
		}
		defer mf.Close()

		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="mask"; filename="%s"`, req.Mask.Filename))
		h.Set("Content-Type", "image/png")
		part, err := writer.CreatePart(h)
		if err != nil {
			end(err)
			return nil, cErr.InternalServer("create mask part failed")
		}
		if _, err := io.Copy(part, mf); err != nil {
			end(err)
			return nil, cErr.InternalServer("copy mask failed")
		}
	}

	// 其他欄位
	if err := writer.WriteField("prompt", req.Prompt); err != nil {
		end(err)
		return nil, cErr.InternalServer("write prompt field failed")
	}
	if err := writer.WriteField("model", string(req.Model)); err != nil {
		end(err)
		return nil, cErr.InternalServer("write model field failed")
	}
	if req.N > 0 {
		if err := writer.WriteField("n", fmt.Sprintf("%d", req.N)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write n field failed")
		}
	}
	if req.Size != "" {
		if err := writer.WriteField("size", string(req.Size)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write size field failed")
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", string(req.ResponseFormat)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write response_format field failed")
		}
	}
	if req.OutputFormat != "" {
		if err := writer.WriteField("output_format", string(req.OutputFormat)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write output_format field failed")
		}
	}
	if req.Quality != "" {
		if err := writer.WriteField("quality", string(req.Quality)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write quality field failed")
		}
	}
	if req.Background != "" {
		if err := writer.WriteField("background", req.Background); err != nil {
			end(err)
			return nil, cErr.InternalServer("write background field failed")
		}
	}
	if req.OutputCompression > 0 {
		if err := writer.WriteField("output_compression", fmt.Sprintf("%d", req.OutputCompression)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write output_compression field failed")
		}
	}
	if req.User != "" {
		if err := writer.WriteField("user", req.User); err != nil {
			end(err)
			return nil, cErr.InternalServer("write user field failed")
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
		return nil, cErr.ExternalRequestError("openai edit request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// 4) 狀態碼
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		end(newHTTPError(resp, b))
		return nil, cErr.ExternalRequestError("openai edit error: " + trimBody(b))
	}

	// 5) 解析
	var result ImageGenerationResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode edit response failed")
	}
	return &result, nil
}

// VariationV1 呼叫 OpenAI /images/variations（multipart）。
// 失敗分類同上；檔案處理/欄位寫入錯誤：InternalServer。
func (s *OpenAIService) VariationV1(ctx context.Context, req *ImageVariantRequestBody, apiKey string) (*ImageGenerationResponse, error) {
	url := string(core.OpenAIAPIBaseURL) + "/v1/" + string(core.OpenAIImageVariationEndpoint)
	ctx, span, end := s.trace.WithSpan(ctx, "openai.images.variations")
	defer end(nil)

	span.SetAttributes(
		attribute.String("ai.provider", "openai"),
		attribute.String("http.url", url),
	)

	// 1) multipart 組裝
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// image
	img, err := req.Image.Open()
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("open image failed")
	}
	defer img.Close()

	part, err := writer.CreateFormFile("image", req.Image.Filename)
	if err != nil {
		end(err)
		return nil, cErr.InternalServer("create form file failed")
	}
	if _, err := io.Copy(part, img); err != nil {
		end(err)
		return nil, cErr.InternalServer("copy image failed")
	}

	// fields
	if err := writer.WriteField("model", req.Model); err != nil {
		end(err)
		return nil, cErr.InternalServer("write model field failed")
	}
	if req.N > 0 {
		if err := writer.WriteField("n", fmt.Sprintf("%d", req.N)); err != nil {
			end(err)
			return nil, cErr.InternalServer("write n field failed")
		}
	}
	if req.Size != "" {
		if err := writer.WriteField("size", req.Size); err != nil {
			end(err)
			return nil, cErr.InternalServer("write size field failed")
		}
	}
	if req.ResponseFormat != "" {
		if err := writer.WriteField("response_format", req.ResponseFormat); err != nil {
			end(err)
			return nil, cErr.InternalServer("write response_format field failed")
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
		return nil, cErr.ExternalRequestError("openai variation request failed")
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// 4) 狀態碼
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		b, _ := io.ReadAll(resp.Body)
		end(newHTTPError(resp, b))
		return nil, cErr.ExternalRequestError("openai variation error: " + trimBody(b))
	}

	// 5) 解析
	var result ImageGenerationResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		end(err)
		return nil, cErr.ExternalResponseFormatError("decode openai variation response failed")
	}
	return &result, nil
}

// ===== helpers =====

func newHTTPError(resp *http.Response, body []byte) error {
	return fmt.Errorf("openai non-2xx: %s (%d) %s", resp.Status, resp.StatusCode, trimBody(body))
}

func trimBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 1000 {
		return s[:1000] + "..."
	}
	return s
}
