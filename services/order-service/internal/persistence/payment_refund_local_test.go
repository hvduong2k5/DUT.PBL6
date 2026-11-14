package persistence

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/sepay"
	"strings"
	"testing"
	"time"
)

// F-42 slice 2: the local refund authority carries a manual refund from
// approval to verified settlement with three different people — approver,
// executor, issuer/verifier — and every self-dealing shortcut fails.

func f56Actor(p *Payment, codes ...string) PaymentActor {
	perms := map[string]bool{}
	for _, code := range codes {
		perms[code] = true
	}
	return PaymentActor{ID: uuid.NewString(), Connection: p.Connection, Receivers: []string{p.Receiver}, MFA: time.Now(), Permissions: perms}
}

func f56Receipt(t *testing.T, h *harness, p *Payment, tr sepay.Transaction) string {
	t.Helper()
	if e := p.Ingest(h.ctx, tr); e != nil {
		t.Fatal(e)
	}
	if _, e := p.WorkFinalize(h.ctx, inventoryProof{}); e != nil {
		t.Fatal(e)
	}
	var receipt string
	if e := h.db.QueryRow(h.ctx, "SELECT id FROM receipts WHERE transaction_id=$1", tr.ID).Scan(&receipt); e != nil {
		t.Fatal(e)
	}
	return receipt
}

func TestLocalRefundAuthorityFullFlow(t *testing.T) {
	h, p, _, tr := paymentFixture(t)
	receipt := f56Receipt(t, h, p, tr)
	approver := f56Actor(p, "payment.refund.approve", "payment.refund.accept", "payment.refund.read")
	acceptor := f56Actor(p, "payment.refund.accept", "payment.refund.approve", "payment.refund.read")
	executor := f56Actor(p, "payment.refund.execute", "payment.evidence.write", "payment.refund.read")
	issuer := f56Actor(p, "payment.evidence.issue", "payment.refund.verify", "payment.refund.read")

	// A pending approval counts against the receipt budget; withdrawing
	// it frees the amount again. Only the approver may withdraw.
	small, e := p.CreateRefundApproval(h.ctx, approver, receipt, 1, "000012345678", "VCB", "LOCAL TEST", "probe")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.CreateRefundApproval(h.ctx, approver, receipt, tr.In, "000012345678", "VCB", "LOCAL TEST", "full"); !errors.Is(e, ErrConflict) {
		t.Fatal("pending approval must count against the budget", e)
	}
	if _, e = p.WithdrawRefundApproval(h.ctx, acceptor, small.ID, "WD-PROBE-0001", "not mine"); !errors.Is(e, ErrForbidden) {
		t.Fatal("non-approver withdrew", e)
	}
	withdrawn, e := p.WithdrawRefundApproval(h.ctx, approver, small.ID, "WD-PROBE-0001", "issued in error")
	if e != nil || withdrawn.State != "WITHDRAWN" || withdrawn.WithdrawalReference != "WD-PROBE-0001" {
		t.Fatal(withdrawn, e)
	}
	if _, e = p.AcceptRefundFromApproval(h.ctx, acceptor, small.ID, "f56-accept-withdrawn"); !errors.Is(e, ErrInput) {
		t.Fatal("withdrawn approval accepted", e)
	}

	// Full approval: the approver cannot accept their own approval, the
	// acceptor can.
	full, e := p.CreateRefundApproval(h.ctx, approver, receipt, tr.In, "000012345678", "VCB", "LOCAL TEST", "refund the late receipt in full")
	if e != nil {
		t.Fatal(e)
	}
	if full.Masked != "****5678" || full.State != "APPROVED" {
		t.Fatal(full)
	}
	stored, e := p.LocalRefundApproval().RefundApproval(h.ctx, full.ID)
	if e != nil || stored.Approver != approver.ID || stored.Amount != tr.In || stored.Policy != LocalRefundPolicyVersion {
		t.Fatal(stored, e)
	}
	if _, e = p.AcceptRefundFromApproval(h.ctx, approver, full.ID, "f56-accept-self"); !errors.Is(e, ErrForbidden) {
		t.Fatal("approver accepted own approval", e)
	}
	refund, e := p.AcceptRefundFromApproval(h.ctx, acceptor, full.ID, "f56-accept-full-01")
	if e != nil || refund.Execution != "AWAITING_EXECUTOR" {
		t.Fatal(refund, e)
	}
	reader := p.LocalRefundApproval()
	claimed, e := p.RefundCommand(h.ctx, executor, reader, refund.ID, "claim", "f56-claim-full-001", refund.Version, "taking it")
	if e != nil {
		t.Fatal(e)
	}
	started, e := p.RefundCommand(h.ctx, executor, reader, refund.ID, "start", "f56-start-full-001", claimed.Version, "sending")
	if e != nil || started.AttemptID == nil {
		t.Fatal(started, e)
	}
	instruction, e := p.ManualRefundInstructions(h.ctx, executor, reader, refund.ID)
	if e != nil || instruction.Beneficiary != "000012345678" {
		t.Fatal(instruction, e)
	}

	// Certificates: neither the executor nor the approver may issue.
	var sourceBank string
	if e = h.db.QueryRow(h.ctx, "SELECT bank_account_uuid FROM payment_receiver_profiles WHERE id=$1", p.Receiver).Scan(&sourceBank); e != nil {
		t.Fatal(e)
	}
	certIn := RefundEvidenceInput{Refund: refund.ID, Attempt: *started.AttemptID, Decision: "SENT", Definitive: true, SHA256: strings.Repeat("b", 64), Amount: tr.In, Beneficiary: "000012345678", Bank: "VCB", Holder: "LOCAL TEST", BankIdentity: uuid.NewString(), BankReference: "FT-LOCAL-0001", Occurred: time.Now().UTC().Truncate(time.Microsecond)}
	if _, e = p.CreateRefundEvidenceRecord(h.ctx, executor, certIn); !errors.Is(e, ErrForbidden) {
		t.Fatal("executor issued own certificate", e)
	}
	if _, e = p.CreateRefundEvidenceRecord(h.ctx, approver, certIn); !errors.Is(e, ErrForbidden) {
		t.Fatal("approver issued certificate", e)
	}
	cert, e := p.CreateRefundEvidenceRecord(h.ctx, issuer, certIn)
	if e != nil || cert.SourceBank != sourceBank || cert.Source != "BANK_STATEMENT" {
		t.Fatal(cert, e)
	}
	attached, e := p.AttachRefundEvidenceBatch(h.ctx, executor, p.LocalRefundEvidence(), refund.ID, *started.AttemptID, "f56-attach-full-01", started.Version, "statement attached", []string{cert.Reference})
	if e != nil || attached.Execution != "EVIDENCE_PENDING" {
		t.Fatal(attached, e)
	}

	// Verification: the executor cannot verify even with a verify grant,
	// and the issuer/verifier settles with matching bank evidence.
	selfVerifier := executor
	selfVerifier.Permissions = map[string]bool{"payment.refund.verify": true}
	bene, _ := json.Marshal(map[string]string{"account_number": "000012345678", "bank_code": "VCB", "account_holder": "LOCAL TEST"})
	bank := &RefundBankEvidence{Source: sourceBank, Reference: cert.BankReference, Amount: tr.In, Currency: "VND", BeneficiaryHash: hash(bene), At: cert.Occurred}
	if _, e = p.VerifyRefundEvidenceSet(h.ctx, selfVerifier, p.LocalRefundEvidence(), refund.ID, *started.AttemptID, "SENT", "f56-verify-self-01", attached.Version, "self", []string{cert.Reference}, bank); !errors.Is(e, ErrForbidden) {
		t.Fatal("executor verified own refund", e)
	}
	settled, e := p.VerifyRefundEvidenceSet(h.ctx, issuer, p.LocalRefundEvidence(), refund.ID, *started.AttemptID, "SENT", "f56-verify-full-01", attached.Version, "bank statement matches", []string{cert.Reference}, bank)
	if e != nil || settled.Status != "SUCCEEDED" || settled.Execution != "VERIFIED_SENT" {
		t.Fatal(settled, e)
	}
	if h.count(t, "SELECT count(*) FROM orders WHERE payment_status='REFUNDED'") != 1 || h.count(t, "SELECT count(*) FROM outbox_events WHERE event_type='vn.omama.payment.refund_settled.v1'") != 1 {
		t.Fatal("refund settlement not reflected on the order or the outbox")
	}
}

