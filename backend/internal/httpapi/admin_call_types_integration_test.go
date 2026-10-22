//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

func TestAdminCallTypesListsAndCreatesStablePublicType(t *testing.T) {
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
	handler.ServeHTTP(listed, httptest.NewRequest(http.MethodGet, "/api/v1/admin/call-types", nil))
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", listed.Code, listed.Body.String())
	}
	var list generated.AdminCallTypeList
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Items) != 3 || list.Items[0].DurationMinutes != 15 || list.Items[0].Schedule.Name != "Основное" {
		t.Fatalf("unexpected seeded list: %+v", list.Items)
	}
	if !list.Items[0].CanDelete || list.Items[0].DeleteBlockedReason != nil {
		t.Errorf("seeded deletion state = can_delete %v reason %v", list.Items[0].CanDelete, list.Items[0].DeleteBlockedReason)
	}

	var scheduleID string
	if err := pool.QueryRow(ctx, `SELECT id FROM schedules WHERE name = 'Основное'`).Scan(&scheduleID); err != nil {
		t.Fatalf("read schedule id: %v", err)
	}
	created := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/call-types", bytes.NewBufferString(`{
		"name":"  Разбор объекта  ",
		"description":"  Обсудим архитектуру  ",
		"duration_minutes":45,
		"schedule_id":"`+scheduleID+`"
	}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", created.Code, created.Body.String())
	}
	if created.Header().Get("ETag") != `"1"` || created.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("create headers = ETag %q Cache-Control %q", created.Header().Get("ETag"), created.Header().Get("Cache-Control"))
	}
	var callType generated.AdminCallType
	if err := json.Unmarshal(created.Body.Bytes(), &callType); err != nil {
		t.Fatalf("decode created call type: %v", err)
	}
	if callType.Name != "Разбор объекта" || callType.Slug != "razbor-obekta" || callType.Status != generated.ACTIVE {
		t.Errorf("created call type = %+v", callType)
	}
	if callType.Description == nil || *callType.Description != "Обсудим архитектуру" {
		t.Errorf("description = %v, want trimmed text", callType.Description)
	}

	public := httptest.NewRecorder()
	handler.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types/razbor-obekta", nil))
	if public.Code != http.StatusOK {
		t.Fatalf("public status = %d, want 200; body=%s", public.Code, public.Body.String())
	}
}

