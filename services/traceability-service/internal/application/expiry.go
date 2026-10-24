package application

import (
	"context"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type ExpiryStore interface {
	SweepExpiry(context.Context, string, int) (string, int, error)
}

// ExpiryWorker has one owner goroutine. Concurrent processes are serialized by
// profile row locks and the unique observation key in PostgreSQL.
type ExpiryWorker struct {
	Store    ExpiryStore
	Cursor   string
	PageSize int
}

func (w *ExpiryWorker) Tick(ctx context.Context) (int, error) {
	if w == nil || w.Store == nil || w.PageSize < 1 || w.PageSize > 100 {
		return 0, domain.ErrInvalidCommand
	}
	next, count, err := w.Store.SweepExpiry(ctx, w.Cursor, w.PageSize)
	if err == nil {
		w.Cursor = next
	}
	return count, err
}
