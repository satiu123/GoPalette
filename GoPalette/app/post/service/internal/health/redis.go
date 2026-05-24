package health

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisChecker struct {
	rdb *redis.Client
}

func NewRedisChecker(rdb *redis.Client) *RedisChecker {
	return &RedisChecker{
		rdb: rdb,
	}
}

func (c *RedisChecker) Name() string {
	return "redis"
}

func (c *RedisChecker) Check(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
