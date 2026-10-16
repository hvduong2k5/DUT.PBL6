package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/sepay"
	"time"
)

// PaymentCheckout persists one attempt before returning any externally usable form.
// It never uses a return URL or client-supplied amount as payment evidence.
func (s *Service) PaymentCheckout(ctx context.Context, p domain.Principal, orderID string) (sepay.Checkout, error) {
	if !s.MVP() || s.SePay == nil {
		return sepay.Checkout{}, domain.E("SEPAY_DISABLED", 409)
	}
	if p.Role != "CUSTOMER" {
		return sepay.Checkout{}, domain.E("CUSTOMER_LOGIN_REQUIRED", 403)
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return sepay.Checkout{}, domain.E("ORDER_NOT_FOUND", 404)
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return sepay.Checkout{}, err
	}
	defer tx.Rollback(ctx)
	o, err := s.lockOrder(ctx, tx, orderID)
	if err != nil {
		return sepay.Checkout{}, err
	}
	if p.Scope() != o.Scope {
		return sepay.Checkout{}, domain.E("ORDER_NOT_FOUND", 404)
	}
	var eligible bool
	err = tx.QueryRow(ctx, "SELECT clock_timestamp()+interval '30 seconds'<$1", o.ExpiresAt).Scan(&eligible)
	if err != nil {
		return sepay.Checkout{}, err
	}
	if !eligible || o.Method != "VIETQR" || o.Status != "PENDING_PAYMENT" || o.Hold || o.Stock != "RESERVED" {
		return sepay.Checkout{}, domain.E("PAYMENT_NOT_AVAILABLE", 409)
	}
	var id, invoice, merchant string
	err = tx.QueryRow(ctx, "SELECT id,merchant_invoice,merchant_id FROM payment_attempts WHERE order_id=$1 AND provider='SEPAY' AND environment=$2", o.ID, s.C.SePay.Environment).Scan(&id, &invoice, &merchant)
	if err != nil {
		return sepay.Checkout{}, err
	}
	if merchant != s.C.SePay.MerchantID {
		return sepay.Checkout{}, domain.E("PAYMENT_MERCHANT_MISMATCH", 409)
	}
	result := s.C.SePay.Checkout(id, invoice, o.Final, o.ExpiresAt)
	return result, tx.Commit(ctx)
}

// ReceiveSePay only acknowledges once the authenticated payload is durable.
// A separate worker interprets the message; unknown/void events remain reviewable.
func (s *Service) ReceiveSePay(ctx context.Context, raw json.RawMessage) error {
	if !s.MVP() {
		return domain.E("SEPAY_DISABLED", 409)
	}
	var n struct {
		Timestamp int64  `json:"timestamp"`
		Type      string `json:"notification_type"`
	}
	if err := json.Unmarshal(raw, &n); err != nil || n.Timestamp <= 0 || n.Type == "" {
		return domain.E("INVALID_SEPAY_NOTIFICATION", 400)
	}
	sealed, err := s.Seal(raw)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, "INSERT INTO sepay_notifications(id,payload_hash,payload_cipher) VALUES($1,$2,$3) ON CONFLICT(payload_hash) DO NOTHING", uuid.NewString(), domain.Hash(raw), sealed)
	return err
}

func (s *Service) applySePayPayment(ctx context.Context, o sepay.Order, t sepay.Transaction) error {
	p, err := sepay.Normalize(o, t)
	if err != nil {
		return domain.E("SEPAY_REVIEW_REQUIRED", 422)
	}
	// Bind the provider resource to the immutable local attempt. No missing attempt
	// is silently created from an IPN. Unmatched money still becomes a receipt.
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var existing *string
	err = tx.QueryRow(ctx, "SELECT provider_order_id FROM payment_attempts WHERE provider='SEPAY' AND environment=$1 AND merchant_id=$2 AND merchant_invoice=$3 FOR UPDATE", s.C.SePay.Environment, s.C.SePay.MerchantID, p.Invoice).Scan(&existing)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		if _, err = tx.Exec(ctx, "UPDATE payment_attempts SET provider_order_id=COALESCE(provider_order_id,$1) WHERE provider='SEPAY' AND environment=$2 AND merchant_id=$3 AND merchant_invoice=$4", p.OrderID, s.C.SePay.Environment, s.C.SePay.MerchantID, p.Invoice); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	r := domain.Receipt{ProviderOrderID: p.OrderID, Provider: s.C.SePay.Provider(), TransactionID: p.TransactionID, Reference: p.Invoice, Receiver: s.C.SePay.MerchantID, Amount: p.Amount, Currency: "VND", Status: "SUCCEEDED", PaidAt: p.PaidAt}
	return s.RecordReceipt(ctx, r)
}

