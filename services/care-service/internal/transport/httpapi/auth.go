package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type actorContextKey struct{}

type Actor struct {
	ID            uuid.UUID
	PrincipalType string
	Permissions   map[string]struct{}
}

func AuthenticatedActor(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	return actor, ok && actor.ID != uuid.Nil
}

func TrustedGatewayAuth(internalToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			supplied := r.Header.Get("X-Internal-Token")
			if internalToken == "" || subtle.ConstantTimeCompare([]byte(supplied), []byte(internalToken)) != 1 {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
				return
			}
			actorID, err := uuid.Parse(r.Header.Get("X-User-ID"))
			if err != nil || actorID == uuid.Nil {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "A valid trusted user identity is required", nil)
				return
			}
			permissions := make(map[string]struct{})
			for _, permission := range strings.Split(r.Header.Get("X-Permissions"), ",") {
				if normalized := canonicalCarePermission(permission); normalized != "" {
					permissions[normalized] = struct{}{}
				}
			}
			actor := Actor{ID: actorID, Permissions: permissions}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorContextKey{}, actor)))
		})
	}
}

// GatewayOrStaffJWTAuth preserves the existing trusted-gateway path while allowing
// employee routes to validate Identity-issued JWTs directly. A presented Bearer
// token is never downgraded to the forwarded-header path when it is invalid.
func GatewayOrStaffJWTAuth(internalToken string, staff StaffAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.Header.Values("Authorization")) > 0 {
				if staff == nil || len(r.Header.Values("Authorization")) != 1 {
					writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
					return
				}
				actor, err := staff.Authenticate(r.Context(), r.Header.Get("Authorization"))
				if err != nil {
					status, code, message := http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required"
					if errors.Is(err, errStaffForbidden) {
						status, code, message = http.StatusForbidden, "FORBIDDEN", "The operation is not permitted"
					}
					writeError(w, r, status, code, message, nil)
					return
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorContextKey{}, actor)))
				return
			}
			TrustedGatewayAuth(internalToken)(next).ServeHTTP(w, r)
		})
	}
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	permission = canonicalCarePermission(permission)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := AuthenticatedActor(r.Context())
			if !ok {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
				return
			}
			if actor.PrincipalType != "" && actor.PrincipalType != "EMPLOYEE" {
				writeError(w, r, http.StatusForbidden, "FORBIDDEN", "The operation is not permitted", nil)
				return
			}
			if _, allowed := actor.Permissions[permission]; !allowed {
				writeError(w, r, http.StatusForbidden, "FORBIDDEN", "The operation is not permitted", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func canonicalCarePermission(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "care.tickets.read", "ticket_queue_view":
		return "TICKET_QUEUE_VIEW"
	case "care.tickets.claim", "ticket_claim":
		return "TICKET_CLAIM"
	case "care.tickets.notes.write", "ticket_note_write":
		return "TICKET_NOTE_WRITE"
	case "care.tickets.reply", "ticket_reply":
		return "TICKET_REPLY"
	case "care.tickets.transition", "ticket_transition":
		return "TICKET_TRANSITION"
	case "care.return.cases.view":
		return "RETURN_CASE_VIEW"
	case "care.return.cases.review":
		return "RETURN_CASE_REVIEW"
	case "care.return.cases.decide", "care.return.approve":
		return "RETURN_CASE_DECIDE"
	case "care.return.refund_without_return":
		return "RETURN_CASE_REFUND_WITHOUT_RETURN"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}
