package contract_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	profilev1 "github.com/omamx/profile-service/internal/gen/profile/v1"
	"github.com/omamx/profile-service/internal/repository"
	transport "github.com/omamx/profile-service/internal/transport/grpc"
	"github.com/omamx/profile-service/internal/usecase"
	"github.com/omamx/profile-service/tests/testhelper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"testing"
	"time"
)

func TestRealGRPCProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("real gRPC and PostgreSQL integration")
	}
	pool, cleanup := testhelper.SetupPostgresContainer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	customers := repository.NewCustomerRepository(pool)
	addresses := repository.NewAddressRepository(pool)
	id := uuid.New()
	now := time.Now().UTC()
	require.NoError(t, customers.Create(ctx, &domain.CustomerProfile{ID: id, UserID: uuid.New(), FullName: "GRPC Customer", PhoneNumber: "0905111111", Version: 1, Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}))
	a := &domain.ShippingAddress{ID: uuid.New(), CustomerID: id, RecipientName: "Recipient", PhoneNumber: "0905111111", StreetAddress: "15 Lê Lợi", WardCode: "WARD-TH-001", ProvinceCode: "75", Label: "HOME", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, addresses.CreateAddress(ctx, a))
	server, err := transport.NewServer("0", transport.NewProfileGRPCServer(usecase.NewCustomerUsecase(customers, nil, pool), usecase.NewAddressUsecase(addresses, usecase.NewAddressFuzzyMatcher(), nil, pool)), "contract-test-only")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- server.Start() }()
	defer func() { server.Stop(); require.NoError(t, <-done) }()
	_, port, err := net.SplitHostPort(server.Addr().String())
	require.NoError(t, err)
	connection, err := grpc.NewClient("127.0.0.1:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer connection.Close()
	client := profilev1.NewProfileServiceClient(connection)
	_, err = client.GetDeliveryAddress(ctx, &profilev1.GetDeliveryAddressRequest{CustomerId: id.String(), AddressId: a.ID.String()})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	authenticated := metadata.AppendToOutgoingContext(ctx, "x-internal-token", "contract-test-only")
	response, err := client.GetDeliveryAddress(authenticated, &profilev1.GetDeliveryAddressRequest{CustomerId: id.String(), AddressId: a.ID.String()})
	require.NoError(t, err)
	require.Equal(t, a.ID.String(), response.AddressId)
	require.Equal(t, id.String(), response.CustomerId)
	require.Equal(t, a.WardCode, response.WardCode)
	require.Equal(t, a.ProvinceCode, response.ProvinceCode)
	require.Equal(t, int32(1), response.Version)
	require.Equal(t, a.WardName, response.Address.Ward)
	require.Equal(t, "", response.Address.District)
	require.True(t, response.IsDefault)
	_, err = client.GetDeliveryAddress(authenticated, &profilev1.GetDeliveryAddressRequest{CustomerId: uuid.New().String(), AddressId: a.ID.String()})
	require.Equal(t, codes.NotFound, status.Code(err))
	_, err = client.GetDeliveryAddress(authenticated, &profilev1.GetDeliveryAddressRequest{AddressId: a.ID.String()})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	defaultAddress, err := client.GetDeliveryAddress(authenticated, &profilev1.GetDeliveryAddressRequest{CustomerId: id.String()})
	require.NoError(t, err)
	require.Equal(t, a.ID.String(), defaultAddress.AddressId)
	profile, err := client.GetCustomerProfile(authenticated, &profilev1.GetCustomerProfileRequest{CustomerId: id.String()})
	require.NoError(t, err)
	require.Equal(t, id.String(), profile.Profile.CustomerId)
	list, err := client.ListDeliveryAddresses(authenticated, &profilev1.ListDeliveryAddressesRequest{CustomerId: id.String()})
	require.NoError(t, err)
	require.Len(t, list.Addresses, 1)
	require.NoError(t, addresses.DeleteAddress(ctx, id, a.ID))
	_, err = client.GetDeliveryAddress(authenticated, &profilev1.GetDeliveryAddressRequest{CustomerId: id.String(), AddressId: a.ID.String()})
	require.Equal(t, codes.NotFound, status.Code(err))
}