func (s *Service) WorkSePayNotification(ctx context.Context) (bool, error) {
	if !s.MVP() {
		return false, nil
	}
	var id string
	var sealed []byte
	err := s.DB.QueryRow(ctx, "SELECT id,payload_cipher FROM sepay_notifications WHERE status='PENDING' ORDER BY received_at LIMIT 1").Scan(&id, &sealed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var raw json.RawMessage
	if err = s.Open(sealed, &raw); err != nil {
		return true, err
	}
	var n sepay.Notification
	status, reason := "APPLIED", ""
	if err = json.Unmarshal(raw, &n); err != nil {
		status, reason = "MANUAL_REVIEW", "INVALID_PROVIDER_PAYLOAD"
		var lookup struct {
			Order struct {
				Invoice string `json:"order_invoice_number"`
			} `json:"order"`
		}
		if json.Unmarshal(raw, &lookup) == nil {
			n.Order.Invoice = lookup.Order.Invoice
		}
	} else if n.Type == "QUERY_CAPTURED_WITHOUT_TRANSACTION" {
		status, reason = "MANUAL_REVIEW", "PROVIDER_TRANSACTION_PROOF_MISSING"
	} else if n.Type != "ORDER_PAID" {
		status, reason = "MANUAL_REVIEW", "UNSUPPORTED_NOTIFICATION"
	} else {
		err = s.applySePayPayment(ctx, n.Order, n.Transaction)
		if err != nil {
			code, http := s.ErrorCode(err)
			if http >= 500 {
				return true, err
			}
			status, reason = "MANUAL_REVIEW", code
		}
	}
	if status == "MANUAL_REVIEW" && n.Order.Invoice != "" && n.Type != "QUERY_CAPTURED_WITHOUT_TRANSACTION" {
		// Adverse/unrecognized financial evidence must block new fulfillment
		// acceptance. This local hold does not claim an accepted remote task stopped.
		// A query with missing proof is only an observation: holding here would
		// incorrectly prevent a subsequent valid, on-time IPN from settling.
		if err = s.holdSePayReview(ctx, n.Order.Invoice, id, reason); err != nil {
			return true, err
		}
	}
	_, err = s.DB.Exec(ctx, "UPDATE sepay_notifications SET status=$2,reason=$3,applied_at=clock_timestamp() WHERE id=$1 AND status='PENDING'", id, status, reason)
	return true, err
}

func (s *Service) holdSePayReview(ctx context.Context, invoice, notificationID, reason string) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var id string
	e = tx.QueryRow(ctx, `SELECT order_id FROM payment_attempts WHERE merchant_invoice=$1 AND provider='SEPAY' AND environment=$2 AND merchant_id=$3`, invoice, s.C.SePay.Environment, s.C.SePay.MerchantID).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	o, e := s.lockOrder(ctx, tx, id)
	if e != nil {
		return e
	}
	if !o.Hold || o.Ready {
		o.Hold = true
		o.Ready = false
		if e = s.touch(ctx, tx, &o, "SEPAY_FINANCIAL_REVIEW"); e != nil {
			return e
		}
	}
	if e = s.Audit(ctx, tx, id, "SEPAY", "NOTIFICATION_REVIEW", map[string]string{"notification_id": notificationID, "reason": reason}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s *Service) SePayNotificationReview(ctx context.Context, p domain.Principal, id string) (map[string]any, error) {
	if !s.MVP() || p.Role != "ADMIN" {
		return nil, domain.E("PERMISSION_DENIED", 403)
	}
	if _, e := uuid.Parse(id); e != nil {
		return nil, domain.E("NOT_FOUND", 404)
	}
	var sealed []byte
	var status string
	var reason *string
	e := s.DB.QueryRow(ctx, "SELECT payload_cipher,status,reason FROM sepay_notifications WHERE id=$1", id).Scan(&sealed, &status, &reason)
	if e != nil {
		return nil, e
	}
	var payload json.RawMessage
	if e = s.Open(sealed, &payload); e != nil {
		return nil, e
	}
	return map[string]any{"notification_id": id, "status": status, "reason": reason, "payload": payload}, nil
}

// One page per minute, cycling the entire provider history, including expired
// local orders. Page progress advances only after every remote order is durable.
func (s *Service) ReconcileSePay(ctx context.Context) (err error) {
	if !s.MVP() || s.SePay == nil {
		return nil
	}
	// Finish before the two-minute lease expires, even with slow sandbox calls.
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	provider := s.C.SePay.Provider()
	if _, err = s.DB.Exec(ctx, "INSERT INTO sepay_scan(provider) VALUES($1) ON CONFLICT DO NOTHING", provider); err != nil {
		return err
	}
	var page int
	err = s.DB.QueryRow(ctx, `UPDATE sepay_scan SET lease_until=clock_timestamp()+interval '2 minutes',next_run=clock_timestamp()+interval '60 seconds' WHERE provider=$1 AND next_run<=clock_timestamp() AND (lease_until IS NULL OR lease_until<clock_timestamp()) RETURNING page`, provider).Scan(&page)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, _ = s.DB.Exec(cleanup, "UPDATE sepay_scan SET lease_until=NULL,error_code='QUERY_OR_RECORD_FAILED' WHERE provider=$1", provider)
		}
	}()
	orders, last, err := s.SePay.Orders(ctx, page)
	if err != nil {
		return err
	}
	for _, remote := range orders {
		if remote.OrderID == "" || remote.Invoice == "" {
			return domain.E("INVALID_SEPAY_QUERY", 503)
		}
		detail, e := s.SePay.Detail(ctx, remote.Invoice, remote.OrderID)
		if e != nil {
			return e
		}
		if detail.Status == "CAPTURED" && len(detail.Transactions) == 0 {
			var known bool
			if e = s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM receipts WHERE provider=$1 AND receiver=$2 AND reference=$3 AND provider_status='SUCCEEDED')", provider, s.C.SePay.MerchantID, detail.Invoice).Scan(&known); e != nil {
				return e
			}
			if !known {
				// Preserve an authenticated query observation for operator review;
				// never manufacture a transaction/receipt from aggregate CAPTURED.
				raw, e := json.Marshal(sepay.Notification{Timestamp: 1, Type: "QUERY_CAPTURED_WITHOUT_TRANSACTION", Order: detail})
				if e != nil {
					return e
				}
				if e = s.ReceiveSePay(ctx, raw); e != nil {
					return e
				}
			}
		}
		for _, t := range detail.Transactions {
			if t.Type != "PAYMENT" || t.Status != "APPROVED" {
				continue
			}
			orderOnly := detail
			orderOnly.Transactions = nil
			n := sepay.Notification{Timestamp: time.Now().Unix(), Type: "ORDER_PAID", Order: orderOnly, Transaction: t}
			// Exclude the changing send timestamp from recovery idempotency.
			n.Timestamp = 1
			raw, e := json.Marshal(n)
			if e != nil {
				return e
			}
			if e = s.ReceiveSePay(ctx, raw); e != nil {
				return e
			}
		}
	}
	next := page + 1
	if page >= last {
		next = 1
	}
	_, err = s.DB.Exec(ctx, "UPDATE sepay_scan SET page=$2,lease_until=NULL,last_success=clock_timestamp(),error_code=NULL WHERE provider=$1", provider, next)
	return err
}
