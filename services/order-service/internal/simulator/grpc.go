package simulator

import (
	"context"
	"crypto/hmac"
	"errors"
	"github.com/omamx/order-service/internal/domain"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InventoryRPC struct {
	inventory.UnimplementedInventoryServiceServer
	S *Simulator
}

func (r *InventoryRPC) auth(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	v := md.Get("x-internal-token")
	if len(v) != 1 || !hmac.Equal([]byte(v[0]), []byte(r.S.Token)) {
		return status.Error(codes.Unauthenticated, "trusted caller required")
	}
	return nil
}
func inventoryError(e error) error {
	var d *domain.Error
	if errors.As(e, &d) {
		c := codes.FailedPrecondition
		if d.HTTP >= 500 {
			c = codes.Unavailable
		}
		if d.HTTP == 400 || d.HTTP == 422 {
			c = codes.InvalidArgument
		}
		if d.HTTP == 404 {
			c = codes.NotFound
		}
		return status.Error(c, d.Code)
	}
	return status.Error(codes.Internal, "inventory unavailable")
}
func outcome(v domain.Reservation) *inventory.ReservationOutcome {
	return &inventory.ReservationOutcome{OrderId: v.OrderID, ReservationId: v.ID, State: v.State, ExpiresAt: timestamppb.New(v.ExpiresAt)}
}
func (r *InventoryRPC) ReserveStock(ctx context.Context, in *inventory.ReserveStockRequest) (*inventory.ReserveStockResponse, error) {
	if e := r.auth(ctx); e != nil {
		return nil, e
	}
	items := []domain.Item{}
	for _, i := range in.Items {
		items = append(items, domain.Item{SKU: i.SkuCode, Quantity: int(i.Quantity)})
	}
	ttl := in.TtlMinutes
	if ttl == 0 {
		ttl = 15
	}
	res, e := r.S.inventory(ctx, "reserve", in.OrderId, in.IdempotencyKey, items, int(ttl)*60, "", false)
	if e != nil {
		return nil, inventoryError(e)
	}
	if res.State != "RESERVED" {
		return nil, status.Error(codes.FailedPrecondition, res.State)
	}
	return &inventory.ReserveStockResponse{Status: inventory.ReservationStatus_RESERVATION_STATUS_SUCCESS, ReservationId: &res.ID, ExpiresAt: timestamppb.New(res.ExpiresAt)}, nil
}
func (r *InventoryRPC) ReleaseReservation(ctx context.Context, in *inventory.ReleaseReservationRequest) (*inventory.ReleaseReservationResponse, error) {
	if e := r.auth(ctx); e != nil {
		return nil, e
	}
	current, e := r.S.inventory(ctx, "query", in.OrderId, "query-operation-key", nil, 0, "", false)
	if e != nil {
		return nil, inventoryError(e)
	}
	if current.ID != in.ReservationId {
		return nil, status.Error(codes.FailedPrecondition, "reservation mismatch")
	}
	res, e := r.S.inventory(ctx, "release", in.OrderId, in.IdempotencyKey, nil, 0, "", false)
	if e != nil {
		return nil, inventoryError(e)
	}
	return &inventory.ReleaseReservationResponse{IsReleased: res.State == "RELEASED" || res.State == "EXPIRED", Message: res.State}, nil
}
func (r *InventoryRPC) command(ctx context.Context, action string, in *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	if e := r.auth(ctx); e != nil {
		return nil, e
	}
	res, e := r.S.inventory(ctx, action, in.OrderId, in.IdempotencyKey, nil, 0, in.ApprovalReference, in.CodUnpaid)
	if e != nil {
		return nil, inventoryError(e)
	}
	return outcome(res), nil
}
func (r *InventoryRPC) GetReservationByOrder(ctx context.Context, in *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	return r.command(ctx, "query", in)
}
func (r *InventoryRPC) FinalizeReservation(ctx context.Context, in *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	return r.command(ctx, "finalize", in)
}
func (r *InventoryRPC) ReleaseReservationByOrder(ctx context.Context, in *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	return r.command(ctx, "release", in)
}
func (r *InventoryRPC) ReverseCommittedStock(ctx context.Context, in *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	return r.command(ctx, "reverse", in)
}
func (r *InventoryRPC) GetStockLevel(ctx context.Context, in *inventory.GetStockLevelRequest) (*inventory.GetStockLevelResponse, error) {
	if e := r.auth(ctx); e != nil {
		return nil, e
	}
	out := &inventory.GetStockLevelResponse{}
	for _, sku := range in.SkuCodes {
		var physical, available int32
		if e := r.S.DB.QueryRow(ctx, "SELECT physical,available FROM stock WHERE sku=$1", sku).Scan(&physical, &available); e != nil {
			return nil, status.Error(codes.NotFound, "SKU_NOT_FOUND")
		}
		out.Items = append(out.Items, &inventory.StockLevelItem{SkuCode: sku, PhysicalQuantity: physical, AvailableQuantity: available, ReservedQuantity: physical - available, StockStatus: "SYNTHETIC"})
	}
	return out, nil
}
func (r *InventoryRPC) GetBatchFEFODetails(context.Context, *inventory.GetBatchFEFODetailsRequest) (*inventory.GetBatchFEFODetailsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "real batch/FEFO ledger is outside this simulator")
}
