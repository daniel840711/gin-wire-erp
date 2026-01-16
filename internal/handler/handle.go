package handler

import (
	"interchange/internal/handler/proxy"

	"github.com/google/wire"
)

// ProviderSet Provider对象集合
var ProviderSet = wire.NewSet(
	NewAdminUserHandler,
	NewAdminUserAPIKeyHandler,
	proxy.NewChatHandler,
	proxy.NewAudioHandler,
	proxy.NewImageHandler,
	proxy.NewEmbeddingHandler,
	proxy.NewModelsHandler,
	NewProxyHandler,
	NewHealthHandler,
)
