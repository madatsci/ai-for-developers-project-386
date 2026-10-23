//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

func TestPublicCallTypeCatalogReadsActiveTypesFromPostgres(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var scheduleID string
	if err := pool.QueryRow(ctx, `SELECT id FROM schedules`).Scan(&scheduleID); err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO call_types (id, schedule_id, slug, name, name_key, description, duration_minutes, status)
		VALUES
			('018f0000-0000-7000-8000-000000000001', $1, 'alfavitnyy', 'Алфавитный', 'алфавитный', 'Описание', 30, 'active'),
			('018f0000-0000-7000-8000-000000000002', $1, 'otklyuchennyy', 'Отключённый', 'отключённый', NULL, 15, 'disabled'),
			('018f0000-0000-7000-8000-000000000003', $1, 'udalennyy', 'Удалённый', 'удалённый', NULL, 15, 'active')
	`, scheduleID); err != nil {
		t.Fatalf("insert catalog fixtures: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE call_types SET deleted_at = now() WHERE slug = 'udalennyy'`); err != nil {
		t.Fatalf("soft-delete catalog fixture: %v", err)
	}

	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	var body generated.PublicCallTypeList
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantNames := []string{"Звонок на 15 минут", "Алфавитный", "Звонок на 30 минут", "Звонок на 60 минут"}
	if len(body.Items) != len(wantNames) {
		t.Fatalf("items = %d, want %d: %s", len(body.Items), len(wantNames), response.Body.String())
	}
	for index, want := range wantNames {
		if body.Items[index].Name != want {
			t.Errorf("items[%d].name = %q, want %q", index, body.Items[index].Name, want)
		}
	}
	if body.Items[0].Description != nil {
		t.Errorf("nullable description = %q, want null", *body.Items[0].Description)
	}
	if body.Items[1].Description == nil || *body.Items[1].Description != "Описание" {
		t.Errorf("description = %v, want Описание", body.Items[1].Description)
	}
}

func TestPublicCallTypeReadsActiveAndDisabledTypesByStableSlug(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE call_types SET status = 'disabled', version = 7 WHERE slug = 'zvonok-na-30-minut'`); err != nil {
		t.Fatalf("disable call type: %v", err)
	}

	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types/zvonok-na-30-minut", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var body generated.PublicCallType
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Slug != "zvonok-na-30-minut" || body.Status != generated.DISABLED || body.CallTypeVersion != "7" {
		t.Fatalf("unexpected call type: %+v", body)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}

	if _, err := pool.Exec(ctx, `UPDATE call_types SET deleted_at = now() WHERE slug = 'zvonok-na-30-minut'`); err != nil {
		t.Fatalf("soft-delete call type: %v", err)
	}
	for _, slug := range []string{"zvonok-na-30-minut", "never-existed"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/public/call-types/"+slug, nil))
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"call_type_not_found"`) {
			t.Errorf("slug %q: status=%d body=%s, want call_type_not_found", slug, response.Code, response.Body.String())
		}
	}
}

func TestPublicCallTypeReadsRejectUnknownQueryParameters(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	for _, requestPath := range []string{
		"/api/v1/public/call-types?unexpected=1",
		"/api/v1/public/call-types/zvonok-na-30-minut?unexpected=1",
		"/api/v1/public/call-types/zvonok-na-30-minut/availability?unexpected=1",
	} {
		t.Run(requestPath, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
			if response.Code != http.StatusUnprocessableEntity ||
				!strings.Contains(response.Body.String(), `"path":"query.unexpected"`) {
				t.Errorf("status=%d body=%s, want 422 query.unexpected", response.Code, response.Body.String())
			}
		})
	}
}

