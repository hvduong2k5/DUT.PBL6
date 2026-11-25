package application

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/domain"
)

type DriverWorkbenchRepository interface {
	ListDriverShipments(context.Context, uuid.UUID, int, int) (*domain.DriverShipmentList, error)
	GetDriverShipment(context.Context, uuid.UUID, uuid.UUID) (*domain.DriverShipmentDetail, error)
}

func (s *Service) ListDriverShipments(ctx context.Context, actor uuid.UUID, limit, offset int) (*domain.DriverShipmentList, error) {
	if actor == uuid.Nil || limit < 1 || limit > 100 || offset < 0 || offset > 10000 {
		return nil, fmt.Errorf("%w: invalid driver pagination", domain.ErrInvalidShipment)
	}
	r, ok := s.repository.(DriverWorkbenchRepository)
	if !ok {
		return nil, fmt.Errorf("driver workbench repository is not configured")
	}
	return r.ListDriverShipments(ctx, actor, limit, offset)
}

func (s *Service) GetDriverShipment(ctx context.Context, actor, id uuid.UUID) (*domain.DriverShipmentDetail, error) {
	if actor == uuid.Nil || id == uuid.Nil {
		return nil, fmt.Errorf("%w: driver and shipment UUIDs required", domain.ErrInvalidShipment)
	}
	r, ok := s.repository.(DriverWorkbenchRepository)
	if !ok {
		return nil, fmt.Errorf("driver workbench repository is not configured")
	}
	return r.GetDriverShipment(ctx, actor, id)
}
