package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

type ListFilter struct {
	Status      domain.ShipmentStatus
	CarrierCode string
	OrderID     string
	Limit       int
	Offset      int
}

type Repository interface {
	Ping(context.Context) error
	BeginCreation(context.Context, string, *domain.Shipment) (*domain.Shipment, bool, error)
	Save(context.Context, *domain.Shipment, int64, *domain.TrackingEvent, *domain.DomainEvent) error
	GetByID(context.Context, uuid.UUID) (*domain.Shipment, error)
	GetByTracking(context.Context, string, string) (*domain.Shipment, error)
	ListByOrder(context.Context, string) ([]domain.Shipment, error)
	List(context.Context, ListFilter) ([]domain.Shipment, error)
	TrackingTimeline(context.Context, uuid.UUID) ([]domain.TrackingEvent, error)
}

type ProviderError struct {
	Shipment *domain.Shipment
	Cause    error
}

type PackingCompleted struct {
	EventID   string
	OrderID   string
	PackageID string
	SealCode  string
	Parcel    domain.Parcel
}

type OrderShippingSnapshot struct {
	OrderID             string
	PackageID           string
	Recipient           domain.Recipient
	CheckoutFeeVND      int64
	CODAmountVND        int64
	CarrierCode         string
	ServiceCode         string
	EligibleForShipping bool
}

type ShippingSnapshotProvider interface {
	GetShippingSnapshot(context.Context, string, string) (OrderShippingSnapshot, error)
}

func (e *ProviderError) Error() string { return "carrier operation failed: " + e.Cause.Error() }
func (e *ProviderError) Unwrap() error { return e.Cause }

type Service struct {
	repository Repository
	carriers   *carrier.Registry
	now        func() time.Time
}

func NewService(repository Repository, carriers *carrier.Registry) *Service {
	return &Service{repository: repository, carriers: carriers, now: time.Now}
}

func (s *Service) Ping(ctx context.Context) error { return s.repository.Ping(ctx) }

func (s *Service) Quote(ctx context.Context, carrierCode string, request carrier.QuoteRequest) (carrier.Quote, error) {
	provider, ok := s.carriers.Get(carrierCode)
	if !ok {
		return carrier.Quote{}, fmt.Errorf("%w: %s", domain.ErrUnsupportedCarrier, carrierCode)
	}
	return provider.Quote(ctx, request)
}

func (s *Service) CreateShipment(ctx context.Context, cmd domain.CreateShipmentCommand) (*domain.Shipment, error) {
	shipment, err := domain.NewCreatingShipment(cmd, s.now())
	if err != nil {
		return nil, err
	}
	shipment, inserted, err := s.repository.BeginCreation(ctx, cmd.IdempotencyKey, shipment)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return shipment, nil
	}

	provider, ok := s.carriers.Get(shipment.CarrierCode)
	if !ok {
		return s.failCreation(ctx, shipment, "UNSUPPORTED_CARRIER", domain.ErrUnsupportedCarrier)
	}
	result, providerErr := provider.CreateShipment(ctx, carrier.CreateRequest{
		IdempotencyKey: cmd.IdempotencyKey, ShipmentID: shipment.ID.String(), OrderID: shipment.OrderID,
		PackageID: shipment.PackageID, SealCode: shipment.SealCode, ServiceCode: shipment.ServiceCode,
		Recipient: shipment.Recipient, Parcel: shipment.Parcel, CODAmountVND: shipment.CODAmountVND,
		ShippingNotes: cmd.ShippingNotes,
	})
	if providerErr != nil {
		return s.failCreation(ctx, shipment, "CARRIER_REJECTED", providerErr)
	}
	expectedRevision := shipment.Revision
	now := s.now()
	if err := shipment.MarkCreated(result.CarrierOrderID, result.TrackingCode, result.LabelReference, result.EstimatedFeeVND, result.EstimatedPickupAt, result.EstimatedDeliveryAt, now); err != nil {
		return nil, err
	}
	event := domain.NewDomainEvent("vn.omama.shipping.shipment.created.v1", shipment, now)
	if err := s.repository.Save(ctx, shipment, expectedRevision, nil, &event); err != nil {
		return nil, err
	}
	return shipment, nil
}

