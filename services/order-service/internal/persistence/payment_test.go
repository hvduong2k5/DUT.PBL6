package persistence

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/sepay"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func paymentFixture(t *testing.T) (*harness, *Payment, core.Aggregate, sepay.Transaction) {
	t.Helper()
	h := database(t, false)
	_, place := prepared(t, h, 0)
	a, e := h.store.Place(h.ctx, place)
	if e != nil {
		var raw []byte
		_ = h.db.QueryRow(h.ctx, "SELECT canonical_cipher FROM checkout_operations WHERE id=$1", place.OperationID).Scan(&raw)
		data, _ := h.store.cipher.Open("canonical:"+place.OperationID, raw)
		var debug core.SnapshotInput
		_ = json.Unmarshal(data, &debug)
		var now time.Time
		_ = h.db.QueryRow(h.ctx, "SELECT clock_timestamp()").Scan(&now)
		t.Fatalf("place: %v accepted=%s canonical=%s db_now=%s payment_expiry=%v", e, debug.Provenance.AcceptedAt, debug.Provenance.CanonicalizedAt, now, debug.Payment.ExpiresAt)
	}
	p := &Payment{Store: h.store, Receiver: *a.Snapshot().Data().Payment.ReceiverProfileID, Egress: "isolated-test-egress", Secret: strings.Repeat("s", 32), KeyVersion: "hmac-v1", Bootstrap: time.Now().Add(-time.Minute)}
	var raw []byte
	var bank, code string
	if e = h.db.QueryRow(h.ctx, "SELECT r.connection_id,r.bank_account_uuid,r.config_cipher,i.payment_code FROM payment_receiver_profiles r JOIN payment_intents i ON i.receiver_profile_id=r.id WHERE r.id=$1", p.Receiver).Scan(&p.Connection, &bank, &raw, &code); e != nil {
		t.Fatal(e)
	}
	data, e := h.store.cipher.Open("receiver:"+p.Receiver, raw)
	if e != nil {
		t.Fatal(e)
	}
	var receiver sepay.Receiver
	if json.Unmarshal(data, &receiver) != nil {
		t.Fatal("bad receiver")
	}
	if e = p.Onboard(h.ctx, receiver); e != nil {
		t.Fatal(e)
	}
	transaction := sepay.Transaction{ID: uuid.NewString(), BankAccountID: bank, Account: receiver.Account, Direction: "in", In: a.Snapshot().Data().Money.FinalAmount, Date: time.Now().Truncate(time.Second).Format(time.RFC3339), Content: code, Code: code, Reference: uuid.NewString()}
	return h, p, a, transaction
}
func hook(t sepay.Transaction) []byte {
	at, _ := sepay.BankTime(t.Date)
	raw, _ := json.Marshal(sepay.Webhook{ID: 9007199254740993, Gateway: "VCB", Date: at.In(time.FixedZone("VN", 7*3600)).Format("2006-01-02 15:04:05"), Account: t.Account, Content: t.Content, Code: t.Code, Direction: t.Direction, Amount: t.In, Reference: t.Reference})
	return raw
}
func observe(p *Payment, raw []byte) error {
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	h := hmac.New(sha256.New, []byte(p.Secret))
	h.Write([]byte(stamp + "."))
	h.Write(raw)
	return p.Observe(context.Background(), raw, stamp, "sha256="+hex.EncodeToString(h.Sum(nil)))
}

type inventoryProof struct{ release bool }

