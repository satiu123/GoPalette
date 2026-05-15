package server

import (
	"context"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/satiu123/GoPalette/pkg/auth"
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
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewWhiteListMatcher() selector.MatchFunc {
	whiteList := make(map[string]struct{})
	whiteList["/api.post.v1.Post/GetPost"] = struct{}{}
	whiteList["/api.post.v1.Post/ListPosts"] = struct{}{}
	whiteList["/api.post.v1.Post/ListPostsForIndex"] = struct{}{}
	whiteList["/api.post.v1.Post/ListAuthorPosts"] = struct{}{}
	whiteList["/api.post.v1.Post/GetAuthorPostStats"] = struct{}{}
	whiteList["/api.post.v1.Post/ListTopAuthorPosts"] = struct{}{}
	whiteList["/api.post.v1.Post/IncrCommentCount"] = struct{}{}
	whiteList["/api.post.v1.Post/RecordPostView"] = struct{}{}
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
func NewHTTPServer(
	c *conf.Server,
	_ *conf.Auth,
	post *service.PostService,
	category *service.CategoryService,
	tag *service.TagService,
	logger log.Logger,
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
	p.RegisterPostHTTPServer(srv, post)
	p.RegisterCategoryHTTPServer(srv, category)
	p.RegisterTagHTTPServer(srv, tag)
	return srv
}
