package storage

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const schema = `
CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;

CREATE TABLE IF NOT EXISTS organizers (
    id uuid UNIQUE NOT NULL,
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    timezone text NOT NULL CHECK (timezone = 'Europe/Moscow'),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS working_hours (
    id uuid PRIMARY KEY,
    organizer_id uuid UNIQUE NOT NULL REFERENCES organizers(id) ON DELETE RESTRICT,
    version bigint NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (substring(id::text, 15, 1) = '7')
);

CREATE TABLE IF NOT EXISTS working_hour_intervals (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    working_hours_id uuid NOT NULL REFERENCES working_hours(id) ON DELETE CASCADE,
    day_of_week smallint NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time time without time zone NOT NULL,
    end_time time without time zone NOT NULL,
    minute_range int4range GENERATED ALWAYS AS (
        int4range(
            (extract(epoch FROM start_time) / 60)::integer,
            (extract(epoch FROM end_time) / 60)::integer,
            '[)'
        )
    ) STORED,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (end_time > start_time),
    CHECK (start_time < time '24:00' AND end_time < time '24:00'),
    CHECK (extract(second FROM start_time) = 0 AND extract(minute FROM start_time)::integer % 5 = 0),
    CHECK (extract(second FROM end_time) = 0 AND extract(minute FROM end_time)::integer % 5 = 0),
    CHECK (end_time - start_time >= interval '15 minutes'),
    EXCLUDE USING gist (working_hours_id WITH =, day_of_week WITH =, minute_range WITH &&)
);

CREATE TABLE IF NOT EXISTS schedules (
    id uuid PRIMARY KEY,
    organizer_id uuid NOT NULL REFERENCES organizers(id) ON DELETE RESTRICT,
    name text NOT NULL,
    name_key text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organizer_id, name_key),
    CHECK (char_length(name) BETWEEN 1 AND 80),
    CHECK (substring(id::text, 15, 1) = '7')
);

CREATE TABLE IF NOT EXISTS schedule_intervals (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    schedule_id uuid NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    day_of_week smallint NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time time without time zone NOT NULL,
    end_time time without time zone NOT NULL,
    minute_range int4range GENERATED ALWAYS AS (
        int4range(
            (extract(epoch FROM start_time) / 60)::integer,
            (extract(epoch FROM end_time) / 60)::integer,
            '[)'
        )
    ) STORED,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (end_time > start_time),
    CHECK (start_time < time '24:00' AND end_time < time '24:00'),
    CHECK (extract(second FROM start_time) = 0 AND extract(minute FROM start_time)::integer % 5 = 0),
    CHECK (extract(second FROM end_time) = 0 AND extract(minute FROM end_time)::integer % 5 = 0),
    CHECK (end_time - start_time >= interval '15 minutes'),
    EXCLUDE USING gist (schedule_id WITH =, day_of_week WITH =, minute_range WITH &&)
);

CREATE TABLE IF NOT EXISTS call_types (
    id uuid PRIMARY KEY,
    schedule_id uuid NOT NULL REFERENCES schedules(id) ON DELETE RESTRICT,
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    name_key text NOT NULL,
    description text,
    duration_minutes integer NOT NULL,
    status text NOT NULL DEFAULT 'active',
    version bigint NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (substring(id::text, 15, 1) = '7'),
    CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$' AND char_length(slug) <= 100),
    CHECK (char_length(name) BETWEEN 1 AND 80),
    CHECK (description IS NULL OR char_length(description) <= 500),
    CHECK (duration_minutes BETWEEN 15 AND 240 AND duration_minutes % 5 = 0),
    CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS call_types_active_name_key_idx
    ON call_types (name_key) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS calls (
    id uuid PRIMARY KEY,
    call_type_id uuid NOT NULL REFERENCES call_types(id) ON DELETE RESTRICT,
    call_type_name text NOT NULL,
    duration_minutes integer NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz GENERATED ALWAYS AS (
        ((starts_at AT TIME ZONE 'UTC') + duration_minutes * interval '1 minute') AT TIME ZONE 'UTC'
    ) STORED,
    visitor_name text NOT NULL,
    visitor_email text NOT NULL,
    visitor_notes text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (substring(id::text, 15, 1) = '7'),
    CHECK (char_length(call_type_name) BETWEEN 1 AND 80),
    CHECK (duration_minutes BETWEEN 15 AND 240 AND duration_minutes % 5 = 0),
    CHECK (char_length(visitor_name) BETWEEN 1 AND 100),
    CHECK (char_length(visitor_email) BETWEEN 3 AND 254),
    CHECK (visitor_notes IS NULL OR char_length(visitor_notes) <= 1000),
    EXCLUDE USING gist (tstzrange(starts_at, ends_at, '[)') WITH &&)
);

CREATE TABLE IF NOT EXISTS call_idempotency (
    key uuid PRIMARY KEY,
    fingerprint text NOT NULL,
    call_id uuid UNIQUE NOT NULL REFERENCES calls(id) ON DELETE RESTRICT,
    response_snapshot jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (substring(key::text, 15, 1) = '4')
);
`

