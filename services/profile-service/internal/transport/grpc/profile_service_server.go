package grpc

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	commonv1 "github.com/omamx/profile-service/internal/gen/common/v1"
	profilev1 "github.com/omamx/profile-service/internal/gen/profile/v1"
	"github.com/omamx/profile-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProfileGRPCServer struct {
	profilev1.UnimplementedProfileServiceServer
	customerUsecase *usecase.CustomerUsecase
	addressUsecase  *usecase.AddressUsecase
}

func NewProfileGRPCServer(c *usecase.CustomerUsecase, a *usecase.AddressUsecase) *ProfileGRPCServer {
	return &ProfileGRPCServer{customerUsecase: c, addressUsecase: a}
}
func customerUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "customer_id must be a non-zero UUID")
	}
	return id, nil
}
func rpcError(err error) error {
	var validation *domain.ValidationError
	switch {
	case errors.Is(err, domain.ErrAddressNotFound), errors.Is(err, domain.ErrCustomerNotFound):
		return status.Error(codes.NotFound, "resource not found")
	case errors.As(err, &validation):
		return status.Error(codes.InvalidArgument, validation.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	default:
		return status.Error(codes.Internal, "profile storage unavailable")
	}
}
func wireAddress(a *domain.ShippingAddress) *commonv1.Address {
	return &commonv1.Address{RecipientName: a.RecipientName, PhoneNumber: a.PhoneNumber, StreetAddress: a.StreetAddress, Ward: a.WardName, Province: a.ProvinceName, CountryCode: "VN"}
}
func (s *ProfileGRPCServer) GetDeliveryAddress(ctx context.Context, req *profilev1.GetDeliveryAddressRequest) (*profilev1.GetDeliveryAddressResponse, error) {
	customerID, err := customerUUID(req.GetCustomerId())
	if err != nil {
		return nil, err
	}
	var addressID uuid.UUID
	if req.GetAddressId() != "" {
		addressID, err = uuid.Parse(req.GetAddressId())
		if err != nil || addressID == uuid.Nil {
			return nil, status.Error(codes.InvalidArgument, "invalid address_id")
		}
	}
	address, err := s.addressUsecase.GetDeliveryAddressCheckout(ctx, addressID, customerID)
	if err != nil {
		return nil, rpcError(err)
	}
	return &profilev1.GetDeliveryAddressResponse{AddressId: address.ID.String(), Address: wireAddress(address), IsDefault: address.IsDefault, Label: address.Label, WardCode: address.WardCode, ProvinceCode: address.ProvinceCode, Latitude: address.Latitude, Longitude: address.Longitude, Version: int32(address.Version), CustomerId: address.CustomerID.String()}, nil
}
func (s *ProfileGRPCServer) GetCustomerProfile(ctx context.Context, req *profilev1.GetCustomerProfileRequest) (*profilev1.GetCustomerProfileResponse, error) {
	id, err := customerUUID(req.GetCustomerId())
	if err != nil {
		return nil, err
	}
	p, err := s.customerUsecase.GetProfile(ctx, id)
	if err != nil {
		return nil, rpcError(err)
	}
	// Loyalty belongs to Promotion; the legacy loyalty fields remain unset.
	return &profilev1.GetCustomerProfileResponse{Profile: &profilev1.CustomerProfileDetail{CustomerId: p.ID.String(), FullName: p.FullName, PhoneNumber: p.PhoneNumber, Email: &p.Email, CreatedAt: timestamppb.New(p.CreatedAt), UpdatedAt: timestamppb.New(p.UpdatedAt)}}, nil
}
func (s *ProfileGRPCServer) ListDeliveryAddresses(ctx context.Context, req *profilev1.ListDeliveryAddressesRequest) (*profilev1.ListDeliveryAddressesResponse, error) {
	id, err := customerUUID(req.GetCustomerId())
	if err != nil {
		return nil, err
	}
	addresses, err := s.addressUsecase.ListAddresses(ctx, id)
	if err != nil {
		return nil, rpcError(err)
	}
	response := &profilev1.ListDeliveryAddressesResponse{}
	for _, a := range addresses {
		response.Addresses = append(response.Addresses, &profilev1.DeliveryAddressEntry{AddressId: a.ID.String(), Address: wireAddress(a), IsDefault: a.IsDefault, Label: a.Label, WardCode: a.WardCode, ProvinceCode: a.ProvinceCode, Version: int32(a.Version)})
	}
	return response, nil
}
