package biz

import "github.com/euskadi31/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewPostUsecase, NewCategoryUsecase, NewTagUsecase)
