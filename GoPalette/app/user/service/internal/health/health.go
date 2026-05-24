package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/euskadi31/wire"
	"github.com/satiu123/GoPalette/app/user/service/internal/data"
)

var ProviderSet = wire.NewSet(NewMySQLCheckerWrapper, NewRedisCheckerWrapper, NewHealthWrapper)

// 桥接 Data 内部的 DB 实例
func NewMySQLCheckerWrapper(d *data.Data) *MySQLChecker {
	return NewMySQLChecker(d.DB())
}

// 桥接 Data 内部的 Redis 实例
func NewRedisCheckerWrapper(d *data.Data) *RedisChecker {
	return NewRedisChecker(d.Redis())
}

// 组合并生成总的 Health 检查器
func NewHealthWrapper(mysql *MySQLChecker, redis *RedisChecker) *Health {
	return New(mysql, redis)
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
