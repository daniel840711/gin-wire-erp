package telemetry

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewMetric, NewTrace)
