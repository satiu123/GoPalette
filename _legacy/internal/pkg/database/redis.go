package database

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/satiu123/GoPalette/internal/pkg/config"
)

func InitRedis() *redis.Client {
	cfg := config.GlobalConfig.Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic("failed to connect redis: " + err.Error())
	}
	return rdb
}
