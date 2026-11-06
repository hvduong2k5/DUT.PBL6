package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain/core"
	"strings"
	"sync"
	"testing"
	"time"
)

type racingRefundInventory struct {
	p    *Payment
	care approvalStub
}

func (i racingRefundInventory) Finalize(ctx context.Context, a core.Aggregate, key string) (core.ResourceProof, error) {
	if _, e := i.p.AcceptManualRefund(ctx, i.care, i.care.value.ID, "racing-refund-0001"); e != nil {
		return core.ResourceProof{}, e
	}
	return (inventoryProof{}).Finalize(ctx, a, key)
}
func TestRefundReservationDuringInventoryCommitBlocksPaid(t *testing.T) {
	h, p, a, tr := paymentFixture(t)
	if e := p.Ingest(h.ctx, tr); e != nil {
		t.Fatal(e)
	}
	var receipt string
	if e := h.db.QueryRow(h.ctx, "SELECT id FROM receipts").Scan(&receipt); e != nil {
		t.Fatal(e)
	}
	care := approvalStub{RefundApproval{ID: uuid.NewString(), Receipt: receipt, Connection: p.Connection, Receiver: p.Receiver, Approver: uuid.NewString(), Amount: tr.In, Currency: "VND", State: "APPROVED", Beneficiary: "000012345678", Bank: "VCB", Holder: "TEST", Policy: "manual-v1", Version: 1}}
	if _, e := p.WorkFinalize(h.ctx, racingRefundInventory{p, care}); e != nil {
		t.Fatal(e)
	}
	current, e := h.store.LoadOrder(h.ctx, a.ID(), ownerScope(a.Owner()))
	if e != nil {
		t.Fatal(e)
	}
	if current.HasMilestone(core.FirstPaid) || h.count(t, "SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 0 || h.count(t, "SELECT count(*) FROM reconciliation_cases WHERE reason='BLOCKED_AFTER_RESOURCE_COMMIT'") != 1 || h.count(t, "SELECT count(*) FROM jobs WHERE kind='PAYMENT_FINALIZE' AND status='PARKED'") != 1 {
		t.Fatal("refund/commit race produced PAID or lost recovery evidence")
	}
}

type approvalStub struct{ value RefundApproval }

func (a approvalStub) RefundApproval(context.Context, string) (RefundApproval, error) {
	return a.value, nil
}

type evidenceStub map[string]RefundEvidence

