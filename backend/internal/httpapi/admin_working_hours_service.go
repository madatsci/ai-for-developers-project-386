package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

type adminWorkingHoursService struct {
	pool *pgxpool.Pool
}

func NewAdminWorkingHoursService(pool *pgxpool.Pool) generated.AdminWorkingHoursAPIServicer {
	return adminWorkingHoursService{pool: pool}
}

func (s adminWorkingHoursService) GetAdminWorkingHours(ctx context.Context) (generated.ImplResponse, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return internalWorkingHoursReadError(ctx, "begin read", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var workingHoursID string
	var version int64
	if err := tx.QueryRow(ctx, `
		SELECT wh.id, wh.version
		FROM working_hours wh
		JOIN organizers o ON o.id = wh.organizer_id
		WHERE o.singleton
	`).Scan(&workingHoursID, &version); err != nil {
		return internalWorkingHoursReadError(ctx, "read version", err), nil
	}

	intervals, err := readWeeklyIntervals(ctx, tx, `
		SELECT day_of_week,
		       (extract(hour FROM start_time)::integer * 60 + extract(minute FROM start_time)::integer),
		       (extract(hour FROM end_time)::integer * 60 + extract(minute FROM end_time)::integer)
		FROM working_hour_intervals
		WHERE working_hours_id = $1
		ORDER BY day_of_week, start_time
	`, workingHoursID)
	if err != nil {
		return internalWorkingHoursReadError(ctx, "read intervals", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalWorkingHoursReadError(ctx, "commit read", err), nil
	}

	return workingHoursResponse(version, intervals), nil
}

func (s adminWorkingHoursService) UpdateAdminWorkingHours(
	ctx context.Context,
	ifMatch string,
	request generated.UpdateWorkingHoursRequest,
) (generated.ImplResponse, error) {
	expectedVersion, valid := parseEntityTag(ifMatch)
	if ifMatch == "" {
		return domainResponse(http.StatusPreconditionRequired, map[string]string{
			"code": "precondition_required", "message": "Укажите текущую версию режима работы",
		}), nil
	}
	if !valid {
		return domainResponse(http.StatusBadRequest, map[string]string{
			"code": "invalid_request", "message": "Некорректный If-Match",
		}), nil
	}
	intervals, fieldErrors := validateWeeklyDays(request.Days)
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return internalWorkingHoursError(ctx, "begin update", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var workingHoursID string
	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT wh.id, wh.version
		FROM working_hours wh
		JOIN organizers o ON o.id = wh.organizer_id
		WHERE o.singleton
		FOR UPDATE OF wh
	`).Scan(&workingHoursID, &currentVersion)
	if err != nil {
		return internalWorkingHoursError(ctx, "lock for update", err), nil
	}
	if currentVersion != expectedVersion {
		return domainResponse(http.StatusPreconditionFailed, map[string]string{
			"code": "version_conflict", "message": "Режим работы уже изменён",
		}), nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM working_hour_intervals WHERE working_hours_id = $1`, workingHoursID); err != nil {
		return internalWorkingHoursError(ctx, "replace intervals", err), nil
	}
	for day := 1; day <= 7; day++ {
		for _, interval := range intervals[day] {
			if _, err := tx.Exec(ctx, `
				INSERT INTO working_hour_intervals (working_hours_id, day_of_week, start_time, end_time)
				VALUES ($1, $2, $3, $4)
			`, workingHoursID, day, formatLocalTime(interval.start), formatLocalTime(interval.end)); err != nil {
				return internalWorkingHoursError(ctx, "insert interval", err), nil
			}
		}
	}
	newVersion := currentVersion + 1
	if _, err := tx.Exec(ctx, `
		UPDATE working_hours SET version = $2, updated_at = now() WHERE id = $1
	`, workingHoursID, newVersion); err != nil {
		return internalWorkingHoursError(ctx, "increment version", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalWorkingHoursError(ctx, "commit update", err), nil
	}
	return workingHoursResponse(newVersion, intervals), nil
}

var strongEntityTagPattern = regexp.MustCompile(`^"([1-9][0-9]*)"$`)

func parseEntityTag(value string) (int64, bool) {
	match := strongEntityTagPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, false
	}
	version, err := strconv.ParseInt(match[1], 10, 64)
	return version, err == nil
}

func validateWeeklyDays(days []generated.WeeklyDay) (map[int][]minuteInterval, []generated.FieldError) {
	if len(days) != 7 {
		return nil, []generated.FieldError{fieldError("body.days", "invalid_length", "Передайте все семь дней недели")}
	}
	result := make(map[int][]minuteInterval, 7)
	seen := make(map[int]bool, 7)
	for dayIndex, day := range days {
		dayNumber := int(day.DayOfWeek)
		path := fmt.Sprintf("body.days.%d", dayIndex)
		if dayNumber < 1 || dayNumber > 7 {
			return nil, []generated.FieldError{fieldError(path+".day_of_week", "invalid", "День недели должен быть от 1 до 7")}
		}
		if seen[dayNumber] {
			return nil, []generated.FieldError{fieldError(path+".day_of_week", "duplicate", "День недели указан повторно")}
		}
		seen[dayNumber] = true
		for intervalIndex, interval := range day.Intervals {
			intervalPath := fmt.Sprintf("%s.intervals.%d", path, intervalIndex)
			start, startValid := parseLocalTime(interval.StartTime)
			end, endValid := parseLocalTime(interval.EndTime)
			if !startValid {
				return nil, []generated.FieldError{fieldError(intervalPath+".start_time", "invalid", "Время должно быть на пятиминутной сетке")}
			}
			if !endValid {
				return nil, []generated.FieldError{fieldError(intervalPath+".end_time", "invalid", "Время должно быть на пятиминутной сетке")}
			}
			if end <= start {
				return nil, []generated.FieldError{fieldError(intervalPath+".end_time", "invalid_range", "Конец должен быть позже начала")}
			}
			if end-start < 15 {
				return nil, []generated.FieldError{fieldError(intervalPath+".end_time", "too_short", "Минимальная длительность — 15 минут")}
			}
			result[dayNumber] = append(result[dayNumber], minuteInterval{start: start, end: end})
		}
		sort.Slice(result[dayNumber], func(left, right int) bool {
			return result[dayNumber][left].start < result[dayNumber][right].start
		})
		for index := 1; index < len(result[dayNumber]); index++ {
			if result[dayNumber][index].start < result[dayNumber][index-1].end {
				return nil, []generated.FieldError{fieldError(path+".intervals", "overlap", "Интервалы дня не должны пересекаться")}
			}
		}
	}
	return result, nil
}

func parseLocalTime(value string) (int, bool) {
	parsed, err := time.Parse("15:04", value)
	if err != nil || parsed.Format("15:04") != value || parsed.Minute()%5 != 0 {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func internalWorkingHoursError(ctx context.Context, operation string, err error) generated.ImplResponse {
	if errors.Is(err, context.Canceled) {
		slog.InfoContext(ctx, "admin working hours request canceled", "operation", operation)
	} else {
		slog.ErrorContext(ctx, "admin working hours", "operation", operation, "error", err)
	}
	return domainResponse(http.StatusInternalServerError, map[string]string{
		"code": "internal_error", "message": "Не удалось сохранить режим работы",
	})
}

func internalWorkingHoursReadError(ctx context.Context, operation string, err error) generated.ImplResponse {
	slog.ErrorContext(ctx, "get admin working hours", "operation", operation, "error", err)
	return domainResponse(http.StatusInternalServerError, map[string]string{
		"code": "internal_error", "message": "Не удалось загрузить режим работы",
	})
}

func workingHoursResponse(version int64, intervals map[int][]minuteInterval) generated.ImplResponse {
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
	versionString := strconv.FormatInt(version, 10)
	return generated.ResponseWithHeaders(http.StatusOK, map[string][]string{
		"Cache-Control": {"no-store"},
		"ETag":          {`"` + versionString + `"`},
	}, generated.WorkingHours{
		Timezone: "Europe/Moscow",
		Days:     days,
		Version:  versionString,
	})
}

func formatLocalTime(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
