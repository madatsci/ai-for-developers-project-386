//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
	"github.com/madatsci/ai-for-developers-project-386/backend/internal/storage"
)

func TestAdminCallsClassifiesAndOrdersSnapshot(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Production rejects overlapping calls. This isolated schema drops the constraint only to
	// exercise the API's defensive UUID tie-breaker required for equal start timestamps.
	if _, err := pool.Exec(ctx, `ALTER TABLE calls DROP CONSTRAINT calls_tstzrange_excl`); err != nil {
		t.Fatalf("drop overlap constraint in ordering fixture: %v", err)
	}
	var callTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM call_types WHERE slug = 'zvonok-na-30-minut'`).Scan(&callTypeID); err != nil {
		t.Fatalf("read call type: %v", err)
	}
	for _, call := range []struct {
		id, startsAt, name string
	}{
		{"018f0000-0000-7000-8000-000000000503", "2026-10-20T11:45:00+03:00", "Текущий"},
		{"018f0000-0000-7000-8000-000000000502", "2026-10-20T13:00:00+03:00", "Второй при равном начале"},
		{"018f0000-0000-7000-8000-000000000501", "2026-10-20T13:00:00+03:00", "Первый при равном начале"},
		{"018f0000-0000-7000-8000-000000000504", "2026-10-20T10:00:00+03:00", "Прошедший"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO calls (
				id, call_type_id, call_type_name, duration_minutes, starts_at,
				visitor_name, visitor_email, created_at
			)
			VALUES ($1, $2, 'Историческое название', 30, $3, $4, 'visitor@example.com', '2026-10-20T12:00:00+03:00')
		`, call.id, callTypeID, call.startsAt, call.name); err != nil {
			t.Fatalf("insert call %s: %v", call.id, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO calls (
			id, call_type_id, call_type_name, duration_minutes, starts_at,
			visitor_name, visitor_email, created_at
		) VALUES (
			'018f0000-0000-7000-8000-000000000505', $1, 'Историческое название', 30,
			'2026-10-20T14:00:00+03:00', 'Создан после публичного снимка',
			'visitor@example.com', '2026-10-20T12:00:00.250+03:00'
		)
	`, callTypeID); err != nil {
		t.Fatalf("insert call after public snapshot: %v", err)
	}

	moscow := time.FixedZone("Europe/Moscow", 3*60*60)
	now := time.Date(2026, time.October, 20, 12, 0, 0, 500_000_000, moscow)
	handler := newAdminCallsIntegrationHandler(pool, newAdminCallsService(pool, func() time.Time { return now }))

	upcoming := httptest.NewRecorder()
	handler.ServeHTTP(upcoming, httptest.NewRequest(http.MethodGet, "/api/v1/admin/calls?period=upcoming", nil))
	if upcoming.Code != http.StatusOK {
		t.Fatalf("upcoming status = %d, want 200; body=%s", upcoming.Code, upcoming.Body.String())
	}
	var page generated.AdminCallPage
	if err := json.Unmarshal(upcoming.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode upcoming: %v", err)
	}
	if !page.AsOf.Equal(now.Truncate(time.Second)) || page.Timezone != "Europe/Moscow" || page.NextCursor != nil {
		t.Errorf("snapshot metadata = %+v", page)
	}
	if !bytes.Contains(upcoming.Body.Bytes(), []byte(`"as_of":"2026-10-20T12:00:00+03:00"`)) {
		t.Errorf("as_of must not contain fractional seconds: %s", upcoming.Body.String())
	}
	wantUpcoming := []string{
		"018f0000-0000-7000-8000-000000000503",
		"018f0000-0000-7000-8000-000000000501",
		"018f0000-0000-7000-8000-000000000502",
	}
	if len(page.Items) != len(wantUpcoming) {
		t.Fatalf("upcoming items = %d, want %d: %+v", len(page.Items), len(wantUpcoming), page.Items)
	}
	for index, wantID := range wantUpcoming {
		if page.Items[index].Id != wantID {
			t.Errorf("upcoming item %d = %s, want %s", index, page.Items[index].Id, wantID)
		}
	}
	if page.Items[0].CallTypeName != "Историческое название" || page.Items[0].DurationMinutes != 30 {
		t.Errorf("call snapshot = %+v", page.Items[0])
	}

	past := httptest.NewRecorder()
	handler.ServeHTTP(past, httptest.NewRequest(http.MethodGet, "/api/v1/admin/calls?period=past", nil))
	if past.Code != http.StatusOK {
		t.Fatalf("past status = %d, want 200; body=%s", past.Code, past.Body.String())
	}
	if err := json.Unmarshal(past.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode past: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Id != "018f0000-0000-7000-8000-000000000504" {
		t.Errorf("past items = %+v", page.Items)
	}
}

func TestAdminCallsFiltersAndPaginatesStableSnapshot(t *testing.T) {
	pool := catalogTestPool(t)
	ctx := context.Background()
	if err := storage.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := storage.Seed(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}
	callTypeIDs := map[int]string{}
	rows, err := pool.Query(ctx, `SELECT duration_minutes, id FROM call_types`)
	if err != nil {
		t.Fatalf("list call types: %v", err)
	}
	for rows.Next() {
		var duration int
		var id string
		if err := rows.Scan(&duration, &id); err != nil {
			t.Fatalf("scan call type: %v", err)
		}
		callTypeIDs[duration] = id
	}
	rows.Close()

	moscow := time.FixedZone("Europe/Moscow", 3*60*60)
	asOf := time.Date(2026, time.October, 20, 12, 0, 0, 500_000_000, moscow)
	for index := 0; index < 55; index++ {
		id := fmt.Sprintf("018f0000-0000-7000-8000-%012x", 0x600+index)
		startsAt := asOf.Truncate(time.Second).Add(time.Hour + time.Duration(index)*20*time.Minute)
		name := fmt.Sprintf("Посетитель %02d", index)
		email := fmt.Sprintf("visitor%02d@example.com", index)
		notes := "Обычный комментарий"
		if index == 52 {
			name = "Анна Смирнова"
			notes = "Обсудить API"
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO calls (
				id, call_type_id, call_type_name, duration_minutes, starts_at,
				visitor_name, visitor_email, visitor_notes, created_at
			) VALUES ($1, $2, 'Снимок типа', 15, $3, $4, $5, $6, $7)
		`, id, callTypeIDs[15], startsAt, name, email, notes, asOf.Add(-time.Hour)); err != nil {
			t.Fatalf("insert call %d: %v", index, err)
		}
	}
	handler := newAdminCallsIntegrationHandler(pool, newAdminCallsService(pool, func() time.Time { return asOf }))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/admin/calls", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first page status = %d, want 200; body=%s", first.Code, first.Body.String())
	}
	var firstPage generated.AdminCallPage
	if err := json.Unmarshal(first.Body.Bytes(), &firstPage); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if len(firstPage.Items) != 50 || firstPage.NextCursor == nil {
		t.Fatalf("first page = %d items cursor %v, want 50 and cursor", len(firstPage.Items), firstPage.NextCursor)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO calls (
			id, call_type_id, call_type_name, duration_minutes, starts_at,
			visitor_name, visitor_email, created_at
		) VALUES (
			'018f0000-0000-7000-8000-000000000699', $1, 'Поздний тип', 15,
			$2, 'Поздний посетитель', 'late@example.com', $3
		)
	`, callTypeIDs[15], asOf.Truncate(time.Second).Add(20*time.Hour), asOf.Add(100*time.Millisecond)); err != nil {
		t.Fatalf("insert after snapshot: %v", err)
	}

	second := httptest.NewRecorder()
	secondURL := "/api/v1/admin/calls?cursor=" + url.QueryEscape(*firstPage.NextCursor)
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, secondURL, nil))
	if second.Code != http.StatusOK {
		t.Fatalf("second page status = %d, want 200; body=%s", second.Code, second.Body.String())
	}
	var secondPage generated.AdminCallPage
	if err := json.Unmarshal(second.Body.Bytes(), &secondPage); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if !secondPage.AsOf.Equal(firstPage.AsOf) || len(secondPage.Items) != 5 || secondPage.NextCursor != nil {
		t.Errorf("second page = as_of %s items %d cursor %v", secondPage.AsOf, len(secondPage.Items), secondPage.NextCursor)
	}
	seen := map[string]bool{}
	for _, item := range append(firstPage.Items, secondPage.Items...) {
		if seen[item.Id] {
			t.Errorf("duplicate call %s", item.Id)
		}
		seen[item.Id] = true
	}
	if len(seen) != 55 {
		t.Errorf("snapshot call count = %d, want 55", len(seen))
	}

	filtered := httptest.NewRecorder()
	filterURL := "/api/v1/admin/calls?call_type_id=" + callTypeIDs[15] + "&q=" + url.QueryEscape("  АННА  ")
	handler.ServeHTTP(filtered, httptest.NewRequest(http.MethodGet, filterURL, nil))
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered status = %d, want 200; body=%s", filtered.Code, filtered.Body.String())
	}
	var filteredPage generated.AdminCallPage
	if err := json.Unmarshal(filtered.Body.Bytes(), &filteredPage); err != nil {
		t.Fatalf("decode filtered page: %v", err)
	}
	if len(filteredPage.Items) != 1 || filteredPage.Items[0].VisitorName != "Анна Смирнова" {
		t.Errorf("filtered items = %+v", filteredPage.Items)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE calls
		SET visitor_name = 'Straße'
		WHERE id = '018f0000-0000-7000-8000-000000000600'
	`); err != nil {
		t.Fatalf("update Unicode search fixture: %v", err)
	}
	unicodeSearch := httptest.NewRecorder()
	handler.ServeHTTP(unicodeSearch, httptest.NewRequest(http.MethodGet, "/api/v1/admin/calls?q=STRASSE", nil))
	if unicodeSearch.Code != http.StatusOK {
		t.Fatalf("Unicode search status = %d, want 200; body=%s", unicodeSearch.Code, unicodeSearch.Body.String())
	}
	var unicodePage generated.AdminCallPage
	if err := json.Unmarshal(unicodeSearch.Body.Bytes(), &unicodePage); err != nil {
		t.Fatalf("decode Unicode search: %v", err)
	}
	if len(unicodePage.Items) != 1 || unicodePage.Items[0].VisitorName != "Straße" {
		t.Errorf("Unicode search items = %+v", unicodePage.Items)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE calls
		SET visitor_name = 'ẖ'
		WHERE id = '018f0000-0000-7000-8000-000000000601'
	`); err != nil {
		t.Fatalf("update Unicode normalization fixture: %v", err)
	}
	normalizedSearch := httptest.NewRecorder()
	handler.ServeHTTP(normalizedSearch, httptest.NewRequest(http.MethodGet, "/api/v1/admin/calls?q="+url.QueryEscape("ẖ"), nil))
	if normalizedSearch.Code != http.StatusOK {
		t.Fatalf("normalized search status = %d, want 200; body=%s", normalizedSearch.Code, normalizedSearch.Body.String())
	}
	var normalizedPage generated.AdminCallPage
	if err := json.Unmarshal(normalizedSearch.Body.Bytes(), &normalizedPage); err != nil {
		t.Fatalf("decode normalized search: %v", err)
	}
	if len(normalizedPage.Items) != 1 || normalizedPage.Items[0].VisitorName != "ẖ" {
		t.Errorf("normalized search items = %+v", normalizedPage.Items)
	}

	fractionalCursor, err := encodeAdminCallsCursor(adminCallsCursor{
		AsOf: "2026-10-20T12:00:00.5+03:00", Period: "upcoming",
		StartsAt: "2026-10-20T13:00:00+03:00", ID: "018f0000-0000-7000-8000-000000000600",
	})
	if err != nil {
		t.Fatalf("encode fractional cursor: %v", err)
	}

	for name, requestURL := range map[string]string{
		"changed filters":     secondURL + "&q=other",
		"changed period":      secondURL + "&period=past",
		"corrupted":           "/api/v1/admin/calls?cursor=not-a-cursor",
		"fractional snapshot": "/api/v1/admin/calls?cursor=" + url.QueryEscape(fractionalCursor),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestURL, nil))
			if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"invalid_cursor"`)) {
				t.Errorf("status = %d body %s, want invalid_cursor", response.Code, response.Body.String())
			}
		})
	}

	unknownType := httptest.NewRecorder()
	handler.ServeHTTP(unknownType, httptest.NewRequest(http.MethodGet, "/api/v1/admin/calls?call_type_id=018f0000-0000-7000-8000-000000009999", nil))
	if unknownType.Code != http.StatusOK || !bytes.Contains(unknownType.Body.Bytes(), []byte(`"items":[]`)) {
		t.Errorf("unknown type = %d %s, want empty 200", unknownType.Code, unknownType.Body.String())
	}

	for name, requestURL := range map[string]string{
		"invalid call type": "/api/v1/admin/calls?call_type_id=wrong",
		"empty call type":   "/api/v1/admin/calls?call_type_id=",
		"empty cursor":      "/api/v1/admin/calls?cursor=",
		"long search":       "/api/v1/admin/calls?q=" + strings.Repeat("я", 201),
		"unknown query":     "/api/v1/admin/calls?unexpected=1",
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestURL, nil))
			if response.Code != http.StatusUnprocessableEntity || !bytes.Contains(response.Body.Bytes(), []byte(`"code":"validation_error"`)) {
				t.Errorf("status = %d body %s, want validation_error", response.Code, response.Body.String())
			}
		})
	}
}

func newAdminCallsIntegrationHandler(pool interface {
	Ping(context.Context) error
}, adminCalls generated.AdminCallsAPIServicer) http.Handler {
	return NewHandler(pool.Ping, HandlerServices{
		PublicCallTypes:   publicCallTypesStub{},
		PublicCalls:       publicCallsStub{},
		AdminCallTypes:    adminCallTypesStub{},
		AdminCalls:        adminCalls,
		AdminSchedules:    adminSchedulesStub{},
		AdminWorkingHours: adminWorkingHoursStub{},
	})
}
