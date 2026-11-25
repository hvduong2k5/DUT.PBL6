package postgres

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/omamx/promotion-service/internal/domain"
)

func TestValidateCouponInput(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	valid := CouponInput{Code: "SAVE_10", DiscountType: domain.DiscountPercentage, DiscountValue: 10, StartsAt: now, EndsAt: now.Add(time.Hour)}
	tests := []struct {
		name    string
		edit    func(*CouponInput)
		wantErr bool
	}{
		{name: "valid percentage"},
		{name: "code too short", edit: func(v *CouponInput) { v.Code = "AB" }, wantErr: true},
		{name: "lowercase code rejected by repository validator", edit: func(v *CouponInput) { v.Code = "save10" }, wantErr: true},
		{name: "invalid code punctuation", edit: func(v *CouponInput) { v.Code = "SAVE 10" }, wantErr: true},
		{name: "end must follow start", edit: func(v *CouponInput) { v.EndsAt = v.StartsAt }, wantErr: true},
		{name: "negative minimum", edit: func(v *CouponInput) { v.MinimumOrderAmount = -1 }, wantErr: true},
		{name: "percentage maximum", edit: func(v *CouponInput) { v.DiscountValue = 101 }, wantErr: true},
		{name: "fixed amount must be positive", edit: func(v *CouponInput) { v.DiscountType, v.DiscountValue = domain.DiscountFixedAmount, 0 }, wantErr: true},
		{name: "free shipping has no value", edit: func(v *CouponInput) { v.DiscountType, v.DiscountValue = domain.DiscountFreeShipping, 1 }, wantErr: true},
		{name: "max discount must be positive", edit: func(v *CouponInput) { v.MaxDiscountAmount = ptr(int64(0)) }, wantErr: true},
		{name: "customer limit cannot exceed total limit", edit: func(v *CouponInput) { v.MaxRedemptions, v.MaxRedemptionsPerCustomer = ptr(int64(3)), ptr(int64(4)) }, wantErr: true},
		{name: "positive limits accepted", edit: func(v *CouponInput) { v.MaxRedemptions, v.MaxRedemptionsPerCustomer = ptr(int64(5)), ptr(int64(2)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			if tt.edit != nil {
				tt.edit(&input)
			}
			err := validateCouponInput(input)
			if tt.wantErr && !errors.Is(err, domain.ErrCouponConfigInvalid) {
				t.Fatalf("error = %v, want ErrCouponConfigInvalid", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoyaltyPolicyCalculations(t *testing.T) {
	tests := []struct {
		name    string
		policy  LoyaltyPolicy
		amount  int64
		want    int64
		wantErr bool
		refund  bool
	}{
		{name: "floor purchase earn", policy: LoyaltyPolicy{EarnRateBasisPoints: 100}, amount: 199, want: 1},
		{name: "purchase no points under floor", policy: LoyaltyPolicy{EarnRateBasisPoints: 100}, amount: 99, want: 0},
		{name: "purchase percentage", policy: LoyaltyPolicy{EarnRateBasisPoints: 250}, amount: 10000, want: 250},
		{name: "purchase negative amount rejected", policy: LoyaltyPolicy{EarnRateBasisPoints: 100}, amount: -1, wantErr: true},
		{name: "earn rate above 100 percent rejected", policy: LoyaltyPolicy{EarnRateBasisPoints: 10001}, amount: 100, wantErr: true},
		{name: "refund conversion integer ratio", policy: LoyaltyPolicy{PointsPerVNDNumerator: 1, PointsPerVNDDenominator: 1}, amount: 500, want: 500, refund: true},
		{name: "refund rounds down", policy: LoyaltyPolicy{PointsPerVNDNumerator: 1, PointsPerVNDDenominator: 3}, amount: 5, want: 1, refund: true},
		{name: "refund zero rejected", policy: LoyaltyPolicy{PointsPerVNDNumerator: 1, PointsPerVNDDenominator: 1}, amount: 0, wantErr: true, refund: true},
		{name: "refund overflow rejected", policy: LoyaltyPolicy{PointsPerVNDNumerator: math.MaxInt64, PointsPerVNDDenominator: 1}, amount: 2, wantErr: true, refund: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int64
			var err error
			if tt.refund {
				got, err = tt.policy.CalculateRefundPoints(tt.amount)
			} else {
				got, err = tt.policy.CalculatePurchasePoints(tt.amount)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPointsToVNDIntegerRate(t *testing.T) {
	for _, tt := range []struct {
		name                         string
		numerator, denominator, want int64
		wantErr                      bool
	}{
		{name: "integer rate", numerator: 4, denominator: 2, want: 2},
		{name: "fractional rate is not exposed", numerator: 1, denominator: 2, wantErr: true},
		{name: "zero denominator rejected", numerator: 1, denominator: 0, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (LoyaltyPolicy{PointsPerVNDNumerator: tt.numerator, PointsPerVNDDenominator: tt.denominator}).PointsToVNDIntegerRate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %d, %v; want %d", got, err, tt.want)
			}
		})
	}
}

func TestValidateLoyaltyEntry(t *testing.T) {
	basis := int64(100)
	base := LoyaltyEntryInput{CustomerID: "customer-1", EntryType: "PURCHASE_EARN", PointsDelta: 1, BasisAmountVND: &basis, SourceType: "ORDER_COMPLETION", SourceID: "order-1", SourceDecisionID: "decision-1", IdempotencyKey: "earn-order-1-0001", PolicyVersion: "loyalty-dev-v1"}
	tests := []struct {
		name    string
		edit    func(*LoyaltyEntryInput)
		wantErr bool
	}{
		{name: "valid purchase earn"},
		{name: "refund requires consent", edit: func(v *LoyaltyEntryInput) {
			v.EntryType = "REFUND_CONVERSION"
			v.SourceType = "REFUND_DECISION"
			v.SourceID = "refund-1"
			v.SourceDecisionID = "refund-decision"
		}, wantErr: true},
		{name: "refund with consent", edit: func(v *LoyaltyEntryInput) {
			v.EntryType = "REFUND_CONVERSION"
			v.SourceType = "REFUND_DECISION"
			v.SourceID = "refund-1"
			v.SourceDecisionID = "refund-decision"
			v.ConsentReferenceID = "consent-1"
		}},
		{name: "credit must be positive", edit: func(v *LoyaltyEntryInput) { v.PointsDelta = -1 }, wantErr: true},
		{name: "debit must reference original entry", edit: func(v *LoyaltyEntryInput) {
			v.EntryType = "REVERSAL"
			v.SourceType = "ORDER_OR_RETURN_REVERSAL"
			v.PointsDelta = -1
		}, wantErr: true},
		{name: "debit with reference", edit: func(v *LoyaltyEntryInput) {
			v.EntryType = "REVERSAL"
			v.SourceType = "ORDER_OR_RETURN_REVERSAL"
			v.PointsDelta = -1
			v.ReferenceEntryID = "entry-1"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			if tt.edit != nil {
				tt.edit(&input)
			}
			err := validateLoyaltyEntry(input)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func ptr(v int64) *int64 { return &v }
