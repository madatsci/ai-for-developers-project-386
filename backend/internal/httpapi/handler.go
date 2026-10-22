package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

func NewHandler(
	ping func(context.Context) error,
	publicCallTypes generated.PublicCallTypesAPIServicer,
) http.Handler {
	healthController := generated.NewHealthAPIController(healthService{ping: ping})
	publicController := generated.NewPublicCallTypesAPIController(publicCallTypes)
	listRoute := publicController.Routes()["ListPublicCallTypes"]
	routers := []generated.Router{healthController, routeSubset{routes: []generated.Route{listRoute}}}
	return exactJSONContentType(generated.NewRouter(routers...))
}

func exactJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&exactJSONContentTypeWriter{ResponseWriter: w}, r)
	})
}

type exactJSONContentTypeWriter struct {
	http.ResponseWriter
}

func (w *exactJSONContentTypeWriter) WriteHeader(code int) {
	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
	}
	w.ResponseWriter.WriteHeader(code)
}
