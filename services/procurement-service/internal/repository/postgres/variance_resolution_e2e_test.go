package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/repository/postgres"
)

// Runs after the canonical two-event broker proof. Extra synthetic inspections
// stay in this disposable schema's outbox; they are never sent to Inventory.
func assertVarianceResolutionE2E(t *testing.T, ctx context.Context, db *pgxpool.Pool, repo *postgres.Repository, service *application.Service, supplierID, materialID uuid.UUID) {
	t.Helper()
	// Retry keys/hashes are internal metadata, deliberately absent from the
	// durable public snapshot. Compare the complete serialized HTTP projection.
	sameReceipt := func(left, right domain.GoodsReceipt) bool {
		t.Helper()
		a, err := json.Marshal(left)
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(right)
		if err != nil {
			t.Fatal(err)
		}
		return string(a) == string(b)
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	sentPO := func() domain.PurchaseOrder {
		t.Helper()
		p, err := service.CreatePurchaseOrder(ctx, application.PurchaseOrderRequest{SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "2"}}}, "variance-buyer")
		if err != nil {
			t.Fatal(err)
		}
		if p, err = service.SubmitPurchaseOrder(ctx, p.ID, "variance-buyer"); err != nil {
			t.Fatal(err)
		}
		if p, err = service.ApprovePurchaseOrder(ctx, p.ID, "checker"); err != nil {
			t.Fatal(err)
		}
		if p, err = service.MarkPurchaseOrderSent(ctx, p.ID, "variance-buyer"); err != nil {
			t.Fatal(err)
		}
		return p
	}
	inspect := func(p domain.PurchaseOrder, quantity, accepted, rejected string) domain.GoodsReceipt {
		t.Helper()
		r, err := service.CreateGoodsReceipt(ctx, application.GoodsReceiptRequest{PurchaseOrderID: p.ID, WarehouseCode: "WH-HUE", Lines: []application.GoodsReceiptLineRequest{{PurchaseOrderLineID: p.Lines[0].ID, QuantityReceived: quantity}}}, "receiver", "variance-receipt-"+uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		r, err = service.InspectGoodsReceiptWithIdempotency(ctx, r.ID, "inspector", "variance-inspect-"+r.ID.String(), []domain.GoodsReceiptInspectionLine{{GoodsReceiptLineID: r.Lines[0].ID, QuantityAccepted: accepted, QuantityRejected: rejected, InspectionNotes: "Synthetic receiving exception"}})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	p := sentPO()
	first, second := inspect(p, "2", "1", "1"), inspect(p, "2", "1", "1")
	var overID uuid.UUID
	for _, v := range second.Variances {
		if v.Type == domain.GoodsReceiptVarianceOverReceived {
			overID = v.ID
		}
	}
	if overID == uuid.Nil || len(first.Variances) != 1 {
		t.Fatal("missing synthetic open variances")
	}
	const actor = "variance-operator"
	const key = "variance-resolution-pg-0001"
	beforePO, err := repo.GetPurchaseOrder(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	events := count(`SELECT count(*) FROM goods_received_outbox`)
	if _, err := db.Exec(ctx, `CREATE FUNCTION reject_test_variance_result() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected variance journal failure'; END $$;
CREATE TRIGGER reject_test_variance_result BEFORE INSERT ON receipt_variance_resolution_results FOR EACH ROW EXECUTE FUNCTION reject_test_variance_result()`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, second.ID, overID, "ACCEPT_VARIANCE", actor, "Reviewed excess", key, second.Version); err == nil || !strings.Contains(err.Error(), "injected variance journal failure") {
		t.Fatalf("variance journal fault: %v", err)
	}
	stored, err := repo.GetGoodsReceipt(ctx, second.ID)
	if err != nil || stored.Version != second.Version || !reflect.DeepEqual(stored.Variances, second.Variances) {
		t.Fatal("variance decision did not roll back")
	}
	afterPO, err := repo.GetPurchaseOrder(ctx, p.ID)
	if err != nil || afterPO.Version != beforePO.Version || afterPO.Status != beforePO.Status {
		t.Fatal("variance fault leaked PO transition")
	}
	if count(`SELECT count(*) FROM receipt_variance_resolution_results WHERE receipt_id=$1`, second.ID) != 0 || count(`SELECT count(*) FROM goods_receipt_status_history WHERE goods_receipt_id=$1 AND action='VARIANCE_RESOLVED'`, second.ID) != 0 {
		t.Fatal("failed decision consumed key/history")
	}
	if _, err := db.Exec(ctx, `DROP TRIGGER reject_test_variance_result ON receipt_variance_resolution_results; DROP FUNCTION reject_test_variance_result()`); err != nil {
		t.Fatal(err)
	}
	type result struct {
		receipt domain.GoodsReceipt
		err     error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			r, err := application.NewService(postgres.New(db)).ResolveGoodsReceiptVarianceWithIdempotency(ctx, second.ID, overID, "ACCEPT_VARIANCE", actor, "Reviewed excess", key, second.Version)
			results <- result{r, err}
		}()
	}
	close(start)
	var frozen domain.GoodsReceipt
	for i := 0; i < 2; i++ {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if i == 1 && !sameReceipt(frozen, got.receipt) {
			t.Fatal("duplicate decisions differ")
		}
		frozen = got.receipt
	}
	if frozen.Version != second.Version+1 || count(`SELECT count(*) FROM goods_receipt_status_history WHERE goods_receipt_id=$1 AND action='VARIANCE_RESOLVED'`, second.ID) != 1 {
		t.Fatal("duplicate variance resolution history/version")
	}
	if _, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, first.ID, first.Variances[0].ID, "ACCEPT_VARIANCE", actor, "Reviewed excess", key, first.Version); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("key reused across receipts")
	}
	// Resolve the last quality variance on both receipts concurrently. PO locking
	// must let the last committer see the other's decision and reach RECEIVED.
	currentSecond, err := repo.GetGoodsReceipt(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	var qualityID uuid.UUID
	for _, v := range currentSecond.Variances {
		if v.Status == "OPEN" {
			qualityID = v.ID
		}
	}
	results = make(chan result, 2)
	start = make(chan struct{})
	for _, cmd := range []struct {
		receipt  domain.GoodsReceipt
		variance uuid.UUID
		key      string
	}{{first, first.Variances[0].ID, "variance-parallel-first-0001"}, {currentSecond, qualityID, "variance-parallel-second-0001"}} {
		go func(r domain.GoodsReceipt, v uuid.UUID, k string) {
			<-start
			got, err := application.NewService(postgres.New(db)).ResolveGoodsReceiptVarianceWithIdempotency(ctx, r.ID, v, "REJECT_VARIANCE", actor, "Keep rejected stock quarantined", k, r.Version)
			results <- result{got, err}
		}(cmd.receipt, cmd.variance, cmd.key)
	}
	close(start)
	for i := 0; i < 2; i++ {
		if got := <-results; got.err != nil {
			t.Fatal(got.err)
		}
	}
	completed, err := repo.GetPurchaseOrder(ctx, p.ID)
	if err != nil || completed.Status != domain.PurchaseOrderReceived {
		t.Fatalf("last parallel resolution left PO stale: %v status=%s", err, completed.Status)
	}
	if completed.Lines[0].AcceptedQuantity != beforePO.Lines[0].AcceptedQuantity || completed.Lines[0].ReceivedQuantity != beforePO.Lines[0].ReceivedQuantity || count(`SELECT count(*) FROM goods_received_outbox`) != events {
		t.Fatal("variance decision changed quantity or emitted event")
	}
	replay, err := application.NewService(postgres.New(db)).ResolveGoodsReceiptVarianceWithIdempotency(ctx, second.ID, overID, " accept_variance ", actor, " Reviewed excess ", key, second.Version)
	if err != nil || !sameReceipt(frozen, replay) {
		t.Fatalf("restart/normalized decision replay: %v", err)
	}
	if count(`SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action='VARIANCE_RESOLVED'`, p.ID) != 1 {
		t.Fatal("parallel decisions duplicated PO completion history")
	}
	// A committed replay remains valid after closure, but fresh commands must
	// never reopen the closed PO or fabricate evidence for legacy decisions.
	partialPO := sentPO()
	partial := inspect(partialPO, "1", "0.5", "0.5")
	v := partial.Variances[0].ID
	frozenPartial, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, partial.ID, v, "REJECT_VARIANCE", actor, "Partial receipt accepted as evidence", "variance-close-replay-0001", partial.Version)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := service.CloseRemainingPurchaseOrder(ctx, partialPO.ID, "buyer", "Cancel unreceived remainder")
	if err != nil {
		t.Fatal(err)
	}
	replay, err = service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, partial.ID, v, "REJECT_VARIANCE", actor, "Partial receipt accepted as evidence", "variance-close-replay-0001", partial.Version)
	if err != nil || !sameReceipt(frozenPartial, replay) {
		t.Fatal("closed PO broke committed replay")
	}
	if _, err := service.ResolveGoodsReceiptVarianceWithIdempotency(ctx, partial.ID, v, "REJECT_VARIANCE", actor, "Partial receipt accepted as evidence", "variance-new-after-close-0001", frozenPartial.Version); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("new resolution reopened closed PO: %v", err)
	}
	if got, err := repo.GetPurchaseOrder(ctx, partialPO.ID); err != nil || got.Status != domain.PurchaseOrderClosed || got.Version != closed.Version {
		t.Fatal("closed PO changed")
	}
}