func TestAdminCallTypeUpdateIsConditionalAndPreservesCallSnapshot(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var id, scheduleID string
	if err := pool.QueryRow(ctx, `
		SELECT ct.id, ct.schedule_id FROM call_types ct WHERE ct.slug = 'zvonok-na-30-minut'
	`).Scan(&id, &scheduleID); err != nil {
		t.Fatalf("read call type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO calls (id, call_type_id, call_type_name, duration_minutes, starts_at, visitor_name, visitor_email)
		VALUES ('018f0000-0000-7000-8000-000000000301', $1, 'Звонок на 30 минут', 30,
		        '2026-10-23T12:00:00+03:00', 'Анна', 'anna@example.com')
	`, id); err != nil {
		t.Fatalf("insert call snapshot: %v", err)
	}
	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))

	updateBody := `{
		"name":"Архитектурный разбор",
		"description":"",
		"duration_minutes":45,
		"status":"disabled",
		"schedule_id":"` + scheduleID + `"
	}`
	updated := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/call-types/"+id, bytes.NewBufferString(updateBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"1"`)
	handler.ServeHTTP(updated, request)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", updated.Code, updated.Body.String())
	}
	var callType generated.AdminCallType
	if err := json.Unmarshal(updated.Body.Bytes(), &callType); err != nil {
		t.Fatalf("decode updated call type: %v", err)
	}
	if callType.Slug != "zvonok-na-30-minut" || callType.Version != "2" || callType.Status != generated.DISABLED {
		t.Errorf("updated call type = %+v", callType)
	}
	if callType.Description != nil || callType.CanDelete || callType.DeleteBlockedReason == nil || *callType.DeleteBlockedReason != generated.CALL_TYPE_HAS_CALLS {
		t.Errorf("updated nullable/deletion state = description %v can_delete %v reason %v", callType.Description, callType.CanDelete, callType.DeleteBlockedReason)
	}
	publicDisabled := httptest.NewRecorder()
	handler.ServeHTTP(publicDisabled, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types/zvonok-na-30-minut", nil))
	if publicDisabled.Code != http.StatusOK ||
		!bytes.Contains(publicDisabled.Body.Bytes(), []byte(`"name":"Архитектурный разбор"`)) ||
		!bytes.Contains(publicDisabled.Body.Bytes(), []byte(`"duration_minutes":45`)) ||
		!bytes.Contains(publicDisabled.Body.Bytes(), []byte(`"status":"disabled"`)) ||
		!bytes.Contains(publicDisabled.Body.Bytes(), []byte(`"call_type_version":"2"`)) {
		t.Errorf("disabled public detail = status %d body %s", publicDisabled.Code, publicDisabled.Body.String())
	}
	publicList := httptest.NewRecorder()
	handler.ServeHTTP(publicList, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types", nil))
	if bytes.Contains(publicList.Body.Bytes(), []byte(`"slug":"zvonok-na-30-minut"`)) {
		t.Errorf("disabled type remains public: %s", publicList.Body.String())
	}

	stale := httptest.NewRecorder()
	staleRequest := httptest.NewRequest(http.MethodPut, "/api/v1/admin/call-types/"+id, bytes.NewBufferString(updateBody))
	staleRequest.Header.Set("Content-Type", "application/json")
	staleRequest.Header.Set("If-Match", `"1"`)
	handler.ServeHTTP(stale, staleRequest)
	if stale.Code != http.StatusPreconditionFailed || !bytes.Contains(stale.Body.Bytes(), []byte(`"code":"version_conflict"`)) {
		t.Errorf("stale update = status %d body %s", stale.Code, stale.Body.String())
	}

	var organizerID string
	if err := pool.QueryRow(ctx, `SELECT id FROM organizers WHERE singleton`).Scan(&organizerID); err != nil {
		t.Fatalf("read organizer: %v", err)
	}
	emptyScheduleID := "018f0000-0000-7000-8000-000000000302"
	if _, err := pool.Exec(ctx, `
		INSERT INTO schedules (id, organizer_id, name, name_key)
		VALUES ($1, $2, 'Без доступности', 'без доступности')
	`, emptyScheduleID, organizerID); err != nil {
		t.Fatalf("insert empty schedule: %v", err)
	}
	reenabled := httptest.NewRecorder()
	reenableRequest := httptest.NewRequest(http.MethodPut, "/api/v1/admin/call-types/"+id, bytes.NewBufferString(`{
		"name":"Архитектурный разбор",
		"description":null,
		"duration_minutes":45,
		"status":"active",
		"schedule_id":"`+emptyScheduleID+`"
	}`))
	reenableRequest.Header.Set("Content-Type", "application/json")
	reenableRequest.Header.Set("If-Match", `"2"`)
	handler.ServeHTTP(reenabled, reenableRequest)
	if reenabled.Code != http.StatusOK ||
		!bytes.Contains(reenabled.Body.Bytes(), []byte(`"slug":"zvonok-na-30-minut"`)) ||
		!bytes.Contains(reenabled.Body.Bytes(), []byte(`"version":"3"`)) {
		t.Fatalf("re-enable = status %d body %s", reenabled.Code, reenabled.Body.String())
	}
	availability := httptest.NewRecorder()
	handler.ServeHTTP(availability, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types/zvonok-na-30-minut/availability", nil))
	if availability.Code != http.StatusOK {
		t.Fatalf("availability status = %d, want 200; body=%s", availability.Code, availability.Body.String())
	}
	var availabilityBody generated.AvailabilityResponseBody
	if err := json.Unmarshal(availability.Body.Bytes(), &availabilityBody); err != nil {
		t.Fatalf("decode availability: %v", err)
	}
	if availabilityBody.CallType.Slug != "zvonok-na-30-minut" || availabilityBody.CallType.CallTypeVersion != "3" {
		t.Errorf("re-enabled public snapshot = %+v", availabilityBody.CallType)
	}
	for _, day := range availabilityBody.Days {
		if len(day.Slots) != 0 {
			t.Errorf("reassigned empty schedule returned slots on %s: %+v", day.Date, day.Slots)
		}
	}
	publicList = httptest.NewRecorder()
	handler.ServeHTTP(publicList, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types", nil))
	if !bytes.Contains(publicList.Body.Bytes(), []byte(`"slug":"zvonok-na-30-minut"`)) {
		t.Errorf("re-enabled type missing from public catalog: %s", publicList.Body.String())
	}

	deleted := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/call-types/"+id, nil)
	deleteRequest.Header.Set("If-Match", `"3"`)
	handler.ServeHTTP(deleted, deleteRequest)
	if deleted.Code != http.StatusConflict || !bytes.Contains(deleted.Body.Bytes(), []byte(`"code":"call_type_has_calls"`)) {
		t.Errorf("delete with history = status %d body %s", deleted.Code, deleted.Body.String())
	}

	var snapshotName string
	var snapshotDuration int
	if err := pool.QueryRow(ctx, `SELECT call_type_name, duration_minutes FROM calls WHERE call_type_id = $1`, id).Scan(&snapshotName, &snapshotDuration); err != nil {
		t.Fatalf("read call snapshot: %v", err)
	}
	if snapshotName != "Звонок на 30 минут" || snapshotDuration != 30 {
		t.Errorf("call snapshot = %q %d, want original name and duration", snapshotName, snapshotDuration)
	}
}

func TestAdminCallTypeCreateReportsScheduleDeletedConcurrently(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var organizerID string
	if err := pool.QueryRow(ctx, `SELECT id FROM organizers WHERE singleton`).Scan(&organizerID); err != nil {
		t.Fatalf("read organizer: %v", err)
	}
	scheduleID := "018f0000-0000-7000-8000-000000000401"
	if _, err := pool.Exec(ctx, `
		INSERT INTO schedules (id, organizer_id, name, name_key)
		VALUES ($1, $2, 'Удаляемое', 'удаляемое')
	`, scheduleID, organizerID); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	deleting, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin schedule delete: %v", err)
	}
	defer func() { _ = deleting.Rollback(ctx) }()
	if _, err := deleting.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, scheduleID); err != nil {
		t.Fatalf("delete schedule: %v", err)
	}

	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/call-types", bytes.NewBufferString(`{
		"name":"Параллельное создание","description":null,"duration_minutes":30,
		"schedule_id":"`+scheduleID+`"
	}`))
	request.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()
	if err := deleting.Commit(ctx); err != nil {
		t.Fatalf("commit schedule delete: %v", err)
	}
	<-done
	if response.Code != http.StatusConflict ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"code":"schedule_unavailable"`)) {
		t.Errorf("concurrent schedule delete = status %d body %s", response.Code, response.Body.String())
	}
}

