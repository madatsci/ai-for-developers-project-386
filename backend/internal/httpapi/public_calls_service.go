package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/unicode/norm"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

var (
	idempotencyKeyPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	callTypeSlugPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type publicCallsService struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPublicCallsService(pool *pgxpool.Pool) generated.PublicCallsAPIServicer {
	return newPublicCallsService(pool, time.Now)
}

func newPublicCallsService(pool *pgxpool.Pool, now func() time.Time) generated.PublicCallsAPIServicer {
	return publicCallsService{pool: pool, now: now}
}

type normalizedCallRequest struct {
	callTypeSlug    string
	callTypeVersion string
	startsAt        time.Time
	visitorName     string
	visitorEmail    string
	visitorNotes    *string
}

type lockedCallType struct {
	id              string
	slug            string
	name            string
	durationMinutes int32
	status          generated.CallTypeStatus
	version         int64
	scheduleID      string
	workingHoursID  string
}

func (s publicCallsService) CreateCall(
	ctx context.Context,
	idempotencyKey string,
	request generated.CreateCallRequest,
) (generated.ImplResponse, error) {
	normalized, fieldErrors := normalizeCallRequest(idempotencyKey, request)
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}
	fingerprint, err := callFingerprint(normalized)
	if err != nil {
		return internalCallError(ctx, "fingerprint call request", err), nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalCallError(ctx, "begin call transaction", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, idempotencyKey); err != nil {
		return internalCallError(ctx, "lock idempotency key", err), nil
	}
	if response, found, conflict, err := readIdempotentResponse(ctx, tx, idempotencyKey, fingerprint); err != nil {
		return internalCallError(ctx, "read idempotency record", err), nil
	} else if conflict {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "idempotency_conflict", "message": "Ключ уже связан с другим запросом",
		}), nil
	} else if found {
		if err := tx.Commit(ctx); err != nil {
			return internalCallError(ctx, "commit idempotent response", err), nil
		}
		return domainResponse(http.StatusCreated, response), nil
	}

	callType, err := lockCallTypeGraph(ctx, tx, normalized.callTypeSlug)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "call_type_unavailable", "message": "Тип звонка недоступен",
		}), nil
	}
	if err != nil {
		return internalCallError(ctx, "lock call type graph", err), nil
	}
	if callType.status != generated.ACTIVE {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "call_type_unavailable", "message": "Тип звонка недоступен",
		}), nil
	}
	if normalized.callTypeVersion != strconv.FormatInt(callType.version, 10) {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "call_type_changed", "message": "Тип звонка изменился",
		}), nil
	}

	endsAt := normalized.startsAt.Add(time.Duration(callType.durationMinutes) * time.Minute)
	available, err := callSlotAvailable(ctx, tx, callType, normalized.startsAt, endsAt, s.now())
	if err != nil {
		return internalCallError(ctx, "check slot availability", err), nil
	}
	if !available {
		return slotUnavailableResponse(), nil
	}

	callID, err := storage.NewUUIDv7()
	if err != nil {
		return internalCallError(ctx, "generate call id", err), nil
	}
	created := generated.CreatedCall{
		Id: callID, CallTypeSlug: callType.slug, CallTypeName: callType.name,
		DurationMinutes: callType.durationMinutes,
		StartsAt:        normalized.startsAt.In(moscowLocation), EndsAt: endsAt.In(moscowLocation),
		Timezone: "Europe/Moscow", VisitorName: normalized.visitorName,
		VisitorEmail: normalized.visitorEmail, VisitorNotes: normalized.visitorNotes,
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO calls (
			id, call_type_id, call_type_name, duration_minutes, starts_at,
			visitor_name, visitor_email, visitor_notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, callID, callType.id, callType.name, callType.durationMinutes, normalized.startsAt,
		normalized.visitorName, normalized.visitorEmail, normalized.visitorNotes)
	if err != nil {
		if isExclusionViolation(err) {
			return slotUnavailableResponse(), nil
		}
		return internalCallError(ctx, "insert call", err), nil
	}
	responseSnapshot, err := json.Marshal(created)
	if err != nil {
		return internalCallError(ctx, "encode call snapshot", err), nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO call_idempotency (key, fingerprint, call_id, response_snapshot)
		VALUES ($1, $2, $3, $4)
	`, idempotencyKey, fingerprint, callID, responseSnapshot); err != nil {
		return internalCallError(ctx, "insert idempotency record", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalCallError(ctx, "commit call transaction", err), nil
	}
	return domainResponse(http.StatusCreated, created), nil
}

func normalizeCallRequest(key string, request generated.CreateCallRequest) (normalizedCallRequest, []generated.FieldError) {
	result := normalizedCallRequest{
		callTypeSlug: request.CallTypeSlug, callTypeVersion: request.CallTypeVersion,
		startsAt: request.StartsAt,
	}
	fieldErrors := make([]generated.FieldError, 0)
	if !idempotencyKeyPattern.MatchString(key) {
		fieldErrors = append(fieldErrors, fieldError("headers.Idempotency-Key", "invalid", "Должен быть UUIDv4"))
	}
	result.visitorName = norm.NFC.String(strings.TrimSpace(request.VisitorName))
	if utf8.RuneCountInString(result.visitorName) < 1 || utf8.RuneCountInString(result.visitorName) > 100 || strings.ContainsAny(result.visitorName, "\r\n") || containsControl(result.visitorName) {
		fieldErrors = append(fieldErrors, fieldError("body.visitor_name", "invalid", "Укажите однострочное имя длиной до 100 символов"))
	}
	result.visitorEmail = normalizeEmail(request.VisitorEmail)
	if !validVisitorEmail(result.visitorEmail) {
		fieldErrors = append(fieldErrors, fieldError("body.visitor_email", "invalid", "Укажите корректный ASCII email длиной до 254 символов"))
	}
	if request.VisitorNotes != nil {
		notes := strings.ReplaceAll(strings.ReplaceAll(*request.VisitorNotes, "\r\n", "\n"), "\r", "\n")
		notes = norm.NFC.String(strings.TrimSpace(notes))
		if notes != "" {
			result.visitorNotes = &notes
		}
	}
	if result.visitorNotes != nil && utf8.RuneCountInString(*result.visitorNotes) > 1000 {
		fieldErrors = append(fieldErrors, fieldError("body.visitor_notes", "too_long", "Комментарий должен быть не длиннее 1000 символов"))
	}
	if request.CallTypeSlug == "" {
		fieldErrors = append(fieldErrors, fieldError("body.call_type_slug", "required", "Укажите тип звонка"))
	} else if len(request.CallTypeSlug) > 100 || !callTypeSlugPattern.MatchString(request.CallTypeSlug) {
		fieldErrors = append(fieldErrors, fieldError("body.call_type_slug", "invalid", "Укажите корректный slug типа звонка"))
	}
	if request.CallTypeVersion == "" {
		fieldErrors = append(fieldErrors, fieldError("body.call_type_version", "required", "Укажите версию типа звонка"))
	}
	moscowStart := request.StartsAt.In(moscowLocation)
	if request.StartsAt.IsZero() || request.StartsAt.Second() != 0 || request.StartsAt.Nanosecond() != 0 || moscowStart.Minute()%5 != 0 {
		fieldErrors = append(fieldErrors, fieldError("body.starts_at", "invalid", "Начало должно быть на пятиминутной сетке"))
	}
	return result, fieldErrors
}

func normalizeEmail(value string) string {
	value = strings.TrimSpace(value)
	at := strings.LastIndexByte(value, '@')
	if at < 0 {
		return value
	}
	return value[:at+1] + strings.ToLower(value[at+1:])
}

func validVisitorEmail(value string) bool {
	if len(value) < 3 || len(value) > 254 || strings.Count(value, "@") != 1 {
		return false
	}
	for _, character := range value {
		if character > unicode.MaxASCII || unicode.IsControl(character) {
			return false
		}
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Name == "" && address.Address == value
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func fieldError(path, code, message string) generated.FieldError {
	return generated.FieldError{Path: path, Code: code, Message: message}
}

func validationResponse(fieldErrors []generated.FieldError) generated.ImplResponse {
	return domainResponse(http.StatusUnprocessableEntity, generated.ValidationError{
		Code: "validation_error", Message: "Проверьте поля запроса", FieldErrors: fieldErrors,
	})
}

func callFingerprint(request normalizedCallRequest) (string, error) {
	payload := struct {
		CallTypeSlug    string  `json:"call_type_slug"`
		CallTypeVersion string  `json:"call_type_version"`
		StartsAt        string  `json:"starts_at"`
		VisitorName     string  `json:"visitor_name"`
		VisitorEmail    string  `json:"visitor_email"`
		VisitorNotes    *string `json:"visitor_notes"`
	}{
		request.callTypeSlug, request.callTypeVersion, request.startsAt.UTC().Format(time.RFC3339),
		request.visitorName, request.visitorEmail, request.visitorNotes,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func readIdempotentResponse(
	ctx context.Context,
	tx pgx.Tx,
	key string,
	fingerprint string,
) (generated.CreatedCall, bool, bool, error) {
	var storedFingerprint string
	var snapshot []byte
	err := tx.QueryRow(ctx, `SELECT fingerprint, response_snapshot FROM call_idempotency WHERE key = $1`, key).Scan(&storedFingerprint, &snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return generated.CreatedCall{}, false, false, nil
	}
	if err != nil {
		return generated.CreatedCall{}, false, false, err
	}
	if storedFingerprint != fingerprint {
		return generated.CreatedCall{}, false, true, nil
	}
	var response generated.CreatedCall
	if err := json.Unmarshal(snapshot, &response); err != nil {
		return generated.CreatedCall{}, false, false, err
	}
	return response, true, false, nil
}

func lockCallTypeGraph(ctx context.Context, tx pgx.Tx, slug string) (lockedCallType, error) {
	var item lockedCallType
	err := tx.QueryRow(ctx, `
		SELECT ct.id, ct.slug, ct.name, ct.duration_minutes, ct.status, ct.version,
		       s.id, wh.id
		FROM call_types ct
		JOIN schedules s ON s.id = ct.schedule_id
		JOIN working_hours wh ON wh.organizer_id = s.organizer_id
		WHERE ct.slug = $1 AND ct.deleted_at IS NULL
		FOR UPDATE OF ct, s, wh
	`, slug).Scan(&item.id, &item.slug, &item.name, &item.durationMinutes, &item.status,
		&item.version, &item.scheduleID, &item.workingHoursID)
	return item, err
}

func callSlotAvailable(
	ctx context.Context,
	tx pgx.Tx,
	callType lockedCallType,
	startsAt time.Time,
	endsAt time.Time,
	now time.Time,
) (bool, error) {
	localStart := startsAt.In(moscowLocation)
	localEnd := endsAt.In(moscowLocation)
	windowStart := moscowDayStart(now)
	windowEnd := windowStart.AddDate(0, 0, 14)
	if startsAt.Before(now) || startsAt.Before(windowStart) || !startsAt.Before(windowEnd) || localStart.YearDay() != localEnd.YearDay() || localStart.Year() != localEnd.Year() {
		return false, nil
	}
	weekday := int(localStart.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	workingHours, err := readWeeklyIntervals(ctx, tx, `
		SELECT day_of_week,
		       (extract(hour FROM start_time)::integer * 60 + extract(minute FROM start_time)::integer),
		       (extract(hour FROM end_time)::integer * 60 + extract(minute FROM end_time)::integer)
		FROM working_hour_intervals
		WHERE working_hours_id = $1 AND day_of_week = $2
		ORDER BY start_time
	`, callType.workingHoursID, weekday)
	if err != nil {
		return false, err
	}
	schedule, err := readWeeklyIntervals(ctx, tx, `
		SELECT day_of_week,
		       (extract(hour FROM start_time)::integer * 60 + extract(minute FROM start_time)::integer),
		       (extract(hour FROM end_time)::integer * 60 + extract(minute FROM end_time)::integer)
		FROM schedule_intervals
		WHERE schedule_id = $1 AND day_of_week = $2
		ORDER BY start_time
	`, callType.scheduleID, weekday)
	if err != nil {
		return false, err
	}
	available := intersectIntervals(mergeIntervals(workingHours[weekday]), mergeIntervals(schedule[weekday]))
	startMinute := localStart.Hour()*60 + localStart.Minute()
	endMinute := localEnd.Hour()*60 + localEnd.Minute()
	withinAvailability := false
	for _, interval := range available {
		if interval.start <= startMinute && endMinute <= interval.end {
			withinAvailability = true
			break
		}
	}
	if !withinAvailability {
		return false, nil
	}
	var overlaps bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM calls WHERE starts_at < $2 AND ends_at > $1
	)`, startsAt, endsAt).Scan(&overlaps); err != nil {
		return false, err
	}
	return !overlaps, nil
}

func isExclusionViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23P01"
}

func slotUnavailableResponse() generated.ImplResponse {
	return domainResponse(http.StatusConflict, map[string]string{
		"code": "slot_unavailable", "message": "Выбранное время больше недоступно",
	})
}

func internalCallError(ctx context.Context, operation string, err error) generated.ImplResponse {
	slog.ErrorContext(ctx, operation, "error", err)
	return domainResponse(http.StatusInternalServerError, map[string]string{
		"code": "internal_error", "message": "Не удалось создать звонок",
	})
}
