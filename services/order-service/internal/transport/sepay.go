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
	coreRecoveryRoutes(s, r)
	sepayStaffRoutes(s, r)
	r.Delete("/api/v1/guest-sessions/current", func(w http.ResponseWriter, r *http.Request) {
		if s.GuestSessions() == nil {
			failure(s, w, domain.E("GUEST_SESSION_UNAVAILABLE", 503))
			return
		}
		who, e := s.Authenticate(r.Header.Get("Authorization"))
		if e != nil {
			failure(s, w, e)
			return
		}
		if who.Role != "GUEST" {
			if s.C.AuthMode == "keycloak" {
				failure(s, w, domain.E("PERMISSION_DENIED", 403))
			} else {
				failure(s, w, domain.E("UNAUTHENTICATED", 401))
			}
			return
		}
		e = s.GuestSessions().RevokeGuestSession(r.Context(), who.ID, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
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
				if s.C.AuthMode == "keycloak" {
					failure(s, w, domain.E("PERMISSION_DENIED", 403))
				} else {
					failure(s, w, domain.E("UNAUTHENTICATED", 401))
				}
				return
			}
			grant, e := s.Payment.GuestAccess(r.Context(), who.ID, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), chi.URLParam(r, "order_id"), commandKey(r), method == "DELETE")
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
			scope := ""
			var e error
			bearer := r.Header.Get("Authorization")
			if app.IsGuestPaymentGrantCandidate(bearer) {
				scope, e = s.Payment.AuthorizeGuest(r.Context(), bearer, chi.URLParam(r, "order_id"))
				if e == nil {
					e = app.AuthorizeRoute(domain.Principal{Kind: domain.GuestPaymentGrant}, r.Method, r.URL.Path)
				}
				if e != nil {
					failure(s, w, domain.E("UNAUTHENTICATED", 401))
					return
				}
			} else {
				var who domain.Principal
				who, e = s.AuthenticateContext(r.Context(), bearer)
				if e != nil {
					failure(s, w, e)
					return
				}
				if who.PrincipalKind() != domain.Customer {
					if s.C.AuthMode == "keycloak" {
						failure(s, w, domain.E("PERMISSION_DENIED", 403))
					} else {
						failure(s, w, domain.E("UNAUTHENTICATED", 401))
					}
					return
				}
				who, e = s.Resolve(r.Context(), who)
				if e != nil {
					failure(s, w, e)
					return
				}
				scope = who.Scope()
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
