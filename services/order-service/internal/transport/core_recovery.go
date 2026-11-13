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
		if len(r.Header.Values("Authorization")) != 1 {
			failure(s, w, domain.E("UNAUTHENTICATED", 401))
			return
		}
		p, e := s.AuthenticateContext(r.Context(), r.Header.Get("Authorization"))
		if e != nil {
			failure(s, w, e)
			return
		}
		if !p.Manager() {
			failure(s, w, domain.E("PERMISSION_DENIED", 403))
			return
		}
		scope := domain.PermissionScope{Global: p.Role == "ADMIN", IDs: []string{p.Warehouse}}
		if p.IAM {
			scope = p.PermissionScope("ORDER_RECOVERY_EXECUTE")
		}
		if !scope.Global && (len(scope.IDs) == 0 || scope.IDs[0] == "") {
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
		e = s.CoreCheckout.RecoverAuthorized(r.Context(), chi.URLParam(r, "id"), p.Scope(), scope, os.Getenv("ORDER_MERCHANT_ID"), commandKey(r), in.Reason)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 202, map[string]string{"operation_id": chi.URLParam(r, "id"), "status": "RETRY_SCHEDULED"})
	})
}
