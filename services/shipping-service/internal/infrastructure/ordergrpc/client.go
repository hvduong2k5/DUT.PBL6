package ordergrpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/domain"
	commonv1 "github.com/omamx/shipping-service/internal/gen/common/v1"
	orderv1 "github.com/omamx/shipping-service/internal/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	connection *grpc.ClientConn
	client     orderv1.OrderServiceClient
	token      string
}

func New(endpoint, token string) (*Client, error) {
	connection, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create Order gRPC client: %w", err)
	}
	return &Client{connection: connection, client: orderv1.NewOrderServiceClient(connection), token: token}, nil
}

func (c *Client) Close() error { return c.connection.Close() }

func (c *Client) GetShippingSnapshot(ctx context.Context, orderID, packageID string) (application.OrderShippingSnapshot, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token)
	response, err := c.client.GetShippingSnapshot(ctx, &orderv1.GetShippingSnapshotRequest{
		OrderId: orderID, PackageId: packageID, RequestingService: "shipping-service",
	})
	if err != nil {
		return application.OrderShippingSnapshot{}, err
	}
	if response.GetError() != nil {
		return application.OrderShippingSnapshot{}, fmt.Errorf("Order rejected shipping snapshot: %s", response.GetError().GetUserMessage())
	}
	return application.OrderShippingSnapshot{
		OrderID: response.GetOrderId(), PackageID: response.GetPackageId(),
		Recipient:           recipientFromProto(response.GetRecipient()),
		CheckoutFeeVND:      moneyUnits(response.GetCheckoutShippingFee()),
		CODAmountVND:        moneyUnits(response.GetCodAmount()),
		CarrierCode:         strings.TrimSpace(response.GetCarrierCode()),
		ServiceCode:         strings.TrimSpace(response.GetServiceCode()),
		EligibleForShipping: response.GetEligibleForShipping(),
	}, nil
}

func recipientFromProto(address *commonv1.Address) domain.Recipient {
	if address == nil {
		return domain.Recipient{}
	}
	return domain.Recipient{
		Name: address.GetRecipientName(), Phone: address.GetPhoneNumber(),
		Street: address.GetStreetAddress(), Ward: address.GetWard(),
		District: address.GetDistrict(), Province: address.GetProvince(),
		CountryCode: address.GetCountryCode(), PostalCode: address.GetPostalCode(),
	}
}

func moneyUnits(money *commonv1.Money) int64 {
	if money == nil {
		return 0
	}
	return money.GetUnits()
}
