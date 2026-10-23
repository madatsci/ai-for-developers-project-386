package httpapi

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

// NewApplicationHandler serves the API and a production SPA from one origin.
func NewApplicationHandler(api http.Handler, staticDir string) (http.Handler, error) {
	staticFiles := os.DirFS(staticDir)
	if _, err := fs.Stat(staticFiles, "index.html"); err != nil {
		return nil, fmt.Errorf("open frontend index: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticFiles))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if info, err := fs.Stat(staticFiles, path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}

		request := r.Clone(r.Context())
		request.URL.Path = "/"
		fileServer.ServeHTTP(w, request)
	}), nil
}
