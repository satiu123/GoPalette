package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/satiu123/GoPalette/pkg/auth"
	"github.com/satiu123/GoPalette/pkg/id"

	pb "github.com/satiu123/GoPalette/api/user/v1"

	"github.com/satiu123/GoPalette/app/user/service/internal/conf"
	"github.com/satiu123/GoPalette/app/user/service/internal/pkg/util"

	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

type AuthSessionRepo interface {
	// 保存 refresh 会话: key = userID + sid, value = tokenHash, ttl = refresh ttl
	SaveRefreshSession(ctx context.Context, userID int64, sid string, tokenHash string, ttl time.Duration) error

	// 获取 refresh 会话哈希
	GetRefreshSessionHash(ctx context.Context, userID int64, sid string) (string, error)

	// 删除 refresh 会话
	DeleteRefreshSession(ctx context.Context, userID int64, sid string) error
}

type AuthUsecase struct {
	authConf    *conf.Auth
	userRepo    UserRepo
	sessionRepo AuthSessionRepo
	logger      *log.Helper
}

func NewAuthUsecase(c *conf.Auth, userRepo UserRepo, sessionRepo AuthSessionRepo, logger log.Logger) *AuthUsecase {
	return &AuthUsecase{
		authConf:    c,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		logger:      log.NewHelper(log.With(logger, "module", "usecase/auth")),
	}
}

func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (string, string, error) {
	uc.logger.WithContext(ctx).Infof("登录尝试: email=%s", email)

	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("查询用户失败: email=%s err=%v", email, err)
		return "", "", pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}
	if user == nil {
		uc.logger.WithContext(ctx).Warnf("登录失败: 用户不存在 email=%s", email)
		return "", "", pb.ErrorUserNotFound("用户 %s 不存在", email)
	}

	// 验证密码
	if !util.CheckPasswordHash(password, user.Password) {
		uc.logger.WithContext(ctx).Warnf("登录失败: 密码错误 user_id=%d email=%s", user.ID, email)
		return "", "", pb.ErrorPasswordIncorrect("%s", "密码错误")
	}

	sid := id.NewUUID() // 生成新的会话 ID
	// 生成 JWT token
	accessToken, refreshToken, err := uc.generateToken(user, sid)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("登录失败: 生成 token 失败 user_id=%d err=%v", user.ID, err)
		return "", "", pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}
	uc.logger.WithContext(ctx).Infof("登录成功: user_id=%d role=%d", user.ID, user.Role)

	// 保存 refresh token 会话
	if err := uc.sessionRepo.SaveRefreshSession(ctx, user.ID, sid, sha256Hex(refreshToken), uc.authConf.JwtRefreshExpire.AsDuration()); err != nil {
		uc.logger.WithContext(ctx).Errorf("保存 refresh 会话失败: user_id=%d err=%v", user.ID, err)
		return "", "", pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}

	return accessToken, refreshToken, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, userID int64) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return pb.ErrorUnauthenticated("未认证")
	}
	// 删除所有会话
	return uc.sessionRepo.DeleteRefreshSession(ctx, claims.UserID, claims.SID)
}

func (uc *AuthUsecase) generateToken(user *User, sid string) (string, string, error) {
	t := time.Now()
	accessClaims := &auth.AuthClaims{
		UserID: user.ID,
		Role:   user.Role,
		SID:    sid,
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
		SID:    sid,
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
		uc.logger.Warnf("token 解析失败: reason=%s err=%v", err.Error(), err)
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
		return "", "", pb.ErrorTokenInvalid("%s", "无效的刷新令牌")
	}
	if claims.SID == "" {
		return "", "", pb.ErrorTokenInvalid("%s", "刷新令牌缺少 sid")
	}

	// 验证 refresh token 是否在有效的会话中
	savedHash, err := uc.sessionRepo.GetRefreshSessionHash(ctx, claims.UserID, claims.SID)
	if err != nil {
		return "", "", pb.ErrorInternalServerError("%s", "服务器开小差了，请稍后再试")
	}
	if savedHash == "" || savedHash != sha256Hex(refreshToken) {
		return "", "", pb.ErrorTokenInvalid("%s", "刷新令牌无效或已过期")
	}

	user, err := uc.userRepo.Get(ctx, claims.UserID)
	if err != nil || user == nil {
		return "", "", pb.ErrorUserNotFound("用户 ID %d 不存在", claims.UserID)
	}

	// 轮转刷新令牌，删旧sid，发新sid
	_ = uc.sessionRepo.DeleteRefreshSession(ctx, claims.UserID, claims.SID)

	newSID := id.NewUUID()
	accessToken, newRefreshToken, err := uc.generateToken(user, newSID)
	if err != nil {
		return "", "", err
	}
	uc.logger.WithContext(ctx).Infof("刷新令牌成功: user_id=%d role=%d", user.ID, user.Role)

	return accessToken, newRefreshToken, nil
}

// CheckOwner 检查用户是否是资源的所有者
func CheckOwner(ctx context.Context, targetID int64) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return pb.ErrorUnauthenticated("未认证")
	}
	if claims.UserID != targetID && claims.Role != int32(pb.Role_ADMIN) {
		return pb.ErrorAccessDenied("无权限访问")
	}
	return nil
}

// CheckAdmin 检查用户是否具有管理员权限
func CheckAdmin(ctx context.Context) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return pb.ErrorUnauthenticated("未认证")
	}
	if claims.Role != int32(pb.Role_ADMIN) {
		return pb.ErrorAccessDenied("无权限访问")
	}
	return nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
