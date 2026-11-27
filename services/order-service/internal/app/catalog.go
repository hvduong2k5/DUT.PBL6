package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/catalogprovider"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/shippingprovider"
)

type catalogSnapshot struct {
	ID      string    `json:"snapshot_id"`
	Version int64     `json:"catalog_version"`
	Until   time.Time `json:"valid_until"`
	Items   []struct {
		SKU       string `json:"sku_code"`
		Quantity  int    `json:"quantity"`
		Name      string `json:"product_name"`
		Price     int64  `json:"unit_price"`
		Weight    int64  `json:"weight_grams"`
		Packaging string `json:"packaging_type"`
		Active    bool   `json:"is_active"`
	} `json:"items"`
	Subtotal int64  `json:"subtotal"`
	Contract string `json:"contract_version"`
	Channel  string `json:"channel"`
}

func (s *Service) EnableCatalogFromEnv(ctx context.Context) error {
	target := s.C.CatalogGRPC
	if target == "" {
		target = os.Getenv("ORDER_CATALOG_GRPC")
	}
	token := s.C.CatalogToken
	if token == "" {
		token = os.Getenv("ORDER_CATALOG_TOKEN")
	}
	tokenFile := s.C.CatalogTokenFile
	if tokenFile == "" {
		tokenFile = os.Getenv("ORDER_CATALOG_TOKEN_FILE")
	}
	caFile := s.C.CatalogCAFile
	if caFile == "" {
		caFile = os.Getenv("ORDER_CATALOG_CA_FILE")
	}
	serverName := s.C.CatalogServerName
	if serverName == "" {
		serverName = os.Getenv("ORDER_CATALOG_SERVER_NAME")
	}
	allowLocal := s.C.CatalogAllowLocalInsecure || os.Getenv("ORDER_CATALOG_ALLOW_LOCAL_INSECURE") == "true"

	if target == "" {
		if token != "" || tokenFile != "" || caFile != "" || serverName != "" || allowLocal {
			return errors.New("Catalog configuration requires ORDER_CATALOG_GRPC")
		}
		return nil
	}
	if token == "" && tokenFile != "" {
		var err error
		token, err = shippingprovider.TokenFile(tokenFile)
		if err != nil {
			return err
		}
	}
	if token == "" {
		return errors.New("Catalog configuration requires ORDER_CATALOG_TOKEN or ORDER_CATALOG_TOKEN_FILE")
	}
	adapter, conn, err := catalogprovider.Dial(ctx, target, token, caFile, serverName, s.C.Mode == "sandbox" && allowLocal)
	if err != nil {
		return err
	}
	s.Catalog = adapter
	prior := s.PaymentClose
	s.PaymentClose = func() error {
		if prior != nil {
			_ = prior()
		}
		return conn.Close()
	}
	return nil
}

func (s *Service) catalogSnapshot(ctx context.Context, items []domain.Item) (catalogSnapshot, error) {
	if s.Catalog != nil {
		snap, err := s.Catalog.ValidateCheckoutPricing(ctx, items, false)
		if err != nil {
			return catalogSnapshot{}, err
		}
		converted := catalogSnapshot{
			ID:       snap.ID,
			Version:  snap.Version,
			Until:    snap.Until,
			Subtotal: snap.Subtotal,
			Contract: snap.Contract,
			Channel:  snap.Channel,
		}
		for _, item := range snap.Items {
			converted.Items = append(converted.Items, struct {
				SKU       string `json:"sku_code"`
				Quantity  int    `json:"quantity"`
				Name      string `json:"product_name"`
				Price     int64  `json:"unit_price"`
				Weight    int64  `json:"weight_grams"`
				Packaging string `json:"packaging_type"`
				Active    bool   `json:"is_active"`
			}{
				SKU:       item.SKU,
				Quantity:  item.Quantity,
				Name:      item.Name,
				Price:     item.Price,
				Weight:    item.Weight,
				Packaging: item.Packaging,
				Active:    item.Active,
			})
		}
		return converted, nil
	}
	var snapshot catalogSnapshot
	err := s.callURL(ctx, s.C.CatalogURL, "POST", "/api/v1/internal/catalog/snapshots", map[string]any{"items": items, "channel": "D2C_WEB"}, &snapshot)
	if err != nil {
		return snapshot, err
	}
	if snapshot.Contract != "catalog-checkout-1.0" || snapshot.Channel != "D2C_WEB" || snapshot.Version < 1 || !time.Now().Before(snapshot.Until) || len(snapshot.Items) != len(items) {
		return snapshot, domain.E("CATALOG_INCONSISTENT", 503)
	}
	if _, err = uuid.Parse(snapshot.ID); err != nil {
		return snapshot, domain.E("CATALOG_INCONSISTENT", 503)
	}
	var total int64
	for index, item := range snapshot.Items {
		if item.SKU != items[index].SKU || item.Quantity != items[index].Quantity || !item.Active || item.Weight < 1 || item.Name == "" || item.Packaging == "" || item.Quantity < 1 || item.Quantity > 999 || item.Price < 1 || item.Price > domain.MaxMoney/int64(item.Quantity) {
			return snapshot, domain.E("CATALOG_INCONSISTENT", 503)
		}
		line := item.Price * int64(item.Quantity)
		if total > domain.MaxMoney-line {
			return snapshot, domain.E("INVALID_TOTAL", 422)
		}
		total += line
	}
	if total != snapshot.Subtotal {
		return snapshot, domain.E("CATALOG_INCONSISTENT", 503)
	}
	return snapshot, nil
}

