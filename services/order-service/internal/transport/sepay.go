package transport

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"io"
	"net/http"
	"strings"
)

func sepayError(s *app.Service, w http.ResponseWriter, err error) {
	var d *domain.Error
	if errors.As(err, &d) {
		failure(s, w, err)
		return
	}
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		failure(s, w, domain.E("NOT_FOUND", 404))
	case errors.Is(err, persistence.ErrInput), errors.Is(err, sepay.ErrInvalid):
		failure(s, w, domain.E("INVALID_PAYMENT_REQUEST", 400))
	case errors.Is(err, persistence.ErrExpired):
		failure(s, w, domain.E("PAYMENT_INSTRUCTIONS_UNAVAILABLE", 409))
	case errors.Is(err, persistence.ErrConflict), errors.Is(err, persistence.ErrReplayExpired):
		failure(s, w, domain.E("GUEST_GRANT_CONFLICT", 409))
	case errors.Is(err, persistence.ErrScope):
		failure(s, w, domain.E("UNAUTHENTICATED", 401))
	case errors.Is(err, persistence.ErrForbidden):
		failure(s, w, domain.E("PERMISSION_DENIED", 403))
	default:
		failure(s, w, domain.E("PAYMENT_UNAVAILABLE", 503))
	}
}
func sepayRoutes(s *app.Service, r chi.Router) {
	sepayStaffRoutes(s, r)
	r.Delete("/api/v1/guest-sessions/current", func(w http.ResponseWriter, r *http.Request) {
		if s.Payment == nil {
			failure(s, w, domain.E("GUEST_SESSION_UNAVAILABLE", 503))
			return
		}
		who, e := s.Authenticate(r.Header.Get("Authorization"))
		if e != nil {
			failure(s, w, e)
			return
		}
		if who.Role != "GUEST" {
			failure(s, w, domain.E("UNAUTHENTICATED", 401))
			return
		}
		e = s.Payment.RevokeGuestSession(r.Context(), who.ID, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		w.WriteHeader(204)
	})
	for _, method := range []string{"POST", "DELETE"} {
		method := method
		r.MethodFunc(method, "/api/v1/orders/{order_id}/guest-payment-access", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if s.Payment == nil {
				failure(s, w, domain.E("GUEST_SESSION_UNAVAILABLE", 503))
				return
			}
			who, e := s.Authenticate(r.Header.Get("Authorization"))
			if e != nil {
				failure(s, w, e)
				return
			}
			if who.Role != "GUEST" {
				failure(s, w, domain.E("UNAUTHENTICATED", 401))
				return
			}
			grant, e := s.Payment.GuestAccess(r.Context(), who.ID, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), chi.URLParam(r, "order_id"), r.Header.Get("X-Idempotency-Key"), method == "DELETE")
			if e != nil {
				sepayError(s, w, e)
				return
			}
			if method == "DELETE" {
				w.WriteHeader(204)
				return
			}
			write(w, 201, grant)
		})
	}
	r.Post("/api/v1/payments/sepay/webhook", func(w http.ResponseWriter, r *http.Request) {
		if s.Payment == nil {
			write(w, 503, map[string]bool{"success": false})
			return
		}
		if strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
			write(w, 415, map[string]bool{"success": false})
			return
		}
		raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
		if e != nil {
			var large *http.MaxBytesError
			if errors.As(e, &large) {
				write(w, 413, map[string]bool{"success": false})
			} else {
				write(w, 400, map[string]bool{"success": false})
			}
			return
		}
		e = s.Payment.Observe(r.Context(), raw, r.Header.Get("X-SePay-Timestamp"), r.Header.Get("X-SePay-Signature"))
		if e != nil {
			status := 503
			if errors.Is(e, sepay.ErrAuth) {
				status = 401
			} else if errors.Is(e, sepay.ErrInvalid) || errors.Is(e, persistence.ErrInput) {
				status = 400
			}
			write(w, status, map[string]bool{"success": false})
			return
		}
		write(w, 200, map[string]bool{"success": true})
	})
	for _, instructions := range []bool{false, true} {
		instructions := instructions
		path := "/api/v1/orders/{order_id}/payment"
		if instructions {
			path = "/api/v1/orders/{order_id}/payment-instructions"
		}
		r.Get(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			if s.Payment == nil {
				failure(s, w, domain.E("PAYMENT_UNAVAILABLE", 503))
				return
			}
			who, e := s.Authenticate(r.Header.Get("Authorization"))
			scope := ""
			if e == nil && who.Role == "CUSTOMER" {
				who, e = s.Resolve(r.Context(), who)
				if e != nil {
					failure(s, w, e)
					return
				}
				scope = who.Scope()
			} else {
				scope, e = s.Payment.AuthorizeGuest(r.Context(), r.Header.Get("Authorization"), chi.URLParam(r, "order_id"))
				if e != nil {
					failure(s, w, domain.E("UNAUTHENTICATED", 401))
					return
				}
			}
			if instructions {
				data, e := s.Payment.Instructions(r.Context(), chi.URLParam(r, "order_id"), scope)
				if e != nil {
					sepayError(s, w, e)
					return
				}
				write(w, 200, data)
			} else {
				data, e := s.Payment.Status(r.Context(), chi.URLParam(r, "order_id"), scope)
				if e != nil {
					sepayError(s, w, e)
					return
				}
				write(w, 200, data)
			}
		})
	}
}
