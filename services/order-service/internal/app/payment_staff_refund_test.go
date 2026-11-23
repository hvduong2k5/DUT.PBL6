package app

import (
	"github.com/omamx/order-service/internal/domain"
	"testing"
)

// F-42 slice 2: the refund tiers join the write tier — approve, execute
// and verify map only for GLOBAL grants on a token whose auth_time is
// fresh, and no tier implies another (duty separation is per person).
func TestPaymentStaffPermissionsRefundTiers(t *testing.T) {
	grants := func(codes ...string) []domain.PermissionGrant {
		out := []domain.PermissionGrant{}
		for _, code := range codes {
			out = append(out, domain.PermissionGrant{PermissionCode: code, ScopeType: "GLOBAL"})
		}
		return out
	}
	approve := paymentStaffPermissions(grants("PAYMENT_REFUND_APPROVE"), true)
	if !approve["payment.refund.approve"] || !approve["payment.refund.accept"] {
		t.Fatal("fresh PAYMENT_REFUND_APPROVE must map to approve and accept")
	}
	if approve["payment.refund.execute"] || approve["payment.refund.verify"] {
		t.Fatal("approve must not imply execute or verify")
	}
	if stale := paymentStaffPermissions(grants("PAYMENT_REFUND_APPROVE"), false); stale["payment.refund.approve"] || stale["payment.refund.accept"] {
		t.Fatal("stale auth_time must not map the approve tier")
	}
	scoped := paymentStaffPermissions([]domain.PermissionGrant{{PermissionCode: "PAYMENT_REFUND_APPROVE", ScopeType: "WAREHOUSE", ScopeID: "WH-1"}}, true)
	if scoped["payment.refund.approve"] {
		t.Fatal("non-GLOBAL approve grant must not map")
	}
	execute := paymentStaffPermissions(grants("PAYMENT_REFUND_EXECUTE"), true)
	if !execute["payment.refund.execute"] || !execute["payment.evidence.write"] {
		t.Fatal("fresh PAYMENT_REFUND_EXECUTE must map to execute and evidence write")
	}
	if execute["payment.refund.verify"] || execute["payment.evidence.issue"] {
		t.Fatal("execute must not imply verify or evidence issue")
	}
	verify := paymentStaffPermissions(grants("PAYMENT_REFUND_VERIFY"), true)
	if !verify["payment.refund.verify"] || !verify["payment.evidence.issue"] {
		t.Fatal("fresh PAYMENT_REFUND_VERIFY must map to verify and evidence issue")
	}
	if verify["payment.refund.execute"] {
		t.Fatal("verify must not imply execute")
	}
}
