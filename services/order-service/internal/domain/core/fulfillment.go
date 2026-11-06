package core

import "time"

type FulfillmentRecord struct {
	TaskID             string    `json:"task_id"`
	Generation         int64     `json:"generation"`
	Seal               string    `json:"seal_code"`
	ShipmentID         string    `json:"shipment_id"`
	AcceptedAt         time.Time `json:"accepted_at"`
	SealedAt           time.Time `json:"sealed_at"`
	ShipmentCreatedAt  time.Time `json:"shipment_created_at"`
	DispatchedAt       time.Time `json:"dispatched_at"`
	DeliveredAt        time.Time `json:"delivered_at"`
	LastShippingAt     time.Time `json:"last_shipping_at"`
	CompletionDecision string    `json:"completion_decision"`
	CompletedAt        time.Time `json:"completed_at"`
}

func (f FulfillmentRecord) validate(state State, placed, decided time.Time) error {
	switch state {
	case Processing, Packed, Shipped, Delivered, DeliveryFailed, Completed:
	default:
		if f != (FulfillmentRecord{}) {
			return ErrProof
		}
		return nil
	}
	if !validID(f.TaskID) || f.Generation < 1 || f.AcceptedAt.IsZero() || f.AcceptedAt.Before(placed) || f.AcceptedAt.After(decided) {
		return ErrProof
	}
	if state == Completed {
		if f.CompletionDecision == "" || f.CompletedAt.Before(f.DeliveredAt) || f.CompletedAt.After(decided) {
			return ErrProof
		}
	} else if f.CompletionDecision != "" || !f.CompletedAt.IsZero() {
		return ErrProof
	}
	if state == Processing {
		if f.Seal != "" || !f.SealedAt.IsZero() || f.ShipmentID != "" || !f.ShipmentCreatedAt.IsZero() || !f.DispatchedAt.IsZero() || !f.LastShippingAt.IsZero() || !f.DeliveredAt.IsZero() {
			return ErrProof
		}
		return nil
	}
	if f.Seal == "" || f.SealedAt.Before(f.AcceptedAt) || f.SealedAt.After(decided) {
		return ErrProof
	}
	if state == Packed {
		if !f.DispatchedAt.IsZero() || !f.DeliveredAt.IsZero() {
			return ErrProof
		}
		if f.ShipmentID == "" {
			if !f.ShipmentCreatedAt.IsZero() || !f.LastShippingAt.IsZero() {
				return ErrProof
			}
		} else if !validID(f.ShipmentID) || f.ShipmentCreatedAt.Before(f.SealedAt) || !f.LastShippingAt.Equal(f.ShipmentCreatedAt) {
			return ErrProof
		}
		return nil
	}
	if !validID(f.ShipmentID) || f.ShipmentCreatedAt.Before(f.SealedAt) || f.DispatchedAt.Before(f.ShipmentCreatedAt) || f.LastShippingAt.Before(f.DispatchedAt) || f.LastShippingAt.After(decided) {
		return ErrProof
	}
	if state == Delivered || state == Completed {
		if f.DeliveredAt.IsZero() || f.DeliveredAt.Before(f.DispatchedAt) || f.DeliveredAt.After(decided) {
			return ErrProof
		}
	} else if !f.DeliveredAt.IsZero() {
		return ErrProof
	}
	return nil
}

// FulfillmentProof comes from authenticated, version-specific source history.
// Public event payloads do not establish that the remote action happened.
type FulfillmentProof struct {
	OrderID, ResourceID, Kind, Seal string
	Version, Generation             int64
	At                              time.Time
}

func (a Aggregate) ApplyFulfillment(c DecisionContext, p FulfillmentProof) (Aggregate, error) {
	if e := a.guard(c); e != nil {
		return a, e
	}
	if p.OrderID != a.ID() || !validID(p.ResourceID) || p.Version < 1 || p.At.IsZero() || (p.At.Before(a.snapshot.Data().PlacedAt) && p.Kind != "CASE_OPENED" && p.Kind != "CASE_CLOSED") || p.At.After(c.At) {
		return a, ErrProof
	}
	var target State
	switch p.Kind {
	case "SHIPMENT_CREATED":
		if a.state != Packed || p.At.Before(a.fulfillment.SealedAt) || (a.fulfillment.ShipmentID != "" && a.fulfillment.ShipmentID != p.ResourceID) {
			return a, ErrState
		}
		a.fulfillment.ShipmentID = p.ResourceID
		a.fulfillment.ShipmentCreatedAt = p.At
		a.fulfillment.LastShippingAt = p.At
		a.version++
		a.decidedAt = c.At
		return a, nil
	case "CASE_OPENED", "CASE_CLOSED":
		if a.state.Terminal() {
			return a, ErrState
		}
		a.version++
		a.decidedAt = c.At
		return a, nil
	case "PACKING_ACCEPTED":
		if !a.ReadyEligible() || p.Generation < 1 {
			return a, ErrBlocked
		}
		a.fulfillment = FulfillmentRecord{TaskID: p.ResourceID, Generation: p.Generation, AcceptedAt: p.At}
		target = Processing
	case "PACKING_SEALED":
		if a.state != Processing || p.Seal == "" || p.ResourceID != a.fulfillment.TaskID || p.Generation != a.fulfillment.Generation || p.At.Before(a.fulfillment.AcceptedAt) {
			return a, ErrState
		}
		a.fulfillment.Seal = p.Seal
		a.fulfillment.SealedAt = p.At
		target = Packed
	case "SHIPMENT_DISPATCHED":
		if a.state != Packed && a.state != DeliveryFailed {
			return a, ErrState
		}
		if p.ResourceID != a.fulfillment.ShipmentID || p.At.Before(a.fulfillment.LastShippingAt) {
			return a, ErrProof
		}
		a.fulfillment.DispatchedAt = p.At
		a.fulfillment.LastShippingAt = p.At
		target = Shipped
	case "SHIPMENT_DELIVERED":
		if a.state != Shipped {
			return a, ErrState
		}
		if p.ResourceID != a.fulfillment.ShipmentID || p.At.Before(a.fulfillment.LastShippingAt) {
			return a, ErrProof
		}
		a.fulfillment.DeliveredAt = p.At
		a.fulfillment.LastShippingAt = p.At
		target = Delivered
	case "SHIPMENT_FAILED":
		if a.state != Shipped {
			return a, ErrState
		}
		if p.ResourceID != a.fulfillment.ShipmentID || p.At.Before(a.fulfillment.LastShippingAt) {
			return a, ErrProof
		}
		a.fulfillment.LastShippingAt = p.At
		target = DeliveryFailed
	default:
		return a, ErrProof
	}
	if !StateAllowed(a.state, target) {
		return a, ErrState
	}
	a.state = target
	a.version++
	a.decidedAt = c.At
	return a, nil
}
func (a Aggregate) Complete(c DecisionContext, eligibility CompletionEligibility) (Aggregate, error) {
	if e := a.guard(c); e != nil {
		return a, e
	}
	if a.blocks != 0 {
		return a, ErrBlocked
	}
	if a.state != Delivered || eligibility.State != a.state || eligibility.OrderID != a.ID() || !eligibility.At.Equal(c.At) {
		return a, ErrProof
	}
	if !eligibility.DeliveredAt.Equal(a.fulfillment.DeliveredAt) {
		return a, ErrProof
	}
	if e := eligibility.Validate(); e != nil {
		return a, e
	}
	a.state = Completed
	a.fulfillment.CompletionDecision = eligibility.CareDecisionID
	a.fulfillment.CompletedAt = c.At
	a.version++
	a.decidedAt = c.At
	return a, nil
}
