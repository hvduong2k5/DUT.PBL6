package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
)

// Verifier must cryptographically verify credentials (issuer, audience, expiry,
// signature) or authenticate a trusted gateway before returning an identity.
// A token that is valid but not an EMPLOYEE staff token returns
// domain.ErrForbidden; every other failure is treated as unauthenticated.
// No implementation that trusts client identity headers is provided.
type Verifier interface {
	Verify(*http.Request) (domain.Identity, error)
}
type identityKey struct{}

func Identity(ctx context.Context) domain.Identity {
	i, _ := ctx.Value(identityKey{}).(domain.Identity)
	return i
}

// Authenticate admits only verified staff. Individual operations still check
// their own permission in the use case.
func Authenticate(v Verifier, deny func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v == nil {
				deny(w, r, domain.ErrUnauthenticated)
				return
			}
			i, err := v.Verify(r)
			if errors.Is(err, domain.ErrForbidden) {
				deny(w, r, domain.ErrForbidden)
				return
			}
			if err != nil || i.ID == [16]byte{} {
				deny(w, r, domain.ErrUnauthenticated)
				return
			}
			if !i.Staff() {
				deny(w, r, domain.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, i)))
		})
	}
}
