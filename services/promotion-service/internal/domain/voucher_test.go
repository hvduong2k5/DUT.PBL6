package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestCouponDiscount(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cap := int64(1200)
	base := Coupon{Status: "ACTIVE", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour)}
	tests := []struct {
		name        string
		coupon      Coupon
		subtotal    int64
		shippingFee int64
		currency    string
		at          time.Time
		want        int64
		wantErr     error
		wantAnyErr  bool
	}{
		{name: "fixed amount", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 500}, subtotal: 2000, currency: "VND", at: now, want: 500},
		{name: "percentage and maximum cap", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountPercentage, DiscountValue: 50, MaxDiscountAmount: &cap}, subtotal: 5000, currency: "VND", at: now, want: cap},
		{name: "percentage avoids multiplication overflow", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountPercentage, DiscountValue: 100}, subtotal: math.MaxInt64, currency: "VND", at: now, want: math.MaxInt64},
		{name: "fixed amount capped at subtotal", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 900}, subtotal: 400, currency: "VND", at: now, want: 400},
		{name: "free shipping capped at fee", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFreeShipping}, subtotal: 1000, shippingFee: 700, currency: "VND", at: now, want: 700},
		{name: "minimum not met", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 10, MinimumOrderAmount: 100}, subtotal: 99, currency: "VND", at: now, wantErr: ErrMinimumNotMet},
		{name: "not started", coupon: Coupon{Status: base.Status, StartsAt: now, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 10}, subtotal: 100, currency: "VND", at: now.Add(-time.Nanosecond), wantErr: ErrVoucherInactive},
		{name: "inactive status", coupon: Coupon{Status: "DRAFT", StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 10}, subtotal: 100, currency: "VND", at: now, wantErr: ErrVoucherInactive},
		{name: "end is exclusive", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: now, DiscountType: DiscountFixedAmount, DiscountValue: 10}, subtotal: 100, currency: "VND", at: now, wantErr: ErrVoucherExpired},
		{name: "unsupported currency", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 10}, subtotal: 100, currency: "USD", at: now, wantAnyErr: true},
		{name: "negative subtotal", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: 10}, subtotal: -1, currency: "VND", at: now, wantAnyErr: true},
		{name: "invalid percentage", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountPercentage, DiscountValue: 101}, subtotal: 100, currency: "VND", at: now, wantAnyErr: true},
		{name: "unsupported discount type", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: "BOGO"}, subtotal: 100, currency: "VND", at: now, wantAnyErr: true},
		{name: "negative fixed discount", coupon: Coupon{Status: base.Status, StartsAt: base.StartsAt, EndsAt: base.EndsAt, DiscountType: DiscountFixedAmount, DiscountValue: -1}, subtotal: 100, currency: "VND", at: now, wantAnyErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.coupon.Discount(tt.subtotal, tt.shippingFee, tt.currency, tt.at)
			if tt.wantAnyErr {
				if err == nil {
					t.Fatal("Discount() expected an error")
				}
				return
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Discount() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Discount() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Discount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeCode(t *testing.T) {
	if got := NormalizeCode("  spring_10  "); got != "SPRING_10" {
		t.Fatalf("NormalizeCode() = %q", got)
	}
}
