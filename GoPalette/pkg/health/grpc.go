package health

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	ServiceLiveness  = "liveness"
	ServiceReadiness = "readiness"
	ServiceStartup   = "startup"
)

type grpcServer struct {
	grpc_health_v1.UnimplementedHealthServer

	h *Health
}

func (h *Health) RegisterGRPC(srv grpc.ServiceRegistrar) {
	grpc_health_v1.RegisterHealthServer(
		srv,
		&grpcServer{
			h: h,
		},
	)
}

func (s *grpcServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	service := req.GetService()

	s.h.mu.RLock()
	status := s.h.status[service]
	s.h.mu.RUnlock()

	return &grpc_health_v1.HealthCheckResponse{
		Status: status,
	}, nil
}

func (s *grpcServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	service := req.GetService()

	ch := make(chan grpc_health_v1.HealthCheckResponse_ServingStatus, 1)

	s.h.mu.Lock()
	s.h.watchers[service] = append(s.h.watchers[service], ch)

	currentStatus := s.h.status[service]
	s.h.mu.Unlock()

	if err := stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: currentStatus,
	}); err != nil {
		return err
	}

	for {
		select {
		case status := <-ch:
			if err := stream.Send(&grpc_health_v1.HealthCheckResponse{
				Status: status,
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}
