package grpc

import (
	"context"
	"crypto/sha256"
	"database/sql"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	packing "dut-pbl6/inventory-service/pkg/proto/packing/v1"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Uses only an explicitly supplied isolated test database, never DATABASE_URL.
func TestTerminalPostgresLifecycle(t *testing.T) {
	dsn := os.Getenv("INVENTORY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("INVENTORY_TEST_DATABASE_URL required for real PostgreSQL concurrency evidence")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema := "terminal_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = db.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	_, err = db.Exec("SET search_path TO " + schema + ", public")
	require.NoError(t, err)
	for _, file := range []string{"000001_init_inventory_schema.up.sql", "000002_terminal_profile.up.sql", "000003_packing_reads.up.sql", "000004_stock_projection.up.sql"} {
		b, e := os.ReadFile(filepath.Join("../../../migrations", file))
		require.NoError(t, e)
		_, e = db.Exec(string(b))
		require.NoError(t, e)
	}
	warehouse := "00000000-0000-0000-0000-000000000001"
	_, err = db.Exec(`INSERT INTO inventory_items(sku,warehouse_id,physical_qty) VALUES('A',$1,10);`, warehouse)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,status) VALUES($1,'LOT','A',$2,CURRENT_DATE-1,CURRENT_DATE+30,10,'ACTIVE')`, uuid.NewString(), uuid.NewString())
	require.NoError(t, err)
	s := &TerminalServer{DB: db, Token: "secret"}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	items := []map[string]any{{"quantity": 2, "sku_code": "A"}}
	b, _ := json.Marshal(items)
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	r := &inventoryv1.ReserveStockRequest{OrderId: "order", ContractVersion: "order-3.2", TtlMinutes: 15, PreferredWarehouseId: &warehouse, ItemsDigest: digest, Items: []*inventoryv1.ReserveItem{{SkuCode: "A", Quantity: 2}}, IdempotencyKey: "reserve-key-0001"}
	r.RequestHash = requestHash(r.ProtoReflect())
	out, err := s.ReserveStock(ctx, r)
	require.NoError(t, err)
	require.Equal(t, "RESERVED", out.Outcome.State)
	c := &inventoryv1.ReservationCommand{OrderId: "order", ContractVersion: "order-3.2", IdempotencyKey: "finalize-key-001"}
	c.RequestHash = requestHash(c.ProtoReflect())
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.FinalizeReservation(ctx, c); errs <- e }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	var physical, reserved int
	require.NoError(t, db.QueryRow(`SELECT physical_qty,reserved_qty FROM inventory_items WHERE sku='A'`).Scan(&physical, &reserved))
	require.Equal(t, 8, physical)
	require.Zero(t, reserved)
	// A separate, read-only Fulfillment capability proves committed lot coverage.
	packingToken := strings.Repeat("p", 40)
	packer := &PackingServer{Terminal: s, Token: packingToken}
	packingCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+packingToken))
	read := &packing.GetCommittedPackingAllocationRequest{OrderId: "order", ReservationId: out.Outcome.ReservationId, ReadyGeneration: 1, ItemsDigest: digest, WarehouseId: warehouse, ContractVersion: "order-3.2", Lines: []*packing.PackingLine{{OrderLineId: uuid.NewString(), SkuCode: "A", Quantity: 2}}}
	_, err = packer.GetCommittedPackingAllocation(ctx, read)
	require.Error(t, err, "Order stock credential must not authorize packing reads")
	allocation, err := packer.GetCommittedPackingAllocation(packingCtx, read)
	require.NoError(t, err)
	require.NotEmpty(t, allocation.OriginalCommitDecisionId)
	require.Len(t, allocation.Slots, 1)
	slot := allocation.Slots[0]
	validation := &packing.ValidatePackingAllocationRequest{OrderId: "order", TaskId: uuid.NewString(), ReservationId: read.ReservationId, ReadyGeneration: 1, ItemsDigest: digest, WarehouseId: warehouse, ContractVersion: "order-3.2", AllocationVersion: allocation.AllocationVersion, Picks: []*packing.PickedSlot{{AllocationId: slot.AllocationId, OrderLineId: slot.OrderLineId, BatchId: slot.BatchId, CountedQuantity: 2}}}
	checked, err := packer.ValidatePackingAllocation(packingCtx, validation)
	require.NoError(t, err)
	require.True(t, checked.Allowed)
	_, err = db.Exec("UPDATE batches SET status='QUARANTINE'")
	require.NoError(t, err)
	checked, err = packer.ValidatePackingAllocation(packingCtx, validation)
	require.NoError(t, err)
	require.False(t, checked.Allowed)
	require.Greater(t, checked.Slots[0].EligibilityRevision, slot.EligibilityRevision)
	validation.Picks[0].CountedQuantity = 1
	_, err = packer.ValidatePackingAllocation(packingCtx, validation)
	require.Error(t, err, "wrong counted quantity must fail closed")
	require.NoError(t, db.QueryRow(`SELECT physical_qty,reserved_qty FROM inventory_items WHERE sku='A'`).Scan(&physical, &reserved))
	require.Equal(t, 8, physical)
	require.Zero(t, reserved)
	// Release after commit must carry REJECTED evidence, never restore physical stock.
	c.IdempotencyKey = "release-key-0001"
	c.RequestHash = requestHash(c.ProtoReflect())
	rejected, err := s.ReleaseReservationByOrder(ctx, c)
	require.NoError(t, err)
	require.Equal(t, "REJECTED", rejected.Command.Result)
	// Cancel-before-reserve survives replay and blocks late reservation.
	c.OrderId = "cancelled"
	c.IdempotencyKey = "cancel-key-00001"
	c.RequestHash = requestHash(c.ProtoReflect())
	tomb, err := s.ReleaseReservationByOrder(ctx, c)
	require.NoError(t, err)
	require.True(t, tomb.CancellationTombstone)
	r.OrderId = "cancelled"
	r.RequestHash = requestHash(r.ProtoReflect())
	_, err = s.ReserveStock(ctx, r)
	require.Error(t, err)
	// TTL is based on the database clock, not a client-supplied timestamp.
	require.True(t, out.ExpiresAt.AsTime().After(time.Now()))
	// Same rule at RPC read time and in the snapshot, including wall-clock expiry.
	_, err = db.Exec("UPDATE batches SET status='ACTIVE' WHERE sku='A'")
	require.NoError(t, err)
	s.CatalogToken = "catalog-read-credential-32-bytes-long"
	readCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+s.CatalogToken))
	levels, e := s.GetStockLevel(readCtx, &inventoryv1.GetStockLevelRequest{SkuCodes: []string{"A"}})
	require.NoError(t, e)
	require.Equal(t, int32(8), levels.Items[0].AvailableQuantity)
	_, e = db.Exec("UPDATE batches SET status='QUARANTINE' WHERE sku='A'")
	require.NoError(t, e)
	levels, e = s.GetStockLevel(readCtx, &inventoryv1.GetStockLevelRequest{SkuCodes: []string{"A"}})
	require.NoError(t, e)
	require.Zero(t, levels.Items[0].AvailableQuantity)
	require.Equal(t, int32(8), levels.Items[0].PhysicalQuantity)
	var snapshotQuantity int
	require.NoError(t, db.QueryRow(`SELECT (payload->>'sellable_quantity')::int FROM outbox_events WHERE event_type='inventory.stock-changed.v1' AND aggregate_id='A' ORDER BY (payload->>'inventory_version')::bigint DESC LIMIT 1`).Scan(&snapshotQuantity))
	require.Zero(t, snapshotQuantity)
	_, e = db.Exec("UPDATE batches SET status='ACTIVE', exp_date=CURRENT_DATE WHERE sku='A'")
	require.NoError(t, e)
	levels, e = s.GetStockLevel(readCtx, &inventoryv1.GetStockLevelRequest{SkuCodes: []string{"A"}})
	require.NoError(t, e)
	require.Zero(t, levels.Items[0].AvailableQuantity)
	_, e = s.FinalizeReservation(readCtx, c)
	require.Equal(t, codes.Unauthenticated, status.Code(e))

}
