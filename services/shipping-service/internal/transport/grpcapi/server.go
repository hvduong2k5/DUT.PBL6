package grpcapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
	commonv1 "github.com/omamx/shipping-service/internal/gen/common/v1"
	shippingv1 "github.com/omamx/shipping-service/internal/gen/shipping/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	shippingv1.UnimplementedShippingServiceServer
	service *application.Service
}

func NewServer(service *application.Service) *Server { return &Server{service: service} }

func (s *Server) CalculateShippingFee(ctx context.Context, request *shippingv1.CalculateShippingFeeRequest) (*shippingv1.CalculateShippingFeeResponse, error) {
	if request.GetDestinationAddress() == nil || request.GetTotalWeight() == nil {
		return nil, status.Error(codes.InvalidArgument, "destination_address and total_weight are required")
	}
	parcel, err := parcelFromProto(request.GetTotalWeight(), request.GetParcelDimensions(), request.GetIsFragile())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	carrierCode := request.GetPreferredCarrier()
	if carrierCode == "" {
		carrierCode = carrier.FakeCode
	}
	quote, err := s.service.Quote(ctx, carrierCode, carrier.QuoteRequest{
		Destination: recipientFromProto(request.GetDestinationAddress()), Parcel: parcel,
		ServiceCode: request.GetServiceCode(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &shippingv1.CalculateShippingFeeResponse{
		ShippingFee: &commonv1.Money{CurrencyCode: "VND", Units: quote.FeeVND},
		Carrier:     quote.CarrierCode, ServiceName: quote.ServiceName,
		EstimatedDeliveryDays: int32(max(1, int(math.Ceil(time.Until(quote.EstimatedDeliveryAt).Hours()/24)))),
		ServiceCode:           quote.ServiceCode, QuoteId: quote.CarrierCode + ":" + request.GetIdempotencyKey(),
		QuoteExpiresAt: timestamppb.New(quote.ExpiresAt), EstimatedDeliveryTime: timestamppb.New(quote.EstimatedDeliveryAt),
	}, nil
}

func (s *Server) CreateShipment(ctx context.Context, request *shippingv1.CreateShipmentRequest) (*shippingv1.CreateShipmentResponse, error) {
	if request.GetDestinationAddress() == nil || request.GetParcelWeight() == nil {
		return nil, status.Error(codes.InvalidArgument, "destination_address and parcel_weight are required")
	}
	parcel, err := parcelFromProto(request.GetParcelWeight(), request.GetParcelDimensions(), request.GetIsFragile())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	carrierCode := request.GetCarrier()
	if carrierCode == "" {
		carrierCode = carrier.FakeCode
	}
	serviceCode := request.GetServiceCode()
	if serviceCode == "" {
		serviceCode = "STANDARD"
	}
	shipment, err := s.service.CreateShipment(ctx, domain.CreateShipmentCommand{
		IdempotencyKey: request.GetIdempotencyKey(), OrderID: request.GetOrderId(), PackageID: request.GetPackageId(),
		SealCode: request.GetSealCode(), CarrierCode: carrierCode, ServiceCode: serviceCode,
		Recipient: recipientFromProto(request.GetDestinationAddress()), Parcel: parcel,
		CODAmountVND: moneyUnits(request.GetCodAmount()), CheckoutFeeVND: moneyUnits(request.GetCheckoutShippingFee()),
		QuoteID:       request.GetQuoteId(),
		ShippingNotes: request.GetShippingNotes(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	response := &shippingv1.CreateShipmentResponse{
		TrackingCode: shipment.TrackingCode, CarrierOrderId: shipment.CarrierOrderID,
		ShipmentId: shipment.ID.String(), Status: statusToProto(shipment.Status),
		ProviderEstimatedFee: &commonv1.Money{CurrencyCode: "VND", Units: shipment.ProviderEstimatedFeeVND},
	}
	if shipment.LabelReference != "" {
		response.ShippingLabelReference = &shipment.LabelReference
	}
	if shipment.EstimatedPickupAt != nil {
		response.EstimatedPickupTime = timestamppb.New(*shipment.EstimatedPickupAt)
	}
	if shipment.EstimatedDeliveryAt != nil {
		response.EstimatedDeliveryTime = timestamppb.New(*shipment.EstimatedDeliveryAt)
	}
	return response, nil
}

func (s *Server) TrackShipment(ctx context.Context, request *shippingv1.TrackShipmentRequest) (*shippingv1.TrackShipmentResponse, error) {
	if request.GetTrackingCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "tracking_code is required")
	}
	carrierCode := carrierCodeFromTracking(request.GetTrackingCode())
	shipment, timeline, err := s.service.TrackByCode(ctx, carrierCode, request.GetTrackingCode())
	if err != nil {
		return nil, toStatus(err)
	}
	response := &shippingv1.TrackShipmentResponse{
		TrackingCode: shipment.TrackingCode, Carrier: shipment.CarrierCode,
		CurrentStatus: statusToProto(shipment.Status), ShipmentId: shipment.ID.String(),
		OrderId: shipment.OrderID, PackageId: shipment.PackageID,
	}
	if shipment.EstimatedDeliveryAt != nil {
		response.EstimatedDeliveryTime = timestamppb.New(*shipment.EstimatedDeliveryAt)
	}
	for _, event := range timeline {
		if event.MappingResult != "APPLIED" {
			continue
		}
		response.Checkpoints = append(response.Checkpoints, &shippingv1.TrackingCheckpoint{
			Timestamp: timestamppb.New(event.EventAt), Location: event.Location,
			Description: event.PublicDescription, EventId: event.ExternalEventID,
			ProviderStatusCode: event.ProviderStatusCode, ReceivedAt: timestamppb.New(event.ReceivedAt),
		})
	}
	return response, nil
}

func AuthUnaryInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		values := metadata.ValueFromIncomingContext(ctx, "authorization")
		provided := ""
		if len(values) > 0 {
			provided = strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer "))
		}
		if len(provided) != len(token) || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "valid service authentication is required")
		}
		return handler(ctx, request)
	}
}

