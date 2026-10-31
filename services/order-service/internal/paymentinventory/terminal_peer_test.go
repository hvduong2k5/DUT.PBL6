package paymentinventory

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain/core"
	api "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type lostTerminalACK struct {
	api.InventoryServiceClient
	reserve, commit, release bool
}

func (c *lostTerminalACK) ReserveStock(ctx context.Context, r *api.ReserveStockRequest, o ...grpc.CallOption) (*api.ReserveStockResponse, error) {
	out, e := c.InventoryServiceClient.ReserveStock(ctx, r, o...)
	if e == nil && c.reserve {
		c.reserve = false
		return nil, status.Error(codes.Unavailable, "injected lost reserve ACK")
	}
	return out, e
}
func (c *lostTerminalACK) FinalizeReservation(ctx context.Context, r *api.ReservationCommand, o ...grpc.CallOption) (*api.ReservationOutcome, error) {
	out, e := c.InventoryServiceClient.FinalizeReservation(ctx, r, o...)
	if e == nil && c.commit {
		c.commit = false
		return nil, status.Error(codes.Unavailable, "injected lost commit ACK")
	}
	return out, e
}
func (c *lostTerminalACK) ReleaseReservationByOrder(ctx context.Context, r *api.ReservationCommand, o ...grpc.CallOption) (*api.ReservationOutcome, error) {
	out, e := c.InventoryServiceClient.ReleaseReservationByOrder(ctx, r, o...)
	if e == nil && c.release {
		c.release = false
		return nil, status.Error(codes.Unavailable, "injected lost release ACK")
	}
	return out, e
}

func terminalSnapshot(t *testing.T) core.SnapshotInput {
	t.Helper()
	raw, e := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "03_api_specs", "order-snapshot.v1.examples.json"))
	if e != nil {
		t.Fatal(e)
	}
	var examples struct {
		Cases []struct {
			Snapshot core.SnapshotInput `json:"snapshot"`
		} `json:"cases"`
	}
	if e = json.Unmarshal(raw, &examples); e != nil {
		t.Fatal(e)
	}
	s := examples.Cases[0].Snapshot
	s.OrderID = uuid.NewString()
	s.WarehouseID = "00000000-0000-0000-0000-000000000001"
	s.Items = s.Items[:1]
	s.Items[0].SKU = "PEER"
	s.Items[0].Quantity = 2
	s.Items[0].UnitPrice = 1000
	s.Items[0].LineSubtotal = 2000
	s.Money.Subtotal = 2000
	s.Money.FinalAmount = 2000 + s.Money.ShippingFee
	s.Provenance.ItemsDigest, _ = core.HashManifest([]core.SelectionLine{{SKU: "PEER", Quantity: 2}})
	s.Provenance.SelectionDigest, _ = core.HashSelection([]core.SelectionLine{{SKU: "PEER", Quantity: 2}})
	s.PlacedAt = time.Now().UTC().Add(time.Second)
	s.Provenance.AcceptedAt = s.PlacedAt.Add(-time.Second)
	s.Provenance.CanonicalizedAt = s.PlacedAt
	s.Catalog.ValidUntil = s.PlacedAt.Add(10 * time.Minute)
	s.Shipping.ValidUntil = s.PlacedAt.Add(10 * time.Minute)
	expiry := s.PlacedAt.Add(5 * time.Minute)
	s.Payment.ExpiresAt = &expiry
	return s
}

