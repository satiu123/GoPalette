package biz

import (
	pb "GoPalette/api/user/v1"
	"GoPalette/app/user/service/internal/conf"
	"GoPalette/app/user/service/internal/pkg/util"
	"GoPalette/pkg/auth"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

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
	accessClaims := &auth.AuthClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(t.Add(uc.authConf.JwtAccessExpire.AsDuration())),
			IssuedAt:  jwtv5.NewNumericDate(t),
			Issuer:    uc.authConf.Issuer,
		},
	}
	accessToken, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, accessClaims).SignedString([]byte(uc.authConf.JwtAccessSecret))
	if err != nil {
		uc.logger.Errorf("签发 access token 失败: user_id=%d err=%v", user.ID, err)
		return "", "", err
	}

	refreshClaims := &auth.AuthClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(t.Add(uc.authConf.JwtRefreshExpire.AsDuration())),
			IssuedAt:  jwtv5.NewNumericDate(t),
			Issuer:    uc.authConf.Issuer,
		},
	}
	refreshToken, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, refreshClaims).SignedString([]byte(uc.authConf.JwtRefreshSecret))
	if err != nil {
		uc.logger.Errorf("签发 refresh token 失败: user_id=%d err=%v", user.ID, err)
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (uc *AuthUsecase) ParseToken(tokenStr, secret string) (*auth.AuthClaims, error) {
	// 解析 JWT token
	token, err := jwtv5.ParseWithClaims(tokenStr, &auth.AuthClaims{}, func(token *jwtv5.Token) (any, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, jwtv5.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		uc.logger.Warnf("token 解析失败: reason=%s err=%v", classifyTokenError(err), err)
		return nil, err
	}
	// 验证 token 是否有效
	if claims, ok := token.Claims.(*auth.AuthClaims); ok && token.Valid {
		return claims, nil
	} else {
		uc.logger.Warn("token 校验失败: claims 无效或 token 不合法")
		return nil, jwtv5.ErrInvalidKey
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

// FromContext 封装获取 Claims 的逻辑
func FromContext(ctx context.Context) (*auth.AuthClaims, error) {
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, pb.ErrorUnauthenticated("未认证的请求")
	}
	res, ok := claims.(*auth.AuthClaims)
	if !ok {
		return nil, pb.ErrorUnauthenticated("非法的 Token 格式")
	}
	return res, nil
}

// CheckOwner 检查用户是否是资源的所有者
func CheckOwner(ctx context.Context, targetID int64) error {
	claims, err := FromContext(ctx)
	if err != nil {
		return err
	}
	if claims.UserID != targetID && claims.Role != int32(pb.Role_ROLE_ADMIN) {
		return pb.ErrorAccessDenied("无权限访问")
	}
	return nil
}

// CheckAdmin 检查用户是否具有管理员权限
func CheckAdmin(ctx context.Context) error {
	claims, err := FromContext(ctx)
	if err != nil {
		return err
	}
	if claims.Role != int32(pb.Role_ROLE_ADMIN) {
		return pb.ErrorAccessDenied("无权限访问")
	}
	return nil
}

func classifyTokenError(err error) string {
	switch {
	case errors.Is(err, jwtv5.ErrTokenExpired):
		return "expired"
	case errors.Is(err, jwtv5.ErrSignatureInvalid):
		return "signature_invalid"
	case errors.Is(err, jwtv5.ErrTokenMalformed):
		return "malformed"
	case errors.Is(err, jwtv5.ErrTokenNotValidYet):
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
