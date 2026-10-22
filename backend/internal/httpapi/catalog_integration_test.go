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

	handler := NewHandler(pool.Ping, NewPublicCallTypesService(pool))
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
