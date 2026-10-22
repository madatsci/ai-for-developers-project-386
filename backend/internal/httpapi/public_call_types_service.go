package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madatsci/ai-for-developers-project-386/backend/internal/httpapi/generated"
)

type publicCallType struct {
	slug            string
	name            string
	description     *string
	durationMinutes int32
}

type publicCallTypeRepository interface {
	listActive(context.Context) ([]publicCallType, error)
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
}

func NewPublicCallTypesService(pool *pgxpool.Pool) generated.PublicCallTypesAPIServicer {
	return publicCallTypesService{repository: postgresPublicCallTypeRepository{pool: pool}}
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

func (publicCallTypesService) GetPublicCallType(context.Context, string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotFound, nil), nil
}

func (publicCallTypesService) GetPublicCallTypeAvailability(context.Context, string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotFound, nil), nil
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
