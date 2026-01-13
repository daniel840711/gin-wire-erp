//go:build wireinject
// +build wireinject

package main

import (
	"interchange/config"
	"interchange/internal/command"
	"interchange/internal/cron"
	"interchange/internal/database"
	"interchange/internal/handler"
	"interchange/internal/middleware"
	"interchange/internal/router"
	"interchange/internal/service"
	"interchange/internal/telemetry"

	"github.com/google/wire"
	"go.uber.org/zap"
)

// wireApp init application.
func wireApp(*config.Configuration, *zap.Logger) (*App, func(), error) {
	panic(
		wire.Build(
			database.ProviderSet,
			service.ProviderSet,
			handler.ProviderSet,
			middleware.ProviderSet,
			router.ProviderSet,
			cron.ProviderSet,
			newHttpServer,
			newHttpClient,
			telemetry.ProviderSet,
			newApp,
		),
	)
}

// wireCommand init application.
func wireCommand(*config.Configuration, *zap.Logger) (*command.Command, func(), error) {
	panic(wire.Build(command.ProviderSet))
}
