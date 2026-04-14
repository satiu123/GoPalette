//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/satiu123/GoPalette/search-service/internal/biz"
	"github.com/satiu123/GoPalette/search-service/internal/conf"
	"github.com/satiu123/GoPalette/search-service/internal/data"
	"github.com/satiu123/GoPalette/search-service/internal/server"
	"github.com/satiu123/GoPalette/search-service/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
