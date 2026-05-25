package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware/selector"
)

func AuthMatcher() selector.MatchFunc {
	return func(ctx context.Context, operation string) bool {
		return !IsPublicOperation(operation)
	}
}

func ObservabilityMatcher() selector.MatchFunc {
	return func(ctx context.Context, operation string) bool {
		return !IsHealthOperation(operation)
	}
}
