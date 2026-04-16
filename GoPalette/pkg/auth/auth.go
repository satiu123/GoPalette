package auth

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const (
	MetadataUserIDKey = "x-auth-user-id"
	MetadataRoleKey   = "x-auth-role"
	MetadataSIDKey    = "x-auth-sid"
)

type AuthClaims struct {
	UserID int64  `json:"user_id"`
	Role   int32  `json:"role"`
	SID    string `json:"sid,omitempty"`
	jwtv5.RegisteredClaims
}

// FromContext 提取认证信息，优先使用 jwt middleware 注入的 claims，
// 再回退到从服务端 metadata 中读取网关透传字段。
func FromContext(ctx context.Context) (*AuthClaims, bool) {
	claims, ok := jwt.FromContext(ctx)
	if ok {
		res, ok := claims.(*AuthClaims)
		if ok {
			return res, true
		}
	}

	uidText, roleText, sidText, ok := authFieldsFromServerContext(ctx)
	if !ok {
		return nil, false
	}
	if uidText == "" {
		return nil, false
	}
	userID, err := strconv.ParseInt(uidText, 10, 64)
	if err != nil || userID <= 0 {
		return nil, false
	}

	role := int32(0)
	if roleText != "" {
		roleParsed, err := strconv.ParseInt(roleText, 10, 32)
		if err != nil {
			return nil, false
		}
		role = int32(roleParsed)
	}

	return &AuthClaims{
		UserID: userID,
		Role:   role,
		SID:    sidText,
	}, true
}

// ForwardMetadataToClientContext 将当前服务端上下文中的认证 metadata 透传到客户端上下文，
// 便于 BFF 继续调用下游微服务。
func ForwardMetadataToClientContext(ctx context.Context) context.Context {
	userID, role, sid, ok := authFieldsFromServerContext(ctx)
	if !ok {
		return ctx
	}

	clientMD := metadata.Metadata{}
	if userID != "" {
		clientMD.Set(MetadataUserIDKey, userID)
	}
	if role != "" {
		clientMD.Set(MetadataRoleKey, role)
	}
	if sid != "" {
		clientMD.Set(MetadataSIDKey, sid)
	}

	if len(clientMD) == 0 {
		return ctx
	}
	return metadata.MergeToClientContext(ctx, clientMD)
}

func authFieldsFromServerContext(ctx context.Context) (userID, role, sid string, ok bool) {
	if md, has := metadata.FromServerContext(ctx); has {
		userID = strings.TrimSpace(md.Get(MetadataUserIDKey))
		role = strings.TrimSpace(md.Get(MetadataRoleKey))
		sid = strings.TrimSpace(md.Get(MetadataSIDKey))
		if userID != "" {
			return userID, role, sid, true
		}
	}

	tr, has := transport.FromServerContext(ctx)
	if !has {
		return "", "", "", false
	}
	h := tr.RequestHeader()
	if h == nil {
		return "", "", "", false
	}
	userID = strings.TrimSpace(h.Get(MetadataUserIDKey))
	role = strings.TrimSpace(h.Get(MetadataRoleKey))
	sid = strings.TrimSpace(h.Get(MetadataSIDKey))
	if userID == "" {
		return "", "", "", false
	}
	return userID, role, sid, true
}

// Server 基于 metadata claims 执行鉴权（一般配合 selector 白名单使用）。
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if _, ok := FromContext(ctx); !ok {
				return nil, errors.Unauthorized("UNAUTHENTICATED", "未认证")
			}
			return handler(ctx, req)
		}
	}
}