func (e evidenceStub) RefundEvidence(_ context.Context, ref string) (RefundEvidence, error) {
	v, ok := e[ref]
	if !ok {
		return v, ErrCapability
	}
	return v, nil
}
func refundActor(p *Payment) PaymentActor {
	return PaymentActor{ID: uuid.NewString(), Connection: p.Connection, Receivers: []string{p.Receiver}, MFA: time.Now(), Permissions: map[string]bool{"payment.refund.read": true, "payment.refund.execute": true, "payment.refund.verify": true, "payment.refund.accept": true, "payment.evidence.write": true, "payment.evidence.read": true}}
}
func refundFixture(t *testing.T) (*harness, *Payment, approvalStub, RefundView, PaymentActor) {
	t.Helper()
	h, p, _, tr := paymentFixture(t)
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
	care := approvalStub{RefundApproval{ID: uuid.NewString(), Receipt: receipt, Connection: p.Connection, Receiver: p.Receiver, Approver: uuid.NewString(), Amount: tr.In, Currency: "VND", State: "APPROVED", Beneficiary: "000012345678", Bank: "VCB", Holder: "SYNTHETIC TEST", Policy: "manual-v1", Version: 1}}
	v, e := p.AcceptManualRefund(h.ctx, care, care.value.ID, "accept-refund-0001")
	if e != nil {
		t.Fatal(e)
	}
	return h, p, care, v, refundActor(p)
}
func TestManualRefundBudgetAndStartGuards(t *testing.T) {
	h, p, care, v, actor := refundFixture(t)
	replay, e := p.AcceptManualRefund(h.ctx, nil, care.value.ID, "accept-refund-0001")
	if e != nil || replay.ID != v.ID {
		t.Fatal(replay, e)
	}
	if _, e = p.AcceptManualRefund(h.ctx, care, uuid.NewString(), "accept-refund-0001"); !errors.Is(e, ErrConflict) {
		t.Fatal("changed accept replay", e)
	}
	another := care
	another.value.ID = uuid.NewString()
	if _, e = p.AcceptManualRefund(h.ctx, another, another.value.ID, "accept-refund-0002"); !errors.Is(e, ErrConflict) {
		t.Fatal("budget exceeded", e)
	}
	approver := actor
	approver.ID = care.value.Approver
	if _, e = p.RefundCommand(h.ctx, approver, care, v.ID, "claim", "claim-approver-0001", v.Version, "test"); e == nil {
		t.Fatal("approver executed")
	}
	claimed, e := p.RefundCommand(h.ctx, actor, care, v.ID, "claim", "claim-refund-0001", v.Version, "test")
	if e != nil {
		t.Fatal(e)
	}
	noMFA := actor
	noMFA.MFA = time.Time{}
	if _, e = p.RefundCommand(h.ctx, noMFA, care, v.ID, "start", "start-no-mfa-0001", claimed.Version, "test"); !errors.Is(e, ErrForbidden) {
		t.Fatal("MFA bypass", e)
	}
	started, e := p.RefundCommand(h.ctx, actor, care, v.ID, "start", "start-refund-0001", claimed.Version, "test")
	if e != nil || started.AttemptID == nil {
		t.Fatal(started, e)
	}
	replay, e = p.RefundCommand(h.ctx, actor, care, v.ID, "start", "start-refund-0001", claimed.Version, "test")
	if e != nil || replay.Attempt != 1 || *replay.AttemptID != *started.AttemptID {
		t.Fatal("attempt replay", replay, e)
	}
	unknown, e := p.RefundCommand(h.ctx, actor, care, v.ID, "report-uncertain", "uncertain-refund-0001", started.Version, "timeout")
	if e != nil || unknown.Execution != "UNKNOWN" {
		t.Fatal(unknown, e)
	}
	if _, e = p.RefundCommand(h.ctx, actor, care, v.ID, "retry-attempt", "retry-refund-0001", unknown.Version, "timeout"); !errors.Is(e, ErrConflict) {
		t.Fatal("unknown retried", e)
	}
	if _, e = p.ManualRefundInstructions(h.ctx, actor, care, v.ID); e == nil {
		t.Fatal("instructions shown after unknown")
	}
	if h.count(t, "SELECT count(*) FROM refunds WHERE budget_held") != 1 || h.count(t, "SELECT count(*) FROM refund_execution_attempts") != 1 {
		t.Fatal("unknown released budget or created attempt")
	}
}
func TestManualRefundIndependentEvidenceAndSettlement(t *testing.T) {
	for _, decision := range []string{"SENT", "NOT_SENT", "INCONCLUSIVE"} {
		t.Run(decision, func(t *testing.T) {
			h, p, care, v, actor := refundFixture(t)
			v, e := p.RefundCommand(h.ctx, actor, care, v.ID, "claim", "claim-refund-0001", v.Version, "test")
			if e != nil {
				t.Fatal(e)
			}
			v, e = p.RefundCommand(h.ctx, actor, care, v.ID, "start", "start-refund-0001", v.Version, "test")
			if e != nil {
				t.Fatal(e)
			}
			var source string
			if e = h.db.QueryRow(h.ctx, "SELECT bank_account_uuid FROM payment_receiver_profiles WHERE id=$1", p.Receiver).Scan(&source); e != nil {
				t.Fatal(e)
			}
			ref := uuid.NewString()
			cert := RefundEvidence{Reference: ref, Refund: v.ID, Attempt: *v.AttemptID, SHA256: strings.Repeat("a", 64), Source: "BANK_STATEMENT", Scan: "CLEAN", Decision: decision, Definitive: true, Amount: v.Amount, Currency: "VND", Beneficiary: care.value.Beneficiary, Bank: care.value.Bank, Holder: care.value.Holder, SourceBank: source, BankIdentity: uuid.NewString(), BankReference: uuid.NewString(), Occurred: time.Now().UTC().Truncate(time.Microsecond)}
			authority := evidenceStub{ref: cert}
			if _, e = p.AttachRefundEvidenceBatch(h.ctx, actor, authority, v.ID, *v.AttemptID, "attach-bad-batch-0001", v.Version, "test", []string{ref, uuid.NewString()}); e == nil || h.count(t, "SELECT count(*) FROM payment_evidence") != 0 {
				t.Fatal("partial evidence batch", e)
			}
			v, e = p.AttachRefundEvidenceBatch(h.ctx, actor, authority, v.ID, *v.AttemptID, "attach-refund-0001", v.Version, "test", []string{ref})
			if e != nil {
				t.Fatal(e)
			}
			var bank *RefundBankEvidence
			if decision == "SENT" {
				bene, _ := json.Marshal(map[string]string{"account_number": cert.Beneficiary, "bank_code": cert.Bank, "account_holder": cert.Holder})
				bank = &RefundBankEvidence{Source: source, Reference: cert.BankReference, Amount: cert.Amount, Currency: "VND", BeneficiaryHash: hash(bene), At: cert.Occurred}
			}
			if _, e = p.VerifyRefundEvidenceSet(h.ctx, actor, authority, v.ID, *v.AttemptID, decision, "verify-self-0001", v.Version, "test", []string{ref}, bank); !errors.Is(e, ErrForbidden) {
				t.Fatal("self verify", e)
			}
			verifier := refundActor(p)
			result, e := p.VerifyRefundEvidenceSet(h.ctx, verifier, authority, v.ID, *v.AttemptID, decision, "verify-refund-0001", v.Version, "test", []string{ref}, bank)
			if e != nil {
				t.Fatal(e)
			}
			repeated, e := p.VerifyRefundEvidenceSet(h.ctx, verifier, authority, v.ID, *v.AttemptID, decision, "verify-refund-0001", v.Version, "test", []string{ref}, bank)
			if e != nil || repeated.Version != result.Version {
				t.Fatal("verification replay", e)
			}
			expected := map[string]string{"SENT": "VERIFIED_SENT", "NOT_SENT": "VERIFIED_NOT_SENT", "INCONCLUSIVE": "UNKNOWN"}[decision]
			if result.Execution != expected {
				t.Fatal(result)
			}
			if h.count(t, "SELECT count(*) FROM refund_verifications") != 1 || h.count(t, "SELECT count(*) FROM refunds WHERE budget_held") != 1 {
				t.Fatal("verification not exactly once or budget forgotten")
			}
			if decision == "SENT" {
				if h.count(t, "SELECT count(*) FROM orders WHERE payment_status='REFUNDED'") != 1 || h.count(t, "SELECT count(*) FROM outbox_events WHERE event_type='vn.omama.payment.refund_settled.v1'") != 1 {
					t.Fatal("refund summary/event")
				}
			}
			if decision == "NOT_SENT" {
				next, e := p.RefundCommand(h.ctx, actor, care, v.ID, "retry-attempt", "retry-safe-refund-0001", result.Version, "confirmed not sent")
				if e != nil || next.Execution != "AWAITING_EXECUTOR" {
					t.Fatal(next, e)
				}
			}
		})
	}
}
func TestManualRefundConcurrentReceiptBudget(t *testing.T) {
	h, p, _, tr := paymentFixture(t)
	tr.Content = "unmatched"
	tr.Code = ""
	if e := p.Ingest(h.ctx, tr); e != nil {
		t.Fatal(e)
	}
	var receipt string
	if e := h.db.QueryRow(h.ctx, "SELECT id FROM receipts").Scan(&receipt); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := RefundApproval{ID: uuid.NewString(), Receipt: receipt, Connection: p.Connection, Receiver: p.Receiver, Approver: uuid.NewString(), Amount: tr.In, Currency: "VND", State: "APPROVED", Beneficiary: "000012345678", Bank: "VCB", Holder: "TEST", Policy: "manual-v1", Version: 1}
			_, e := p.AcceptManualRefund(h.ctx, approvalStub{a}, a.ID, uuid.NewString())
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else if !errors.Is(e, ErrConflict) {
			t.Fatal(e)
		}
	}
	if success != 1 || h.count(t, "SELECT count(*) FROM refunds WHERE budget_held") != 1 {
		t.Fatal("overspent receipt", success)
	}
}
