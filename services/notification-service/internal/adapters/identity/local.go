// Package identity supplies a loopback-only local credential adapter.
// It is not a substitute for the gated production service identity contract.
package identity

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"github.com/omamx/notification-service/internal/domain"
	"github.com/omamx/notification-service/internal/ports"
	"strings"
)

type Local struct{ Token string }

func (l Local) Authenticate(_ context.Context, header string) (ports.Principal, error) {
	if len(l.Token) < 32 || !strings.HasPrefix(header, "Bearer ") {
		return ports.Principal{}, domain.ErrUnauthenticated
	}
	got := sha256.Sum256([]byte(strings.TrimPrefix(header, "Bearer ")))
	want := sha256.Sum256([]byte(l.Token))
	if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
		return ports.Principal{}, domain.ErrUnauthenticated
	}
	return ports.Principal{Producer: "order-service", Purposes: map[string]bool{"ORDER_CONFIRMATION": true}}, nil
}
