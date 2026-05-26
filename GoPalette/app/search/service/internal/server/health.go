package server

import (
	"github.com/satiu123/GoPalette/app/search/service/internal/data"
	"github.com/satiu123/GoPalette/pkg/health"
)

// 手动组装 Checkers
func NewHealthEngine(d *data.Data) *health.Health {
	return health.New(
		health.NewMeilisearchChecker(d.Meili()),
	)
}
