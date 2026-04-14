package data

import (
	"context"
	"fmt"
	"time"

	"github.com/satiu123/GoPalette/comment-service/internal/biz"
)

type rateLimitRepo struct {
	data *Data
}

func NewRateLimitRepo(data *Data) biz.RateLimitRepo {
	return &rateLimitRepo{data: data}
}

func (r *rateLimitRepo) AllowCreate(ctx context.Context, userID int64, limit int64, window time.Duration) (bool, error) {
	key := fmt.Sprintf("comment:rate:%d", userID)
	count, err := r.data.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if count == 1 {
		if err := r.data.rdb.Expire(ctx, key, window).Err(); err != nil {
			return true, err
		}
	}
	return count <= limit, nil
}
