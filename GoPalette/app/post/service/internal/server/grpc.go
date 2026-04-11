package server

import (
	p "GoPalette/api/post/v1"
	"GoPalette/app/post/service/internal/conf"
	"GoPalette/app/post/service/internal/service"
	"GoPalette/pkg/auth"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, ca *conf.Auth, post *service.PostService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
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
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	p.RegisterPostServer(srv, post)
	return srv
}
