package core

import "errors"

var ErrVoucherRejected = errors.New("VOUCHER_HOLD_REJECTED")

type VoucherDecision struct {
	HoldID     string `json:"hold_id"`
	DecisionID string `json:"decision_id"`
}
