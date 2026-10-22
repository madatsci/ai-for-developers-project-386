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

func NewHandler(
	ping func(context.Context) error,
	publicCallTypes generated.PublicCallTypesAPIServicer,
	publicCalls generated.PublicCallsAPIServicer,
) http.Handler {
	healthController := generated.NewHealthAPIController(healthService{ping: ping})
	publicController := generated.NewPublicCallTypesAPIController(publicCallTypes)
	publicCallsController := generated.NewPublicCallsAPIController(
		publicCalls,
		generated.WithPublicCallsAPIErrorHandler(publicCallErrorHandler),
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
	}
	return exactJSONContentType(createCallRequestGuard(generated.NewRouter(routers...)))
}

func createCallRequestGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/public/calls" {
			next.ServeHTTP(w, r)
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeDomainError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Поддерживается только application/json")
			return
		}
		if query := r.URL.Query(); len(query) > 0 {
			keys := make([]string, 0, len(query))
			for key := range query {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			writeValidationError(w, []generated.FieldError{
				fieldError("query."+keys[0], "unknown", "Неизвестный параметр запроса"),
			})
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes+1))
		if err != nil || len(body) > maxJSONBodyBytes || !json.Valid(body) {
			writeDomainError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
			return
		}
		var rawFields map[string]json.RawMessage
		if err := json.Unmarshal(body, &rawFields); err == nil {
			if rawStartsAt, exists := rawFields["starts_at"]; exists {
				var startsAt string
				if err := json.Unmarshal(rawStartsAt, &startsAt); err == nil && !requestSlotDateTimePattern.MatchString(startsAt) {
					writeValidationError(w, []generated.FieldError{
						fieldError("body.starts_at", "invalid", "Укажите RFC 3339 время на пятиминутной сетке без долей секунды"),
					})
					return
				}
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		next.ServeHTTP(w, r)
	})
}

func publicCallErrorHandler(w http.ResponseWriter, _ *http.Request, err error, result *generated.ImplResponse) {
	var requiredError *generated.RequiredError
	if errors.As(err, &requiredError) {
		writeValidationError(w, []generated.FieldError{
			fieldError("body."+requiredError.Field, "required", "Обязательное поле отсутствует"),
		})
		return
	}
	var parsingError *generated.ParsingError
	if errors.As(err, &parsingError) {
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
