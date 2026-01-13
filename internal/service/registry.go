package service

import (
	"interchange/internal/core"
	"interchange/internal/service/audio"
	"interchange/internal/service/chat"
	"interchange/internal/service/embedding"
	"interchange/internal/service/images"
	"interchange/internal/service/models"
)

type Registry struct {
	ChatServices     map[core.ProviderName]chat.Service
	ImagesServices   map[core.ProviderName]images.Service
	AudioServices    map[core.ProviderName]audio.Service
	EmbeddingService map[core.ProviderName]embedding.Service
	ModelsServices   map[core.ProviderName]models.Service
}

func (r *Registry) RegisterChat(provider core.ProviderName, service chat.Service) {
	r.ChatServices[provider] = service
}
func (r *Registry) GetChat(provider core.ProviderName) (chat.Service, bool) {
	svc, ok := r.ChatServices[provider]
	return svc, ok
}

func (r *Registry) RegisterImages(provider core.ProviderName, service images.Service) {
	r.ImagesServices[provider] = service
}
func (r *Registry) GetImages(provider core.ProviderName) (images.Service, bool) {
	svc, ok := r.ImagesServices[provider]
	return svc, ok
}
func (r *Registry) RegisterAudio(provider core.ProviderName, service audio.Service) {
	r.AudioServices[provider] = service
}
func (r *Registry) GetAudio(provider core.ProviderName) (audio.Service, bool) {
	svc, ok := r.AudioServices[provider]
	return svc, ok
}

func (r *Registry) RegisterEmbedding(provider core.ProviderName, service embedding.Service) {
	r.EmbeddingService[provider] = service
}
func (r *Registry) GetEmbedding(provider core.ProviderName) (embedding.Service, bool) {
	svc, ok := r.EmbeddingService[provider]
	return svc, ok
}
func (r *Registry) RegisterModels(provider core.ProviderName, service models.Service) {
	r.ModelsServices[provider] = service
}
func (r *Registry) GetModels(provider core.ProviderName) (models.Service, bool) {
	svc, ok := r.ModelsServices[provider]
	return svc, ok
}
