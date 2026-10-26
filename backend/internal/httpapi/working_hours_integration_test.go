//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

func TestAdminWorkingHoursReturnsTheCompleteVersionedWeek(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := newIntegrationHandler(
		pool,
		NewPublicCallTypesService(pool),
		NewPublicCallsService(pool),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/working-hours", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("ETag"); got != `"1"` {
		t.Errorf("ETag = %q, want %q", got, `"1"`)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	var body generated.WorkingHours
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Timezone != "Europe/Moscow" || body.Version != "1" {
		t.Errorf("metadata = %q version %q", body.Timezone, body.Version)
	}
	if len(body.Days) != 7 {
		t.Fatalf("days = %d, want 7", len(body.Days))
	}
	for index, day := range body.Days {
		wantDay := int32(index + 1)
		if day.DayOfWeek != wantDay {
			t.Errorf("days[%d].day_of_week = %d, want %d", index, day.DayOfWeek, wantDay)
		}
		if wantDay <= 5 {
			if len(day.Intervals) != 1 || day.Intervals[0].StartTime != "09:00" || day.Intervals[0].EndTime != "18:00" {
				t.Errorf("days[%d].intervals = %+v, want 09:00-18:00", index, day.Intervals)
			}
		} else if len(day.Intervals) != 0 {
			t.Errorf("days[%d].intervals = %+v, want empty", index, day.Intervals)
		}
	}
}

func TestAdminWorkingHoursReplacesTheWeekOnlyForTheCurrentVersion(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(
		pool,
		NewPublicCallTypesService(pool),
		NewPublicCallsService(pool),
	)
	body := []byte(`{"days":[
		{"day_of_week":1,"intervals":[]},
		{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[{"start_time":"08:30","end_time":"12:00"},{"start_time":"13:00","end_time":"17:15"}]},
		{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[]},
		{"day_of_week":6,"intervals":[]},
		{"day_of_week":7,"intervals":[]}
	]}`)

	updated := putWorkingHours(handler, `"1"`, body)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", updated.Code, updated.Body.String())
	}
	if got := updated.Header().Get("ETag"); got != `"2"` {
		t.Errorf("updated ETag = %q, want %q", got, `"2"`)
	}
	var updatedBody generated.WorkingHours
	if err := json.Unmarshal(updated.Body.Bytes(), &updatedBody); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updatedBody.Version != "2" || len(updatedBody.Days[2].Intervals) != 2 {
		t.Errorf("updated working hours = %+v", updatedBody)
	}

	stale := putWorkingHours(handler, `"1"`, body)
	if stale.Code != http.StatusPreconditionFailed || !bytes.Contains(stale.Body.Bytes(), []byte(`"code":"version_conflict"`)) {
		t.Errorf("stale update = %d %s, want 412 version_conflict", stale.Code, stale.Body.String())
	}

	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/admin/working-hours", nil))
	if read.Header().Get("ETag") != `"2"` || !bytes.Equal(read.Body.Bytes(), updated.Body.Bytes()) {
		t.Errorf("confirmed state changed after stale update: ETag=%q body=%s", read.Header().Get("ETag"), read.Body.String())
	}
}

