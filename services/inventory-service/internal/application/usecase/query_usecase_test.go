package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetStockLevelUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("HappyPath_VariousStockLevels", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetStockLevelUseCase(mockRepo)

		item1 := &entity.InventoryItem{
			SKU:         "SKU-IN-STOCK",
			PhysicalQty: 100,
			ReservedQty: 10,
			Status:      entity.ItemStatusActive,
		}
		item2 := &entity.InventoryItem{
			SKU:         "SKU-LOW-STOCK",
			PhysicalQty: 10,
			ReservedQty: 5, // avail = 5 <= 10
			Status:      entity.ItemStatusActive,
		}
		item3 := &entity.InventoryItem{
			SKU:         "SKU-OUT-OF-STOCK",
			PhysicalQty: 10,
			ReservedQty: 10, // avail = 0
			Status:      entity.ItemStatusActive,
		}
		item4 := &entity.InventoryItem{
			SKU:         "SKU-SUSPENDED",
			PhysicalQty: 100,
			ReservedQty: 0,
			Status:      entity.ItemStatusSuspended,
		}

		mockRepo.On("GetItemsBySKUs", ctx, []string{"SKU-IN-STOCK", "SKU-LOW-STOCK", "SKU-OUT-OF-STOCK", "SKU-SUSPENDED", "SKU-MISSING"}).
			Return([]*entity.InventoryItem{item1, item2, item3, item4}, nil)

		outputs, err := uc.Execute(ctx, []string{"SKU-IN-STOCK", "SKU-LOW-STOCK", "SKU-OUT-OF-STOCK", "SKU-SUSPENDED", "SKU-MISSING"})
		require.NoError(t, err)
		require.Len(t, outputs, 5)

		assert.Equal(t, "SKU-IN-STOCK", outputs[0].SKU)
		assert.Equal(t, 90, outputs[0].AvailableQty)
		assert.Equal(t, "IN_STOCK", outputs[0].StockStatus)

		assert.Equal(t, "SKU-LOW-STOCK", outputs[1].SKU)
		assert.Equal(t, 5, outputs[1].AvailableQty)
		assert.Equal(t, "LOW_STOCK", outputs[1].StockStatus)

		assert.Equal(t, "SKU-OUT-OF-STOCK", outputs[2].SKU)
		assert.Equal(t, 0, outputs[2].AvailableQty)
		assert.Equal(t, "OUT_OF_STOCK", outputs[2].StockStatus)

		assert.Equal(t, "SKU-SUSPENDED", outputs[3].SKU)
		assert.Equal(t, "OUT_OF_STOCK", outputs[3].StockStatus)

		assert.Equal(t, "SKU-MISSING", outputs[4].SKU)
		assert.Equal(t, 0, outputs[4].AvailableQty)
		assert.Equal(t, "OUT_OF_STOCK", outputs[4].StockStatus)

		mockRepo.AssertExpectations(t)
	})

	t.Run("EmptySKUsList", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetStockLevelUseCase(mockRepo)

		outputs, err := uc.Execute(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, outputs)

		outputs, err = uc.Execute(ctx, []string{"  ", ""})
		require.NoError(t, err)
		assert.Empty(t, outputs)
	})

	t.Run("DuplicateSKUs_Deduplicated", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetStockLevelUseCase(mockRepo)

		item := &entity.InventoryItem{
			SKU:         "SKU-DUP",
			PhysicalQty: 50,
			ReservedQty: 0,
			Status:      entity.ItemStatusActive,
		}

		mockRepo.On("GetItemsBySKUs", ctx, []string{"SKU-DUP"}).
			Return([]*entity.InventoryItem{item}, nil)

		outputs, err := uc.Execute(ctx, []string{"SKU-DUP", "SKU-DUP", " SKU-DUP "})
		require.NoError(t, err)
		require.Len(t, outputs, 1)
		assert.Equal(t, "SKU-DUP", outputs[0].SKU)
		assert.Equal(t, 50, outputs[0].AvailableQty)

		mockRepo.AssertExpectations(t)
	})

	t.Run("NilContext_DoesNotPanic", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetStockLevelUseCase(mockRepo)

		mockRepo.On("GetItemsBySKUs", mock.Anything, []string{"SKU-001"}).
			Return([]*entity.InventoryItem{}, nil)

		outputs, err := uc.Execute(nil, []string{"SKU-001"})
		require.NoError(t, err)
		require.Len(t, outputs, 1)
		assert.Equal(t, "OUT_OF_STOCK", outputs[0].StockStatus)
	})

	t.Run("RepoError_Propagated", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetStockLevelUseCase(mockRepo)

		mockRepo.On("GetItemsBySKUs", ctx, []string{"SKU-ERR"}).
			Return(nil, errors.New("db connection down"))

		_, err := uc.Execute(ctx, []string{"SKU-ERR"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db connection down")
	})
}

func TestGetBatchFEFODetailsUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("HappyPath_FEFOSorted", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetBatchFEFODetailsUseCase(mockRepo)
		sku := "SKU-ME-XUNG-HUY-HOANG"

		item := &entity.InventoryItem{
			SKU:         sku,
			PhysicalQty: 150,
			ReservedQty: 0,
			Status:      entity.ItemStatusActive,
		}

		now := time.Now().UTC()
		b1 := &entity.Batch{
			ID:          uuid.New(),
			BatchCode:   "LOT-LATER",
			SKU:         sku,
			MfgDate:     now.AddDate(0, -1, 0),
			ExpDate:     now.AddDate(0, 5, 0),
			PhysicalQty: 50,
			ReservedQty: 0,
			Status:      entity.BatchStatusActive,
			CreatedAt:   now.AddDate(0, -1, 0),
		}
		b2 := &entity.Batch{
			ID:          uuid.New(),
			BatchCode:   "LOT-EARLIEST",
			SKU:         sku,
			MfgDate:     now.AddDate(0, -2, 0),
			ExpDate:     now.AddDate(0, 1, 0), // Expiring sooner
			PhysicalQty: 100,
			ReservedQty: 10,
			Status:      entity.BatchStatusActive,
			CreatedAt:   now.AddDate(0, -2, 0),
		}

		mockRepo.On("GetItemBySKU", ctx, sku).Return(item, nil)
		// Returned out of order from DB to verify UseCase guarantees FEFO sort
		mockRepo.On("GetBatchesBySKU", ctx, sku).Return([]*entity.Batch{b1, b2}, nil)

		batches, err := uc.Execute(ctx, sku)
		require.NoError(t, err)
		require.Len(t, batches, 2)

		assert.Equal(t, "LOT-EARLIEST", batches[0].BatchCode)
		assert.Equal(t, 90, batches[0].AvailableQty())
		assert.Equal(t, "LOT-LATER", batches[1].BatchCode)
		assert.Equal(t, 50, batches[1].AvailableQty())

		mockRepo.AssertExpectations(t)
	})

	t.Run("ItemNotFound", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetBatchFEFODetailsUseCase(mockRepo)
		sku := "SKU-NON-EXISTENT"

		mockRepo.On("GetItemBySKU", ctx, sku).Return(nil, entity.ErrItemNotFound)

		batches, err := uc.Execute(ctx, sku)
		require.ErrorIs(t, err, entity.ErrItemNotFound)
		assert.Nil(t, batches)
	})

	t.Run("EmptySKU_ReturnsInvalidBatchData", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetBatchFEFODetailsUseCase(mockRepo)

		batches, err := uc.Execute(ctx, "")
		require.ErrorIs(t, err, entity.ErrInvalidBatchData)
		assert.Nil(t, batches)

		batches, err = uc.Execute(ctx, "   ")
		require.ErrorIs(t, err, entity.ErrInvalidBatchData)
		assert.Nil(t, batches)
	})

	t.Run("NilContext_DoesNotPanic", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetBatchFEFODetailsUseCase(mockRepo)
		sku := "SKU-NIL-CTX"

		mockRepo.On("GetItemBySKU", mock.Anything, sku).Return(nil, entity.ErrItemNotFound)

		_, err := uc.Execute(nil, sku)
		require.ErrorIs(t, err, entity.ErrItemNotFound)
	})

	t.Run("BatchesRepoError_Propagated", func(t *testing.T) {
		mockRepo := new(MockInventoryRepository)
		uc := usecase.NewGetBatchFEFODetailsUseCase(mockRepo)
		sku := "SKU-DB-ERR"

		item := &entity.InventoryItem{SKU: sku, Status: entity.ItemStatusActive}
		mockRepo.On("GetItemBySKU", ctx, sku).Return(item, nil)
		mockRepo.On("GetBatchesBySKU", ctx, sku).Return(nil, errors.New("timeout querying batches"))

		batches, err := uc.Execute(ctx, sku)
		require.Error(t, err)
		assert.Nil(t, batches)
		assert.Contains(t, err.Error(), "timeout querying batches")
	})
}
