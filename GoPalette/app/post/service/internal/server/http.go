package server

import (
	"context"

	"github.com/satiu123/GoPalette/pkg/auth"

	p "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/post-service/internal/conf"
	"github.com/satiu123/GoPalette/post-service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := make(map[string]struct{})
	whiteList["/api.post.v1.Post/GetPost"] = struct{}{}
	whiteList["/api.post.v1.Post/ListPosts"] = struct{}{}
	whiteList["/api.post.v1.Post/ListPostsForIndex"] = struct{}{}
	whiteList["/api.post.v1.Post/IncrCommentCount"] = struct{}{}
	whiteList["/api.post.v1.Category/GetCategory"] = struct{}{}
	whiteList["/api.post.v1.Category/ListCategories"] = struct{}{}
	whiteList["/api.post.v1.Tag/GetTag"] = struct{}{}
	whiteList["/api.post.v1.Tag/ListTags"] = struct{}{}
	return func(ctx context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, ca *conf.Auth, post *service.PostService, category *service.CategoryService, tag *service.TagService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			tracing.Server(),
			logging.Server(logger),
			selector.Server(
				jwt.Server(func(token *jwtv5.Token) (any, error) {
					return []byte(ca.JwtAccessSecret), nil
				}, jwt.WithClaims(func() jwtv5.Claims {
					return &auth.AuthClaims{}
				})),
			).Match(NewWhiteListMatcher()).Build(),
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	p.RegisterPostHTTPServer(srv, post)
	p.RegisterCategoryHTTPServer(srv, category)
	p.RegisterTagHTTPServer(srv, tag)
	return srv
}
