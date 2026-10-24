package core

import "time"

// StateRecord is a persistence-only record, never a public mutation DTO. The
// repository authenticates/decrypts it and verifies indexed projections before
// reconstituting the aggregate. New lifecycle phases extend this validation.
type StateRecord struct {
	SchemaVersion int                `json:"state_schema_version"`
	OrderID       string             `json:"order_id"`
	Owner         Owner              `json:"current_owner"`
	State         State              `json:"state"`
	Version       int64              `json:"version"`
	DecidedAt     time.Time          `json:"decided_at"`
	Blocks        Blocker            `json:"blocks"`
	Reservation   ResourceProof      `json:"reservation"`
	Allocation    AllocationEvidence `json:"allocation"`
	AllocatedAt   time.Time          `json:"allocated_at"`
	FirstPaid     bool               `json:"first_paid"`
	Fulfillment   FulfillmentRecord  `json:"fulfillment"`
}

func (a Aggregate) Record() StateRecord {
	return StateRecord{1, a.ID(), a.Owner(), a.state, a.version, a.decidedAt, a.blocks, a.reservation, a.allocation, a.allocatedAt, a.firstPaid, a.fulfillment}
}

func Restore(snapshot Snapshot, r StateRecord) (Aggregate, error) {
	s := snapshot.Data()
	if snapshot.canonical == "" || r.SchemaVersion != 1 || r.OrderID != s.OrderID || r.Owner.Validate() != nil || r.Version < 1 || r.DecidedAt.IsZero() || r.DecidedAt.Before(s.PlacedAt) ||
		r.Reservation.validate(s) != nil || r.Blocks & ^(HoldBlock|CancellationBlock|RefundBlock) != 0 {
		return Aggregate{}, ErrState
	}
	allocated := r.Allocation.AllocationID != ""
	if allocated {
		p := r.Allocation
		if s.Payment.Method != "VIETQR" || p.OrderID != s.OrderID || !validID(p.AllocationID) || !validID(p.ReceiptID) ||
			p.Currency != "VND" || p.Amount != s.Money.FinalAmount || p.ReceiverProfileID != *s.Payment.ReceiverProfileID ||
			r.AllocatedAt.Before(s.PlacedAt) || !r.AllocatedAt.Before(*s.Payment.ExpiresAt) || r.DecidedAt.Before(r.AllocatedAt) {
			return Aggregate{}, ErrState
		}
	} else if r.Allocation != (AllocationEvidence{}) || !r.AllocatedAt.IsZero() {
		return Aggregate{}, ErrState
	}
	switch r.State {
	case PendingPayment:
		if s.Payment.Method != "VIETQR" || allocated || r.FirstPaid || r.Reservation.State != "RESERVED" {
			return Aggregate{}, ErrState
		}
	case PaymentFinalizing:
		if !allocated || r.FirstPaid {
			return Aggregate{}, ErrState
		}
	case Paid:
		if !allocated || !r.FirstPaid || r.Reservation.State != "COMMITTED" {
			return Aggregate{}, ErrState
		}
	case ConfirmedCOD:
		if s.Payment.Method != "COD" || allocated || r.FirstPaid || r.Reservation.State != "COMMITTED" {
			return Aggregate{}, ErrState
		}
	case Processing, Packed, Shipped, Delivered, DeliveryFailed, Completed:
		if r.Reservation.State != "COMMITTED" || (s.Payment.Method == "VIETQR" && (!allocated || !r.FirstPaid)) || (s.Payment.Method == "COD" && (allocated || r.FirstPaid)) {
			return Aggregate{}, ErrState
		}
	case CancelledByUser, CancelledByAdmin:
		if s.Payment.Method != "VIETQR" || allocated || r.FirstPaid {
			return Aggregate{}, ErrState
		}
	case CancelledTimeout:
		if s.Payment.Method != "VIETQR" || r.FirstPaid || (allocated && r.Reservation.State != "EXPIRED" && r.Reservation.State != "RELEASED") {
			return Aggregate{}, ErrState
		}
	default:
		return Aggregate{}, ErrState
	}
	if r.Fulfillment.validate(r.State, s.PlacedAt, r.DecidedAt) != nil {
		return Aggregate{}, ErrState
	}
	owner := r.Owner
	if owner.CustomerID != nil {
		id := *owner.CustomerID
		owner.CustomerID = &id
	}
	if owner.GuestSessionID != nil {
		id := *owner.GuestSessionID
		owner.GuestSessionID = &id
	}
	return Aggregate{snapshot, owner, r.State, r.Version, r.DecidedAt, r.Blocks, r.Reservation, r.Allocation, r.AllocatedAt, r.FirstPaid, r.Fulfillment}, nil
}
