package app

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"time"
)

// Explicit HTTP representation consumed by the pinned customer MVP contract.
// The canonical core representation and persisted snapshots stay unchanged.
const CustomerMediaType = "application/vnd.omama.customer-mvp+json"

func (s *Service) CustomerOperation(ctx context.Context, o domain.Operation, p domain.Principal) (any, error) {
	var stage string
	var updated time.Time
	if err := s.DB.QueryRow(ctx, "SELECT stage,updated_at FROM checkout_operations WHERE id=$1 AND scope=$2", o.ID, p.Scope()).Scan(&stage, &updated); err != nil {
		return nil, err
	}
	mapped := map[string]string{"ACCEPTED": "VALIDATING", "VALIDATE": "VALIDATING", "RESERVE": "ACQUIRING", "PLACE": "PLACING", "COMPENSATE": "COMPENSATING", "SUCCEEDED": "DONE", "FAILED": "DONE"}[stage]
	if o.Status == "SUCCEEDED" || o.Status == "FAILED" {
		mapped = "DONE"
	}
	if mapped == "" {
		return nil, domain.E("UNKNOWN_CHECKOUT_STAGE", 503)
	}
	return map[string]any{"operation_id": o.ID, "status": o.Status, "stage": mapped, "order_id": o.OrderID, "accepted_at": o.AcceptedAt, "updated_at": updated, "order_status": o.OrderStatus, "error_code": o.ErrorCode}, nil
}

func frozenCustomerQuote(t core.SnapshotInput) domain.Quote {
	expiry := t.Catalog.ValidUntil
	if t.Shipping.ValidUntil.Before(expiry) {
		expiry = t.Shipping.ValidUntil
	}
	q := domain.Quote{ID: t.Provenance.QuoteID, Revision: t.Provenance.CartRevision, Method: t.Payment.Method, CatalogSnapshotID: t.Catalog.SnapshotID, CatalogExpiresAt: t.Catalog.ValidUntil, ShippingReference: t.Shipping.QuoteReference, ShippingExpiresAt: t.Shipping.ValidUntil, ExpiresAt: expiry, Subtotal: t.Money.Subtotal, Shipping: t.Money.ShippingFee, Final: t.Money.FinalAmount, Address: domain.Address{Name: t.Address.RecipientName, Phone: t.Address.PhoneE164, Street: t.Address.Street, Ward: t.Address.WardCode, Province: t.Address.ProvinceCode}, Items: []domain.Item{}}
	for _, i := range t.Items {
		q.Items = append(q.Items, domain.Item{SKU: i.SKU, Name: i.Name, Quantity: int(i.Quantity), UnitPrice: i.UnitPrice, Weight: i.WeightGrams})
	}
	if t.Owner.CustomerID != nil {
		q.CustomerID = *t.Owner.CustomerID
	}
	if t.Address.Source.ProfileAddressID != nil {
		q.AddressID = *t.Address.Source.ProfileAddressID
	}
	if t.Address.Source.AddressVersion != nil {
		q.AddressVersion = *t.Address.Source.AddressVersion
	}
	return q
}

func (s *Service) CustomerOrder(o domain.Order, p domain.Principal, detail bool) any {
	raw, _ := json.Marshal(o)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	delete(out, "commercial_snapshot")
	// JSON decoding through any would round monetary integers. Populate all
	// numeric fields directly from the authoritative typed projection.
	out["subtotal"] = o.Subtotal
	out["shipping_fee"] = o.Shipping
	out["final_amount"] = o.Final
	out["version"] = o.Version
	out["ready_generation"] = o.Generation
	out["items_digest"] = string(o.ItemsDigest)
	actions := []string{"CONTACT_SUPPORT"}
	if p.Scope() == o.Scope && o.Status == "PENDING_PAYMENT" {
		actions = append(actions, "PAY", "CANCEL")
	}
	if o.Shipment != nil {
		actions = append(actions, "VIEW_TRACKING")
	}
	out["allowed_actions"] = actions
	out["compensation_status"] = "NOT_REQUIRED"
	out["cancellation_status"] = "NONE"
	if o.Status == "CANCELLED_BY_USER" || o.Status == "CANCELLED_TIMEOUT" || o.Status == "CANCELLED_BY_ADMIN" {
		out["cancellation_status"] = "SUCCEEDED"
		out["compensation_status"] = "PENDING"
		if o.Stock == "RELEASED" || o.Stock == "EXPIRED" {
			out["compensation_status"] = "SUCCEEDED"
		}
	}
	if detail && o.CoreSnapshot != nil {
		out["snapshot"] = frozenCustomerQuote(*o.CoreSnapshot)
	} else {
		delete(out, "snapshot")
	}
	return out
}

func (s *Service) CustomerCancellation(ctx context.Context, p domain.Principal, id, key string) (any, error) {
	o, err := s.GetOrder(ctx, id, p, false)
	if err != nil {
		return nil, err
	}
	state := "PENDING"
	if o.Status == "CANCELLED_BY_USER" {
		state = "SUCCEEDED"
	}
	compensation := "PENDING"
	if o.Stock == "RELEASED" || o.Stock == "EXPIRED" {
		compensation = "SUCCEEDED"
	}
	reference := uuid.NewSHA1(uuid.NameSpaceOID, []byte(p.Scope()+":CORE_CANCEL:"+id+":"+key)).String()
	return map[string]any{"cancellation_id": reference, "order_id": id, "status": state, "compensation_status": compensation, "version": o.Version}, nil
}
