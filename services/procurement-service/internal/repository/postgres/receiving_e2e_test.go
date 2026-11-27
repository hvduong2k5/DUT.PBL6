package postgres_test

import (
	"context"
	"encoding/json"
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
	if _, err = service.SubmitPurchaseOrder(ctx, po.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ApprovePurchaseOrder(ctx, po.ID, "approver"); err != nil {
		t.Fatal(err)
	}
	po, err = service.MarkPurchaseOrderSent(ctx, po.ID, "sender")
	if err != nil {
		t.Fatal(err)
	}
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
}
