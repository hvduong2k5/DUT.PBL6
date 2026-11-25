package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
)

func TestGuestCartMutationWithSingleDatabaseConnection(t *testing.T) {
	h := newHarness(t)
	previous := h.S.DB
	cfg := previous.Config()
	cfg.MaxConns = 1
	limited, err := pgxpool.NewWithConfig(h.ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	h.S.DB = limited
	defer func() { h.S.DB = previous; limited.Close() }()
	enableKeycloakGuest(t, h)
	ctx, cancel := context.WithTimeout(h.ctx, time.Second)
	defer cancel()
	cart, err := h.S.MutateCart(ctx, h.Principal, "MX-GION-500G", 1, 0, true)
	if err != nil || cart.Revision != 1 || len(cart.Items) != 1 {
		t.Fatal("guest mutation borrowed a nested database connection", cart, err)
	}
	if err = h.S.GuestSessions().RevokeGuestSession(ctx, h.Principal.ID, h.Token); err != nil {
		t.Fatal(err)
	}
	if _, err = h.S.MutateCart(ctx, h.Principal, "MX-GION-500G", 1, 1, true); err == nil {
		t.Fatal("revoked guest mutated the cart")
	}
}

func mergeCarts(t *testing.T) (*harness, domain.Principal, domain.Principal, string) {
	t.Helper()
	h := newHarness(t)
	enableKeycloakGuest(t, h)
	h.S.C.GuestCartMergeEnabled = true
	guest, token := h.Principal, h.Token
	customer := domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER", Kind: domain.Customer, CustomerID: uuid.NewString()}
	if _, e := h.S.MutateCart(h.ctx, guest, "MX-GION-500G", 2, 0, true); e != nil {
		t.Fatal(e)
	}
	if _, e := h.S.MutateCart(h.ctx, customer, "MX-GION-500G", 3, 0, true); e != nil {
		t.Fatal(e)
	}
	return h, guest, customer, token
}

func TestGuestCartMergePersistsReplayRevocationAndFreshPrices(t *testing.T) {
	h, guest, customer, token := mergeCarts(t)
	// Stored prices are untrusted, even if a stale guest cart contains them.
	raw, _ := json.Marshal(domain.Cart{Revision: 2, Items: []domain.Item{{SKU: "MX-GION-500G", Quantity: 2, UnitPrice: 1}, {SKU: "DELETED-SKU", Quantity: 1, UnitPrice: 999999}}})
	if e := h.S.Redis.Set(h.ctx, "cart:"+guest.Scope(), raw, time.Hour).Err(); e != nil {
		t.Fatal(e)
	}
	key := uuid.NewString()
	result, e := h.S.MergeGuestCart(h.ctx, customer, token, key)
	if e != nil {
		t.Fatal(e)
	}
	if result.Cart.Revision != 2 || len(result.Cart.Items) != 1 || result.Cart.Items[0].Quantity != 5 || result.Cart.Items[0].UnitPrice != 110000 {
		t.Fatal("merge sum/revision/fresh price", result)
	}
	dropped := false
	for _, item := range result.Items {
		if item.SKU == "DELETED-SKU" && item.Status == "dropped" && item.Reason == "SKU_NOT_AVAILABLE" {
			dropped = true
		}
	}
	if !dropped {
		t.Fatal("deleted SKU outcome missing", result.Items)
	}
	if e = h.S.GuestSessions().CheckGuestSession(h.ctx, guest.ID); e == nil {
		t.Fatal("source session still active")
	}
	if h.count("SELECT count(*) FROM guest_cart_merges") != 1 || h.count("SELECT count(*) FROM storefront_carts") != 2 {
		t.Fatal("atomic merge boundary missing")
	}
	dependency := h.S.C.Dependencies
	h.S.C.Dependencies = "http://127.0.0.1:1"
	again, e := h.S.MergeGuestCart(h.ctx, customer, token, key)
	h.S.C.Dependencies = dependency
	if e != nil || again.Cart.Items[0].Quantity != 5 || !again.MergedAt.Equal(result.MergedAt) {
		t.Fatal("retry re-merged or refetched Catalog", again, e)
	}
	fetched, e := h.S.CartMergeResult(h.ctx, customer, key)
	if e != nil || fetched.Cart.Items[0].Quantity != 5 {
		t.Fatal("customer cannot retrieve result after revocation", e)
	}
	other := domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER"}
	if _, e = h.S.CartMergeResult(h.ctx, other, key); !errors.Is(e, pgx.ErrNoRows) {
		t.Fatal("cross-customer merge result visible", e)
	}
	if _, e = h.S.MergeGuestCart(h.ctx, customer, token, uuid.NewString()); !errors.Is(e, persistence.ErrScope) {
		t.Fatal("new key re-merged revoked source", e)
	}
	h.S.C.GuestCartMergeEnabled = false
	cart, e := h.S.Cart(h.ctx, customer)
	if e != nil || cart.Items[0].Quantity != 5 {
		t.Fatal("flag-off resurrected Redis cart", cart, e)
	}
	cart, e = h.S.MutateCart(h.ctx, customer, "MX-GION-500G", 6, cart.Revision, false)
	if e != nil || cart.Revision != 3 || cart.Items[0].Quantity != 6 {
		t.Fatal("converted cart mutation", cart, e)
	}
}

func TestGuestCartMergeRollbackKeepsCartLedgerAndSessionTogether(t *testing.T) {
	h, guest, customer, token := mergeCarts(t)
	// Fail the final revocation after both SQL cart writes and ledger insertion.
	_, e := h.S.DB.Exec(h.ctx, `CREATE FUNCTION reject_guest_revocation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected revocation failure'; END $$;
 CREATE TRIGGER reject_guest_revocation BEFORE UPDATE ON guest_checkout_sessions FOR EACH ROW EXECUTE FUNCTION reject_guest_revocation()`)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.S.MergeGuestCart(h.ctx, customer, token, uuid.NewString()); e == nil {
		t.Fatal("injected transaction failure ignored")
	}
	if h.count("SELECT count(*) FROM guest_cart_merges") != 0 || h.count("SELECT count(*) FROM storefront_carts") != 0 {
		t.Fatal("partial merge committed")
	}
	if e = h.S.GuestSessions().CheckGuestSession(h.ctx, guest.ID); e != nil {
		t.Fatal("failed merge revoked source", e)
	}
	cart, e := h.S.Cart(h.ctx, customer)
	if e != nil || cart.Items[0].Quantity != 3 || cart.Revision != 1 {
		t.Fatal("failed merge changed target", cart, e)
	}
}

func TestMergedCartRetainsCatalogDisplayMetadata(t *testing.T) {
	h, _, customer, token := mergeCarts(t)
	key := uuid.NewString()
	if _, err := h.S.MergeGuestCart(h.ctx, customer, token, key); err != nil {
		t.Fatal(err)
	}
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Items []domain.Item `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Items) != 1 {
			http.Error(w, "invalid selection", 400)
			return
		}
		quantity := in.Items[0].Quantity
		json.NewEncoder(w).Encode(map[string]any{"snapshot_id": uuid.NewString(), "catalog_version": 2, "valid_until": time.Now().Add(time.Minute), "contract_version": "catalog-checkout-1.0", "channel": "D2C_WEB", "subtotal": 120000 * quantity, "items": []map[string]any{{"sku_code": "MX-GION-500G", "quantity": quantity, "product_name": "Current catalog name", "unit_price": 120000, "weight_grams": 500, "packaging_type": "BOX", "is_active": true}}})
	}))
	defer catalog.Close()
	h.S.C.CatalogURL = catalog.URL
	cart, err := h.S.Cart(h.ctx, customer)
	if err != nil || cart.Items[0].Name != "Current catalog name" || cart.Items[0].UnitPrice != 120000 {
		t.Fatal("SQL cart read lost current Catalog display metadata", cart, err)
	}
	cart, err = h.S.MutateCart(h.ctx, customer, "MX-GION-500G", 6, cart.Revision, false)
	if err != nil || cart.Items[0].Name != "Current catalog name" || cart.Items[0].Quantity != 6 || cart.Items[0].UnitPrice != 120000 {
		t.Fatal("SQL cart mutation lost Catalog display metadata", cart, err)
	}
	result, err := h.S.CartMergeResult(h.ctx, customer, key)
	if err != nil || result.Cart.Items[0].Quantity != 5 || result.Cart.Items[0].UnitPrice != 110000 {
		t.Fatal("display enrichment changed the immutable merge result", result, err)
	}
}

func TestGuestCartMergeConcurrentRetryAndQuantityCap(t *testing.T) {
	h, guest, customer, token := mergeCarts(t)
	if _, e := h.S.MutateCart(h.ctx, guest, "MX-GION-500G", 999, 1, false); e != nil {
		t.Fatal(e)
	}
	key := uuid.NewString()
	var wg sync.WaitGroup
	results := make(chan domain.CartMergeResult, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := h.S.MergeGuestCart(h.ctx, customer, token, key)
			if e != nil {
				t.Error(e)
				return
			}
			results <- r
		}()
	}
	wg.Wait()
	close(results)
	for result := range results {
		if result.Cart.Revision != 2 || result.Cart.Items[0].Quantity != 999 || result.Items[0].Status != "adjusted" || result.Items[0].Reason != "QUANTITY_LIMIT" {
			t.Fatal(result)
		}
	}
	if h.count("SELECT count(*) FROM guest_cart_merges") != 1 {
		t.Fatal("concurrent retry duplicated ledger")
	}
	// A valid signature with an inactive source is rejected under a new key.
	if _, e := h.S.AuthenticateGuestSession(h.ctx, "Bearer "+token, true); e == nil {
		t.Fatal("revoked guest authenticated")
	}
	if _, e := h.S.MergeGuestCart(h.ctx, customer, token, strings.Repeat("x", 32)); !errors.Is(e, persistence.ErrScope) {
		t.Fatal(e)
	}
}

func TestMergedSQLCartCheckoutCleanupAndConcurrentEdits(t *testing.T) {
	for _, edit := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup", true: "preserve-edit"}[edit], func(t *testing.T) {
			customer := domain.Principal{ID: uuid.NewString(), Role: "CUSTOMER", Kind: domain.Customer, CustomerID: uuid.NewString()}
			profile := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{"id": customer.CustomerID, "user_id": r.Header.Get("X-User-ID"), "status": "ACTIVE"})
			}))
			defer profile.Close()
			h, _, _ := connectedHarness(t, func(h *harness) {
				enableKeycloakGuest(t, h)
				h.S.C.GuestCartMergeEnabled = true
				h.S.C.Profile = profile.URL
				h.S.C.ProfileToken = "profile-test-internal-token"
			})
			guest, token := h.Principal, h.Token
			if _, e := h.S.MutateCart(h.ctx, guest, "MX-GION-500G", 2, 0, true); e != nil {
				t.Fatal(e)
			}
			if _, e := h.S.MutateCart(h.ctx, customer, "MX-GION-500G", 3, 0, true); e != nil {
				t.Fatal(e)
			}
			if _, e := h.S.MergeGuestCart(h.ctx, customer, token, uuid.NewString()); e != nil {
				t.Fatal(e)
			}
			q, e := h.S.CreateQuote(h.ctx, customer, domain.QuoteInput{Revision: 2, Method: "COD", Address: &domain.Address{Name: "Customer", Phone: "0905123456", Street: "Test only", Ward: "WARD-TH-001", Province: "75"}})
			if e != nil {
				t.Fatal(e)
			}
			lock, e := h.S.DB.Begin(h.ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer lock.Rollback(h.ctx)
			if e = persistence.LockCartScopes(h.ctx, lock, customer.Scope()); e != nil {
				t.Fatal(e)
			}
			op, _, e := h.S.Accept(h.ctx, customer, uuid.NewString(), domain.CheckoutInput{QuoteID: q.ID, Revision: 2, Method: "COD"})
			if e != nil {
				t.Fatal(e)
			}
			untilConnected(t, func() bool {
				op, e = h.S.Operation(h.ctx, op.ID, customer)
				return e == nil && op.Status == "SUCCEEDED"
			})
			if edit {
				cart, e := h.S.Cart(h.ctx, customer)
				if e != nil {
					t.Fatal(e)
				}
				cart.Revision++
				cart.Items[0].Quantity = 6
				var id string
				if e = lock.QueryRow(h.ctx, "SELECT cart_id FROM storefront_carts WHERE scope=$1", customer.Scope()).Scan(&id); e != nil {
					t.Fatal(e)
				}
				if e = persistence.WriteStorefrontCart(h.ctx, lock, customer.Scope(), cart, id); e != nil {
					t.Fatal(e)
				}
			}
			if e = lock.Commit(h.ctx); e != nil {
				t.Fatal(e)
			}
			untilConnected(t, func() bool {
				return h.count("SELECT count(*) FROM jobs WHERE kind='CART_CLEANUP' AND status='DONE'") == 1
			})
			cart, e := h.S.Cart(h.ctx, customer)
			if e != nil {
				t.Fatal(e)
			}
			if edit {
				if len(cart.Items) != 1 || cart.Items[0].Quantity != 6 {
					t.Fatal("cleanup erased later edit", cart)
				}
			} else if len(cart.Items) != 0 {
				t.Fatal("SQL cart not cleaned", cart)
			}
		})
	}
}
