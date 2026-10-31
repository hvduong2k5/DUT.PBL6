package entity

import "errors"

var (
	// ErrInsufficientStock is returned when requested quantity exceeds available quantity (INV-BI-01)
	ErrInsufficientStock = errors.New("insufficient stock")

	// ErrInvalidBatchData is returned when batch attributes fail invariant checks (INV-BI-04)
	ErrInvalidBatchData = errors.New("invalid batch data")

	// ErrInvalidAllocationRequest is returned when allocation request quantity is <= 0
	ErrInvalidAllocationRequest = errors.New("invalid allocation request: quantity must be greater than zero")

	// ErrBatchExpired is returned when attempting to allocate from an expired batch (INV-BI-02)
	ErrBatchExpired = errors.New("batch has expired")

	// ErrBatchCannotAllocate is returned when a batch is not in allocatable status (e.g. QUARANTINE, EXPIRED)
	ErrBatchCannotAllocate = errors.New("batch cannot be allocated in current status")

	// ErrReservationNotFound is returned when a reservation cannot be found
	ErrReservationNotFound = errors.New("stock reservation not found")

	// ErrReservationAlreadyProcessed is returned when a reservation has already been committed, released, or expired
	ErrReservationAlreadyProcessed = errors.New("reservation has already been processed")

	// ErrConcurrentUpdate is returned when a distributed lock or optimistic lock cannot be acquired
	ErrConcurrentUpdate = errors.New("concurrent update conflict: lock could not be acquired")

	// ErrInvalidQuantity is returned when quantity is negative or invalid
	ErrInvalidQuantity = errors.New("quantity must be non-negative")

	// ErrItemNotFound is returned when the inventory item is not found
	ErrItemNotFound = errors.New("inventory item not found")
)
