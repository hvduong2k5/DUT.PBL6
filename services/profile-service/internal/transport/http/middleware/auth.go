package middleware

import (
	"context"
	"crypto/subtle"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserIDKey     contextKey = "user_id"
	UserRoleKey   contextKey = "user_role"
	CustomerIDKey contextKey = "customer_id"
)

// Forwarded identities are accepted only from an authenticated internal caller.
// The token must remain server-side and the service must not be publicly exposed.
func AuthContextWithToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			supplied := r.Header.Get("X-Internal-Token")
			if token == "" || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				authError(w, 401, "UNAUTHENTICATED")
				return
			}
			id, err := uuid.Parse(r.Header.Get("X-User-ID"))
			if err != nil || id == uuid.Nil {
				authError(w, 401, "UNAUTHENTICATED")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, id)
			ctx = context.WithValue(ctx, UserRoleKey, strings.ToUpper(r.Header.Get("X-User-Role")))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type CustomerResolver interface {
	GetByUserID(context.Context, uuid.UUID) (*domain.CustomerProfile, error)
}

func ResolveCustomer(repo CustomerResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, ok := GetUserIDFromContext(r.Context())
			if !ok {
				authError(w, 401, "UNAUTHENTICATED")
				return
			}
			profile, err := repo.GetByUserID(r.Context(), uid)
			if err != nil {
				if err == domain.ErrCustomerNotFound {
					authError(w, 404, "CUSTOMER_NOT_FOUND")
				} else {
					authError(w, 500, "INTERNAL_ERROR")
				}
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), CustomerIDKey, profile.ID)))
		})
	}
}
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := GetUserIDFromContext(r.Context()); !ok {
				authError(w, 401, "UNAUTHENTICATED")
				return
			}
			role, _ := r.Context().Value(UserRoleKey).(string)
			for _, allowed := range roles {
				if strings.EqualFold(role, allowed) {
					next.ServeHTTP(w, r)
					return
				}
			}
			authError(w, 403, "FORBIDDEN")
		})
	}
}
func authError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"access denied","code":"` + code + `"}`))
}
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
func GetCustomerIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(CustomerIDKey).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
