package paymentinventory

import (
	"context"
	"encoding/json"
	"github.com/omamx/order-service/internal/domain/core"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type inventoryServer struct {
	inventory.UnimplementedInventoryServiceServer
	mu      sync.Mutex
	mode    string
	applied bool
	calls   int
	root    core.Aggregate
}

func (s *inventoryServer) out(r *inventory.ReservationCommand) *inventory.ReservationOutcome {
	p := s.root.Record().Reservation
	state := "RESERVED"
	if s.applied {
		state = "COMMITTED"
	}
	out := &inventory.ReservationOutcome{OrderId: s.root.ID(), ReservationId: p.ReservationID, State: state, ContractVersion: "order-3.2", ItemsDigest: string(p.ItemsDigest), WarehouseId: p.WarehouseID, ResourceVersion: p.ResourceVersion + 1, ExpiresAt: timestamppb.New(p.ExpiresAt), ObservedAt: timestamppb.Now()}
	if s.applied {
		out.Command = &common.CommandEvidence{IdempotencyKey: r.IdempotencyKey, RequestHash: r.RequestHash, Action: "FINALIZE", Result: "APPLIED", ResultingState: state, DecisionId: "test-finalize-decision", DecisionAt: timestamppb.Now()}
	}
	switch s.mode {
	case "legacy":
		out.ContractVersion = "sandbox-3.1"
		out.Command = nil
	case "wronghash":
		if out.Command != nil {
			out.Command.RequestHash = "wrong"
		}
	case "rejectcommit":
		if out.Command != nil {
			out.Command.Result = "REJECTED"
		}
	}
	return out
}
func (s *inventoryServer) GetReservationByOrder(ctx context.Context, r *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("x-internal-token")) != 1 || md.Get("x-internal-token")[0] != "synthetic-internal-token" {
		return nil, status.Error(codes.Unauthenticated, "auth")
	}
	return s.out(r), nil
}
func (s *inventoryServer) FinalizeReservation(ctx context.Context, r *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.applied = true
	if s.mode == "acklost" {
		return nil, status.Error(codes.Unavailable, "ack lost after commit")
	}
	return s.out(r), nil
}
func rootFixture(t *testing.T) core.Aggregate {
	raw, e := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "03_api_specs", "order-snapshot.v1.examples.json"))
	if e != nil {
		t.Fatal(e)
	}
	var file struct {
		Cases []struct {
			Snapshot json.RawMessage `json:"snapshot"`
		} `json:"cases"`
	}
	if e = json.Unmarshal(raw, &file); e != nil {
		t.Fatal(e)
	}
	snap, e := core.DecodeSnapshot(file.Cases[0].Snapshot)
	if e != nil {
		t.Fatal(e)
	}
	d := snap.Data()
	proof := core.ResourceProof{OrderID: d.OrderID, ReservationID: "test-reservation", ItemsDigest: d.Provenance.ItemsDigest, WarehouseID: d.WarehouseID, State: "RESERVED", ResourceVersion: 1, DecisionID: "reserve-decision", DecisionAt: d.PlacedAt.Add(-time.Second), ExpiresAt: d.PlacedAt.Add(15 * time.Minute)}
	a, e := core.Place(snap, d.OrderID, proof, d.PlacedAt)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func TestAuthenticatedGRPCFinalizeAndUnknownRecovery(t *testing.T) {
	for _, mode := range []string{"ok", "legacy", "wronghash", "rejectcommit", "acklost"} {
		t.Run(mode, func(t *testing.T) {
			root := rootFixture(t)
			provider := &inventoryServer{mode: mode, root: root}
			listener, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			server := grpc.NewServer()
			inventory.RegisterInventoryServiceServer(server, provider)
			go server.Serve(listener)
			defer server.Stop()
			conn, e := grpc.Dial(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if e != nil {
				t.Fatal(e)
			}
			defer conn.Close()
			adapter := Adapter{Client: inventory.NewInventoryServiceClient(conn), Token: "synthetic-internal-token"}
			proof, e := adapter.Finalize(context.Background(), root, "inventory:finalize:"+root.ID())
			if mode == "acklost" {
				if e == nil {
					t.Fatal("lost ack reported success")
				}
				proof, e = adapter.Finalize(context.Background(), root, "inventory:finalize:"+root.ID())
			}
			if mode == "ok" || mode == "acklost" {
				if e != nil || proof.State != "COMMITTED" || provider.calls != 1 {
					t.Fatal(proof, e, provider.calls)
				}
			} else if e == nil {
				t.Fatal("invalid inventory proof accepted")
			}
		})
	}
}
