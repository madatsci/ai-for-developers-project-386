package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func NewHandler(ping func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		code, status := http.StatusOK, "ok"
		if err := ping(ctx); err != nil {
			slog.ErrorContext(ctx, "database health check failed", "error", err)
			code, status = http.StatusServiceUnavailable, "unavailable"
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(code)
		if err := json.NewEncoder(w).Encode(struct {
			Status string `json:"status"`
		}{Status: status}); err != nil {
			slog.ErrorContext(ctx, "write health response", "error", err)
		}
	})
	return mux
}
