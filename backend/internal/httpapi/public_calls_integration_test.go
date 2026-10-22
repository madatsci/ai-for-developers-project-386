//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

func TestCreateCallPersistsNormalizedSnapshot(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}

	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(
		pool,
		newPublicCallTypesService(pool, func() time.Time { return now }),
		newPublicCallsService(pool, func() time.Time { return now }),
	)
	body := []byte(`{
		"call_type_slug":"zvonok-na-30-minut",
		"call_type_version":"1",
		"starts_at":"2026-10-23T10:00:00+03:00",
		"visitor_name":"  И\u0306рина  ",
		"visitor_email":"  irina@EXAMPLE.COM  ",
		"visitor_notes":"  Первая строка\r\nВторая строка  "
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/public/calls", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if got := response.Header().Get("Location"); got != "" {
		t.Errorf("Location = %q, want empty", got)
	}
	var created generated.CreatedCall
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.CallTypeSlug != "zvonok-na-30-minut" || created.CallTypeName != "Звонок на 30 минут" || created.DurationMinutes != 30 {
		t.Errorf("unexpected type snapshot: %+v", created)
	}
	if created.VisitorName != "Йрина" || created.VisitorEmail != "irina@example.com" {
		t.Errorf("normalized visitor = %q <%s>", created.VisitorName, created.VisitorEmail)
	}
	if created.VisitorNotes == nil || *created.VisitorNotes != "Первая строка\nВторая строка" {
		t.Errorf("visitor_notes = %v", created.VisitorNotes)
	}
	if got := created.StartsAt.Format(time.RFC3339); got != "2026-10-23T10:00:00+03:00" {
		t.Errorf("starts_at = %s", got)
	}
	if got := created.EndsAt.Format(time.RFC3339); got != "2026-10-23T10:30:00+03:00" {
		t.Errorf("ends_at = %s", got)
	}

	var storedName, storedEmail string
	var storedNotes *string
	if err := pool.QueryRow(ctx, `SELECT visitor_name, visitor_email, visitor_notes FROM calls WHERE id = $1`, created.Id).Scan(&storedName, &storedEmail, &storedNotes); err != nil {
		t.Fatalf("read stored call: %v", err)
	}
	if storedName != created.VisitorName || storedEmail != created.VisitorEmail || storedNotes == nil || *storedNotes != *created.VisitorNotes {
		t.Errorf("stored visitor = %q <%s> %v, response = %q <%s> %v", storedName, storedEmail, storedNotes, created.VisitorName, created.VisitorEmail, created.VisitorNotes)
	}
}

func TestCreateCallAcceptsSlotAcrossTouchingAvailabilityIntervals(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var workingHoursID, scheduleID string
	if err := pool.QueryRow(ctx, `SELECT id FROM working_hours`).Scan(&workingHoursID); err != nil {
		t.Fatalf("read working hours: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM schedules`).Scan(&scheduleID); err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	for _, change := range []struct {
		query string
		id    string
	}{
		{`DELETE FROM working_hour_intervals WHERE working_hours_id = $1 AND day_of_week = 5`, workingHoursID},
		{`DELETE FROM schedule_intervals WHERE schedule_id = $1 AND day_of_week = 5`, scheduleID},
		{`INSERT INTO working_hour_intervals (working_hours_id, day_of_week, start_time, end_time) VALUES ($1, 5, '09:00', '10:00'), ($1, 5, '10:00', '11:00')`, workingHoursID},
		{`INSERT INTO schedule_intervals (schedule_id, day_of_week, start_time, end_time) VALUES ($1, 5, '09:00', '10:00'), ($1, 5, '10:00', '11:00')`, scheduleID},
	} {
		if _, err := pool.Exec(ctx, change.query, change.id); err != nil {
			t.Fatalf("replace Friday intervals: %v", err)
		}
	}
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(pool, newPublicCallTypesService(pool, func() time.Time { return now }), newPublicCallsService(pool, func() time.Time { return now }))
	body := `{"call_type_slug":"zvonok-na-30-minut","call_type_version":"1","starts_at":"2026-10-23T09:45:00+03:00","visitor_name":"Анна","visitor_email":"anna@example.com"}`

	response := postCall(t, handler, "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", body)
	if response.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body=%s", response.Code, response.Body.String())
	}
}

