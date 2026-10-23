package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

const maxJSONBodyBytes = 64 * 1024

var requestSlotDateTimePattern = regexp.MustCompile(`^[0-9]{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12][0-9]|3[01])T(?:[01][0-9]|2[0-3]):[0-5][05]:00(?:Z|[+-](?:[01][0-9]|2[0-3]):[0-5][0-9])$`)

type HandlerServices struct {
	PublicCallTypes   generated.PublicCallTypesAPIServicer
	PublicCalls       generated.PublicCallsAPIServicer
	AdminCallTypes    generated.AdminCallTypesAPIServicer
	AdminCalls        generated.AdminCallsAPIServicer
	AdminSchedules    generated.AdminSchedulesAPIServicer
	AdminWorkingHours generated.AdminWorkingHoursAPIServicer
}

func NewHandler(ping func(context.Context) error, services HandlerServices) http.Handler {
	healthController := generated.NewHealthAPIController(healthService{ping: ping})
	publicController := generated.NewPublicCallTypesAPIController(services.PublicCallTypes)
	publicCallsController := generated.NewPublicCallsAPIController(
		services.PublicCalls,
		generated.WithPublicCallsAPIErrorHandler(controllerErrorHandler),
	)
	adminSchedulesController := generated.NewAdminSchedulesAPIController(
		services.AdminSchedules,
		generated.WithAdminSchedulesAPIErrorHandler(controllerErrorHandler),
	)
	adminCallTypesController := generated.NewAdminCallTypesAPIController(
		services.AdminCallTypes,
		generated.WithAdminCallTypesAPIErrorHandler(controllerErrorHandler),
	)
	adminCallsController := generated.NewAdminCallsAPIController(
		services.AdminCalls,
		generated.WithAdminCallsAPIErrorHandler(controllerErrorHandler),
	)
	adminWorkingHoursController := generated.NewAdminWorkingHoursAPIController(
		services.AdminWorkingHours,
		generated.WithAdminWorkingHoursAPIErrorHandler(controllerErrorHandler),
	)
	publicRoutes := publicController.Routes()
	routers := []generated.Router{
		healthController,
		routeSubset{routes: []generated.Route{
			publicRoutes["ListPublicCallTypes"],
			publicRoutes["GetPublicCallType"],
			publicRoutes["GetPublicCallTypeAvailability"],
		}},
		publicCallsController,
		adminCallTypesController,
		adminCallsController,
		adminSchedulesController,
		adminWorkingHoursController,
	}
	contracts := map[string]requestContract{
		"/api/v1/public/call-types": {},
		"/api/v1/public/calls":      {jsonMethod: http.MethodPost, validateJSON: validateCreateCallJSON},
		"/api/v1/admin/call-types":  {jsonMethod: http.MethodPost},
		"/api/v1/admin/calls": {
			allowedQuery:  map[string]bool{"period": true, "call_type_id": true, "q": true, "cursor": true},
			nonEmptyQuery: map[string]bool{"call_type_id": true, "cursor": true},
		},
		"/api/v1/admin/schedules":     {jsonMethod: http.MethodPost},
		"/api/v1/admin/working-hours": {jsonMethod: http.MethodPut},
	}
	return exactJSONContentType(productRequestGuard(contracts, generated.NewRouter(routers...)))
}

type requestContract struct {
	jsonMethod    string
	validateJSON  func(http.ResponseWriter, []byte) bool
	allowedQuery  map[string]bool
	nonEmptyQuery map[string]bool
}

