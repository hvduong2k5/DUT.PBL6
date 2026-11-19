package promotionprovider

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain/core"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	api "github.com/omamx/order-service/internal/gen/promotion/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type peer struct {
	api.UnimplementedPromotionServiceServer
	hold                   *api.VoucherHoldOutcome
	commands               map[string]*common.CommandEvidence
	locks, finalizes       int
	loseLock, loseFinalize bool
}

func (p *peer) CalculateDiscount(context.Context, *api.CalculateDiscountRequest) (*api.CalculateDiscountResponse, error) {
	return &api.CalculateDiscountResponse{IsApplicable: true, EstimatedDiscount: money(1000), DiscountType: api.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT, ValidUntil: timestamppb.New(time.Now().Add(time.Hour)), PolicyVersion: "promotion-voucher-1"}, nil
}
func (p *peer) ValidateAndLockVoucher(_ context.Context, r *api.ValidateAndLockVoucherRequest) (*api.ValidateAndLockVoucherResponse, error) {
	p.locks++
	p.hold = &api.VoucherHoldOutcome{OrderId: r.OrderId, HoldId: uuid.NewString(), State: "HELD", ResourceVersion: 1, ExpiresAt: timestamppb.New(time.Now().Add(15 * time.Minute)), ObservedAt: timestamppb.Now(), DiscountAmount: money(1000), ContractVersion: contract}
	p.commands["ValidateAndLockVoucher"] = &common.CommandEvidence{IdempotencyKey: r.IdempotencyKey, RequestHash: r.RequestHash, Action: "ValidateAndLockVoucher", Result: "APPLIED", DecisionId: p.hold.HoldId, DecisionAt: timestamppb.Now(), ResultingState: "HELD"}
	if p.loseLock {
		p.loseLock = false
		return nil, status.Error(codes.DeadlineExceeded, "lost acknowledgement")
	}
	out := proto.Clone(p.hold).(*api.VoucherHoldOutcome)
	out.Command = p.commands["ValidateAndLockVoucher"]
	return &api.ValidateAndLockVoucherResponse{IsValid: true, Outcome: out, DiscountType: api.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT}, nil
}
func (p *peer) GetVoucherHoldByOrder(_ context.Context, r *api.VoucherHoldCommand) (*api.VoucherHoldOutcome, error) {
	if p.hold == nil {
		return nil, status.Error(codes.NotFound, "missing")
	}
	out := proto.Clone(p.hold).(*api.VoucherHoldOutcome)
	if c := p.commands[r.QueryAction]; c != nil && c.IdempotencyKey == r.IdempotencyKey && c.RequestHash == r.RequestHash {
		out.Command = proto.Clone(c).(*common.CommandEvidence)
	} else if r.QueryAction != "" {
		return nil, status.Error(codes.NotFound, "command receipt missing")
	}
	return out, nil
}
func (p *peer) FinalizeVoucherHold(_ context.Context, r *api.VoucherHoldCommand) (*api.VoucherHoldOutcome, error) {
	p.finalizes++
	p.hold.State = "FINALIZED"
	p.hold.ResourceVersion = 2
	c := &common.CommandEvidence{IdempotencyKey: r.IdempotencyKey, RequestHash: r.RequestHash, Action: "FinalizeVoucherHold", Result: "APPLIED", DecisionId: p.hold.HoldId, DecisionAt: timestamppb.Now(), ResultingState: "FINALIZED"}
	p.commands[c.Action] = c
	if p.loseFinalize {
		p.loseFinalize = false
		return nil, status.Error(codes.Unavailable, "lost acknowledgement")
	}
	out := proto.Clone(p.hold).(*api.VoucherHoldOutcome)
	out.Command = c
	return out, nil
}
func (p *peer) ReleaseVoucherHoldByOrder(_ context.Context, r *api.VoucherHoldCommand) (*api.VoucherHoldOutcome, error) {
	if p.hold == nil {
		p.hold = &api.VoucherHoldOutcome{OrderId: r.OrderId, HoldId: r.GetExpectedHoldId(), State: "RELEASED", ResourceVersion: 1, ObservedAt: timestamppb.Now(), ContractVersion: contract}
	}
	p.hold.State = "RELEASED"
	return proto.Clone(p.hold).(*api.VoucherHoldOutcome), nil
}
func testAdapter(t *testing.T, p *peer) *Adapter {
	t.Helper()
	p.commands = map[string]*common.CommandEvidence{}
	l := bufconn.Listen(1 << 20)
	token := strings.Repeat("p", 32)
	s := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, r any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		if v := metadata.ValueFromIncomingContext(ctx, "authorization"); len(v) != 1 || v[0] != "Bearer "+token {
			return nil, status.Error(codes.Unauthenticated, "missing credential")
		}
		return h(ctx, r)
	}))
	api.RegisterPromotionServiceServer(s, p)
	go s.Serve(l)
	t.Cleanup(s.Stop)
	c, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return l.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(); _ = l.Close() })
	return &Adapter{Client: api.NewPromotionServiceClient(c), Token: token}
}
func TestVoucherCommandRecoveryAndAuthenticatedTransport(t *testing.T) {
	p := &peer{loseLock: true, loseFinalize: true}
	a := testAdapter(t, p)
	ctx := context.Background()
	customer := uuid.NewString()
	v, err := a.Quote(ctx, "WELCOME", customer, 10000, 2000)
	if err != nil {
		t.Fatal(err)
	}
	s := core.SnapshotInput{OrderID: uuid.NewString(), Owner: core.Owner{CustomerID: &customer}, Voucher: v, Money: core.SnapshotMoney{Subtotal: 10000, ShippingFee: 2000}}
	if _, err = a.Hold(ctx, s); err == nil {
		t.Fatal("expected lost lock ACK")
	}
	s.Voucher, err = a.Hold(ctx, s)
	if err != nil || p.locks != 1 {
		t.Fatal("lock retry must recover original effect", err, p.locks)
	}
	if _, err = a.Finalize(ctx, s); err == nil {
		t.Fatal("expected lost finalize ACK")
	}
	d, err := a.Finalize(ctx, s)
	if err != nil || p.finalizes != 1 || d.HoldID != *s.Voucher.HoldID {
		t.Fatal("finalize retry must recover original decision", err, p.finalizes)
	}
}
func TestVoucherRejectsChangedAmountAndTerminalHold(t *testing.T) {
	p := &peer{}
	a := testAdapter(t, p)
	customer := uuid.NewString()
	v, _ := a.Quote(context.Background(), "WELCOME", customer, 10000, 2000)
	s := core.SnapshotInput{OrderID: uuid.NewString(), Owner: core.Owner{CustomerID: &customer}, Voucher: v, Money: core.SnapshotMoney{Subtotal: 10000, ShippingFee: 2000}}
	held, err := a.Hold(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	s.Voucher = held
	originalID := p.hold.HoldId
	p.hold.HoldId = uuid.NewString()
	if _, err := a.Hold(context.Background(), s); err == nil {
		t.Fatal("accepted a replacement hold")
	}
	if err := a.Release(context.Background(), s); err == nil {
		t.Fatal("released a replacement hold")
	}
	p.hold.HoldId = originalID
	p.hold.DiscountAmount = money(1001)
	if _, err := a.Hold(context.Background(), s); err == nil {
		t.Fatal("accepted changed discount")
	}
	p.hold.DiscountAmount = money(1000)
	p.hold.State = "EXPIRED"
	if _, err := a.Hold(context.Background(), s); err != core.ErrVoucherRejected {
		t.Fatal(err)
	}
	if err := a.Release(context.Background(), s); err != nil {
		t.Fatal("authoritative expiry settles release", err)
	}
}
