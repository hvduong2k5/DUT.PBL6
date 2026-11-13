package dependency

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Live struct {
	CatalogURL, CatalogToken, InventoryToken string
	HTTP                                     *http.Client
	Inventory                                traceabilityv1.BatchSourceClient
}

func (s Live) Snapshot(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
	if b.Validate() != nil || b.SourceNamespace != "inventory" || !domain.ValidUUID(b.BatchID) {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	checkedAt := time.Now().UTC()
	path := "/api/v1/internal/traceability/products/" + url.PathEscape(b.ProductID) + "/skus/" + url.PathEscape(b.SKUCode)
	if strings.ContainsAny(b.SKUCode, "/\\") || b.SKUCode == "." || b.SKUCode == ".." {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.CatalogURL+path, nil)
	if err != nil {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	req.Header.Set("X-Internal-Token", s.CatalogToken)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	var catalog struct {
		ProductID string `json:"product_id"`
		SKU       string `json:"sku_code"`
		Version   string `json:"catalog_version"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&catalog) != nil || catalog.Version == "" {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	if catalog.ProductID != b.ProductID || catalog.SKU != b.SKUCode {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	batch, err := s.Inventory.GetTraceabilityBatch(
		metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+s.InventoryToken),
		&traceabilityv1.BatchRequest{BatchId: b.BatchID, SkuCode: b.SKUCode},
	)
	if status.Code(err) == codes.NotFound {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	if err != nil || batch.GetBatchVersion() < 1 {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	if batch.GetBatchId() != b.BatchID || batch.GetSkuCode() != b.SKUCode {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	expires, err := time.ParseInLocation("2006-01-02", batch.GetExpiryDate(), time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
	if err != nil {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	return domain.DependencySnapshot{
		Binding: b, BatchCode: batch.GetBatchCode(), ProductionDate: batch.GetProductionDate(),
		ExpiryDate: batch.GetExpiryDate(), BestBefore: expires.AddDate(0, 0, 1).UTC(),
		Restricted: batch.GetRestricted(), Depleted: batch.GetDepleted(),
		CatalogVersion: catalog.Version, BatchVersion: strconv.FormatInt(batch.GetBatchVersion(), 10), CheckedAt: checkedAt,
	}, nil
}

func ServiceOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("an explicit internal HTTP(S) service origin is required")
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}
