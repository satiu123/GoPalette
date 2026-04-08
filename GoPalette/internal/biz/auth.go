package biz

import (
	pb "GoPalette/api/user/v1"
	"GoPalette/internal/conf"
	"GoPalette/internal/pkg/util"
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthClaims struct {
	UserID int64 `json:"user_id"`
	Role   int32 `json:"role"`
	jwt.RegisteredClaims
}

type AuthUsecase struct {
	authConf *conf.Auth
	userRepo UserRepo
}

func NewAuthUsecase(c *conf.Auth, userRepo UserRepo) *AuthUsecase {
	return &AuthUsecase{authConf: c, userRepo: userRepo}
}

func (uc *AuthUsecase) Login(email, password string) (string, string, error) {
	user, err := uc.userRepo.FindByEmail(context.Background(), email)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", pb.ErrorUserNotFound("用户 %s 不存在", email)
	}

	// 验证密码
	if !util.CheckPasswordHash(password, user.Password) {
		return "", "", pb.ErrorPasswordIncorrect("%s", "密码错误")
	}

	// 生成 JWT token
	accessToken, refreshToekn, err := uc.generateToken(user)
	if err != nil {
		return "", "", pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}

	return accessToken, refreshToekn, nil
}

func (uc *AuthUsecase) generateToken(user *User) (string, string, error) {
	t := time.Now()
	accessClaims := &AuthClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(t.Add(uc.authConf.JwtAccessExpire.AsDuration())),
			IssuedAt:  jwt.NewNumericDate(t),
			Issuer:    uc.authConf.Issuer,
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(uc.authConf.JwtAccessSecret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := &AuthClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(t.Add(uc.authConf.JwtRefreshExpire.AsDuration())),
			IssuedAt:  jwt.NewNumericDate(t),
			Issuer:    uc.authConf.Issuer,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(uc.authConf.JwtRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (uc *AuthUsecase) ParseToken(tokenStr, secret string) (*AuthClaims, error) {
	// 解析 JWT token
	token, err := jwt.ParseWithClaims(tokenStr, &AuthClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	// 验证 token 是否有效
	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, jwt.ErrInvalidKey
	}
}
func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := uc.ParseToken(refreshToken, uc.authConf.JwtRefreshSecret)
	if err != nil {
		return "", "", pb.ErrorTokenInvalid("%s", "无效的刷新令牌")
	}

	user, err := uc.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", pb.ErrorUserNotFound("用户 ID %d 不存在", claims.UserID)
	}

	return uc.generateToken(user)
}