func (i inventoryProof) Finalize(ctx context.Context, a core.Aggregate, key string) (core.ResourceProof, error) {
	proof := a.Record().Reservation
	proof.ResourceVersion++
	proof.DecisionID = uuid.NewString()
	proof.DecisionAt = time.Now()
	proof.State = "COMMITTED"
	if i.release {
		proof.State = "RELEASED"
	}
	return proof, nil
}
func TestSePayObserveAllocateFinalizeReplay(t *testing.T) {
	h, p, a, transaction := paymentFixture(t)
	raw := hook(transaction)
	if e := observe(p, raw); e != nil {
		t.Fatal(e)
	}
	if e := observe(p, raw); e != nil {
		t.Fatal(e)
	}
	if h.count(t, "SELECT count(*) FROM receipts") != 0 || h.count(t, "SELECT count(*) FROM payment_provider_observations") != 1 {
		t.Fatal("webhook directly booked money or duplicated")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- p.Ingest(h.ctx, transaction) }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	current, e := h.store.LoadOrder(h.ctx, a.ID(), ownerScope(a.Owner()))
	if e != nil || current.State() != core.PaymentFinalizing {
		t.Fatal(current.State(), e)
	}
	if h.count(t, "SELECT count(*) FROM receipts") != 1 || h.count(t, "SELECT count(*) FROM allocations") != 1 || h.count(t, "SELECT count(*) FROM payment_provider_observations WHERE status='LINKED'") != 1 {
		t.Fatal("financial dedup/link failed")
	}
	if worked, e := p.WorkFinalize(h.ctx, inventoryProof{}); e != nil || !worked {
		t.Fatal(worked, e)
	}
	current, e = h.store.LoadOrder(h.ctx, a.ID(), ownerScope(a.Owner()))
	if e != nil || current.State() != core.Paid {
		t.Fatal(current.State(), e)
	}
	if e = p.Ingest(h.ctx, transaction); e != nil {
		t.Fatal(e)
	}
	if h.count(t, "SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 1 || h.count(t, "SELECT count(*) FROM outbox_events WHERE event_type='vn.omama.order.paid.v3'") != 1 {
		t.Fatal("paid repeated")
	}
	extra := transaction
	extra.ID = uuid.NewString()
	extra.Reference = uuid.NewString()
	if e = p.Ingest(h.ctx, extra); e != nil {
		t.Fatal(e)
	}
	if h.count(t, "SELECT count(*) FROM reconciliation_cases WHERE reason='EXTRA_RECEIPT'") != 1 || h.count(t, "SELECT count(*) FROM allocations") != 1 {
		t.Fatal("extra money replaced allocation")
	}
}
func TestSePayAnomaliesAndFinancialConflicts(t *testing.T) {
	for _, kind := range []string{"under", "over", "unmatched", "cancelled", "stock-released", "changed-facts"} {
		t.Run(kind, func(t *testing.T) {
			h, p, a, tr := paymentFixture(t)
			switch kind {
			case "under":
				tr.In--
			case "over":
				tr.In = 1000000000001
			case "unmatched":
				tr.Content = "no code"
				tr.Code = ""
			case "cancelled":
				e := h.store.runner.Do(h.ctx, func(tx pgx.Tx) error {
					var now time.Time
					if e := tx.QueryRow(h.ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
						return e
					}
					next, e := a.CancelPending(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, a.Owner())
					if e != nil {
						return e
					}
					return h.store.savePaymentRoot(h.ctx, tx, a, next, "UNPAID", "TEST_CANCEL")
				})
				if e != nil {
					t.Fatal(e)
				}
			}
			if e := p.Ingest(h.ctx, tr); e != nil {
				t.Fatal(e)
			}
			if kind == "changed-facts" {
				tr.In++
				if e := p.Ingest(h.ctx, tr); e != nil {
					t.Fatal(e)
				}
				if h.count(t, "SELECT count(*) FROM reconciliation_cases WHERE reason='PROVIDER_DATA_CONFLICT'") != 1 {
					t.Fatal("no conflict")
				}
				return
			}
			if kind == "stock-released" {
				if _, e := p.WorkFinalize(h.ctx, inventoryProof{release: true}); e != nil {
					t.Fatal(e)
				}
				current, e := h.store.LoadOrder(h.ctx, a.ID(), ownerScope(a.Owner()))
				if e != nil || current.State() != core.CancelledTimeout {
					t.Fatal(current.State(), e)
				}
				return
			}
			if h.count(t, "SELECT count(*) FROM allocations") != 0 || h.count(t, "SELECT count(*) FROM reconciliation_cases") != 1 {
				t.Fatal("anomaly auto allocated")
			}
		})
	}
}
func TestSePayInstructionsOwnershipCircuitAndWatch(t *testing.T) {
	h, p, a, tr := paymentFixture(t)
	scope := ownerScope(a.Owner())
	if _, e := p.Instructions(h.ctx, a.ID(), scope); !errors.Is(e, ErrExpired) {
		t.Fatal("stale circuit allowed QR", e)
	}
	if _, e := h.db.Exec(h.ctx, "UPDATE payment_sync_checkpoints SET circuit='HEALTHY',last_scanned_end=clock_timestamp() WHERE receiver_id=$1 AND stream='FAST'", p.Receiver); e != nil {
		t.Fatal(e)
	}
	first, e := p.Instructions(h.ctx, a.ID(), scope)
	if e != nil {
		t.Fatal(e)
	}
	second, e := p.Instructions(h.ctx, a.ID(), scope)
	if e != nil || first.Code != second.Code || !first.Expires.Equal(second.Expires) {
		t.Fatal("instruction refresh minted intent", e)
	}
	if _, e = p.Instructions(h.ctx, a.ID(), "customer:"+uuid.NewString()); !errors.Is(e, pgx.ErrNoRows) {
		t.Fatal("ownership leak", e)
	}
	if e = observe(p, hook(tr)); e != nil {
		t.Fatal(e)
	}
	status, e := p.Status(h.ctx, a.ID(), scope)
	if e != nil || status.Display != "VERIFYING" || status.CanPay {
		t.Fatal(status, e)
	}
	if _, e = h.db.Exec(h.ctx, "UPDATE payment_order_projections SET verification_due_at=clock_timestamp()-interval '1 second' WHERE order_id=$1", a.ID()); e != nil {
		t.Fatal(e)
	}
	status, e = p.Status(h.ctx, a.ID(), scope)
	if e != nil || status.Display != "VERIFICATION_DELAYED" || status.Poll != 10 {
		t.Fatal(status, e)
	}
}

