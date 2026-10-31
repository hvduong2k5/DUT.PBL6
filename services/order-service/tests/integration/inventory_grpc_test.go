package integration

import (
	"github.com/google/uuid"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestV10InventoryTerminalRPCsRegisteredAndIdempotent(t *testing.T) {
	h := newHarness(t)
	conn, e := grpc.DialContext(h.ctx, h.MockGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	client := inventory.NewInventoryServiceClient(conn)
	ctx := metadata.NewOutgoingContext(h.ctx, metadata.Pairs("x-internal-token", internalToken))
	id := uuid.NewString()
	req := &inventory.ReserveStockRequest{OrderId: id, IdempotencyKey: uuid.NewString(), TtlMinutes: 15, Items: []*inventory.ReserveItem{{SkuCode: "MX-GION-500G", Quantity: 2}}}
	if _, e = client.ReserveStock(h.ctx, req); e == nil {
		t.Fatal("unauthenticated gRPC call accepted")
	}
	first, e := client.ReserveStock(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	second, e := client.ReserveStock(ctx, req)
	if e != nil || first.GetReservationId() != second.GetReservationId() {
		t.Fatal("RPC reserve duplicated")
	}
	command := &inventory.ReservationCommand{OrderId: id, IdempotencyKey: uuid.NewString()}
	committed, e := client.FinalizeReservation(ctx, command)
	if e != nil || committed.State != "COMMITTED" {
		t.Fatal("terminal finalize failed")
	}
	again, e := client.FinalizeReservation(ctx, command)
	if e != nil || again.State != "COMMITTED" || h.mockCount("SELECT physical FROM stock WHERE sku='MX-GION-500G'") != 98 {
		t.Fatal("finalize deducted twice")
	}
	released, e := client.ReleaseReservationByOrder(ctx, &inventory.ReservationCommand{OrderId: id, IdempotencyKey: uuid.NewString()})
	if e != nil || released.State != "COMMITTED" {
		t.Fatal("release changed committed stock")
	}
}
