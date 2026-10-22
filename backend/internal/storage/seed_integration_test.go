//go:build integration

package storage_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

func TestSeedLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)

	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	assertSeededGraph(t, pool)

	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("repeated seed: %v", err)
	}
	assertSeededGraph(t, pool)

	if _, err := pool.Exec(ctx, `UPDATE call_types SET name = 'Изменённый тип' WHERE duration_minutes = 15`); err != nil {
		t.Fatalf("change seeded data: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed after change: %v", err)
	}
	var name string
	if err := pool.QueryRow(ctx, `SELECT name FROM call_types WHERE duration_minutes = 15`).Scan(&name); err != nil {
		t.Fatalf("read changed type: %v", err)
	}
	if name != "Изменённый тип" {
		t.Fatalf("seed restored changed data: name = %q", name)
	}
	if _, err := pool.Exec(ctx, `UPDATE call_types SET deleted_at = now() WHERE duration_minutes = 60`); err != nil {
		t.Fatalf("delete seeded type: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed after delete: %v", err)
	}
	var deletedTypeCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM call_types WHERE duration_minutes = 60 AND deleted_at IS NOT NULL`).Scan(&deletedTypeCount); err != nil {
		t.Fatalf("count deleted type: %v", err)
	}
	if deletedTypeCount != 1 {
		t.Fatalf("seed restored deleted data: count = %d", deletedTypeCount)
	}

	if _, err := pool.Exec(ctx, `SET session_replication_role = replica; DELETE FROM organizers; SET session_replication_role = origin`); err != nil {
		t.Fatalf("prepare partial graph: %v", err)
	}
	if err := storage.Seed(ctx, pool); err == nil || !strings.Contains(err.Error(), "partial domain data") {
		t.Fatalf("seed partial graph error = %v, want partial domain data", err)
	}
	var organizers int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM organizers`).Scan(&organizers); err != nil {
		t.Fatalf("count organizers: %v", err)
	}
	if organizers != 0 {
		t.Fatalf("partial seed wrote organizer: count = %d", organizers)
	}
}

func assertSeededGraph(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	var timezone, scheduleName string
	var organizerCount, workingDayIntervals, scheduleDayIntervals, validTypes int
	var durations []int32
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM organizers),
			(SELECT timezone FROM organizers LIMIT 1),
			(SELECT count(*) FROM working_hour_intervals),
			(SELECT name FROM schedules LIMIT 1),
			(SELECT count(*) FROM schedule_intervals),
			(SELECT count(*) FROM call_types ct JOIN schedules s ON s.id = ct.schedule_id
			 WHERE s.name = 'Основное' AND ct.status = 'active' AND ct.description IS NULL
			   AND ct.version = 1)
	`).Scan(&organizerCount, &timezone, &workingDayIntervals, &scheduleName, &scheduleDayIntervals, &validTypes)
	if err != nil {
		t.Fatalf("read seeded graph: %v", err)
	}
	rows, err := pool.Query(ctx, `SELECT duration_minutes FROM call_types WHERE status = 'active' ORDER BY duration_minutes`)
	if err != nil {
		t.Fatalf("read call types: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var duration int32
		if err := rows.Scan(&duration); err != nil {
			t.Fatalf("scan duration: %v", err)
		}
		durations = append(durations, duration)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate call types: %v", err)
	}
	var exactIntervals int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM working_hour_intervals WHERE day_of_week BETWEEN 1 AND 5 AND start_time = '09:00' AND end_time = '18:00') +
			(SELECT count(*) FROM schedule_intervals WHERE day_of_week BETWEEN 1 AND 5 AND start_time = '09:00' AND end_time = '18:00')
	`).Scan(&exactIntervals); err != nil {
		t.Fatalf("verify seeded intervals: %v", err)
	}
	if organizerCount != 1 || timezone != "Europe/Moscow" || workingDayIntervals != 5 || scheduleName != "Основное" || scheduleDayIntervals != 5 || exactIntervals != 10 || validTypes != 3 || fmt.Sprint(durations) != "[15 30 60]" {
		t.Fatalf("unexpected seeded graph: organizers=%d timezone=%q working=%d schedule=%q availability=%d exact_intervals=%d valid_types=%d durations=%v", organizerCount, timezone, workingDayIntervals, scheduleName, scheduleDayIntervals, exactIntervals, validTypes, durations)
	}
}

func TestDatabaseProtectsWeeklyIntervalInvariantsAndRelations(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
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

	for name, statement := range map[string]string{
		"24:00 boundary": `INSERT INTO working_hour_intervals (working_hours_id, day_of_week, start_time, end_time) VALUES ('` + workingHoursID + `', 6, '23:45', '24:00')`,
		"overlap":        `INSERT INTO schedule_intervals (schedule_id, day_of_week, start_time, end_time) VALUES ('` + scheduleID + `', 1, '10:00', '11:00')`,
		"short interval": `INSERT INTO schedule_intervals (schedule_id, day_of_week, start_time, end_time) VALUES ('` + scheduleID + `', 6, '09:00', '09:10')`,
	} {
		if _, err := pool.Exec(ctx, statement); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, scheduleID); err == nil {
		t.Error("schedule referenced by call types was deleted")
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
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
