package catalogprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	catalog "github.com/omamx/order-service/internal/gen/catalog/v1"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type SnapshotItem struct {
	SKU          string
	Quantity     int
	Name         string
	Price        int64
	Weight       int64
	Packaging    string
	Active       bool
	PriceVersion int64
}

type Snapshot struct {
	ID          string
	Version     int64
	Until       time.Time
	EvaluatedAt time.Time
	Items       []SnapshotItem
	Subtotal    int64
	Contract    string
	Channel     string
}

type circuitBreaker struct {
	mu          sync.Mutex
	windowStart time.Time
	total       int
	failures    int
	openUntil   time.Time
}

func (cb *circuitBreaker) allow(now time.Time) bool {
	if cb == nil {
		return true
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.openUntil.IsZero() {
		if now.Before(cb.openUntil) {
			return false
		}
		cb.openUntil = time.Time{}
		cb.windowStart = now
		cb.total = 0
		cb.failures = 0
	}
	if cb.windowStart.IsZero() || now.Sub(cb.windowStart) > 10*time.Second {
		cb.windowStart = now
		cb.total = 0
		cb.failures = 0
	}
	return true
}

func (cb *circuitBreaker) record(now time.Time, failed bool) {
	if cb == nil {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.windowStart.IsZero() || now.Sub(cb.windowStart) > 10*time.Second {
		cb.windowStart = now
		cb.total = 0
		cb.failures = 0
	}
	cb.total++
	if failed {
		cb.failures++
	}
	if cb.total >= 5 && cb.failures*2 > cb.total {
		cb.openUntil = now.Add(10 * time.Second)
	}
}

type Adapter struct {
	Client  catalog.CatalogServiceClient
	Token   string
	breaker *circuitBreaker
}

// Dial connects to the Catalog gRPC endpoint using either pinned TLS or explicit loopback test credentials.
func Dial(ctx context.Context, target, token, ca, serverName string, local bool) (*Adapter, *grpc.ClientConn, error) {
	if len(token) < 32 {
		return nil, nil, errors.New("Catalog token must contain at least 32 bytes")
	}
	var cred credentials.TransportCredentials
	if local {
		host, _, err := net.SplitHostPort(target)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) || ca != "" || serverName != "" {
			return nil, nil, errors.New("insecure Catalog allowed only on explicit loopback test endpoint")
		}
		cred = insecure.NewCredentials()
	} else {
		if ca == "" || serverName == "" {
			return nil, nil, errors.New("Catalog requires pinned CA and server name")
		}
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, nil, errors.New("invalid Catalog CA")
		}
		cred = credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: serverName, MinVersion: tls.VersionTLS12})
	}
	deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(deadline, target, grpc.WithTransportCredentials(cred), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}
	return &Adapter{Client: catalog.NewCatalogServiceClient(conn), Token: token, breaker: &circuitBreaker{}}, conn, nil
}

