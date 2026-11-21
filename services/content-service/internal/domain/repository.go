package domain

import (
	"context"
	"github.com/google/uuid"
)

// Mutate must lock, check expectedVersion, apply the callback and persist atomically.
type Repository interface {
	Create(context.Context, *Article) error
	Get(context.Context, uuid.UUID) (*Article, error)
	GetPublished(context.Context, string) (*Article, error)
	List(context.Context, Filter) ([]Article, int64, error)
	Mutate(context.Context, uuid.UUID, int64, func(*Article) error) (*Article, error)
}

// IdempotencyRecord is the stored outcome of one Idempotency-Key. A record is
// scoped to the verified principal, so keys never collide across users.
type IdempotencyRecord struct {
	RequestHash string
	Completed   bool
	Status      int
	Headers     map[string]string
	Body        []byte
}

// IdempotencyStore guarantees a retried write returns the first response.
// Begin reports created=true when the caller owns the key and must run the
// operation; otherwise the existing record is returned.
type IdempotencyStore interface {
	Begin(ctx context.Context, principal uuid.UUID, key, requestHash string) (rec IdempotencyRecord, created bool, err error)
	Complete(ctx context.Context, principal uuid.UUID, key string, status int, headers map[string]string, body []byte) error
	Abandon(ctx context.Context, principal uuid.UUID, key string) error
}
