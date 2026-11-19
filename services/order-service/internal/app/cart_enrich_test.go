package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/omamx/order-service/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

func cartTestService(t *testing.T, catalogURL string) (*Service, domain.Principal) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	principal := domain.Principal{ID: "customer-1", Role: "CUSTOMER"}
	cart := domain.Cart{
		Revision: 3,
		Items:    []domain.Item{{ItemID: "MX-500", SKU: "MX-500", Quantity: 2}},
	}
	raw, err := json.Marshal(cart)
	if err != nil {
		t.Fatal(err)
	}
	mr.Set("cart:"+principal.Scope(), string(raw))
	return &Service{
		C:      Config{CatalogURL: catalogURL, InternalToken: "test-internal-token"},
		Redis:  client,
		Client: http.DefaultClient,
	}, principal
}

func catalogStub(t *testing.T, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/internal/catalog/snapshots" {
			http.NotFound(w, r)
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"snapshot_id": "11111111-1111-1111-1111-111111111111",
			"catalog_version": 7,
			"valid_until": %q,
			"items": [{
				"sku_code": "MX-500",
				"quantity": 2,
				"product_name": "Me Xung Gion",
				"unit_price": 345000,
				"weight_grams": 500,
				"packaging_type": "BOX",
				"is_active": true
			}],
			"subtotal": 690000,
			"contract_version": "catalog-checkout-1.0",
			"channel": "D2C_WEB"
		}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano))
	}))
	t.Cleanup(server.Close)
	return server
}

// F-35: a cart read must carry the catalog's name/price/weight for each
// item - the cart itself persists only SKU + quantity.
func TestCartEnrichesItemsFromCatalog(t *testing.T) {
	server := catalogStub(t, http.StatusOK)
	service, principal := cartTestService(t, server.URL)

	cart, err := service.Cart(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if cart.Revision != 3 || len(cart.Items) != 1 {
		t.Fatalf("unexpected cart: %+v", cart)
	}
	item := cart.Items[0]
	if item.Name != "Me Xung Gion" || item.UnitPrice != 345000 || item.Weight != 500 || item.Version != 7 {
		t.Fatalf("item not enriched: %+v", item)
	}
}

// F-35: when enrichment is impossible (catalog failing), the cart read
// degrades to the SKU-only cart instead of failing.
func TestCartDegradesWhenCatalogFails(t *testing.T) {
	server := catalogStub(t, http.StatusInternalServerError)
	service, principal := cartTestService(t, server.URL)

	cart, err := service.Cart(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("unexpected cart: %+v", cart)
	}
	item := cart.Items[0]
	if item.SKU != "MX-500" || item.Quantity != 2 || item.Name != "" || item.UnitPrice != 0 {
		t.Fatalf("expected untouched SKU-only item, got %+v", item)
	}
}

// F-35: without a catalog URL configured there is nothing to enrich
// with; the cart read must behave exactly as before.
func TestCartWithoutCatalogConfigured(t *testing.T) {
	service, principal := cartTestService(t, "")

	cart, err := service.Cart(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Name != "" || cart.Items[0].UnitPrice != 0 {
		t.Fatalf("expected SKU-only cart, got %+v", cart)
	}
}
