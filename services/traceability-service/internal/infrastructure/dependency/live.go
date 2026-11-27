package dependency

import (
	"context"
	"errors"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	catalogv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/catalog/v1"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
)

type Live struct {
	CatalogToken, InventoryToken string
	CatalogSource                catalogv1.CatalogTraceabilitySourceClient
	Inventory                    traceabilityv1.BatchSourceClient
}

func (s Live) Snapshot(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
	if b.Validate() != nil || b.SourceNamespace != "inventory" || !domain.ValidUUID(b.BatchID) || !validSKU(b.SKUCode) {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	checkedAt := time.Now().UTC()
	product, version, err := s.Catalog(ctx, b.SKUCode)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	if err != nil {
		return domain.DependencySnapshot{}, err
	}
	if product != b.ProductID {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	batch, err := s.Batch(ctx, b.SourceNamespace, b.BatchID, b.SKUCode)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	if err != nil {
		return domain.DependencySnapshot{}, err
	}
	batch.Binding = b
	batch.CatalogVersion = version
	batch.CheckedAt = checkedAt
	return batch, nil
}
