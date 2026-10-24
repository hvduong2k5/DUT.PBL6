package core

import "time"

type State string

const (
	PendingPayment    State = "PENDING_PAYMENT"
	PaymentFinalizing State = "PAYMENT_FINALIZING"
	Paid              State = "PAID"
	ConfirmedCOD      State = "CONFIRMED_COD"
	Processing        State = "PROCESSING"
	Packed            State = "PACKED"
	Shipped           State = "SHIPPED"
	Delivered         State = "DELIVERED"
	DeliveryFailed    State = "DELIVERY_FAILED"
	Completed         State = "COMPLETED"
	CancelledTimeout  State = "CANCELLED_TIMEOUT"
	CancelledByUser   State = "CANCELLED_BY_USER"
	CancelledByAdmin  State = "CANCELLED_BY_ADMIN"
)

func (s State) Terminal() bool {
	return s == Completed || s == CancelledTimeout || s == CancelledByUser || s == CancelledByAdmin
}

// StateAllowed is only a structural graph, never authorization or proof that
// an effect occurred. Apply business decisions through the guarded operations.
func StateAllowed(from, to State) bool {
	switch from {
	case PendingPayment:
		return to == PaymentFinalizing || to == CancelledTimeout || to == CancelledByUser || to == CancelledByAdmin
	case PaymentFinalizing:
		return to == Paid || to == CancelledTimeout
	case Paid:
		return to == Processing || to == CancelledByAdmin
	case ConfirmedCOD:
		return to == Processing || to == CancelledByUser || to == CancelledByAdmin
	case Processing:
		return to == Packed
	case Packed:
		return to == Shipped
	case Shipped:
		return to == Delivered || to == DeliveryFailed
	case DeliveryFailed:
		return to == Shipped
	case Delivered:
		return to == Completed
	}
	return false
}

type Milestone string

const (
	FirstPaid      Milestone = "FIRST_PAID"
	OrderCompleted Milestone = "COMPLETED"
)

// DecisionContext must use the current locked root version and DB decision
// clock. The domain deliberately does not own a wall clock or lock manager.
type DecisionContext struct {
	ExpectedVersion int64
	At              time.Time
}

// ResourceProof contains validated adapter evidence, not a client DTO.
// Authentication, command hash/decision verification and authoritative queries
// belong to the port; these association/expiry/version guards remain mandatory.
type ResourceProof struct {
	OrderID         string
	ReservationID   string
	ItemsDigest     ManifestHash
	WarehouseID     string
	State           string
	ResourceVersion int64
	DecisionID      string
	DecisionAt      time.Time
	ExpiresAt       time.Time
}

func (p ResourceProof) validate(s SnapshotInput) error {
	if p.OrderID != s.OrderID || p.ReservationID == "" || p.ItemsDigest != s.Provenance.ItemsDigest ||
		p.WarehouseID != s.WarehouseID || p.ResourceVersion < 1 || p.DecisionID == "" ||
		p.DecisionAt.IsZero() || p.ExpiresAt.IsZero() {
		return ErrProof
	}
	switch p.State {
	case "RESERVED", "COMMITTED":
		if !p.DecisionAt.Before(p.ExpiresAt) {
			return ErrProof
		}
	case "EXPIRED":
		if p.DecisionAt.Before(p.ExpiresAt) {
			return ErrProof
		}
	case "RELEASED":
	default:
		return ErrProof
	}
	return nil
}

type AllocationEvidence struct {
	OrderID           string
	AllocationID      string
	ReceiptID         string
	ReceiverProfileID string
	Amount            int64
	Currency          string
}

type Blocker uint8

const (
	HoldBlock Blocker = 1 << iota
	CancellationBlock
	RefundBlock
)

