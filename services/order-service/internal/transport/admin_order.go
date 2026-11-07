package transport

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
)

func adminOrderProjection(s *app.Service, queue, detail bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		actor, err := app.NewAdminOrderReader(principal(r))
		if err != nil {
			failure(s, w, err)
			return
		}
		query := r.URL.Query()
		for key, values := range query {
			if detail || (key != "status" && key != "cursor" && key != "limit") || len(values) != 1 {
				failure(s, w, domain.E("INVALID_QUERY", 400))
				return
			}
		}
		var result any
		if detail {
			result, err = s.AdminOrderDetail(r.Context(), actor, chi.URLParam(r, "id"))
		} else {
			limit := 25
			if query.Has("limit") {
				limit, err = strconv.Atoi(query.Get("limit"))
				if err != nil || limit < 1 || limit > 100 {
					failure(s, w, domain.E("INVALID_LIMIT", 400))
					return
				}
			}
			if len(query.Get("status")) > 64 || len(query.Get("cursor")) > 2048 {
				failure(s, w, domain.E("INVALID_QUERY", 400))
				return
			}
			result, err = s.AdminOrderPage(r.Context(), actor, query.Get("status"), query.Get("cursor"), limit, queue)
		}
		if err != nil {
			failure(s, w, err)
			return
		}
		write(w, 200, result)
	}
}

// The Admin UI opts into its read projection without replacing the existing
// order response used by direct API consumers and integration callers.
func adminOrderList(s *app.Service, queue bool, fallback http.HandlerFunc) http.HandlerFunc {
	projection := adminOrderProjection(s, queue, false)
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), app.AdminOrderMediaType) {
			projection(w, r)
			return
		}
		fallback(w, r)
	}
}