func TestPublicCallTypeAvailabilityUsesMoscowWindowAndAllExistingCalls(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var workingHoursID, scheduleID, sixtyMinuteTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM working_hours`).Scan(&workingHoursID); err != nil {
		t.Fatalf("read working hours: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM schedules`).Scan(&scheduleID); err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM call_types WHERE duration_minutes = 60`).Scan(&sixtyMinuteTypeID); err != nil {
		t.Fatalf("read occupied call type: %v", err)
	}
	for _, change := range []struct {
		query string
		id    string
	}{
		{`DELETE FROM working_hour_intervals WHERE working_hours_id = $1 AND day_of_week = 3`, workingHoursID},
		{`DELETE FROM schedule_intervals WHERE schedule_id = $1 AND day_of_week = 3`, scheduleID},
		{`INSERT INTO working_hour_intervals (working_hours_id, day_of_week, start_time, end_time) VALUES
			($1, 3, '09:00', '12:00'), ($1, 3, '12:00', '13:00')`, workingHoursID},
		{`INSERT INTO schedule_intervals (schedule_id, day_of_week, start_time, end_time) VALUES
			($1, 3, '08:00', '10:30'), ($1, 3, '10:30', '14:00')`, scheduleID},
	} {
		if _, err := pool.Exec(ctx, change.query, change.id); err != nil {
			t.Fatalf("replace Wednesday intervals: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO calls (
			id, call_type_id, call_type_name, duration_minutes, starts_at,
			visitor_name, visitor_email, visitor_notes
		) VALUES (
			'018f0000-0000-7000-8000-000000000201', $1, 'Занятый час', 60,
			'2026-10-21T10:30:00+03:00', 'Анна', 'anna@example.com', NULL
		)
	`, sixtyMinuteTypeID); err != nil {
		t.Fatalf("insert occupied call: %v", err)
	}

	moscow := time.FixedZone("Europe/Moscow", 3*60*60)
	now := time.Date(2026, time.October, 21, 10, 2, 34, 123_000_000, moscow)
	handler := newIntegrationHandler(pool, newPublicCallTypesService(pool, func() time.Time { return now }), newPublicCallsService(pool, func() time.Time { return now }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/public/call-types/zvonok-na-30-minut/availability",
		nil,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var body generated.AvailabilityResponseBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.GeneratedAt.Equal(now.Truncate(time.Second)) || body.Timezone != "Europe/Moscow" {
		t.Errorf("snapshot metadata = %s %q, want %s Europe/Moscow", body.GeneratedAt, body.Timezone, now.Truncate(time.Second))
	}
	if body.CallType.Slug != "zvonok-na-30-minut" || body.CallType.CallTypeVersion != "1" {
		t.Errorf("unexpected call type snapshot: %+v", body.CallType)
	}
	if len(body.Days) != 14 {
		t.Fatalf("days = %d, want 14", len(body.Days))
	}
	if body.Days[0].Date != "2026-10-21" || body.Days[13].Date != "2026-11-03" {
		t.Errorf("date window = %s..%s", body.Days[0].Date, body.Days[13].Date)
	}
	if len(body.Days[0].Slots) != 13 {
		t.Fatalf("today slots = %d, want 13: %+v", len(body.Days[0].Slots), body.Days[0].Slots)
	}
	if got := body.Days[0].Slots[0].StartsAt.Format(time.RFC3339); got != "2026-10-21T11:30:00+03:00" {
		t.Errorf("first slot = %s, want boundary-touching 11:30 slot", got)
	}
	if got := body.Days[0].Slots[12].EndsAt.Format(time.RFC3339); got != "2026-10-21T13:00:00+03:00" {
		t.Errorf("last slot end = %s, want 13:00", got)
	}
	if len(body.Days[3].Slots) != 0 || len(body.Days[4].Slots) != 0 {
		t.Errorf("weekend slots must be empty: Saturday=%d Sunday=%d", len(body.Days[3].Slots), len(body.Days[4].Slots))
	}
}

func TestPublicCallTypeAvailabilityDistinguishesUnavailableAndMissingTypes(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE call_types SET status = 'disabled' WHERE slug = 'zvonok-na-15-minut'`); err != nil {
		t.Fatalf("disable call type: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE call_types SET deleted_at = now() WHERE slug = 'zvonok-na-60-minut'`); err != nil {
		t.Fatalf("soft-delete call type: %v", err)
	}

	handler := newIntegrationHandler(pool, NewPublicCallTypesService(pool), NewPublicCallsService(pool))
	for _, example := range []struct {
		slug string
		code int
		body string
	}{
		{"zvonok-na-15-minut", http.StatusConflict, `"code":"call_type_unavailable"`},
		{"zvonok-na-60-minut", http.StatusNotFound, `"code":"call_type_not_found"`},
		{"never-existed", http.StatusNotFound, `"code":"call_type_not_found"`},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/public/call-types/"+example.slug+"/availability",
			nil,
		))
		if response.Code != example.code || !strings.Contains(response.Body.String(), example.body) {
			t.Errorf("slug %q: status=%d body=%s, want %d %s", example.slug, response.Code, response.Body.String(), example.code, example.body)
		}
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("slug %q: Cache-Control = %q, want no-store", example.slug, got)
		}
	}
}

func catalogTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://calendar:calendar@127.0.0.1:5433/calendar?sslmode=disable"
	}
	admin, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("create admin pool: %v", err)
	}
	t.Cleanup(admin.Close)
	schema := "test_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	if _, err := admin.Exec(context.Background(), `DROP SCHEMA IF EXISTS "`+schema+`" CASCADE; CREATE SCHEMA "`+schema+`"`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA IF EXISTS "`+schema+`" CASCADE`)
	})
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newIntegrationHandler(
	pool *pgxpool.Pool,
	publicCallTypes generated.PublicCallTypesAPIServicer,
	publicCalls generated.PublicCallsAPIServicer,
) http.Handler {
	return NewHandler(pool.Ping, HandlerServices{
		PublicCallTypes:   publicCallTypes,
		PublicCalls:       publicCalls,
		AdminCallTypes:    NewAdminCallTypesService(pool),
		AdminCalls:        NewAdminCallsService(pool),
		AdminSchedules:    NewAdminSchedulesService(pool),
		AdminWorkingHours: NewAdminWorkingHoursService(pool),
	})
}
