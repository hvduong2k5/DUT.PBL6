package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ReturnCaseStatus string

const (
	ReturnCaseRequested         ReturnCaseStatus = "REQUESTED"
	ReturnCaseNeedsInformation  ReturnCaseStatus = "NEEDS_INFORMATION"
	ReturnCaseReviewing         ReturnCaseStatus = "REVIEWING"
	ReturnCaseApproved          ReturnCaseStatus = "APPROVED"
	ReturnCasePartiallyApproved ReturnCaseStatus = "PARTIALLY_APPROVED"
	ReturnCaseRejected          ReturnCaseStatus = "REJECTED"
	ReturnCaseInFulfillment     ReturnCaseStatus = "IN_FULFILLMENT"
	ReturnCaseCompleted         ReturnCaseStatus = "COMPLETED"
	ReturnCaseClosed            ReturnCaseStatus = "CLOSED"
)

type ReturnedGoodsStatus string

const (
	ReturnedGoodsNotRequired              ReturnedGoodsStatus = "NOT_REQUIRED"
	ReturnedGoodsAwaitingCustomerShipment ReturnedGoodsStatus = "AWAITING_CUSTOMER_SHIPMENT"
	ReturnedGoodsInTransit                ReturnedGoodsStatus = "IN_TRANSIT"
	ReturnedGoodsReceived                 ReturnedGoodsStatus = "RECEIVED"
	ReturnedGoodsInspected                ReturnedGoodsStatus = "INSPECTED"
)

type RefundStatus string

const (
	RefundNotRequired     RefundStatus = "NOT_REQUIRED"
	RefundPendingApproval RefundStatus = "PENDING_APPROVAL"
	RefundApproved        RefundStatus = "APPROVED"
	RefundRequested       RefundStatus = "REQUESTED"
	RefundProcessing      RefundStatus = "PROCESSING"
	RefundSucceeded       RefundStatus = "SUCCEEDED"
	RefundFailed          RefundStatus = "FAILED"
)

type ReturnLineSnapshot struct {
	ID                                uuid.UUID
	OrderLineID                       string
	PurchasedQuantity                 int
	PreviouslyCommittedReturnQuantity int
	RequestedQuantity                 int
	RefundableCeilingMinor            int64
	Currency                          string
}

type ReturnDecisionLine struct {
	ReturnCaseLineID    uuid.UUID
	ApprovedQuantity    int
	ApprovedAmountMinor int64
}

func (s ReturnCaseStatus) Valid() bool {
	switch s {
	case ReturnCaseRequested, ReturnCaseNeedsInformation, ReturnCaseReviewing,
		ReturnCaseApproved, ReturnCasePartiallyApproved, ReturnCaseRejected,
		ReturnCaseInFulfillment, ReturnCaseCompleted, ReturnCaseClosed:
		return true
	default:
		return false
	}
}

func (s ReturnCaseStatus) CanTransitionTo(target ReturnCaseStatus) bool {
	switch s {
	case ReturnCaseRequested:
		return target == ReturnCaseNeedsInformation || target == ReturnCaseReviewing
	case ReturnCaseNeedsInformation:
		return target == ReturnCaseReviewing
	case ReturnCaseReviewing:
		return target == ReturnCaseNeedsInformation || target == ReturnCaseApproved ||
			target == ReturnCasePartiallyApproved || target == ReturnCaseRejected
	case ReturnCaseApproved, ReturnCasePartiallyApproved:
		return target == ReturnCaseInFulfillment
	case ReturnCaseInFulfillment:
		return target == ReturnCaseCompleted
	case ReturnCaseRejected:
		return target == ReturnCaseReviewing || target == ReturnCaseClosed
	case ReturnCaseCompleted:
		return target == ReturnCaseClosed
	default:
		return false
	}
}

func (s ReturnedGoodsStatus) CanTransitionTo(target ReturnedGoodsStatus) bool {
	switch s {
	case ReturnedGoodsAwaitingCustomerShipment:
		return target == ReturnedGoodsInTransit
	case ReturnedGoodsInTransit:
		return target == ReturnedGoodsReceived
	case ReturnedGoodsReceived:
		return target == ReturnedGoodsInspected
	default:
		return false
	}
}

func (s RefundStatus) CanTransitionTo(target RefundStatus) bool {
	switch s {
	case RefundPendingApproval:
		return target == RefundApproved
	case RefundApproved:
		return target == RefundRequested
	case RefundRequested:
		return target == RefundProcessing
	case RefundProcessing:
		return target == RefundSucceeded || target == RefundFailed
	case RefundFailed:
		return target == RefundRequested
	default:
		return false
	}
}

func ValidateReturnLine(line ReturnLineSnapshot) error {
	if line.ID == uuid.Nil || strings.TrimSpace(line.OrderLineID) == "" {
		return fmt.Errorf("%w: immutable order line identity is required", ErrInvalidArgument)
	}
	if line.PurchasedQuantity < 1 || line.PreviouslyCommittedReturnQuantity < 0 || line.RequestedQuantity < 1 {
		return fmt.Errorf("%w: return quantities must be positive and internally consistent", ErrInvalidArgument)
	}
	if line.PreviouslyCommittedReturnQuantity+line.RequestedQuantity > line.PurchasedQuantity {
		return fmt.Errorf("%w: requested return quantity exceeds the authoritative remaining quantity", ErrConflict)
	}
	if line.RefundableCeilingMinor < 0 || !validCurrency(line.Currency) {
		return fmt.Errorf("%w: authoritative refundable ceiling and ISO currency are required", ErrInvalidArgument)
	}
	return nil
}

func ValidateReturnDecision(lines []ReturnDecisionLine, approvedTotalMinor, authoritativeCeilingMinor int64, currency string) error {
	if len(lines) == 0 {
		return fmt.Errorf("%w: decision must contain line-level scope", ErrInvalidArgument)
	}
	if approvedTotalMinor < 0 || authoritativeCeilingMinor < 0 || approvedTotalMinor > authoritativeCeilingMinor {
		return fmt.Errorf("%w: approved amount exceeds the authoritative refundable ceiling", ErrConflict)
	}
	if !validCurrency(currency) {
		return fmt.Errorf("%w: ISO currency must contain three uppercase letters", ErrInvalidArgument)
	}
	seen := make(map[uuid.UUID]struct{}, len(lines))
	var lineTotal int64
	for _, line := range lines {
		if line.ReturnCaseLineID == uuid.Nil || line.ApprovedQuantity < 0 || line.ApprovedAmountMinor < 0 {
			return fmt.Errorf("%w: invalid decision line", ErrInvalidArgument)
		}
		if _, duplicate := seen[line.ReturnCaseLineID]; duplicate {
			return fmt.Errorf("%w: duplicate decision line", ErrInvalidArgument)
		}
		seen[line.ReturnCaseLineID] = struct{}{}
		lineTotal += line.ApprovedAmountMinor
	}
	if lineTotal != approvedTotalMinor {
		return fmt.Errorf("%w: line-level approved amounts do not equal the decision total", ErrInvalidArgument)
	}
	return nil
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}
