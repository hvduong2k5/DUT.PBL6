package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// GoRedisAdapter adapts a go-redis client to the RedisClient interface required by RedlockService.
type GoRedisAdapter struct {
	client *goredis.Client
}

// NewGoRedisAdapter creates a new GoRedisAdapter.
func NewGoRedisAdapter(client *goredis.Client) *GoRedisAdapter {
	return &GoRedisAdapter{client: client}
}

// SetNX wraps go-redis SetNX.
func (a *GoRedisAdapter) SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	return a.client.SetNX(ctx, key, value, expiration).Result()
}

// Eval wraps go-redis Eval.
func (a *GoRedisAdapter) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return a.client.Eval(ctx, script, keys, args...).Result()
}

// Del wraps go-redis Del.
func (a *GoRedisAdapter) Del(ctx context.Context, keys ...string) error {
	return a.client.Del(ctx, keys...).Err()
}
