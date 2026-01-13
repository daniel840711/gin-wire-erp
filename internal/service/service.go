package service

import (
	"interchange/internal/core"
	"interchange/internal/service/audio"
	"interchange/internal/service/chat"
	"interchange/internal/service/embedding"
	"interchange/internal/service/images"
	"interchange/internal/service/models"

	"github.com/google/wire"
)

// 這裡只用一個 provider 實例化 Registry 並同時註冊
var ProviderSet = wire.NewSet(
	NewUserService,
	NewUserAPIKeyService,
	chat.NewOpenAIService,
	images.NewOpenAIService,
	audio.NewOpenAIService,
	embedding.NewOpenAIService,
	models.NewOpenAIService,
	NewProxyService,
	ProvideRegistryWithServices,
)

// ProvideRegistryWithServices
func ProvideRegistryWithServices(
	openAIChat chat.Service,
	openAIImages images.Service,
	openAIAudio audio.Service,
	openAIEmbedding embedding.Service,
	openAIModels models.Service,
) *Registry {
	reg := &Registry{
		ChatServices:     make(map[core.ProviderName]chat.Service),
		ImagesServices:   make(map[core.ProviderName]images.Service),
		AudioServices:    make(map[core.ProviderName]audio.Service),
		EmbeddingService: make(map[core.ProviderName]embedding.Service),
		ModelsServices:   make(map[core.ProviderName]models.Service),
	}
	reg.RegisterChat(core.ProviderOpenAI, openAIChat)
	reg.RegisterImages(core.ProviderOpenAI, openAIImages)
	reg.RegisterAudio(core.ProviderOpenAI, openAIAudio)
	reg.RegisterEmbedding(core.ProviderOpenAI, openAIEmbedding)
	reg.RegisterModels(core.ProviderOpenAI, openAIModels)
	return reg
}
