package authmetadata

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const (
	defaultJWTAccessSecret = "GPA"
	envJWTAccessSecret     = "JWT_ACCESS_SECRET"

	headerAuthorization = "Authorization"
	headerAuthUserID    = "X-Auth-User-Id"
	headerAuthRole      = "X-Auth-Role"
	headerAuthSID       = "X-Auth-Sid"
)

type accessClaims struct {
	UserID int64  `json:"user_id"`
	Role   int32  `json:"role"`
	SID    string `json:"sid,omitempty"`
	jwtv5.RegisteredClaims
}

func init() {
	middleware.Register("authmetadata", Middleware)
}

func Middleware(_ *config.Middleware) (middleware.Middleware, error) {
	secret := strings.TrimSpace(os.Getenv(envJWTAccessSecret))
	if secret == "" {
		secret = defaultJWTAccessSecret
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			sanitizeAuthHeaders(req.Header)

			token := extractBearerToken(req.Header.Get(headerAuthorization))
			req.Header.Del(headerAuthorization)
			if token == "" {
				return next.RoundTrip(req)
			}

			claims := &accessClaims{}
			parsed, err := jwtv5.ParseWithClaims(token, claims, func(token *jwtv5.Token) (any, error) {
				if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
					return nil, jwtv5.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !parsed.Valid || claims.UserID <= 0 {
				return unauthorized(req)
			}

			req.Header.Set(headerAuthUserID, strconv.FormatInt(claims.UserID, 10))
			req.Header.Set(headerAuthRole, strconv.FormatInt(int64(claims.Role), 10))
			if claims.SID != "" {
				req.Header.Set(headerAuthSID, claims.SID)
			}
			return next.RoundTrip(req)
		})
	}, nil
}

func sanitizeAuthHeaders(header http.Header) {
	header.Del(headerAuthUserID)
	header.Del(headerAuthRole)
	header.Del(headerAuthSID)
}

func extractBearerToken(authorization string) string {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authorization[7:])
}

func unauthorized(req *http.Request) (*http.Response, error) {
	body := `{"error":"UNAUTHENTICATED","message":"无效或过期的访问令牌"}`
	return &http.Response{
		StatusCode:    http.StatusUnauthorized,
		Status:        http.StatusText(http.StatusUnauthorized),
		Header:        http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:          io.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}, nil
}
