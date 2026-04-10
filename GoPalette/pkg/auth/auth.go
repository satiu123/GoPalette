package auth

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

type AuthClaims struct {
	UserID int64  `json:"user_id"`
	Role   int32  `json:"role"`
	SID    string `json:"sid,omitempty"`
	jwtv5.RegisteredClaims
}

// FromContext 提取信息，返回原始数据
func FromContext(ctx context.Context) (*AuthClaims, bool) {
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, false
	}
	res, ok := claims.(*AuthClaims)
	return res, ok
}
