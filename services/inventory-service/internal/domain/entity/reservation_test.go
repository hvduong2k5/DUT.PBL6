package entity_test

import (
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStockReservation_NewValidation(t *testing.T) {
	alloc := entity.ReservationItemAllocation{
		SKU:          "SKU-001",
		BatchID:      uuid.New(),
		AllocatedQty: 10,
	}

	// Empty orderID
	res1, err1 := entity.NewStockReservation("", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	assert.ErrorIs(t, err1, entity.ErrInvalidBatchData)
	assert.Nil(t, res1)

	// Empty allocations
	res2, err2 := entity.NewStockReservation("ORD-001", entity.DefaultReservationTTL, nil)
	assert.ErrorIs(t, err2, entity.ErrInvalidAllocationRequest)
	assert.Nil(t, res2)

	// Valid creation with non-positive TTL defaults to 15 mins
	res3, err3 := entity.NewStockReservation("ORD-001", 0, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err3)
	require.NotNil(t, res3)
	assert.Equal(t, entity.ReservationStatusPending, res3.Status)
	assert.Equal(t, "ORD-001", res3.OrderID)
	assert.Len(t, res3.Allocations, 1)
	assert.Equal(t, res3.ID, res3.Allocations[0].ReservationID)
	assert.True(t, res3.ExpiresAt.After(res3.CreatedAt))
}

func TestStockReservation_LifecycleTransitions(t *testing.T) {
	alloc := entity.ReservationItemAllocation{
		SKU:          "SKU-001",
		BatchID:      uuid.New(),
		AllocatedQty: 10,
	}

	// Test Release
	resRelease, err := entity.NewStockReservation("ORD-REL", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resRelease.Release())
	assert.Equal(t, entity.ReservationStatusReleased, resRelease.Status)
	// Second release fails
	assert.ErrorIs(t, resRelease.Release(), entity.ErrReservationAlreadyProcessed)

	// Test Cancel
	resCancel, err := entity.NewStockReservation("ORD-CANCEL", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resCancel.Cancel())
	assert.Equal(t, entity.ReservationStatusCancelled, resCancel.Status)
	assert.ErrorIs(t, resCancel.Cancel(), entity.ErrReservationAlreadyProcessed)

	// Test Commit
	resCommit, err := entity.NewStockReservation("ORD-COMMIT", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resCommit.Commit())
	assert.Equal(t, entity.ReservationStatusCommitted, resCommit.Status)
	assert.ErrorIs(t, resCommit.Commit(), entity.ErrReservationAlreadyProcessed)

	// Test Expire
	resExpire, err := entity.NewStockReservation("ORD-EXP", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resExpire.Expire())
	assert.Equal(t, entity.ReservationStatusExpired, resExpire.Status)
	assert.ErrorIs(t, resExpire.Expire(), entity.ErrReservationAlreadyProcessed)
}

func TestStockReservation_IsExpired(t *testing.T) {
	alloc := entity.ReservationItemAllocation{
		SKU:          "SKU-001",
		BatchID:      uuid.New(),
		AllocatedQty: 10,
	}
	res, err := entity.NewStockReservation("ORD-001", 10*time.Minute, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)

	now := time.Now()
	assert.False(t, res.IsExpired(now))
	assert.False(t, res.IsExpired(now.Add(5*time.Minute)))
	assert.True(t, res.IsExpired(now.Add(15*time.Minute)))
}

func TestStockReservation_TimezoneUTC(t *testing.T) {
	alloc := entity.ReservationItemAllocation{
		SKU:          "SKU-001",
		BatchID:      uuid.New(),
		AllocatedQty: 10,
	}
	res, err := entity.NewStockReservation("ORD-UTC", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	assert.Equal(t, time.UTC, res.CreatedAt.Location())
	assert.Equal(t, time.UTC, res.UpdatedAt.Location())
	assert.Equal(t, time.UTC, res.ExpiresAt.Location())

	require.NoError(t, res.Release())
	assert.Equal(t, time.UTC, res.UpdatedAt.Location())

	// Test Cancel UTC
	resCancel, err := entity.NewStockReservation("ORD-UTC-CANCEL", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resCancel.Cancel())
	assert.Equal(t, time.UTC, resCancel.UpdatedAt.Location())

	// Test Commit UTC
	resCommit, err := entity.NewStockReservation("ORD-UTC-COMMIT", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resCommit.Commit())
	assert.Equal(t, time.UTC, resCommit.UpdatedAt.Location())

	// Test Expire UTC
	resExpire, err := entity.NewStockReservation("ORD-UTC-EXPIRE", entity.DefaultReservationTTL, []entity.ReservationItemAllocation{alloc})
	require.NoError(t, err)
	require.NoError(t, resExpire.Expire())
	assert.Equal(t, time.UTC, resExpire.UpdatedAt.Location())
}

