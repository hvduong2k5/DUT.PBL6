package dependency

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	catalogv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/catalog/v1"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s Live) Batch(ctx context.Context, namespace, batchID, sku string) (domain.DependencySnapshot, error) {
	if namespace != "inventory" || !domain.ValidUUID(batchID) || batchID != strings.ToLower(batchID) || !validSKU(sku) {
		return domain.DependencySnapshot{}, domain.ErrInvalidCommand
	}
	if s.Inventory == nil {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	checked := time.Now().UTC()
	batch, err := s.Inventory.GetTraceabilityBatch(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+s.InventoryToken), &traceabilityv1.BatchRequest{BatchId: batchID, SkuCode: sku})
	if status.Code(err) == codes.NotFound {
		return domain.DependencySnapshot{}, domain.ErrNotFound
	}
	if err != nil || batch == nil || batch.GetBatchVersion() < 1 {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	if batch.GetBatchId() != batchID || batch.GetSkuCode() != sku {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	if !domain.ValidUUID(batch.GetWarehouseId()) || batch.GetBatchCode() == "" {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	location := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	produced, err := time.ParseInLocation("2006-01-02", batch.GetProductionDate(), location)
	if err != nil {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	expires, err := time.ParseInLocation("2006-01-02", batch.GetExpiryDate(), location)
	if err != nil || expires.Before(produced) {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	return domain.DependencySnapshot{Binding: domain.Binding{SourceNamespace: namespace, BatchID: batchID, SKUCode: sku}, WarehouseID: strings.ToLower(batch.GetWarehouseId()), BatchCode: batch.GetBatchCode(), ProductionDate: batch.GetProductionDate(), ExpiryDate: batch.GetExpiryDate(), BestBefore: expires.AddDate(0, 0, 1).UTC(), Restricted: batch.GetRestricted(), Depleted: batch.GetDepleted(), BatchVersion: strconv.FormatInt(batch.GetBatchVersion(), 10), CheckedAt: checked}, nil
}

func validSKU(sku string) bool {
	return sku != "" && len([]rune(sku)) <= 100 && strings.TrimSpace(sku) == sku && !strings.ContainsAny(sku, "/\\\x00") && sku != "." && sku != ".."
}

func (s Live) Catalog(ctx context.Context, sku string) (string, string, error) {
	if !validSKU(sku) {
		return "", "", domain.ErrInvalidCommand
	}
	if s.CatalogSource == nil {
		return "", "", domain.ErrUnavailable
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-token", s.CatalogToken)
	response, err := s.CatalogSource.GetProductVariant(ctx, &catalogv1.GetProductVariantRequest{SkuCode: sku}, grpc.MaxCallRecvMsgSize(8192))
	if err != nil {
		return catalogError(err)
	}
	if response == nil || response.GetVariant() == nil || !domain.ValidUUID(response.GetVariant().GetProductId()) {
		return "", "", domain.ErrUnavailable
	}
	if response.GetVariant().GetSkuCode() != sku {
		return "", "", domain.ErrConflict
	}
	return s.catalogBinding(ctx, response.GetVariant().GetProductId(), sku)
}

func catalogError(err error) (string, string, error) {
	switch status.Code(err) {
	case codes.NotFound:
		return "", "", domain.ErrNotFound
	case codes.InvalidArgument:
		return "", "", domain.ErrInvalidCommand
	default:
		return "", "", domain.ErrUnavailable
	}
}

func (s Live) catalogBinding(ctx context.Context, product, sku string) (string, string, error) {
	if s.CatalogSource == nil {
		return "", "", domain.ErrUnavailable
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set("x-internal-token", s.CatalogToken)
	ctx = metadata.NewOutgoingContext(ctx, md)
	response, err := s.CatalogSource.GetTraceabilityBinding(ctx, &catalogv1.GetTraceabilityBindingRequest{ProductId: product, SkuCode: sku}, grpc.MaxCallRecvMsgSize(8192))
	if err != nil {
		return catalogError(err)
	}
	if response == nil || !domain.ValidUUID(response.GetProductId()) || response.GetVersionFingerprint() == "" {
		return "", "", domain.ErrUnavailable
	}
	if response.GetProductId() != product || response.GetSkuCode() != sku {
		return "", "", domain.ErrConflict
	}
	return strings.ToLower(response.GetProductId()), response.GetVersionFingerprint(), nil
}

func (f *Fixture) Batch(ctx context.Context, namespace, batchID, sku string) (domain.DependencySnapshot, error) {
	if err := ctx.Err(); err != nil {
		return domain.DependencySnapshot{}, err
	}
	if namespace != "fixture" {
		return domain.DependencySnapshot{}, domain.ErrInvalidCommand
	}
	result, ok := f.batches[batchID]
	if !ok {
		return domain.DependencySnapshot{}, domain.ErrNotFound
	}
	if result.SKUCode != sku {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	if !domain.ValidUUID(result.WarehouseID) {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	result.CheckedAt = f.clock().UTC()
	return result, nil
}
func (f *Fixture) Catalog(ctx context.Context, sku string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	id, version := "", ""
	for _, s := range f.batches {
		if s.SKUCode == sku {
			if id != "" && (id != s.ProductID || version != s.CatalogVersion) {
				return "", "", domain.ErrUnavailable
			}
			id, version = s.ProductID, s.CatalogVersion
		}
	}
	if id == "" {
		return "", "", domain.ErrNotFound
	}
	return id, version, nil
}
