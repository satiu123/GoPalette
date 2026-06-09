package health

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisChecker struct {
	name string
	rdb  *redis.Client
}

func NewRedisChecker(rdb *redis.Client) *RedisChecker {
	return NewNamedRedisChecker("redis", rdb)
}

func NewNamedRedisChecker(name string, rdb *redis.Client) *RedisChecker {
	if name == "" {
		name = "redis"
	}
	return &RedisChecker{
		name: name,
		rdb:  rdb,
	}
}

func (c *RedisChecker) Name() string {
	return c.name
}

func (c *RedisChecker) Check(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
