package simulator

import (
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"net/http"
	"strconv"
	"time"
)

// Explicit TEST provider contract. It obtains price/weight from the provider DB;
// it never loads example Order snapshots or forges consumer-side Inventory proof.
func (s *Simulator) coreRoutes(api chi.Router) {
	api.Post("/order-core/terms", func(w http.ResponseWriter, r *http.Request) {
		var q domain.Quote
		if e := read(r, &q); e != nil || q.Address.Validate() != nil || q.Address.Province != "75" || q.Address.Ward != "HUE-01" {
			errResponse(w, domain.E("INVALID_QUOTE_INPUT", 422))
			return
		}
		ctx := r.Context()
		var now time.Time
		if e := s.DB.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			errResponse(w, e)
			return
		}
		terms := core.SnapshotInput{WarehouseID: "HUE", Catalog: core.CatalogEvidence{SnapshotID: uuid.NewString(), Version: "1", ValidUntil: now.Add(15 * time.Minute)}, Items: []core.SnapshotItem{}, Address: core.ShippingAddress{RecipientName: q.Address.Name, PhoneE164: q.Address.Phone, Street: q.Address.Street, ProvinceCode: q.Address.Province, ProvinceName: "Hue", WardCode: q.Address.Ward, WardName: "Hue test ward", Source: core.AddressSource{Kind: "VALIDATED_INLINE", ValidationReference: uuid.NewString()}}, Shipping: core.ShippingTerms{CarrierCode: "TEST_CARRIER", ServiceCode: "STANDARD", QuoteReference: uuid.NewString(), QuoteVersion: "test-shipping-1", ValidUntil: now.Add(15 * time.Minute), QuotedFee: 30000}, Policies: core.Policies{Order: "order-core-1.0", Checkout: "cart-checkout-1.0", Payment: "payment-sepay-1.1", Shipping: "test-shipping-1"}}
		if q.AddressID != "" {
			id, version := q.AddressID, q.AddressVersion
			terms.Address.Source.Kind = "PROFILE_ADDRESS"
			terms.Address.Source.ProfileAddressID = &id
			terms.Address.Source.AddressVersion = &version
		}
		var subtotal, weight int64
		for _, i := range q.Items {
			var price, grams, version int64
			var name string
			var active bool
			if e := s.DB.QueryRow(ctx, "SELECT price,weight,name,active,version FROM catalog WHERE sku=$1", i.SKU).Scan(&price, &grams, &name, &active, &version); e != nil || !active || i.Quantity < 1 || i.Quantity > 999 {
				errResponse(w, domain.E("SKU_NOT_AVAILABLE", 422))
				return
			}
			if price < 1 || price > domain.MaxMoney/int64(i.Quantity) || subtotal > domain.MaxMoney-price*int64(i.Quantity) {
				errResponse(w, domain.E("INVALID_TOTAL", 422))
				return
			}
			subtotal += price * int64(i.Quantity)
			weight += grams * int64(i.Quantity)
			// This sandbox exposes a coherent catalog revision; per-item revision changes
			// are pinned in its composite version and rechecked before reservation.
			terms.Catalog.Version += "/" + i.SKU + ":" + strconv.FormatInt(version, 10)
			terms.Items = append(terms.Items, core.SnapshotItem{LineID: uuid.NewString(), SKU: i.SKU, Name: name, Quantity: int64(i.Quantity), UnitPrice: price, LineSubtotal: price * int64(i.Quantity), WeightGrams: grams, Packaging: core.Packaging{Description: "Test sealed box", UnitCode: "BOX", Version: "test-pack-1"}})
		}
		for i := range terms.Items {
			terms.Items[i].CatalogVersion = terms.Catalog.Version
		}
		terms.Shipping.PackageWeightGrams = weight
		terms.Money = core.SnapshotMoney{Currency: "VND", Subtotal: subtotal, ShippingFee: 30000, FinalAmount: subtotal + 30000}
		terms.Payment.Method = q.Method
		if q.Method == "COD" {
			policy := "test-cod-1"
			amount := terms.Money.FinalAmount
			terms.Payment.CODPolicyReference = &policy
			terms.Payment.CollectAmount = &amount
		}
		response(w, 200, terms)
	})
	api.Post("/order-core/revalidate", func(w http.ResponseWriter, r *http.Request) {
		var terms core.SnapshotInput
		if e := read(r, &terms); e != nil {
			errResponse(w, domain.E("INVALID_TERMS", 400))
			return
		}
		valid := time.Now().Before(terms.Catalog.ValidUntil) && time.Now().Before(terms.Shipping.ValidUntil)
		version := "1"
		for _, i := range terms.Items {
			var price, grams, v int64
			var name string
			var active bool
			if e := s.DB.QueryRow(r.Context(), "SELECT price,weight,name,active,version FROM catalog WHERE sku=$1", i.SKU).Scan(&price, &grams, &name, &active, &v); e != nil {
				errResponse(w, e)
				return
			}
			valid = valid && active && i.UnitPrice == price && i.WeightGrams == grams && i.Name == name
			version += "/" + i.SKU + ":" + strconv.FormatInt(v, 10)
		}
		valid = valid && version == terms.Catalog.Version && terms.Shipping.QuotedFee == 30000 && terms.Address.ProvinceCode == "75" && terms.Address.WardCode == "HUE-01"
		response(w, 200, map[string]bool{"valid": valid})
	})
}
