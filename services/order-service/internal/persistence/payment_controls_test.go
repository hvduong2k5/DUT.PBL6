package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

type caseStub struct {
	decision CaseDecision
	allowed  bool
}

func (c caseStub) CaseDecision(context.Context, string) (CaseDecision, error) { return c.decision, nil }
func (c caseStub) CanAssign(context.Context, string, string) (bool, error)    { return c.allowed, nil }
func TestPaymentCaseScopeAssignmentAndResolution(t *testing.T) {
	h, p, _, tr := paymentFixture(t)
	tr.In--
	if e := p.Ingest(h.ctx, tr); e != nil {
		t.Fatal(e)
	}
	actor := refundActor(p)
	actor.Permissions["payment.reconciliation.manage"] = true
	actor.Permissions["payment.reconciliation.read"] = true
	page, e := p.CaseQueue(h.ctx, actor, "", "", 1)
	if e != nil || len(page.Items) != 1 {
		t.Fatal(page, e)
	}
	v := page.Items[0]
	another := actor
	another.ID = uuid.NewString()
	another.Permissions = map[string]bool{"payment.reconciliation.read": true}
	if _, e = p.ReadCase(h.ctx, another, v.ID); !errors.Is(e, ErrForbidden) {
		t.Fatal("unassigned case visible", e)
	}
	authority := caseStub{allowed: true}
	v, e = p.CaseCommand(h.ctx, actor, authority, v.ID, "assign", "assign-case-00001", v.Version, "test", actor.ID, "", "", nil)
	if e != nil {
		t.Fatal(e)
	}
	evidence := []string{uuid.NewString()}
	authority.decision = CaseDecision{Reference: uuid.NewString(), Case: v.ID, Connection: p.Connection, Receiver: p.Receiver, Resolution: "REFUNDED", Allowed: true, Evidence: evidence, Policy: "finance-v1"}
	if _, e = p.CaseCommand(h.ctx, actor, authority, v.ID, "resolve", "resolve-case-00001", v.Version, "test", "", authority.decision.Reference, "REFUNDED", evidence); !errors.Is(e, ErrConflict) {
		t.Fatal("unpaid refund obligation resolved", e)
	}
	authority.decision.Resolution = "EXTERNAL_FUNDS_RESOLVED"
	resolved, e := p.CaseCommand(h.ctx, actor, authority, v.ID, "resolve", "resolve-case-00002", v.Version, "test", "", authority.decision.Reference, authority.decision.Resolution, evidence)
	if e != nil || resolved.Status != "RESOLVED" {
		t.Fatal(resolved, e)
	}
	if h.count(t, "SELECT count(*) FROM orders WHERE operational_hold") != 1 {
		t.Fatal("case resolution removed operational safety hold")
	}
	cursor := p.queueCursor(actor, "CASE", "", v.ID)
	if _, e = p.queueAfter(another, "CASE", "", cursor); e == nil {
		t.Fatal("cross-actor cursor accepted")
	}
	t.Log("PAYMENT_CONTRACT ReconciliationCase " + mustJSON(resolved))
}

type vaultStub struct {
	ticket UploadTicket
	view   EvidenceView
}

