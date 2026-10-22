package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

type adminSchedulesService struct {
	pool *pgxpool.Pool
}

var resourceIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func NewAdminSchedulesService(pool *pgxpool.Pool) generated.AdminSchedulesAPIServicer {
	return adminSchedulesService{pool: pool}
}

func (s adminSchedulesService) ListAdminSchedules(ctx context.Context) (generated.ImplResponse, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return internalSchedulesError(ctx, "begin list", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT s.id, s.name, s.version
		FROM schedules s
		JOIN organizers o ON o.id = s.organizer_id
		WHERE o.singleton
		ORDER BY s.name_key, s.id
	`)
	if err != nil {
		return internalSchedulesError(ctx, "list schedules", err), nil
	}
	type scheduleRoot struct {
		id      string
		name    string
		version int64
	}
	roots := make([]scheduleRoot, 0)
	for rows.Next() {
		var root scheduleRoot
		if err := rows.Scan(&root.id, &root.name, &root.version); err != nil {
			rows.Close()
			return internalSchedulesError(ctx, "scan schedule", err), nil
		}
		roots = append(roots, root)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return internalSchedulesError(ctx, "iterate schedules", err), nil
	}
	rows.Close()

	items := make([]generated.AdminSchedule, 0, len(roots))
	for _, root := range roots {
		item, err := readAdminSchedule(ctx, tx, root.id, root.name, root.version, len(roots))
		if err != nil {
			return internalSchedulesError(ctx, "read schedule details", err), nil
		}
		items = append(items, item)
	}
	if err := tx.Commit(ctx); err != nil {
		return internalSchedulesError(ctx, "commit list", err), nil
	}
	return domainResponse(http.StatusOK, generated.AdminScheduleList{Items: items}), nil
}

func (s adminSchedulesService) CreateAdminSchedule(
	ctx context.Context,
	request generated.CreateScheduleRequest,
) (generated.ImplResponse, error) {
	name, nameKey, intervals, fieldErrors := validateScheduleRequest(request.Name, request.Days)
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalSchedulesError(ctx, "begin create", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var organizerID string
	if err := tx.QueryRow(ctx, `SELECT id FROM organizers WHERE singleton`).Scan(&organizerID); err != nil {
		return internalSchedulesError(ctx, "read organizer", err), nil
	}
	id, err := storage.NewUUIDv7()
	if err != nil {
		return internalSchedulesError(ctx, "generate id", err), nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO schedules (id, organizer_id, name, name_key) VALUES ($1, $2, $3, $4)
	`, id, organizerID, name, nameKey); err != nil {
		if isUniqueViolation(err) {
			return scheduleNameConflictResponse(), nil
		}
		return internalSchedulesError(ctx, "insert schedule", err), nil
	}
	if err := replaceScheduleIntervals(ctx, tx, id, intervals); err != nil {
		return internalSchedulesError(ctx, "insert intervals", err), nil
	}
	var scheduleCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schedules WHERE organizer_id = $1`, organizerID).Scan(&scheduleCount); err != nil {
		return internalSchedulesError(ctx, "count schedules", err), nil
	}
	item, err := readAdminSchedule(ctx, tx, id, name, 1, scheduleCount)
	if err != nil {
		return internalSchedulesError(ctx, "read created schedule", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalSchedulesError(ctx, "commit create", err), nil
	}
	return scheduleResponse(http.StatusCreated, item), nil
}

func (s adminSchedulesService) UpdateAdminSchedule(
	ctx context.Context,
	id string,
	ifMatch string,
	request generated.UpdateScheduleRequest,
) (generated.ImplResponse, error) {
	if !resourceIDPattern.MatchString(id) {
		return validationResponse([]generated.FieldError{
			fieldError("path.id", "invalid", "Должен быть UUIDv7"),
		}), nil
	}
	expectedVersion, response, ok := schedulePrecondition(ifMatch)
	if !ok {
		return response, nil
	}
	name, nameKey, intervals, fieldErrors := validateScheduleRequest(request.Name, request.Days)
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalSchedulesError(ctx, "begin update", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var organizerID string
	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT organizer_id, version FROM schedules WHERE id = $1 FOR UPDATE
	`, id).Scan(&organizerID, &currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return scheduleNotFoundResponse(), nil
	}
	if err != nil {
		return internalSchedulesError(ctx, "lock update", err), nil
	}
	if currentVersion != expectedVersion {
		return scheduleVersionConflictResponse(), nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE schedules
		SET name = $2, name_key = $3, version = version + 1, updated_at = now()
		WHERE id = $1
	`, id, name, nameKey); err != nil {
		if isUniqueViolation(err) {
			return scheduleNameConflictResponse(), nil
		}
		return internalSchedulesError(ctx, "update schedule", err), nil
	}
	if err := replaceScheduleIntervals(ctx, tx, id, intervals); err != nil {
		return internalSchedulesError(ctx, "replace intervals", err), nil
	}
	var scheduleCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schedules WHERE organizer_id = $1`, organizerID).Scan(&scheduleCount); err != nil {
		return internalSchedulesError(ctx, "count schedules", err), nil
	}
	item, err := readAdminSchedule(ctx, tx, id, name, currentVersion+1, scheduleCount)
	if err != nil {
		return internalSchedulesError(ctx, "read updated schedule", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalSchedulesError(ctx, "commit update", err), nil
	}
	return scheduleResponse(http.StatusOK, item), nil
}

