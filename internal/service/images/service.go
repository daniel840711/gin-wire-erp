package images

import (
	"context"
	"mime/multipart"
)

// 圖片生成 Model
type ImageModel string

const (
	ImageModelDallE2    ImageModel = "dall-e-2"
	ImageModelDallE3    ImageModel = "dall-e-3"
	ImageModelGptImage1 ImageModel = "gpt-image-1"
)

// 圖片尺寸
type ImageSize string

const (
	ImageSize256       ImageSize = "256x256"
	ImageSize512       ImageSize = "512x512"
	ImageSize1024      ImageSize = "1024x1024"
	ImageSize1536x1024 ImageSize = "1536x1024"
	ImageSize1024x1536 ImageSize = "1024x1536"
	ImageSize1792x1024 ImageSize = "1792x1024"
	ImageSize1024x1792 ImageSize = "1024x1792"
	ImageSizeAuto      ImageSize = "auto"
)

// 圖片品質
type ImageQuality string

const (
	ImageQualityAuto     ImageQuality = "auto"
	ImageQualityHigh     ImageQuality = "high"
	ImageQualityMedium   ImageQuality = "medium"
	ImageQualityLow      ImageQuality = "low"
	ImageQualityHD       ImageQuality = "hd"
	ImageQualityStandard ImageQuality = "standard"
)

// 圖片風格
type ImageStyle string

const (
	ImageStyleVivid   ImageStyle = "vivid"
	ImageStyleNatural ImageStyle = "natural"
)

// 回應格式
type ImageResponseFormat string

const (
	ImageResponseURL     ImageResponseFormat = "url"
	ImageResponseB64Json ImageResponseFormat = "b64_json"
)

// Output Format
type ImageOutputFormat string

const (
	ImageOutputPNG  ImageOutputFormat = "png"
	ImageOutputJPEG ImageOutputFormat = "jpeg"
	ImageOutputWebP ImageOutputFormat = "webp"
)

type ImageGenerationRequestBody struct {
	Model          ImageModel          `json:"model"`
	Prompt         string              `json:"prompt"`
	N              int                 `json:"n"`
	Size           ImageSize           `json:"size"`
	Quality        ImageQuality        `json:"quality"`
	Style          ImageStyle          `json:"style"`
	ResponseFormat ImageResponseFormat `json:"response_format"`
}

type ImageVariantRequestBody struct {
	Model          string                `form:"model" binding:"required"`
	Image          *multipart.FileHeader `form:"image" binding:"required"`
	N              int                   `form:"n" binding:"required"`
	Size           string                `form:"size" binding:"required"`
	ResponseFormat string                `form:"response_format" binding:"required"`
}
type ImageEditRequestBody struct {
	Model             ImageModel              `form:"model" binding:"required"`
	Prompt            string                  `form:"prompt" binding:"required"`
	Images            []*multipart.FileHeader `form:"image" binding:"required"`
	Mask              *multipart.FileHeader   `form:"mask,omitempty"`
	Background        string                  `form:"background,omitempty"`
	N                 int                     `form:"n,omitempty"`
	OutputCompression int                     `form:"output_compression,omitempty"`
	OutputFormat      ImageOutputFormat       `form:"output_format,omitempty"`
	Quality           ImageQuality            `form:"quality,omitempty"`
	ResponseFormat    ImageResponseFormat     `form:"response_format,omitempty"`
	Size              ImageSize               `form:"size,omitempty"`
	User              string                  `form:"user,omitempty"`
}
type ImageGenerationResponse struct {
	Background   string                `json:"background,omitempty"`    // "transparent" | "opaque"
	Created      int64                 `json:"created"`                 // unix timestamp (秒)
	Data         []ImageGenerationData `json:"data"`                    // 圖片列表
	OutputFormat ImageOutputFormat     `json:"output_format,omitempty"` // "png", "jpeg", "webp"
	Quality      ImageQuality          `json:"quality,omitempty"`       // "low", "medium", "high"
	Size         ImageSize             `json:"size,omitempty"`          // "1024x1024"...
	Usage        *ImageGenerationUsage `json:"usage,omitempty"`         // token usage info (GPT-Image-1 only)
}

type ImageGenerationData struct {
	B64Json       string `json:"b64_json,omitempty"`       // base64-encoded image
	RevisedPrompt string `json:"revised_prompt,omitempty"` // dall-e-3 only
	URL           string `json:"url,omitempty"`            // url for images (dall-e-2, dall-e-3)
}

type ImageGenerationUsage struct {
	InputTokens        int                               `json:"input_tokens"`
	InputTokensDetails *ImageGenerationInputTokensDetail `json:"input_tokens_details,omitempty"`
	OutputTokens       int                               `json:"output_tokens"`
	TotalTokens        int                               `json:"total_tokens"`
}

type ImageGenerationInputTokensDetail struct {
	ImageTokens int `json:"image_tokens"`
	TextTokens  int `json:"text_tokens"`
}
type Service interface {
	// 生成圖片
	GenerateV1(ctx context.Context, req *ImageGenerationRequestBody, key string) (*ImageGenerationResponse, error)
	// 圖片編輯
	EditV1(ctx context.Context, req *ImageEditRequestBody, key string) (*ImageGenerationResponse, error)
	// 產生變體
	VariationV1(ctx context.Context, req *ImageVariantRequestBody, key string) (*ImageGenerationResponse, error)
}
