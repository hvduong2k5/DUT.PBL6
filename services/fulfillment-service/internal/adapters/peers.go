package adapters

import (
	"context"
	"encoding/json"
	"github.com/omamx/fulfillment-service/internal/application"
	d "github.com/omamx/fulfillment-service/internal/domain"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	o "github.com/omamx/fulfillment-service/internal/gen/order/v1"
	p "github.com/omamx/fulfillment-service/internal/gen/packing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"time"
)

// The candidate endpoints are opt-in, explicit peer capabilities. UNIMPLEMENTED
// is returned to the caller without substituting a legacy/HTTP/fake provider.
type Peers struct {
	Order          o.OrderServiceClient
	Packing        p.OrderPackingInputsCandidateClient
	Inventory      p.InventoryPackingInputsCandidateClient
	Token          string
	InventoryToken string
}

func (pv *Peers) AcquirePermit(ctx context.Context, event d.Ready, key string) (*f.AcceptTaskRequest, error) {
	c, cancel := pv.ctx(ctx)
	defer cancel()
	out, e := pv.Order.GetFulfillmentPermit(c, &o.GetFulfillmentPermitRequest{OrderId: event.Data.Order, Generation: event.Data.Generation, TaskRequestKey: key, ItemsDigest: event.Data.Digest, ContractVersion: d.Contract})
	if e != nil {
		return nil, e
	}
	if !out.Allowed || out.Permit == nil || out.ContractVersion != d.Contract {
		return nil, d.Fail("PERMIT_UNAVAILABLE", 503)
	}
	in := &f.AcceptTaskRequest{OrderId: event.Data.Order, Generation: event.Data.Generation, IdempotencyKey: key, Permit: out.Permit, ContractVersion: d.Contract}
	in.RequestHash, e = application.ProtoHash(in)
	return in, e
}

