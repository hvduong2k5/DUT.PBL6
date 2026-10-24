package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain/core"
)

// Integration tests for the G4/G5 fixes. They run against real PostgreSQL
// via the shared harness (ORDER_CORE_TEST_DATABASE) and are skipped when no
// database is configured.

type fakeCheckoutSource struct{ calls int }

func (f *fakeCheckoutSource) Revalidate(context.Context, core.SnapshotInput) error {
	f.calls++
	return nil
}

type fakeCheckoutInventory struct {
	reserveErr   error
	lookupProof  core.ResourceProof
	lookupErr    error
	releaseProof core.ResourceProof
	releaseErr   error
	releaseCalls int
}

func (f *fakeCheckoutInventory) Lookup(context.Context, core.SnapshotInput, string) (core.ResourceProof, error) {
	return f.lookupProof, f.lookupErr
}
func (f *fakeCheckoutInventory) Reserve(context.Context, core.SnapshotInput, string) (core.ResourceProof, error) {
	return core.ResourceProof{}, f.reserveErr
}
func (f *fakeCheckoutInventory) Commit(context.Context, core.SnapshotInput, core.ResourceProof, string) (core.ResourceProof, error) {
	return core.ResourceProof{}, errors.New("not used")
}
func (f *fakeCheckoutInventory) Release(context.Context, core.SnapshotInput, core.ResourceProof, string) (core.ResourceProof, error) {
	f.releaseCalls++
	return f.releaseProof, f.releaseErr
}