func TestAdminWorkingHoursRejectsInvalidPreconditionsAndIntervals(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(
		pool,
		NewPublicCallTypesService(pool),
		NewPublicCallsService(pool),
	)
	valid := []byte(`{"days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[]},{"day_of_week":6,"intervals":[]},
		{"day_of_week":7,"intervals":[]}
	]}`)
	overlapping := []byte(`{"days":[
		{"day_of_week":1,"intervals":[{"start_time":"09:00","end_time":"10:00"},{"start_time":"09:30","end_time":"11:00"}]},
		{"day_of_week":2,"intervals":[]},{"day_of_week":3,"intervals":[]},
		{"day_of_week":4,"intervals":[]},{"day_of_week":5,"intervals":[]},
		{"day_of_week":6,"intervals":[]},{"day_of_week":7,"intervals":[]}
	]}`)
	invalidDay := bytes.Replace(valid, []byte(`"day_of_week":7`), []byte(`"day_of_week":8`), 1)
	invalidTime := bytes.Replace(overlapping, []byte(`"09:00"`), []byte(`"09:03"`), 1)
	tooFewDays := []byte(`{"days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[]},{"day_of_week":6,"intervals":[]}
	]}`)
	tooManyDays := bytes.Replace(
		valid,
		[]byte(`{"day_of_week":7,"intervals":[]}`),
		[]byte(`{"day_of_week":7,"intervals":[]},{"day_of_week":7,"intervals":[]}`),
		1,
	)

	for _, example := range []struct {
		name        string
		ifMatch     string
		body        []byte
		contentType string
		status      int
		code        string
	}{
		{"missing If-Match", "", valid, "application/json", http.StatusPreconditionRequired, "precondition_required"},
		{"malformed If-Match", "1", valid, "application/json", http.StatusBadRequest, "invalid_request"},
		{"malformed JSON", `"1"`, []byte(`{"days":`), "application/json", http.StatusBadRequest, "invalid_request"},
		{"wrong content type", `"1"`, valid, "text/plain", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"overlapping intervals", `"1"`, overlapping, "application/json", http.StatusUnprocessableEntity, "validation_error"},
		{"invalid ISO day", `"1"`, invalidDay, "application/json", http.StatusUnprocessableEntity, "validation_error"},
		{"time outside five-minute grid", `"1"`, invalidTime, "application/json", http.StatusUnprocessableEntity, "validation_error"},
		{"fewer than seven days", `"1"`, tooFewDays, "application/json", http.StatusUnprocessableEntity, "validation_error"},
		{"more than seven days", `"1"`, tooManyDays, "application/json", http.StatusUnprocessableEntity, "validation_error"},
	} {
		t.Run(example.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/working-hours", bytes.NewReader(example.body))
			request.Header.Set("Content-Type", example.contentType)
			if example.ifMatch != "" {
				request.Header.Set("If-Match", example.ifMatch)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != example.status || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"`+example.code+`"`)) {
				t.Errorf("response = %d %s, want %d %s", response.Code, response.Body.String(), example.status, example.code)
			}
		})
	}

	unknownQuery := httptest.NewRecorder()
	handler.ServeHTTP(unknownQuery, httptest.NewRequest(http.MethodGet, "/api/v1/admin/working-hours?unexpected=1", nil))
	if unknownQuery.Code != http.StatusUnprocessableEntity || !bytes.Contains(unknownQuery.Body.Bytes(), []byte(`"path":"query.unexpected"`)) {
		t.Errorf("GET unknown query = %d %s, want 422 query.unexpected", unknownQuery.Code, unknownQuery.Body.String())
	}
}

func TestWorkingHoursChangesImmediatelyConstrainPublicAvailability(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Date(2026, time.October, 22, 8, 0, 0, 0, moscowLocation)
	handler := newIntegrationHandler(
		pool,
		newPublicCallTypesService(pool, func() time.Time { return now }),
		newPublicCallsService(pool, func() time.Time { return now }),
	)

	availabilitySlots := func() int {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/public/call-types/zvonok-na-30-minut/availability",
			nil,
		))
		if response.Code != http.StatusOK {
			t.Fatalf("availability status = %d; body=%s", response.Code, response.Body.String())
		}
		var body generated.AvailabilityResponseBody
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode availability: %v", err)
		}
		return len(body.Days[1].Slots)
	}
	if got := availabilitySlots(); got != 18 {
		t.Fatalf("initial Friday slots = %d, want 18", got)
	}

	narrowed := []byte(`{"days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[{"start_time":"10:00","end_time":"11:00"}]},
		{"day_of_week":6,"intervals":[]},{"day_of_week":7,"intervals":[]}
	]}`)
	if response := putWorkingHours(handler, `"1"`, narrowed); response.Code != http.StatusOK {
		t.Fatalf("narrow status = %d; body=%s", response.Code, response.Body.String())
	}
	if got := availabilitySlots(); got != 2 {
		t.Errorf("narrowed Friday slots = %d, want 2", got)
	}

	expanded := bytes.Replace(narrowed, []byte(`"10:00","end_time":"11:00"`), []byte(`"09:00","end_time":"12:00"`), 1)
	if response := putWorkingHours(handler, `"2"`, expanded); response.Code != http.StatusOK {
		t.Fatalf("expand status = %d; body=%s", response.Code, response.Body.String())
	}
	if got := availabilitySlots(); got != 6 {
		t.Errorf("expanded Friday slots = %d, want 6", got)
	}
}

func TestAdminWorkingHoursSerializesConcurrentUpdates(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(
		pool,
		NewPublicCallTypesService(pool),
		NewPublicCallsService(pool),
	)
	body := []byte(`{"days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[]},{"day_of_week":6,"intervals":[]},
		{"day_of_week":7,"intervals":[]}
	]}`)

	start := make(chan struct{})
	responses := make([]*httptest.ResponseRecorder, 2)
	var wait sync.WaitGroup
	for index := range responses {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			responses[index] = putWorkingHours(handler, `"1"`, body)
		}()
	}
	close(start)
	wait.Wait()

	succeeded, conflicted := 0, 0
	for _, response := range responses {
		switch {
		case response.Code == http.StatusOK:
			succeeded++
		case response.Code == http.StatusPreconditionFailed && bytes.Contains(response.Body.Bytes(), []byte(`"code":"version_conflict"`)):
			conflicted++
		default:
			t.Errorf("unexpected response = %d %s", response.Code, response.Body.String())
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Errorf("succeeded=%d conflicted=%d, want 1/1", succeeded, conflicted)
	}
}

func putWorkingHours(handler http.Handler, ifMatch string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/working-hours", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