// ValidateCheckoutPricing validates items and prices via Catalog gRPC and builds a checkout pricing snapshot.
func (a *Adapter) ValidateCheckoutPricing(ctx context.Context, items []domain.Item, sendClientPrice bool) (Snapshot, error) {
	var snap Snapshot
	if a == nil || a.Client == nil || len(items) == 0 {
		return snap, domain.E("CATALOG_UNAVAILABLE", 503)
	}
	if a.breaker == nil {
		a.breaker = &circuitBreaker{}
	}
	if !a.breaker.allow(time.Now()) {
		return snap, domain.E("CATALOG_SERVICE_UNAVAILABLE", 503)
	}
	reqItems := make([]*catalog.CheckoutItemInput, 0, len(items))
	for _, item := range items {
		in := &catalog.CheckoutItemInput{
			SkuCode:  item.SKU,
			Quantity: int32(item.Quantity),
		}
		if sendClientPrice && item.UnitPrice > 0 {
			in.ClientUnitPrice = &common.Money{
				CurrencyCode: "VND",
				Units:        item.UnitPrice,
				Nanos:        0,
			}
		}
		reqItems = append(reqItems, in)
	}
	req := &catalog.ValidateCheckoutPricingRequest{
		Channel: catalog.SalesChannel_SALES_CHANNEL_D2C_WEB,
		Items:   reqItems,
	}
	var resp *catalog.ValidateCheckoutPricingResponse
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		rpcCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
		if a.Token != "" {
			rpcCtx = metadata.AppendToOutgoingContext(rpcCtx, "x-internal-token", a.Token)
		}
		resp, err = a.Client.ValidateCheckoutPricing(rpcCtx, req)
		cancel()
		if err == nil {
			break
		}
		code := status.Code(err)
		if (code != codes.Unavailable && code != codes.DeadlineExceeded) || attempt == 2 {
			break
		}
		select {
		case <-ctx.Done():
			a.breaker.record(time.Now(), true)
			return snap, domain.E("CATALOG_SERVICE_UNAVAILABLE", 503)
		case <-time.After(50 * time.Millisecond):
		}
	}
	if err != nil {
		if status.Code(err) == codes.InvalidArgument {
			a.breaker.record(time.Now(), false)
			return snap, domain.E("INVALID_CART_INPUT", 400)
		}
		a.breaker.record(time.Now(), true)
		return snap, domain.E("CATALOG_SERVICE_UNAVAILABLE", 503)
	}
	a.breaker.record(time.Now(), false)
	if !resp.GetIsValid() {
		for _, d := range resp.GetDiscrepancies() {
			if d.GetDiscrepancyReason() == catalog.ValidationIssue_VALIDATION_ISSUE_PRICE_CHANGED ||
				d.GetDiscrepancyReason() == catalog.ValidationIssue_VALIDATION_ISSUE_CURRENCY_MISMATCH {
				return snap, domain.E("PRICE_CHANGED", 409)
			}
		}
		return snap, domain.E("CATALOG_ITEM_UNAVAILABLE", 422)
	}
	if len(resp.GetItems()) != len(items) {
		return snap, domain.E("CATALOG_INCONSISTENT", 503)
	}
	var total int64
	var maxVersion int64 = 1
	outItems := make([]SnapshotItem, 0, len(resp.GetItems()))
	for idx, v := range resp.GetItems() {
		if v.GetSkuCode() != items[idx].SKU || int(v.GetQuantity()) != items[idx].Quantity || !v.GetIsSellable() {
			return snap, domain.E("CATALOG_INCONSISTENT", 503)
		}
		price := v.GetAuthoritativeUnitPrice()
		if price == nil || price.GetCurrencyCode() != "VND" || price.GetNanos() != 0 || price.GetUnits() < 1 || price.GetUnits() > domain.MaxMoney/int64(items[idx].Quantity) {
			return snap, domain.E("CATALOG_INCONSISTENT", 503)
		}
		weightGrams, err := weightInGrams(v.GetNetWeight())
		if err != nil || weightGrams < 1 {
			return snap, domain.E("CATALOG_INCONSISTENT", 503)
		}
		name := strings.TrimSpace(v.GetProductName())
		if name == "" {
			name = strings.TrimSpace(v.GetVariantName())
		}
		if name == "" {
			return snap, domain.E("CATALOG_INCONSISTENT", 503)
		}
		line := price.GetUnits() * int64(items[idx].Quantity)
		if total > domain.MaxMoney-line {
			return snap, domain.E("INVALID_TOTAL", 422)
		}
		total += line
		if v.GetPriceVersion() > maxVersion {
			maxVersion = v.GetPriceVersion()
		}
		if v.GetVariantVersion() > maxVersion {
			maxVersion = v.GetVariantVersion()
		}
		outItems = append(outItems, SnapshotItem{
			SKU:          v.GetSkuCode(),
			Quantity:     int(v.GetQuantity()),
			Name:         name,
			Price:        price.GetUnits(),
			Weight:       weightGrams,
			Packaging:    packagingName(v.GetPackagingType()),
			Active:       v.GetIsSellable(),
			PriceVersion: v.GetPriceVersion(),
		})
	}
	sub := resp.GetCanonicalSubtotal()
	if sub == nil || sub.GetCurrencyCode() != "VND" || sub.GetNanos() != 0 || sub.GetUnits() != total {
		return snap, domain.E("CATALOG_INCONSISTENT", 503)
	}
	evaluatedAt := time.Now().UTC()
	if resp.GetEvaluatedAt() != nil && resp.GetEvaluatedAt().CheckValid() == nil {
		if ts := resp.GetEvaluatedAt().AsTime(); ts.After(evaluatedAt) {
			evaluatedAt = ts
		}
	}
	snap = Snapshot{
		ID:          uuid.NewString(),
		Version:     maxVersion,
		Until:       evaluatedAt.Add(15 * time.Minute),
		EvaluatedAt: evaluatedAt,
		Items:       outItems,
		Subtotal:    total,
		Contract:    "catalog-checkout-1.0",
		Channel:     "D2C_WEB",
	}
	return snap, nil
}

func weightInGrams(w *common.Weight) (int64, error) {
	if w == nil || strings.TrimSpace(w.GetDecimalValue()) == "" {
		return 0, errors.New("missing weight")
	}
	r, ok := new(big.Rat).SetString(strings.TrimSpace(w.GetDecimalValue()))
	if !ok {
		return 0, errors.New("invalid weight decimal")
	}
	if w.GetUnit() == common.WeightUnit_WEIGHT_UNIT_KILOGRAM {
		r.Mul(r, big.NewRat(1000, 1))
	} else if w.GetUnit() != common.WeightUnit_WEIGHT_UNIT_GRAM {
		return 0, errors.New("unsupported weight unit")
	}
	f, _ := r.Float64()
	grams := int64(f + 0.5)
	if grams < 1 {
		return 0, errors.New("non-positive weight")
	}
	return grams, nil
}

func packagingName(p catalog.PackagingType) string {
	switch p {
	case catalog.PackagingType_PACKAGING_TYPE_HOP_GIAY_KRAFT:
		return "HOP_GIAY_KRAFT"
	case catalog.PackagingType_PACKAGING_TYPE_HOP_THIEC:
		return "HOP_THIEC"
	case catalog.PackagingType_PACKAGING_TYPE_GOI_HUT_CHAN_KHONG:
		return "GOI_HUT_CHAN_KHONG"
	case catalog.PackagingType_PACKAGING_TYPE_HOP_QUA_TANG:
		return "HOP_QUA_TANG"
	case catalog.PackagingType_PACKAGING_TYPE_DANG_ROI:
		return "DANG_ROI"
	default:
		return "GOI_HUT_CHAN_KHONG"
	}
}
