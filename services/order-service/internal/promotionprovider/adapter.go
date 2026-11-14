package promotionprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	api "github.com/omamx/order-service/internal/gen/promotion/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const contract = "order-3.2"

type Adapter struct {
	Client api.PromotionServiceClient
	Token  string
}

func money(n int64) *common.Money { return &common.Money{CurrencyCode: "VND", Units: n} }
func validMoney(m *common.Money, n int64) bool {
	return m != nil && m.CurrencyCode == "VND" && m.Nanos == 0 && m.Units == n
}
func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func kind(k api.DiscountType) string {
	switch k {
	case api.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT:
		return "FIXED_AMOUNT"
	case api.DiscountType_DISCOUNT_TYPE_PERCENTAGE:
		return "PERCENTAGE"
	case api.DiscountType_DISCOUNT_TYPE_FREE_SHIPPING:
		return "FREE_SHIPPING"
	}
	return ""
}
func (a *Adapter) context(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.Token), cancel
}

// Quote never reserves a use. All prices and customer IDs are supplied by Order.
func (a *Adapter) Quote(ctx context.Context, code, customer string, subtotal, shipping int64) (*core.VoucherTerms, error) {
	ctx, cancel := a.context(ctx)
	defer cancel()
	r, err := a.Client.CalculateDiscount(ctx, &api.CalculateDiscountRequest{VoucherCode: code, CustomerId: customer, OrderSubtotal: money(subtotal), ShippingFee: money(shipping)})
	if err != nil {
		return nil, domain.E("PROMOTION_UNAVAILABLE", 503)
	}
	if !r.GetIsApplicable() {
		return nil, domain.E("VOUCHER_NOT_APPLICABLE", 422)
	}
	m := r.GetEstimatedDiscount()
	if m == nil || !validMoney(m, m.Units) || m.Units <= 0 || kind(r.GetDiscountType()) == "" || r.GetPolicyVersion() != "promotion-voucher-1" || r.GetValidUntil() == nil || r.GetValidUntil().CheckValid() != nil || !time.Now().Before(r.GetValidUntil().AsTime()) {
		return nil, domain.E("INVALID_PROMOTION_PROOF", 503)
	}
	cap := subtotal
	if kind(r.DiscountType) == "FREE_SHIPPING" {
		cap = shipping
	}
	if m.Units > cap {
		return nil, domain.E("INVALID_PROMOTION_PROOF", 503)
	}
	return &core.VoucherTerms{Code: code, Kind: kind(r.DiscountType), Amount: m.Units, Policy: r.PolicyVersion, ValidUntil: r.ValidUntil.AsTime()}, nil
}

func lockHash(s core.SnapshotInput) string {
	return fingerprint([]any{"LOCK", s.OrderID, *s.Owner.CustomerID, s.Voucher.Code, s.Money.Subtotal, s.Money.ShippingFee, 15})
}
func key(action, order string) string { return "promotion:" + action + ":" + order }
func rejected() error                 { return core.ErrVoucherRejected }
func validOutcome(r *api.VoucherHoldOutcome, order string) bool {
	if r == nil || r.OrderId != order || r.ContractVersion != contract || r.ObservedAt == nil || r.ObservedAt.CheckValid() != nil || r.ObservedAt.AsTime().After(time.Now().Add(5*time.Second)) {
		return false
	}
	switch r.State {
	case "HELD", "FINALIZED", "RELEASED", "EXPIRED":
		return r.ResourceVersion > 0
	default:
		return false
	}
}
func commandProof(r *api.VoucherHoldOutcome, k, h, action string) bool {
	c := r.GetCommand()
	return c != nil && c.IdempotencyKey == k && c.RequestHash == h && c.Action == action && c.DecisionId != "" && c.DecisionAt != nil && c.DecisionAt.CheckValid() == nil && (c.Result == "APPLIED" || c.Result == "ALREADY_APPLIED")
}

// Hold recovers unknown command outcomes before retrying the same original key.
func (a *Adapter) Hold(ctx context.Context, s core.SnapshotInput) (*core.VoucherTerms, error) {
	ctx, cancel := a.context(ctx)
	defer cancel()
	k, h := key("lock", s.OrderID), lockHash(s)
	r, err := a.Client.GetVoucherHoldByOrder(ctx, &api.VoucherHoldCommand{OrderId: s.OrderID, ContractVersion: contract, QueryAction: "ValidateAndLockVoucher", IdempotencyKey: k, RequestHash: h})
	if status.Code(err) == codes.NotFound {
		v, e := a.Client.ValidateAndLockVoucher(ctx, &api.ValidateAndLockVoucherRequest{IdempotencyKey: k, RequestHash: h, ContractVersion: contract, VoucherCode: s.Voucher.Code, CustomerId: *s.Owner.CustomerID, OrderId: s.OrderID, OrderSubtotal: money(s.Money.Subtotal), ShippingFee: money(s.Money.ShippingFee), TtlMinutes: 15})
		if e != nil {
			return nil, e
		}
		if !v.IsValid {
			return nil, rejected()
		}
		if kind(v.DiscountType) != s.Voucher.Kind {
			return nil, rejected()
		}
		r = v.Outcome
	} else if err != nil {
		return nil, err
	}
	if r.GetCommand().GetResult() == "REJECTED" {
		return nil, rejected()
	}
	if !validOutcome(r, s.OrderID) || !commandProof(r, k, h, "ValidateAndLockVoucher") || r.HoldId == "" || !validMoney(r.DiscountAmount, s.Voucher.Amount) || r.ExpiresAt == nil || r.ExpiresAt.CheckValid() != nil {
		return nil, fmt.Errorf("invalid voucher lock evidence")
	}
	if r.State != "HELD" || !time.Now().Add(30*time.Second).Before(r.ExpiresAt.AsTime()) {
		return nil, rejected()
	}
	if _, err := uuid.Parse(r.HoldId); err != nil {
		return nil, err
	}
	if (s.Voucher.HoldID != nil && *s.Voucher.HoldID != r.HoldId) ||
		(s.Voucher.HoldExpiresAt != nil && !s.Voucher.HoldExpiresAt.Equal(r.ExpiresAt.AsTime())) {
		return nil, fmt.Errorf("voucher hold no longer matches the canonical checkout")
	}
	v := *s.Voucher
	v.HoldID = &r.HoldId
	expiry := r.ExpiresAt.AsTime()
	v.HoldExpiresAt = &expiry
	return &v, nil
}

