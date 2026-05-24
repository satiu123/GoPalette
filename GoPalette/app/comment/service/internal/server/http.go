package server

import (
	"context"
	"strings"

	net_http "net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "github.com/satiu123/GoPalette/api/comment/v1"
	"github.com/satiu123/GoPalette/pkg/auth"
	"go.opentelemetry.io/otel/metric"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/satiu123/GoPalette/app/comment/service/internal/conf"
	"github.com/satiu123/GoPalette/app/comment/service/internal/health"
	"github.com/satiu123/GoPalette/app/comment/service/internal/service"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := make(map[string]struct{})
	whiteList["/api.comment.v1.Comment/ListComments"] = struct{}{}
	return func(ctx context.Context, operation string) bool {
		if strings.HasPrefix(operation, "/grpc.health.v1.Health/") {
			return false
		}
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	_ *conf.Auth,
	comment *service.CommentService,
	logger log.Logger,
	h *health.Health,
	counter metric.Int64Counter,
	histogram metric.Float64Histogram,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			logging.Server(logger),
			metrics.Server(
				metrics.WithSeconds(histogram),
				metrics.WithRequests(counter),
			),
			selector.Server(
				auth.Server(),
			).Match(NewWhiteListMatcher()).Build(),
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
	srv.Handle("/metrics", promhttp.Handler())

	// 注册健康检查路由
	mux := net_http.NewServeMux()
	health.RegisterHTTP(mux, h)
	srv.HandlePrefix("/healthz", mux)

	// 注册 HTTP 服务器和服务实现
	v1.RegisterCommentHTTPServer(srv, comment)
	return srv
}
