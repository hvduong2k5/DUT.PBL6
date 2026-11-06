package app

import (
	"context"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
)

func (s *Service) QuoteResponse(ctx context.Context, q domain.Quote, p domain.Principal) (any, error) {
	if s.CoreCheckout == nil {
		return q, nil
	}
	raw, e := s.Redis.Get(ctx, "quote:"+q.ID).Bytes()
	if e != nil {
		return nil, e
	}
	var stored domain.StoredQuote
	if e = s.Open(raw, &stored); e != nil {
		return nil, e
	}
	if stored.Core == nil || stored.Scope != p.Scope() {
		return nil, domain.E("QUOTE_NOT_FOUND", 404)
	}
	t := stored.Core
	items := []any{}
	selected := []string{}
	for _, i := range t.Items {
		selected = append(selected, i.SKU)
		items = append(items, map[string]any{"sku_code": i.SKU, "quantity": i.Quantity, "product_name": i.Name, "unit_price": i.UnitPrice, "line_total": i.LineSubtotal, "weight_grams": i.WeightGrams, "catalog_version": i.CatalogVersion})
	}
	return map[string]any{"quote_id": q.ID, "cart_id": t.Provenance.CartID, "cart_revision": q.Revision, "selected_sku_codes": selected, "selection_digest": t.Provenance.SelectionDigest, "quote_hash": t.Provenance.QuoteHash, "items": items, "payment_method": q.Method, "shipping_service_code": t.Shipping.ServiceCode, "subtotal": q.Subtotal, "shipping_fee": q.Shipping, "merchandise_discount": 0, "shipping_discount": 0, "final_amount": q.Final, "currency": "VND", "expires_at": q.ExpiresAt, "stock_reserved": false}, nil
}
func (s *Service) OperationResponse(ctx context.Context, o domain.Operation, p domain.Principal) (any, error) {
	if s.CoreCheckout == nil {
		return o, nil
	}
	var cohort, stage string
	var errorCode *string
	e := s.DB.QueryRow(ctx, "SELECT cohort,stage,error_code FROM checkout_operations WHERE id=$1 AND scope=$2", o.ID, p.Scope()).Scan(&cohort, &stage, &errorCode)
	if e != nil {
		return nil, e
	}
	if cohort != "order-core-1" {
		return o, nil
	}
	switch stage {
	case "SUCCEEDED":
		stage = "PLACE"
	case "FAILED":
		stage = "COMPENSATE"
	}
	cleanup := "NOT_SCHEDULED"
	if o.Status == "SUCCEEDED" {
		if e = s.DB.QueryRow(ctx, "SELECT CASE WHEN status='DONE' THEN COALESCE(payload->>'cleanup_status','APPLIED') WHEN status='PARKED' THEN 'MANUAL_REVIEW' ELSE 'PENDING' END FROM jobs WHERE order_id=$1 AND kind='CART_CLEANUP'", o.OrderID).Scan(&cleanup); e != nil {
			return nil, e
		}
	}
	return map[string]any{"operation_id": o.ID, "status": o.Status, "stage": stage, "order_id": o.OrderID, "accepted_at": o.AcceptedAt, "error_code": errorCode, "cart_cleanup_status": cleanup}, nil
}
func (s *Service) OrderResponse(o domain.Order, p domain.Principal, detail bool) any {
	if o.Cohort != "order-core-1" && o.ItemsDigest == "" {
		return o
	}
	actions := []string{}
	if o.Status == string(core.PendingPayment) && p.Scope() == o.Scope {
		actions = append(actions, "CANCEL_PENDING")
	}
	out := map[string]any{"order_id": o.ID, "order_code": o.Code, "status": o.Status, "payment_status": o.Payment, "stock_status": o.Stock, "payment_method": o.Method, "final_amount": o.Final, "currency": "VND", "version": o.Version, "created_at": o.CreatedAt, "sla_due_at": o.SLA, "blocking_hold": o.Hold, "cancellation_pending": false, "ready_generation": o.Generation, "ready_authorized": o.Ready, "allowed_actions": actions}
	if !detail {
		return out
	}
	t := o.CoreSnapshot
	if t == nil {
		return out
	}
	items := []any{}
	for _, i := range t.Items {
		items = append(items, map[string]any{"sku_code": i.SKU, "product_name": i.Name, "quantity": i.Quantity, "unit_price": i.UnitPrice, "line_subtotal": i.LineSubtotal})
	}
	recipient := map[string]any{"visibility": "MASKED", "recipient_name": nil, "phone": nil, "street": nil, "province_name": t.Address.ProvinceName, "ward_name": t.Address.WardName}
	if p.Scope() == o.Scope {
		recipient = map[string]any{"visibility": "FULL", "recipient_name": t.Address.RecipientName, "phone": t.Address.PhoneE164, "street": t.Address.Street, "province_name": t.Address.ProvinceName, "ward_name": t.Address.WardName}
	}
	out["snapshot"] = map[string]any{"snapshot_id": t.SnapshotID, "snapshot_schema_version": t.SchemaVersion, "items": items, "subtotal": t.Money.Subtotal, "shipping_fee": t.Money.ShippingFee, "final_amount": t.Money.FinalAmount, "currency": "VND", "shipping_service_code": t.Shipping.ServiceCode, "recipient": recipient}
	out["holds"] = []any{}
	out["compensation_status"] = "NOT_REQUIRED"
	if o.Status == "CANCELLED_BY_USER" || o.Status == "CANCELLED_TIMEOUT" || o.Status == "CANCELLED_BY_ADMIN" {
		out["compensation_status"] = "PENDING"
		if o.Stock == "RELEASED" || o.Stock == "EXPIRED" {
			out["compensation_status"] = "SUCCEEDED"
		}
	}
	out["payment_expires_at"] = o.ExpiresAt
	out["task_id"] = o.Task
	out["shipment_id"] = o.Shipment
	return out
}
