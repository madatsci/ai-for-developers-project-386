package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

type adminCallsService struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

type adminCallsCursor struct {
	AsOf       string `json:"as_of"`
	Period     string `json:"period"`
	CallTypeID string `json:"call_type_id"`
	Query      string `json:"q"`
	StartsAt   string `json:"starts_at"`
	ID         string `json:"id"`
}

func NewAdminCallsService(pool *pgxpool.Pool) generated.AdminCallsAPIServicer {
	return newAdminCallsService(pool, time.Now)
}

func newAdminCallsService(pool *pgxpool.Pool, now func() time.Time) generated.AdminCallsAPIServicer {
	return adminCallsService{pool: pool, now: now}
}

func (s adminCallsService) ListAdminCalls(
	ctx context.Context,
	period generated.CallPeriod,
	callTypeID string,
	queryValue string,
	cursorValue string,
) (generated.ImplResponse, error) {
	queryValue = norm.NFC.String(strings.TrimSpace(queryValue))
	queryKey := norm.NFC.String(cases.Fold().String(queryValue))
	fieldErrors := make([]generated.FieldError, 0)
	if period != generated.UPCOMING && period != generated.PAST {
		fieldErrors = append(fieldErrors, fieldError("query.period", "invalid", "Период должен быть upcoming или past"))
	}
	if callTypeID != "" && !resourceIDPattern.MatchString(callTypeID) {
		fieldErrors = append(fieldErrors, fieldError("query.call_type_id", "invalid", "Должен быть UUIDv7"))
	}
	if utf8.RuneCountInString(queryValue) > 200 {
		fieldErrors = append(fieldErrors, fieldError("query.q", "invalid_length", "Поиск должен быть не длиннее 200 символов"))
	}
	if len(fieldErrors) > 0 {
		return validationResponse(fieldErrors), nil
	}

	snapshotCutoff := s.now().Truncate(time.Second)
	var pageCursor *adminCallsCursor
	if cursorValue != "" {
		decoded, err := decodeAdminCallsCursor(cursorValue)
		if err != nil || decoded.Period != string(period) || decoded.CallTypeID != callTypeID || decoded.Query != queryKey {
			return invalidAdminCallsCursorResponse(), nil
		}
		snapshotCutoff, err = parseAdminCallsSnapshot(decoded.AsOf)
		if err != nil {
			return invalidAdminCallsCursorResponse(), nil
		}
		pageCursor = &decoded
	}

	comparison := ">"
	direction := "ASC"
	if period == generated.PAST {
		comparison = "<="
		direction = "DESC"
	}
	conditions := []string{"created_at <= $1", "ends_at " + comparison + " $1"}
	args := []any{snapshotCutoff}
	if callTypeID != "" {
		args = append(args, callTypeID)
		conditions = append(conditions, fmt.Sprintf("call_type_id = $%d", len(args)))
	}
	if queryKey != "" {
		args = append(args, queryKey)
		placeholder := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, "("+
			"strpos(normalize(casefold(visitor_name COLLATE \"und-x-icu\"), NFC), "+placeholder+") > 0 OR "+
			"strpos(normalize(casefold(visitor_email COLLATE \"und-x-icu\"), NFC), "+placeholder+") > 0 OR "+
			"strpos(normalize(casefold(COALESCE(visitor_notes, '') COLLATE \"und-x-icu\"), NFC), "+placeholder+") > 0)")
	}
	if pageCursor != nil {
		lastStart, err := time.Parse(time.RFC3339, pageCursor.StartsAt)
		if err != nil || !resourceIDPattern.MatchString(pageCursor.ID) {
			return invalidAdminCallsCursorResponse(), nil
		}
		args = append(args, lastStart, pageCursor.ID)
		startPlaceholder := fmt.Sprintf("$%d", len(args)-1)
		idPlaceholder := fmt.Sprintf("$%d", len(args))
		cursorComparison := ">"
		if period == generated.PAST {
			cursorComparison = "<"
		}
		conditions = append(conditions, "(starts_at "+cursorComparison+" "+startPlaceholder+
			" OR (starts_at = "+startPlaceholder+" AND id > "+idPlaceholder+"))")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, call_type_id, call_type_name, duration_minutes, starts_at, ends_at,
		       visitor_name, visitor_email, visitor_notes
		FROM calls
		WHERE `+strings.Join(conditions, " AND ")+`
		ORDER BY starts_at `+direction+`, id ASC
		LIMIT 51
	`, args...)
	if err != nil {
		return adminCallsInternalError(ctx, "query calls", err), nil
	}
	defer rows.Close()
	items := make([]generated.AdminCall, 0)
	for rows.Next() {
		var item generated.AdminCall
		if err := rows.Scan(
			&item.Id, &item.CallTypeId, &item.CallTypeName, &item.DurationMinutes,
			&item.StartsAt, &item.EndsAt, &item.VisitorName, &item.VisitorEmail, &item.VisitorNotes,
		); err != nil {
			return adminCallsInternalError(ctx, "scan call", err), nil
		}
		item.StartsAt = item.StartsAt.In(moscowLocation)
		item.EndsAt = item.EndsAt.In(moscowLocation)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return adminCallsInternalError(ctx, "iterate calls", err), nil
	}
	var nextCursor *string
	if len(items) > 50 {
		items = items[:50]
		last := items[len(items)-1]
		encoded, err := encodeAdminCallsCursor(adminCallsCursor{
			AsOf: snapshotCutoff.Format(time.RFC3339Nano), Period: string(period), CallTypeID: callTypeID,
			Query: queryKey, StartsAt: last.StartsAt.Format(time.RFC3339), ID: last.Id,
		})
		if err != nil {
			return adminCallsInternalError(ctx, "encode cursor", err), nil
		}
		nextCursor = &encoded
	}
	return domainResponse(http.StatusOK, generated.AdminCallPage{
		AsOf:     snapshotCutoff.In(moscowLocation),
		Timezone: "Europe/Moscow", Items: items, NextCursor: nextCursor,
	}), nil
}

func parseAdminCallsSnapshot(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Nanosecond() != 0 || parsed.Format(time.RFC3339) != value {
		return time.Time{}, fmt.Errorf("snapshot is not a whole-second RFC 3339 timestamp")
	}
	_, offset := parsed.Zone()
	if offset != 3*60*60 {
		return time.Time{}, fmt.Errorf("snapshot is not in Europe/Moscow")
	}
	return parsed, nil
}

func encodeAdminCallsCursor(cursor adminCallsCursor) (string, error) {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeAdminCallsCursor(value string) (adminCallsCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return adminCallsCursor{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(decoded))
	decoder.DisallowUnknownFields()
	var cursor adminCallsCursor
	if err := decoder.Decode(&cursor); err != nil {
		return adminCallsCursor{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return adminCallsCursor{}, fmt.Errorf("cursor contains trailing data")
	}
	if cursor.AsOf == "" || cursor.Period == "" || cursor.StartsAt == "" || cursor.ID == "" {
		return adminCallsCursor{}, fmt.Errorf("cursor is incomplete")
	}
	return cursor, nil
}

func invalidAdminCallsCursorResponse() generated.ImplResponse {
	return domainResponse(http.StatusBadRequest, map[string]string{
		"code": "invalid_cursor", "message": "Курсор повреждён или не соответствует запросу",
	})
}

func adminCallsInternalError(ctx context.Context, operation string, err error) generated.ImplResponse {
	slog.ErrorContext(ctx, "admin calls operation failed", "operation", operation, "error", err)
	return domainResponse(http.StatusInternalServerError, map[string]string{
		"code": "internal_error", "message": "Не удалось загрузить звонки",
	})
}