func TestCreateCallSerializesConcurrentRequests(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(pool, newPublicCallTypesService(pool, func() time.Time { return now }), newPublicCallsService(pool, func() time.Time { return now }))

	t.Run("identical requests create one call", func(t *testing.T) {
		body := `{"call_type_slug":"zvonok-na-30-minut","call_type_version":"1","starts_at":"2026-10-23T10:00:00+03:00","visitor_name":"Анна","visitor_email":"anna@example.com"}`
		responses := concurrentCallResponses(handler, []string{
			"8f0902ca-91a1-4f87-87ed-e3b5b00c86b7",
			"8f0902ca-91a1-4f87-87ed-e3b5b00c86b7",
			"8f0902ca-91a1-4f87-87ed-e3b5b00c86b7",
			"8f0902ca-91a1-4f87-87ed-e3b5b00c86b7",
		}, body)
		for index, response := range responses {
			if response.Code != http.StatusCreated || response.Body.String() != responses[0].Body.String() {
				t.Errorf("response[%d] = %d %s, want original 201 %s", index, response.Code, response.Body.String(), responses[0].Body.String())
			}
		}
	})

	t.Run("overlapping requests create one call", func(t *testing.T) {
		body := `{"call_type_slug":"zvonok-na-60-minut","call_type_version":"1","starts_at":"2026-10-23T11:00:00+03:00","visitor_name":"Анна","visitor_email":"anna@example.com"}`
		keys := make([]string, 8)
		for index := range keys {
			keys[index] = fmt.Sprintf("8f0902ca-91a1-4f87-87ed-e3b5b00c%04x", index)
		}
		responses := concurrentCallResponses(handler, keys, body)
		created, unavailable := 0, 0
		for _, response := range responses {
			switch {
			case response.Code == http.StatusCreated:
				created++
			case response.Code == http.StatusConflict && strings.Contains(response.Body.String(), `"code":"slot_unavailable"`):
				unavailable++
			default:
				t.Errorf("unexpected response = %d %s", response.Code, response.Body.String())
			}
		}
		if created != 1 || unavailable != len(keys)-1 {
			t.Errorf("created=%d unavailable=%d, want 1/%d", created, unavailable, len(keys)-1)
		}
	})

	var calls, records int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM calls), (SELECT count(*) FROM call_idempotency)`).Scan(&calls, &records); err != nil {
		t.Fatalf("count persisted rows: %v", err)
	}
	if calls != 2 || records != 2 {
		t.Errorf("calls=%d records=%d, want 2/2", calls, records)
	}
}

func concurrentCallResponses(handler http.Handler, keys []string, body string) []*httptest.ResponseRecorder {
	start := make(chan struct{})
	responses := make([]*httptest.ResponseRecorder, len(keys))
	var wait sync.WaitGroup
	for index, key := range keys {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/api/v1/public/calls", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", key)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			responses[index] = response
		}()
	}
	close(start)
	wait.Wait()
	return responses
}

func TestCreateCallRejectsInvalidHTTPAndFields(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(
		pool,
		newPublicCallTypesService(pool, func() time.Time { return now }),
		newPublicCallsService(pool, func() time.Time { return now }),
	)
	validBody := `{"call_type_slug":"zvonok-na-30-minut","call_type_version":"1","starts_at":"2026-10-23T10:00:00+03:00","visitor_name":"Анна","visitor_email":"anna@example.com","visitor_notes":null}`

	for _, example := range []struct {
		name        string
		contentType string
		key         string
		body        string
		query       string
		status      int
		code        string
		path        string
	}{
		{"wrong content type", "text/plain", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", validBody, "", http.StatusUnsupportedMediaType, "unsupported_media_type", ""},
		{"body over 64 KiB", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", validBody + strings.Repeat(" ", 64*1024), "", http.StatusBadRequest, "invalid_request", ""},
		{"unknown field", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, `,"visitor_notes":null`, `,"visitor_notes":null,"duration_minutes":30`, 1), "", http.StatusBadRequest, "invalid_request", ""},
		{"unknown query", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", validBody, "unexpected=1", http.StatusUnprocessableEntity, "validation_error", "query.unexpected"},
		{"trailing JSON", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", validBody + `{}`, "", http.StatusBadRequest, "invalid_request", ""},
		{"missing idempotency key", "application/json", "", validBody, "", http.StatusUnprocessableEntity, "validation_error", "headers.Idempotency-Key"},
		{"non-v4 idempotency key", "application/json", "018f0000-0000-7000-8000-000000000001", validBody, "", http.StatusUnprocessableEntity, "validation_error", "headers.Idempotency-Key"},
		{"invalid call type slug", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, "zvonok-na-30-minut", "Zvonok na 30 minut", 1), "", http.StatusUnprocessableEntity, "validation_error", "body.call_type_slug"},
		{"long call type slug", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, "zvonok-na-30-minut", strings.Repeat("a", 101), 1), "", http.StatusUnprocessableEntity, "validation_error", "body.call_type_slug"},
		{"fractional start", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, "10:00:00+03:00", "10:00:00.000+03:00", 1), "", http.StatusUnprocessableEntity, "validation_error", "body.starts_at"},
		{"off-grid absolute start", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, "+03:00", "+03:02", 1), "", http.StatusUnprocessableEntity, "validation_error", "body.starts_at"},
		{"multiline name", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, `"Анна"`, `"Анна\nИванова"`, 1), "", http.StatusUnprocessableEntity, "validation_error", "body.visitor_name"},
		{"long name", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, `"Анна"`, `"`+strings.Repeat("Я", 101)+`"`, 1), "", http.StatusUnprocessableEntity, "validation_error", "body.visitor_name"},
		{"display-name email", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, `"anna@example.com"`, `"Anna <anna@example.com>"`, 1), "", http.StatusUnprocessableEntity, "validation_error", "body.visitor_email"},
		{"unicode email", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, `"anna@example.com"`, `"анна@example.com"`, 1), "", http.StatusUnprocessableEntity, "validation_error", "body.visitor_email"},
		{"long notes", "application/json", "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7", strings.Replace(validBody, "null", `"`+strings.Repeat("я", 1001)+`"`, 1), "", http.StatusUnprocessableEntity, "validation_error", "body.visitor_notes"},
	} {
		t.Run(example.name, func(t *testing.T) {
			path := "/api/v1/public/calls"
			if example.query != "" {
				path += "?" + example.query
			}
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(example.body))
			request.Header.Set("Content-Type", example.contentType)
			if example.key != "" {
				request.Header.Set("Idempotency-Key", example.key)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != example.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, example.status, response.Body.String())
			}
			var body struct {
				Code        string                 `json:"code"`
				FieldErrors []generated.FieldError `json:"field_errors"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error: %v; body=%s", err, response.Body.String())
			}
			if body.Code != example.code {
				t.Errorf("code = %q, want %q", body.Code, example.code)
			}
			if example.path != "" && (len(body.FieldErrors) == 0 || body.FieldErrors[0].Path != example.path) {
				t.Errorf("field_errors = %+v, want first path %q", body.FieldErrors, example.path)
			}
			if got := response.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", got)
			}
		})
	}
}

