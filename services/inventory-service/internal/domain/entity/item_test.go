package entity_test

import (
	"testing"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryItem_NewItemValidation(t *testing.T) {
	// Empty SKU returns ErrInvalidBatchData
	item1, err1 := entity.NewInventoryItem("", entity.DefaultWarehouseID, 100)
	assert.ErrorIs(t, err1, entity.ErrInvalidBatchData)
	assert.Nil(t, item1)

	// Negative physical quantity returns ErrInvalidQuantity
	item2, err2 := entity.NewInventoryItem("SKU-001", entity.DefaultWarehouseID, -10)
	assert.ErrorIs(t, err2, entity.ErrInvalidQuantity)
	assert.Nil(t, item2)

	// Nil warehouseID defaults to DefaultWarehouseID
	item3, err3 := entity.NewInventoryItem("SKU-001", uuid.Nil, 50)
	require.NoError(t, err3)
	require.NotNil(t, item3)
	assert.Equal(t, entity.DefaultWarehouseID, item3.WarehouseID)
	assert.Equal(t, 50, item3.PhysicalQty)
	assert.Equal(t, 0, item3.ReservedQty)
	assert.Equal(t, 50, item3.AvailableQty())
	assert.Equal(t, entity.ItemStatusActive, item3.Status)
}

func TestInventoryItem_ReserveAndRelease(t *testing.T) {
	item, err := entity.NewInventoryItem("SKU-001", entity.DefaultWarehouseID, 100)
	require.NoError(t, err)

	// Reserve non-positive quantity
	assert.ErrorIs(t, item.Reserve(0), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, item.Reserve(-5), entity.ErrInvalidQuantity)

	// Valid reservation
	require.NoError(t, item.Reserve(40))
	assert.Equal(t, 40, item.ReservedQty)
	assert.Equal(t, 60, item.AvailableQty())

	// Over-reservation
	assert.ErrorIs(t, item.Reserve(70), entity.ErrInsufficientStock)
	assert.Equal(t, 40, item.ReservedQty)

	// Release non-positive quantity
	assert.ErrorIs(t, item.Release(0), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, item.Release(-1), entity.ErrInvalidQuantity)

	// Over-release
	assert.ErrorIs(t, item.Release(50), entity.ErrInvalidQuantity)
	assert.Equal(t, 40, item.ReservedQty)

	// Valid release
	require.NoError(t, item.Release(15))
	assert.Equal(t, 25, item.ReservedQty)
	assert.Equal(t, 75, item.AvailableQty())
}

func TestInventoryItem_Commit(t *testing.T) {
	item, err := entity.NewInventoryItem("SKU-001", entity.DefaultWarehouseID, 100)
	require.NoError(t, err)

	require.NoError(t, item.Reserve(50))
	assert.Equal(t, 50, item.ReservedQty)
	assert.Equal(t, 50, item.AvailableQty())

	// Invalid quantity
	assert.ErrorIs(t, item.Commit(0), entity.ErrInvalidQuantity)
	assert.ErrorIs(t, item.Commit(-10), entity.ErrInvalidQuantity)

	// Over-commit (greater than reserved)
	assert.ErrorIs(t, item.Commit(60), entity.ErrInsufficientStock)

	// Valid commit
	require.NoError(t, item.Commit(30))
	assert.Equal(t, 70, item.PhysicalQty)
	assert.Equal(t, 20, item.ReservedQty)
	assert.Equal(t, 50, item.AvailableQty())
}
