package auth

import "github.com/golang-jwt/jwt/v5"

type AuthClaims struct {
	UserID int64 `json:"user_id"`
	Role   int32 `json:"role"`
	jwt.RegisteredClaims
}
