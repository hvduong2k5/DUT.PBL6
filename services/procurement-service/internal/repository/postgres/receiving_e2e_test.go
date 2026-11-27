package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	producer "github.com/omamx/procurement-service/internal/infrastructure/kafka"
	"github.com/omamx/procurement-service/internal/repository/postgres"
	"github.com/omamx/procurement-service/internal/worker"
	"github.com/omamx/procurement-service/migrations"
)

// Invoked by the isolated Inventory adapter only after its consumer group has
// committed a probe. No live service credentials or existing schemas are used.
func TestPublishInspectedReceiptsE2E(t *testing.T) {
	if os.Getenv("PROCUREMENT_E2E_ISOLATED") != "1" {
		t.Skip("run scripts/run-receiving-integration.ps1 for isolated DB/broker coverage")
	}
	dsn, broker, report := os.Getenv("PROCUREMENT_E2E_DATABASE_URL"), os.Getenv("PROCUREMENT_E2E_KAFKA_BROKERS"), os.Getenv("PROCUREMENT_E2E_REPORT")
	if dsn == "" || broker == "" || report == "" {
		t.Fatal("isolated test DB, broker and report path are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "proc_receiving_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = migrations.Apply(ctx, db); err != nil {
		t.Fatal(err)
	}
	if applied, err := migrations.Apply(ctx, db); err != nil || len(applied) != 0 {
		t.Fatalf("migration restart: %v %v", applied, err)
	}
	repo := postgres.New(db)
	service := application.NewService(repo)
	supplier, err := domain.NewSupplier("E2E-SUPPLIER", "Synthetic receiving supplier", "COMPANY", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	supplier, err = service.CreateSupplier(ctx, supplier)
	if err != nil {
		t.Fatal(err)
	}
	material, err := domain.NewRawMaterial("PROC-E2E-KG", "Synthetic sesame", "KG", "test only")
	if err != nil {
		t.Fatal(err)
	}
	material, err = service.CreateMaterial(ctx, material)
	if err != nil {
		t.Fatal(err)
	}
	offer, err := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 10000, "VND", "0.01", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateOffer(ctx, offer); err != nil {
		t.Fatal(err)
	}
	po, err := service.CreatePurchaseOrderWithIdempotency(ctx, application.PurchaseOrderRequest{SupplierID: supplier.ID,
		Lines: []application.PurchaseOrderLineRequest{{MaterialID: material.ID, Quantity: "10"}}}, "buyer", "receiving-e2e-po-0001")
	if err != nil {
		t.Fatal(err)
	}
	// Fail after PO/history writes, precisely at the durable command boundary.
	if _, err := db.Exec(ctx, `CREATE FUNCTION reject_test_po_action() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected PO action failure'; END $$;
CREATE TRIGGER reject_test_po_action BEFORE INSERT ON purchase_order_action_results FOR EACH ROW EXECUTE FUNCTION reject_test_po_action()`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "buyer", "e2e-po-submit-0001", "SUBMIT", po.Version, ""); err == nil || !strings.Contains(err.Error(), "injected PO action failure") {
		t.Fatalf("PO command fault not observed: %v", err)
	}
	afterFailure, err := repo.GetPurchaseOrder(ctx, po.ID)
	if err != nil || afterFailure.Version != po.Version || afterFailure.Status != domain.PurchaseOrderDraft {
		t.Fatalf("PO command did not roll back: %v", err)
	}
	var commands int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_action_results`).Scan(&commands); err != nil || commands != 0 {
		t.Fatalf("failed command consumed key: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action='SUBMITTED'`, po.ID).Scan(&commands); err != nil || commands != 0 {
		t.Fatalf("PO command history leaked: %v", err)
	}
	if _, err := db.Exec(ctx, `DROP TRIGGER reject_test_po_action ON purchase_order_action_results; DROP FUNCTION reject_test_po_action()`); err != nil {
		t.Fatal(err)
	}
	type actionResult struct {
		order domain.PurchaseOrder
		err   error
	}
	concurrentAction := func(current domain.PurchaseOrder, actor, key, action, reason string) domain.PurchaseOrder {
		t.Helper()
		results := make(chan actionResult, 2)
		start := make(chan struct{})
		for i := 0; i < 2; i++ {
			go func() {
				<-start
				order, err := application.NewService(postgres.New(db)).ExecutePurchaseOrderAction(ctx, current.ID, actor, key, action, current.Version, reason)
				results <- actionResult{order, err}
			}()
		}
		close(start)
		var committed domain.PurchaseOrder
		for i := 0; i < 2; i++ {
			got := <-results
			if got.err != nil || got.order.Version != current.Version+1 {
				t.Fatalf("concurrent PO action: %v version=%d", got.err, got.order.Version)
			}
			committed = got.order
		}
		replay, err := application.NewService(postgres.New(db)).ExecutePurchaseOrderAction(ctx, current.ID, actor, key, action, current.Version, reason)
		if err != nil || replay.Version != committed.Version {
			t.Fatalf("PO command restart replay: %v", err)
		}
		if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_action_results WHERE actor_id=$1 AND idempotency_key=$2`, actor, key).Scan(&commands); err != nil || commands != 1 {
			t.Fatalf("PO command duplicate: %v count=%d", err, commands)
		}
		historyAction := map[string]string{"SUBMIT": "SUBMITTED", "APPROVE": "APPROVED", "REJECT": "REJECTED", "SEND": "SENT", "ACKNOWLEDGE": "SUPPLIER_ACKNOWLEDGED", "REQUEST_CHANGE": "SUPPLIER_CHANGE_REQUESTED", "DECLINE": "SUPPLIER_DECLINED", "CANCEL": "CANCELLED", "CLOSE": "CLOSED"}[action]
		if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action=$2`, current.ID, historyAction).Scan(&commands); err != nil || commands != 1 {
			t.Fatalf("PO command history duplicate: %v count=%d", err, commands)
		}
		return committed
	}
	initialVersion := po.Version
	po = concurrentAction(po, "buyer", "e2e-po-submit-0001", "SUBMIT", "")
	if _, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "buyer", "e2e-po-self-approve-0001", "APPROVE", po.Version, ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("self approval accepted: %v", err)
	}
	po = concurrentAction(po, "approver", "e2e-po-approve-0001", "APPROVE", "")
	po = concurrentAction(po, "sender", "e2e-po-send-0001", "SEND", "")
	oldSubmit, err := service.ExecutePurchaseOrderAction(ctx, po.ID, "buyer", "e2e-po-submit-0001", "SUBMIT", initialVersion, "")
	if err != nil || oldSubmit.Status != domain.PurchaseOrderPendingApproval || oldSubmit.Version != initialVersion+1 {
		t.Fatalf("old PO result was not frozen: %v", err)
	}
	// Two different POs race on one actor/key. Only the winning PO may change;
	// the other's already-attempted UPDATE/history must be rolled back.
	poRequest := application.PurchaseOrderRequest{SupplierID: supplier.ID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material.ID, Quantity: "10"}}}
	var racing []domain.PurchaseOrder
	for _, key := range []string{"e2e-racing-po-create-0001", "e2e-racing-po-create-0002"} {
		order, err := service.CreatePurchaseOrderWithIdempotency(ctx, poRequest, "race-buyer", key)
		if err != nil {
			t.Fatal(err)
		}
		racing = append(racing, order)
	}
	results := make(chan actionResult, 2)
	start := make(chan struct{})
	for _, order := range racing {
		go func(current domain.PurchaseOrder) {
			<-start
			r, e := application.NewService(postgres.New(db)).ExecutePurchaseOrderAction(ctx, current.ID, "race-buyer", "e2e-shared-po-action-0001", "SUBMIT", current.Version, "")
			results <- actionResult{r, e}
		}(order)
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
		t.Fatalf("cross-resource key race: wins=%d conflicts=%d", wins, conflicts)
	}
	changed := 0
	for _, order := range racing {
		current, err := repo.GetPurchaseOrder(ctx, order.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == domain.PurchaseOrderPendingApproval {
			changed++
		} else {
			if current.Status != domain.PurchaseOrderDraft || current.Version != order.Version {
				t.Fatal("losing PO changed")
			}
			if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_status_history WHERE purchase_order_id=$1 AND action='SUBMITTED'`, current.ID).Scan(&commands); err != nil || commands != 0 {
				t.Fatal("losing PO history leaked")
			}
		}
	}
	if changed != 1 {
		t.Fatal("cross-resource race mutated both POs")
	}
	// Every new command must roll back business state if its journal insert fails,
	// then allow the same key to succeed concurrently after the fault is removed.
	for _, action := range []string{"ACKNOWLEDGE", "REQUEST_CHANGE", "DECLINE", "CANCEL", "CLOSE"} {
		p, err := service.CreatePurchaseOrder(ctx, poRequest, "terminal-buyer")
		if err != nil {
			t.Fatal(err)
		}
		if action != "CANCEL" {
			if p, err = service.SubmitPurchaseOrder(ctx, p.ID, "terminal-buyer"); err != nil {
				t.Fatal(err)
			}
			if p, err = service.ApprovePurchaseOrder(ctx, p.ID, "checker"); err != nil {
				t.Fatal(err)
			}
			if p, err = service.MarkPurchaseOrderSent(ctx, p.ID, "terminal-buyer"); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.Exec(ctx, `CREATE FUNCTION reject_test_po_action() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected PO action failure'; END $$;
CREATE TRIGGER reject_test_po_action BEFORE INSERT ON purchase_order_action_results FOR EACH ROW EXECUTE FUNCTION reject_test_po_action()`); err != nil {
			t.Fatal(err)
		}
		key := "terminal-rollback-" + action + "-0001"
		if _, err := service.ExecutePurchaseOrderAction(ctx, p.ID, "terminal-buyer", key, action, p.Version, "Synthetic evidence"); err == nil || !strings.Contains(err.Error(), "injected PO action failure") {
			t.Fatalf("%s journal fault: %v", action, err)
		}
		stored, err := repo.GetPurchaseOrder(ctx, p.ID)
		if err != nil || stored.Status != p.Status || stored.Version != p.Version {
			t.Fatalf("%s leaked PO state: %v", action, err)
		}
		if err := db.QueryRow(ctx, `SELECT count(*) FROM purchase_order_action_results WHERE actor_id='terminal-buyer' AND idempotency_key=$1`, key).Scan(&commands); err != nil || commands != 0 {
			t.Fatal("failed command consumed key")
		}
		if _, err := db.Exec(ctx, `DROP TRIGGER reject_test_po_action ON purchase_order_action_results; DROP FUNCTION reject_test_po_action()`); err != nil {
			t.Fatal(err)
		}
		concurrentAction(p, "terminal-buyer", key, action, "Synthetic evidence")
	}
	// The loser is still a draft; prove concurrent rejection on another submitted PO.
	for _, order := range racing {
		current, err := repo.GetPurchaseOrder(ctx, order.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == domain.PurchaseOrderPendingApproval {
			concurrentAction(current, "rejector", "e2e-po-reject-0001", "REJECT", "Synthetic rejection")
		}
	}
	assertPORevisionE2E(t, ctx, db, repo, service, supplier.ID, material.ID, offer.ID)
	create := func(quantity, key string) domain.GoodsReceipt {
		t.Helper()
		request := application.GoodsReceiptRequest{PurchaseOrderID: po.ID, WarehouseCode: "WH-HUE",
			Lines: []application.GoodsReceiptLineRequest{{PurchaseOrderLineID: po.Lines[0].ID, QuantityReceived: quantity}}}
		r, err := service.CreateGoodsReceipt(ctx, request, "receiver", key)
		if err != nil {
			t.Fatal(err)
		}
		retry, err := service.CreateGoodsReceipt(ctx, request, "receiver", key)
		if err != nil || retry.ID != r.ID {
			t.Fatalf("receipt idempotency: %v", err)
		}
		return r
	}
	for _, tc := range []struct{ quantity, accepted, rejected, key string }{
		{"2.5", "2", "0.5", "receiving-e2e-partial-0001"}, {"1", "0", "1", "receiving-e2e-rejected-0001"},
	} {
		r := create(tc.quantity, tc.key)
		lines := []domain.GoodsReceiptInspectionLine{{
			GoodsReceiptLineID: r.Lines[0].ID, QuantityAccepted: tc.accepted, QuantityRejected: tc.rejected, InspectionNotes: "Synthetic quality rejection",
		}}
		type result struct {
			receipt domain.GoodsReceipt
			err     error
		}
		results := make(chan result, 2)
		start := make(chan struct{})
		for i := 0; i < 2; i++ {
			go func() {
				<-start
				receipt, err := application.NewService(postgres.New(db)).InspectGoodsReceiptWithIdempotency(ctx, r.ID, "inspector", "inspect-"+tc.key, lines)
				results <- result{receipt, err}
			}()
		}
		close(start)
		for i := 0; i < 2; i++ {
			got := <-results
			if got.err != nil || got.receipt.Version != r.Version+1 || got.receipt.InspectionIdempotencyKey != "inspect-"+tc.key {
				t.Fatalf("concurrent inspection replay: %v version=%d", got.err, got.receipt.Version)
			}
		}
		// A fresh service/repository instance must replay from PostgreSQL evidence.
		if _, err := application.NewService(postgres.New(db)).InspectGoodsReceiptWithIdempotency(ctx, r.ID, "inspector", "inspect-"+tc.key, lines); err != nil {
			t.Fatal(err)
		}
		var inspections int
		if err := db.QueryRow(ctx, "SELECT count(*) FROM goods_receipt_status_history WHERE goods_receipt_id=$1 AND action='INSPECTED'", r.ID).Scan(&inspections); err != nil || inspections != 1 {
			t.Fatalf("inspection replay added history: count=%d err=%v", inspections, err)
		}
	}
	// Inject a failure at the outbox boundary. Receipt, PO and history writes must
	// all roll back, rather than publishing a quantity from an uncommitted inspection.
	r := create("1", "receiving-e2e-rollback-0001")
	before, err := repo.GetPurchaseOrder(ctx, po.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `CREATE FUNCTION reject_test_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected outbox failure'; END $$;
CREATE TRIGGER reject_test_outbox BEFORE INSERT ON goods_received_outbox FOR EACH ROW EXECUTE FUNCTION reject_test_outbox()`); err != nil {
		t.Fatal(err)
	}
	_, err = service.InspectGoodsReceiptWithIdempotency(ctx, r.ID, "inspector", "inspect-rollback-00001", []domain.GoodsReceiptInspectionLine{{GoodsReceiptLineID: r.Lines[0].ID, QuantityAccepted: "1", QuantityRejected: "0"}})
	if err == nil || !strings.Contains(err.Error(), "injected outbox failure") {
		t.Fatalf("fault was not observed: %v", err)
	}
	after, err := repo.GetGoodsReceipt(ctx, r.ID)
	if err != nil || after.Status != domain.GoodsReceiptDraft || after.Version != r.Version || after.InspectionIdempotencyKey != "" || after.InspectionRequestHash != "" {
		t.Fatalf("receipt did not roll back: %v", err)
	}
	poAfter, err := repo.GetPurchaseOrder(ctx, po.ID)
	if err != nil || poAfter.Version != before.Version || poAfter.Lines[0].ReceivedQuantity != before.Lines[0].ReceivedQuantity {
		t.Fatalf("PO did not roll back: %v", err)
	}
	var count int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM goods_received_outbox").Scan(&count); err != nil || count != 2 {
		t.Fatalf("unexpected outbox count %d: %v", count, err)
	}
	if err := db.QueryRow(ctx, "SELECT count(*) FROM goods_receipt_status_history WHERE goods_receipt_id=$1 AND action='INSPECTED'", r.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("inspection history leaked: %v", err)
	}
	if _, err := db.Exec(ctx, "DROP TRIGGER reject_test_outbox ON goods_received_outbox; DROP FUNCTION reject_test_outbox()"); err != nil {
		t.Fatal(err)
	}
	// Bypass the application preflight to reproduce a stale close decision: the
	// repository must still reject open receipt variances under the PO row lock.
	current, err := repo.GetPurchaseOrder(ctx, po.ID)
	if err != nil {
		t.Fatal(err)
	}
	closing := current
	if err := closing.CloseRemaining("buyer", "Close remainder"); err != nil {
		t.Fatal(err)
	}
	cmd := domain.PurchaseOrderActionCommand{Kind: "CLOSE", HistoryAction: "CLOSED", Actor: "buyer", Key: "close-open-variance-0001", ExpectedVersion: current.Version, Reason: "Close remainder", RequestHash: strings.Repeat("a", 64)}
	if _, err := repo.SavePurchaseOrderAction(ctx, closing, current.Status, cmd); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("repository closed over open variance: %v", err)
	}
	if got, err := repo.GetPurchaseOrder(ctx, po.ID); err != nil || got.Status != current.Status || got.Version != current.Version {
		t.Fatal("blocked close changed PO")
	}
	receiptRows, err := db.Query(ctx, `SELECT id FROM goods_receipts WHERE purchase_order_id=$1 AND status='INSPECTED'`, po.ID)
	if err != nil {
		t.Fatal(err)
	}
	var receiptIDs []uuid.UUID
	for receiptRows.Next() {
		var id uuid.UUID
		if err := receiptRows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		receiptIDs = append(receiptIDs, id)
	}
	if err := receiptRows.Err(); err != nil {
		t.Fatal(err)
	}
	receiptRows.Close()
	for _, id := range receiptIDs {
		receipt, err := repo.GetGoodsReceipt(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		for _, variance := range receipt.Variances {
			if _, err := service.ResolveGoodsReceiptVariance(ctx, id, variance.ID, "REJECT_VARIANCE", "variance-operator", "Keep rejected stock quarantined"); err != nil {
				t.Fatal(err)
			}
		}
	}
	current, err = repo.GetPurchaseOrder(ctx, po.ID)
	if err != nil {
		t.Fatal(err)
	}
	closed := concurrentAction(current, "buyer", "close-remainder-e2e-0001", "CLOSE", "Close remainder")
	if closed.Lines[0].ReceivedQuantity != current.Lines[0].ReceivedQuantity || closed.Lines[0].AcceptedQuantity != current.Lines[0].AcceptedQuantity {
		t.Fatal("close rewrote receiving quantities")
	}
	if _, err := service.CreateGoodsReceipt(ctx, application.GoodsReceiptRequest{PurchaseOrderID: po.ID, WarehouseCode: "WH-HUE", Lines: []application.GoodsReceiptLineRequest{{PurchaseOrderLineID: po.Lines[0].ID, QuantityReceived: "1"}}}, "receiver", "receipt-after-close-0001"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("receipt after close: %v", err)
	}
	pub, err := producer.NewProducer(strings.Split(broker, ","))
	if err != nil {
		t.Fatal(err)
	}
	defer pub.Close()
	dispatch := worker.NewGoodsReceivedOutboxWorker(repo, pub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 0; i < 2; i++ {
		if found, err := dispatch.DispatchOne(ctx); err != nil || !found {
			t.Fatalf("dispatch %d: %v", i, err)
		}
	}
	if err := db.QueryRow(ctx, "SELECT count(*) FROM goods_received_outbox WHERE published_at IS NOT NULL").Scan(&count); err != nil || count != 2 {
		t.Fatalf("publication acknowledgement: %v", err)
	}
	rows, err := db.Query(ctx, "SELECT payload FROM goods_received_outbox ORDER BY event_id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var events []json.RawMessage
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			t.Fatal(err)
		}
		events = append(events, json.RawMessage(payload))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	bytes, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(report, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	if found, err := dispatch.DispatchOne(ctx); err != nil || found {
		t.Fatalf("published rows were reclaimed: %v", err)
	}
	if _, err := service.InspectGoodsReceipt(ctx, r.ID, "inspector", nil); err == nil {
		t.Fatalf("invalid inspection accepted: %v", err)
	}
	assertVarianceResolutionE2E(t, ctx, db, repo, service, supplier.ID, material.ID)
	assertMasterDataCreateE2E(t, ctx, db)
	assertMasterDataMutationE2E(t, ctx, db)
	assertSupplierContractE2E(t, ctx, db)
}