func parcelFromProto(weight *commonv1.Weight, dimensions *shippingv1.ParcelDimensions, fragile bool) (domain.Parcel, error) {
	value, err := strconv.ParseFloat(weight.GetDecimalValue(), 64)
	if err != nil || value <= 0 {
		return domain.Parcel{}, fmt.Errorf("parcel weight must be a positive decimal")
	}
	grams := int64(value)
	if weight.GetUnit() == commonv1.WeightUnit_WEIGHT_UNIT_KILOGRAM {
		grams = int64(value * 1000)
	}
	parcel := domain.Parcel{WeightGrams: grams, LengthCM: 20, WidthCM: 20, HeightCM: 10, Fragile: fragile}
	if dimensions != nil {
		parcel.LengthCM, parcel.WidthCM, parcel.HeightCM = dimensions.GetLengthCm(), dimensions.GetWidthCm(), dimensions.GetHeightCm()
	}
	return parcel, parcel.Validate()
}

func recipientFromProto(address *commonv1.Address) domain.Recipient {
	return domain.Recipient{Name: address.GetRecipientName(), Phone: address.GetPhoneNumber(), Street: address.GetStreetAddress(), Ward: address.GetWard(), District: address.GetDistrict(), Province: address.GetProvince(), CountryCode: address.GetCountryCode(), PostalCode: address.GetPostalCode()}
}

func moneyUnits(money *commonv1.Money) int64 {
	if money == nil {
		return 0
	}
	return money.GetUnits()
}

func carrierCodeFromTracking(code string) string {
	upper := strings.ToUpper(code)
	for _, prefix := range []string{"VIETTEL_POST", "VIETTELPOST", "GHN", "FAKE"} {
		if strings.HasPrefix(upper, prefix) {
			return prefix
		}
	}
	return carrier.FakeCode
}

func statusToProto(value domain.ShipmentStatus) shippingv1.ShipmentStatus {
	switch value {
	case domain.StatusCreating:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_CREATING
	case domain.StatusCreationFailed:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_CREATION_FAILED
	case domain.StatusCreated:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_CREATED
	case domain.StatusReadyForHandover:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_READY_FOR_HANDOVER
	case domain.StatusHandedOver:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_HANDED_OVER
	case domain.StatusInTransit:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_IN_TRANSIT
	case domain.StatusOutForDelivery:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_OUT_FOR_DELIVERY
	case domain.StatusDelivered:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_DELIVERED
	case domain.StatusDeliveryFailed:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_DELIVERY_FAILED
	case domain.StatusReturning:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_RETURNING
	case domain.StatusReturned:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_RETURNED
	case domain.StatusCancelled:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_CANCELLED
	default:
		return shippingv1.ShipmentStatus_SHIPMENT_STATUS_UNSPECIFIED
	}
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrShipmentNotFound), errors.Is(err, domain.ErrTrackingNotFound),
		errors.Is(err, domain.ErrShipmentRevisionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidShipment), errors.Is(err, domain.ErrUnsupportedCarrier):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrConcurrentUpdate):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, domain.ErrIdempotencyConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.As(err, new(*application.ProviderError)):
		return status.Error(codes.Unavailable, "carrier operation failed")
	default:
		return status.Error(codes.Internal, "shipping operation failed")
	}
}
