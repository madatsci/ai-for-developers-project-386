package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code int
		body string
	}{
		{"database available", nil, http.StatusOK, "{\"status\":\"ok\"}\n"},
		{"database unavailable", errors.New("connection failed"), http.StatusServiceUnavailable, "{\"status\":\"unavailable\"}\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := NewHandler(func(ctx context.Context) error {
				called = true
				if _, ok := ctx.Deadline(); !ok {
					t.Error("database check must have a deadline")
				}
				return tt.err
			})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))

			if !called {
				t.Error("database was not checked")
			}
			if response.Code != tt.code {
				t.Errorf("status = %d, want %d", response.Code, tt.code)
			}
			if response.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.body)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
		})
	}
}

func TestRoutes(t *testing.T) {
	handler := NewHandler(func(context.Context) error {
		t.Fatal("unexpected database check")
		return nil
	})
	for _, tt := range []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodPost, "/api/health", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/missing", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
		if response.Code != tt.code {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, response.Code, tt.code)
		}
	}
}