func (s *Service) CreateFromPacking(ctx context.Context, event PackingCompleted, snapshots ShippingSnapshotProvider, defaultCarrier, defaultService string) (*domain.Shipment, error) {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.OrderID) == "" ||
		strings.TrimSpace(event.PackageID) == "" || strings.TrimSpace(event.SealCode) == "" {
		return nil, fmt.Errorf("%w: packing event id, order id, package id and seal code are required", domain.ErrInvalidShipment)
	}
	if err := event.Parcel.Validate(); err != nil {
		return nil, err
	}
	snapshot, err := snapshots.GetShippingSnapshot(ctx, event.OrderID, event.PackageID)
	if err != nil {
		return nil, fmt.Errorf("get immutable order shipping snapshot: %w", err)
	}
	if !snapshot.EligibleForShipping {
		return nil, fmt.Errorf("%w: order is not eligible for shipping", domain.ErrInvalidShipment)
	}
	if snapshot.OrderID != event.OrderID || snapshot.PackageID != event.PackageID {
		return nil, fmt.Errorf("%w: order shipping snapshot identity mismatch", domain.ErrInvalidShipment)
	}
	carrierCode := strings.TrimSpace(snapshot.CarrierCode)
	if carrierCode == "" {
		carrierCode = defaultCarrier
	}
	serviceCode := strings.TrimSpace(snapshot.ServiceCode)
	if serviceCode == "" {
		serviceCode = defaultService
	}
	return s.CreateShipment(ctx, domain.CreateShipmentCommand{
		IdempotencyKey: "packing-package:" + event.PackageID,
		OrderID:        event.OrderID, PackageID: event.PackageID, SealCode: event.SealCode,
		CarrierCode: carrierCode, ServiceCode: serviceCode, Recipient: snapshot.Recipient,
		Parcel: event.Parcel, CODAmountVND: snapshot.CODAmountVND,
		CheckoutFeeVND: snapshot.CheckoutFeeVND,
	})
}

func (s *Service) failCreation(ctx context.Context, shipment *domain.Shipment, code string, cause error) (*domain.Shipment, error) {
	expectedRevision := shipment.Revision
	now := s.now()
	if transitionErr := shipment.MarkCreationFailed(code, cause.Error(), now); transitionErr != nil {
		return nil, errors.Join(cause, transitionErr)
	}
	event := domain.NewDomainEvent("vn.omama.shipping.shipment.creation-failed.v1", shipment, now)
	if saveErr := s.repository.Save(ctx, shipment, expectedRevision, nil, &event); saveErr != nil {
		return nil, errors.Join(cause, saveErr)
	}
	return shipment, &ProviderError{Shipment: shipment, Cause: cause}
}

func (s *Service) ConfirmHandover(ctx context.Context, id uuid.UUID) (*domain.Shipment, error) {
	shipment, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	expectedRevision := shipment.Revision
	now := s.now()
	if err := shipment.Transition(domain.StatusHandedOver, now); err != nil {
		return nil, err
	}
	event := domain.NewDomainEvent("vn.omama.shipping.shipment.handed-over.v1", shipment, now)
	if err := s.repository.Save(ctx, shipment, expectedRevision, nil, &event); err != nil {
		return nil, err
	}
	return shipment, nil
}