func acceptWorkable(t *testing.T, h *harness) Acceptance {
	t.Helper()
	in := acceptanceInput(t, 0)
	d := snapshotInput(t, 0)
	now := time.Now()
	d.Catalog.ValidUntil = now.Add(20 * time.Minute)
	d.Shipping.ValidUntil = now.Add(20 * time.Minute)
	raw, err := json.Marshal(AcceptedCheckout{Terms: d, QuoteExpires: now.Add(20 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	in.AcceptedJSON = raw
	in.QuoteExpiresAt = now.Add(20 * time.Minute)
	a, err := h.store.Accept(h.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// errInventoryDown is the only error driveToPark tolerates from Work: it is
// the fake Reserve failure that accumulates attempts toward the park.
var errInventoryDown = errors.New("inventory unavailable")

func driveToPark(t *testing.T, h *harness, rt *CheckoutRuntime, operation string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		if _, err := rt.Work(h.ctx); err != nil && !errors.Is(err, errInventoryDown) {
			t.Fatalf("unexpected Work error while parking: %v", err)
		}
		var status string
		if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", operation).Scan(&status); err != nil {
			t.Fatal(err)
		}
		switch status {
		case "MANUAL_REVIEW":
			return
		case "FAILED":
			t.Fatal("operation FAILED instead of parking")
		}
		if time.Now().After(deadline) {
			t.Fatal("operation did not park")
		}
		// A failed attempt reschedules the job 1s out (run_after), so calling
		// Work in a tight loop does not execute real attempts. Pull the job
		// due inside the fixture instead of sleeping: each loop iteration
		// then runs one genuine attempt.
		if _, err := h.db.Exec(h.ctx, "UPDATE jobs SET run_after=clock_timestamp() WHERE workflow_id IN (SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_id=$1) AND status IN ('READY','RETRY_WAIT')", operation); err != nil {
			t.Fatal(err)
		}
	}
}

func jobStatus(t *testing.T, h *harness, kind, operation string) string {
	t.Helper()
	var status string
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM jobs WHERE kind=$1 AND key=$2", kind, kind+":"+operation).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestReconcileResumePreservesRemainingBudget(t *testing.T) {
	h := database(t, true)
	a := acceptWorkable(t, h)
	inv := &fakeCheckoutInventory{
		reserveErr:  errInventoryDown,
		lookupProof: core.ResourceProof{State: "RESERVED", ReservationID: uuid.NewString(), ExpiresAt: time.Now().Add(10 * time.Minute)},
	}
	rt := &CheckoutRuntime{Store: h.store, Source: &fakeCheckoutSource{}, Inventory: inv}
	driveToPark(t, h, rt, a.OperationID)
	// Simulate a park that still has budget left (the reconciler may only
	// resume those); a budget-spent park is covered by the review test.
	if _, err := h.db.Exec(h.ctx, "UPDATE jobs SET attempts=5 WHERE status='PARKED' AND workflow_id IN (SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_id=$1)", a.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.WorkReconcile(h.ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	var attempts int
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "WAITING_RETRY" {
		t.Fatalf("operation after resume: %s", status)
	}
	if err := h.db.QueryRow(h.ctx, "SELECT attempts FROM jobs WHERE workflow_id IN (SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_id=$1)", a.OperationID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 5 {
		t.Fatalf("reconcile resume must preserve the remaining budget, got attempts=%d", attempts)
	}
}

func TestReconcileReviewThenRecoverAndSecondParkRearms(t *testing.T) {
	h := database(t, true)
	a := acceptWorkable(t, h)
	d := snapshotInput(t, 0)
	inv := &fakeCheckoutInventory{
		reserveErr:  errInventoryDown,
		lookupProof: core.ResourceProof{State: "RESERVED", ReservationID: uuid.NewString(), ExpiresAt: time.Now().Add(10 * time.Minute)},
	}
	rt := &CheckoutRuntime{Store: h.store, Source: &fakeCheckoutSource{}, Inventory: inv}
	driveToPark(t, h, rt, a.OperationID)
	if got := jobStatus(t, h, "RESOURCE_RECONCILE", a.OperationID); got != "READY" {
		t.Fatalf("reconcile job after first park: %s", got)
	}
	// Budget is spent (20 attempts): a valid RESERVED proof alone must not
	// resume the checkout; the operation stays for audited admin Recover.
	if _, err := rt.WorkReconcile(h.ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "MANUAL_REVIEW" {
		t.Fatalf("budget-spent park must stay MANUAL_REVIEW after reconcile, got %s", status)
	}
	if got := jobStatus(t, h, "RESOURCE_RECONCILE", a.OperationID); got != "DONE" {
		t.Fatalf("reconcile job after review disposition: %s", got)
	}
	if err := rt.Recover(h.ctx, a.OperationID, "manager:test", "", d.MerchantID, "recover-key-1", "manual review approved"); err != nil {
		t.Fatal(err)
	}
	// Park a second time: the finished reconcile generation must be rearmed.
	driveToPark(t, h, rt, a.OperationID)
	if got := jobStatus(t, h, "RESOURCE_RECONCILE", a.OperationID); got != "READY" {
		t.Fatalf("reconcile job after second park: %s (want rearmed READY)", got)
	}
	if n := h.count(t, "SELECT count(*) FROM workflows WHERE kind='RESOURCE_RECONCILE' AND aggregate_id=$1", a.OperationID); n != 2 {
		t.Fatalf("reconcile generations: %d, want 2", n)
	}
}

func TestWorkBudgetGateBlocksDispatchBeyondBudget(t *testing.T) {
	h := database(t, true)
	a := acceptWorkable(t, h)
	src := &fakeCheckoutSource{}
	inv := &fakeCheckoutInventory{reserveErr: errInventoryDown}
	rt := &CheckoutRuntime{Store: h.store, Source: src, Inventory: inv}
	// The next claim is attempt 21: Work must park without dispatching
	// anything (no revalidation, no remote call, no stage mutation).
	if _, err := h.db.Exec(h.ctx, "UPDATE jobs SET attempts=20 WHERE workflow_id IN (SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_id=$1)", a.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.Work(h.ctx); err != nil {
		t.Fatal(err)
	}
	if src.calls != 0 {
		t.Fatalf("budget gate dispatched revalidation %d times", src.calls)
	}
	var status string
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "MANUAL_REVIEW" {
		t.Fatalf("operation after budget gate: %s", status)
	}
	if got := jobStatus(t, h, "RESOURCE_RECONCILE", a.OperationID); got != "READY" {
		t.Fatalf("reconcile job after budget park: %s", got)
	}
}

func TestReconcileProofStaleAfterAdminRecover(t *testing.T) {
	h := database(t, true)
	a := acceptWorkable(t, h)
	d := snapshotInput(t, 0)
	inv := &fakeCheckoutInventory{
		reserveErr:  errInventoryDown,
		lookupProof: core.ResourceProof{State: "RELEASED", ReservationID: uuid.NewString(), ExpiresAt: time.Now().Add(10 * time.Minute)},
	}
	rt := &CheckoutRuntime{Store: h.store, Source: &fakeCheckoutSource{}, Inventory: inv}
	driveToPark(t, h, rt, a.OperationID)
	// Admin wins the race: Recover resumes the checkout before reconcile runs.
	if err := rt.Recover(h.ctx, a.OperationID, "manager:test", "", d.MerchantID, "recover-key-1", "manual review approved"); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.WorkReconcile(h.ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "WAITING_RETRY" {
		t.Fatalf("reconcile must not apply a stale RELEASED proof after Recover, operation: %s", status)
	}
	if got := jobStatus(t, h, "RESOURCE_RECONCILE", a.OperationID); got != "DONE" {
		t.Fatalf("reconcile job after stale proof: %s", got)
	}
}

func TestReconcileLapsedReservedNeedsAuthoritativeRelease(t *testing.T) {
	h := database(t, true)
	a := acceptWorkable(t, h)
	inv := &fakeCheckoutInventory{
		reserveErr:   errInventoryDown,
		lookupProof:  core.ResourceProof{State: "RESERVED", ReservationID: uuid.NewString(), ExpiresAt: time.Now().Add(-time.Minute)},
		releaseErr:   errors.New("release unavailable"),
		releaseProof: core.ResourceProof{State: "EXPIRED", ReservationID: uuid.NewString()},
	}
	rt := &CheckoutRuntime{Store: h.store, Source: &fakeCheckoutSource{}, Inventory: inv}
	driveToPark(t, h, rt, a.OperationID)
	// Release fails: the obligation must stay open.
	if _, err := rt.WorkReconcile(h.ctx); err == nil {
		t.Fatal("expected reconcile to fail while Release is unavailable")
	}
	var status string
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "MANUAL_REVIEW" {
		t.Fatalf("operation must stay MANUAL_REVIEW without release proof, got %s", status)
	}
	if n := h.count(t, "SELECT count(*) FROM checkout_submission_guards WHERE operation_id=$1 AND state='ACTIVE'", a.OperationID); n != 1 {
		t.Fatal("submission guard must stay ACTIVE without release proof")
	}
	if got := jobStatus(t, h, "RESOURCE_RECONCILE", a.OperationID); got == "DONE" {
		t.Fatal("reconcile job must not finish without release proof")
	}
	// Release now returns the authoritative EXPIRED proof: obligation closes.
	inv.releaseErr = nil
	if _, err := h.db.Exec(h.ctx, "UPDATE jobs SET run_after=clock_timestamp() WHERE kind='RESOURCE_RECONCILE'"); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.WorkReconcile(h.ctx); err != nil {
		t.Fatal(err)
	}
	if inv.releaseCalls == 0 {
		t.Fatal("reconciler never called Release")
	}
	if err := h.db.QueryRow(h.ctx, "SELECT status FROM checkout_operations WHERE id=$1", a.OperationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "FAILED" {
		t.Fatalf("operation after authoritative EXPIRED proof: %s", status)
	}
	if n := h.count(t, "SELECT count(*) FROM checkout_submission_guards WHERE operation_id=$1 AND state='RELEASED'", a.OperationID); n != 1 {
		t.Fatal("submission guard must be RELEASED after authoritative proof")
	}
}
