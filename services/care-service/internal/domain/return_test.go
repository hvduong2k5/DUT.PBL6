package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReturnCaseLifecycleKeepsDecisionGoodsAndRefundSeparate(t *testing.T) {
	require.True(t, ReturnCaseRequested.CanTransitionTo(ReturnCaseReviewing))
	require.True(t, ReturnCaseReviewing.CanTransitionTo(ReturnCaseApproved))
	require.False(t, ReturnCaseApproved.CanTransitionTo(ReturnCaseCompleted))
	require.True(t, ReturnCaseApproved.CanTransitionTo(ReturnCaseInFulfillment))
	require.True(t, ReturnCaseInFulfillment.CanTransitionTo(ReturnCaseCompleted))
	require.False(t, ReturnCaseCompleted.CanTransitionTo(ReturnCaseReviewing))
}

func TestReturnedGoodsLifecycleDoesNotImplyRefund(t *testing.T) {
	require.True(t, ReturnedGoodsAwaitingCustomerShipment.CanTransitionTo(ReturnedGoodsInTransit))
	require.True(t, ReturnedGoodsInTransit.CanTransitionTo(ReturnedGoodsReceived))
	require.True(t, ReturnedGoodsReceived.CanTransitionTo(ReturnedGoodsInspected))
	require.False(t, ReturnedGoodsInspected.CanTransitionTo(ReturnedGoodsReceived))
}

func TestRefundLifecycleAllowsOnlyControlledRetry(t *testing.T) {
	require.True(t, RefundProcessing.CanTransitionTo(RefundFailed))
	require.True(t, RefundFailed.CanTransitionTo(RefundRequested))
	require.False(t, RefundSucceeded.CanTransitionTo(RefundRequested))
	require.False(t, RefundFailed.CanTransitionTo(RefundSucceeded))
}

func TestValidateReturnLineRejectsQuantityAboveAuthoritativeRemaining(t *testing.T) {
	err := ValidateReturnLine(ReturnLineSnapshot{
		ID: uuid.New(), OrderLineID: "line-1", PurchasedQuantity: 3,
		PreviouslyCommittedReturnQuantity: 2, RequestedQuantity: 2,
		RefundableCeilingMinor: 100_000, Currency: "VND",
	})
	require.ErrorIs(t, err, ErrConflict)
}

func TestValidateReturnDecisionRequiresLineTotalsAndCeiling(t *testing.T) {
	lineID := uuid.New()
	require.NoError(t, ValidateReturnDecision([]ReturnDecisionLine{{
		ReturnCaseLineID: lineID, ApprovedQuantity: 1, ApprovedAmountMinor: 50_000,
	}}, 50_000, 60_000, "VND"))

	err := ValidateReturnDecision([]ReturnDecisionLine{{
		ReturnCaseLineID: lineID, ApprovedQuantity: 1, ApprovedAmountMinor: 50_000,
	}}, 50_000, 40_000, "VND")
	require.ErrorIs(t, err, ErrConflict)

	err = ValidateReturnDecision([]ReturnDecisionLine{
		{ReturnCaseLineID: lineID, ApprovedQuantity: 1, ApprovedAmountMinor: 25_000},
		{ReturnCaseLineID: lineID, ApprovedQuantity: 1, ApprovedAmountMinor: 25_000},
	}, 50_000, 60_000, "VND")
	require.ErrorIs(t, err, ErrInvalidArgument)
}