func (s *Service) ApplyWebhook(ctx context.Context, carrierCode string, body []byte) (*domain.Shipment, string, error) {
	provider, ok := s.carriers.Get(carrierCode)
	if !ok {
		return nil, "", fmt.Errorf("%w: %s", domain.ErrUnsupportedCarrier, carrierCode)
	}
	update, err := provider.ParseWebhook(body)
	if err != nil {
		return nil, "", err
	}
	shipment, err := s.repository.GetByTracking(ctx, provider.Code(), update.TrackingCode)
	if err != nil {
		return nil, "", err
	}

	now := s.now().UTC()
	tracking := &domain.TrackingEvent{
		ID: uuid.New(), ShipmentID: shipment.ID, ExternalEventID: update.ExternalEventID,
		CarrierCode: provider.Code(), ProviderStatusCode: update.ProviderStatusCode,
		ProviderStatusLabel: update.ProviderStatusLabel, MappedStatus: update.MappedStatus,
		Location: update.Location, PublicDescription: update.PublicDescription,
		InternalNote: update.InternalNote, EventAt: update.EventAt.UTC(), ReceivedAt: now,
		MappingResult: "APPLIED",
	}
	expectedRevision := shipment.Revision
	if shipment.LastEventAt != nil && update.EventAt.Before(*shipment.LastEventAt) {
		tracking.MappingResult = "STALE"
		if err := s.repository.Save(ctx, shipment, expectedRevision, tracking, nil); err != nil {
			return nil, "", err
		}
		return shipment, tracking.MappingResult, nil
	}
	if err := shipment.Transition(update.MappedStatus, now); err != nil {
		tracking.MappingResult = "REJECTED"
		tracking.InternalNote = strings.TrimSpace(tracking.InternalNote + " " + err.Error())
		if saveErr := s.repository.Save(ctx, shipment, expectedRevision, tracking, nil); saveErr != nil {
			return nil, "", saveErr
		}
		return shipment, tracking.MappingResult, nil
	}
	eventAt := update.EventAt.UTC()
	shipment.LastEventAt = &eventAt
	eventType := eventTypeForStatus(shipment.Status)
	var event *domain.DomainEvent
	if eventType != "" {
		value := domain.NewDomainEvent(eventType, shipment, now)
		event = &value
	}
	if err := s.repository.Save(ctx, shipment, expectedRevision, tracking, event); err != nil {
		if errors.Is(err, domain.ErrDuplicateEvent) {
			return shipment, "DUPLICATE", nil
		}
		return nil, "", err
	}
	return shipment, tracking.MappingResult, nil
}

func eventTypeForStatus(status domain.ShipmentStatus) string {
	switch status {
	case domain.StatusHandedOver:
		return "vn.omama.shipping.shipment.handed-over.v1"
	case domain.StatusDelivered:
		return "vn.omama.shipping.shipment.delivered.v1"
	case domain.StatusDeliveryFailed:
		return "vn.omama.shipping.shipment.delivery-failed.v1"
	case domain.StatusReturning:
		return "vn.omama.shipping.shipment.returning.v1"
	case domain.StatusReturned:
		return "vn.omama.shipping.shipment.returned.v1"
	default:
		return "vn.omama.shipping.shipment.status-changed.v1"
	}
}

func (s *Service) GetShipment(ctx context.Context, id uuid.UUID) (*domain.Shipment, []domain.TrackingEvent, error) {
	shipment, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	timeline, err := s.repository.TrackingTimeline(ctx, id)
	return shipment, timeline, err
}

func (s *Service) GetShipmentTimeline(ctx context.Context, id uuid.UUID) ([]domain.TrackingEvent, error) {
	return s.repository.TrackingTimeline(ctx, id)
}

func (s *Service) TrackOrder(ctx context.Context, orderID string) ([]domain.Shipment, error) {
	return s.repository.ListByOrder(ctx, orderID)
}

func (s *Service) TrackByCode(ctx context.Context, carrierCode, trackingCode string) (*domain.Shipment, []domain.TrackingEvent, error) {
	shipment, err := s.repository.GetByTracking(ctx, strings.ToUpper(carrierCode), trackingCode)
	if err != nil {
		return nil, nil, err
	}
	timeline, err := s.repository.TrackingTimeline(ctx, shipment.ID)
	return shipment, timeline, err
}

func (s *Service) ListShipments(ctx context.Context, filter ListFilter) ([]domain.Shipment, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repository.List(ctx, filter)
}
