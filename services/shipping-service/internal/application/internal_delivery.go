package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/domain"
)

var (
	ErrEmployeeUnauthenticated      = errors.New("employee credential rejected by Identity")
	ErrEmployeeForbidden            = errors.New("active employee authorization required")
	ErrEmployeeDirectoryUnavailable = errors.New("employee directory unavailable")
	ErrDriverIneligible             = errors.New("driver must be an active employee")
)

type AssignmentDirectory interface {
	ValidateActor(context.Context, string, uuid.UUID, string) error
	ValidateDriver(context.Context, string, uuid.UUID) error
}

type InternalDeliveryRepository interface {
	AssignInternalDelivery(context.Context, domain.AssignInternalDelivery) (*domain.InternalDeliveryAssignment, error)
	GetInternalDeliveryAssignment(context.Context, uuid.UUID) (*domain.InternalDeliveryAssignment, error)
	InternalDeliveryAssignmentHistory(context.Context, uuid.UUID) ([]domain.InternalDeliveryAssignmentEvent, error)
}

// The calling transport must enforce employee authorization and live directory
// validation before invoking this application primitive.
func (s *Service) AssignInternalDelivery(ctx context.Context, command domain.AssignInternalDelivery) (*domain.InternalDeliveryAssignment, error) {
	if err := command.Validate(); err != nil {
		return nil, err
	}
	repo, ok := s.repository.(InternalDeliveryRepository)
	if !ok {
		return nil, fmt.Errorf("internal delivery assignment repository is not configured")
	}
	return repo.AssignInternalDelivery(ctx, command)
}

func (s *Service) GetInternalDeliveryAssignment(ctx context.Context, shipmentID uuid.UUID) (*domain.InternalDeliveryAssignment, error) {
	if shipmentID == uuid.Nil {
		return nil, fmt.Errorf("%w: shipment UUID is required", domain.ErrInvalidShipment)
	}
	repo, ok := s.repository.(InternalDeliveryRepository)
	if !ok {
		return nil, fmt.Errorf("internal delivery assignment repository is not configured")
	}
	return repo.GetInternalDeliveryAssignment(ctx, shipmentID)
}

func (s *Service) InternalDeliveryAssignmentHistory(ctx context.Context, shipmentID uuid.UUID) ([]domain.InternalDeliveryAssignmentEvent, error) {
	if shipmentID == uuid.Nil {
		return nil, fmt.Errorf("%w: shipment UUID is required", domain.ErrInvalidShipment)
	}
	repo, ok := s.repository.(InternalDeliveryRepository)
	if !ok {
		return nil, fmt.Errorf("internal delivery assignment repository is not configured")
	}
	return repo.InternalDeliveryAssignmentHistory(ctx, shipmentID)
}
