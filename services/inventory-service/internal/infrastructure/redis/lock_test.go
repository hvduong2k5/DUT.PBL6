package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	infraRedis "dut-pbl6/inventory-service/internal/infrastructure/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) SetNX(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisClient) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	callArgs := m.Called(ctx, script, keys, args)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func TestRedlockService_AcquireAndRelease(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockRedisClient)
	key := "lock:inventory:sku:SKU-OC-001"

	// Mock SetNX returns true
	mockClient.On("SetNX", ctx, key, mock.AnythingOfType("string"), 3*time.Second).Return(true, nil)
	// Mock Eval returns int64(1)
	mockClient.On("Eval", ctx, mock.AnythingOfType("string"), []string{key}, mock.Anything).Return(int64(1), nil)

	lockService := infraRedis.NewRedlockService(mockClient)

	// Acquire lock
	acquired, err := lockService.AcquireLock(ctx, key, 3*time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Release lock
	err = lockService.ReleaseLock(ctx, key)
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestRedlockService_AcquireFailed_AlreadyLocked(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockRedisClient)
	key := "lock:inventory:sku:SKU-OC-001"

	// Mock SetNX returns false (lock held by someone else)
	mockClient.On("SetNX", ctx, key, mock.AnythingOfType("string"), 3*time.Second).Return(false, nil)

	lockService := infraRedis.NewRedlockService(mockClient)

	acquired, err := lockService.AcquireLock(ctx, key, 3*time.Second)
	require.NoError(t, err)
	assert.False(t, acquired)

	mockClient.AssertExpectations(t)
}

func TestRedlockService_AcquireError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockRedisClient)
	key := "lock:inventory:sku:SKU-OC-001"

	mockClient.On("SetNX", ctx, key, mock.AnythingOfType("string"), 3*time.Second).Return(false, errors.New("redis connection refused"))

	lockService := infraRedis.NewRedlockService(mockClient)

	acquired, err := lockService.AcquireLock(ctx, key, 3*time.Second)
	assert.Error(t, err)
	assert.False(t, acquired)

	mockClient.AssertExpectations(t)
}

func TestRedlockService_ReleaseLock_WithCleanupContext_WhenOriginalContextCancelled(t *testing.T) {
	origCtx, cancel := context.WithCancel(context.Background())
	cancel() // canceled upfront

	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(origCtx), 2*time.Second)
	defer cleanupCancel()

	mockClient := new(MockRedisClient)
	key := "lock:inventory:sku:SKU-CLEANUP-001"

	mockClient.On("SetNX", mock.Anything, key, mock.AnythingOfType("string"), 3*time.Second).Return(true, nil)
	// Eval MUST receive cleanupCtx which has Err() == nil
	mockClient.On("Eval", mock.MatchedBy(func(c context.Context) bool {
		return c != nil && c.Err() == nil
	}), mock.AnythingOfType("string"), []string{key}, mock.Anything).Return(int64(1), nil)

	lockService := infraRedis.NewRedlockService(mockClient)

	acquired, err := lockService.AcquireLock(context.Background(), key, 3*time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)

	err = lockService.ReleaseLock(cleanupCtx, key)
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestRedlockService_ReleaseLock_RetryOnEvalError(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockRedisClient)
	key := "lock:inventory:sku:SKU-RETRY-001"

	mockClient.On("SetNX", ctx, key, mock.AnythingOfType("string"), 3*time.Second).Return(true, nil)
	// First Eval fails with a temporary network error
	mockClient.On("Eval", ctx, mock.AnythingOfType("string"), []string{key}, mock.Anything).
		Return(nil, errors.New("network timeout")).Once()
	// Second Eval (retry) succeeds
	mockClient.On("Eval", ctx, mock.AnythingOfType("string"), []string{key}, mock.Anything).
		Return(int64(1), nil).Once()

	lockService := infraRedis.NewRedlockService(mockClient)

	acquired, err := lockService.AcquireLock(ctx, key, 3*time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)

	// First release attempt fails
	err = lockService.ReleaseLock(ctx, key)
	assert.Error(t, err)
	assert.Equal(t, "network timeout", err.Error())

	// Second release attempt (retry) succeeds because token was preserved
	err = lockService.ReleaseLock(ctx, key)
	assert.NoError(t, err)

	// Third release attempt: token was deleted upon success, returns nil early
	err = lockService.ReleaseLock(ctx, key)
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestRedlockService_NilContextHandling(t *testing.T) {
	mockClient := new(MockRedisClient)
	key := "lock:inventory:sku:SKU-NIL-CTX"

	mockClient.On("SetNX", mock.Anything, key, mock.AnythingOfType("string"), 3*time.Second).Return(true, nil)
	mockClient.On("Eval", mock.Anything, mock.AnythingOfType("string"), []string{key}, mock.Anything).Return(int64(1), nil)

	lockService := infraRedis.NewRedlockService(mockClient)

	acquired, err := lockService.AcquireLock(nil, key, 3*time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)

	err = lockService.ReleaseLock(nil, key)
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
}


