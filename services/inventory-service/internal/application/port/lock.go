package port

import (
	"context"
	"time"
)

// LockService provides distributed locking (e.g. Redis Redlock) to guard against concurrent checkout races.
type LockService interface {
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error
}
