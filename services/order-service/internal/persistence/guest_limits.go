package persistence

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

var ErrGuestPendingLimit = errors.New("GUEST_PENDING_ORDER_LIMIT")

// Serialize different checkout keys for this guest, including accepted work
// which has not yet produced an order in the asynchronous placement worker.
func GuardGuestPending(ctx context.Context, tx pgx.Tx, sid string, limit int) error {
	var active bool
	if e := tx.QueryRow(ctx, `SELECT revoked_at IS NULL AND expires_at>clock_timestamp() FROM guest_checkout_sessions WHERE sid=$1 FOR UPDATE`, sid).Scan(&active); e != nil {
		return e
	}
	if !active {
		return ErrScope
	}
	var pending int
	e := tx.QueryRow(ctx, `SELECT count(*) FROM checkout_operations c LEFT JOIN orders o ON o.operation_id=c.id WHERE c.scope=$1 AND (
 (o.id IS NULL AND c.status NOT IN ('FAILED','SUCCEEDED')) OR
 (o.status IN ('PENDING_PAYMENT','PAYMENT_FINALIZING','CONFIRMED_COD') AND o.payment_status IN ('UNPAID','PENDING','RECONCILIATION_REQUIRED')))`, "guest:"+sid).Scan(&pending)
	if e != nil {
		return e
	}
	if pending >= limit {
		return ErrGuestPendingLimit
	}
	return nil
}
