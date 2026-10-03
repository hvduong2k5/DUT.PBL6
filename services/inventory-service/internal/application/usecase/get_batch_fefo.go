package usecase

import (
	"context"
	"sort"
	"strings"
	"time"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
)

// GetBatchFEFODetailsUseCase retrieves production batches sorted by FEFO (exp_date ASC).
type GetBatchFEFODetailsUseCase struct {
	repo port.InventoryRepository
}

// NewGetBatchFEFODetailsUseCase creates a new GetBatchFEFODetailsUseCase.
func NewGetBatchFEFODetailsUseCase(repo port.InventoryRepository) *GetBatchFEFODetailsUseCase {
	return &GetBatchFEFODetailsUseCase{
		repo: repo,
	}
}

// Execute retrieves batches for the given SKU in strict FEFO order.
func (uc *GetBatchFEFODetailsUseCase) Execute(ctx context.Context, sku string) ([]*entity.Batch, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, entity.ErrInvalidBatchData
	}

	// Verify that the SKU exists in inventory
	_, err := uc.repo.GetItemBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	batches, err := uc.repo.GetBatchesBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	for _, b := range batches {
		b.CheckExpiry(now)
	}

	// Guarantee FEFO ordering: earliest expiry date first, then earliest creation time
	sort.SliceStable(batches, func(i, j int) bool {
		if batches[i].ExpDate.Equal(batches[j].ExpDate) {
			return batches[i].CreatedAt.Before(batches[j].CreatedAt)
		}
		return batches[i].ExpDate.Before(batches[j].ExpDate)
	})

	return batches, nil
}
