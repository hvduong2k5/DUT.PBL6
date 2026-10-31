package redis

import (
	"context"
	"sync"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"github.com/google/uuid"
)

// RedisClient defines the subset of Redis commands required by Redlock.
type RedisClient interface {
	SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error)
	Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
	Del(ctx context.Context, keys ...string) error
}

// RedlockService implements port.LockService with distributed locking.
type RedlockService struct {
	client     RedisClient
	lockTokens sync.Map // map[string]string (key -> token)
}

// NewRedlockService creates a new RedlockService instance.
func NewRedlockService(client RedisClient) *RedlockService {
	return &RedlockService{
		client: client,
	}
}

// Lua script for atomic unlock only if the token matches
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

// AcquireLock attempts to acquire a distributed lock on the specified resource key.
// Pattern: lock:inventory:sku:{sku}, TTL default: 3 seconds.
func (s *RedlockService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ttl <= 0 {
		ttl = 3 * time.Second
	}
	token := uuid.NewString()

	if s.client == nil {
		// Fallback for testing when no client is configured
		return true, nil
	}

	ok, err := s.client.SetNX(ctx, key, token, ttl)
	if err != nil {
		return false, err
	}
	if ok {
		s.lockTokens.Store(key, token)
		return true, nil
	}
	return false, nil
}

// ReleaseLock releases the acquired lock atomically using a Lua script.
func (s *RedlockService) ReleaseLock(ctx context.Context, key string) error {
	if s.client == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	val, ok := s.lockTokens.Load(key)
	if !ok {
		return nil
	}
	token, _ := val.(string)

	_, err := s.client.Eval(ctx, unlockScript, []string{key}, token)
	if err == nil {
		s.lockTokens.Delete(key)
	}
	return err
}

// Ensure interface compliance
var _ port.LockService = (*RedlockService)(nil)
