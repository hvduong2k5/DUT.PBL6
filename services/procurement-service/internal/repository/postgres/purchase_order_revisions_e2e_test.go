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

// Called only by the disposable producer/consumer stack, never a live database.
func assertPORevisionE2E(t *testing.T, ctx context.Context, db *pgxpool.Pool, repo *postgres.Repository, service *application.Service, supplierID, materialID, offerID uuid.UUID) {
	t.Helper()
	request := application.PurchaseOrderRequest{SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "10"}}}
	changedPO := func(withReceipt bool) domain.PurchaseOrder {
		t.Helper()
		p, err := service.CreatePurchaseOrder(ctx, request, "original-buyer")
		if err != nil {
			t.Fatal(err)
		}
		if p, err = service.SubmitPurchaseOrder(ctx, p.ID, "original-buyer"); err != nil {
			t.Fatal(err)
		}
		if p, err = service.ApprovePurchaseOrder(ctx, p.ID, "checker"); err != nil {
			t.Fatal(err)
		}
		if p, err = service.MarkPurchaseOrderSent(ctx, p.ID, "original-buyer"); err != nil {
			t.Fatal(err)
		}
		if withReceipt {
			_, err := service.CreateGoodsReceipt(ctx, application.GoodsReceiptRequest{PurchaseOrderID: p.ID, WarehouseCode: "WH-HUE", Lines: []application.GoodsReceiptLineRequest{{PurchaseOrderLineID: p.Lines[0].ID, QuantityReceived: "1"}}}, "receiver", "revision-receipt-"+p.ID.String())
			if err != nil {
				t.Fatal(err)
			}
		}
		if p, err = service.RecordSupplierResponse(ctx, p.ID, "buyer", domain.SupplierPOChangeRequested, "Change quantity"); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p := changedPO(false)
	revisedRequest := application.PurchaseOrderRequest{SupplierID: supplierID, Notes: "Revision two", Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "8"}}}
	const actor = "revision-author"
	const key = "revision-command-e2e-0001"
	if _, err := db.Exec(ctx, `CREATE FUNCTION reject_test_revision_command() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='REVISION' THEN RAISE EXCEPTION 'injected revision journal failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER reject_test_revision_command BEFORE INSERT ON purchase_order_action_results FOR EACH ROW EXECUTE FUNCTION reject_test_revision_command()`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RevisePurchaseOrderWithIdempotency(ctx, p.ID, revisedRequest, actor, key, p.Version, "Change quantity"); err == nil || !strings.Contains(err.Error(), "injected revision journal failure") {
		t.Fatalf("revision fault: %v", err)
	}
	stored, err := repo.GetPurchaseOrder(ctx, p.ID)
	if err != nil || stored.Version != p.Version || stored.Revision != p.Revision || stored.Status != p.Status || stored.Lines[0].ID != p.Lines[0].ID {
		t.Fatal("failed revision changed live PO")
	}
	var count int
	for _, query := range []string{`SELECT count(*) FROM purchase_order_revision_snapshots WHERE purchase_order_id=$1`, `SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action='REVISION_CREATED'`, `SELECT count(*) FROM purchase_order_action_results WHERE purchase_order_id=$1 AND action='REVISION'`} {
		if err := db.QueryRow(ctx, query, p.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("failed revision left evidence: %v count=%d", err, count)
		}
	}
	if _, err := db.Exec(ctx, `DROP TRIGGER reject_test_revision_command ON purchase_order_action_results; DROP FUNCTION reject_test_revision_command()`); err != nil {
		t.Fatal(err)
	}
	type result struct {
		po  domain.PurchaseOrder
		err error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			got, err := application.NewService(postgres.New(db)).RevisePurchaseOrderWithIdempotency(ctx, p.ID, revisedRequest, actor, key, p.Version, "Change quantity")
			results <- result{got, err}
		}()
	}
	close(start)
	var revision domain.PurchaseOrder
	for i := 0; i < 2; i++ {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if i == 1 && !reflect.DeepEqual(revision, got.po) {
			t.Fatal("concurrent retry returned different drafts")
		}
		revision = got.po
	}
	if revision.Revision != 2 || revision.Version != p.Version+1 || revision.Status != domain.PurchaseOrderDraft || revision.CreatedBy != actor || revision.ApprovedBy != "" || revision.SentBy != "" || revision.Lines[0].ID == p.Lines[0].ID {
		t.Fatal("revision metadata not reset")
	}
	history, err := repo.ListPurchaseOrderRevisions(ctx, p.ID, 100, 0)
	if err != nil || len(history) != 1 || history[0].Snapshot.Lines[0].Quantity != p.Lines[0].Quantity || history[0].Snapshot.Lines[0].UnitPriceMinor != p.Lines[0].UnitPriceMinor || history[0].ChangedBy != actor || history[0].ChangeReason != "Change quantity" {
		t.Fatalf("issued terms lost: %v", err)
	}
	originalArchive, _ := json.Marshal(history[0])
	for _, query := range []string{`UPDATE purchase_order_revision_snapshots SET snapshot=snapshot WHERE purchase_order_id=$1`, `DELETE FROM purchase_order_revision_snapshots WHERE purchase_order_id=$1`} {
		if _, err := db.Exec(ctx, query, p.ID); err == nil || !strings.Contains(err.Error(), "immutable") {
			t.Fatalf("snapshot mutation allowed: %v", err)
		}
	}
	if _, err := db.Exec(ctx, `TRUNCATE purchase_order_revision_snapshots`); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("snapshot truncate allowed: %v", err)
	}
	if _, err := service.ExecutePurchaseOrderAction(ctx, p.ID, actor, key, "SUBMIT", revision.Version, ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("revision key was reused as action")
	}
	if _, err := service.SetOfferActive(ctx, supplierID, offerID, false); err != nil {
		t.Fatal(err)
	}
	replay, err := application.NewService(postgres.New(db)).RevisePurchaseOrderWithIdempotency(ctx, p.ID, revisedRequest, actor, key, p.Version, " Change quantity ")
	if err != nil || !reflect.DeepEqual(revision, replay) {
		t.Fatalf("restart replay repriced draft: %v", err)
	}
	if _, err := service.SetOfferActive(ctx, supplierID, offerID, true); err != nil {
		t.Fatal(err)
	}
	next, err := service.SubmitPurchaseOrder(ctx, p.ID, actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApprovePurchaseOrder(ctx, p.ID, actor); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("reviser self-approved")
	}
	if next, err = service.ApprovePurchaseOrder(ctx, p.ID, "checker"); err != nil {
		t.Fatal(err)
	}
	if next, err = service.MarkPurchaseOrderSent(ctx, p.ID, actor); err != nil {
		t.Fatal(err)
	}
	if next, err = service.RecordSupplierResponse(ctx, p.ID, actor, domain.SupplierPOChangeRequested, "Change again"); err != nil {
		t.Fatal(err)
	}
	revisedRequest.Lines[0].Quantity = "6"
	third, err := service.RevisePurchaseOrderWithIdempotency(ctx, p.ID, revisedRequest, actor, "revision-command-e2e-0002", next.Version, "Change again")
	if err != nil || third.Revision != 3 {
		t.Fatalf("second revision: %v", err)
	}
	assertPODraftEditE2E(t, ctx, db, repo, service, third, supplierID, materialID, offerID)
	all, err := repo.ListPurchaseOrderRevisions(ctx, p.ID, 100, 0)
	if err != nil || len(all) != 2 || all[0].Revision != 2 || all[1].Revision != 1 {
		t.Fatal("archive order is not descending")
	}
	afterArchive, _ := json.Marshal(all[1])
	if string(afterArchive) != string(originalArchive) {
		t.Fatal("later revision changed original archive")
	}
	page, err := repo.ListPurchaseOrderRevisions(ctx, p.ID, 1, 1)
	if err != nil || len(page) != 1 || page[0].Revision != 1 {
		t.Fatal("archive pagination")
	}
	empty, err := repo.ListPurchaseOrderRevisions(ctx, p.ID, 1, 99)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("archive empty page")
	}
	revisedRequest.Lines[0].Quantity = "8"
	replay, err = service.RevisePurchaseOrderWithIdempotency(ctx, p.ID, revisedRequest, actor, key, p.Version, "Change quantity")
	if err != nil || replay.Revision != 2 || replay.Status != domain.PurchaseOrderDraft {
		t.Fatalf("old command affected newer revision: %v", err)
	}
	blocked := changedPO(true)
	if _, err := service.RevisePurchaseOrderWithIdempotency(ctx, blocked.ID, revisedRequest, actor, "revision-with-receipt-0001", blocked.Version, "Change quantity"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("revision with receipt accepted: %v", err)
	}
	if history, err := repo.ListPurchaseOrderRevisions(ctx, blocked.ID, 100, 0); err != nil || len(history) != 0 {
		t.Fatal("blocked receipt revision archived PO")
	}
	// Cross-resource journal race must also roll back the loser's archived row.
	left, right := changedPO(false), changedPO(false)
	results = make(chan result, 2)
	start = make(chan struct{})
	for _, current := range []domain.PurchaseOrder{left, right} {
		go func(p domain.PurchaseOrder) {
			<-start
			got, err := application.NewService(postgres.New(db)).RevisePurchaseOrderWithIdempotency(ctx, p.ID, revisedRequest, actor, "revision-cross-key-0001", p.Version, "Change quantity")
			results <- result{got, err}
		}(current)
	}
	close(start)
	wins, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		got := <-results
		if got.err == nil {
			wins++
		} else if errors.Is(got.err, domain.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(got.err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal("cross-resource revision key race")
	}
	for _, p := range []domain.PurchaseOrder{left, right} {
		stored, err := repo.GetPurchaseOrder(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		history, err := repo.ListPurchaseOrderRevisions(ctx, p.ID, 100, 0)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Revision == 1 && (stored.Version != p.Version || len(history) != 0) {
			t.Fatal("losing revision leaked archive or state")
		}
	}
}
