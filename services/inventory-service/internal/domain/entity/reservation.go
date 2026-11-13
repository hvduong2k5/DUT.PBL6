package entity

import (
	"time"

	"github.com/google/uuid"
)

type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "PENDING"
	ReservationStatusCommitted ReservationStatus = "COMMITTED"
	ReservationStatusReleased  ReservationStatus = "RELEASED"
	ReservationStatusCancelled ReservationStatus = "CANCELLED"
	ReservationStatusExpired   ReservationStatus = "EXPIRED"
)

// DefaultReservationTTL is 15 minutes as specified in INV-BI-05
const DefaultReservationTTL = 15 * time.Minute

// ReservationItemAllocation is a Value Object recording stock allocation for a specific batch.
type ReservationItemAllocation struct {
	ID            uuid.UUID `json:"id"`
	ReservationID uuid.UUID `json:"reservation_id"`
	SKU           string    `json:"sku"`
	BatchID       uuid.UUID `json:"batch_id"`
	AllocatedQty  int       `json:"allocated_qty"`
}

// StockReservation is the Aggregate Root managing reserved stock during checkout/Saga.
type StockReservation struct {
	ID          uuid.UUID                   `json:"id"`
	OrderID     string                      `json:"order_id"`
	Status      ReservationStatus           `json:"status"`
	ExpiresAt   time.Time                   `json:"expires_at"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
	Allocations []ReservationItemAllocation `json:"allocations"`
}

// NewStockReservation creates a new StockReservation with default 15-minute TTL (INV-BI-05).
func NewStockReservation(orderID string, ttl time.Duration, allocations []ReservationItemAllocation) (*StockReservation, error) {
	if orderID == "" {
		return nil, ErrInvalidBatchData
	}
	if len(allocations) == 0 {
		return nil, ErrInvalidAllocationRequest
	}
	if ttl <= 0 {
		ttl = DefaultReservationTTL
	}

	now := time.Now().UTC()
	resID := uuid.New()

	for i := range allocations {
		if allocations[i].ID == uuid.Nil {
			allocations[i].ID = uuid.New()
		}
		allocations[i].ReservationID = resID
	}

	return &StockReservation{
		ID:          resID,
		OrderID:     orderID,
		Status:      ReservationStatusPending,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		UpdatedAt:   now,
		Allocations: allocations,
	}, nil
}

// IsExpired checks whether this reservation has exceeded its TTL.
func (r *StockReservation) IsExpired(now time.Time) bool {
	return now.After(r.ExpiresAt)
}

// Release marks the reservation as released.
func (r *StockReservation) Release() error {
	if r.Status != ReservationStatusPending {
		return ErrReservationAlreadyProcessed
	}
	r.Status = ReservationStatusReleased
	r.UpdatedAt = time.Now().UTC()
	return nil
}

// Cancel marks the reservation as cancelled.
func (r *StockReservation) Cancel() error {
	if r.Status != ReservationStatusPending {
		return ErrReservationAlreadyProcessed
	}
	r.Status = ReservationStatusCancelled
	r.UpdatedAt = time.Now().UTC()
	return nil
}

// Commit marks the reservation as committed following successful order payment.
func (r *StockReservation) Commit() error {
	if r.Status != ReservationStatusPending {
		return ErrReservationAlreadyProcessed
	}
	r.Status = ReservationStatusCommitted
	r.UpdatedAt = time.Now().UTC()
	return nil
}

// Expire marks the reservation as expired when TTL passes.
func (r *StockReservation) Expire() error {
	if r.Status != ReservationStatusPending {
		return ErrReservationAlreadyProcessed
	}
	r.Status = ReservationStatusExpired
	r.UpdatedAt = time.Now().UTC()
	return nil
}
