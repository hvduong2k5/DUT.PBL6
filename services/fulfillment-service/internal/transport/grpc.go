package transport

import (
	"context"
	"errors"
	"github.com/omamx/fulfillment-service/internal/application"
	d "github.com/omamx/fulfillment-service/internal/domain"
	c "github.com/omamx/fulfillment-service/internal/gen/common/v1"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	p "github.com/omamx/fulfillment-service/internal/gen/packing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RPC struct {
	f.UnimplementedFulfillmentServiceServer
	p.UnimplementedFulfillmentPackageProofCandidateServer
	Service *application.Service
	Auth    Auth
}

func (a Auth) Interceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("authorization")
	if len(values) != 1 {
		return nil, status.Error(codes.Unauthenticated, "verified service identity required")
	}
	p, e := a.Verify(values[0], "fulfillment-internal")
	if e != nil {
		return nil, rpcError(e)
	}
	return handler(context.WithValue(ctx, principalKey{}, p), req)
}
func rpcError(e error) error {
	if e == nil {
		return nil
	}
	if _, ok := status.FromError(e); ok {
		return e
	}
	var de *d.Error
	if !errors.As(e, &de) {
		return status.Error(codes.Unavailable, "DEPENDENCY_UNAVAILABLE")
	}
	code := codes.FailedPrecondition
	switch de.HTTP {
	case 400:
		code = codes.InvalidArgument
	case 401:
		code = codes.Unauthenticated
	case 403:
		code = codes.PermissionDenied
	case 404:
		code = codes.NotFound
	case 409:
		if de.Code == "IDEMPOTENCY_CONFLICT" || de.Code == "HASH_MISMATCH" {
			code = codes.AlreadyExists
		}
	case 503:
		code = codes.Unavailable
	case 504:
		code = codes.DeadlineExceeded
	}
	return status.Error(code, de.Code)
}
func wire(v d.Decision) *f.TaskDecision {
	out := &f.TaskDecision{OrderId: v.OrderID, Generation: v.Generation, State: v.State, TaskId: v.TaskID, ResourceVersion: v.Version, ObservedAt: timestamppb.New(v.Observed), ItemsDigest: v.Digest, WarehouseId: v.Warehouse, ContractVersion: v.Contract}
	if v.Command != nil {
		x := v.Command
		out.Command = &c.CommandEvidence{IdempotencyKey: x.Key, Action: x.Action, RequestHash: x.Hash, Result: x.Result, ReasonCode: x.Reason, DecisionId: x.ID, DecisionAt: timestamppb.New(x.At), ResultingState: x.State}
	}
	if v.Reference != nil {
		x := v.Reference
		out.Decision = &c.DecisionReference{Service: x.Service, DecisionId: x.ID, OrderId: x.OrderID, Generation: x.Generation, ResourceVersion: x.Version}
	}
	return out
}
func (r *RPC) AcceptTask(ctx context.Context, in *f.AcceptTaskRequest) (*f.TaskDecision, error) {
	v, e := r.Service.Accept(ctx, principal(ctx), in)
	if e != nil {
		return nil, rpcError(e)
	}
	return wire(v), nil
}
func (r *RPC) CancelTask(ctx context.Context, in *f.CancelTaskRequest) (*f.TaskDecision, error) {
	v, e := r.Service.Cancel(ctx, principal(ctx), in)
	if e != nil {
		return nil, rpcError(e)
	}
	return wire(v), nil
}
func (r *RPC) GetTaskDecision(ctx context.Context, in *f.GetTaskDecisionRequest) (*f.TaskDecision, error) {
	caller := principal(ctx).Service
	if caller != "order-service" && caller != "inventory-service" && caller != "fulfillment-service" {
		return nil, status.Error(codes.PermissionDenied, "query scope denied")
	}
	if in.ContractVersion != d.Contract {
		return nil, status.Error(codes.InvalidArgument, "UNSUPPORTED_CONTRACT")
	}
	v, e := r.Service.Store.Decision(ctx, in.OrderId, in.Generation, in.CommandKey, in.CommandAction)
	if e != nil {
		return nil, rpcError(e)
	}
	return wire(v), nil
}
func (r *RPC) GetTaskRevision(ctx context.Context, in *f.GetTaskRevisionRequest) (*f.TaskRevision, error) {
	caller := principal(ctx).Service
	if caller != "order-service" && caller != "shipping-service" {
		return nil, status.Error(codes.PermissionDenied, "source adapter identity required")
	}
	v, e := r.Service.Store.Revision(ctx, in.TaskId, in.ResourceVersion)
	if e != nil {
		return nil, rpcError(e)
	}
	if v.Kind == "TASK_UPDATED" {
		return nil, status.Error(codes.FailedPrecondition, "NOT_A_SOURCE_CHECKPOINT")
	}
	t := v.Task
	seal := ""
	if t.Package.Seal != nil {
		seal = *t.Package.Seal
	}
	return &f.TaskRevision{OrderId: t.OrderID, TaskId: t.ID, ResourceVersion: t.Version, ReadyGeneration: t.Generation, Kind: v.Kind, SealCode: seal, OccurredAt: timestamppb.New(v.Occurred), ItemsDigest: t.Digest, WarehouseId: t.Warehouse}, nil
}
func (r *RPC) GetPackageRevision(ctx context.Context, in *p.GetPackageRevisionRequest) (*p.PackageRevision, error) {
	caller := principal(ctx).Service
	if caller != "order-service" && caller != "shipping-service" {
		return nil, status.Error(codes.PermissionDenied, "source adapter identity required")
	}
	v, e := r.Service.Store.PackageRevision(ctx, in.PackageId, in.SealedTaskVersion)
	if e != nil {
		return nil, rpcError(e)
	}
	t := v.Task
	out := &p.PackageRevision{OrderId: t.OrderID, TaskId: t.ID, PackageId: t.Package.ID, SealedTaskVersion: t.Version, ReadyGeneration: t.Generation, ItemsDigest: t.Digest, WarehouseId: t.Warehouse, SealCode: *t.Package.Seal, PackedAt: timestamppb.New(v.Occurred), ContractVersion: d.Contract}
	x := t.Package.Parcel
	out.Parcel = &p.ParcelMeasurements{WeightGrams: x.Weight, LengthCm: x.Length, WidthCm: x.Width, HeightCm: x.Height, Fragile: x.Fragile}
	if t.Package.EvidenceID != nil {
		media, e := r.Service.Store.File(ctx, *t.Package.EvidenceID)
		if e != nil {
			return nil, rpcError(e)
		}
		out.ImmutableVideoObjectKey = media.FinalKey
		if media.Checksum != nil {
			out.VideoChecksumSha256 = *media.Checksum
		}
	}
	return out, nil
}
func (r *RPC) GetPackingVideoUrl(context.Context, *f.GetPackingVideoUrlRequest) (*f.GetPackingVideoUrlResponse, error) {
	return nil, status.Error(codes.FailedPrecondition, "CARE_DELEGATION_CAPABILITY_UNAVAILABLE")
}
func (r *RPC) SubmitPackingEvidence(context.Context, *f.SubmitPackingEvidenceRequest) (*f.SubmitPackingEvidenceResponse, error) {
	return nil, status.Error(codes.FailedPrecondition, "LEGACY_ADAPTER_DISABLED")
}
