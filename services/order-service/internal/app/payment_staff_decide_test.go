package app

import (
	"github.com/omamx/order-service/internal/domain"
	"testing"
)

// F-55: PAYMENT_CASE_DECIDE joins the write tier — it maps to the internal
// payment.case.decide code only for GLOBAL grants on a token whose
// auth_time is fresh, and it never implies the manage tier.
func TestPaymentStaffPermissionsCaseDecide(t *testing.T) {
	grants := func(codes ...string) []domain.PermissionGrant {
		out := []domain.PermissionGrant{}
		for _, code := range codes {
			out = append(out, domain.PermissionGrant{PermissionCode: code, ScopeType: "GLOBAL"})
		}
		return out
	}
	fresh := paymentStaffPermissions(grants("PAYMENT_CASE_DECIDE"), true)
	if !fresh["payment.case.decide"] {
		t.Fatal("fresh PAYMENT_CASE_DECIDE must map to payment.case.decide")
	}
	if fresh["payment.reconciliation.manage"] {
		t.Fatal("decide must not imply manage")
	}
	stale := paymentStaffPermissions(grants("PAYMENT_CASE_DECIDE"), false)
	if stale["payment.case.decide"] {
		t.Fatal("stale auth_time must not map the decide tier")
	}
	scoped := paymentStaffPermissions([]domain.PermissionGrant{{PermissionCode: "PAYMENT_CASE_DECIDE", ScopeType: "WAREHOUSE", ScopeID: "WH-1"}}, true)
	if scoped["payment.case.decide"] {
		t.Fatal("non-GLOBAL decide grant must not map")
	}
	both := paymentStaffPermissions(grants("PAYMENT_CASE_RESOLVE", "PAYMENT_CASE_DECIDE"), true)
	if !both["payment.reconciliation.manage"] || !both["payment.case.decide"] {
		t.Fatal("resolve and decide tiers must coexist for a two-hat test actor")
	}
}