func TestLocalRefundApprovalWithdrawCancelsObligation(t *testing.T) {
	h, p, _, tr := paymentFixture(t)
	receipt := f56Receipt(t, h, p, tr)
	approver := f56Actor(p, "payment.refund.approve", "payment.refund.accept", "payment.refund.read")
	acceptor := f56Actor(p, "payment.refund.accept", "payment.refund.read")
	approval, e := p.CreateRefundApproval(h.ctx, approver, receipt, 5000, "000012345678", "VCB", "LOCAL TEST", "to be withdrawn")
	if e != nil {
		t.Fatal(e)
	}
	refund, e := p.AcceptRefundFromApproval(h.ctx, acceptor, approval.ID, "f56b-accept-000001")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.WithdrawRefundApproval(h.ctx, approver, approval.ID, "WD-CANCEL-0001", "customer withdrew the request"); e != nil {
		t.Fatal(e)
	}
	cancelled, e := p.RefundCommand(h.ctx, acceptor, p.LocalRefundApproval(), refund.ID, "cancel-obligation", "f56b-cancel-000001", refund.Version, "approval withdrawn", "WD-CANCEL-0001")
	if e != nil || cancelled.Obligation != "CANCELLED" {
		t.Fatal(cancelled, e)
	}
	if h.count(t, "SELECT count(*) FROM refunds WHERE budget_held") != 0 {
		t.Fatal("cancelled obligation still holds the receipt budget")
	}
	if _, e = p.CreateRefundApproval(h.ctx, approver, receipt, tr.In, "000012345678", "VCB", "LOCAL TEST", "budget freed"); e != nil {
		t.Fatal("budget not freed after cancellation", e)
	}
}
