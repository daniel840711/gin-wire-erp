package middleware

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewCors,
	NewLogger,
	NewRecovery,
	// NewTraceEntry,
	// NewAPIKey,
	// NewRateLimit,
	NewResponse,
	// NewUser,
)
