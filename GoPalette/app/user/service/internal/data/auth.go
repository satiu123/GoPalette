package data

import (
	"GoPalette/app/user/service/internal/biz"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type authSessionRepo struct {
	data *Data
}

func NewAuthSessionRepo(data *Data) biz.AuthSessionRepo {
	return &authSessionRepo{data: data}
}

func refreshKey(userID int64, sid string) string {
	return fmt.Sprintf("auth:refresh:%d:%s", userID, sid)
}

func (r *authSessionRepo) SaveRefreshSession(ctx context.Context, userID int64, sid string, tokenHash string, ttl time.Duration) error {
	return r.data.rdb.Set(ctx, refreshKey(userID, sid), tokenHash, ttl).Err()
}

func (r *authSessionRepo) GetRefreshSessionHash(ctx context.Context, userID int64, sid string) (string, error) {
	val, err := r.data.rdb.Get(ctx, refreshKey(userID, sid)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // session 不存在
		}
		return "", err
	}
	return val, nil
}

func (r *authSessionRepo) DeleteRefreshSession(ctx context.Context, userID int64, sid string) error {
	err := r.data.rdb.Del(ctx, refreshKey(userID, sid)).Err()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}
