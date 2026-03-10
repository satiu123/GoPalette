package repository

import (
	"context"
	"time"
)

type TokenRepository interface {
	SetRefreshToken(ctx context.Context, token string, userID int64, duration time.Duration) error
	GetUserIDByRefreshToken(ctx context.Context, token string) (int64, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}
