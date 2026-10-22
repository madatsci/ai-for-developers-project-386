package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

const validCreateCallBody = `{
  "call_type_slug": "zvonok-na-30-minut",
  "call_type_version": "7",
  "starts_at": "2026-10-21T15:00:00+03:00",
  "visitor_name": "Anna",
  "visitor_email": "anna@example.com"
}`

func TestGeneratedPublicCallsControllerParsesRequestBody(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantServiced bool
	}{
		{
			name:       "body is required",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "required body field is missing",
			body: `{
  "call_type_slug": "zvonok-na-30-minut",
  "call_type_version": "7",
  "starts_at": "2026-10-21T15:00:00+03:00",
  "visitor_name": "Anna"
}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "unknown body field is rejected",
			body: `{
  "call_type_slug": "zvonok-na-30-minut",
  "call_type_version": "7",
  "starts_at": "2026-10-21T15:00:00+03:00",
  "visitor_name": "Anna",
  "visitor_email": "anna@example.com",
  "unexpected": true
}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed JSON is rejected",
			body:       `{"call_type_slug":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid date is rejected",
			body: `{
  "call_type_slug": "zvonok-na-30-minut",
  "call_type_version": "7",
  "starts_at": "not-a-date",
  "visitor_name": "Anna",
  "visitor_email": "anna@example.com"
}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:         "valid body reaches servicer",
			body:         validCreateCallBody,
			wantStatus:   http.StatusNoContent,
			wantServiced: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &publicCallsServicerSpy{}
			controller := generated.NewPublicCallsAPIController(service)
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/public/calls",
				strings.NewReader(tt.body),
			)
			request.Header.Set("Idempotency-Key", "booking-attempt-1")
			response := httptest.NewRecorder()

			generated.NewRouter(controller).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := service.calls == 1; got != tt.wantServiced {
				t.Errorf("servicer called = %t, want %t", got, tt.wantServiced)
			}
			if tt.wantServiced && service.idempotencyKey != "booking-attempt-1" {
				t.Errorf("Idempotency-Key = %q, want booking-attempt-1", service.idempotencyKey)
			}
		})
	}
}

func TestGeneratedAdminCallTypesControllerChecksNumericBounds(t *testing.T) {
	tests := []struct {
		duration     int
		wantStatus   int
		wantServiced bool
	}{
		{duration: 14, wantStatus: http.StatusBadRequest},
		{duration: 15, wantStatus: http.StatusNoContent, wantServiced: true},
		{duration: 240, wantStatus: http.StatusNoContent, wantServiced: true},
		{duration: 241, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("duration_%d", tt.duration), func(t *testing.T) {
			service := &adminCallTypesServicerSpy{}
			controller := generated.NewAdminCallTypesAPIController(service)
			body := fmt.Sprintf(
				`{"name":"Consultation","duration_minutes":%d,"schedule_id":"018f47de-8a9b-7c6d-8123-456789abcdef"}`,
				tt.duration,
			)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/admin/call-types",
				strings.NewReader(body),
			)

			generated.NewRouter(controller).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := service.createCalls == 1; got != tt.wantServiced {
				t.Errorf("servicer called = %t, want %t", got, tt.wantServiced)
			}
		})
	}
}

func TestGeneratedPublicCallTypesControllerReadsPathParameter(t *testing.T) {
	service := &publicCallTypesServicerSpy{}
	controller := generated.NewPublicCallTypesAPIController(service)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/public/call-types/zvonok-na-30-minut",
		nil,
	)

	generated.NewRouter(controller).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if service.slug != "zvonok-na-30-minut" {
		t.Errorf("slug = %q, want zvonok-na-30-minut", service.slug)
	}
}

type publicCallsServicerSpy struct {
	calls          int
	idempotencyKey string
}

func (s *publicCallsServicerSpy) CreateCall(
	_ context.Context,
	idempotencyKey string,
	_ generated.CreateCallRequest,
) (generated.ImplResponse, error) {
	s.calls++
	s.idempotencyKey = idempotencyKey
	return generated.Response(http.StatusNoContent, nil), nil
}

type adminCallTypesServicerSpy struct {
	createCalls int
}

func (s *adminCallTypesServicerSpy) ListAdminCallTypes(context.Context) (generated.ImplResponse, error) {
	panic("unexpected ListAdminCallTypes call")
}

func (s *adminCallTypesServicerSpy) CreateAdminCallType(
	_ context.Context,
	_ generated.CreateCallTypeRequest,
) (generated.ImplResponse, error) {
	s.createCalls++
	return generated.Response(http.StatusNoContent, nil), nil
}

func (s *adminCallTypesServicerSpy) UpdateAdminCallType(
	context.Context,
	string,
	string,
	generated.UpdateCallTypeRequest,
) (generated.ImplResponse, error) {
	panic("unexpected UpdateAdminCallType call")
}

func (s *adminCallTypesServicerSpy) DeleteAdminCallType(
	context.Context,
	string,
	string,
) (generated.ImplResponse, error) {
	panic("unexpected DeleteAdminCallType call")
}

type publicCallTypesServicerSpy struct {
	slug string
}

func (s *publicCallTypesServicerSpy) ListPublicCallTypes(context.Context) (generated.ImplResponse, error) {
	panic("unexpected ListPublicCallTypes call")
}

func (s *publicCallTypesServicerSpy) GetPublicCallType(
	_ context.Context,
	slug string,
) (generated.ImplResponse, error) {
	s.slug = slug
	return generated.Response(http.StatusNoContent, nil), nil
}

func (s *publicCallTypesServicerSpy) GetPublicCallTypeAvailability(
	context.Context,
	string,
) (generated.ImplResponse, error) {
	panic("unexpected GetPublicCallTypeAvailability call")
}