func (s adminSchedulesService) DeleteAdminSchedule(
	ctx context.Context,
	id string,
	ifMatch string,
) (generated.ImplResponse, error) {
	if !resourceIDPattern.MatchString(id) {
		return validationResponse([]generated.FieldError{
			fieldError("path.id", "invalid", "Должен быть UUIDv7"),
		}), nil
	}
	expectedVersion, response, ok := schedulePrecondition(ifMatch)
	if !ok {
		return response, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalSchedulesError(ctx, "begin delete", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var organizerID string
	if err := tx.QueryRow(ctx, `SELECT id FROM organizers WHERE singleton FOR UPDATE`).Scan(&organizerID); err != nil {
		return internalSchedulesError(ctx, "lock organizer", err), nil
	}
	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT version FROM schedules WHERE id = $1 AND organizer_id = $2 FOR UPDATE
	`, id, organizerID).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return scheduleNotFoundResponse(), nil
	}
	if err != nil {
		return internalSchedulesError(ctx, "lock delete", err), nil
	}
	if currentVersion != expectedVersion {
		return scheduleVersionConflictResponse(), nil
	}

	var assignedTypes int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM call_types WHERE schedule_id = $1`, id).Scan(&assignedTypes); err != nil {
		return internalSchedulesError(ctx, "count assigned types", err), nil
	}
	if assignedTypes > 0 {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "schedule_in_use", "message": "Расписание назначено типу звонка",
		}), nil
	}
	var scheduleCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM schedules WHERE organizer_id = $1`, organizerID).Scan(&scheduleCount); err != nil {
		return internalSchedulesError(ctx, "count schedules", err), nil
	}
	if scheduleCount <= 1 {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "last_schedule", "message": "Нельзя удалить последнее расписание",
		}), nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id); err != nil {
		return internalSchedulesError(ctx, "delete schedule", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalSchedulesError(ctx, "commit delete", err), nil
	}
	return generated.ResponseWithHeaders(http.StatusNoContent, map[string][]string{
		"Cache-Control": {"no-store"},
	}, nil), nil
}

func schedulePrecondition(ifMatch string) (int64, generated.ImplResponse, bool) {
	if ifMatch == "" {
		return 0, domainResponse(http.StatusPreconditionRequired, map[string]string{
			"code": "precondition_required", "message": "Укажите текущую версию расписания",
		}), false
	}
	expectedVersion, valid := parseEntityTag(ifMatch)
	if !valid {
		return 0, domainResponse(http.StatusBadRequest, map[string]string{
			"code": "invalid_request", "message": "Некорректный If-Match",
		}), false
	}
	return expectedVersion, generated.ImplResponse{}, true
}

func validateScheduleRequest(
	name string,
	days []generated.WeeklyDay,
) (string, string, map[int][]minuteInterval, []generated.FieldError) {
	normalizedName := norm.NFC.String(strings.TrimSpace(name))
	fieldErrors := make([]generated.FieldError, 0)
	if utf8.RuneCountInString(normalizedName) < 1 || utf8.RuneCountInString(normalizedName) > 80 {
		fieldErrors = append(fieldErrors, fieldError("body.name", "invalid_length", "Название должно содержать от 1 до 80 символов"))
	}
	intervals, intervalErrors := validateWeeklyDays(days)
	fieldErrors = append(fieldErrors, intervalErrors...)
	return normalizedName, norm.NFC.String(cases.Fold().String(normalizedName)), intervals, fieldErrors
}

func replaceScheduleIntervals(ctx context.Context, tx pgx.Tx, scheduleID string, intervals map[int][]minuteInterval) error {
	if _, err := tx.Exec(ctx, `DELETE FROM schedule_intervals WHERE schedule_id = $1`, scheduleID); err != nil {
		return err
	}
	for day := 1; day <= 7; day++ {
		for _, interval := range intervals[day] {
			if _, err := tx.Exec(ctx, `
				INSERT INTO schedule_intervals (schedule_id, day_of_week, start_time, end_time)
				VALUES ($1, $2, $3, $4)
			`, scheduleID, day, formatLocalTime(interval.start), formatLocalTime(interval.end)); err != nil {
				return err
			}
		}
	}
	return nil
}

func readAdminSchedule(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	name string,
	version int64,
	scheduleCount int,
) (generated.AdminSchedule, error) {
	intervals, err := readWeeklyIntervals(ctx, tx, `
		SELECT day_of_week,
		       (extract(hour FROM start_time)::integer * 60 + extract(minute FROM start_time)::integer),
		       (extract(hour FROM end_time)::integer * 60 + extract(minute FROM end_time)::integer)
		FROM schedule_intervals
		WHERE schedule_id = $1
		ORDER BY day_of_week, start_time
	`, id)
	if err != nil {
		return generated.AdminSchedule{}, err
	}
	rows, err := tx.Query(ctx, `
		SELECT id, name, status
		FROM call_types
		WHERE schedule_id = $1 AND deleted_at IS NULL
		ORDER BY name_key, id
	`, id)
	if err != nil {
		return generated.AdminSchedule{}, err
	}
	assignments := make([]generated.AssignedCallType, 0)
	for rows.Next() {
		var item generated.AssignedCallType
		if err := rows.Scan(&item.Id, &item.Name, &item.Status); err != nil {
			rows.Close()
			return generated.AdminSchedule{}, err
		}
		assignments = append(assignments, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return generated.AdminSchedule{}, err
	}
	rows.Close()

	var allAssignments int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM call_types WHERE schedule_id = $1`, id).Scan(&allAssignments); err != nil {
		return generated.AdminSchedule{}, err
	}
	var blockedReason *generated.ScheduleDeleteBlockedReason
	if allAssignments > 0 {
		reason := generated.SCHEDULE_IN_USE
		blockedReason = &reason
	} else if scheduleCount == 1 {
		reason := generated.LAST_SCHEDULE
		blockedReason = &reason
	}
	return generated.AdminSchedule{
		Id:                  id,
		Name:                name,
		Days:                scheduleDays(intervals),
		Version:             strconv.FormatInt(version, 10),
		AssignedCallTypes:   assignments,
		CanDelete:           blockedReason == nil,
		DeleteBlockedReason: blockedReason,
	}, nil
}

