package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/redis/go-redis/v9"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *Service) Cart(ctx context.Context, p domain.Principal) (domain.Cart, error) {
	v, e := s.Redis.Get(ctx, "cart:"+p.Scope()).Bytes()
	if errors.Is(e, redis.Nil) {
		return domain.Cart{Items: []domain.Item{}}, nil
	}
	if e != nil {
		return domain.Cart{}, domain.E("CART_UNAVAILABLE", 503)
	}
	var c domain.Cart
	e = json.Unmarshal(v, &c)
	return c, e
}
func (s *Service) MutateCart(ctx context.Context, p domain.Principal, sku string, qty int, revision int64, add bool) (domain.Cart, error) {
	if len(sku) == 0 || len(sku) > 64 || qty < 0 || qty > 999 || revision < 0 {
		return domain.Cart{}, domain.E("INVALID_CART_INPUT", 400)
	}
	var result domain.Cart
	key := "cart:" + p.Scope()
	for attempt := 0; attempt < 10; attempt++ {
		e := s.Redis.Watch(ctx, func(tx *redis.Tx) error {
			var c domain.Cart
			b, e := tx.Get(ctx, key).Bytes()
			if e != nil && !errors.Is(e, redis.Nil) {
				return domain.E("CART_UNAVAILABLE", 503)
			}
			if len(b) > 0 {
				if e = json.Unmarshal(b, &c); e != nil {
					return e
				}
			}
			if c.Revision != revision {
				return domain.E("VERSION_CONFLICT", 409)
			}
			items := map[string]int{}
			for _, i := range c.Items {
				items[i.SKU] = i.Quantity
			}
			if add {
				qty += items[sku]
			}
			if qty > 999 {
				return domain.E("INVALID_QUANTITY", 422)
			}
			if qty == 0 {
				delete(items, sku)
			} else {
				items[sku] = qty
			}
			if len(items) > 100 {
				return domain.E("TOO_MANY_ITEMS", 422)
			}
			c.Items = []domain.Item{}
			for code, count := range items {
				c.Items = append(c.Items, domain.Item{ItemID: code, SKU: code, Quantity: count})
			}
			sort.Slice(c.Items, func(i, j int) bool { return c.Items[i].SKU < c.Items[j].SKU })
			c.Revision++
			data, e := json.Marshal(c)
			if e != nil {
				return e
			}
			_, e = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error { pipe.Set(ctx, key, data, 30*24*time.Hour); return nil })
			result = c
			return e
		}, key)
		if errors.Is(e, redis.TxFailedErr) {
			continue
		}
		return result, e
	}
	return result, domain.E("VERSION_CONFLICT", 409)
}
func (s *Service) Address(ctx context.Context, p domain.Principal, id string) (domain.Address, int64, error) {
	if p.Role != "CUSTOMER" {
		return domain.Address{}, 0, domain.E("ADDRESS_NOT_FOUND", 404)
	}
	if _, e := uuid.Parse(id); e != nil {
		return domain.Address{}, 0, domain.E("INVALID_ADDRESS_ID", 400)
	}
	if s.C.Profile != "" {
		r, e := http.NewRequestWithContext(ctx, "GET", s.C.Profile+"/api/v1/profile/addresses/"+id, nil)
		if e != nil {
			return domain.Address{}, 0, e
		}
		r.Header.Set("X-Internal-Token", s.C.ProfileToken)
		r.Header.Set("X-User-ID", p.ID)
		r.Header.Set("X-User-Role", "CUSTOMER")
		resp, e := s.Client.Do(r)
		if e != nil {
			return domain.Address{}, 0, domain.E("PROFILE_UNAVAILABLE", 503)
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return domain.Address{}, 0, domain.E("ADDRESS_NOT_FOUND", 404)
		}
		if resp.StatusCode != 200 {
			return domain.Address{}, 0, domain.E("PROFILE_UNAVAILABLE", 503)
		}
		var saved struct {
			ID         string `json:"id"`
			CustomerID string `json:"customer_id"`
			Name       string `json:"recipient_name"`
			Phone      string `json:"phone_number"`
			Street     string `json:"street_address"`
			Ward       string `json:"ward_code"`
			Province   string `json:"province_code"`
			Version    int64  `json:"version"`
		}
		if e = json.NewDecoder(resp.Body).Decode(&saved); e != nil {
			return domain.Address{}, 0, e
		}
		if saved.ID != id || saved.CustomerID != p.CustomerID || saved.Version <= 0 {
			return domain.Address{}, 0, domain.E("ADDRESS_NOT_FOUND", 404)
		}
		return domain.Address{Name: saved.Name, Phone: saved.Phone, Street: saved.Street, Ward: saved.Ward, Province: saved.Province}, saved.Version, nil
	}
	var out struct {
		Address domain.Address `json:"address"`
		Version int64          `json:"version"`
	}
	e := s.Call(ctx, "GET", "/profile/address/"+id+"?customer_id="+p.CustomerID, nil, &out)
	return out.Address, out.Version, e
}
func (s *Service) Canonical(ctx context.Context, q domain.Quote) (domain.Quote, error) {
	var catalog struct {
		SnapshotID string        `json:"snapshot_id"`
		ExpiresAt  time.Time     `json:"expires_at"`
		Items      []domain.Item `json:"items"`
	}
	if e := s.Call(ctx, "POST", "/catalog/validate", map[string]any{"items": q.Items}, &catalog); e != nil {
		return q, e
	}
	if catalog.SnapshotID == "" || !time.Now().Before(catalog.ExpiresAt) {
		return q, domain.E("CATALOG_INCONSISTENT", 503)
	}
	if len(catalog.Items) != len(q.Items) {
		return q, domain.E("CATALOG_INCONSISTENT", 503)
	}
	for i := range catalog.Items {
		if catalog.Items[i].SKU != q.Items[i].SKU || catalog.Items[i].Quantity != q.Items[i].Quantity {
			return q, domain.E("CATALOG_INCONSISTENT", 503)
		}
		if q.Items[i].UnitPrice != 0 && q.Items[i].UnitPrice != catalog.Items[i].UnitPrice {
			return q, domain.E("PRICE_CHANGED", 409)
		}
	}
	subtotal, e := domain.Total(catalog.Items)
	if e != nil {
		return q, e
	}
	var ship struct {
		Reference string    `json:"quote_reference"`
		Fee       int64     `json:"fee"`
		ExpiresAt time.Time `json:"expires_at"`
		COD       bool      `json:"cod_eligible"`
	}
	if e = s.Call(ctx, "POST", "/shipping/quote", map[string]any{"address": q.Address, "items": catalog.Items}, &ship); e != nil {
		return q, e
	}
	if ship.Fee < 0 || ship.Fee > domain.MaxMoney-subtotal || ship.ExpiresAt.Before(time.Now()) {
		return q, domain.E("INVALID_SHIPPING_QUOTE", 503)
	}
	if q.Final != 0 && q.Shipping != ship.Fee {
		return q, domain.E("PRICE_CHANGED", 409)
	}
	if q.Method == "COD" && !ship.COD {
		return q, domain.E("COD_NOT_AVAILABLE", 422)
	}
	q.CatalogSnapshotID = catalog.SnapshotID
	q.CatalogExpiresAt = catalog.ExpiresAt
	q.ShippingReference = ship.Reference
	q.Items = catalog.Items
	q.Subtotal = subtotal
	q.Shipping = ship.Fee
	q.Final = subtotal + ship.Fee
	q.ShippingExpiresAt = ship.ExpiresAt
	return q, nil
}
func (s *Service) CreateQuote(ctx context.Context, p domain.Principal, in domain.QuoteInput) (domain.Quote, error) {
	if p.Role != "CUSTOMER" && p.Role != "GUEST" {
		return domain.Quote{}, domain.E("PERMISSION_DENIED", 403)
	}
	if in.Voucher != "" {
		return domain.Quote{}, domain.E("VOUCHER_FEATURE_DISABLED", 422)
	}
	if in.Method != "VIETQR" && in.Method != "COD" {
		return domain.Quote{}, domain.E("INVALID_PAYMENT_METHOD", 422)
	}
	p, e := s.Resolve(ctx, p)
	if e != nil {
		return domain.Quote{}, e
	}
	cart, e := s.Cart(ctx, p)
	if e != nil {
		return domain.Quote{}, e
	}
	if cart.Revision != in.Revision {
		return domain.Quote{}, domain.E("VERSION_CONFLICT", 409)
	}
	q := domain.Quote{ID: uuid.NewString(), Scope: p.Scope(), Principal: p, Revision: cart.Revision, Method: in.Method, Items: cart.Items, CustomerID: p.CustomerID, AddressID: in.AddressID}
	if in.AddressID != "" {
		if in.Address != nil {
			return q, domain.E("AMBIGUOUS_ADDRESS", 400)
		}
		q.Address, q.AddressVersion, e = s.Address(ctx, p, in.AddressID)
	} else if in.Address != nil {
		q.Address = *in.Address
	} else {
		e = domain.E("ADDRESS_REQUIRED", 422)
	}
	if e != nil {
		return q, e
	}
	q.Address.Phone = strings.TrimSpace(q.Address.Phone)
	if strings.HasPrefix(q.Address.Phone, "0") {
		q.Address.Phone = "+84" + q.Address.Phone[1:]
	}
	if e = q.Address.Validate(); e != nil {
		return q, e
	}
	if in.AddressID == "" || s.C.Profile == "" {
		if e = s.Call(ctx, "POST", "/address/validate", q.Address, nil); e != nil {
			return q, e
		}
	}
	q, e = s.Canonical(ctx, q)
	if e != nil {
		return q, e
	}
	q.ExpiresAt = time.Now().UTC().Add(5 * time.Minute)
	data, e := s.Seal(domain.StoredQuote{Quote: q, Scope: p.Scope(), Principal: p})
	if e != nil {
		return q, e
	}
	if e = s.Redis.Set(ctx, "quote:"+q.ID, data, 5*time.Minute).Err(); e != nil {
		return q, domain.E("QUOTE_UNAVAILABLE", 503)
	}
	return q, nil
}
func (s *Service) Operation(ctx context.Context, id string, p domain.Principal) (domain.Operation, error) {
	if _, e := uuid.Parse(id); e != nil {
		return domain.Operation{}, domain.E("NOT_FOUND", 404)
	}
	var o domain.Operation
	var errCode *string
	var status *string
	e := s.DB.QueryRow(ctx, `SELECT c.id,c.status,c.order_id,c.error_code,c.accepted_at,o.status FROM checkout_operations c LEFT JOIN orders o ON o.id=c.order_id WHERE c.id=$1 AND c.scope=$2`, id, p.Scope()).Scan(&o.ID, &o.Status, &o.OrderID, &errCode, &o.AcceptedAt, &status)
	if errCode != nil {
		o.ErrorCode = *errCode
	}
	if status != nil {
		o.OrderStatus = *status
	}
	return o, e
}
func (s *Service) replay(ctx context.Context, p domain.Principal, key, hash string) (domain.Operation, int, error) {
	var id, old string
	var expired bool
	var failedHTTP *int
	e := s.DB.QueryRow(ctx, `SELECT i.operation_id,i.request_hash,clock_timestamp()>=i.response_expires_at,c.error_http FROM idempotency_records i JOIN checkout_operations c ON c.id=i.operation_id WHERE i.scope=$1 AND i.kind='CHECKOUT' AND i.key=$2`, p.Scope(), key).Scan(&id, &old, &expired, &failedHTTP)
	if e != nil {
		return domain.Operation{}, 0, e
	}
	if old != hash {
		return domain.Operation{}, 409, domain.E("IDEMPOTENCY_CONFLICT", 409)
	}
	o, e := s.Operation(ctx, id, p)
	if e != nil {
		return o, 0, e
	}
	if (o.Status == "SUCCEEDED" || o.Status == "FAILED") && expired {
		return o, 410, domain.E("IDEMPOTENCY_RESPONSE_EXPIRED", 410)
	}
	if o.Status == "SUCCEEDED" {
		return o, 201, nil
	}
	if o.Status == "FAILED" {
		status := 422
		if failedHTTP != nil {
			status = *failedHTTP
		}
		return o, status, nil
	}
	return o, 202, nil
}
func (s *Service) Accept(ctx context.Context, p domain.Principal, key string, in domain.CheckoutInput) (domain.Operation, int, error) {
	if p.Role != "CUSTOMER" && p.Role != "GUEST" {
		return domain.Operation{}, 403, domain.E("PERMISSION_DENIED", 403)
	}
	if len(key) < 16 || len(key) > 128 {
		return domain.Operation{}, 400, domain.E("INVALID_IDEMPOTENCY_KEY", 400)
	}
	hash := domain.Hash(in)
	if o, code, e := s.replay(ctx, p, key, hash); !errors.Is(e, pgx.ErrNoRows) {
		return o, code, e
	}
	if _, e := uuid.Parse(in.QuoteID); e != nil {
		return domain.Operation{}, 400, domain.E("INVALID_QUOTE", 400)
	}
	data, e := s.Redis.Get(ctx, "quote:"+in.QuoteID).Bytes()
	if errors.Is(e, redis.Nil) {
		return domain.Operation{}, 409, domain.E("QUOTE_EXPIRED", 409)
	}
	if e != nil {
		return domain.Operation{}, 503, domain.E("QUOTE_UNAVAILABLE", 503)
	}
	var stored domain.StoredQuote
	if e = s.Open(data, &stored); e != nil {
		return domain.Operation{}, 400, domain.E("INVALID_QUOTE", 400)
	}
	q := stored.Quote
	if stored.Scope != p.Scope() {
		return domain.Operation{}, 404, domain.E("QUOTE_NOT_FOUND", 404)
	}
	if q.Method != in.Method || q.Revision != in.Revision {
		return domain.Operation{}, 409, domain.E("QUOTE_INPUT_MISMATCH", 409)
	}
	cart, e := s.Cart(ctx, p)
	if e != nil {
		return domain.Operation{}, 503, e
	}
	if cart.Revision != in.Revision {
		return domain.Operation{}, 409, domain.E("VERSION_CONFLICT", 409)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return domain.Operation{}, 503, e
	}
	defer tx.Rollback(ctx)
	var now time.Time
	if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return domain.Operation{}, 503, e
	}
	if !now.Before(q.ExpiresAt) {
		return domain.Operation{}, 409, domain.E("QUOTE_EXPIRED", 409)
	}
	id, planned := uuid.NewString(), uuid.NewString()
	_, e = tx.Exec(ctx, `INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash,accepted_at) VALUES($1,$2,$3,$4,'ACCEPTED',$5,$6,$7)`, id, p.Scope(), p.ID, planned, data, hash, now)
	if e != nil {
		return domain.Operation{}, 503, e
	}
	tag, e := tx.Exec(ctx, `INSERT INTO idempotency_records(scope,kind,key,request_hash,operation_id) VALUES($1,'CHECKOUT',$2,$3,$4) ON CONFLICT DO NOTHING`, p.Scope(), key, hash, id)
	if e != nil {
		return domain.Operation{}, 503, e
	}
	if tag.RowsAffected() == 0 {
		_ = tx.Rollback(ctx)
		return s.replay(ctx, p, key, hash)
	}
	if e = s.Audit(ctx, tx, "", p.ID, "CHECKOUT_ACCEPTED", map[string]any{"operation_id": id}); e != nil {
		return domain.Operation{}, 503, e
	}
	if e = tx.Commit(ctx); e != nil {
		return domain.Operation{}, 503, domain.E("ACCEPTANCE_UNKNOWN", 503)
	}
	return domain.Operation{ID: id, Status: "ACCEPTED", OrderID: nil, AcceptedAt: now}, 202, nil
}
