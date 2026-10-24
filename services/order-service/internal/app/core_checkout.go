package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/redis/go-redis/v9"
	"os"
	"time"
)

type coreSource struct{ s *Service }

func (c coreSource) Revalidate(ctx context.Context, terms core.SnapshotInput) error {
	var out struct {
		Valid bool `json:"valid"`
	}
	e := c.s.Call(ctx, "POST", "/order-core/revalidate", terms, &out)
	if e != nil {
		return e
	}
	if !out.Valid {
		return domain.E("TERMS_CHANGED", 409)
	}
	return nil
}
func (s *Service) EnableCoreCheckout() error {
	if os.Getenv("ORDER_CORE_ENABLED") != "true" {
		return nil
	}
	if s.Payment == nil {
		return errors.New("ORDER_CORE_ENABLED requires the SePay TEST runtime")
	}
	inventory, ok := s.PaymentInventory.(persistence.CheckoutInventory)
	if !ok {
		return errors.New("target checkout requires Inventory3.2 reserve/commit/release capability")
	}
	if _, e := uuid.Parse(os.Getenv("ORDER_MERCHANT_ID")); e != nil {
		return errors.New("ORDER_MERCHANT_ID must be a stable UUID")
	}
	s.CoreCheckout = &persistence.CheckoutRuntime{Store: s.Payment.Store, Source: coreSource{s}, Inventory: inventory}
	return nil
}
func (s *Service) coreQuote(ctx context.Context, p domain.Principal, q domain.Quote) (domain.Quote, error) {
	var terms core.SnapshotInput
	if e := s.Call(ctx, "POST", "/order-core/terms", q, &terms); e != nil {
		return q, e
	}
	// Bind authority fields locally. Provider terms cannot select a different owner,
	// quote, payment receiver, or cart identity.
	owner := core.Owner{Kind: p.Role, PrincipalID: p.ID}
	if p.Role == "CUSTOMER" {
		customer := p.CustomerID
		owner.CustomerID = &customer
	} else {
		sid := p.ID
		owner.GuestSessionID = &sid
	}
	cartKey := "core-cart-id:" + p.Scope()
	cartID := uuid.NewString()
	if e := s.Redis.SetNX(ctx, cartKey, cartID, 30*24*time.Hour).Err(); e != nil {
		return q, e
	}
	cartID, e := s.Redis.Get(ctx, cartKey).Result()
	if e != nil {
		return q, e
	}
	lines := []core.SelectionLine{}
	for _, i := range q.Items {
		lines = append(lines, core.SelectionLine{SKU: i.SKU, Quantity: int64(i.Quantity)})
	}
	selection, e := core.HashSelection(lines)
	if e != nil {
		return q, domain.E("INVALID_SELECTION", 422)
	}
	manifest, _ := core.HashManifest(lines)
	now := time.Now().UTC()
	terms.SchemaVersion = "1.0"
	terms.SnapshotID = uuid.NewString()
	terms.OrderID = uuid.NewString()
	terms.OperationID = uuid.NewString()
	terms.Owner = owner
	terms.Channel = "D2C_WEB"
	terms.MerchantID = os.Getenv("ORDER_MERCHANT_ID")
	terms.PlacedAt = now
	terms.Provenance = core.Provenance{QuoteID: q.ID, CartID: cartID, CartRevision: q.Revision, SelectionDigest: selection, ItemsDigest: manifest, QuoteHash: core.PayloadHash(domain.Hash(q)), AcceptedAt: now, CanonicalizedAt: now}
	if terms.Payment.Method != q.Method {
		return q, domain.E("INVALID_PAYMENT_TERMS", 503)
	}
	if q.Method == "VIETQR" {
		provider := "SEPAY"
		receiver := s.Payment.Receiver
		version := int64(1)
		expires := now.Add(15 * time.Minute)
		terms.Payment = core.PaymentTerms{Method: q.Method, Provider: &provider, ReceiverProfileID: &receiver, ReceiverProfileVersion: &version, ExpiresAt: &expires}
	}
	if _, e = core.NewSnapshot(terms); e != nil {
		return q, domain.E("INVALID_CANONICAL_TERMS", 503)
	}
	if len(terms.Items) != len(q.Items) {
		return q, domain.E("INVALID_SELECTION", 503)
	}
	if terms.Address.PhoneE164 != q.Address.Phone || terms.Address.RecipientName != q.Address.Name || terms.Address.Street != q.Address.Street || terms.Address.WardCode != q.Address.Ward || terms.Address.ProvinceCode != q.Address.Province {
		return q, domain.E("ADDRESS_CHANGED", 409)
	}
	q.Items = []domain.Item{}
	for _, i := range terms.Items {
		q.Items = append(q.Items, domain.Item{SKU: i.SKU, Quantity: int(i.Quantity), UnitPrice: i.UnitPrice, Weight: i.WeightGrams, Name: i.Name})
	}
	q.Subtotal = terms.Money.Subtotal
	q.Shipping = terms.Money.ShippingFee
	q.Final = terms.Money.FinalAmount
	q.CatalogSnapshotID = terms.Catalog.SnapshotID
	q.CatalogExpiresAt = terms.Catalog.ValidUntil
	q.ShippingReference = terms.Shipping.QuoteReference
	q.ShippingExpiresAt = terms.Shipping.ValidUntil
	q.ExpiresAt = now.Add(5 * time.Minute)
	if terms.Catalog.ValidUntil.Before(q.ExpiresAt) {
		q.ExpiresAt = terms.Catalog.ValidUntil
	}
	if terms.Shipping.ValidUntil.Before(q.ExpiresAt) {
		q.ExpiresAt = terms.Shipping.ValidUntil
	}
	terms.Provenance.QuoteHash = core.PayloadHash(domain.Hash(q))
	stored := domain.StoredQuote{Quote: q, Scope: p.Scope(), Principal: p, Core: &terms}
	data, e := s.Seal(stored)
	if e != nil {
		return q, e
	}
	if e = s.Redis.Set(ctx, "quote:"+q.ID, data, time.Until(q.ExpiresAt)).Err(); e != nil {
		return q, e
	}
	return q, nil
}
func (s *Service) acceptCore(ctx context.Context, p domain.Principal, key string, in domain.CheckoutInput) (domain.Operation, int, error) {
	if p.Role != "CUSTOMER" && p.Role != "GUEST" {
		return domain.Operation{}, 403, domain.E("PERMISSION_DENIED", 403)
	}
	hash := domain.Hash(in)
	if prior, e := s.Payment.Store.LookupAcceptance(ctx, p.Scope(), key, hash); e == nil {
		return s.coreOperation(ctx, prior.OperationID, p)
	} else if !errors.Is(e, pgx.ErrNoRows) && !errors.Is(e, persistence.ErrLegacy) {
		return domain.Operation{}, 409, coreFailure(e)
	}
	data, e := s.Redis.Get(ctx, "quote:"+in.QuoteID).Bytes()
	if errors.Is(e, redis.Nil) {
		return domain.Operation{}, 409, domain.E("QUOTE_EXPIRED", 409)
	}
	if e != nil {
		return domain.Operation{}, 503, e
	}
	var stored domain.StoredQuote
	if e = s.Open(data, &stored); e != nil {
		return domain.Operation{}, 503, e
	}
	if stored.Scope != p.Scope() {
		return domain.Operation{}, 404, domain.E("QUOTE_NOT_FOUND", 404)
	}
	if stored.Core == nil || stored.Quote.Method != in.Method || stored.Quote.Revision != in.Revision {
		return domain.Operation{}, 409, domain.E("QUOTE_INPUT_MISMATCH", 409)
	}
	cart, e := s.Cart(ctx, p)
	if e != nil {
		return domain.Operation{}, 503, e
	}
	lines := []core.SelectionLine{}
	for _, i := range cart.Items {
		lines = append(lines, core.SelectionLine{SKU: i.SKU, Quantity: int64(i.Quantity)})
	}
	selection, e := core.HashSelection(lines)
	if e != nil || cart.Revision != in.Revision || selection != stored.Core.Provenance.SelectionDigest {
		return domain.Operation{}, 409, domain.E("CART_CHANGED", 409)
	}
	if p.Role == "GUEST" {
		if e = s.Payment.CheckGuestSession(ctx, p.ID); e != nil {
			return domain.Operation{}, 401, e
		}
	}
	raw, _ := json.Marshal(persistence.AcceptedCheckout{Terms: *stored.Core, QuoteExpires: stored.Quote.ExpiresAt})
	a, e := s.Payment.Store.Accept(ctx, persistence.AcceptanceInput{Owner: stored.Core.Owner, Method: in.Method, Key: key, RequestHash: hash, QuoteID: in.QuoteID, CartID: stored.Core.Provenance.CartID, CartRevision: in.Revision, SelectionDigest: selection, QuoteHash: stored.Core.Provenance.QuoteHash, QuoteExpiresAt: stored.Quote.ExpiresAt, AcceptedJSON: raw})
	if e != nil {
		return domain.Operation{}, 409, coreFailure(e)
	}
	return s.coreOperation(ctx, a.OperationID, p)
}
func coreFailure(e error) error {
	switch {
	case errors.Is(e, persistence.ErrQuoteUsed):
		return domain.E("QUOTE_ALREADY_USED", 409)
	case errors.Is(e, persistence.ErrSubmitted):
		return domain.E("CHECKOUT_ALREADY_SUBMITTED", 409)
	case errors.Is(e, persistence.ErrConflict):
		return domain.E("IDEMPOTENCY_CONFLICT", 409)
	case errors.Is(e, persistence.ErrInput):
		return domain.E("INVALID_REQUEST", 400)
	case errors.Is(e, persistence.ErrReplayExpired):
		return domain.E("IDEMPOTENCY_RESPONSE_EXPIRED", 410)
	case errors.Is(e, persistence.ErrExpired):
		return domain.E("QUOTE_EXPIRED", 409)
	}
	return e
}
func (s *Service) coreOperation(ctx context.Context, id string, p domain.Principal) (domain.Operation, int, error) {
	a, e := s.Payment.Store.ReadAcceptance(ctx, id, p.Scope())
	if e != nil {
		return domain.Operation{}, 404, e
	}
	out := domain.Operation{ID: a.OperationID, OrderID: a.OrderID, Status: a.Status, AcceptedAt: a.AcceptedAt}
	status := 202
	if a.Status == "SUCCEEDED" {
		status = 201
		if e = s.DB.QueryRow(ctx, "SELECT status FROM orders WHERE id=$1", a.OrderID).Scan(&out.OrderStatus); e != nil {
			return out, 503, e
		}
	}
	if a.Status == "FAILED" {
		status = 409
		_ = s.DB.QueryRow(ctx, "SELECT COALESCE(error_code,'CHECKOUT_FAILED') FROM checkout_operations WHERE id=$1", id).Scan(&out.ErrorCode)
	}
	return out, status, nil
}
