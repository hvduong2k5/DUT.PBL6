package domain

import (
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PurchaseOrderStatus string

const (
	PurchaseOrderDraft             PurchaseOrderStatus = "DRAFT"
	PurchaseOrderPendingApproval   PurchaseOrderStatus = "PENDING_APPROVAL"
	PurchaseOrderApproved          PurchaseOrderStatus = "APPROVED"
	PurchaseOrderRejected          PurchaseOrderStatus = "REJECTED"
	PurchaseOrderSent              PurchaseOrderStatus = "SENT"
	PurchaseOrderAcknowledged      PurchaseOrderStatus = "ACKNOWLEDGED"
	PurchaseOrderChangeRequested   PurchaseOrderStatus = "CHANGE_REQUESTED"
	PurchaseOrderSupplierDeclined  PurchaseOrderStatus = "SUPPLIER_DECLINED"
	PurchaseOrderPartiallyReceived PurchaseOrderStatus = "PARTIALLY_RECEIVED"
	PurchaseOrderReceived          PurchaseOrderStatus = "RECEIVED"
	PurchaseOrderCancelled         PurchaseOrderStatus = "CANCELLED"
	PurchaseOrderClosed            PurchaseOrderStatus = "CLOSED"
)

type SupplierPOResponse string

const (
	SupplierPOAcknowledged    SupplierPOResponse = "ACKNOWLEDGED"
	SupplierPOChangeRequested SupplierPOResponse = "CHANGE_REQUESTED"
	SupplierPODeclined        SupplierPOResponse = "DECLINED"
)

type PurchaseOrderLine struct {
	ID               uuid.UUID `json:"id"`
	LineNumber       int       `json:"line_number"`
	MaterialID       uuid.UUID `json:"material_id"`
	MaterialCode     string    `json:"material_code"`
	MaterialName     string    `json:"material_name"`
	Quantity         string    `json:"quantity"`
	ReceivedQuantity string    `json:"received_quantity"`
	AcceptedQuantity string    `json:"accepted_quantity"`
	Unit             string    `json:"unit"`
	UnitPriceMinor   int64     `json:"unit_price_minor"`
	LineAmountMinor  int64     `json:"line_amount_minor"`
}

type PurchaseOrder struct {
	ID                       uuid.UUID           `json:"id"`
	Number                   string              `json:"number"`
	Revision                 int                 `json:"revision"`
	SupplierID               uuid.UUID           `json:"supplier_id"`
	SupplierName             string              `json:"supplier_name"`
	Status                   PurchaseOrderStatus `json:"status"`
	Currency                 string              `json:"currency"`
	Lines                    []PurchaseOrderLine `json:"lines"`
	TotalAmountMinor         int64               `json:"total_amount_minor"`
	Notes                    string              `json:"notes,omitempty"`
	ExpectedDeliveryDate     *time.Time          `json:"expected_delivery_date,omitempty"`
	CreatedBy                string              `json:"created_by"`
	SubmittedBy              string              `json:"submitted_by,omitempty"`
	SubmittedAt              *time.Time          `json:"submitted_at,omitempty"`
	ApprovedBy               string              `json:"approved_by,omitempty"`
	ApprovedAt               *time.Time          `json:"approved_at,omitempty"`
	RejectedBy               string              `json:"rejected_by,omitempty"`
	RejectedAt               *time.Time          `json:"rejected_at,omitempty"`
	RejectionReason          string              `json:"rejection_reason,omitempty"`
	SentBy                   string              `json:"sent_by,omitempty"`
	SentAt                   *time.Time          `json:"sent_at,omitempty"`
	SupplierResponseNote     string              `json:"supplier_response_note,omitempty"`
	SupplierRespondedBy      string              `json:"supplier_responded_by,omitempty"`
	SupplierRespondedAt      *time.Time          `json:"supplier_responded_at,omitempty"`
	CancelledBy              string              `json:"cancelled_by,omitempty"`
	CancelledAt              *time.Time          `json:"cancelled_at,omitempty"`
	CancellationReason       string              `json:"cancellation_reason,omitempty"`
	ClosedBy                 string              `json:"closed_by,omitempty"`
	ClosedAt                 *time.Time          `json:"closed_at,omitempty"`
	CloseReason              string              `json:"close_reason,omitempty"`
	SourceSupplyPlanID       *uuid.UUID          `json:"supply_plan_id,omitempty"`
	SourceSupplyPlanRevision *int                `json:"supply_plan_revision,omitempty"`
	CreatedAt                time.Time           `json:"created_at"`
	UpdatedAt                time.Time           `json:"updated_at"`
	Version                  int64               `json:"version"`
}

type PurchaseOrderRevision struct {
	Revision     int           `json:"revision"`
	Snapshot     PurchaseOrder `json:"snapshot"`
	ChangedBy    string        `json:"changed_by"`
	ChangeReason string        `json:"change_reason"`
	CreatedAt    time.Time     `json:"created_at"`
}

type PurchaseOrderLineInput struct {
	MaterialID     uuid.UUID
	MaterialCode   string
	MaterialName   string
	Quantity       string
	Unit           string
	UnitPriceMinor int64
}

var purchaseQuantityPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,6})?$`)

func NewPurchaseOrder(number string, supplierID uuid.UUID, supplierName, currency, notes, createdBy string, expectedDeliveryDate *time.Time, lines []PurchaseOrderLineInput) (PurchaseOrder, error) {
	number = strings.TrimSpace(number)
	supplierName = strings.TrimSpace(supplierName)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	createdBy = strings.TrimSpace(createdBy)
	if number == "" || supplierID == uuid.Nil || supplierName == "" || len(currency) != 3 || createdBy == "" || len(lines) == 0 {
		return PurchaseOrder{}, ErrInvalidArgument
	}
	if expectedDeliveryDate != nil {
		date := expectedDeliveryDate.UTC()
		if date.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
			return PurchaseOrder{}, ErrInvalidArgument
		}
		expectedDeliveryDate = &date
	}

	now := time.Now().UTC()
	order := PurchaseOrder{
		ID: uuid.New(), Number: number, SupplierID: supplierID, SupplierName: supplierName,
		Status: PurchaseOrderDraft, Currency: currency, Notes: strings.TrimSpace(notes),
		Revision:  1,
		CreatedBy: createdBy, ExpectedDeliveryDate: expectedDeliveryDate,
		CreatedAt: now, UpdatedAt: now, Version: 1,
		Lines: make([]PurchaseOrderLine, 0, len(lines)),
	}
	seenMaterials := make(map[uuid.UUID]struct{}, len(lines))
	for i, input := range lines {
		line, err := newPurchaseOrderLine(i+1, input)
		if err != nil {
			return PurchaseOrder{}, err
		}
		if _, exists := seenMaterials[input.MaterialID]; exists {
			return PurchaseOrder{}, ErrInvalidArgument
		}
		seenMaterials[input.MaterialID] = struct{}{}
		if order.TotalAmountMinor > int64(^uint64(0)>>1)-line.LineAmountMinor {
			return PurchaseOrder{}, ErrInvalidArgument
		}
		order.TotalAmountMinor += line.LineAmountMinor
		order.Lines = append(order.Lines, line)
	}
	return order, nil
}

func newPurchaseOrderLine(lineNumber int, input PurchaseOrderLineInput) (PurchaseOrderLine, error) {
	quantityText := strings.TrimSpace(input.Quantity)
	unit := strings.ToUpper(strings.TrimSpace(input.Unit))
	quantity, ok := new(big.Rat).SetString(quantityText)
	if input.MaterialID == uuid.Nil || strings.TrimSpace(input.MaterialCode) == "" || strings.TrimSpace(input.MaterialName) == "" ||
		!purchaseQuantityPattern.MatchString(quantityText) || !ok || quantity.Sign() <= 0 || unit == "" || input.UnitPriceMinor <= 0 {
		return PurchaseOrderLine{}, ErrInvalidArgument
	}
	lineAmount, ok := multiplyAndRoundHalfUp(input.UnitPriceMinor, quantity)
	if !ok || lineAmount <= 0 {
		return PurchaseOrderLine{}, ErrInvalidArgument
	}
	return PurchaseOrderLine{
		ID: uuid.New(), LineNumber: lineNumber, MaterialID: input.MaterialID,
		MaterialCode: strings.TrimSpace(input.MaterialCode), MaterialName: strings.TrimSpace(input.MaterialName),
		Quantity: quantityText, ReceivedQuantity: "0", AcceptedQuantity: "0", Unit: unit, UnitPriceMinor: input.UnitPriceMinor, LineAmountMinor: lineAmount,
	}, nil
}

func multiplyAndRoundHalfUp(unitPriceMinor int64, quantity *big.Rat) (int64, bool) {
	value := new(big.Rat).Mul(new(big.Rat).SetInt64(unitPriceMinor), quantity)
	numerator := new(big.Int).Set(value.Num())
	denominator := new(big.Int).Set(value.Denom())
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return 0, false
	}
	return quotient.Int64(), true
}

func (o *PurchaseOrder) ReplaceDraft(updated PurchaseOrder) error {
	if o.Status != PurchaseOrderDraft {
		return ErrInvalidState
	}
	updated.ID = o.ID
	updated.Number = o.Number
	updated.Status = o.Status
	updated.CreatedBy = o.CreatedBy
	updated.CreatedAt = o.CreatedAt
	updated.Version = o.Version + 1
	updated.UpdatedAt = time.Now().UTC()
	*o = updated
	return nil
}

func (o *PurchaseOrder) Submit(actor string) error {
	if o.Status != PurchaseOrderDraft || strings.TrimSpace(actor) == "" {
		return ErrInvalidState
	}
	o.Status = PurchaseOrderPendingApproval
	o.SubmittedBy = strings.TrimSpace(actor)
	now := time.Now().UTC()
	o.SubmittedAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) Approve(actor string) error {
	actor = strings.TrimSpace(actor)
	if o.Status != PurchaseOrderPendingApproval || actor == "" {
		return ErrInvalidState
	}
	if actor == o.CreatedBy {
		return ErrConflict
	}
	o.Status = PurchaseOrderApproved
	o.ApprovedBy = actor
	now := time.Now().UTC()
	o.ApprovedAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) Reject(actor, reason string) error {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if o.Status != PurchaseOrderPendingApproval || actor == "" || reason == "" {
		return ErrInvalidState
	}
	if actor == o.CreatedBy {
		return ErrConflict
	}
	o.Status = PurchaseOrderRejected
	o.RejectedBy = actor
	o.RejectionReason = reason
	now := time.Now().UTC()
	o.RejectedAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) MarkSent(actor string) error {
	if o.Status == PurchaseOrderSent {
		return nil
	}
	if o.Status != PurchaseOrderApproved || strings.TrimSpace(actor) == "" {
		return ErrInvalidState
	}
	o.Status = PurchaseOrderSent
	o.SentBy = strings.TrimSpace(actor)
	now := time.Now().UTC()
	o.SentAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) RecordSupplierResponse(response SupplierPOResponse, actor, note string) error {
	actor, note = strings.TrimSpace(actor), strings.TrimSpace(note)
	if o.Status != PurchaseOrderSent || actor == "" {
		return ErrInvalidState
	}
	if response != SupplierPOAcknowledged && response != SupplierPOChangeRequested && response != SupplierPODeclined {
		return ErrInvalidArgument
	}
	if (response == SupplierPOChangeRequested || response == SupplierPODeclined) && note == "" {
		return ErrInvalidArgument
	}
	switch response {
	case SupplierPOAcknowledged:
		o.Status = PurchaseOrderAcknowledged
	case SupplierPOChangeRequested:
		o.Status = PurchaseOrderChangeRequested
	case SupplierPODeclined:
		o.Status = PurchaseOrderSupplierDeclined
	}
	o.SupplierRespondedBy, o.SupplierResponseNote = actor, note
	now := time.Now().UTC()
	o.SupplierRespondedAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) ReviseAsDraft(updated PurchaseOrder, actor, reason string) error {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if o.Status != PurchaseOrderChangeRequested || actor == "" || reason == "" || len(updated.Lines) == 0 {
		return ErrInvalidState
	}
	if updated.SupplierID != o.SupplierID {
		return ErrInvalidArgument
	}
	updated.ID, updated.Number = o.ID, o.Number
	updated.Revision = o.Revision + 1
	updated.Status, updated.CreatedBy = PurchaseOrderDraft, actor
	updated.CreatedAt = o.CreatedAt
	updated.Version = o.Version + 1
	updated.UpdatedAt = time.Now().UTC()
	*o = updated
	return nil
}

func (o *PurchaseOrder) Cancel(actor, reason string) error {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return ErrInvalidArgument
	}
	switch o.Status {
	case PurchaseOrderDraft, PurchaseOrderPendingApproval, PurchaseOrderApproved, PurchaseOrderRejected:
	default:
		return ErrInvalidState
	}
	o.Status = PurchaseOrderCancelled
	o.CancelledBy = actor
	o.CancellationReason = reason
	now := time.Now().UTC()
	o.CancelledAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) CloseRemaining(actor, reason string) error {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return ErrInvalidArgument
	}
	if o.Status != PurchaseOrderSent && o.Status != PurchaseOrderAcknowledged && o.Status != PurchaseOrderPartiallyReceived {
		return ErrInvalidState
	}
	o.Status = PurchaseOrderClosed
	o.ClosedBy, o.CloseReason = actor, reason
	now := time.Now().UTC()
	o.ClosedAt = &now
	o.touch()
	return nil
}

func (o *PurchaseOrder) RecordReceived(lineID uuid.UUID, quantity string) error {
	return o.RecordReceiptQuantities(lineID, quantity, "0")
}

func (o *PurchaseOrder) RecordReceiptQuantities(lineID uuid.UUID, acceptedQuantity, rejectedQuantity string) error {
	if o.Status != PurchaseOrderSent && o.Status != PurchaseOrderAcknowledged && o.Status != PurchaseOrderPartiallyReceived {
		return ErrInvalidState
	}
	acceptedText, rejectedText := strings.TrimSpace(acceptedQuantity), strings.TrimSpace(rejectedQuantity)
	accepted, acceptedOK := new(big.Rat).SetString(acceptedText)
	rejected, rejectedOK := new(big.Rat).SetString(rejectedText)
	if !acceptedOK || !rejectedOK || !purchaseQuantityPattern.MatchString(acceptedText) || !purchaseQuantityPattern.MatchString(rejectedText) || accepted.Sign() < 0 || rejected.Sign() < 0 || accepted.Sign()+rejected.Sign() <= 0 {
		return ErrInvalidArgument
	}
	for i := range o.Lines {
		if o.Lines[i].ID != lineID {
			continue
		}
		ordered, _ := new(big.Rat).SetString(o.Lines[i].Quantity)
		acceptedTotal, _ := new(big.Rat).SetString(o.Lines[i].AcceptedQuantity)
		if acceptedTotal == nil {
			acceptedTotal = new(big.Rat)
		}
		acceptedTotal.Add(acceptedTotal, accepted)
		if acceptedTotal.Cmp(ordered) > 0 {
			return ErrConflict
		}
		receivedTotal, _ := new(big.Rat).SetString(o.Lines[i].ReceivedQuantity)
		if receivedTotal == nil {
			receivedTotal = new(big.Rat)
		}
		receivedTotal.Add(receivedTotal, new(big.Rat).Add(new(big.Rat).Set(accepted), rejected))
		o.Lines[i].ReceivedQuantity = receivedTotal.FloatString(6)
		o.Lines[i].AcceptedQuantity = acceptedTotal.FloatString(6)
		allReceived := true
		for _, line := range o.Lines {
			orderedLine, _ := new(big.Rat).SetString(line.Quantity)
			acceptedLine, _ := new(big.Rat).SetString(line.AcceptedQuantity)
			if acceptedLine == nil || acceptedLine.Cmp(orderedLine) < 0 {
				allReceived = false
				break
			}
		}
		if allReceived {
			o.Status = PurchaseOrderReceived
		} else {
			o.Status = PurchaseOrderPartiallyReceived
		}
		o.touch()
		return nil
	}
	return ErrNotFound
}

func (o *PurchaseOrder) touch() {
	o.UpdatedAt = time.Now().UTC()
	o.Version++
}
