package server

import (
	net_http "net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "github.com/satiu123/GoPalette/api/search/v1"
	"github.com/satiu123/GoPalette/app/search/service/internal/conf"
	"github.com/satiu123/GoPalette/app/search/service/internal/health"
	"github.com/satiu123/GoPalette/app/search/service/internal/service"
	"go.opentelemetry.io/otel/metric"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	search *service.SearchService,
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

	// 注册健康检查服务
	mux := net_http.NewServeMux()
	health.RegisterHTTP(mux, h)
	srv.HandlePrefix("/health", mux)

	// 注册搜索服务
	v1.RegisterSearchHTTPServer(srv, search)
	return srv
}
