package transport

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
)

func adminOrderReadRoutes(s *app.Service, r chi.Router) {
	read := func(action func(*http.Request, app.AdminOrderReader) (any, error)) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			actor, err := s.AuthenticateAdminOrderReader(req.Context(), req.Header.Get("Authorization"))
			if err != nil {
				failure(s, w, err)
				return
			}
			result, err := action(req, actor)
			if err != nil {
				failure(s, w, err)
				return
			}
			write(w, 200, result)
		}
	}
	list := func(queue bool) http.HandlerFunc {
		return read(func(req *http.Request, actor app.AdminOrderReader) (any, error) {
			query := req.URL.Query()
			for name, values := range query {
				if (name != "status" && name != "cursor" && name != "limit") || len(values) != 1 {
					return nil, domain.E("INVALID_QUERY", 400)
				}
			}
			limit := 25
			if query.Has("limit") {
				var err error
				limit, err = strconv.Atoi(query.Get("limit"))
				if err != nil || limit < 1 || limit > 100 {
					return nil, domain.E("INVALID_LIMIT", 400)
				}
			}
			status, cursor := query.Get("status"), query.Get("cursor")
			if len(status) > 64 || len(cursor) > 2048 {
				return nil, domain.E("INVALID_QUERY", 400)
			}
			return s.AdminOrderPage(req.Context(), actor, status, cursor, limit, queue)
		})
	}
	r.Get("/api/v1/admin/orders", list(false))
	r.Get("/api/v1/admin/orders/queue", list(true))
	r.Get("/api/v1/admin/orders/{id}", read(func(req *http.Request, actor app.AdminOrderReader) (any, error) {
		return s.AdminOrderDetail(req.Context(), actor, chi.URLParam(req, "id"))
	}))
}
