package router

import (
	"interchange/internal/handler"
	"interchange/internal/handler/proxy"
	"interchange/internal/middleware"

	"github.com/gin-gonic/gin"
)

type ProxyRouter struct {
	chatHandler         *proxy.ChatHandler
	imageHandler        *proxy.ImageHandler
	audioHandler        *proxy.AudioHandler
	embeddingHandler    *proxy.EmbeddingHandler
	modelHandler        *proxy.ModelsHandler
	proxyHandler        *handler.ProxyHandler
	apiKeyMiddleware    *middleware.APIKey
	ratelimitMiddleware *middleware.RateLimit
	userMiddleware      *middleware.User
}

func NewProxyRouter(
	chatHandler *proxy.ChatHandler,
	imageHandler *proxy.ImageHandler,
	audioHandler *proxy.AudioHandler,
	embeddingHandler *proxy.EmbeddingHandler,
	modelHandler *proxy.ModelsHandler,
	proxyHandler *handler.ProxyHandler,
	apiKeyMiddleware *middleware.APIKey,
	ratelimitMiddleware *middleware.RateLimit,
	userMiddleware *middleware.User,
) *ProxyRouter {
	return &ProxyRouter{
		chatHandler:         chatHandler,
		imageHandler:        imageHandler,
		audioHandler:        audioHandler,
		embeddingHandler:    embeddingHandler,
		modelHandler:        modelHandler,
		proxyHandler:        proxyHandler,
		apiKeyMiddleware:    apiKeyMiddleware,
		ratelimitMiddleware: ratelimitMiddleware,
		userMiddleware:      userMiddleware,
	}
}

func (proxyRouter *ProxyRouter) RegisterRoutes(engine *gin.Engine) {
	router := engine.Group("/proxy/:version/:provider")
	router.Use(proxyRouter.apiKeyMiddleware.Handler())
	router.Use(proxyRouter.userMiddleware.Handler())
	router.Use(proxyRouter.ratelimitMiddleware.Guard())

	chat := router.Group("/chat")
	{
		chat.POST("/completions", proxyRouter.chatHandler.ChatCompletions)
	}

	image := router.Group("/images")
	{
		image.POST("/generations", proxyRouter.imageHandler.ImagesGenerations)
		image.POST("/variations", proxyRouter.imageHandler.ImagesVariations)
		image.POST("/edits", proxyRouter.imageHandler.ImagesEdits)
	}

	audio := router.Group("/audio")
	{
		audio.POST("/transcriptions", proxyRouter.audioHandler.AudioTranscriptions)
		audio.POST("/translations", proxyRouter.audioHandler.AudioTranslations)
		audio.POST("/speech", proxyRouter.audioHandler.AudioSpeech)
	}
	router.GET("/models", proxyRouter.modelHandler.ListModels)
	router.POST("/embeddings", proxyRouter.embeddingHandler.GenerateEmbedding)
}
