package app

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/promotionprovider"
	"github.com/omamx/order-service/internal/shippingprovider"
)

var voucherCodePattern = regexp.MustCompile(`^[A-Z0-9_-]{3,50}$`)

func (s *Service) EnablePromotionFromEnv(ctx context.Context) error {
	target := strings.TrimSpace(os.Getenv("ORDER_PROMOTION_GRPC"))
	if target == "" {
		for _, name := range []string{"ORDER_PROMOTION_TOKEN_FILE", "ORDER_PROMOTION_CA_FILE", "ORDER_PROMOTION_SERVER_NAME", "ORDER_PROMOTION_ALLOW_LOCAL_INSECURE"} {
			if os.Getenv(name) != "" {
				return errors.New("Promotion configuration requires ORDER_PROMOTION_GRPC")
			}
		}
		return nil
	}
	if s.CoreCheckout == nil {
		return errors.New("Promotion requires the core checkout runtime")
	}
	token, err := shippingprovider.TokenFile(os.Getenv("ORDER_PROMOTION_TOKEN_FILE"))
	if err != nil {
		return err
	}
	a, conn, err := promotionprovider.Dial(ctx, target, token, os.Getenv("ORDER_PROMOTION_CA_FILE"), os.Getenv("ORDER_PROMOTION_SERVER_NAME"), s.C.Mode == "sandbox" && os.Getenv("ORDER_PROMOTION_ALLOW_LOCAL_INSECURE") == "true")
	if err != nil {
		return err
	}
	s.Promotion = a
	s.CoreCheckout.Promotion = a
	s.Payment.Promotion = a
	previous := s.PaymentClose
	s.PaymentClose = func() error {
		if previous != nil {
			_ = previous()
		}
		return conn.Close()
	}
	return nil
}

func (s *Service) applyVoucherQuote(ctx context.Context, q domain.Quote, terms *core.SnapshotInput) error {
	if q.Voucher == "" {
		return nil
	}
	if s.Promotion == nil || terms.Owner.CustomerID == nil {
		return domain.E("VOUCHER_FEATURE_DISABLED", 422)
	}
	v, err := s.Promotion.Quote(ctx, q.Voucher, *terms.Owner.CustomerID, terms.Money.Subtotal, terms.Money.ShippingFee)
	if err != nil {
		return err
	}
	terms.SchemaVersion = "1.1"
	terms.Voucher = v
	terms.Policies.Promotion = &v.Policy
	if v.Kind == "FREE_SHIPPING" {
		terms.Money.ShippingDiscount = v.Amount
	} else {
		terms.Money.MerchandiseDiscount = v.Amount
	}
	terms.Money.FinalAmount = terms.Money.Subtotal + terms.Money.ShippingFee - v.Amount
	if terms.Money.FinalAmount < 1 {
		return domain.E("ZERO_PAYMENT_NOT_SUPPORTED", 422)
	}
	if terms.Payment.ExpiresAt != nil && v.ValidUntil.Before(*terms.Payment.ExpiresAt) {
		expiry := v.ValidUntil
		terms.Payment.ExpiresAt = &expiry
	}
	return nil
}

func (s *Service) revalidateVoucher(ctx context.Context, terms core.SnapshotInput) error {
	if terms.Voucher == nil {
		return nil
	}
	if s.Promotion == nil || terms.Owner.CustomerID == nil {
		return persistence.ErrCapability
	}
	v, err := s.Promotion.Quote(ctx, terms.Voucher.Code, *terms.Owner.CustomerID, terms.Money.Subtotal, terms.Money.ShippingFee)
	if err != nil {
		if _, status := s.ErrorCode(err); status == 422 {
			return persistence.ErrTermsChanged
		}
		return err
	}
	want := terms.Voucher
	if v.Kind != want.Kind || v.Amount != want.Amount || v.Policy != want.Policy || !v.ValidUntil.Equal(want.ValidUntil) {
		return persistence.ErrTermsChanged
	}
	return nil
}