func productRequestGuard(contracts map[string]requestContract, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contract, guarded := contracts[r.URL.Path]
		if !guarded && strings.HasPrefix(r.URL.Path, "/api/v1/admin/schedules/") {
			contract = requestContract{jsonMethod: http.MethodPut}
			guarded = true
		}
		if !guarded && strings.HasPrefix(r.URL.Path, "/api/v1/admin/call-types/") {
			contract = requestContract{jsonMethod: http.MethodPut}
			guarded = true
		}
		if !guarded && strings.HasPrefix(r.URL.Path, "/api/v1/public/call-types/") {
			guarded = true
		}
		if !guarded {
			next.ServeHTTP(w, r)
			return
		}
		if query := r.URL.Query(); len(query) > 0 {
			keys := make([]string, 0, len(query))
			for key := range query {
				if !contract.allowedQuery[key] {
					keys = append(keys, key)
				}
			}
			if len(keys) > 0 {
				sort.Strings(keys)
				writeValidationError(w, []generated.FieldError{
					fieldError("query."+keys[0], "unknown", "Неизвестный параметр запроса"),
				})
				return
			}
			for key := range contract.nonEmptyQuery {
				if query.Has(key) && query.Get(key) == "" {
					writeValidationError(w, []generated.FieldError{
						fieldError("query."+key, "required", "Параметр не должен быть пустым"),
					})
					return
				}
			}
		}
		if r.Method != contract.jsonMethod {
			next.ServeHTTP(w, r)
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeDomainError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Поддерживается только application/json")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes+1))
		if err != nil || len(body) > maxJSONBodyBytes || !json.Valid(body) {
			writeDomainError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
			return
		}
		if contract.validateJSON != nil && !contract.validateJSON(w, body) {
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		next.ServeHTTP(w, r)
	})
}

func validateCreateCallJSON(w http.ResponseWriter, body []byte) bool {
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawFields); err != nil {
		return true
	}
	rawStartsAt, exists := rawFields["starts_at"]
	if !exists {
		return true
	}
	var startsAt string
	if err := json.Unmarshal(rawStartsAt, &startsAt); err != nil || requestSlotDateTimePattern.MatchString(startsAt) {
		return true
	}
	writeValidationError(w, []generated.FieldError{
		fieldError("body.starts_at", "invalid", "Укажите RFC 3339 время на пятиминутной сетке без долей секунды"),
	})
	return false
}

func controllerErrorHandler(w http.ResponseWriter, _ *http.Request, err error, result *generated.ImplResponse) {
	var requiredError *generated.RequiredError
	if errors.As(err, &requiredError) {
		writeValidationError(w, []generated.FieldError{
			fieldError("body."+requiredError.Field, "required", "Обязательное поле отсутствует"),
		})
		return
	}
	var parsingError *generated.ParsingError
	if errors.As(err, &parsingError) {
		if parsingError.Param == "DayOfWeek" {
			writeValidationError(w, []generated.FieldError{
				fieldError("body.days.day_of_week", "invalid", "День недели должен быть от 1 до 7"),
			})
			return
		}
		if parsingError.Param == "DurationMinutes" {
			writeValidationError(w, []generated.FieldError{
				fieldError("body.duration_minutes", "invalid", "Длительность должна быть от 15 до 240 минут с шагом 5"),
			})
			return
		}
		writeDomainError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}
	if result == nil {
		writeDomainError(w, http.StatusInternalServerError, "internal_error", "Не удалось обработать запрос")
		return
	}
	_ = generated.EncodeJSONResponse(result.Body, &result.Code, result.Headers, w)
}

func writeValidationError(w http.ResponseWriter, fieldErrors []generated.FieldError) {
	response := validationResponse(fieldErrors)
	_ = generated.EncodeJSONResponse(response.Body, &response.Code, response.Headers, w)
}

func writeDomainError(w http.ResponseWriter, status int, code, message string) {
	response := domainResponse(status, map[string]string{"code": code, "message": message})
	_ = generated.EncodeJSONResponse(response.Body, &response.Code, response.Headers, w)
}

func exactJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&exactJSONContentTypeWriter{ResponseWriter: w}, r)
	})
}

type exactJSONContentTypeWriter struct {
	http.ResponseWriter
}

func (w *exactJSONContentTypeWriter) WriteHeader(code int) {
	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
	}
	w.ResponseWriter.WriteHeader(code)
}
