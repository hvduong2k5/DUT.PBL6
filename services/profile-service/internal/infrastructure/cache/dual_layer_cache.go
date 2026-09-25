package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

const DefaultInvalidationChannel = "cache:invalidate:profile"

type DualLayerCache struct {
	podID       string
	channelName string
	l1Cache     *lru.Cache[string, *domain.CustomerProfile]
	redisClient *redis.Client
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewDualLayerCache creates an instance of DualLayerCache with L1 LRU and L2 Redis.
func NewDualLayerCache(podID string, l1Capacity int, redisClient *redis.Client) (*DualLayerCache, error) {
	l1, err := lru.New[string, *domain.CustomerProfile](l1Capacity)
	if err != nil {
		return nil, fmt.Errorf("failed creating L1 LRU cache: %w", err)
	}

	return &DualLayerCache{
		podID:       podID,
		channelName: DefaultInvalidationChannel,
		l1Cache:     l1,
		redisClient: redisClient,
		stopChan:    make(chan struct{}),
	}, nil
}

func (c *DualLayerCache) redisKey(customerID string) string {
	return fmt.Sprintf("profile:cust:%s", customerID)
}

// Get checks L1 RAM first, then fallback to L2 Redis.
func (c *DualLayerCache) Get(ctx context.Context, customerID string) (*domain.CustomerProfile, bool, error) {
	// 1. Check L1 Cache
	if val, ok := c.l1Cache.Get(customerID); ok {
		return val, true, nil
	}

	// 2. Check L2 Redis
	key := c.redisKey(customerID)
	data, err := c.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var profile domain.CustomerProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, false, err
	}

	// Backfill L1
	c.l1Cache.Add(customerID, &profile)
	return &profile, true, nil
}

// Set stores profile in both L1 RAM and L2 Redis.
func (c *DualLayerCache) Set(ctx context.Context, customerID string, profile *domain.CustomerProfile, l2TTL time.Duration) error {
	// Store in L1
	c.l1Cache.Add(customerID, profile)

	// Store in L2 Redis
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	key := c.redisKey(customerID)
	return c.redisClient.Set(ctx, key, data, l2TTL).Err()
}

// Invalidate removes profile from local L1, deletes from L2 Redis, and broadcasts Pub/Sub invalidation.
func (c *DualLayerCache) Invalidate(ctx context.Context, customerID string) error {
	// 1. Remove local L1
	c.l1Cache.Remove(customerID)

	// 2. Remove L2 Redis
	key := c.redisKey(customerID)
	if err := c.redisClient.Del(ctx, key).Err(); err != nil && err != redis.Nil {
		return fmt.Errorf("failed deleting redis key %s: %w", key, err)
	}

	// 3. Broadcast invalidation to all other pods
	msg := customerID
	if err := c.redisClient.Publish(ctx, c.channelName, msg).Err(); err != nil {
		return fmt.Errorf("failed publishing invalidation signal: %w", err)
	}

	return nil
}

// GetL1Direct is a test inspection helper to check if an item exists specifically in L1 RAM.
func (c *DualLayerCache) GetL1Direct(customerID string) (*domain.CustomerProfile, bool) {
	return c.l1Cache.Get(customerID)
}

// StartSubscriber starts listening to the Redis Pub/Sub channel to purge local L1 on broadcast messages.
func (c *DualLayerCache) StartSubscriber(ctx context.Context) error {
	pubsub := c.redisClient.Subscribe(ctx, c.channelName)

	// Verify connection
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed subscribing to invalidation channel: %w", err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case <-c.stopChan:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if msg != nil {
					c.l1Cache.Remove(msg.Payload)
				}
			}
		}
	}()

	return nil
}

// Close stops the subscriber goroutine.
func (c *DualLayerCache) Close() {
	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}
	c.wg.Wait()
}
