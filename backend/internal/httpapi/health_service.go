package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

const healthCheckTimeout = 2 * time.Second

type healthService struct {
	ping func(context.Context) error
}

func (s healthService) GetHealth(ctx context.Context) (generated.ImplResponse, error) {
	checkContext, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	headers := map[string][]string{"Cache-Control": {"no-store"}}
	if err := s.ping(checkContext); err != nil {
		slog.ErrorContext(checkContext, "database health check failed", "error", err)
		return generated.ResponseWithHeaders(
			http.StatusServiceUnavailable,
			headers,
			generated.HealthUnavailableBody{Status: "unavailable"},
		), nil
	}

	return generated.ResponseWithHeaders(
		http.StatusOK,
		headers,
		generated.HealthOkBody{Status: "ok"},
	), nil
}
