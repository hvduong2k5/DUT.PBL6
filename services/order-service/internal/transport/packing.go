package transport

import (
	"context"
	"crypto/subtle"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain/core"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	order "github.com/omamx/order-service/internal/gen/order/v1"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	"github.com/omamx/order-service/internal/persistence"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type PackingRPC struct {
	packing.UnimplementedOrderPackingInputsCandidateServer
	S *RPC
}

func packingError(s *app.Service, e error) error {
	switch {
	case errors.Is(e, core.ErrBlocked):
		return status.Error(codes.FailedPrecondition, "ORDER_NOT_READY_FOR_PACKING")
	case errors.Is(e, persistence.ErrInput):
		return status.Error(codes.InvalidArgument, "INVALID_PACKING_REQUEST")
	case errors.Is(e, persistence.ErrScope), errors.Is(e, persistence.ErrIntegrity):
		return status.Error(codes.FailedPrecondition, "PACKING_PROOF_MISMATCH")
	default:
		return rpcError(s, e)
	}
}

func (r *RPC) fulfillmentCaller(ctx context.Context) error {
	if r.S.Fulfillment == nil || len(r.S.Fulfillment.CallerToken) < 32 {
		return status.Error(codes.Unavailable, "FULFILLMENT_CAPABILITY_UNAVAILABLE")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	v := md.Get("authorization")
	if len(v) != 1 || subtle.ConstantTimeCompare([]byte(v[0]), []byte("Bearer "+r.S.Fulfillment.CallerToken)) != 1 {
		return status.Error(codes.Unauthenticated, "FULFILLMENT_IDENTITY_REQUIRED")
	}
	return nil
}
func permitWire(p persistence.PackingPermit) *common.FulfillmentPermit {
	return &common.FulfillmentPermit{PermitId: p.ID, OrderId: p.Order, Generation: p.Generation, ItemsDigest: p.Digest, WarehouseId: p.Warehouse, OrderVersion: p.OrderVersion, PolicyVersion: p.Policy, ExpiresAt: timestamppb.New(p.Expires), TaskRequestKey: p.Key, Audience: "fulfillment-service"}
}
func (r *RPC) GetFulfillmentPermit(ctx context.Context, in *order.GetFulfillmentPermitRequest) (*order.GetFulfillmentPermitResponse, error) {
	if e := r.fulfillmentCaller(ctx); e != nil {
		return nil, e
	}
	if in == nil || in.ContractVersion != "order-3.2" || in.ItemsDigest == "" {
		return nil, status.Error(codes.InvalidArgument, "INVALID_PACKING_REQUEST")
	}
	_, p, e := r.S.Payment.Store.PackingInputs(ctx, in.OrderId, in.Generation, in.ItemsDigest, in.TaskRequestKey, r.S.Fulfillment.Policy.PolicyVersion, nil)
	if e != nil {
		return nil, packingError(r.S, e)
	}
	return &order.GetFulfillmentPermitResponse{Allowed: true, Permit: permitWire(p), ContractVersion: "order-3.2"}, nil
}
func (r *RPC) ValidateFulfillmentPermit(ctx context.Context, in *order.ValidateFulfillmentPermitRequest) (*order.GetFulfillmentPermitResponse, error) {
	if e := r.fulfillmentCaller(ctx); e != nil {
		return nil, e
	}
	if in == nil || in.ContractVersion != "order-3.2" || in.Permit == nil || in.Permit.ExpiresAt == nil || in.Permit.ExpiresAt.CheckValid() != nil || in.Permit.Audience != "fulfillment-service" {
		return nil, status.Error(codes.InvalidArgument, "INVALID_PACKING_PERMIT")
	}
	x := in.Permit
	p := persistence.PackingPermit{ID: x.PermitId, Order: x.OrderId, Generation: x.Generation, Digest: x.ItemsDigest, Warehouse: x.WarehouseId, OrderVersion: x.OrderVersion, Policy: x.PolicyVersion, Key: x.TaskRequestKey, Expires: x.ExpiresAt.AsTime()}
	_, out, e := r.S.Payment.Store.PackingInputs(ctx, x.OrderId, x.Generation, x.ItemsDigest, x.TaskRequestKey, x.PolicyVersion, &p)
	if e != nil {
		return nil, packingError(r.S, e)
	}
	return &order.GetFulfillmentPermitResponse{Allowed: true, Permit: permitWire(out), ContractVersion: "order-3.2"}, nil
}
func (r *PackingRPC) GetPackingSnapshot(ctx context.Context, in *packing.GetPackingSnapshotRequest) (*packing.PackingSnapshot, error) {
	if e := r.S.fulfillmentCaller(ctx); e != nil {
		return nil, e
	}
	if in == nil || in.ContractVersion != "order-3.2" {
		return nil, status.Error(codes.InvalidArgument, "UNSUPPORTED_CONTRACT")
	}
	record, _, e := r.S.S.Payment.Store.PackingInputs(ctx, in.OrderId, in.ReadyGeneration, "", "", "", nil)
	if e != nil {
		return nil, packingError(r.S.S, e)
	}
	s := record.Snapshot
	policy := proto.Clone(r.S.S.Fulfillment.Policy).(*packing.PackingPolicy)
	snapshotID := uuid.NewSHA1(uuid.MustParse(s.SnapshotID), []byte(policy.PolicyVersion)).String()
	out := &packing.PackingSnapshot{SchemaVersion: "packing-inputs.v1", ContractVersion: "order-3.2", SnapshotId: snapshotID, SnapshotVersion: 1, OrderId: in.OrderId, ReadyGeneration: record.Generation, ItemsDigest: string(s.Provenance.ItemsDigest), WarehouseId: s.WarehouseID, ReservationId: record.Stock.ReservationID, Channel: s.Channel, PackingDueAt: timestamppb.New(s.PlacedAt.Add(24 * time.Hour)), DeadlineRevision: 1, Policy: policy}
	for _, line := range s.Items {
		out.Lines = append(out.Lines, &packing.PackingLine{OrderLineId: line.LineID, SkuCode: line.SKU, ProductName: line.Name, PackagingText: line.Packaging.Description, Quantity: int32(line.Quantity)})
	}
	return out, nil
}