// Release also writes an order tombstone if a timed-out lock has not arrived yet.
func (a *Adapter) Release(ctx context.Context, s core.SnapshotInput) error {
	ctx, cancel := a.context(ctx)
	defer cancel()
	r, err := a.Client.GetVoucherHoldByOrder(ctx, &api.VoucherHoldCommand{OrderId: s.OrderID, ContractVersion: contract})
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("voucher-release:"+s.OrderID)).String()
	if status.Code(err) != codes.NotFound {
		if err != nil {
			return err
		}
		if !validOutcome(r, s.OrderID) {
			return fmt.Errorf("invalid voucher release lookup")
		}
		if s.Voucher.HoldID != nil && *s.Voucher.HoldID != r.HoldId {
			return fmt.Errorf("voucher release resource differs from the canonical checkout")
		}
		if r.State == "RELEASED" || r.State == "EXPIRED" {
			return nil
		}
		if r.State != "HELD" {
			return rejected()
		}
		id = r.HoldId
	}
	version := int64(1)
	k := key("release", s.OrderID)
	h := fingerprint([]any{"RELEASE", s.OrderID, id, version})
	out, err := a.Client.ReleaseVoucherHoldByOrder(ctx, &api.VoucherHoldCommand{OrderId: s.OrderID, ContractVersion: contract, IdempotencyKey: k, RequestHash: h, ExpectedHoldId: &id, ExpectedVersion: &version, QueryAction: "CHECKOUT_CANCELLED"})
	if err != nil {
		return err
	}
	if !validOutcome(out, s.OrderID) || (out.State != "RELEASED" && out.State != "EXPIRED") {
		return rejected()
	}
	return nil
}

func (a *Adapter) Finalize(ctx context.Context, s core.SnapshotInput) (core.VoucherDecision, error) {
	ctx, cancel := a.context(ctx)
	defer cancel()
	if s.Voucher == nil || s.Voucher.HoldID == nil {
		return core.VoucherDecision{}, rejected()
	}
	id, version := *s.Voucher.HoldID, int64(1)
	k := key("finalize", s.OrderID)
	h := fingerprint([]any{"FINALIZE", s.OrderID, id, version})
	query := &api.VoucherHoldCommand{OrderId: s.OrderID, ContractVersion: contract, IdempotencyKey: k, RequestHash: h, ExpectedHoldId: &id, QueryAction: "FinalizeVoucherHold"}
	r, err := a.Client.GetVoucherHoldByOrder(ctx, query)
	if status.Code(err) == codes.NotFound {
		// A missing command receipt is not a missing hold. Query current state
		// before dispatching the original conditional command.
		current, lookupErr := a.Client.GetVoucherHoldByOrder(ctx, &api.VoucherHoldCommand{OrderId: s.OrderID, ContractVersion: contract, ExpectedHoldId: &id})
		if lookupErr != nil {
			return core.VoucherDecision{}, lookupErr
		}
		if !validOutcome(current, s.OrderID) || current.HoldId != id || !validMoney(current.DiscountAmount, s.Voucher.Amount) {
			return core.VoucherDecision{}, fmt.Errorf("invalid voucher finalization lookup")
		}
		if current.State != "HELD" {
			return core.VoucherDecision{}, rejected()
		}
		query.ExpectedVersion = &version
		r, err = a.Client.FinalizeVoucherHold(ctx, query)
		if err != nil {
			return core.VoucherDecision{}, err
		}
	} else if err != nil {
		return core.VoucherDecision{}, err
	}
	if r.GetState() == "EXPIRED" || r.GetState() == "RELEASED" || r.GetCommand().GetResult() == "REJECTED" {
		return core.VoucherDecision{}, rejected()
	}
	if !validOutcome(r, s.OrderID) || r.State != "FINALIZED" || r.HoldId != id || !validMoney(r.DiscountAmount, s.Voucher.Amount) || !commandProof(r, k, h, "FinalizeVoucherHold") {
		return core.VoucherDecision{}, fmt.Errorf("invalid voucher finalization evidence")
	}
	return core.VoucherDecision{HoldID: id, DecisionID: r.Command.DecisionId}, nil
}
