package repository

import "context"

// CacheRepository 定义通用缓存操作，便于 service 层实现热点缓存。
type CacheRepository interface {
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSON(ctx context.Context, key string, value any, ttlSeconds int) error
	Delete(ctx context.Context, keys ...string) error
	GetInt64(ctx context.Context, key string) (int64, bool, error)
	SetInt64(ctx context.Context, key string, value int64, ttlSeconds int) error
	Increment(ctx context.Context, key string) (int64, error)
}
