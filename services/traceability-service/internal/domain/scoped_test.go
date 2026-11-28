package domain

import "testing"

func TestWarehousePermissionTuplesCannotCrossMultiply(t *testing.T) {
	const a = "77777777-7777-4777-8777-777777777777"
	const b = "88888888-8888-4888-8888-888888888888"
	p := Principal{ID: "22222222-2222-4222-8222-222222222222", Grants: []PermissionGrant{{ReadInternal, "WAREHOUSE", a}, {Create, "WAREHOUSE", b}, {Approve, "DEPARTMENT", a}}}
	if p.At(a).Can(Create) || p.At(b).Can(ReadInternal) || p.Can(Create) || p.Has(Approve) {
		t.Fatal("grant tuples were flattened")
	}
	p.Permissions = []string{ReadInternal}
	if !p.At(b).Can(Create) || !p.At(b).Can(ReadInternal) || p.At("").Can(Create) {
		t.Fatal("GLOBAL coverage or legacy isolation failed")
	}
}
