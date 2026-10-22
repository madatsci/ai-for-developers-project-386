package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

type publicCallType struct {
	slug            string
	name            string
	description     *string
	durationMinutes int32
	status          generated.CallTypeStatus
	version         int64
}

type publicCallTypeRepository interface {
	listActive(context.Context) ([]publicCallType, error)
	getBySlug(context.Context, string) (publicCallType, error)
	readAvailability(context.Context, string, time.Time) (availabilitySnapshot, error)
}

func (r postgresPublicCallTypeRepository) getBySlug(ctx context.Context, slug string) (publicCallType, error) {
	var item publicCallType
	err := r.pool.QueryRow(ctx, `
		SELECT slug, name, description, duration_minutes, status, version
		FROM call_types
		WHERE slug = $1 AND deleted_at IS NULL
	`, slug).Scan(
		&item.slug,
		&item.name,
		&item.description,
		&item.durationMinutes,
		&item.status,
		&item.version,
	)
	return item, err
}

type minuteInterval struct {
	start int
	end   int
}

type callInterval struct {
	start time.Time
	end   time.Time
}

type availabilitySnapshot struct {
	callType      publicCallType
	workingHours  map[int][]minuteInterval
	schedule      map[int][]minuteInterval
	existingCalls []callInterval
}

func (r postgresPublicCallTypeRepository) readAvailability(
	ctx context.Context,
	slug string,
	observedAt time.Time,
) (availabilitySnapshot, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return availabilitySnapshot{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var snapshot availabilitySnapshot
	var scheduleID, organizerID string
	err = tx.QueryRow(ctx, `
		SELECT ct.slug, ct.name, ct.description, ct.duration_minutes, ct.status, ct.version,
		       ct.schedule_id, s.organizer_id
		FROM call_types ct
		JOIN schedules s ON s.id = ct.schedule_id
		WHERE ct.slug = $1 AND ct.deleted_at IS NULL
	`, slug).Scan(
		&snapshot.callType.slug,
		&snapshot.callType.name,
		&snapshot.callType.description,
		&snapshot.callType.durationMinutes,
		&snapshot.callType.status,
		&snapshot.callType.version,
		&scheduleID,
		&organizerID,
	)
	if err != nil {
		return availabilitySnapshot{}, err
	}
	if snapshot.callType.status == generated.DISABLED {
		if err := tx.Commit(ctx); err != nil {
			return availabilitySnapshot{}, err
		}
		return snapshot, nil
	}

	snapshot.workingHours, err = readWeeklyIntervals(ctx, tx, `
		SELECT whi.day_of_week,
		       (extract(hour FROM whi.start_time)::integer * 60 + extract(minute FROM whi.start_time)::integer),
		       (extract(hour FROM whi.end_time)::integer * 60 + extract(minute FROM whi.end_time)::integer)
		FROM working_hour_intervals whi
		JOIN working_hours wh ON wh.id = whi.working_hours_id
		WHERE wh.organizer_id = $1
		ORDER BY whi.day_of_week, whi.start_time
	`, organizerID)
	if err != nil {
		return availabilitySnapshot{}, err
	}
	snapshot.schedule, err = readWeeklyIntervals(ctx, tx, `
		SELECT day_of_week,
		       (extract(hour FROM start_time)::integer * 60 + extract(minute FROM start_time)::integer),
		       (extract(hour FROM end_time)::integer * 60 + extract(minute FROM end_time)::integer)
		FROM schedule_intervals
		WHERE schedule_id = $1
		ORDER BY day_of_week, start_time
	`, scheduleID)
	if err != nil {
		return availabilitySnapshot{}, err
	}

	windowStart := moscowDayStart(observedAt)
	windowEnd := windowStart.AddDate(0, 0, 14)
	rows, err := tx.Query(ctx, `
		SELECT starts_at, ends_at
		FROM calls
		WHERE starts_at < $2 AND ends_at > $1
		ORDER BY starts_at
	`, windowStart, windowEnd)
	if err != nil {
		return availabilitySnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var existing callInterval
		if err := rows.Scan(&existing.start, &existing.end); err != nil {
			return availabilitySnapshot{}, err
		}
		snapshot.existingCalls = append(snapshot.existingCalls, existing)
	}
	if err := rows.Err(); err != nil {
		return availabilitySnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return availabilitySnapshot{}, err
	}
	return snapshot, nil
}

type weeklyIntervalQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readWeeklyIntervals(
	ctx context.Context,
	querier weeklyIntervalQuerier,
	query string,
	arguments ...any,
) (map[int][]minuteInterval, error) {
	rows, err := querier.Query(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	intervals := make(map[int][]minuteInterval)
	for rows.Next() {
		var day, start, end int
		if err := rows.Scan(&day, &start, &end); err != nil {
			return nil, err
		}
		intervals[day] = append(intervals[day], minuteInterval{start: start, end: end})
	}
	return intervals, rows.Err()
}

type postgresPublicCallTypeRepository struct {
	pool *pgxpool.Pool
}

func (r postgresPublicCallTypeRepository) listActive(ctx context.Context) ([]publicCallType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT slug, name, description, duration_minutes
		FROM call_types
		WHERE status = 'active' AND deleted_at IS NULL
		ORDER BY duration_minutes, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]publicCallType, 0)
	for rows.Next() {
		var item publicCallType
		if err := rows.Scan(&item.slug, &item.name, &item.description, &item.durationMinutes); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type publicCallTypesService struct {
	repository publicCallTypeRepository
	now        func() time.Time
}

func NewPublicCallTypesService(pool *pgxpool.Pool) generated.PublicCallTypesAPIServicer {
	return newPublicCallTypesService(pool, time.Now)
}

func newPublicCallTypesService(pool *pgxpool.Pool, now func() time.Time) generated.PublicCallTypesAPIServicer {
	return publicCallTypesService{repository: postgresPublicCallTypeRepository{pool: pool}, now: now}
}

func (s publicCallTypesService) ListPublicCallTypes(ctx context.Context) (generated.ImplResponse, error) {
	rows, err := s.repository.listActive(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "list public call types", "error", err)
		return domainResponse(http.StatusInternalServerError, map[string]string{
			"code": "internal_error", "message": "Не удалось загрузить типы звонков",
		}), nil
	}
	items := make([]generated.PublicCallTypeSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, generated.PublicCallTypeSummary{
			Slug: row.slug, Name: row.name, Description: row.description, DurationMinutes: row.durationMinutes,
		})
	}
	return domainResponse(http.StatusOK, generated.PublicCallTypeList{Items: items}), nil
}

