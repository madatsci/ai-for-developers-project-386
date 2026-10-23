package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestApplicationHandlerServesAPIAndSinglePageApplication(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(staticDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<main>calendar</main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "assets", "app.js"), []byte("calendar()"), 0o644); err != nil {
		t.Fatal(err)
	}

	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	handler, err := NewApplicationHandler(api, staticDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name        string
		path        string
		status      int
		contentType string
		body        string
	}{
		{"API request", "/api/health", http.StatusOK, "application/json", `{"status":"ok"}`},
		{"frontend asset", "/assets/app.js", http.StatusOK, "text/javascript; charset=utf-8", "calendar()"},
		{"SPA route", "/booking/zvonok-na-15-minut", http.StatusOK, "text/html; charset=utf-8", "<main>calendar</main>"},
		{"missing asset", "/assets/missing.js", http.StatusNotFound, "text/plain; charset=utf-8", "404 page not found\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if response.Code != tt.status {
				t.Errorf("status = %d, want %d", response.Code, tt.status)
			}
			if got := response.Header().Get("Content-Type"); got != tt.contentType {
				t.Errorf("Content-Type = %q, want %q", got, tt.contentType)
			}
			if got := response.Body.String(); got != tt.body {
				t.Errorf("body = %q, want %q", got, tt.body)
			}
		})
	}
}