func (v vaultStub) CreateUpload(context.Context, string, string, UploadRequest) (UploadTicket, error) {
	return v.ticket, nil
}
func (v vaultStub) Download(context.Context, string, string) (EvidenceView, error) {
	return v.view, nil
}
func TestEvidenceVaultHostAndCurrentAssignment(t *testing.T) {
	h, p, care, v, actor := refundFixture(t)
	v, e := p.RefundCommand(h.ctx, actor, care, v.ID, "claim", "claim-refund-0001", v.Version, "test")
	if e != nil {
		t.Fatal(e)
	}
	v, e = p.RefundCommand(h.ctx, actor, care, v.ID, "start", "start-refund-0001", v.Version, "test")
	if e != nil {
		t.Fatal(e)
	}
	expires := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Microsecond)
	vault := vaultStub{ticket: UploadTicket{ID: uuid.NewString(), URL: "https://evidence.example.test/object", Expires: expires}}
	in := UploadRequest{Type: "REFUND", Resource: v.ID, Media: "application/pdf", Size: 1024, SHA256: strings.Repeat("a", 64), Reason: "test"}
	hosts := []string{"evidence.example.test"}
	bad := vault
	bad.ticket.URL = "https://evidence.example.test.attacker.invalid/object"
	if _, e = p.EvidenceUpload(h.ctx, actor, bad, hosts, "upload-evidence-0001", in); !errors.Is(e, ErrIntegrity) {
		t.Fatal("untrusted upload host", e)
	}
	ticket, e := p.EvidenceUpload(h.ctx, actor, vault, hosts, "upload-evidence-0001", in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.EvidenceUpload(h.ctx, actor, vault, hosts, "upload-evidence-0001", in); e != nil {
		t.Fatal("upload replay", e)
	}
	vault.view = EvidenceView{ID: ticket.ID, Type: in.Type, Resource: in.Resource, SHA256: in.SHA256, Status: "SCANNING", URL: &vault.ticket.URL, Expires: &expires}
	if _, e = p.EvidenceDownload(h.ctx, actor, vault, hosts, ticket.ID); !errors.Is(e, ErrIntegrity) {
		t.Fatal("unscanned file downloadable", e)
	}
	vault.view.Status = "CLEAN"
	if _, e = p.EvidenceDownload(h.ctx, actor, vault, hosts, ticket.ID); e != nil {
		t.Fatal(e)
	}
	actor.Receivers = nil
	if _, e = p.EvidenceDownload(h.ctx, actor, vault, hosts, ticket.ID); !errors.Is(e, ErrForbidden) {
		t.Fatal("revoked scope still downloaded", e)
	}
}
func mustJSON(v any) string { raw, _ := json.Marshal(v); return string(raw) }
func TestPaymentContractExamples(t *testing.T) {
	h, p, a, _ := paymentFixture(t)
	if _, e := h.db.Exec(h.ctx, "UPDATE payment_sync_checkpoints SET circuit='HEALTHY',last_scanned_end=clock_timestamp() WHERE receiver_id=$1 AND stream='FAST'", p.Receiver); e != nil {
		t.Fatal(e)
	}
	status, e := p.Status(h.ctx, a.ID(), ownerScope(a.Owner()))
	if e != nil {
		t.Fatal(e)
	}
	instructions, e := p.Instructions(h.ctx, a.ID(), ownerScope(a.Owner()))
	if e != nil {
		t.Fatal(e)
	}
	t.Log("PAYMENT_CONTRACT PaymentStatus " + mustJSON(status))
	t.Log("PAYMENT_CONTRACT PaymentInstructions " + mustJSON(instructions))
	h, p, care, v, actor := refundFixture(t)
	t.Log("PAYMENT_CONTRACT Refund " + mustJSON(v))
	v, e = p.RefundCommand(h.ctx, actor, care, v.ID, "claim", "claim-refund-0001", v.Version, "test")
	if e != nil {
		t.Fatal(e)
	}
	v, e = p.RefundCommand(h.ctx, actor, care, v.ID, "start", "start-refund-0001", v.Version, "test")
	if e != nil {
		t.Fatal(e)
	}
	instruction, e := p.RefundStartInstruction(h.ctx, actor, care, v.ID)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("PAYMENT_CONTRACT RefundStart " + mustJSON(instruction))
	rows, e := h.db.Query(h.ctx, "SELECT payload_bytes FROM outbox_events WHERE event_type IN ('vn.omama.order.paid.v3','vn.omama.payment.receipt_recorded.v1')")
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			t.Fatal(e)
		}
		t.Log("PAYMENT_EVENT " + string(raw))
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
}
