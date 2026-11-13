package paymentinventory

import (
	"context"
	"github.com/omamx/order-service/internal/domain/core"
	api "github.com/omamx/order-service/internal/gen/inventory/v1"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestTerminalInputAndTransportGuards(t *testing.T) {
	s := core.SnapshotInput{OrderID: "order", WarehouseID: "warehouse", Items: []core.SnapshotItem{{SKU: "B", Quantity: 1}, {SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 1}}}
	s.Provenance.ItemsDigest, _ = core.HashManifest([]core.SelectionLine{{SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 2}})
	r, e := terminalRequest(s, "test-command-key-01")
	if e != nil || len(r.Items) != 2 || r.Items[0].SkuCode != "A" || r.Items[1].Quantity != 2 {
		t.Fatal(r, e)
	}
	hash := r.RequestHash
	r.CorrelationId = "another"
	r.IdempotencyKey = "another-key"
	r.RequestHash = "ignored"
	if RequestHash(r.ProtoReflect()) != hash {
		t.Fatal("transport metadata changed business hash")
	}
	r.Items[0].Quantity++
	if RequestHash(r.ProtoReflect()) == hash {
		t.Fatal("quantity not bound")
	}
	s.Items = []core.SnapshotItem{{SKU: "A", Quantity: math.MaxInt32 + 1}}
	if _, e = terminalRequest(s, "key"); e == nil {
		t.Fatal("int32 overflow accepted")
	}
	s.Items = []core.SnapshotItem{{SKU: "A", Quantity: 1}}
	if _, e = terminalRequest(s, "key"); e == nil {
		t.Fatal("digest mismatch accepted")
	}
	zero := int64(0)
	c := &api.ReservationCommand{}
	before := RequestHash(c.ProtoReflect())
	c.ExpectedVersion = &zero
	if before == RequestHash(c.ProtoReflect()) {
		t.Fatal("optional presence not bound")
	}
	if _, _, e = DialTerminal(context.Background(), "target", "short", "", "localhost"); e == nil {
		t.Fatal("invalid TLS configuration accepted")
	}
	p := filepath.Join(t.TempDir(), "ca.pem")
	os.WriteFile(p, []byte("not a certificate"), 0600)
	if _, _, e = DialTerminal(context.Background(), "target", "long-enough-dedicated-token-000000", p, "localhost"); e == nil {
		t.Fatal("invalid CA accepted")
	}
}