type pages struct {
	rows  [][]json.RawMessage
	bad   bool
	calls int
}

func (s *pages) List(ctx context.Context, w sepay.Window) (sepay.Page, error) {
	s.calls++
	if s.bad {
		return sepay.Page{}, &sepay.APIError{Status: 401, RetryAfter: time.Second}
	}
	rows := s.rows[w.Page-1]
	return sepay.Page{Rows: rows, Pagination: sepay.Pagination{Page: w.Page, Size: 100, Last: len(s.rows), More: w.Page < len(s.rows)}}, nil
}
func TestSePayPagedSyncQuarantineAndLostLease(t *testing.T) {
	h, p, a, tr := paymentFixture(t)
	raw, _ := json.Marshal(tr)
	reader := &pages{rows: [][]json.RawMessage{{raw}, {[]byte(`{"id":"broken"}`)}}}
	if _, e := p.WorkSync(h.ctx, reader); e != nil {
		t.Fatal(e)
	}
	if reader.calls != 2 || h.count(t, "SELECT count(*) FROM receipts") != 1 || h.count(t, "SELECT count(*) FROM payment_api_quarantine") != 1 || h.count(t, "SELECT count(*) FROM payment_sync_runs WHERE status='COMPLETED_WITH_REVIEW'") != 1 {
		t.Fatal("page/coverage/quarantine failed")
	}
	j, e := p.claim(h.ctx, "PAYMENT_FINALIZE")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.db.Exec(h.ctx, "UPDATE jobs SET lease_epoch=lease_epoch+1 WHERE id=$1", j.ID); e != nil {
		t.Fatal(e)
	}
	proof, _ := inventoryProof{}.Finalize(h.ctx, a, "")
	if e = p.finalized(h.ctx, j, proof); !errors.Is(e, ErrLease) {
		t.Fatal("stale owner finalized", e)
	}
	if h.count(t, "SELECT count(*) FROM order_milestones WHERE kind='FIRST_PAID'") != 0 {
		t.Fatal("stale paid")
	}
}
