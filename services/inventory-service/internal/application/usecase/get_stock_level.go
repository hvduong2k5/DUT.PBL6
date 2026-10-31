package usecase

import (
	"context"
	"strings"

	"dut-pbl6/inventory-service/internal/application/port"
	"dut-pbl6/inventory-service/internal/domain/entity"
)

// StockLevelOutput contains inventory summary for a SKU.
type StockLevelOutput struct {
	SKU          string `json:"sku"`
	PhysicalQty  int    `json:"physical_qty"`
	ReservedQty  int    `json:"reserved_qty"`
	AvailableQty int    `json:"available_qty"`
	StockStatus  string `json:"stock_status"`
}

// GetStockLevelUseCase retrieves stock levels for requested SKUs.
type GetStockLevelUseCase struct {
	repo port.InventoryRepository
}

// NewGetStockLevelUseCase creates a new GetStockLevelUseCase.
func NewGetStockLevelUseCase(repo port.InventoryRepository) *GetStockLevelUseCase {
	return &GetStockLevelUseCase{
		repo: repo,
	}
}

// Execute retrieves stock level information for the given SKUs.
func (uc *GetStockLevelUseCase) Execute(ctx context.Context, skus []string) ([]StockLevelOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(skus) == 0 {
		return []StockLevelOutput{}, nil
	}

	// Filter and deduplicate SKUs while preserving order
	cleanSKUs := make([]string, 0, len(skus))
	seen := make(map[string]bool)
	for _, s := range skus {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			cleanSKUs = append(cleanSKUs, s)
		}
	}

	if len(cleanSKUs) == 0 {
		return []StockLevelOutput{}, nil
	}

	items, err := uc.repo.GetItemsBySKUs(ctx, cleanSKUs)
	if err != nil {
		return nil, err
	}

	itemMap := make(map[string]*entity.InventoryItem, len(items))
	for _, item := range items {
		itemMap[item.SKU] = item
	}

	results := make([]StockLevelOutput, 0, len(cleanSKUs))
	for _, sku := range cleanSKUs {
		item, exists := itemMap[sku]
		if !exists {
			results = append(results, StockLevelOutput{
				SKU:          sku,
				PhysicalQty:  0,
				ReservedQty:  0,
				AvailableQty: 0,
				StockStatus:  "OUT_OF_STOCK",
			})
			continue
		}

		avail := item.AvailableQty()
		stockStatus := "IN_STOCK"
		if item.Status == entity.ItemStatusSuspended {
			stockStatus = "OUT_OF_STOCK"
		} else if avail <= 0 {
			stockStatus = "OUT_OF_STOCK"
		} else if avail <= 10 {
			stockStatus = "LOW_STOCK"
		}

		results = append(results, StockLevelOutput{
			SKU:          item.SKU,
			PhysicalQty:  item.PhysicalQty,
			ReservedQty:  item.ReservedQty,
			AvailableQty: avail,
			StockStatus:  stockStatus,
		})
	}

	return results, nil
}
