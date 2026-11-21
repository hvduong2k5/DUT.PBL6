package app

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
)

type cartReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Service) sqlCarts() bool { return s.C.AuthMode == "keycloak" && s.DB != nil }

func readSQLCart(ctx context.Context, q cartReader, scope string) (cart domain.Cart, id string, err error) {
	var raw []byte
	var expired bool
	err = q.QueryRow(ctx, "SELECT revision,items,cart_id,expires_at<=clock_timestamp() FROM storefront_carts WHERE scope=$1", scope).Scan(&cart.Revision, &raw, &id, &expired)
	if err != nil {
		return
	}
	if expired {
		cart.Revision++
		cart.Items = []domain.Item{}
		return
	}
	err = json.Unmarshal(raw, &cart.Items)
	return
}

func (s *Service) cartAtMerge(ctx context.Context, tx pgx.Tx, p domain.Principal) (domain.Cart, error) {
	cart, _, e := readSQLCart(ctx, tx, p.Scope())
	if errors.Is(e, pgx.ErrNoRows) {
		return s.redisCart(ctx, p)
	}
	return cart, e
}

func (s *Service) mutateStorefrontCart(ctx context.Context, p domain.Principal, sku string, qty int, revision int64, add bool) (domain.Cart, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return domain.Cart{}, e
	}
	defer tx.Rollback(ctx)
	if e = persistence.LockCartScopes(ctx, tx, p.Scope()); e != nil {
		return domain.Cart{}, e
	}
	if p.PrincipalKind() == domain.Guest {
		if s.GuestSessions() == nil || s.GuestSessions().CheckGuestSession(ctx, p.ID) != nil {
			return domain.Cart{}, domain.E("UNAUTHENTICATED", 401)
		}
	}
	cart, id, e := readSQLCart(ctx, tx, p.Scope())
	if errors.Is(e, pgx.ErrNoRows) {
		// Hold the same SQL scope fence while changing an unconverted Redis cart.
		// A merge can then import exactly one stable Redis revision.
		return s.mutateRedisCart(ctx, p, sku, qty, revision, add)
	}
	if e != nil {
		return cart, e
	}
	if cart.Revision != revision {
		return cart, domain.E("VERSION_CONFLICT", 409)
	}
	counts := map[string]int{}
	for _, item := range cart.Items {
		counts[item.SKU] = item.Quantity
	}
	if add {
		qty += counts[sku]
	}
	if qty > 999 {
		return cart, domain.E("INVALID_QUANTITY", 422)
	}
	if qty == 0 {
		delete(counts, sku)
	} else {
		counts[sku] = qty
	}
	if len(counts) > 100 {
		return cart, domain.E("TOO_MANY_ITEMS", 422)
	}
	cart.Items = []domain.Item{}
	for code, count := range counts {
		cart.Items = append(cart.Items, domain.Item{ItemID: code, SKU: code, Quantity: count})
	}
	sort.Slice(cart.Items, func(i, j int) bool { return cart.Items[i].SKU < cart.Items[j].SKU })
	cart.Revision++
	if e = persistence.WriteStorefrontCart(ctx, tx, p.Scope(), cart, id); e != nil {
		return cart, e
	}
	return cart, tx.Commit(ctx)
}

func (s *Service) coreCartID(ctx context.Context, p domain.Principal, create bool) (string, error) {
	if s.sqlCarts() {
		_, id, e := readSQLCart(ctx, s.DB, p.Scope())
		if e == nil {
			return id, nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return "", e
		}
	}
	key := "core-cart-id:" + p.Scope()
	if create {
		if e := s.Redis.SetNX(ctx, key, uuid.NewString(), 30*24*time.Hour).Err(); e != nil {
			return "", e
		}
	}
	return s.Redis.Get(ctx, key).Result()
}

