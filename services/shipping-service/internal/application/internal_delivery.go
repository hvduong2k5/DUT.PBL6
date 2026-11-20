package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/domain"
)

type InternalDeliveryRepository interface {
	AssignInternalDelivery(context.Context, domain.AssignInternalDelivery) (*domain.InternalDeliveryAssignment, error)
	GetInternalDeliveryAssignment(context.Context, uuid.UUID) (*domain.InternalDeliveryAssignment, error)
	InternalDeliveryAssignmentHistory(context.Context, uuid.UUID) ([]domain.InternalDeliveryAssignmentEvent, error)
}

// Not a public endpoint. Employee authorization and directory validation must be
// supplied by the workbench transport before these methods are made reachable.
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
