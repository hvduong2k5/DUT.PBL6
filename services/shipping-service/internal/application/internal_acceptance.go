package application

import (
	"context"
	"fmt"
	"github.com/omamx/shipping-service/internal/domain"
)

type InternalDeliveryAcceptanceRepository interface {
	AcceptInternalDelivery(context.Context, domain.AcceptInternalDelivery) (*domain.InternalDeliveryAcceptance, error)
}

// Trusted transport only: authenticate and validate live employee state before calling.
// The opt-in driver acceptance transport enforces JWT permission and live employee state.
func (s *Service) AcceptInternalDelivery(ctx context.Context, c domain.AcceptInternalDelivery) (*domain.InternalDeliveryAcceptance, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	repo, ok := s.repository.(InternalDeliveryAcceptanceRepository)
	if !ok {
		return nil, fmt.Errorf("internal delivery acceptance repository is not configured")
	}
	return repo.AcceptInternalDelivery(ctx, c)
}
