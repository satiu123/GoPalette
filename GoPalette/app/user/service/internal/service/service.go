package service

import "github.com/euskadi31/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewUserService)