func TestAdminCallTypeDeleteKeepsSlugReservedAndValidatesDomainFields(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var id, scheduleID string
	if err := pool.QueryRow(ctx, `
		SELECT ct.id, ct.schedule_id FROM call_types ct WHERE ct.slug = 'zvonok-na-15-minut'
	`).Scan(&id, &scheduleID); err != nil {
		t.Fatalf("read call type: %v", err)
	}
	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))

	deleted := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/call-types/"+id, nil)
	deleteRequest.Header.Set("If-Match", `"1"`)
	handler.ServeHTTP(deleted, deleteRequest)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", deleted.Code, deleted.Body.String())
	}

	created := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/call-types", bytes.NewBufferString(`{
		"name":"Звонок на 15 минут",
		"description":null,
		"duration_minutes":15,
		"schedule_id":"`+scheduleID+`"
	}`))
	createRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, createRequest)
	if created.Code != http.StatusCreated {
		t.Fatalf("recreate status = %d, want 201; body=%s", created.Code, created.Body.String())
	}
	var recreated generated.AdminCallType
	if err := json.Unmarshal(created.Body.Bytes(), &recreated); err != nil {
		t.Fatalf("decode recreated call type: %v", err)
	}
	if recreated.Slug != "zvonok-na-15-minut-2" {
		t.Errorf("recreated slug = %q, want never-reused suffix", recreated.Slug)
	}

	invalid := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/call-types", bytes.NewBufferString(`{
		"name":" ","description":null,"duration_minutes":17,"schedule_id":"not-a-uuid"
	}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(invalid, invalidRequest)
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status = %d, want 422; body=%s", invalid.Code, invalid.Body.String())
	}
	for _, path := range []string{"body.name", "body.duration_minutes", "body.schedule_id"} {
		if !bytes.Contains(invalid.Body.Bytes(), []byte(`"path":"`+path+`"`)) {
			t.Errorf("invalid body misses %s: %s", path, invalid.Body.String())
		}
	}

	belowMinimum := httptest.NewRecorder()
	belowMinimumRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/call-types", bytes.NewBufferString(`{
		"name":"Короткий","description":null,"duration_minutes":10,"schedule_id":"`+scheduleID+`"
	}`))
	belowMinimumRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(belowMinimum, belowMinimumRequest)
	if belowMinimum.Code != http.StatusUnprocessableEntity ||
		!bytes.Contains(belowMinimum.Body.Bytes(), []byte(`"path":"body.duration_minutes"`)) {
		t.Errorf("below-minimum duration = status %d body %s", belowMinimum.Code, belowMinimum.Body.String())
	}
}
