package grpc

import (
	"context"
	"crypto/subtle"
	"database/sql"
	packing "dut-pbl6/inventory-service/pkg/proto/packing/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

// Packing receives a separate read-only capability, never Order's stock command
// credential. Order line IDs are join references supplied from Order's snapshot;
// Inventory proves the independently stored SKU/batch/quantity and commit decision.
type PackingServer struct {
	packing.UnimplementedInventoryPackingInputsCandidateServer
	Terminal *TerminalServer
	Token    string
}

func (s *PackingServer) authorize(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	v := md.Get("authorization")
	if len(s.Token) < 32 || len(v) != 1 || subtle.ConstantTimeCompare([]byte(v[0]), []byte("Bearer "+s.Token)) != 1 {
		return status.Error(codes.Unauthenticated, "FULFILLMENT_READ_IDENTITY_REQUIRED")
	}
	return nil
}
func allocationID(reservation, sku, batch string) string {
	return uuid.NewSHA1(uuid.MustParse(reservation), []byte(sku+":"+batch)).String()
}
func packingSlots(ctx context.Context, tx *sql.Tx, r *terminalRecord, lines map[string]string) ([]*packing.BatchSlot, error) {
	out := []*packing.BatchSlot{}
	for _, a := range r.allocations {
		var code, state string
		var until time.Time
		var revision int64
		if e := tx.QueryRowContext(ctx, `SELECT batch_code,status,exp_date::timestamptz,version FROM batches WHERE id=$1 AND sku=$2 FOR SHARE`, a.BatchID, a.SKU).Scan(&code, &state, &until, &revision); e != nil {
			return nil, e
		}
		out = append(out, &packing.BatchSlot{AllocationId: allocationID(r.outcome.ReservationId, a.SKU, a.BatchID.String()), OrderLineId: lines[a.SKU], SkuCode: a.SKU, BatchId: a.BatchID.String(), BatchCode: code, AllocatedQuantity: int32(a.AllocatedQty), BatchValidUntil: timestamppb.New(until), EligibilityRevision: revision, CurrentlyEligible: state == "ACTIVE" || state == "NEAR_EXPIRY"})
	}
	return out, nil
}
func (s *PackingServer) GetCommittedPackingAllocation(ctx context.Context, in *packing.GetCommittedPackingAllocationRequest) (*packing.CommittedPackingAllocation, error) {
	if e := s.authorize(ctx); e != nil {
		return nil, e
	}
	if in == nil || in.ContractVersion != "order-3.2" || in.ReadyGeneration < 1 || len(in.Lines) == 0 {
		return nil, status.Error(codes.InvalidArgument, "INVALID_PACKING_REQUEST")
	}
	lines := map[string]string{}
	quantities := map[string]int64{}
	ids := map[string]bool{}
	for _, l := range in.Lines {
		id, e := uuid.Parse(l.OrderLineId)
		if e != nil || id == uuid.Nil || l.Quantity < 1 || l.SkuCode == "" || lines[l.SkuCode] != "" || ids[l.OrderLineId] {
			return nil, status.Error(codes.InvalidArgument, "INVALID_PACKING_LINES")
		}
		lines[l.SkuCode] = l.OrderLineId
		ids[l.OrderLineId] = true
		quantities[l.SkuCode] = int64(l.Quantity)
	}
	var out *packing.CommittedPackingAllocation
	e := s.Terminal.transact(ctx, in.OrderId, func(tx *sql.Tx, r *terminalRecord, now time.Time) error {
		if r.outcome == nil || r.outcome.State != "COMMITTED" || r.outcome.ReservationId != in.ReservationId || r.outcome.ItemsDigest != in.ItemsDigest || r.outcome.WarehouseId != in.WarehouseId {
			return status.Error(codes.FailedPrecondition, "COMMITTED_ALLOCATION_REQUIRED")
		}
		actual := map[string]int64{}
		for _, a := range r.allocations {
			actual[a.SKU] += int64(a.AllocatedQty)
		}
		if len(actual) != len(quantities) {
			return status.Error(codes.FailedPrecondition, "ALLOCATION_COVERAGE")
		}
		for sku, q := range actual {
			if quantities[sku] != q {
				return status.Error(codes.FailedPrecondition, "ALLOCATION_COVERAGE")
			}
		}
		decision := ""
		for _, c := range r.commands {
			if c.Action == "FinalizeReservation" && c.ResultingState == "COMMITTED" && (c.Result == "APPLIED" || c.Result == "ALREADY_APPLIED") {
				decision = c.DecisionId
				break
			}
		}
		if decision == "" {
			return status.Error(codes.FailedPrecondition, "COMMIT_DECISION_REQUIRED")
		}
		slots, e := packingSlots(ctx, tx, r, lines)
		if e != nil {
			return e
		}
		for _, slot := range slots {
			if !now.Before(slot.BatchValidUntil.AsTime()) {
				slot.CurrentlyEligible = false
			}
		}
		out = &packing.CommittedPackingAllocation{ContractVersion: "order-3.2", OrderId: in.OrderId, ReservationId: in.ReservationId, State: r.outcome.State, AllocationVersion: r.outcome.ResourceVersion, ItemsDigest: r.outcome.ItemsDigest, WarehouseId: r.outcome.WarehouseId, ObservedAt: timestamppb.New(now), Slots: slots, OriginalCommitDecisionId: decision, ReadyGeneration: in.ReadyGeneration}
		return nil
	})
	return out, e
}
func (s *PackingServer) ValidatePackingAllocation(ctx context.Context, in *packing.ValidatePackingAllocationRequest) (*packing.PackingValidation, error) {
	if e := s.authorize(ctx); e != nil {
		return nil, e
	}
	if in == nil || in.ContractVersion != "order-3.2" || in.ReadyGeneration < 1 || len(in.Picks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "INVALID_PACKING_VALIDATION")
	}
	if id, e := uuid.Parse(in.TaskId); e != nil || id == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "INVALID_PACKING_TASK")
	}
	var out *packing.PackingValidation
	e := s.Terminal.transact(ctx, in.OrderId, func(tx *sql.Tx, r *terminalRecord, now time.Time) error {
		if r.outcome == nil || r.outcome.State != "COMMITTED" || r.outcome.ReservationId != in.ReservationId || r.outcome.ItemsDigest != in.ItemsDigest || r.outcome.WarehouseId != in.WarehouseId || r.outcome.ResourceVersion != in.AllocationVersion {
			return status.Error(codes.FailedPrecondition, "COMMITTED_ALLOCATION_CHANGED")
		}
		picks := map[string]*packing.PickedSlot{}
		for _, p := range in.Picks {
			if picks[p.AllocationId] != nil {
				return status.Error(codes.InvalidArgument, "DUPLICATE_PICK")
			}
			picks[p.AllocationId] = p
		}
		lines := map[string]string{}
		for _, a := range r.allocations {
			p := picks[allocationID(r.outcome.ReservationId, a.SKU, a.BatchID.String())]
			if p == nil || p.BatchId != a.BatchID.String() || p.CountedQuantity != int32(a.AllocatedQty) {
				return status.Error(codes.FailedPrecondition, "PICK_COVERAGE_MISMATCH")
			}
			id, e := uuid.Parse(p.OrderLineId)
			if e != nil || id == uuid.Nil || (lines[a.SKU] != "" && lines[a.SKU] != p.OrderLineId) {
				return status.Error(codes.InvalidArgument, "PICK_LINE_MISMATCH")
			}
			lines[a.SKU] = p.OrderLineId
		}
		if len(picks) != len(r.allocations) {
			return status.Error(codes.FailedPrecondition, "PICK_COVERAGE_MISMATCH")
		}
		slots, e := packingSlots(ctx, tx, r, lines)
		if e != nil {
			return e
		}
		allowed := true
		until := now.Add(30 * time.Second)
		for _, slot := range slots {
			if !now.Before(slot.BatchValidUntil.AsTime()) {
				slot.CurrentlyEligible = false
			}
			if !slot.CurrentlyEligible {
				allowed = false
			}
			if slot.BatchValidUntil.AsTime().Before(until) {
				until = slot.BatchValidUntil.AsTime()
			}
		}
		out = &packing.PackingValidation{Allowed: allowed, ValidationId: uuid.NewString(), ObservedAt: timestamppb.New(now), ValidUntil: timestamppb.New(until), OrderId: in.OrderId, TaskId: in.TaskId, ReservationId: in.ReservationId, AllocationVersion: in.AllocationVersion, ReadyGeneration: in.ReadyGeneration, ItemsDigest: in.ItemsDigest, WarehouseId: in.WarehouseId, ContractVersion: "order-3.2", Slots: slots}
		if !allowed {
			out.ReasonCode = "BATCH_NOT_ELIGIBLE"
		}
		return nil
	})
	return out, e
}
