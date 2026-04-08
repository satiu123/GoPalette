package biz

import (
	pb "GoPalette/api/user/v1"
	"GoPalette/internal/conf"
	"GoPalette/internal/pkg/util"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
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
	logger   *log.Helper
}

func NewAuthUsecase(c *conf.Auth, userRepo UserRepo, logger log.Logger) *AuthUsecase {
	return &AuthUsecase{
		authConf: c,
		userRepo: userRepo,
		logger:   log.NewHelper(log.With(logger, "module", "usecase/auth")),
	}
}

func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (string, string, error) {
	maskedEmail := maskEmail(email)
	uc.logger.WithContext(ctx).Infof("登录尝试: email=%s", maskedEmail)

	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("查询用户失败: email=%s err=%v", maskedEmail, err)
		return "", "", err
	}
	if user == nil {
		uc.logger.WithContext(ctx).Warnf("登录失败: 用户不存在 email=%s", maskedEmail)
		return "", "", pb.ErrorUserNotFound("用户 %s 不存在", email)
	}

	// 验证密码
	if !util.CheckPasswordHash(password, user.Password) {
		uc.logger.WithContext(ctx).Warnf("登录失败: 密码错误 user_id=%d email=%s", user.ID, maskedEmail)
		return "", "", pb.ErrorPasswordIncorrect("%s", "密码错误")
	}

	// 生成 JWT token
	accessToken, refreshToken, err := uc.generateToken(user)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("登录失败: 生成 token 失败 user_id=%d err=%v", user.ID, err)
		return "", "", pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}
	uc.logger.WithContext(ctx).Infof("登录成功: user_id=%d role=%d", user.ID, user.Role)

	return accessToken, refreshToken, nil
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
		uc.logger.Errorf("签发 access token 失败: user_id=%d err=%v", user.ID, err)
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
		uc.logger.Errorf("签发 refresh token 失败: user_id=%d err=%v", user.ID, err)
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
		uc.logger.Warnf("token 解析失败: reason=%s err=%v", classifyTokenError(err), err)
		return nil, err
	}
	// 验证 token 是否有效
	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	} else {
		uc.logger.Warn("token 校验失败: claims 无效或 token 不合法")
		return nil, jwt.ErrInvalidKey
	}
}
func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := uc.ParseToken(refreshToken, uc.authConf.JwtRefreshSecret)
	if err != nil {
		uc.logger.WithContext(ctx).Warnf("刷新令牌失败: refresh token 无效 err=%v", err)
		return "", "", pb.ErrorTokenInvalid("%s", "无效的刷新令牌")
	}

	user, err := uc.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("刷新令牌失败: 查询用户异常 user_id=%d err=%v", claims.UserID, err)
		return "", "", err
	}
	if user == nil {
		uc.logger.WithContext(ctx).Warnf("刷新令牌失败: 用户不存在 user_id=%d", claims.UserID)
		return "", "", pb.ErrorUserNotFound("用户 ID %d 不存在", claims.UserID)
	}

	accessToken, newRefreshToken, err := uc.generateToken(user)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("刷新令牌失败: 生成新 token 异常 user_id=%d err=%v", user.ID, err)
		return "", "", err
	}
	uc.logger.WithContext(ctx).Infof("刷新令牌成功: user_id=%d role=%d", user.ID, user.Role)

	return accessToken, newRefreshToken, nil
}

func classifyTokenError(err error) string {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return "expired"
	case errors.Is(err, jwt.ErrSignatureInvalid):
		return "signature_invalid"
	case errors.Is(err, jwt.ErrTokenMalformed):
		return "malformed"
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return "not_valid_yet"
	default:
		return "other"
	}
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}

	local, domain := parts[0], parts[1]
	if len(local) <= 2 {
		return "**@" + domain
	}

	return local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:] + "@" + domain
}