func TestCreateCallIdempotency(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(pool, newPublicCallTypesService(pool, func() time.Time { return now }), newPublicCallsService(pool, func() time.Time { return now }))
	key := "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7"
	firstBody := `{"call_type_slug":"zvonok-na-30-minut","call_type_version":"1","starts_at":"2026-10-23T10:00:00+03:00","visitor_name":"  Анна  ","visitor_email":"anna@EXAMPLE.COM","visitor_notes":null}`

	first := postCall(t, handler, key, firstBody)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d; body=%s", first.Code, first.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE call_types SET name = 'Изменённый тип', duration_minutes = 45, version = version + 1 WHERE slug = 'zvonok-na-30-minut'`); err != nil {
		t.Fatalf("update call type: %v", err)
	}
	repeatedBody := strings.Replace(firstBody, `"  Анна  "`, `"Анна"`, 1)
	repeatedBody = strings.Replace(repeatedBody, `"2026-10-23T10:00:00+03:00"`, `"2026-10-23T07:00:00Z"`, 1)
	repeated := postCall(t, handler, key, repeatedBody)
	if repeated.Code != http.StatusCreated || repeated.Body.String() != first.Body.String() {
		t.Errorf("repeated response = %d %s, want original 201 %s", repeated.Code, repeated.Body.String(), first.Body.String())
	}
	conflict := postCall(t, handler, key, strings.Replace(firstBody, `"visitor_notes":null`, `"visitor_notes":"другое"`, 1))
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"idempotency_conflict"`) {
		t.Errorf("conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	var calls, records int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM calls), (SELECT count(*) FROM call_idempotency)`).Scan(&calls, &records); err != nil {
		t.Fatalf("count persisted rows: %v", err)
	}
	if calls != 1 || records != 1 {
		t.Errorf("calls=%d records=%d, want 1/1", calls, records)
	}
}

func TestCreateCallDoesNotReserveIdempotencyKeyAfterFailure(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(pool, newPublicCallTypesService(pool, func() time.Time { return now }), newPublicCallsService(pool, func() time.Time { return now }))
	key := "8f0902ca-91a1-4f87-87ed-e3b5b00c86b7"
	stale := `{"call_type_slug":"zvonok-na-30-minut","call_type_version":"2","starts_at":"2026-10-23T10:00:00+03:00","visitor_name":"Анна","visitor_email":"anna@example.com"}`
	failed := postCall(t, handler, key, stale)
	if failed.Code != http.StatusConflict || !strings.Contains(failed.Body.String(), `"code":"call_type_changed"`) {
		t.Fatalf("failed response = %d %s", failed.Code, failed.Body.String())
	}
	succeeded := postCall(t, handler, key, strings.Replace(stale, `"call_type_version":"2"`, `"call_type_version":"1"`, 1))
	if succeeded.Code != http.StatusCreated {
		t.Errorf("second status = %d, want 201; body=%s", succeeded.Code, succeeded.Body.String())
	}
}

func postCall(t *testing.T, handler http.Handler, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/public/calls", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
