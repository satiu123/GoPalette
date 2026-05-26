package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	LoopInterval = time.Second * 2
	CheckTimeout = time.Second * 3
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
	if len(h.checkers) == 0 {
		return nil
	}

	var eg errgroup.Group
	checkErrs := make([]error, len(h.checkers))

	for i, c := range h.checkers {
		eg.Go(func() error {
			if err := c.Check(ctx); err != nil {
				checkErrs[i] = fmt.Errorf("%s: %w", c.Name(), err)
			}
			return nil
		})
	}
	_ = eg.Wait()

	return errors.Join(checkErrs...)
}

func (h *Health) loop() {
	ticker := time.NewTicker(LoopInterval)

	for range ticker.C {

		status := grpc_health_v1.HealthCheckResponse_SERVING

		ctx, cancel := context.WithTimeout(
			context.Background(),
			CheckTimeout,
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
