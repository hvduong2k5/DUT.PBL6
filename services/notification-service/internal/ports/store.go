package ports

import (
	"context"
	"github.com/omamx/notification-service/internal/domain"
)

type Principal struct {
	Producer string
	Purposes map[string]bool
}
type Store interface {
	Accept(context.Context, string, string, domain.Command) (domain.Result, error)
	Get(context.Context, string, string) (domain.Request, error)
	Ping(context.Context) error
}
type Authenticator interface {
	Authenticate(context.Context, string) (Principal, error)
}
