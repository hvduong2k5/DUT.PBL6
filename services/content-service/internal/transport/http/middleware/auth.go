package middleware

import (
	"context"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"net/http"
)

// Verifier must cryptographically verify credentials (issuer, audience, expiry,
// signature) or authenticate a trusted gateway before returning an identity.
// No implementation that trusts client identity headers is provided.
type Verifier interface {
	Verify(*http.Request) (domain.Identity, error)
}
type identityKey struct{}

func Identity(ctx context.Context) domain.Identity {
	i, _ := ctx.Value(identityKey{}).(domain.Identity)
	return i
}
func Authenticate(v Verifier, deny func(http.ResponseWriter, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v == nil {
				deny(w, domain.ErrUnauthenticated)
				return
			}
			i, err := v.Verify(r)
			if err != nil || i.ID == [16]byte{} {
				deny(w, domain.ErrUnauthenticated)
				return
			}
			if err = i.Authorize(false); err != nil {
				deny(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, i)))
		})
	}
}