func scheduleDays(intervals map[int][]minuteInterval) []generated.WeeklyDay {
	days := make([]generated.WeeklyDay, 0, 7)
	for day := 1; day <= 7; day++ {
		dayIntervals := make([]generated.WeeklyInterval, 0, len(intervals[day]))
		for _, interval := range intervals[day] {
			dayIntervals = append(dayIntervals, generated.WeeklyInterval{
				StartTime: formatLocalTime(interval.start),
				EndTime:   formatLocalTime(interval.end),
			})
		}
		days = append(days, generated.WeeklyDay{DayOfWeek: int32(day), Intervals: dayIntervals})
	}
	return days
}

func scheduleResponse(status int, schedule generated.AdminSchedule) generated.ImplResponse {
	return generated.ResponseWithHeaders(status, map[string][]string{
		"Cache-Control": {"no-store"},
		"ETag":          {`"` + schedule.Version + `"`},
	}, schedule)
}

func scheduleNameConflictResponse() generated.ImplResponse {
	return domainResponse(http.StatusConflict, map[string]string{
		"code": "schedule_name_conflict", "message": "Расписание с таким названием уже существует",
	})
}

func scheduleNotFoundResponse() generated.ImplResponse {
	return domainResponse(http.StatusNotFound, map[string]string{
		"code": "schedule_not_found", "message": "Расписание не найдено",
	})
}

func scheduleVersionConflictResponse() generated.ImplResponse {
	return domainResponse(http.StatusPreconditionFailed, map[string]string{
		"code": "version_conflict", "message": "Расписание уже изменено",
	})
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

func internalSchedulesError(ctx context.Context, operation string, err error) generated.ImplResponse {
	slog.ErrorContext(ctx, "admin schedules", "operation", operation, "error", err)
	return domainResponse(http.StatusInternalServerError, map[string]string{
		"code": "internal_error", "message": "Не удалось обработать расписание",
	})
}