func (s *Service) mergedCatalogItem(ctx context.Context, sku string) (domain.Item, error) {
	var out struct {
		Items []domain.Item `json:"items"`
	}
	e := s.Call(ctx, "POST", "/catalog/validate", map[string]any{"items": []domain.Item{{SKU: sku, Quantity: 1}}}, &out)
	if e != nil {
		return domain.Item{}, e
	}
	if len(out.Items) != 1 || out.Items[0].SKU != sku || out.Items[0].UnitPrice < 1 || out.Items[0].UnitPrice > domain.MaxMoney || out.Items[0].Weight < 1 {
		return domain.Item{}, domain.E("CATALOG_INCONSISTENT", 503)
	}
	return out.Items[0], nil
}

type mergeCatalogCheck struct {
	item domain.Item
	err  error
}

func (s *Service) mergeCatalogChecks(ctx context.Context, codes []string) (map[string]mergeCatalogCheck, error) {
	result := map[string]mergeCatalogCheck{}
	for start := 0; start < len(codes); start += 100 {
		end := start + 100
		if end > len(codes) {
			end = len(codes)
		}
		batch := codes[start:end]
		selection := make([]domain.Item, 0, len(batch))
		for _, code := range batch {
			selection = append(selection, domain.Item{SKU: code, Quantity: 1})
		}
		var out struct {
			Items []domain.Item `json:"items"`
		}
		err := s.Call(ctx, "POST", "/catalog/validate", map[string]any{"items": selection}, &out)
		if err == nil {
			if len(out.Items) != len(batch) {
				return nil, domain.E("CATALOG_INCONSISTENT", 503)
			}
			for i, item := range out.Items {
				if item.SKU != batch[i] || item.UnitPrice < 1 || item.UnitPrice > domain.MaxMoney || item.Weight < 1 {
					return nil, domain.E("CATALOG_INCONSISTENT", 503)
				}
				result[item.SKU] = mergeCatalogCheck{item: item}
			}
			continue
		}
		var unavailable *domain.Error
		if !errors.As(err, &unavailable) || unavailable.Code != "SKU_NOT_AVAILABLE" {
			return nil, err
		}
		// Only an unavailable batch needs individual checks to retain valid SKUs.
		checks := make([]mergeCatalogCheck, len(batch))
		done := make(chan int, len(batch))
		slots := make(chan struct{}, 8)
		for i, code := range batch {
			go func(i int, code string) {
				slots <- struct{}{}
				checks[i].item, checks[i].err = s.mergedCatalogItem(ctx, code)
				<-slots
				done <- i
			}(i, code)
		}
		for range batch {
			<-done
		}
		for i, code := range batch {
			result[code] = checks[i]
		}
	}
	return result, nil
}

