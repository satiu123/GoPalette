package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/pkg/config"
)

type fakeUserRepo struct {
	findByUsernameFn func(ctx context.Context, username string) (*model.User, error)
	createFn         func(ctx context.Context, user *model.User) error
	findByIDFn       func(ctx context.Context, id int64) (*model.User, error)
	updateFn         func(ctx context.Context, user *model.User) error
}

func (f *fakeUserRepo) Create(ctx context.Context, user *model.User) error {
	if f.createFn != nil {
		return f.createFn(ctx, user)
	}
	return nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	if f.findByUsernameFn != nil {
		return f.findByUsernameFn(ctx, username)
	}
	return nil, nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user *model.User) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, user)
	}
	return nil
}

type fakeTokenRepo struct {
	setFn    func(ctx context.Context, token string, userID int64, duration time.Duration) error
	getFn    func(ctx context.Context, token string) (int64, error)
	deleteFn func(ctx context.Context, token string) error
}

func (f *fakeTokenRepo) SetRefreshToken(ctx context.Context, token string, userID int64, duration time.Duration) error {
	if f.setFn != nil {
		return f.setFn(ctx, token, userID, duration)
	}
	return nil
}

func (f *fakeTokenRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (int64, error) {
	if f.getFn != nil {
		return f.getFn(ctx, token)
	}
	return 0, errors.New("not found")
}

func (f *fakeTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, token)
	}
	return nil
}

func TestUserServiceLoginSuccess(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	config.GlobalConfig = &config.Config{JWT: config.JWTConfig{
		AccessTokenSecret:  "test-access-secret",
		RefreshTokenSecret: "test-refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
	}}

	stored := false
	service := NewUserService(
		&fakeUserRepo{
			findByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return &model.User{ID: 42, Username: username, Password: string(hash), Role: "user"}, nil
			},
		},
		&fakeTokenRepo{
			setFn: func(ctx context.Context, token string, userID int64, duration time.Duration) error {
				stored = token != "" && userID == 42 && duration == 24*time.Hour
				return nil
			},
		},
	)

	accessToken, refreshToken, err := service.Login(context.Background(), "alice", "123456")
	if err != nil {
		t.Fatalf("login should succeed, got err: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("tokens should not be empty")
	}
	if !stored {
		t.Fatal("refresh token should be stored")
	}
}

func TestUserServiceLoginInvalidPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	config.GlobalConfig = &config.Config{JWT: config.JWTConfig{
		AccessTokenSecret:  "test-access-secret",
		RefreshTokenSecret: "test-refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
	}}

	service := NewUserService(
		&fakeUserRepo{
			findByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
				return &model.User{ID: 1, Username: username, Password: string(hash), Role: "user"}, nil
			},
		},
		&fakeTokenRepo{},
	)

	_, _, err = service.Login(context.Background(), "alice", "wrong")
	if err == nil {
		t.Fatal("login should fail on wrong password")
	}
}
