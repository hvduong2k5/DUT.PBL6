package entity

import (
	"time"

	"github.com/google/uuid"
)

type ItemStatus string

const (
	ItemStatusActive    ItemStatus = "ACTIVE"
	ItemStatusSuspended ItemStatus = "SUSPENDED"
)

// DefaultWarehouseID represents the central warehouse for O Ma in Hue (MVP)
var DefaultWarehouseID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// InventoryItem is the Aggregate Root representing stock at the SKU level.
// ProductInventory is an alias for InventoryItem to match ubiquitous language across specs.
type InventoryItem struct {
	SKU         string     `json:"sku"`
	WarehouseID uuid.UUID  `json:"warehouse_id"`
	PhysicalQty int        `json:"physical_qty"`
	ReservedQty int        `json:"reserved_qty"`
	Status      ItemStatus `json:"status"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Batches     []*Batch   `json:"batches,omitempty"`
}

type ProductInventory = InventoryItem

// NewInventoryItem creates a new InventoryItem entity.
func NewInventoryItem(sku string, warehouseID uuid.UUID, physicalQty int) (*InventoryItem, error) {
	if sku == "" {
		return nil, ErrInvalidBatchData
	}
	if physicalQty < 0 {
		return nil, ErrInvalidQuantity
	}
	if warehouseID == uuid.Nil {
		warehouseID = DefaultWarehouseID
	}
	return &InventoryItem{
		SKU:         sku,
		WarehouseID: warehouseID,
		PhysicalQty: physicalQty,
		ReservedQty: 0,
		Status:      ItemStatusActive,
		UpdatedAt:   time.Now().UTC(),
		Batches:     make([]*Batch, 0),
	}, nil
}

// AvailableQty calculates the current available quantity (INV-BI-01: Q_avail = Q_phy - Q_res >= 0).
func (i *InventoryItem) AvailableQty() int {
	avail := i.PhysicalQty - i.ReservedQty
	if avail < 0 {
		return 0
	}
	return avail
}

// Reserve attempts to reserve a given quantity.
// Invariant check: Q_avail >= requested_qty.
// If insufficient, returns ErrInsufficientStock without modifying entity state.
func (i *InventoryItem) Reserve(qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if i.PhysicalQty-i.ReservedQty < qty {
		return ErrInsufficientStock
	}
	i.ReservedQty += qty
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// Release releases a previously reserved quantity.
func (i *InventoryItem) Release(qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if i.ReservedQty < qty {
		return ErrInvalidQuantity
	}
	i.ReservedQty -= qty
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// Commit finalizes deduction of physical and reserved quantities upon order payment.
func (i *InventoryItem) Commit(qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if i.ReservedQty < qty || i.PhysicalQty < qty {
		return ErrInsufficientStock
	}
	i.PhysicalQty -= qty
	i.ReservedQty -= qty
	i.UpdatedAt = time.Now().UTC()
	return nil
}