// Aggregate is the target decision model. The sandbox Order DTO remains a
// legacy-cohort transport projection until explicit persistence integration.
// Value-returning methods leave the original root unchanged on rejected input.
type Aggregate struct {
	snapshot     Snapshot
	currentOwner Owner
	state        State
	version      int64
	decidedAt    time.Time
	blocks       Blocker
	reservation  ResourceProof
	allocation   AllocationEvidence
	allocatedAt  time.Time
	firstPaid    bool
	fulfillment  FulfillmentRecord
}

func Place(snapshot Snapshot, plannedOrderID string, proof ResourceProof, at time.Time) (Aggregate, error) {
	s := snapshot.Data()
	if snapshot.canonical == "" || plannedOrderID != s.OrderID || !at.Equal(s.PlacedAt) || proof.validate(s) != nil {
		return Aggregate{}, ErrProof
	}
	a := Aggregate{snapshot: snapshot, currentOwner: s.Owner, version: 1, decidedAt: at, reservation: proof}
	switch s.Payment.Method {
	case "VIETQR":
		if proof.State != "RESERVED" {
			return Aggregate{}, ErrProof
		}
		if s.Payment.ExpiresAt.After(proof.ExpiresAt) {
			return Aggregate{}, ErrDeadline
		}
		cutoff := proof.ExpiresAt
		if s.Payment.ExpiresAt.Before(cutoff) {
			cutoff = *s.Payment.ExpiresAt
		}
		if !at.Add(30 * time.Second).Before(cutoff) {
			return Aggregate{}, ErrDeadline
		}
		a.state = PendingPayment
	case "COD":
		if proof.State != "COMMITTED" {
			return Aggregate{}, ErrProof
		}
		a.state = ConfirmedCOD
	default:
		return Aggregate{}, ErrSnapshot
	}
	return a, nil
}

func (a Aggregate) ID() string         { return a.snapshot.Data().OrderID }
func (a Aggregate) State() State       { return a.state }
func (a Aggregate) Version() int64     { return a.version }
func (a Aggregate) Snapshot() Snapshot { return a.snapshot }
func (a Aggregate) Owner() Owner {
	o := a.currentOwner
	if o.CustomerID != nil {
		id := *o.CustomerID
		o.CustomerID = &id
	}
	if o.GuestSessionID != nil {
		id := *o.GuestSessionID
		o.GuestSessionID = &id
	}
	return o
}
func (a Aggregate) HasMilestone(m Milestone) bool {
	return m == FirstPaid && a.firstPaid || m == OrderCompleted && a.state == Completed && a.fulfillment.CompletionDecision != ""
}

func (a Aggregate) guard(c DecisionContext) error {
	if a.version < 1 || a.snapshot.canonical == "" || c.At.IsZero() {
		return ErrState
	}
	if c.ExpectedVersion != a.version || a.version == int64(^uint64(0)>>1) {
		return ErrVersion
	}
	if c.At.Before(a.decidedAt) {
		return ErrDeadline
	}
	return nil
}

// Block records a restriction; it never extends deadlines. Removing a hold or
// clearing a refund/cancellation obligation requires the later verified control
// workflow and is intentionally not a generic setter on the root.
func (a Aggregate) Block(c DecisionContext, block Blocker) (Aggregate, error) {
	if err := a.guard(c); err != nil {
		return a, err
	}
	if block == 0 || block & ^(HoldBlock|CancellationBlock|RefundBlock) != 0 {
		return a, ErrBlocked
	}
	if a.state.Terminal() {
		return a, ErrState
	}
	if a.blocks&block == block {
		return a, nil
	}
	a.blocks |= block
	a.version++
	a.decidedAt = c.At
	return a, nil
}

func (a Aggregate) Allocate(c DecisionContext, e AllocationEvidence) (Aggregate, error) {
	if err := a.guard(c); err != nil {
		return a, err
	}
	s := a.snapshot.Data()
	if a.state != PendingPayment || s.Payment.Method != "VIETQR" {
		return a, ErrState
	}
	if a.blocks != 0 {
		return a, ErrBlocked
	}
	if !c.At.Before(*s.Payment.ExpiresAt) {
		return a, ErrDeadline
	}
	if e.OrderID != s.OrderID || !validID(e.AllocationID) || !validID(e.ReceiptID) ||
		e.ReceiverProfileID != *s.Payment.ReceiverProfileID || e.Currency != "VND" || e.Amount != s.Money.FinalAmount {
		return a, ErrPayment
	}
	a.allocation, a.allocatedAt, a.state = e, c.At, PaymentFinalizing
	a.version++
	a.decidedAt = c.At
	return a, nil
}

