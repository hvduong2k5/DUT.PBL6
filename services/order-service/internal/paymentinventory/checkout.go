package paymentinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/omamx/order-service/internal/domain/core"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"time"
)

func ReserveHash(order, digest, warehouse string) string {
	raw, _ := json.Marshal(struct {
		Action, OrderID, Digest, Warehouse string
		TTL                                int
	}{"RESERVE", order, digest, warehouse, 15})
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func CommandHash(action, order, reservation, digest, warehouse string) string {
	raw, _ := json.Marshal(struct{ Action, OrderID, ReservationID, Digest, Warehouse string }{action, order, reservation, digest, warehouse})
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func (a Adapter) Reserve(ctx context.Context, s core.SnapshotInput, key string) (core.ResourceProof, error) {
	if a.Client == nil || len(a.Token) < 16 {
		return core.ResourceProof{}, ErrProof
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-token", a.Token)
	digest := string(s.Provenance.ItemsDigest)
	requestHash := ReserveHash(s.OrderID, digest, s.WarehouseID)
	query := &inventory.ReservationCommand{OrderId: s.OrderID, IdempotencyKey: key, ContractVersion: "order-3.2", RequestHash: requestHash, ItemsDigest: digest, WarehouseId: s.WarehouseID, CorrelationId: s.OrderID, QueryAction: "RESERVE"}
	out, e := a.Client.GetReservationByOrder(ctx, query)
	if status.Code(e) == codes.NotFound {
		items := []*inventory.ReserveItem{}
		for _, i := range s.Items {
			items = append(items, &inventory.ReserveItem{SkuCode: i.SKU, Quantity: int32(i.Quantity)})
		}
		response, err := a.Client.ReserveStock(ctx, &inventory.ReserveStockRequest{IdempotencyKey: key, OrderId: s.OrderID, Items: items, TtlMinutes: 15, Channel: "D2C_WEB", PreferredWarehouseId: &s.WarehouseID, ContractVersion: "order-3.2", RequestHash: requestHash, ItemsDigest: digest, CorrelationId: s.OrderID})
		if err != nil {
			return core.ResourceProof{}, err
		}
		out = response.GetOutcome()
		e = nil
	}
	if e != nil {
		return core.ResourceProof{}, e
	}
	return checked(out, s, core.ResourceProof{}, key, requestHash, "RESERVE")
}
func checked(out *inventory.ReservationOutcome, s core.SnapshotInput, prior core.ResourceProof, key, requestHash, action string) (core.ResourceProof, error) {
	if out == nil {
		return core.ResourceProof{}, ErrProof
	}
	c := out.Command
	if out.ContractVersion != "order-3.2" || out.OrderId != s.OrderID || out.ReservationId == "" || prior.ReservationID != "" && prior.ReservationID != out.ReservationId || out.ItemsDigest != string(s.Provenance.ItemsDigest) || out.WarehouseId != s.WarehouseID || out.ResourceVersion < 1 || out.ResourceVersion < prior.ResourceVersion || out.ExpiresAt == nil || out.ExpiresAt.CheckValid() != nil || c == nil || c.IdempotencyKey != key || c.RequestHash != requestHash || c.Action != action || c.DecisionId == "" || c.DecisionAt == nil || c.DecisionAt.CheckValid() != nil || c.DecisionAt.AsTime().After(time.Now().Add(time.Minute)) {
		return core.ResourceProof{}, ErrProof
	}
	if c.Result != "APPLIED" && c.Result != "ALREADY_APPLIED" && c.Result != "REJECTED" {
		return core.ResourceProof{}, ErrProof
	}
	if out.State != "RESERVED" && out.State != "COMMITTED" && out.State != "RELEASED" && out.State != "EXPIRED" {
		return core.ResourceProof{}, ErrProof
	}
	if action != "RESERVE" && (c.ResultingState != out.State || out.State == "COMMITTED" && c.Result == "REJECTED") {
		return core.ResourceProof{}, ErrProof
	}
	at := c.DecisionAt.AsTime()
	if out.State == "EXPIRED" && out.ObservedAt != nil && out.ObservedAt.CheckValid() == nil {
		at = out.ObservedAt.AsTime()
	}
	return core.ResourceProof{OrderID: out.OrderId, ReservationID: out.ReservationId, ItemsDigest: core.ManifestHash(out.ItemsDigest), WarehouseID: out.WarehouseId, State: out.State, ResourceVersion: out.ResourceVersion, DecisionID: c.DecisionId, DecisionAt: at, ExpiresAt: out.ExpiresAt.AsTime()}, nil
}
func (a Adapter) Commit(ctx context.Context, s core.SnapshotInput, p core.ResourceProof, key string) (core.ResourceProof, error) {
	return a.command(ctx, s, p, key, "FINALIZE")
}
func (a Adapter) Release(ctx context.Context, s core.SnapshotInput, p core.ResourceProof, key string) (core.ResourceProof, error) {
	return a.command(ctx, s, p, key, "RELEASE")
}
func (a Adapter) command(ctx context.Context, s core.SnapshotInput, p core.ResourceProof, key, action string) (core.ResourceProof, error) {
	if a.Client == nil || len(a.Token) < 16 {
		return core.ResourceProof{}, ErrProof
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-token", a.Token)
	digest := string(s.Provenance.ItemsDigest)
	requestHash := CommandHash(action, s.OrderID, p.ReservationID, digest, s.WarehouseID)
	request := &inventory.ReservationCommand{OrderId: s.OrderID, IdempotencyKey: key, ContractVersion: "order-3.2", RequestHash: requestHash, ExpectedReservationId: &p.ReservationID, ItemsDigest: digest, WarehouseId: s.WarehouseID, ExpectedVersion: &p.ResourceVersion, CorrelationId: s.OrderID, QueryAction: action}
	out, e := a.Client.GetReservationByOrder(ctx, request)
	if e != nil {
		return core.ResourceProof{}, e
	}
	if out.Command != nil && out.Command.IdempotencyKey == key {
		return checked(out, s, p, key, requestHash, action)
	}
	if out.OrderId != s.OrderID || out.ReservationId != p.ReservationID || out.ContractVersion != "order-3.2" || out.ItemsDigest != digest || out.WarehouseId != s.WarehouseID {
		return core.ResourceProof{}, ErrProof
	}
	request.QueryAction = ""
	if action == "FINALIZE" {
		out, e = a.Client.FinalizeReservation(ctx, request)
	} else {
		out, e = a.Client.ReleaseReservationByOrder(ctx, request)
	}
	if e != nil {
		return core.ResourceProof{}, e
	}
	return checked(out, s, p, key, requestHash, action)
}
