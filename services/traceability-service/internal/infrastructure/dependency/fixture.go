// Package dependency contains explicit synthetic source adapters, never a fallback.
package dependency

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type Fixture struct {
	batches map[string]domain.DependencySnapshot
	clock   func() time.Time
}

func NewFixture(mode string, snapshots []domain.DependencySnapshot, clock func() time.Time) (*Fixture, error) {
	if mode != "sandbox" || clock == nil || len(snapshots) == 0 {
		return nil, errors.New("fixture requires explicit sandbox mode and synthetic sources")
	}
	f := &Fixture{batches: map[string]domain.DependencySnapshot{}, clock: clock}
	for _, s := range snapshots {
		if s.SourceNamespace != "fixture" || s.Binding.Validate() != nil || s.BatchCode == "" || s.CatalogVersion == "" || s.BatchVersion == "" {
			return nil, errors.New("invalid synthetic source")
		}
		key := s.BatchID
		if _, exists := f.batches[key]; exists {
			return nil, errors.New("duplicate synthetic batch identity")
		}
		s.ProductID = strings.ToLower(s.ProductID)
		f.batches[key] = s
	}
	return f, nil
}

func (f *Fixture) Snapshot(ctx context.Context, binding domain.Binding) (domain.DependencySnapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.DependencySnapshot{}, err
	}
	if f == nil || binding.SourceNamespace != "fixture" {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	snapshot, exists := f.batches[binding.BatchID]
	if !exists {
		return domain.DependencySnapshot{}, domain.ErrNotFound
	}
	binding.ProductID = strings.ToLower(binding.ProductID)
	if snapshot.Binding != binding {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	snapshot.CheckedAt = f.clock().UTC()
	return snapshot, nil
}
