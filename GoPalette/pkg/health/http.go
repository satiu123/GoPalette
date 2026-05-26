package health

import (
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const ReadyTimeout = time.Second * 3

func (h *Health) RegisterHTTP(srv *khttp.Server) {
	srv.HandleFunc("/health/live", h.Live)
	srv.HandleFunc("/health/ready", h.Ready)
	srv.HandleFunc("/health/startup", h.Startup)
}

func (h *Health) Live(w khttp.ResponseWriter, r *khttp.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Health) Ready(w khttp.ResponseWriter, r *khttp.Request) {
	h.mu.RLock()
	status := h.status[ServiceReadiness]
	h.mu.RUnlock()

	if status != grpc_health_v1.HealthCheckResponse_SERVING {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Health) Startup(w khttp.ResponseWriter, r *khttp.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
