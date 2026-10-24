package transport

import (
	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"net/http"
	"os"
)

func coreRecoveryRoutes(s *app.Service, r chi.Router) {
	r.Post("/api/v1/admin/checkout-operations/{id}/recover", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if s.CoreCheckout == nil {
			failure(s, w, domain.E("TARGET_RUNTIME_UNAVAILABLE", 503))
			return
		}
		p, e := s.Authenticate(r.Header.Get("Authorization"))
		if e != nil {
			failure(s, w, e)
			return
		}
		if !p.Manager() {
			failure(s, w, domain.E("PERMISSION_DENIED", 403))
			return
		}
		warehouse := p.Warehouse
		if p.Role == "ADMIN" {
			warehouse = ""
		} else if warehouse == "" {
			failure(s, w, domain.E("WAREHOUSE_SCOPE_REQUIRED", 403))
			return
		}
		var in struct {
			Reason string `json:"reason"`
		}
		if e := decode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		e = s.CoreCheckout.Recover(r.Context(), chi.URLParam(r, "id"), p.Scope(), warehouse, os.Getenv("ORDER_MERCHANT_ID"), r.Header.Get("X-Idempotency-Key"), in.Reason)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 202, map[string]string{"operation_id": chi.URLParam(r, "id"), "status": "RETRY_SCHEDULED"})
	})
}
