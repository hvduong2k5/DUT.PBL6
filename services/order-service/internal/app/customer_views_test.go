package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
)

func f36CoreSnapshot() core.SnapshotInput {
	return core.SnapshotInput{
		SnapshotID: "snap-1",
		Provenance: core.Provenance{QuoteID: "quote-1", CartRevision: 3},
		Catalog:    core.CatalogEvidence{CatalogVersion: "1", EvaluatedAt: time.Now().Add(-time.Minute), Channel: "D2C_WEB"},
		Items: []core.SnapshotItem{
			{SKU: "SKU-002-200", Name: "Mè Xửng Dẻo Ngự Trà", Quantity: 2, UnitPrice: 85000, WeightGrams: 200},
		},
		Address: core.ShippingAddress{
			RecipientName: "Golden Audit", PhoneE164: "+84900000333", Street: "2 Le Loi",
			ProvinceCode: "46", ProvinceName: "Thành phố Huế",
			WardCode: "19753", WardName: "Phường Phú Xuân",
		},
		Shipping: core.ShippingTerms{QuoteReference: "ship-1", ValidUntil: time.Now().Add(2 * time.Hour)},
		Money:    core.SnapshotMoney{Currency: "VND", Subtotal: 170000, ShippingFee: 27000, FinalAmount: 197000},
		Payment:  core.PaymentTerms{Method: "VIETQR"},
	}
}

func f36Order() domain.Order {
	return domain.Order{
		ID: "11111111-1111-1111-1111-111111111111", Code: "OM-TEST-1",
		Scope: "customer:principal-1", Status: "PAID", Payment: "CONFIRMED",
		Stock: "COMMITTED", Method: "VIETQR",
		Subtotal: 170000, Shipping: 27000, Final: 197000, Version: 4,
	}
}

var f36Principal = domain.Principal{Role: "CUSTOMER", ID: "principal-1"}

// The customer list handler projects with the snapshot loaded; the frozen
// snapshot must be emitted even when the projection runs in list mode.
func TestCustomerOrderEmitsCoreSnapshotInListMode(t *testing.T) {
	snap := f36CoreSnapshot()
	o := f36Order()
	o.CoreSnapshot = &snap
	out := (&Service{}).CustomerOrder(o, f36Principal, false).(map[string]any)
	raw, ok := out["snapshot"]
	if !ok {
		t.Fatal("snapshot missing from list-mode customer projection")
	}
	q, ok := raw.(domain.Quote)
	if !ok {
		t.Fatalf("snapshot has type %T, want domain.Quote", raw)
	}
	if len(q.Items) != 1 || q.Items[0].SKU != "SKU-002-200" || q.Items[0].Name != "Mè Xửng Dẻo Ngự Trà" || q.Items[0].Quantity != 2 {
		t.Fatalf("snapshot items wrong: %+v", q.Items)
	}
	if q.Address.WardName != "Phường Phú Xuân" || q.Address.ProvinceName != "Thành phố Huế" {
		t.Fatalf("snapshot address names wrong: %+v", q.Address)
	}
	if q.Address.Ward != "19753" || q.Address.Province != "46" {
		t.Fatalf("snapshot address codes wrong: %+v", q.Address)
	}
}

// A legacy order's stored quote must survive the customer projection instead
// of being deleted when no core snapshot exists.
func TestCustomerOrderKeepsLegacySnapshot(t *testing.T) {
	legacy := &domain.Quote{ID: "legacy-quote", Items: []domain.Item{{SKU: "MX-500", Name: "Mè Xửng giòn", Quantity: 1}}}
	o := f36Order()
	o.Snapshot = legacy
	out := (&Service{}).CustomerOrder(o, f36Principal, true).(map[string]any)
	if _, ok := out["snapshot"]; !ok {
		t.Fatal("legacy snapshot was deleted from the customer projection")
	}
}

func TestCustomerOrderOmitsSnapshotWhenNotLoaded(t *testing.T) {
	o := f36Order()
	out := (&Service{}).CustomerOrder(o, f36Principal, false).(map[string]any)
	if _, ok := out["snapshot"]; ok {
		t.Fatal("snapshot present although none was loaded")
	}
}

// Address names ride along in JSON only when set, so older payloads and
// consumers that ignore them are unaffected.
func TestAddressNamesJSON(t *testing.T) {
	withNames, _ := json.Marshal(domain.Address{Ward: "19753", WardName: "Phường Phú Xuân", Province: "46", ProvinceName: "Thành phố Huế"})
	var decoded map[string]any
	if err := json.Unmarshal(withNames, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["ward_name"] != "Phường Phú Xuân" || decoded["province_name"] != "Thành phố Huế" {
		t.Fatalf("names missing from address JSON: %s", withNames)
	}
	withoutNames, _ := json.Marshal(domain.Address{Ward: "19753", Province: "46"})
	decoded = map[string]any{}
	if err := json.Unmarshal(withoutNames, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["ward_name"]; ok {
		t.Fatalf("empty ward_name should be omitted: %s", withoutNames)
	}
}

func TestAddressNameOrCode(t *testing.T) {
	if got := addressNameOrCode("Phường Phú Xuân", "19753"); got != "Phường Phú Xuân" {
		t.Fatalf("want name, got %q", got)
	}
	if got := addressNameOrCode("", "19753"); got != "19753" {
		t.Fatalf("want code fallback, got %q", got)
	}
}
