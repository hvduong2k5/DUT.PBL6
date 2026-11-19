package app

import (
	"context"
	"errors"
	"testing"

	"github.com/omamx/order-service/internal/domain/core"
	api "github.com/omamx/order-service/internal/gen/promotion/v1"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/promotionprovider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type discountClient struct {
	api.PromotionServiceClient
	err error
}

func (c discountClient) CalculateDiscount(context.Context, *api.CalculateDiscountRequest, ...grpc.CallOption) (*api.CalculateDiscountResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &api.CalculateDiscountResponse{IsApplicable: false}, nil
}

func TestVoucherRevalidationRetriesInfrastructureButRejectsBusinessFailure(t *testing.T) {
	customer := "customer-1"
	terms := core.SnapshotInput{Owner: core.Owner{CustomerID: &customer}, Voucher: &core.VoucherTerms{Code: "WELCOME"}}
	for _, code := range []codes.Code{codes.Unavailable, codes.Internal, codes.DeadlineExceeded} {
		t.Run(code.String(), func(t *testing.T) {
			s := &Service{Promotion: &promotionprovider.Adapter{Client: discountClient{err: status.Error(code, "dependency failed")}}}
			err := s.revalidateVoucher(context.Background(), terms)
			if err == nil || errors.Is(err, persistence.ErrTermsChanged) {
				t.Fatal("infrastructure outage must not cancel checkout", err)
			}
			if _, httpStatus := s.ErrorCode(err); httpStatus != 503 {
				t.Fatal("dependency failure must remain retryable", err)
			}
		})
	}
	s := &Service{Promotion: &promotionprovider.Adapter{Client: discountClient{}}}
	if err := s.revalidateVoucher(context.Background(), terms); !errors.Is(err, persistence.ErrTermsChanged) {
		t.Fatal("actual voucher rejection must invalidate the quote", err)
	}
}
