package grpcapi

import (
	"context"
	"testing"

	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	commonv1 "github.com/omamx/shipping-service/internal/gen/common/v1"
	shippingv1 "github.com/omamx/shipping-service/internal/gen/shipping/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCalculateShippingFee(t *testing.T) {
	server := NewServer(application.NewService(nil, carrier.NewRegistry(carrier.NewFake())))
	response, err := server.CalculateShippingFee(context.Background(), &shippingv1.CalculateShippingFeeRequest{
		IdempotencyKey: "quote-1",
		DestinationAddress: &commonv1.Address{
			RecipientName: "Nguyen Van A", PhoneNumber: "0905000000",
			StreetAddress: "1 Nguyen Hue", Ward: "Thuan Hoa",
			District: "Phu Xuan", Province: "Hue", CountryCode: "VN",
		},
		TotalWeight:      &commonv1.Weight{DecimalValue: "1.2", Unit: commonv1.WeightUnit_WEIGHT_UNIT_KILOGRAM},
		ParcelDimensions: &shippingv1.ParcelDimensions{LengthCm: 30, WidthCm: 20, HeightCm: 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetCarrier() != carrier.FakeCode || response.GetShippingFee().GetUnits() <= 0 || response.GetEstimatedDeliveryDays() != 3 {
		t.Fatalf("unexpected quote response: %+v", response)
	}
}

func TestAuthUnaryInterceptor(t *testing.T) {
	interceptor := AuthUnaryInterceptor("secret")
	handlerCalled := false
	handler := func(context.Context, any) (any, error) {
		handlerCalled = true
		return "ok", nil
	}
	if _, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	response, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil || response != "ok" || !handlerCalled {
		t.Fatalf("authenticated call failed: response=%v err=%v", response, err)
	}
}