func (s *Service) catalogValidation(ctx context.Context, input, output any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	var selection struct {
		Items []domain.Item `json:"items"`
	}
	if err = json.Unmarshal(raw, &selection); err != nil {
		return err
	}
	snapshot, err := s.catalogSnapshot(ctx, selection.Items)
	if err != nil {
		return err
	}
	items := make([]domain.Item, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		items = append(items, domain.Item{SKU: item.SKU, Quantity: item.Quantity, Name: item.Name, UnitPrice: item.Price, Weight: item.Weight, Version: snapshot.Version})
	}
	raw, err = json.Marshal(map[string]any{"items": items, "snapshot_id": snapshot.ID, "expires_at": snapshot.Until})
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, output)
}

// addressNameOrCode prefers the human-readable name captured with the
// address; older flows that only carry codes keep working unchanged.
func addressNameOrCode(name, code string) string {
	if name != "" {
		return name
	}
	return code
}

// Catalog owns its snapshot. Shipping metadata identifies the separately configured
// quote provider; sandbox callers must explicitly label their TEST provider.
func (s *Service) catalogTerms(ctx context.Context, q domain.Quote) (core.SnapshotInput, error) {
	var terms core.SnapshotInput
	policy := s.C.CODPolicyReference
	if policy == "" {
		policy = "cod-collection-1.0"
	}
	if q.Method == "COD" && (strings.TrimSpace(policy) != policy || len(policy) > 255) {
		return terms, domain.E("COD_NOT_AVAILABLE", 422)
	}
	snapshot, err := s.catalogSnapshot(ctx, q.Items)
	if err != nil {
		return terms, err
	}
	version := strconv.FormatInt(snapshot.Version, 10)
	terms.WarehouseID = s.C.CatalogWarehouse
	terms.Catalog = core.CatalogEvidence{SnapshotID: snapshot.ID, Version: version, ValidUntil: snapshot.Until}
	var weight int64
	for _, item := range snapshot.Items {
		terms.Items = append(terms.Items, core.SnapshotItem{LineID: uuid.NewString(), SKU: item.SKU, Name: item.Name, Quantity: int64(item.Quantity), UnitPrice: item.Price, LineSubtotal: item.Price * int64(item.Quantity), WeightGrams: item.Weight, Packaging: core.Packaging{Description: item.Packaging, UnitCode: "UNIT", Version: "catalog-packaging-1"}, CatalogVersion: version})
		weight += item.Weight * int64(item.Quantity)
	}
	terms.Address = core.ShippingAddress{RecipientName: q.Address.Name, PhoneE164: q.Address.Phone, Street: q.Address.Street, ProvinceCode: q.Address.Province, ProvinceName: addressNameOrCode(q.Address.ProvinceName, q.Address.Province), WardCode: q.Address.Ward, WardName: addressNameOrCode(q.Address.WardName, q.Address.Ward), Source: core.AddressSource{Kind: "VALIDATED_INLINE", ValidationReference: uuid.NewString()}}
	if q.AddressID != "" {
		id, v := q.AddressID, q.AddressVersion
		terms.Address.Source.Kind = "PROFILE_ADDRESS"
		terms.Address.Source.ProfileAddressID = &id
		terms.Address.Source.AddressVersion = &v
	}
	quote, err := s.Shipping.Quote(ctx, q.Address, weight, s.C.CatalogShippingCarrier, s.C.CatalogShippingService, s.C.CatalogWarehouse, "shipping:"+q.ID)
	if err != nil {
		return terms, err
	}
	terms.Shipping = core.ShippingTerms{CarrierCode: quote.Carrier, ServiceCode: quote.Service, QuoteReference: quote.ID, QuoteVersion: strconv.FormatInt(quote.Version, 10), ValidUntil: quote.Until, QuotedFee: quote.Fee, PackageWeightGrams: weight}
	terms.Money = core.SnapshotMoney{Currency: "VND", Subtotal: snapshot.Subtotal, ShippingFee: quote.Fee, FinalAmount: snapshot.Subtotal + quote.Fee}
	terms.Policies = core.Policies{Order: "order-core-1.0", Checkout: "cart-checkout-1.0", Payment: "payment-sepay-1.1", Shipping: "shipping-v2"}
	terms.Payment.Method = q.Method
	// Placement records the amount still to collect, never evidence of payment.
	// Completion continues to require the existing Finance settlement authority.
	if q.Method == "COD" {
		amount := terms.Money.FinalAmount
		terms.Payment.CODPolicyReference = &policy
		terms.Payment.CollectAmount = &amount
	}
	return terms, nil
}

func shippingQuoteItems(items []core.SnapshotItem) []domain.Item {
	out := make([]domain.Item, 0, len(items))
	for _, i := range items {
		out = append(out, domain.Item{SKU: i.SKU, Quantity: int(i.Quantity), UnitPrice: i.UnitPrice, Name: i.Name, Weight: i.WeightGrams})
	}
	return out
}

func (s *Service) catalogShippingValidation(ctx context.Context, input, output any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	var selection struct {
		Address domain.Address `json:"address"`
		Items   []domain.Item  `json:"items"`
	}
	if err = json.Unmarshal(raw, &selection); err != nil {
		return err
	}
	var weight int64
	for _, item := range selection.Items {
		if item.Weight < 1 || item.Quantity < 1 || item.Quantity > 999 || item.Weight > 1000000 {
			return domain.E("INVALID_SHIPPING_INPUT", 422)
		}
		weight += item.Weight * int64(item.Quantity)
	}
	quote, err := s.Shipping.Quote(ctx, selection.Address, weight, s.C.CatalogShippingCarrier, s.C.CatalogShippingService, s.C.CatalogWarehouse, "preflight:"+uuid.NewString())
	if err != nil {
		return err
	}
	raw, err = json.Marshal(map[string]any{"quote_reference": quote.ID, "fee": quote.Fee, "expires_at": quote.Until, "cod_eligible": false})
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, output)
}