func TestTerminalPeerFlow(t *testing.T) {
	target := os.Getenv("ORDER_INVENTORY_TEST_ENDPOINT")
	if target == "" {
		t.Skip("use scripts/test-order-inventory.py for real TLS/provider/PostgreSQL")
	}
	token := os.Getenv("PEER_TOKEN")
	ca := os.Getenv("PEER_CA_FILE")
	a, conn, e := DialTerminal(context.Background(), target, token, ca, "localhost")
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	t.Run("TLS hostname mismatch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, c, err := DialTerminal(ctx, target, token, ca, "wrong.inventory.internal"); err == nil {
			c.Close()
			t.Fatal("wrong server identity accepted")
		}
	})
	ctx := context.Background()
	t.Run("durable checkout worker", func(t *testing.T) { testTerminalCheckoutWorker(t, a) })
	for _, drop := range []bool{false, true} {
		s := terminalSnapshot(t)
		client := &lostTerminalACK{InventoryServiceClient: a.Client, reserve: drop, commit: drop}
		adapter := a
		adapter.Client = client
		key := "reserve-" + s.OrderID
		p, e := adapter.Reserve(ctx, s, key)
		if drop {
			if status.Code(e) != codes.Unavailable {
				t.Fatal("reserve ACK injection", e)
			}
			p, e = adapter.Lookup(ctx, s, key)
		}
		if e != nil || p.State != "RESERVED" {
			t.Fatal("reserve", e, p)
		}
		again, e := adapter.Reserve(ctx, s, key)
		if e != nil || again.DecisionID != p.DecisionID {
			t.Fatal("reserve replay", e)
		}
		snapshot, e := core.NewSnapshot(s)
		if e != nil {
			t.Fatal("snapshot fixture", e)
		}
		root, e := core.Place(snapshot, s.OrderID, p, s.PlacedAt)
		if e != nil {
			t.Fatal("place", e)
		}
		commandKey := "finalize-" + s.OrderID
		committed, e := adapter.Finalize(ctx, root, commandKey)
		if drop {
			if status.Code(e) != codes.Unavailable {
				t.Fatal("commit ACK injection", e)
			}
			refreshed, err := adapter.Lookup(ctx, s, key)
			if err != nil {
				t.Fatal(err)
			}
			// Refresh observes version 2; replay must still match the original command hash.
			committed, e = adapter.Commit(ctx, s, refreshed, commandKey)
		}
		if e != nil || committed.State != "COMMITTED" {
			t.Fatal("commit", e, committed)
		}
		replay, e := adapter.Commit(ctx, s, committed, commandKey)
		if e != nil || replay.DecisionID != committed.DecisionID {
			t.Fatal("commit replay", e)
		}
		if _, e = adapter.Release(ctx, s, committed, "reject-release-"+s.OrderID); e == nil {
			t.Fatal("committed stock released")
		}
		// Original decision remains recoverable even after another command is rejected.
		if _, e = adapter.Commit(ctx, s, committed, commandKey); e != nil {
			t.Fatal("original decision after rejected release", e)
		}
	}
	s := terminalSnapshot(t)
	p, e := a.Reserve(ctx, s, "reserve-"+s.OrderID)
	if e != nil {
		t.Fatal(e)
	}
	lossy := a
	lossy.Client = &lostTerminalACK{InventoryServiceClient: a.Client, release: true}
	key := "release-" + s.OrderID
	if _, e = lossy.Release(ctx, s, p, key); status.Code(e) != codes.Unavailable {
		t.Fatal("release ACK injection", e)
	}
	released, e := lossy.Release(ctx, s, p, key)
	if e != nil || released.State != "RELEASED" {
		t.Fatal("release recovery", e)
	}
	replay, e := lossy.Release(ctx, s, released, key)
	if e != nil || replay.DecisionID != released.DecisionID {
		t.Fatal("release replay", e)
	}
	bad := a
	bad.Token = "wrong-but-long-enough-credential-0000"
	if _, e = bad.Lookup(ctx, s, "reserve-"+s.OrderID); status.Code(e) != codes.Unauthenticated {
		t.Fatal("wrong token accepted", e)
	}
	s.Items[0].Quantity = 3
	s.Provenance.ItemsDigest, _ = core.HashManifest([]core.SelectionLine{{SKU: "PEER", Quantity: 3}})
	if _, e = a.Reserve(ctx, s, "reserve-"+s.OrderID); e == nil {
		t.Fatal("same key different payload accepted")
	}
	t.Log("PASS real Order adapter -> TLS gRPC Inventory -> PostgreSQL: reserve/lookup/commit/release, lost ACK/replay, wrong credential and hash conflict")
}
