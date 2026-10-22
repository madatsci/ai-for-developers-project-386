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

func TestAdminSchedulesListsAndCreatesCompleteWeeks(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))

	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, httptest.NewRequest(http.MethodGet, "/api/v1/admin/schedules", nil))
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", listed.Code, listed.Body.String())
	}
	var list generated.AdminScheduleList
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(list.Items))
	}
	seeded := list.Items[0]
	if seeded.Name != "Основное" || len(seeded.Days) != 7 || len(seeded.AssignedCallTypes) != 3 {
		t.Errorf("seeded schedule = %+v", seeded)
	}
	if seeded.CanDelete || seeded.DeleteBlockedReason == nil || *seeded.DeleteBlockedReason != generated.SCHEDULE_IN_USE {
		t.Errorf("seeded deletion state = can_delete %v reason %v", seeded.CanDelete, seeded.DeleteBlockedReason)
	}

	created := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/schedules", bytes.NewBufferString(`{
		"name":" Вечернее ",
		"days":[
			{"day_of_week":1,"intervals":[{"start_time":"18:00","end_time":"20:00"}]},
			{"day_of_week":2,"intervals":[]},{"day_of_week":3,"intervals":[]},
			{"day_of_week":4,"intervals":[]},{"day_of_week":5,"intervals":[]},
			{"day_of_week":6,"intervals":[]},{"day_of_week":7,"intervals":[]}
		]
	}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", created.Code, created.Body.String())
	}
	if created.Header().Get("ETag") != `"1"` || created.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("create headers = ETag %q Cache-Control %q", created.Header().Get("ETag"), created.Header().Get("Cache-Control"))
	}
	var schedule generated.AdminSchedule
	if err := json.Unmarshal(created.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode created schedule: %v", err)
	}
	if schedule.Name != "Вечернее" || len(schedule.Days) != 7 || len(schedule.Days[0].Intervals) != 1 {
		t.Errorf("created schedule = %+v", schedule)
	}
	if !schedule.CanDelete || schedule.DeleteBlockedReason != nil || len(schedule.AssignedCallTypes) != 0 {
		t.Errorf("created deletion state = can_delete %v reason %v assignments %v", schedule.CanDelete, schedule.DeleteBlockedReason, schedule.AssignedCallTypes)
	}
}

func TestAdminScheduleUpdateIsConditionalAndChangesPublicAvailability(t *testing.T) {
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
	seeded := listSchedules(t, handler)[0]

	outsideWorkingHours := []byte(`{"name":"Основное вечером","days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[{"start_time":"17:00","end_time":"20:00"}]},
		{"day_of_week":6,"intervals":[]},{"day_of_week":7,"intervals":[]}
	]}`)
	updated := putSchedule(handler, seeded.Id, `"1"`, outsideWorkingHours)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", updated.Code, updated.Body.String())
	}
	if updated.Header().Get("ETag") != `"2"` {
		t.Errorf("updated ETag = %q, want %q", updated.Header().Get("ETag"), `"2"`)
	}

	stale := putSchedule(handler, seeded.Id, `"1"`, outsideWorkingHours)
	if stale.Code != http.StatusPreconditionFailed || !bytes.Contains(stale.Body.Bytes(), []byte(`"code":"version_conflict"`)) {
		t.Errorf("stale update = %d %s, want 412 version_conflict", stale.Code, stale.Body.String())
	}
	confirmed := listSchedules(t, handler)[0]
	if confirmed.Name != "Основное вечером" || confirmed.Version != "2" || confirmed.Days[4].Intervals[0].EndTime != "20:00" {
		t.Errorf("confirmed schedule after conflict = %+v", confirmed)
	}

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
	if got := availabilitySlots(); got != 7 {
		t.Errorf("slots inside current working hours = %d, want 7", got)
	}

	expandedWorkingHours := []byte(`{"days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[{"start_time":"17:00","end_time":"20:00"}]},
		{"day_of_week":6,"intervals":[]},{"day_of_week":7,"intervals":[]}
	]}`)
	if response := putWorkingHours(handler, `"1"`, expandedWorkingHours); response.Code != http.StatusOK {
		t.Fatalf("expand working hours status = %d; body=%s", response.Code, response.Body.String())
	}
	if got := availabilitySlots(); got != 31 {
		t.Errorf("slots after expanding working hours = %d, want 31", got)
	}
}

