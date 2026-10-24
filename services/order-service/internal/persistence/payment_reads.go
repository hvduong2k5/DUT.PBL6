package persistence

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/sepay"
	"time"
)

type PaymentStatus struct {
	OrderID       string     `json:"order_id"`
	IntentID      string     `json:"intent_id"`
	Provider      string     `json:"provider"`
	Method        string     `json:"method"`
	OrderStatus   string     `json:"order_status"`
	PaymentStatus string     `json:"payment_status"`
	Display       string     `json:"display_state"`
	IntentStatus  string     `json:"intent_status"`
	Expected      int64      `json:"expected_amount"`
	Allocated     int64      `json:"allocated_amount"`
	Refunded      int64      `json:"refunded_amount"`
	Currency      string     `json:"currency"`
	Expires       time.Time  `json:"expires_at"`
	Now           time.Time  `json:"server_time"`
	Review        bool       `json:"has_open_reconciliation"`
	CanPay        bool       `json:"can_pay"`
	Version       int64      `json:"version"`
	Started       *time.Time `json:"verification_started_at"`
	Due           *time.Time `json:"verification_due_at"`
	Support       *string    `json:"support_reference"`
	Reason        *string    `json:"support_reason"`
	Poll          int        `json:"next_poll_after_seconds"`
}

func (p *Payment) Status(ctx context.Context, id, scope string) (out PaymentStatus, err error) {
	if !validID(id) {
		return out, ErrInput
	}
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var e error
		out, e = p.statusLocked(ctx, tx, id, scope)
		return e
	})
	return
}

func (p *Payment) statusLocked(ctx context.Context, tx pgx.Tx, id, scope string) (out PaymentStatus, err error) {
	err = func() error {
		// Read authorization and current projection from the same root lock/snapshot.
		var exists string
		if e := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 AND scope=$2 AND cohort=$3 FOR SHARE", id, scope, Cohort).Scan(&exists); e != nil {
			return e
		}
		a, e := p.Store.loadOrder(ctx, tx, id, scope)
		if e != nil {
			return e
		}
		if a.Snapshot().Data().Payment.Method != "VIETQR" {
			return ErrInput
		}
		e = tx.QueryRow(ctx, `SELECT i.id,i.status,i.amount,i.expires_at,o.payment_status,clock_timestamp(),COALESCE(pr.has_open_reconciliation,false),COALESCE(pr.version,1),pr.verification_started_at,pr.verification_due_at,pr.support_reference,pr.support_reason,COALESCE((SELECT amount FROM allocations WHERE order_id=o.id),0),COALESCE((SELECT sum(f.amount) FROM refunds f JOIN allocations a ON a.receipt_id=f.receipt_id WHERE a.order_id=o.id AND f.status='SUCCEEDED'),0) FROM orders o JOIN payment_intents i ON i.order_id=o.id LEFT JOIN payment_order_projections pr ON pr.order_id=o.id WHERE o.id=$1 AND i.connection_id=$2 AND i.receiver_profile_id=$3`, id, p.Connection, p.Receiver).Scan(&out.IntentID, &out.IntentStatus, &out.Expected, &out.Expires, &out.PaymentStatus, &out.Now, &out.Review, &out.Version, &out.Started, &out.Due, &out.Support, &out.Reason, &out.Allocated, &out.Refunded)
		if e != nil {
			return e
		}
		out.OrderID = id
		out.Provider = "SEPAY"
		out.Method = "VIETQR"
		out.OrderStatus = string(a.State())
		out.Currency = "VND"
		out.Display = "WAITING_PAYMENT"
		out.Poll = 3
		switch {
		case out.Refunded > 0 && out.Refunded == out.Allocated:
			out.Display = "REFUNDED"
		case out.Refunded > 0:
			out.Display = "PARTIALLY_REFUNDED"
		case a.State() == core.Paid:
			out.Display = "PAID"
			out.Poll = 0
		case a.State() == core.PaymentFinalizing:
			out.Display = "FINALIZING"
			out.Poll = 3
		case a.State().Terminal() || !out.Now.Before(out.Expires):
			out.Display = "EXPIRED"
			out.Poll = 0
		case out.Review:
			out.Display = "RECONCILIATION_REQUIRED"
			out.Poll = 0
		case out.Started != nil:
			out.Display = "VERIFYING"
			out.Poll = 3
			if out.Due != nil && !out.Now.Before(*out.Due) {
				out.Display = "VERIFICATION_DELAYED"
				out.Poll = 10
				reason := "VERIFICATION_TIMEOUT"
				out.Reason = &reason
			}
		}
		var healthy bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM payment_sync_checkpoints WHERE receiver_id=$1 AND stream='FAST' AND circuit='HEALTHY' AND last_scanned_end>clock_timestamp()-interval '60 seconds')`, p.Receiver).Scan(&healthy); e != nil {
			return e
		}
		out.CanPay = healthy && a.State() == core.PendingPayment && a.Record().Blocks == 0 && out.IntentStatus == "ACTIVE" && out.Now.Add(30*time.Second).Before(out.Expires) && !out.Review && out.Started == nil
		if out.Reason != nil {
			switch *out.Reason {
			case "UNDERPAYMENT", "OVERPAYMENT", "EXTRA_RECEIPT", "LATE_PAYMENT", "MULTIPLE_PARTIAL_PAYMENTS", "VERIFICATION_TIMEOUT":
			default:
				r := "OTHER_REVIEW"
				out.Reason = &r
			}
		}
		return nil
	}()
	return
}

type Instructions struct {
	OrderID  string            `json:"order_id"`
	IntentID string            `json:"intent_id"`
	Provider string            `json:"provider"`
	Method   string            `json:"method"`
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"`
	Code     string            `json:"payment_code"`
	Receiver map[string]string `json:"receiver"`
	Content  string            `json:"transfer_content"`
	QR       string            `json:"qr_url"`
	Expires  time.Time         `json:"expires_at"`
	Now      time.Time         `json:"server_time"`
	CanPay   bool              `json:"can_pay"`
	Version  int64             `json:"version"`
}

func (p *Payment) Instructions(ctx context.Context, id, scope string) (out Instructions, err error) {
	if !validID(id) {
		return out, ErrInput
	}
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		status, e := p.statusLocked(ctx, tx, id, scope)
		if e != nil {
			return e
		}
		if !status.CanPay {
			return ErrExpired
		}
		var code string
		var raw []byte
		e = tx.QueryRow(ctx, `SELECT i.payment_code,r.config_cipher FROM payment_intents i JOIN payment_receiver_profiles r ON r.id=i.receiver_profile_id JOIN orders o ON o.id=i.order_id WHERE i.order_id=$1 AND o.scope=$2 AND r.id=$3`, id, scope, p.Receiver).Scan(&code, &raw)
		if e != nil {
			return e
		}
		data, e := p.Store.cipher.Open("receiver:"+p.Receiver, raw)
		if e != nil {
			return e
		}
		var r sepay.Receiver
		if json.Unmarshal(data, &r) != nil {
			return ErrIntegrity
		}
		qr, e := r.QR(code, status.Expected)
		if e != nil {
			return e
		}
		out = Instructions{id, status.IntentID, "SEPAY", "VIETQR", status.Expected, "VND", code, map[string]string{"bank_code": r.BankCode, "bank_name": r.BankName, "account_number": r.Account, "account_holder": r.Holder, "mode": r.Mode}, code, qr, status.Expires, status.Now, true, status.Version}
		return nil
	})
	return
}