func NewPeers(order, inventory grpc.ClientConnInterface, token string) *Peers {
	return &Peers{Order: o.NewOrderServiceClient(order), Packing: p.NewOrderPackingInputsCandidateClient(order), Inventory: p.NewInventoryPackingInputsCandidateClient(inventory), Token: token}
}
func (pv *Peers) ctx(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, application.PeerTimeout)
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+pv.Token), cancel
}
func convert(m proto.Message, out any) error {
	b, e := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(m)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, out)
}
func (pv *Peers) Admission(ctx context.Context, in *f.AcceptTaskRequest) (permit d.Permit, inputs d.Inputs, stock d.Stock, e error) {
	if in.Permit == nil || in.Permit.ExpiresAt == nil || in.Permit.ExpiresAt.CheckValid() != nil {
		return permit, inputs, stock, d.Fail("INVALID_PERMIT", 400)
	}
	c, cancel := pv.ctx(ctx)
	verified, err := pv.Order.ValidateFulfillmentPermit(c, &o.ValidateFulfillmentPermitRequest{Permit: in.Permit, ContractVersion: d.Contract})
	cancel()
	if err != nil {
		return permit, inputs, stock, err
	}
	if verified.ContractVersion != d.Contract {
		return permit, inputs, stock, d.Fail("UNSUPPORTED_CONTRACT", 412)
	}
	if !verified.Allowed {
		permit.Allowed = false
		return permit, inputs, stock, nil
	}
	if verified.Permit == nil || !proto.Equal(verified.Permit, in.Permit) {
		return permit, inputs, stock, d.Fail("PERMIT_PROOF_MISMATCH", 412)
	}
	x := verified.Permit
	permit = d.Permit{ID: x.PermitId, OrderID: x.OrderId, Digest: x.ItemsDigest, Warehouse: x.WarehouseId, Key: x.TaskRequestKey, Audience: x.Audience, Policy: x.PolicyVersion, Generation: x.Generation, OrderVersion: x.OrderVersion, Expires: x.ExpiresAt.AsTime(), Allowed: true}
	c, cancel = pv.ctx(ctx)
	snap, err := pv.Packing.GetPackingSnapshot(c, &p.GetPackingSnapshotRequest{OrderId: in.OrderId, ReadyGeneration: in.Generation, ContractVersion: d.Contract})
	cancel()
	if err != nil {
		return permit, inputs, stock, err
	}
	// protojson encodes int64 as decimal strings; map explicitly via typed fields.
	inputs = d.Inputs{Schema: snap.SchemaVersion, Contract: snap.ContractVersion, SnapshotID: snap.SnapshotId, Version: snap.SnapshotVersion, OrderID: snap.OrderId, Generation: snap.ReadyGeneration, Digest: snap.ItemsDigest, Warehouse: snap.WarehouseId, ReservationID: snap.ReservationId, Channel: snap.Channel, DeadlineRevision: snap.DeadlineRevision}
	if snap.PackingDueAt != nil {
		v := snap.PackingDueAt.AsTime()
		inputs.Due = &v
	}
	if snap.Policy == nil {
		return permit, inputs, stock, d.Fail("POLICY_REQUIRED", 412)
	}
	pol := snap.Policy
	inputs.Policy = d.Policy{Version: pol.PolicyVersion, Evidence: pol.EvidenceMode, MaxWeight: pol.MaximumWeightGrams, MaxDimension: pol.MaximumDimensionCm, MaxVideoBytes: pol.MaximumVideoBytes, MaxVideoSeconds: pol.MaximumVideoSeconds}
	for _, v := range snap.Lines {
		inputs.Lines = append(inputs.Lines, d.Line{ID: v.OrderLineId, SKU: v.SkuCode, Name: v.ProductName, Packaging: v.PackagingText, Quantity: v.Quantity, Fragile: v.Fragile})
	}
	for _, v := range pol.Checklist {
		inputs.Policy.Checklist = append(inputs.Policy.Checklist, d.Checklist{Code: v.ItemCode, Definition: v.Definition, Required: v.Required})
	}
	c, cancel = pv.ctx(ctx)
	if pv.InventoryToken != "" {
		c = metadata.NewOutgoingContext(c, metadata.Pairs("authorization", "Bearer "+pv.InventoryToken))
	}
	alloc, err := pv.Inventory.GetCommittedPackingAllocation(c, &p.GetCommittedPackingAllocationRequest{OrderId: in.OrderId, ReservationId: inputs.ReservationID, ReadyGeneration: in.Generation, ItemsDigest: inputs.Digest, WarehouseId: inputs.Warehouse, ContractVersion: d.Contract, Lines: snap.Lines})
	cancel()
	if err != nil {
		return permit, inputs, stock, err
	}
	stock = d.Stock{Contract: alloc.ContractVersion, OrderID: alloc.OrderId, ReservationID: alloc.ReservationId, State: alloc.State, Version: alloc.AllocationVersion, Digest: alloc.ItemsDigest, Warehouse: alloc.WarehouseId, DecisionID: alloc.OriginalCommitDecisionId, Generation: alloc.ReadyGeneration}
	if alloc.ObservedAt != nil {
		stock.Observed = alloc.ObservedAt.AsTime()
	}
	stock.Slots = slots(alloc.Slots, alloc.AllocationVersion)
	return
}
func slots(in []*p.BatchSlot, v int64) []d.Allocation {
	out := []d.Allocation{}
	for _, x := range in {
		until := time.Time{}
		if x.BatchValidUntil != nil {
			until = x.BatchValidUntil.AsTime()
		}
		out = append(out, d.Allocation{ID: x.AllocationId, LineID: x.OrderLineId, SKU: x.SkuCode, BatchID: x.BatchId, BatchCode: x.BatchCode, Allocated: x.AllocatedQuantity, Version: v, Until: until, EligibilityVersion: x.EligibilityRevision, Eligible: x.CurrentlyEligible})
	}
	return out
}
func (pv *Peers) ValidatePacking(ctx context.Context, t d.Task) (*d.Validation, error) {
	in := &p.ValidatePackingAllocationRequest{OrderId: t.OrderID, TaskId: t.ID, ReservationId: t.Inputs.ReservationID, AllocationVersion: t.Stock.Version, ReadyGeneration: t.Generation, ItemsDigest: t.Digest, WarehouseId: t.Warehouse, ContractVersion: d.Contract}
	for _, x := range t.Allocations {
		in.Picks = append(in.Picks, &p.PickedSlot{AllocationId: x.ID, OrderLineId: x.LineID, BatchId: x.BatchID, CountedQuantity: x.Counted})
	}
	c, cancel := pv.ctx(ctx)
	defer cancel()
	if pv.InventoryToken != "" {
		c = metadata.NewOutgoingContext(c, metadata.Pairs("authorization", "Bearer "+pv.InventoryToken))
	}
	v, e := pv.Inventory.ValidatePackingAllocation(c, in)
	if e != nil {
		return nil, e
	}
	out := &d.Validation{Allowed: v.Allowed, ID: v.ValidationId, Contract: v.ContractVersion, OrderID: v.OrderId, TaskID: v.TaskId, ReservationID: v.ReservationId, Version: v.AllocationVersion, Generation: v.ReadyGeneration, Digest: v.ItemsDigest, Warehouse: v.WarehouseId, Slots: slots(v.Slots, v.AllocationVersion)}
	if v.ObservedAt != nil {
		out.Observed = v.ObservedAt.AsTime()
	}
	if v.ValidUntil != nil {
		out.Until = v.ValidUntil.AsTime()
	}
	return out, nil
}
