package server

import (
	"github.com/satiu123/GoPalette/pkg/auth"
	"github.com/satiu123/GoPalette/pkg/health"
	"go.opentelemetry.io/otel/metric"

	p "github.com/satiu123/GoPalette/api/post/v1"

	"github.com/satiu123/GoPalette/app/post/service/internal/conf"
	"github.com/satiu123/GoPalette/app/post/service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Server,
	_ *conf.Auth,
	post *service.PostService,
	category *service.CategoryService,
	tag *service.TagService,
	logger log.Logger,
	h *health.Health,
	counter metric.Int64Counter,
	histogram metric.Float64Histogram,
) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metrics.Server(
				metrics.WithSeconds(histogram),
				metrics.WithRequests(counter),
			),
			selector.Server(
				tracing.Server(),
				logging.Server(logger),
			).Match(ObservabilityMatcher()).Build(),
			selector.Server(
				auth.Server(),
			).Match(AuthMatcher()).Build(),
		),
		grpc.CustomHealth(),
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

	// 注册健康检查服务
	h.RegisterGRPC(srv)

	// 注册 gRPC 服务器和服务实现
	p.RegisterPostServer(srv, post)
	p.RegisterCategoryServer(srv, category)
	p.RegisterTagServer(srv, tag)
	return srv
}
