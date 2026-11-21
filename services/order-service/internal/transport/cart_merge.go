package transport

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
)

func cartMergeRoute(s *app.Service, api chi.Router) {
	// POST /cart is already allowed by Identity's customer proxy. The source
	// credential is an explicit body field, never substituted for Customer auth.
	api.Post("/api/v1/cart", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var in struct {
			GuestToken string `json:"guest_token"`
		}
		if e := decode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		if len(in.GuestToken) == 0 || len(in.GuestToken) > 8192 {
			failure(s, w, domain.E("INVALID_GUEST_CREDENTIAL", 400))
			return
		}
		result, e := s.MergeGuestCart(r.Context(), principal(r), in.GuestToken, commandKey(r))
		if e != nil {
			switch {
			case errors.Is(e, persistence.ErrMergeConflict):
				e = domain.E("CART_MERGE_CONFLICT", 409)
			case errors.Is(e, persistence.ErrScope):
				e = domain.E("GUEST_SESSION_INVALID", 401)
			case errors.Is(e, persistence.ErrInput):
				e = domain.E("INVALID_IDEMPOTENCY_KEY", 400)
			default:
				var known *domain.Error
				if !errors.As(e, &known) {
					e = domain.E("CART_MERGE_UNAVAILABLE", 503)
				}
			}
			failure(s, w, e)
			return
		}
		write(w, 200, result)
	})
}
