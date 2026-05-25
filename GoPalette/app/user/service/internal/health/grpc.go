package health

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
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

func RegisterGRPC(srv grpc.ServiceRegistrar, h *Health) {
	grpc_health_v1.RegisterHealthServer(
		srv,
		&grpcServer{
			h: h,
		},
	)
}

func (s *grpcServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {

	switch req.GetService() {

	case "":
		return serving()

	case ServiceLiveness:
		return serving()

	case ServiceStartup:
		return serving()

	case ServiceReadiness:
		if err := s.h.Check(ctx); err != nil {
			return notServing(), nil
		}

		return serving()

	default:
		return notServing(), nil
	}
}

func (s *grpcServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return status.Error(
		codes.Unimplemented,
		"watch is not implemented",
	)
}

func serving() (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func notServing() *grpc_health_v1.HealthCheckResponse {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}
}
