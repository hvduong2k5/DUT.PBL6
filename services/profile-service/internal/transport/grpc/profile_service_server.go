package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DTOs matching the contracted Protobuf schema with MS-04 order-service
type GetDeliveryAddressRequest struct {
	AddressID  string `json:"address_id"`
	CustomerID string `json:"customer_id"`
}

type DeliveryAddressResponse struct {
	ID            string  `json:"id"`
	CustomerID    string  `json:"customer_id"`
	RecipientName string  `json:"recipient_name"`
	PhoneNumber   string  `json:"phone_number"`
	StreetAddress string  `json:"street_address"`
	WardCode      string  `json:"ward_code"`
	WardName      string  `json:"ward_name"`
	ProvinceCode  string  `json:"province_code"`
	ProvinceName  string  `json:"province_name"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	IsDefault     bool    `json:"is_default"`
}

type ProfileGRPCServer struct {
	customerUsecase *usecase.CustomerUsecase
	addressUsecase  *usecase.AddressUsecase
}

func NewProfileGRPCServer(customerUsecase *usecase.CustomerUsecase, addressUsecase *usecase.AddressUsecase) *ProfileGRPCServer {
	return &ProfileGRPCServer{
		customerUsecase: customerUsecase,
		addressUsecase:  addressUsecase,
	}
}

// GetDeliveryAddress serves the Checkout Critical Path (SLA P99 <= 5ms).
// Queries directly via Primary Key / Default index, bypassing all fuzzy matching logic.
func (s *ProfileGRPCServer) GetDeliveryAddress(ctx context.Context, req *GetDeliveryAddressRequest) (*DeliveryAddressResponse, error) {
	var addrUUID, custUUID uuid.UUID
	var err error

	if req.AddressID != "" {
		addrUUID, err = uuid.Parse(req.AddressID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid address_id: %v", err)
		}
	}

	if req.CustomerID != "" {
		custUUID, err = uuid.Parse(req.CustomerID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid customer_id: %v", err)
		}
	}

	if addrUUID == uuid.Nil && custUUID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "either address_id or customer_id must be provided")
	}

	addr, err := s.addressUsecase.GetDeliveryAddressCheckout(ctx, addrUUID, custUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "shipping address not found: %v", err)
	}

	var lat, lon float64
	if addr.Latitude != nil {
		lat = *addr.Latitude
	}
	if addr.Longitude != nil {
		lon = *addr.Longitude
	}

	return &DeliveryAddressResponse{
		ID:            addr.ID.String(),
		CustomerID:    addr.CustomerID.String(),
		RecipientName: addr.RecipientName,
		PhoneNumber:   addr.PhoneNumber,
		StreetAddress: addr.StreetAddress,
		WardCode:      addr.WardCode,
		WardName:      addr.WardName,
		ProvinceCode:  addr.ProvinceCode,
		ProvinceName:  addr.ProvinceName,
		Latitude:      lat,
		Longitude:     lon,
		IsDefault:     addr.IsDefault,
	}, nil
}
