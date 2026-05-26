package server

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "github.com/satiu123/GoPalette/api/bff/v1"
	"github.com/satiu123/GoPalette/app/bff/service/internal/conf"
	"github.com/satiu123/GoPalette/app/bff/service/internal/service"
	"github.com/satiu123/GoPalette/pkg/auth"
	"github.com/satiu123/GoPalette/pkg/health"
	"go.opentelemetry.io/otel/metric"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	bff *service.BffService,
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

	h.RegisterHTTP(srv)

	v1.RegisterBlogBffHTTPServer(srv, bff)
	return srv
}