func (a Aggregate) checkResource(p ResourceProof) error {
	if p.validate(a.snapshot.Data()) != nil || p.ReservationID != a.reservation.ReservationID ||
		!p.ExpiresAt.Equal(a.reservation.ExpiresAt) || p.ResourceVersion < a.reservation.ResourceVersion {
		return ErrProof
	}
	if p.ResourceVersion == a.reservation.ResourceVersion && !sameResourceProof(p, a.reservation) {
		return ErrProof
	}
	return nil
}

func sameResourceProof(a, b ResourceProof) bool {
	return a.OrderID == b.OrderID && a.ReservationID == b.ReservationID && a.ItemsDigest == b.ItemsDigest &&
		a.WarehouseID == b.WarehouseID && a.State == b.State && a.ResourceVersion == b.ResourceVersion &&
		a.DecisionID == b.DecisionID && a.DecisionAt.Equal(b.DecisionAt) && a.ExpiresAt.Equal(b.ExpiresAt)
}

func (a Aggregate) ConfirmPaid(c DecisionContext, proof ResourceProof) (Aggregate, error) {
	if err := a.guard(c); err != nil {
		return a, err
	}
	if a.state != PaymentFinalizing || a.firstPaid || a.allocation.AllocationID == "" {
		return a, ErrState
	}
	if a.allocatedAt.IsZero() || !a.allocatedAt.Before(*a.snapshot.Data().Payment.ExpiresAt) {
		return a, ErrPayment
	}
	if a.blocks != 0 {
		return a, ErrBlocked
	}
	if a.checkResource(proof) != nil || proof.State != "COMMITTED" {
		return a, ErrProof
	}
	// Allocation was accepted before the payment cutoff; remote COMMITTED proof
	// may arrive after that cutoff. Never reinterpret it as expired merely by now.
	a.state, a.reservation, a.firstPaid = Paid, proof, true
	a.version++
	a.decidedAt = c.At
	return a, nil
}

func (a Aggregate) CancelPending(c DecisionContext, actor Owner) (Aggregate, error) {
	if !a.Owner().SameOwner(actor) {
		return a, ErrForbidden
	}
	return a.cancelPending(c, CancelledByUser)
}

// CancellationApproval is the normalized result of a trusted Care decision
// lookup, not an arbitrary approval ID supplied by a customer or staff request.
type CancellationApproval struct {
	OrderID       string
	OrderVersion  int64
	DecisionID    string
	PolicyVersion string
	Scope         string
	Allowed       bool
}

func (a Aggregate) CancelApprovedPending(c DecisionContext, approval CancellationApproval) (Aggregate, error) {
	if approval.OrderID != a.ID() || approval.OrderVersion != a.version || approval.DecisionID == "" ||
		approval.PolicyVersion != a.snapshot.Data().Policies.Order || approval.Scope != "ORDER_CANCEL_PENDING" || !approval.Allowed {
		return a, ErrProof
	}
	return a.cancelPending(c, CancelledByAdmin)
}

func (a Aggregate) cancelPending(c DecisionContext, target State) (Aggregate, error) {
	if err := a.guard(c); err != nil {
		return a, err
	}
	if a.state != PendingPayment || a.allocation.AllocationID != "" {
		return a, ErrState
	}
	a.state = target
	if !c.At.Before(*a.snapshot.Data().Payment.ExpiresAt) {
		a.state = CancelledTimeout
	}
	a.version++
	a.decidedAt = c.At
	return a, nil
}