func (s *Service) MergeGuestCart(ctx context.Context, customer domain.Principal, token, key string) (domain.CartMergeResult, error) {
	if customer.PrincipalKind() != domain.Customer {
		return domain.CartMergeResult{}, domain.E("PERMISSION_DENIED", 403)
	}
	if s.C.AuthMode != "keycloak" || !s.C.GuestCartMergeEnabled {
		return domain.CartMergeResult{}, domain.E("GUEST_CART_MERGE_DISABLED", 403)
	}
	if s.GuestSessions() == nil {
		return domain.CartMergeResult{}, domain.E("GUEST_SESSION_UNAVAILABLE", 503)
	}
	guest, e := s.AuthenticateGuestSession(ctx, "Bearer "+token, false)
	if e != nil {
		return domain.CartMergeResult{}, e
	}
	return s.GuestSessions().MergeGuestCart(ctx, customer.Scope(), guest.ID, token, key, func(tx pgx.Tx) (domain.CartMergeResult, error) {
		target, e := s.cartAtMerge(ctx, tx, customer)
		if e != nil {
			return domain.CartMergeResult{}, e
		}
		source, e := s.cartAtMerge(ctx, tx, guest)
		if e != nil {
			return domain.CartMergeResult{}, e
		}
		counts := map[string]int{}
		targetSKUs := map[string]bool{}
		for _, cart := range []domain.Cart{target, source} {
			for _, item := range cart.Items {
				if item.Quantity < 1 || item.Quantity > 999 || len(item.SKU) == 0 || len(item.SKU) > 64 {
					return domain.CartMergeResult{}, domain.E("CART_INVALID", 503)
				}
				counts[item.SKU] += item.Quantity
			}
		}
		for _, item := range target.Items {
			targetSKUs[item.SKU] = true
		}
		codes := make([]string, 0, len(counts))
		for code := range counts {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		bySKU, e := s.mergeCatalogChecks(ctx, codes)
		if e != nil {
			return domain.CartMergeResult{}, e
		}
		// Existing target lines have first claim on the 100-line cart limit.
		sort.SliceStable(codes, func(i, j int) bool { return targetSKUs[codes[i]] && !targetSKUs[codes[j]] })
		result := domain.CartMergeResult{Cart: domain.Cart{Revision: target.Revision + 1, Items: []domain.Item{}}, Items: []domain.CartMergeItem{}}
		for _, code := range codes {
			count := counts[code]
			outcome := domain.CartMergeItem{SKU: code, Requested: count, Quantity: count, Status: "merged"}
			check := bySKU[code]
			if check.err != nil {
				var unavailable *domain.Error
				if !errors.As(check.err, &unavailable) || unavailable.Code != "SKU_NOT_AVAILABLE" {
					return result, check.err
				}
				outcome.Status = "dropped"
				outcome.Reason = "SKU_NOT_AVAILABLE"
				outcome.Quantity = 0
			} else if len(result.Cart.Items) >= 100 {
				outcome.Status = "dropped"
				outcome.Reason = "CART_ITEM_LIMIT"
				outcome.Quantity = 0
			} else {
				if count > 999 {
					outcome.Quantity = 999
					outcome.Status = "adjusted"
					outcome.Reason = "QUANTITY_LIMIT"
				}
				item := check.item
				item.ItemID = code
				item.Quantity = outcome.Quantity
				result.Cart.Items = append(result.Cart.Items, item)
			}
			result.Items = append(result.Items, outcome)
		}
		sort.Slice(result.Cart.Items, func(i, j int) bool { return result.Cart.Items[i].SKU < result.Cart.Items[j].SKU })
		if e = persistence.WriteStorefrontCart(ctx, tx, customer.Scope(), result.Cart, uuid.NewString()); e != nil {
			return result, e
		}
		source.Items = []domain.Item{}
		source.Revision++
		if e = persistence.WriteStorefrontCart(ctx, tx, guest.Scope(), source, uuid.NewString()); e != nil {
			return result, e
		}
		return result, nil
	})
}

func (s *Service) CartMergeResult(ctx context.Context, p domain.Principal, key string) (domain.CartMergeResult, error) {
	if p.PrincipalKind() != domain.Customer {
		return domain.CartMergeResult{}, domain.E("PERMISSION_DENIED", 403)
	}
	if s.DB == nil {
		return domain.CartMergeResult{}, domain.E("GUEST_SESSION_UNAVAILABLE", 503)
	}
	if len(key) < 16 || len(key) > 128 || strings.TrimSpace(key) != key {
		return domain.CartMergeResult{}, domain.E("INVALID_IDEMPOTENCY_KEY", 400)
	}
	for _, r := range key {
		if r < 33 || r > 126 {
			return domain.CartMergeResult{}, domain.E("INVALID_IDEMPOTENCY_KEY", 400)
		}
	}
	var raw []byte
	var result domain.CartMergeResult
	err := s.DB.QueryRow(ctx, "SELECT response FROM guest_cart_merges WHERE customer_scope=$1 AND merge_key=$2", p.Scope(), key).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &result)
	}
	return result, err
}
