package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/infrastructure/cache"
	"github.com/omamx/profile-service/tests/testhelper"
	"github.com/stretchr/testify/require"
)

func TestDualLayerCache_MultiPodInvalidation_PurgesL1AcrossPods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	redisClient, cleanup := testhelper.SetupRedisContainer(t)
	defer cleanup()

	// Step 1: Simulate 2 distinct Kubernetes Pod instances sharing Redis
	podA, err := cache.NewDualLayerCache("pod-profile-replica-a", 100, redisClient)
	require.NoError(t, err)
	defer podA.Close()

	podB, err := cache.NewDualLayerCache("pod-profile-replica-b", 100, redisClient)
	require.NoError(t, err)
	defer podB.Close()

	// Step 2: Start background Pub/Sub subscribers on both pods
	err = podA.StartSubscriber(ctx)
	require.NoError(t, err)
	err = podB.StartSubscriber(ctx)
	require.NoError(t, err)

	// Verify both pod subscribers are active on Redis without hard sleep
	require.Eventually(t, func() bool {
		numSub, err := redisClient.PubSubNumSub(ctx, cache.DefaultInvalidationChannel).Result()
		return err == nil && numSub[cache.DefaultInvalidationChannel] >= 2
	}, 2*time.Second, 10*time.Millisecond, "Both pod subscribers must be actively registered in Redis")

	// Step 3: Populate cache with initial customer profile
	custID := uuid.New().String()
	initialProfile := &domain.CustomerProfile{
		ID:          uuid.MustParse(custID),
		FullName:    "Nguyễn Thị Hương",
		PhoneNumber: "0905001122",
		Email:       "huong.nguyen@hue-sweet.vn",
		Status:      "ACTIVE",
		Version:     1,
	}

	// Store in podA and also prime podB's L1 cache
	err = podA.Set(ctx, custID, initialProfile, 15*time.Minute)
	require.NoError(t, err)

	// Simulate podB reading from L2 and populating its L1
	cachedProfileB, found, err := podB.Get(ctx, custID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "0905001122", cachedProfileB.PhoneNumber)

	// Confirm both pods have the item cached in their L1 RAM
	_, inL1A := podA.GetL1Direct(custID)
	require.True(t, inL1A, "podA must have item in L1 RAM")

	_, inL1B := podB.GetL1Direct(custID)
	require.True(t, inL1B, "podB must have item in L1 RAM")

	// Step 4: Write Path Invalidation triggered on podA (e.g. user updated phone number)
	err = podA.Invalidate(ctx, custID)
	require.NoError(t, err)

	// Wait up to 500ms for Pub/Sub message propagation to podB
	require.Eventually(t, func() bool {
		_, stillInL1B := podB.GetL1Direct(custID)
		return !stillInL1B
	}, 1*time.Second, 20*time.Millisecond, "podB L1 cache MUST be evicted via Redis Pub/Sub broadcast")

	// Step 5: Verification of invariants
	// 1. podA's L1 is empty
	_, stillInL1A := podA.GetL1Direct(custID)
	require.False(t, stillInL1A, "podA L1 cache must be empty after Invalidate")

	// 2. podB's L1 is empty
	_, stillInL1B := podB.GetL1Direct(custID)
	require.False(t, stillInL1B, "podB L1 cache must be empty after Invalidate broadcast")

	// 3. podB calling Get misses cache completely (must return found=false), forcing a fresh DB read
	_, foundAfterInvalidate, err := podB.Get(ctx, custID)
	require.NoError(t, err)
	require.False(t, foundAfterInvalidate, "Get after invalidation must miss cache, preventing stale address read")
}
