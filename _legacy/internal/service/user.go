package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/satiu123/GoPalette/internal/model"
	"github.com/satiu123/GoPalette/internal/pkg/config"
	"github.com/satiu123/GoPalette/internal/pkg/jwt"
	"github.com/satiu123/GoPalette/internal/repository"
)

type UserService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
}

func NewUserService(rp repository.UserRepository, tp repository.TokenRepository) *UserService {
	return &UserService{
		userRepo:  rp,
		tokenRepo: tp,
	}
}

// Login 验证用户凭据并生成 Access Token 和 Refresh Token
func (s *UserService) Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return "", "", err
	}
	if user == nil || !user.CheckPassword(password) {
		return "", "", errors.New("用户名或密码错误")
	}
	cfg := config.GlobalConfig.JWT
	accessToken, refreshToken, err = jwt.GenerateToken(int(user.ID), user.Role, cfg)
	if err != nil {
		return "", "", err
	}
	if err = s.tokenRepo.SetRefreshToken(ctx, refreshToken, int64(user.ID), cfg.RefreshTokenTTL); err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (s *UserService) Register(ctx context.Context, username, password, role string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &model.User{
		Username: username,
		Password: string(hashed),
		Role:     role,
	}
	return s.userRepo.Create(ctx, user)
}

// GetByID 按 ID 查询用户（用于 /users/me 接口）
func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// UpdateUser 修改用户信息（用户名、头像或密码），修改密码时需验证旧密码
func (s *UserService) UpdateUser(ctx context.Context, id int64, username, avatarURL, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if username != "" {
		user.Username = username
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}
	if newPassword != "" {
		if !user.CheckPassword(oldPassword) {
			return errors.New("旧密码不正确")
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.Password = string(hashed)
	}
	return s.userRepo.Update(ctx, user)
}

// Refresh 验证 Refresh Token，撤销旧 token，签发新的 Access Token 和 Refresh Token
func (s *UserService) Refresh(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error) {
	cfg := config.GlobalConfig.JWT
	claims, err := jwt.ParseToken(refreshToken, cfg.RefreshTokenSecret)
	if err != nil {
		return "", "", err
	}
	// 校验 token 是否存在于 Redis（防止重放/已撤销）
	if _, err = s.tokenRepo.GetUserIDByRefreshToken(ctx, refreshToken); err != nil {
		return "", "", errors.New("refresh token 已失效")
	}
	// 撤销旧 token（轮转）
	if err = s.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return "", "", err
	}
	// 签发新 token 并存储
	newAccess, newRefresh, err = jwt.GenerateToken(claims.UserID, claims.Role, cfg)
	if err != nil {
		return "", "", err
	}
	if err = s.tokenRepo.SetRefreshToken(ctx, newRefresh, int64(claims.UserID), cfg.RefreshTokenTTL); err != nil {
		return "", "", err
	}
	return newAccess, newRefresh, nil
}