func (s publicCallTypesService) GetPublicCallType(ctx context.Context, slug string) (generated.ImplResponse, error) {
	row, err := s.repository.getBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainResponse(http.StatusNotFound, map[string]string{
			"code": "call_type_not_found", "message": "Тип звонка не найден",
		}), nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "get public call type", "slug", slug, "error", err)
		return domainResponse(http.StatusInternalServerError, map[string]string{
			"code": "internal_error", "message": "Не удалось загрузить тип звонка",
		}), nil
	}
	return domainResponse(http.StatusOK, generated.PublicCallType{
		Slug: row.slug, Name: row.name, Description: row.description,
		DurationMinutes: row.durationMinutes, Status: row.status,
		CallTypeVersion: strconv.FormatInt(row.version, 10),
	}), nil
}

func (s publicCallTypesService) GetPublicCallTypeAvailability(ctx context.Context, slug string) (generated.ImplResponse, error) {
	observedAt := s.now().In(moscowLocation)
	generatedAt := observedAt.Truncate(time.Second)
	snapshot, err := s.repository.readAvailability(ctx, slug, observedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainResponse(http.StatusNotFound, map[string]string{
			"code": "call_type_not_found", "message": "Тип звонка не найден",
		}), nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "get public call type availability", "slug", slug, "error", err)
		return domainResponse(http.StatusInternalServerError, map[string]string{
			"code": "internal_error", "message": "Не удалось загрузить свободное время",
		}), nil
	}
	if snapshot.callType.status == generated.DISABLED {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "call_type_unavailable", "message": "Тип звонка временно недоступен",
		}), nil
	}

	return domainResponse(http.StatusOK, generated.AvailabilityResponseBody{
		GeneratedAt: generatedAt,
		Timezone:    "Europe/Moscow",
		CallType: generated.AvailabilityResponseBodyCallType{
			Slug: snapshot.callType.slug, Name: snapshot.callType.name,
			Description: snapshot.callType.description, DurationMinutes: snapshot.callType.durationMinutes,
			CallTypeVersion: strconv.FormatInt(snapshot.callType.version, 10),
		},
		Days: buildAvailabilityDays(snapshot, observedAt),
	}), nil
}

