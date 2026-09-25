package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

// AuthContext extracts UserID and Role from Kong Gateway forwarded headers or Bearer token.
func AuthContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Check for Gateway forwarded identity headers
		userIDHeader := r.Header.Get("X-User-ID")
		userRoleHeader := r.Header.Get("X-User-Role")

		if userIDHeader != "" {
			if uid, err := uuid.Parse(userIDHeader); err == nil {
				ctx = context.WithValue(ctx, UserIDKey, uid)
			}
		}

		if userRoleHeader != "" {
			ctx = context.WithValue(ctx, UserRoleKey, strings.ToUpper(userRoleHeader))
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole ensures that the authenticated caller possesses one of the allowed roles.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleVal := r.Context().Value(UserRoleKey)
			if roleVal == nil {
				http.Error(w, `{"error": "unauthorized: missing user credentials"}`, http.StatusUnauthorized)
				return
			}

			userRole, ok := roleVal.(string)
			if !ok {
				http.Error(w, `{"error": "unauthorized: invalid role context"}`, http.StatusUnauthorized)
				return
			}

			for _, allowed := range allowedRoles {
				if strings.EqualFold(userRole, allowed) {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error": "forbidden: insufficient permissions for this operation"}`, http.StatusForbidden)
		})
	}
}

// GetUserIDFromContext retrieves UUID from request context.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return uuid.Nil, false
	}
	uid, ok := val.(uuid.UUID)
	return uid, ok
}
