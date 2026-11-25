package application

import (
	"context"
	"fmt"
	"github.com/omamx/shipping-service/internal/domain"
)

type InternalDeliveryProgressRepository interface {
	ProgressInternalDelivery(context.Context, domain.ProgressInternalDelivery) (*domain.InternalDeliveryProgressReceipt, error)
}

// Transport must authenticate the employee and validate live Identity state.
func (s *Service) ProgressInternalDelivery(ctx context.Context, c domain.ProgressInternalDelivery) (*domain.InternalDeliveryProgressReceipt, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	repo, ok := s.repository.(InternalDeliveryProgressRepository)
	if !ok {
		return nil, fmt.Errorf("internal progress repository is not configured")
	}
	return repo.ProgressInternalDelivery(ctx, c)
}
