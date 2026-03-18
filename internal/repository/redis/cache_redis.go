package redis

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type cacheRedisRepository struct {
	rdb *goredis.Client
}

func NewCacheRedisRepository(rdb *goredis.Client) *cacheRedisRepository {
	return &cacheRedisRepository{rdb: rdb}
}

func (r *cacheRedisRepository) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}

func (r *cacheRedisRepository) SetJSON(ctx context.Context, key string, value any, ttlSeconds int) error {
	buf, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	return r.rdb.Set(ctx, key, buf, ttl).Err()
}

func (r *cacheRedisRepository) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.rdb.Del(ctx, keys...).Err()
}

func (r *cacheRedisRepository) GetInt64(ctx context.Context, key string) (int64, bool, error) {
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return 0, false, nil
		}
		return 0, false, err
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

func (r *cacheRedisRepository) SetInt64(ctx context.Context, key string, value int64, ttlSeconds int) error {
	ttl := time.Duration(ttlSeconds) * time.Second
	return r.rdb.Set(ctx, key, strconv.FormatInt(value, 10), ttl).Err()
}

func (r *cacheRedisRepository) Increment(ctx context.Context, key string) (int64, error) {
	return r.rdb.Incr(ctx, key).Result()
}
