//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/app/user/service/internal/biz"
	"github.com/satiu123/GoPalette/app/user/service/internal/conf"
	"github.com/satiu123/GoPalette/app/user/service/internal/data"
	"github.com/satiu123/GoPalette/app/user/service/internal/health"
	"github.com/satiu123/GoPalette/app/user/service/internal/server"
	"github.com/satiu123/GoPalette/app/user/service/internal/service"
	"github.com/satiu123/GoPalette/pkg/opentelemetry"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Auth, *conf.Registry, log.Logger, opentelemetry.ServiceName) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, health.ProviderSet, newApp, NewEtcdClient, NewRegistry))
}
