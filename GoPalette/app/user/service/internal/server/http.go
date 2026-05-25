package server

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	u "github.com/satiu123/GoPalette/api/user/v1"
	"github.com/satiu123/GoPalette/pkg/auth"
	"go.opentelemetry.io/otel/metric"

	"github.com/satiu123/GoPalette/app/user/service/internal/conf"
	"github.com/satiu123/GoPalette/app/user/service/internal/health"
	"github.com/satiu123/GoPalette/app/user/service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server,
	_ *conf.Auth,
	user *service.UserService,
	logger log.Logger,
	h *health.Health,
	counter metric.Int64Counter,
	histogram metric.Float64Histogram,
) *http.Server {

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			metrics.Server(
				metrics.WithSeconds(histogram),
				metrics.WithRequests(counter),
			),
			selector.Server(
				logging.Server(logger),
				tracing.Server(),
			).Match(ObservabilityMatcher()).Build(),
			selector.Server(
				auth.Server(),
			).Match(AuthMatcher()).Build(),
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

	// 注册健康检查 HTTP 处理器
	h.RegisterHTTP(srv)

	// 注册用户服务 HTTP 处理器
	u.RegisterUserHTTPServer(srv, user)
	return srv
}