var moscowLocation = time.FixedZone("Europe/Moscow", 3*60*60)

func moscowDayStart(value time.Time) time.Time {
	local := value.In(moscowLocation)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, moscowLocation)
}

func buildAvailabilityDays(snapshot availabilitySnapshot, generatedAt time.Time) []generated.AvailabilityDay {
	days := make([]generated.AvailabilityDay, 0, 14)
	dayStart := moscowDayStart(generatedAt)
	for dayOffset := range 14 {
		date := dayStart.AddDate(0, 0, dayOffset)
		weekday := int(date.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		available := intersectIntervals(
			mergeIntervals(snapshot.workingHours[weekday]),
			mergeIntervals(snapshot.schedule[weekday]),
		)
		slots := make([]generated.AvailabilitySlot, 0)
		seen := make(map[int64]struct{})
		for _, interval := range available {
			firstMinute := ((interval.start + 4) / 5) * 5
			for minute := firstMinute; minute+int(snapshot.callType.durationMinutes) <= interval.end; minute += 5 {
				startsAt := date.Add(time.Duration(minute) * time.Minute)
				if startsAt.Before(generatedAt) {
					continue
				}
				endsAt := startsAt.Add(time.Duration(snapshot.callType.durationMinutes) * time.Minute)
				if overlapsExistingCall(startsAt, endsAt, snapshot.existingCalls) {
					continue
				}
				if _, exists := seen[startsAt.Unix()]; exists {
					continue
				}
				seen[startsAt.Unix()] = struct{}{}
				slots = append(slots, generated.AvailabilitySlot{StartsAt: startsAt, EndsAt: endsAt})
			}
		}
		sort.Slice(slots, func(left, right int) bool {
			return slots[left].StartsAt.Before(slots[right].StartsAt)
		})
		days = append(days, generated.AvailabilityDay{Date: date.Format(time.DateOnly), Slots: slots})
	}
	return days
}

func mergeIntervals(intervals []minuteInterval) []minuteInterval {
	if len(intervals) == 0 {
		return nil
	}
	merged := make([]minuteInterval, 0, len(intervals))
	for _, interval := range intervals {
		if len(merged) == 0 || interval.start > merged[len(merged)-1].end {
			merged = append(merged, interval)
			continue
		}
		if interval.end > merged[len(merged)-1].end {
			merged[len(merged)-1].end = interval.end
		}
	}
	return merged
}

func intersectIntervals(left, right []minuteInterval) []minuteInterval {
	intersections := make([]minuteInterval, 0)
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(left) && rightIndex < len(right) {
		start := max(left[leftIndex].start, right[rightIndex].start)
		end := min(left[leftIndex].end, right[rightIndex].end)
		if start < end {
			intersections = append(intersections, minuteInterval{start: start, end: end})
		}
		if left[leftIndex].end < right[rightIndex].end {
			leftIndex++
		} else {
			rightIndex++
		}
	}
	return intersections
}

func overlapsExistingCall(start, end time.Time, calls []callInterval) bool {
	for _, existing := range calls {
		if start.Before(existing.end) && existing.start.Before(end) {
			return true
		}
	}
	return false
}

func domainResponse(status int, body any) generated.ImplResponse {
	return generated.ResponseWithHeaders(status, map[string][]string{"Cache-Control": {"no-store"}}, body)
}

type routeSubset struct {
	routes []generated.Route
}

func (r routeSubset) Routes() generated.Routes {
	routes := make(generated.Routes, len(r.routes))
	for _, route := range r.routes {
		routes[route.Name] = route
	}
	return routes
}

func (r routeSubset) OrderedRoutes() []generated.Route {
	return r.routes
}
