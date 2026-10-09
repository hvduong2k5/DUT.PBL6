package transport

import (
	"context"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	order "github.com/omamx/order-service/internal/gen/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RPC struct {
	order.UnimplementedOrderServiceServer
	S *app.Service
}

func (r *RPC) principal(ctx context.Context) (domain.Principal, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	auth := md.Get("authorization")
	if len(auth) != 1 {
		return domain.Principal{}, status.Error(codes.Unauthenticated, "verified principal required")
	}
	p, e := r.S.Authenticate(auth[0])
	if e != nil {
		return p, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	return p, nil
}
func rpcError(s *app.Service, e error) error {
	if e == nil {
		return nil
	}
	code, http := s.ErrorCode(e)
	c := codes.Internal
	switch http {
	case 400, 422:
		c = codes.InvalidArgument
	case 401:
		c = codes.Unauthenticated
	case 403:
		c = codes.PermissionDenied
	case 404:
		c = codes.NotFound
	case 409:
		c = codes.FailedPrecondition
	case 503:
		c = codes.Unavailable
	}
	return status.Error(c, code)
}
func money(n int64) *common.Money { return &common.Money{CurrencyCode: "VND", Units: n, Nanos: 0} }
func orderStatus(s string) order.OrderStatus {
	return order.OrderStatus(order.OrderStatus_value["ORDER_STATUS_"+s])
}
func (r *RPC) CreatePOSOrder(context.Context, *order.CreatePOSOrderRequest) (*order.CreatePOSOrderResponse, error) {
	return nil, status.Error(codes.FailedPrecondition, "POS_FEATURE_DISABLED")
}
func (r *RPC) GetOrderDetail(ctx context.Context, in *order.GetOrderDetailRequest) (*order.GetOrderDetailResponse, error) {
	p, e := r.principal(ctx)
	if e != nil {
		return nil, e
	}
	o, e := r.S.GetOrder(ctx, in.OrderId, p, true)
	if e != nil {
		return nil, rpcError(r.S, e)
	}
	out := &order.GetOrderDetailResponse{OrderId: o.ID, Channel: "D2C_WEB", Status: orderStatus(o.Status), SubtotalAmount: money(o.Subtotal), ShippingFee: money(o.Shipping), FinalAmount: money(o.Final), Payment: &order.PaymentInfo{PaymentMethod: o.Method, PaymentStatus: o.Payment}, Version: o.Version, StockStatus: o.Stock, OperationalHold: o.Hold, CreatedAt: timestamppb.New(o.CreatedAt), UpdatedAt: timestamppb.New(o.UpdatedAt)}
	if o.CustomerID != nil {
		out.CustomerId = *o.CustomerID
	}
	if o.Snapshot != nil {
		a := o.Snapshot.Address
		out.ShippingAddress = &common.Address{RecipientName: a.Name, PhoneNumber: a.Phone, StreetAddress: a.Street, Ward: a.Ward, Province: a.Province, CountryCode: "VN"}
		for _, i := range o.Snapshot.Items {
			out.Items = append(out.Items, &order.OrderItem{SkuCode: i.SKU, ProductName: i.Name, Quantity: int32(i.Quantity), UnitPrice: money(i.UnitPrice), TotalPrice: money(i.UnitPrice * int64(i.Quantity))})
		}
	}
	return out, nil
}
func (r *RPC) CancelOrder(ctx context.Context, in *order.CancelOrderRequest) (*order.CancelOrderResponse, error) {
	p, e := r.principal(ctx)
	if e != nil {
		return nil, e
	}
	if in.ExpectedVersion <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected version required")
	}
	_, e = r.S.Cancel(ctx, p, in.OrderId, in.IdempotencyKey, in.ExpectedVersion, in.CancelReason, false)
	if e != nil {
		return nil, rpcError(r.S, e)
	}
	o, e := r.S.GetOrder(ctx, in.OrderId, p, false)
	if e != nil {
		return nil, rpcError(r.S, e)
	}
	return &order.CancelOrderResponse{IsCancelled: o.Status == "CANCELLED_BY_USER", FinalStatus: orderStatus(o.Status), Message: "durable cancellation accepted; poll Order status"}, nil
}
func (r *RPC) GetCheckoutOperation(ctx context.Context, in *order.GetCheckoutOperationRequest) (*order.GetCheckoutOperationResponse, error) {
	p, e := r.principal(ctx)
	if e != nil {
		return nil, e
	}
	o, e := r.S.Operation(ctx, in.OperationId, p)
	if e != nil {
		return nil, rpcError(r.S, e)
	}
	out := &order.GetCheckoutOperationResponse{OperationId: o.ID, Status: o.Status, OrderId: o.OrderID}
	if o.OrderStatus != "" {
		st := orderStatus(o.OrderStatus)
		out.OrderStatus = &st
	}
	if o.ErrorCode != "" {
		out.ErrorCode = &o.ErrorCode
	}
	return out, nil
}
