package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

type adminCallTypesService struct {
	pool *pgxpool.Pool
}

type normalizedCallType struct {
	name            string
	nameKey         string
	description     *string
	durationMinutes int32
	status          generated.CallTypeStatus
	scheduleID      string
}

var russianTransliteration = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "",
	'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func NewAdminCallTypesService(pool *pgxpool.Pool) generated.AdminCallTypesAPIServicer {
	return adminCallTypesService{pool: pool}
}

func (s adminCallTypesService) ListAdminCallTypes(ctx context.Context) (generated.ImplResponse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ct.id, ct.slug, ct.name, ct.description, ct.duration_minutes, ct.status,
		       s.id, s.name, ct.version,
		       EXISTS (SELECT 1 FROM calls c WHERE c.call_type_id = ct.id)
		FROM call_types ct
		JOIN schedules s ON s.id = ct.schedule_id
		WHERE ct.deleted_at IS NULL
		ORDER BY CASE ct.status WHEN 'active' THEN 0 ELSE 1 END,
		         ct.duration_minutes, ct.name_key, ct.id
	`)
	if err != nil {
		return internalCallTypesError(ctx, "list", err), nil
	}
	defer rows.Close()
	items := make([]generated.AdminCallType, 0)
	for rows.Next() {
		item, err := scanAdminCallType(rows)
		if err != nil {
			return internalCallTypesError(ctx, "scan list", err), nil
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return internalCallTypesError(ctx, "iterate list", err), nil
	}
	return domainResponse(http.StatusOK, generated.AdminCallTypeList{Items: items}), nil
}

func (s adminCallTypesService) CreateAdminCallType(
	ctx context.Context,
	request generated.CreateCallTypeRequest,
) (generated.ImplResponse, error) {
	input, fieldErrors := validateCallTypeRequest(
		request.Name, request.Description, request.DurationMinutes, generated.ACTIVE, request.ScheduleId,
	)
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalCallTypesError(ctx, "begin create", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()
	exists, err := scheduleExists(ctx, tx, input.scheduleID)
	if err != nil {
		return internalCallTypesError(ctx, "read schedule", err), nil
	}
	if !exists {
		return scheduleUnavailableResponse(), nil
	}
	id, err := storage.NewUUIDv7()
	if err != nil {
		return internalCallTypesError(ctx, "generate id", err), nil
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(29129)`); err != nil {
		return internalCallTypesError(ctx, "lock slug namespace", err), nil
	}
	slug, err := nextCallTypeSlug(ctx, tx, input.name, id)
	if err != nil {
		return internalCallTypesError(ctx, "generate slug", err), nil
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO call_types (id, schedule_id, slug, name, name_key, description, duration_minutes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
	`, id, input.scheduleID, slug, input.name, input.nameKey, input.description, input.durationMinutes)
	if err != nil {
		if isUniqueViolation(err) {
			return callTypeNameConflictResponse(), nil
		}
		return internalCallTypesError(ctx, "insert", err), nil
	}
	item, err := readAdminCallType(ctx, tx, id)
	if err != nil {
		return internalCallTypesError(ctx, "read created", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalCallTypesError(ctx, "commit create", err), nil
	}
	return callTypeResponse(http.StatusCreated, item), nil
}

func (s adminCallTypesService) UpdateAdminCallType(
	ctx context.Context,
	id string,
	ifMatch string,
	request generated.UpdateCallTypeRequest,
) (generated.ImplResponse, error) {
	if !resourceIDPattern.MatchString(id) {
		return validationResponse([]generated.FieldError{fieldError("path.id", "invalid", "Должен быть UUIDv7")}), nil
	}
	expectedVersion, response, ok := callTypePrecondition(ifMatch)
	if !ok {
		return response, nil
	}
	input, fieldErrors := validateCallTypeRequest(
		request.Name, request.Description, request.DurationMinutes, request.Status, request.ScheduleId,
	)
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalCallTypesError(ctx, "begin update", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentVersion int64
	err = tx.QueryRow(ctx, `SELECT version FROM call_types WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return callTypeNotFoundResponse(), nil
	}
	if err != nil {
		return internalCallTypesError(ctx, "lock update", err), nil
	}
	if currentVersion != expectedVersion {
		return callTypeVersionConflictResponse(), nil
	}
	exists, err := scheduleExists(ctx, tx, input.scheduleID)
	if err != nil {
		return internalCallTypesError(ctx, "read schedule", err), nil
	}
	if !exists {
		return scheduleUnavailableResponse(), nil
	}
	_, err = tx.Exec(ctx, `
		UPDATE call_types
		SET schedule_id = $2, name = $3, name_key = $4, description = $5,
		    duration_minutes = $6, status = $7, version = version + 1, updated_at = now()
		WHERE id = $1
	`, id, input.scheduleID, input.name, input.nameKey, input.description, input.durationMinutes, input.status)
	if err != nil {
		if isUniqueViolation(err) {
			return callTypeNameConflictResponse(), nil
		}
		return internalCallTypesError(ctx, "update", err), nil
	}
	item, err := readAdminCallType(ctx, tx, id)
	if err != nil {
		return internalCallTypesError(ctx, "read updated", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalCallTypesError(ctx, "commit update", err), nil
	}
	return callTypeResponse(http.StatusOK, item), nil
}

func (s adminCallTypesService) DeleteAdminCallType(
	ctx context.Context,
	id string,
	ifMatch string,
) (generated.ImplResponse, error) {
	if !resourceIDPattern.MatchString(id) {
		return validationResponse([]generated.FieldError{fieldError("path.id", "invalid", "Должен быть UUIDv7")}), nil
	}
	expectedVersion, response, ok := callTypePrecondition(ifMatch)
	if !ok {
		return response, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return internalCallTypesError(ctx, "begin delete", err), nil
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentVersion int64
	err = tx.QueryRow(ctx, `SELECT version FROM call_types WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return callTypeNotFoundResponse(), nil
	}
	if err != nil {
		return internalCallTypesError(ctx, "lock delete", err), nil
	}
	if currentVersion != expectedVersion {
		return callTypeVersionConflictResponse(), nil
	}
	var hasCalls bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM calls WHERE call_type_id = $1)`, id).Scan(&hasCalls); err != nil {
		return internalCallTypesError(ctx, "check history", err), nil
	}
	if hasCalls {
		return domainResponse(http.StatusConflict, map[string]string{
			"code": "call_type_has_calls", "message": "Тип звонка с историей нельзя удалить",
		}), nil
	}
	if _, err := tx.Exec(ctx, `UPDATE call_types SET deleted_at = now(), updated_at = now() WHERE id = $1`, id); err != nil {
		return internalCallTypesError(ctx, "soft delete", err), nil
	}
	if err := tx.Commit(ctx); err != nil {
		return internalCallTypesError(ctx, "commit delete", err), nil
	}
	return generated.ResponseWithHeaders(http.StatusNoContent, map[string][]string{
		"Cache-Control": {"no-store"},
	}, nil), nil
}

type callTypeScanner interface {
	Scan(...any) error
}

func scanAdminCallType(row callTypeScanner) (generated.AdminCallType, error) {
	var item generated.AdminCallType
	var schedule generated.ScheduleReference
	var version int64
	var hasCalls bool
	if err := row.Scan(
		&item.Id, &item.Slug, &item.Name, &item.Description, &item.DurationMinutes, &item.Status,
		&schedule.Id, &schedule.Name, &version, &hasCalls,
	); err != nil {
		return generated.AdminCallType{}, err
	}
	item.Schedule = schedule
	item.Version = strconv.FormatInt(version, 10)
	item.CanDelete = !hasCalls
	if hasCalls {
		reason := generated.CALL_TYPE_HAS_CALLS
		item.DeleteBlockedReason = &reason
	}
	return item, nil
}

func readAdminCallType(ctx context.Context, tx pgx.Tx, id string) (generated.AdminCallType, error) {
	return scanAdminCallType(tx.QueryRow(ctx, `
		SELECT ct.id, ct.slug, ct.name, ct.description, ct.duration_minutes, ct.status,
		       s.id, s.name, ct.version,
		       EXISTS (SELECT 1 FROM calls c WHERE c.call_type_id = ct.id)
		FROM call_types ct
		JOIN schedules s ON s.id = ct.schedule_id
		WHERE ct.id = $1 AND ct.deleted_at IS NULL
	`, id))
}

func validateCallTypeRequest(
	name string,
	description *string,
	durationMinutes int32,
	status generated.CallTypeStatus,
	scheduleID string,
) (normalizedCallType, []generated.FieldError) {
	normalizedName := norm.NFC.String(strings.TrimSpace(name))
	fieldErrors := make([]generated.FieldError, 0)
	if utf8.RuneCountInString(normalizedName) < 1 || utf8.RuneCountInString(normalizedName) > 80 {
		fieldErrors = append(fieldErrors, fieldError("body.name", "invalid_length", "Название должно содержать от 1 до 80 символов"))
	}
	var normalizedDescription *string
	if description != nil {
		value := norm.NFC.String(strings.TrimSpace(*description))
		if utf8.RuneCountInString(value) > 500 {
			fieldErrors = append(fieldErrors, fieldError("body.description", "invalid_length", "Описание должно содержать не более 500 символов"))
		} else if value != "" {
			normalizedDescription = &value
		}
	}
	if durationMinutes < 15 || durationMinutes > 240 || durationMinutes%5 != 0 {
		fieldErrors = append(fieldErrors, fieldError("body.duration_minutes", "invalid", "Длительность должна быть от 15 до 240 минут с шагом 5"))
	}
	if !status.IsValid() {
		fieldErrors = append(fieldErrors, fieldError("body.status", "invalid", "Статус должен быть active или disabled"))
	}
	if !resourceIDPattern.MatchString(scheduleID) {
		fieldErrors = append(fieldErrors, fieldError("body.schedule_id", "invalid", "Должен быть UUIDv7"))
	}
	return normalizedCallType{
		name: normalizedName, nameKey: norm.NFC.String(cases.Fold().String(normalizedName)),
		description: normalizedDescription, durationMinutes: durationMinutes,
		status: status, scheduleID: scheduleID,
	}, fieldErrors
}

func scheduleExists(ctx context.Context, tx pgx.Tx, scheduleID string) (bool, error) {
	var found int
	err := tx.QueryRow(ctx, `SELECT 1 FROM schedules WHERE id = $1 FOR KEY SHARE`, scheduleID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func nextCallTypeSlug(ctx context.Context, tx pgx.Tx, name, id string) (string, error) {
	base := callTypeSlugBase(name)
	if base == "" {
		base = "call-" + strings.ReplaceAll(id, "-", "")[:12]
	}
	if len(base) > 100 {
		base = strings.Trim(base[:100], "-")
	}
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			ending := fmt.Sprintf("-%d", suffix)
			trimmed := strings.TrimRight(base[:min(len(base), 100-len(ending))], "-")
			candidate = trimmed + ending
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM call_types WHERE slug = $1)`, candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
}

func callTypeSlugBase(value string) string {
	var result strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(norm.NFC.String(value)) {
		if piece, recognized := russianTransliteration[r]; recognized {
			result.WriteString(piece)
			if piece != "" {
				lastDash = false
			}
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			result.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(result.String(), "-")
}

func callTypePrecondition(ifMatch string) (int64, generated.ImplResponse, bool) {
	if ifMatch == "" {
		return 0, domainResponse(http.StatusPreconditionRequired, map[string]string{
			"code": "precondition_required", "message": "Укажите текущую версию типа звонка",
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

func callTypeResponse(status int, item generated.AdminCallType) generated.ImplResponse {
	return generated.ResponseWithHeaders(status, map[string][]string{
		"Cache-Control": {"no-store"}, "ETag": {`"` + item.Version + `"`},
	}, item)
}

func callTypeNameConflictResponse() generated.ImplResponse {
	return domainResponse(http.StatusConflict, map[string]string{
		"code": "call_type_name_conflict", "message": "Тип звонка с таким названием уже существует",
	})
}

func scheduleUnavailableResponse() generated.ImplResponse {
	return domainResponse(http.StatusConflict, map[string]string{
		"code": "schedule_unavailable", "message": "Выбранное расписание недоступно",
	})
}

func callTypeNotFoundResponse() generated.ImplResponse {
	return domainResponse(http.StatusNotFound, map[string]string{
		"code": "call_type_not_found", "message": "Тип звонка не найден",
	})
}

func callTypeVersionConflictResponse() generated.ImplResponse {
	return domainResponse(http.StatusPreconditionFailed, map[string]string{
		"code": "version_conflict", "message": "Тип звонка уже изменён",
	})
}

func internalCallTypesError(ctx context.Context, operation string, err error) generated.ImplResponse {
	slog.ErrorContext(ctx, "admin call types", "operation", operation, "error", err)
	return domainResponse(http.StatusInternalServerError, map[string]string{
		"code": "internal_error", "message": "Не удалось обработать тип звонка",
	})
}
