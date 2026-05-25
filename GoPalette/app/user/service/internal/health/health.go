package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/app/user/service/internal/data"
)

var ProviderSet = wire.NewSet(New)

type Health struct {
	checkers []Checker
}

func New(d *data.Data) *Health {
	h := &Health{}

	if d.DB() != nil {
		h.checkers = append(
			h.checkers,
			NewMySQLChecker(d.DB()),
		)
	}

	if d.Redis() != nil {
		h.checkers = append(
			h.checkers,
			NewRedisChecker(d.Redis()),
		)
	}

	return h
}

func (h *Health) Check(
	ctx context.Context,
) error {
	var errs []error

	for _, c := range h.checkers {
		if err := c.Check(ctx); err != nil {
			errs = append(
				errs,
				fmt.Errorf("%s: %w", c.Name(), err),
			)
		}
	}

	return errors.Join(errs...)
}
