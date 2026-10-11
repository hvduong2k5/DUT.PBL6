package domain

import (
	"context"
	"github.com/google/uuid"
)

// Mutate must lock, check expectedVersion, apply the callback and persist atomically.
type Repository interface {
	Create(context.Context, *Article) error
	Get(context.Context, uuid.UUID) (*Article, error)
	GetPublished(context.Context, string) (*Article, error)
	List(context.Context, Filter) ([]Article, int64, error)
	Mutate(context.Context, uuid.UUID, int64, func(*Article) error) (*Article, error)
}
