package entity

import (
	"time"

	"github.com/google/uuid"
)

type BatchStatus string

const (
	BatchStatusQuarantine BatchStatus = "QUARANTINE"
	BatchStatusActive     BatchStatus = "ACTIVE"
	BatchStatusNearExpiry BatchStatus = "NEAR_EXPIRY"
	BatchStatusExpired    BatchStatus = "EXPIRED"
)

// NearExpiryThreshold is 45 days according to OCOP food safety requirements for sesame candy (Mè xửng)
const NearExpiryThreshold = 45 * 24 * time.Hour

// Batch represents a specific production lot of an InventoryItem.
type Batch struct {
	ID          uuid.UUID   `json:"id"`
	BatchCode   string      `json:"batch_code"`
	SKU         string      `json:"sku"`
	SupplierID  uuid.UUID   `json:"supplier_id"`
	MfgDate     time.Time   `json:"mfg_date"`
	ExpDate     time.Time   `json:"exp_date"`
	PhysicalQty int         `json:"physical_qty"`
	ReservedQty int         `json:"reserved_qty"`
	Status      BatchStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
}

// NewBatch creates a new Batch entity, enforcing INV-BI-04.
func NewBatch(
	batchCode string,
	sku string,
	supplierID uuid.UUID,
	mfgDate time.Time,
	expDate time.Time,
	physicalQty int,
) (*Batch, error) {
	if batchCode == "" || sku == "" || supplierID == uuid.Nil {
		return nil, ErrInvalidBatchData
	}
	// INV-BI-04: mfg_date < exp_date and physical_qty > 0
	if !mfgDate.Before(expDate) || physicalQty <= 0 {
		return nil, ErrInvalidBatchData
	}

	return &Batch{
		ID:          uuid.New(),
		BatchCode:   batchCode,
		SKU:         sku,
		SupplierID:  supplierID,
		MfgDate:     mfgDate,
		ExpDate:     expDate,
		PhysicalQty: physicalQty,
		ReservedQty: 0,
		Status:      BatchStatusActive,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// AvailableQty calculates available quantity for this batch (INV-BI-01).
func (b *Batch) AvailableQty() int {
	avail := b.PhysicalQty - b.ReservedQty
	if avail < 0 {
		return 0
	}
	return avail
}

// CanAllocate determines whether this batch is eligible for FEFO allocation.
// Quarantined and Expired batches are strictly barred from allocation (INV-BI-02).
func (b *Batch) CanAllocate() bool {
	return b.CanAllocateAt(time.Now().UTC())
}

// CanAllocateAt determines whether this batch is eligible for FEFO allocation at a specific time (INV-BI-02).
func (b *Batch) CanAllocateAt(currentDate time.Time) bool {
	if b.Status != BatchStatusActive && b.Status != BatchStatusNearExpiry {
		return false
	}
	// INV-BI-02: Zero Expired Sale. If expiry date is reached, auto-expire and reject.
	if !currentDate.Before(b.ExpDate) {
		b.Status = BatchStatusExpired
		return false
	}
	return b.AvailableQty() > 0
}

// CheckExpiry checks and updates the batch status based on currentDate (INV-BI-02).
// Returns true if the batch is expired.
func (b *Batch) CheckExpiry(currentDate time.Time) bool {
	// If current date is at or past expiry date (exp_date <= current_date)
	if !currentDate.Before(b.ExpDate) {
		b.Status = BatchStatusExpired
		return true
	}
	// Check near expiry threshold (< 45 days)
	if b.Status == BatchStatusActive && b.ExpDate.Sub(currentDate) <= NearExpiryThreshold {
		b.Status = BatchStatusNearExpiry
	}
	return false
}

// ReserveAt reserves stock in this batch at a specific point in time (INV-BI-02).
func (b *Batch) ReserveAt(qty int, now time.Time) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if !b.CanAllocateAt(now) {
		return ErrBatchCannotAllocate
	}
	if b.AvailableQty() < qty {
		return ErrInsufficientStock
	}
	b.ReservedQty += qty
	return nil
}

// Reserve reserves stock in this batch using the current time.
func (b *Batch) Reserve(qty int) error {
	return b.ReserveAt(qty, time.Now().UTC())
}

// Release releases previously reserved stock from this batch.
func (b *Batch) Release(qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if b.ReservedQty < qty {
		return ErrInvalidQuantity
	}
	b.ReservedQty -= qty
	return nil
}

// Commit finalizes deduction of physical and reserved stock from this batch.
func (b *Batch) Commit(qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if b.ReservedQty < qty || b.PhysicalQty < qty {
		return ErrInsufficientStock
	}
	b.PhysicalQty -= qty
	b.ReservedQty -= qty
	return nil
}
