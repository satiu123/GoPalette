package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"
)

type Health struct {
	checkers []Checker

	mu       sync.RWMutex
	status   map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
	watchers map[string][]chan grpc_health_v1.HealthCheckResponse_ServingStatus
}

func New(checkers ...Checker) *Health {
	h := &Health{
		checkers: checkers,

		status: map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
			"":               grpc_health_v1.HealthCheckResponse_SERVING,
			ServiceReadiness: grpc_health_v1.HealthCheckResponse_SERVING,
			ServiceLiveness:  grpc_health_v1.HealthCheckResponse_SERVING,
			ServiceStartup:   grpc_health_v1.HealthCheckResponse_SERVING,
		},
		watchers: make(map[string][]chan grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
	go h.loop()

	return h
}

func (h *Health) Check(ctx context.Context) error {
	var errs []error

	for _, c := range h.checkers {
		if err := c.Check(ctx); err != nil {
			errs = append(
				errs,
				fmt.Errorf("%s: %w", c.Name(), err),
			)
		}
	}

	return errors.Join(errs...)
}

func (h *Health) loop() {
	ticker := time.NewTicker(time.Second)

	for range ticker.C {

		status := grpc_health_v1.HealthCheckResponse_SERVING

		ctx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)

		if err := h.Check(ctx); err != nil {
			status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		}

		cancel()

		h.setStatus(ServiceReadiness, status)
	}
}

func (h *Health) setStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	oldStatus := h.status[service]

	if oldStatus == status {
		return
	}

	h.status[service] = status

	for _, ch := range h.watchers[service] {
		select {
			case ch <- status:
			default:
		}
	}

}