// Migrate creates the storage needed by the current application contract.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("apply database schema: %w", err)
	}
	return nil
}

// Seed creates the one initial organizer graph, or leaves an existing graph untouched.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var organizers int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM organizers`).Scan(&organizers); err != nil {
		return fmt.Errorf("inspect organizers: %w", err)
	}
	if organizers > 0 {
		return tx.Commit(ctx)
	}

	var existing int
	if err := tx.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM working_hours) +
			(SELECT count(*) FROM working_hour_intervals) +
			(SELECT count(*) FROM schedules) +
			(SELECT count(*) FROM schedule_intervals) +
			(SELECT count(*) FROM call_types)
	`).Scan(&existing); err != nil {
		return fmt.Errorf("inspect domain data: %w", err)
	}
	if existing > 0 {
		return errors.New("partial domain data exists without an organizer")
	}

	organizerID, err := NewUUIDv7()
	if err != nil {
		return err
	}
	workingHoursID, err := NewUUIDv7()
	if err != nil {
		return err
	}
	scheduleID, err := NewUUIDv7()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO organizers (id, timezone) VALUES ($1, 'Europe/Moscow')`, organizerID); err != nil {
		return fmt.Errorf("insert organizer: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO working_hours (id, organizer_id) VALUES ($1, $2)`, workingHoursID, organizerID); err != nil {
		return fmt.Errorf("insert working hours: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schedules (id, organizer_id, name, name_key) VALUES ($1, $2, $3, $4)`, scheduleID, organizerID, "Основное", normalizedKey("Основное")); err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}
	for day := 1; day <= 5; day++ {
		if _, err := tx.Exec(ctx, `INSERT INTO working_hour_intervals (working_hours_id, day_of_week, start_time, end_time) VALUES ($1, $2, '09:00', '18:00')`, workingHoursID, day); err != nil {
			return fmt.Errorf("insert working-hours interval: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schedule_intervals (schedule_id, day_of_week, start_time, end_time) VALUES ($1, $2, '09:00', '18:00')`, scheduleID, day); err != nil {
			return fmt.Errorf("insert schedule interval: %w", err)
		}
	}
	for _, duration := range []int{15, 30, 60} {
		id, err := NewUUIDv7()
		if err != nil {
			return err
		}
		name := fmt.Sprintf("Звонок на %d минут", duration)
		slug := fmt.Sprintf("zvonok-na-%d-minut", duration)
		if _, err := tx.Exec(ctx, `
			INSERT INTO call_types (id, schedule_id, slug, name, name_key, description, duration_minutes, status)
			VALUES ($1, $2, $3, $4, $5, NULL, $6, 'active')
		`, id, scheduleID, slug, name, normalizedKey(name), duration); err != nil {
			return fmt.Errorf("insert call type: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}

func normalizedKey(value string) string {
	canonical := norm.NFC.String(strings.TrimSpace(value))
	return norm.NFC.String(cases.Fold().String(canonical))
}

// NewUUIDv7 returns a canonical UUIDv7 for persisted domain resources.
func NewUUIDv7() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	milliseconds := uint64(time.Now().UnixMilli())
	value[0] = byte(milliseconds >> 40)
	value[1] = byte(milliseconds >> 32)
	value[2] = byte(milliseconds >> 24)
	value[3] = byte(milliseconds >> 16)
	value[4] = byte(milliseconds >> 8)
	value[5] = byte(milliseconds)
	value[6] = (value[6] & 0x0f) | 0x70
	value[8] = (value[8] & 0x3f) | 0x80
	parts := []uint32{
		binary.BigEndian.Uint32(value[0:4]),
		uint32(binary.BigEndian.Uint16(value[4:6])),
		uint32(binary.BigEndian.Uint16(value[6:8])),
		uint32(binary.BigEndian.Uint16(value[8:10])),
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", parts[0], parts[1], parts[2], parts[3], value[10:]), nil
}
