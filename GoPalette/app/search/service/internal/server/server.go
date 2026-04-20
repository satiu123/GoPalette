package server

import (
	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/pkg/opentelemetry"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, opentelemetry.NewRequestCounter, opentelemetry.NewSecondsHistogram)
