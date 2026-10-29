package grpcapi

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
	commonv1 "github.com/omamx/shipping-service/internal/gen/common/v1"
	shippingv2 "github.com/omamx/shipping-service/internal/gen/shipping/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const shippingV2ContractVersion = "order-3.2"

type V2Server struct {
	shippingv2.UnimplementedShippingServiceServer
	service *application.Service
}

func NewV2Server(service *application.Service) *V2Server { return &V2Server{service: service} }

func (s *V2Server) CalculateShippingFee(ctx context.Context, request *shippingv2.CalculateShippingFeeRequest) (*shippingv2.CalculateShippingFeeResponse, error) {
	if request.GetContractVersion() != shippingV2ContractVersion || strings.TrimSpace(request.GetRequestHash()) == "" {
		return nil, status.Error(codes.InvalidArgument, "contract_version order-3.2 and request_hash are required")
	}
	if request.GetDestinationAddress() == nil || request.GetTotalWeight() == nil {
		return nil, status.Error(codes.InvalidArgument, "destination_address and total_weight are required")
	}
	parcel, err := parcelFromV2Proto(request.GetTotalWeight(), request.GetParcelDimensions(), request.GetIsFragile())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	canonicalHash, err := canonicalMessageHash(request.ProtoReflect())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	quote, err := s.service.BoundQuote(ctx, domain.BoundQuoteCommand{
		IdempotencyKey: request.GetIdempotencyKey(), ContractVersion: request.GetContractVersion(),
		RequestHash: request.GetRequestHash(), CanonicalHash: canonicalHash,
		Destination: recipientFromProto(request.GetDestinationAddress()), Parcel: parcel,
		PreferredCarrier: request.GetPreferredCarrier(), ServiceCode: request.GetServiceCode(),
		OriginWarehouseID: request.GetOriginWarehouseId(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &shippingv2.CalculateShippingFeeResponse{
		ContractVersion:       shippingV2ContractVersion,
		RequestHash:           quote.RequestHash,
		QuoteVersion:          quote.Version,
		ShippingFee:           &commonv1.Money{CurrencyCode: "VND", Units: quote.FeeVND},
		Carrier:               quote.CarrierCode,
		ServiceName:           quote.ServiceName,
		EstimatedDeliveryDays: int32(max(1, int(math.Ceil(quote.EstimatedDeliveryAt.Sub(quote.CreatedAt).Hours()/24)))),
		ServiceCode:           quote.ServiceCode,
		QuoteId:               quote.ID.String(),
		QuoteExpiresAt:        timestamppb.New(quote.ExpiresAt),
		EstimatedDeliveryTime: timestamppb.New(quote.EstimatedDeliveryAt),
	}, nil
}

func (s *V2Server) CreateShipment(ctx context.Context, request *shippingv2.CreateShipmentRequest) (*shippingv2.CreateShipmentResponse, error) {
	if request.GetReadyGeneration() < 1 || request.GetOrderSnapshotVersion() < 1 ||
		strings.TrimSpace(request.GetItemsDigest()) == "" || strings.TrimSpace(request.GetSealCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ready_generation, items_digest, order_snapshot_version and seal_code are required")
	}
	if request.GetDestinationAddress() == nil || request.GetParcelWeight() == nil {
		return nil, status.Error(codes.InvalidArgument, "destination_address and parcel_weight are required")
	}
	parcel, err := parcelFromV2Proto(request.GetParcelWeight(), request.GetParcelDimensions(), request.GetIsFragile())
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
		QuoteID: request.GetQuoteId(), ReadyGeneration: request.GetReadyGeneration(),
		ItemsDigest: request.GetItemsDigest(), OrderSnapshotVersion: request.GetOrderSnapshotVersion(),
		ShippingNotes: request.GetShippingNotes(),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return shipmentResponseV2(shipment), nil
}

func (s *V2Server) TrackShipment(ctx context.Context, request *shippingv2.TrackShipmentRequest) (*shippingv2.TrackShipmentResponse, error) {
	var shipment *domain.Shipment
	var timeline []domain.TrackingEvent
	var err error
	if rawID := strings.TrimSpace(request.GetShipmentId()); rawID != "" {
		shipmentID, parseErr := uuid.Parse(rawID)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "shipment_id must be a UUID")
		}
		shipment, timeline, err = s.service.GetShipment(ctx, shipmentID)
	} else if trackingCode := strings.TrimSpace(request.GetTrackingCode()); trackingCode != "" {
		shipment, timeline, err = s.service.TrackByCode(ctx, carrierCodeFromTracking(trackingCode), trackingCode)
	} else {
		return nil, status.Error(codes.InvalidArgument, "shipment_id or tracking_code is required")
	}
	if err != nil {
		return nil, toStatus(err)
	}
	response := &shippingv2.TrackShipmentResponse{
		SealCode: shipment.SealCode, ResourceVersion: shipment.Revision,
		ReadyGeneration: shipment.ReadyGeneration, TrackingCode: shipment.TrackingCode,
		Carrier: shipment.CarrierCode, CurrentStatus: statusToV2Proto(shipment.Status),
		ShipmentId: shipment.ID.String(), OrderId: shipment.OrderID, PackageId: shipment.PackageID,
	}
	if shipment.EstimatedDeliveryAt != nil {
		response.EstimatedDeliveryTime = timestamppb.New(*shipment.EstimatedDeliveryAt)
	}
	for _, event := range timeline {
		if event.MappingResult != "APPLIED" {
			continue
		}
		response.Checkpoints = append(response.Checkpoints, &shippingv2.TrackingCheckpoint{
			Timestamp: timestamppb.New(event.EventAt), Location: event.Location,
			Description: event.PublicDescription, EventId: event.ExternalEventID,
			ProviderStatusCode: event.ProviderStatusCode, ReceivedAt: timestamppb.New(event.ReceivedAt),
		})
	}
	return response, nil
}

func (s *V2Server) GetShipmentRevision(ctx context.Context, request *shippingv2.GetShipmentRevisionRequest) (*shippingv2.ShipmentRevision, error) {
	shipmentID, err := uuid.Parse(strings.TrimSpace(request.GetShipmentId()))
	if err != nil || request.GetResourceVersion() < 1 {
		return nil, status.Error(codes.InvalidArgument, "shipment_id must be a UUID and resource_version must be positive")
	}
	revision, err := s.service.GetShipmentRevision(ctx, shipmentID, request.GetResourceVersion())
	if err != nil {
		return nil, toStatus(err)
	}
	return &shippingv2.ShipmentRevision{
		OrderId: revision.OrderID, ShipmentId: revision.ShipmentID.String(),
		PackageId: revision.PackageID, ResourceVersion: revision.ResourceVersion,
		ReadyGeneration: revision.ReadyGeneration, Kind: revision.Kind,
		SealCode: revision.SealCode, OccurredAt: timestamppb.New(revision.OccurredAt),
		SettlementVerified: revision.SettlementVerified,
	}, nil
}

func parcelFromV2Proto(weight *commonv1.Weight, dimensions *shippingv2.ParcelDimensions, fragile bool) (domain.Parcel, error) {
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

func shipmentResponseV2(shipment *domain.Shipment) *shippingv2.CreateShipmentResponse {
	response := &shippingv2.CreateShipmentResponse{
		TrackingCode: shipment.TrackingCode, CarrierOrderId: shipment.CarrierOrderID,
		ShipmentId: shipment.ID.String(), Status: statusToV2Proto(shipment.Status),
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
	return response
}

func statusToV2Proto(value domain.ShipmentStatus) shippingv2.ShipmentStatus {
	switch value {
	case domain.StatusCreating:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_CREATING
	case domain.StatusCreationFailed:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_CREATION_FAILED
	case domain.StatusCreated:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_CREATED
	case domain.StatusReadyForHandover:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_READY_FOR_HANDOVER
	case domain.StatusHandedOver:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_HANDED_OVER
	case domain.StatusInTransit:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_IN_TRANSIT
	case domain.StatusOutForDelivery:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_OUT_FOR_DELIVERY
	case domain.StatusDelivered:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_DELIVERED
	case domain.StatusDeliveryFailed:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_DELIVERY_FAILED
	case domain.StatusReturning:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_RETURNING
	case domain.StatusReturned:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_RETURNED
	case domain.StatusCancelled:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_CANCELLED
	default:
		return shippingv2.ShipmentStatus_SHIPMENT_STATUS_UNSPECIFIED
	}
}
