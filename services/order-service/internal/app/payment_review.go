package app

import (
	"context"
	"github.com/omamx/order-service/internal/domain"
)

// Bounded operational queues. Full bank/payee details are only returned by the
// authorized refund detail endpoint, never in a list or a public Order.
func (s *Service) PaymentReview(ctx context.Context, p domain.Principal) (map[string]any, error) {
	if e := s.manualAccess(p); e != nil {
		return nil, e
	}
	rows, e := s.DB.Query(ctx, `SELECT r.id,r.order_id,r.amount,COALESCE(c.reason,''),COALESCE(c.status,''),r.created_at FROM receipts r LEFT JOIN orders o ON o.id=r.order_id LEFT JOIN reconciliation_cases c ON c.receipt_id=r.id WHERE ($1='ADMIN' OR o.warehouse_id=$2) ORDER BY (c.status='OPEN') DESC NULLS LAST,r.created_at,r.id LIMIT 100`, p.Role, p.Warehouse)
	if e != nil {
		return nil, e
	}
	items := []map[string]any{}
	for rows.Next() {
		var id, reason, state string
		var orderID *string
		var amount int64
		var created any
		if e = rows.Scan(&id, &orderID, &amount, &reason, &state, &created); e != nil {
			rows.Close()
			return nil, e
		}
		items = append(items, map[string]any{"receipt_id": id, "order_id": orderID, "amount": amount, "reason": reason, "reconciliation_status": state, "created_at": created})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	refundRows, e := s.DB.Query(ctx, `SELECT f.id FROM manual_refunds f LEFT JOIN orders o ON o.id=f.order_id WHERE ($1='ADMIN' OR o.warehouse_id=$2) AND f.status<>'SUCCEEDED' ORDER BY f.created_at,f.id LIMIT 100`, p.Role, p.Warehouse)
	if e != nil {
		return nil, e
	}
	refunds := []string{}
	for refundRows.Next() {
		var id string
		if e = refundRows.Scan(&id); e != nil {
			refundRows.Close()
			return nil, e
		}
		refunds = append(refunds, id)
	}
	e = refundRows.Err()
	refundRows.Close()
	if e != nil {
		return nil, e
	}
	result := map[string]any{"receipts": items, "pending_manual_refund_ids": refunds, "limit": 100}
	cancelRows, e := s.DB.Query(ctx, `SELECT c.id FROM cancellation_requests c JOIN orders o ON o.id=c.order_id WHERE c.status='PENDING_REVIEW' AND ($1='ADMIN' OR o.warehouse_id=$2) ORDER BY c.created_at,c.id LIMIT 100`, p.Role, p.Warehouse)
	if e != nil {
		return nil, e
	}
	cancels := []string{}
	for cancelRows.Next() {
		var id string
		if e = cancelRows.Scan(&id); e != nil {
			cancelRows.Close()
			return nil, e
		}
		cancels = append(cancels, id)
	}
	e = cancelRows.Err()
	cancelRows.Close()
	if e != nil {
		return nil, e
	}
	result["pending_cancellation_ids"] = cancels
	if p.Role == "ADMIN" {
		var pending, review int64
		if e = s.DB.QueryRow(ctx, "SELECT count(*) FILTER(WHERE status='PENDING'),count(*) FILTER(WHERE status='MANUAL_REVIEW') FROM sepay_notifications").Scan(&pending, &review); e != nil {
			return nil, e
		}
		result["notification_pending"] = pending
		result["notification_manual_review"] = review
		notifications, e := s.DB.Query(ctx, "SELECT id FROM sepay_notifications WHERE status='MANUAL_REVIEW' ORDER BY received_at,id LIMIT 100")
		if e != nil {
			return nil, e
		}
		ids := []string{}
		for notifications.Next() {
			var id string
			if e = notifications.Scan(&id); e != nil {
				notifications.Close()
				return nil, e
			}
			ids = append(ids, id)
		}
		e = notifications.Err()
		notifications.Close()
		if e != nil {
			return nil, e
		}
		result["notification_review_ids"] = ids
	}
	return result, nil
}
