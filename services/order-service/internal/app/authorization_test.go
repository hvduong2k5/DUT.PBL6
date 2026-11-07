package app

import (
	"github.com/omamx/order-service/internal/domain"
	"testing"
)

func TestOrderPermissionsCannotBorrowAnotherRolesScope(t *testing.T) {
	p := domain.Principal{ID: "staff", Role: "EMPLOYEE", IAM: true, Grants: []domain.PermissionGrant{
		{PermissionCode: "ORDER_VIEW", ScopeType: "WAREHOUSE", ScopeID: "A"},
		{PermissionCode: "ORDER_SENSITIVE_VIEW", ScopeType: "WAREHOUSE", ScopeID: "B"},
		{PermissionCode: "ORDER_HOLD_MANAGE", ScopeType: "WAREHOUSE", ScopeID: "B"},
		{PermissionCode: "ORDER_RECOVERY_EXECUTE", ScopeType: "DEPARTMENT", ScopeID: "A"},
	}}
	a, b := domain.Order{Warehouse: "A"}, domain.Order{Warehouse: "B"}
	if !CanRead(p, a) || CanRead(p, b) {
		t.Fatal("read scope was widened")
	}
	if p.CanSeeRecipient(a) || !p.CanSeeRecipient(b) {
		t.Fatal("sensitive grant was widened")
	}
	if p.AllowsAction("ORDER_HOLD_MANAGE", "A") || !p.AllowsAction("ORDER_HOLD_MANAGE", "B") {
		t.Fatal("mutation borrowed read scope")
	}
	if p.HasPermission("ORDER_RECOVERY_EXECUTE") {
		t.Fatal("unsupported scope accepted")
	}
	p.Grants = []domain.PermissionGrant{{PermissionCode: "ORDER_VIEW", ScopeType: "GLOBAL"}}
	if !CanRead(p, a) || !CanRead(p, b) || p.CanSeeRecipient(a) || p.Manager() {
		t.Fatal("global read elevated unrelated actions")
	}
	p.Grants = nil
	if CanRead(p, a) || p.Manager() {
		t.Fatal("missing grants elevated employee")
	}
	p.Role = "ADMIN"
	if CanRead(p, a) {
		t.Fatal("IAM role name bypass")
	}
}
