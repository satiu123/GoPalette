package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	refreshTokenPrefix = "user:refresh_token:"
)

type tokenRedisRepository struct {
	rdb *redis.Client
}

func NewTokenRedisRepository(rdb *redis.Client) *tokenRedisRepository {
	return &tokenRedisRepository{rdb: rdb}
}

func tokenKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return refreshTokenPrefix + hex.EncodeToString(h[:])
}

func (r *tokenRedisRepository) SetRefreshToken(ctx context.Context, token string, userID int64, duration time.Duration) error {
	return r.rdb.Set(ctx, tokenKey(token), strconv.FormatInt(userID, 10), duration).Err()
}

func (r *tokenRedisRepository) GetUserIDByRefreshToken(ctx context.Context, token string) (int64, error) {
	userIDStr, err := r.rdb.Get(ctx, tokenKey(token)).Result()
	if err != nil {
		return 0, err
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *tokenRedisRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.rdb.Del(ctx, tokenKey(token)).Err()
}