func (a Aggregate) Timeout(c DecisionContext) (Aggregate, error) {
	if err := a.guard(c); err != nil {
		return a, err
	}
	if a.state != PendingPayment || a.allocation.AllocationID != "" {
		return a, ErrState
	}
	if c.At.Before(*a.snapshot.Data().Payment.ExpiresAt) {
		return a, ErrDeadline
	}
	// Operational holds do not stop unpaid timeout or extend a stock lease.
	a.state = CancelledTimeout
	a.version++
	a.decidedAt = c.At
	return a, nil
}

func (a Aggregate) CancelUncommittedFinalization(c DecisionContext, proof ResourceProof) (Aggregate, error) {
	if err := a.guard(c); err != nil {
		return a, err
	}
	if a.state != PaymentFinalizing {
		return a, ErrState
	}
	if a.checkResource(proof) != nil || (proof.State != "EXPIRED" && proof.State != "RELEASED") {
		return a, ErrProof
	}
	a.state, a.reservation = CancelledTimeout, proof
	a.version++
	a.decidedAt = c.At
	return a, nil // Caller preserves allocated money and reconciliation obligation.
}

func (a Aggregate) ReadyEligible() bool {
	return (a.state == Paid || a.state == ConfirmedCOD) && a.reservation.State == "COMMITTED" && a.blocks == 0
}
func (a Aggregate) AuthorizeReady(c DecisionContext) (Aggregate, error) {
	if e := a.guard(c); e != nil {
		return a, e
	}
	if !a.ReadyEligible() {
		return a, ErrBlocked
	}
	a.version++
	a.decidedAt = c.At
	return a, nil
}
func (a Aggregate) ResumeAdmissionHold(c DecisionContext, order, decision string, noTask bool) (Aggregate, error) {
	if e := a.guard(c); e != nil {
		return a, e
	}
	if order != a.ID() || decision == "" || !noTask {
		return a, ErrProof
	}
	if a.state != PendingPayment && a.state != Paid && a.state != ConfirmedCOD {
		return a, ErrState
	}
	if a.blocks & ^HoldBlock != 0 {
		return a, ErrBlocked
	}
	if a.blocks&HoldBlock == 0 {
		return a, nil
	}
	a.blocks &^= HoldBlock
	a.version++
	a.decidedAt = c.At
	return a, nil
}
func (a Aggregate) ConfirmReleased(c DecisionContext, p ResourceProof) (Aggregate, error) {
	if e := a.guard(c); e != nil {
		return a, e
	}
	if !a.state.Terminal() || a.state == Completed || a.allocation.AllocationID != "" || a.checkResource(p) != nil || (p.State != "RELEASED" && p.State != "EXPIRED") {
		return a, ErrProof
	}
	a.reservation = p
	a.version++
	a.decidedAt = c.At
	return a, nil
}

// CompletionEligibility is a pure guard over trusted, freshly queried decisions.
// It does not manufacture a Care barrier from a case projection or emit events.
type CompletionEligibility struct {
	State                                                                            State
	At, DeliveredAt, WindowEndsAt                                                    time.Time
	CareOrderID, OrderID, CareDecisionID                                             string
	CareAllowed                                                                      bool
	SettlementVerified                                                               bool
	BlockingHold, ActiveCase, CancellationPending, UnresolvedRefund, UnresolvedStock bool
}

func (e CompletionEligibility) Validate() error {
	if e.State != Delivered {
		return ErrState
	}
	if !validID(e.OrderID) || e.CareOrderID != e.OrderID || e.CareDecisionID == "" || !e.CareAllowed || !e.SettlementVerified {
		return ErrProof
	}
	if e.DeliveredAt.IsZero() || e.WindowEndsAt.Before(e.DeliveredAt) || e.At.IsZero() || e.At.Before(e.WindowEndsAt) {
		return ErrDeadline
	}
	if e.BlockingHold || e.ActiveCase || e.CancellationPending || e.UnresolvedRefund || e.UnresolvedStock {
		return ErrBlocked
	}
	return nil
}
