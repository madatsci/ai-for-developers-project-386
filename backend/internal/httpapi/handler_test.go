package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
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
			handler := newTestHandler(func(ctx context.Context) error {
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
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", got)
			}
		})
	}
}

func TestRoutes(t *testing.T) {
	handler := newTestHandler(func(context.Context) error {
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
		{http.MethodGet, "/api/v1/public/call-types", http.StatusOK},
		{http.MethodPost, "/api/v1/public/calls", http.StatusUnsupportedMediaType},
		{http.MethodGet, "/api/v1/admin/schedules", http.StatusOK},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
		if response.Code != tt.code {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, response.Code, tt.code)
		}
	}
}

type publicCallTypesStub struct{}

func (publicCallTypesStub) ListPublicCallTypes(context.Context) (generated.ImplResponse, error) {
	return domainResponse(http.StatusOK, generated.PublicCallTypeList{Items: []generated.PublicCallTypeSummary{}}), nil
}

func (publicCallTypesStub) GetPublicCallType(context.Context, string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotFound, nil), nil
}

func (publicCallTypesStub) GetPublicCallTypeAvailability(context.Context, string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotFound, nil), nil
}

func newTestHandler(ping func(context.Context) error) http.Handler {
	return NewHandler(ping, HandlerServices{
		PublicCallTypes:   publicCallTypesStub{},
		PublicCalls:       publicCallsStub{},
		AdminCallTypes:    adminCallTypesStub{},
		AdminCalls:        adminCallsStub{},
		AdminSchedules:    adminSchedulesStub{},
		AdminWorkingHours: adminWorkingHoursStub{},
	})
}

type publicCallsStub struct{}

func (publicCallsStub) CreateCall(context.Context, string, generated.CreateCallRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusCreated, nil), nil
}

type adminWorkingHoursStub struct{}

type adminSchedulesStub struct{}

type adminCallTypesStub struct{}

type adminCallsStub struct{}

func (adminCallsStub) ListAdminCalls(context.Context, generated.CallPeriod, string, string, string) (generated.ImplResponse, error) {
	return domainResponse(http.StatusOK, generated.AdminCallPage{Items: []generated.AdminCall{}}), nil
}

func (adminCallTypesStub) ListAdminCallTypes(context.Context) (generated.ImplResponse, error) {
	return domainResponse(http.StatusOK, generated.AdminCallTypeList{Items: []generated.AdminCallType{}}), nil
}

func (adminCallTypesStub) CreateAdminCallType(context.Context, generated.CreateCallTypeRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusCreated, nil), nil
}

func (adminCallTypesStub) UpdateAdminCallType(context.Context, string, string, generated.UpdateCallTypeRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusOK, nil), nil
}

func (adminCallTypesStub) DeleteAdminCallType(context.Context, string, string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNoContent, nil), nil
}

func (adminSchedulesStub) ListAdminSchedules(context.Context) (generated.ImplResponse, error) {
	return domainResponse(http.StatusOK, generated.AdminScheduleList{Items: []generated.AdminSchedule{}}), nil
}

func (adminSchedulesStub) CreateAdminSchedule(context.Context, generated.CreateScheduleRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusCreated, nil), nil
}

func (adminSchedulesStub) UpdateAdminSchedule(context.Context, string, string, generated.UpdateScheduleRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusOK, nil), nil
}

func (adminSchedulesStub) DeleteAdminSchedule(context.Context, string, string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNoContent, nil), nil
}

func (adminWorkingHoursStub) GetAdminWorkingHours(context.Context) (generated.ImplResponse, error) {
	return generated.Response(http.StatusOK, nil), nil
}

func (adminWorkingHoursStub) UpdateAdminWorkingHours(context.Context, string, generated.UpdateWorkingHoursRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusOK, nil), nil
}
