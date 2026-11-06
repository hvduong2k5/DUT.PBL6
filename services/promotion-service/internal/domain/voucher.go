package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type DiscountType string

const (
	DiscountFixedAmount  DiscountType = "FIXED_AMOUNT"
	DiscountPercentage   DiscountType = "PERCENTAGE"
	DiscountFreeShipping DiscountType = "FREE_SHIPPING"
)

type Coupon struct {
	ID                        string
	Code                      string
	DiscountType              DiscountType
	DiscountValue             int64
	MaxDiscountAmount         *int64
	MinimumOrderAmount        int64
	MaxRedemptions            *int64
	MaxRedemptionsPerCustomer *int64
	StartsAt                  time.Time
	EndsAt                    time.Time
	Status                    string
}

var (
	ErrVoucherNotFound      = errors.New("voucher not found")
	ErrVoucherInactive      = errors.New("voucher is not active")
	ErrVoucherExpired       = errors.New("voucher is expired")
	ErrMinimumNotMet        = errors.New("minimum order amount not met")
	ErrUsageLimit           = errors.New("voucher usage limit exceeded")
	ErrVoucherAlreadyLocked = errors.New("voucher already has a hold for this order")
	ErrIdempotencyConflict  = errors.New("idempotency key reused with different request")
	ErrHoldNotFound         = errors.New("voucher hold not found")
	ErrHoldTransition       = errors.New("invalid voucher hold state transition")
)

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (c Coupon) Discount(subtotal, shippingFee int64, currency string, now time.Time) (int64, error) {
	if currency != "VND" {
		return 0, fmt.Errorf("unsupported currency %q", currency)
	}
	if subtotal < 0 || shippingFee < 0 {
		return 0, errors.New("subtotal and shipping fee must not be negative")
	}
	if c.Status != "ACTIVE" || now.Before(c.StartsAt) {
		return 0, ErrVoucherInactive
	}
	if !now.Before(c.EndsAt) {
		return 0, ErrVoucherExpired
	}
	if subtotal < c.MinimumOrderAmount {
		return 0, ErrMinimumNotMet
	}

	var discount int64
	switch c.DiscountType {
	case DiscountFixedAmount:
		discount = c.DiscountValue
	case DiscountPercentage:
		if c.DiscountValue < 0 || c.DiscountValue > 100 {
			return 0, errors.New("percentage discount must be from 0 through 100")
		}
		// Split the calculation so subtotal*percentage cannot overflow int64.
		discount = subtotal/100*c.DiscountValue + (subtotal%100)*c.DiscountValue/100
	case DiscountFreeShipping:
		discount = shippingFee
	default:
		return 0, fmt.Errorf("unsupported discount type %q", c.DiscountType)
	}
	if c.MaxDiscountAmount != nil && discount > *c.MaxDiscountAmount {
		discount = *c.MaxDiscountAmount
	}
	cap := subtotal
	if c.DiscountType == DiscountFreeShipping {
		cap = shippingFee
	}
	if discount > cap {
		discount = cap
	}
	if discount < 0 {
		return 0, errors.New("discount must not be negative")
	}
	return discount, nil
}
