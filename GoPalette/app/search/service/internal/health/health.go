package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/app/search/service/internal/data"
)

var ProviderSet = wire.NewSet(NewMeilisearchCheckerWrapper, NewHealthWrapper)

// 桥接 Data 内部的 Meilisearch 实例
func NewMeilisearchCheckerWrapper(d *data.Data) *MeilisearchChecker {
	return NewMeilisearchChecker(d.Meili())
}

// 组合并生成总的 Health 检查器
func NewHealthWrapper(meili *MeilisearchChecker) *Health {
	return New(meili)
}

type Health struct {
	checkers []Checker
}

func New(cs ...Checker) *Health {
	return &Health{
		checkers: cs,
	}
}

func (h *Health) Check(ctx context.Context) error {
	var errs []error
	for _, c := range h.checkers {
		if err := c.Check(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name(), err))
		}
	}

	// errors.Join 如果传入空切片会安全返回 nil，否则合并所有非空错误
	return errors.Join(errs...)
}
