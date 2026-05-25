package health

import (
	"context"
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
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
	ctx, cancel := context.WithTimeout(
		r.Context(),
		ReadyTimeout,
	)
	defer cancel()

	if err := h.Check(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Health) Startup(w khttp.ResponseWriter, r *khttp.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
