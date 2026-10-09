package httpapi

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type actorContextKey struct{}

type Actor struct {
	ID          uuid.UUID
	Permissions map[string]struct{}
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
				if normalized := strings.ToLower(strings.TrimSpace(permission)); normalized != "" {
					permissions[normalized] = struct{}{}
				}
			}
			actor := Actor{ID: actorID, Permissions: permissions}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorContextKey{}, actor)))
		})
	}
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	permission = strings.ToLower(permission)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := AuthenticatedActor(r.Context())
			if !ok {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
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
