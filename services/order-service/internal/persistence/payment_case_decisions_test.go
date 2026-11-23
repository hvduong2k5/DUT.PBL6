package persistence

import (
	"errors"
	"github.com/google/uuid"
	"testing"
)

// F-55: the local case authority must support the full two-person flow —
// assign, decide by a different principal, resolve by the assignee — and
// refuse every shortcut around the maker-checker rule.
func TestLocalCaseAuthorityDecisionFlow(t *testing.T) {
	h, p, _, tr := paymentFixture(t)
	tr.In--
	if e := p.Ingest(h.ctx, tr); e != nil {
		t.Fatal(e)
	}
	resolver := refundActor(p)
	resolver.Permissions["payment.reconciliation.manage"] = true
	resolver.Permissions["payment.reconciliation.read"] = true
	decider := resolver
	decider.ID = uuid.NewString()
	decider.Permissions = map[string]bool{"payment.case.decide": true}
	page, e := p.CaseQueue(h.ctx, resolver, "", "", 1)
	if e != nil || len(page.Items) != 1 {
		t.Fatal(page, e)
	}
	v := page.Items[0]
	authority := p.LocalCaseAuthority()
	evidence := []string{uuid.NewString()}
	if _, e = p.CreateCaseDecision(h.ctx, decider, v.ID, "EXTERNAL_FUNDS_RESOLVED", "test", evidence); !errors.Is(e, ErrConflict) {
		t.Fatal("decision before assignment", e)
	}
	v, e = p.CaseCommand(h.ctx, resolver, authority, v.ID, "assign", "assign-case-10001", v.Version, "test", resolver.ID, "", "", nil)
	if e != nil {
		t.Fatal(e)
	}
	resolver.Permissions["payment.case.decide"] = true
	if _, e = p.CreateCaseDecision(h.ctx, resolver, v.ID, "EXTERNAL_FUNDS_RESOLVED", "test", evidence); !errors.Is(e, ErrForbidden) {
		t.Fatal("assignee decided own case", e)
	}
	if _, e = p.CreateCaseDecision(h.ctx, decider, v.ID, "REFUNDED", "test", evidence); !errors.Is(e, ErrConflict) {
		t.Fatal("unpaid refund obligation decided", e)
	}
	decision, e := p.CreateCaseDecision(h.ctx, decider, v.ID, "EXTERNAL_FUNDS_RESOLVED", "test", evidence)
	if e != nil {
		t.Fatal(e)
	}
	if decision.Case != v.ID || decision.Connection != p.Connection || decision.Receiver != p.Receiver || !decision.Allowed || decision.Policy == "" {
		t.Fatal("decision shape", decision)
	}
	listed, e := p.ListCaseDecisions(h.ctx, decider, v.ID)
	if e != nil || len(listed) != 1 || listed[0].Reference != decision.Reference {
		t.Fatal("decision list", listed, e)
	}
	v, e = p.CaseCommand(h.ctx, resolver, authority, v.ID, "assign", "assign-case-10002", v.Version, "test", decider.ID, "", "", nil)
	if e != nil {
		t.Fatal(e)
	}
	loaded, e := authority.CaseDecision(h.ctx, decision.Reference)
	if e != nil || loaded.Allowed {
		t.Fatal("decision usable by its maker", loaded, e)
	}
	v, e = p.CaseCommand(h.ctx, resolver, authority, v.ID, "assign", "assign-case-10003", v.Version, "test", resolver.ID, "", "", nil)
	if e != nil {
		t.Fatal(e)
	}
	resolved, e := p.CaseCommand(h.ctx, resolver, authority, v.ID, "resolve", "resolve-case-10001", v.Version, "test", "", decision.Reference, "EXTERNAL_FUNDS_RESOLVED", evidence)
	if e != nil || resolved.Status != "RESOLVED" {
		t.Fatal(resolved, e)
	}
	if _, e = p.CreateCaseDecision(h.ctx, decider, v.ID, "EXTERNAL_FUNDS_RESOLVED", "test", evidence); !errors.Is(e, ErrConflict) {
		t.Fatal("decision on resolved case", e)
	}
	if _, e = authority.CaseDecision(h.ctx, uuid.NewString()); !errors.Is(e, ErrCapability) {
		t.Fatal("unknown decision reference", e)
	}
}
