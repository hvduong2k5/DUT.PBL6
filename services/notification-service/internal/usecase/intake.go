package usecase

import (
	"context"
	"github.com/omamx/notification-service/internal/domain"
	"github.com/omamx/notification-service/internal/ports"
)

type Service struct{ Store ports.Store }

func (s Service) Accept(ctx context.Context, p ports.Principal, key string, c domain.Command) (domain.Result, error) {
	if p.Producer == "" {
		return domain.Result{}, domain.ErrUnauthenticated
	}
	if p.Producer != c.SourceEvent.Source || !p.Purposes[c.Purpose] {
		return domain.Result{}, domain.ErrForbidden
	}
	if !domain.ValidKey(key) {
		return domain.Result{}, domain.ErrInvalid
	}
	if err := c.Validate(); err != nil {
		return domain.Result{}, err
	}
	return s.Store.Accept(ctx, p.Producer, key, c)
}
func (s Service) Get(ctx context.Context, p ports.Principal, id string) (domain.Request, error) {
	if p.Producer == "" {
		return domain.Request{}, domain.ErrUnauthenticated
	}
	if !p.Purposes["ORDER_CONFIRMATION"] {
		return domain.Request{}, domain.ErrForbidden
	}
	return s.Store.Get(ctx, p.Producer, id)
}