func TestAdminScheduleUpdatesSerializeOnTheVersion(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	seeded := listSchedules(t, handler)[0]
	body := []byte(`{"name":"Конкурентное","days":[
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
			responses[index] = putSchedule(handler, seeded.Id, `"1"`, body)
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

func TestAdminScheduleDeletionRechecksRestrictionsWithDocumentedPriority(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	seeded := listSchedules(t, handler)[0]

	blocked := deleteSchedule(handler, seeded.Id, `"1"`)
	if blocked.Code != http.StatusConflict || !bytes.Contains(blocked.Body.Bytes(), []byte(`"code":"schedule_in_use"`)) {
		t.Errorf("in-use last schedule deletion = %d %s, want schedule_in_use", blocked.Code, blocked.Body.String())
	}

	created := createEmptySchedule(t, handler, "Удаляемое")
	stale := deleteSchedule(handler, created.Id, `"2"`)
	if stale.Code != http.StatusPreconditionFailed || !bytes.Contains(stale.Body.Bytes(), []byte(`"code":"version_conflict"`)) {
		t.Errorf("stale deletion = %d %s, want version_conflict", stale.Code, stale.Body.String())
	}
	deleted := deleteSchedule(handler, created.Id, `"1"`)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Errorf("delete response = %d %q, want empty 204", deleted.Code, deleted.Body.String())
	}
	if got := len(listSchedules(t, handler)); got != 1 {
		t.Errorf("remaining schedules = %d, want 1", got)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM call_types`); err != nil {
		t.Fatalf("remove schedule assignments: %v", err)
	}
	last := deleteSchedule(handler, seeded.Id, `"1"`)
	if last.Code != http.StatusConflict || !bytes.Contains(last.Body.Bytes(), []byte(`"code":"last_schedule"`)) {
		t.Errorf("last schedule deletion = %d %s, want last_schedule", last.Code, last.Body.String())
	}
}

func TestAdminSchedulesValidateContractAndNameUniqueness(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	validDays := `[{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},{"day_of_week":5,"intervals":[]},{"day_of_week":6,"intervals":[]},{"day_of_week":7,"intervals":[]}]`

	for _, example := range []struct {
		name        string
		method      string
		path        string
		contentType string
		body        string
		status      int
		code        string
	}{
		{"duplicate normalized name", http.MethodPost, "/api/v1/admin/schedules", "application/json", `{"name":" основное ","days":` + validDays + `}`, http.StatusConflict, "schedule_name_conflict"},
		{"missing days", http.MethodPost, "/api/v1/admin/schedules", "application/json", `{"name":"Без недели"}`, http.StatusUnprocessableEntity, "validation_error"},
		{"wrong content type", http.MethodPost, "/api/v1/admin/schedules", "text/plain", `{"name":"Новое","days":` + validDays + `}`, http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"unknown query", http.MethodGet, "/api/v1/admin/schedules?unexpected=1", "", "", http.StatusUnprocessableEntity, "validation_error"},
	} {
		t.Run(example.name, func(t *testing.T) {
			request := httptest.NewRequest(example.method, example.path, bytes.NewBufferString(example.body))
			if example.contentType != "" {
				request.Header.Set("Content-Type", example.contentType)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != example.status || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"`+example.code+`"`)) {
				t.Errorf("response = %d %s, want %d %s", response.Code, response.Body.String(), example.status, example.code)
			}
		})
	}

	validUpdate := []byte(`{"name":"Основное","days":` + validDays + `}`)
	for _, response := range []*httptest.ResponseRecorder{
		putSchedule(handler, "not-a-uuid", `"1"`, validUpdate),
		deleteSchedule(handler, "not-a-uuid", `"1"`),
	} {
		if response.Code != http.StatusUnprocessableEntity ||
			!bytes.Contains(response.Body.Bytes(), []byte(`"code":"validation_error"`)) ||
			!bytes.Contains(response.Body.Bytes(), []byte(`"path":"path.id"`)) {
			t.Errorf("malformed schedule id response = %d %s, want path.id validation error", response.Code, response.Body.String())
		}
	}
}

func listSchedules(t *testing.T, handler http.Handler) []generated.AdminSchedule {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/schedules", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", response.Code, response.Body.String())
	}
	var body generated.AdminScheduleList
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode schedules: %v", err)
	}
	return body.Items
}

func createEmptySchedule(t *testing.T, handler http.Handler, name string) generated.AdminSchedule {
	t.Helper()
	body := []byte(`{"name":"` + name + `","days":[
		{"day_of_week":1,"intervals":[]},{"day_of_week":2,"intervals":[]},
		{"day_of_week":3,"intervals":[]},{"day_of_week":4,"intervals":[]},
		{"day_of_week":5,"intervals":[]},{"day_of_week":6,"intervals":[]},
		{"day_of_week":7,"intervals":[]}
	]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/schedules", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", response.Code, response.Body.String())
	}
	var schedule generated.AdminSchedule
	if err := json.Unmarshal(response.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode created schedule: %v", err)
	}
	return schedule
}

func putSchedule(handler http.Handler, id, ifMatch string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/schedules/"+id, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func deleteSchedule(handler http.Handler, id, ifMatch string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/schedules/"+id, nil)
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
