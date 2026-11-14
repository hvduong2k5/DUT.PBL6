package persistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
)

// A rejected voucher after funds/stock commitment needs reconciliation; it must
// never silently turn into a full-price order or authorize fulfillment.
func (p *Payment) parkVoucherFinalization(ctx context.Context, j PaymentJob, a core.Aggregate) error {
	return p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		if e := p.fence(ctx, tx, j); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "INSERT INTO reconciliation_cases(id,receipt_id,reason) VALUES($1,$2,'VOUCHER_FINALIZATION_REJECTED') ON CONFLICT DO NOTHING", uuid.NewString(), a.Record().Allocation.ReceiptID); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE payment_order_projections SET has_open_reconciliation=true,payment_hold=true,version=version+1 WHERE order_id=$1", a.ID()); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, "UPDATE jobs SET status='PARKED',error_code='VOUCHER_FINALIZATION_REJECTED',lease_owner=NULL,lease_until=NULL WHERE id=$1", j.ID)
		return e
	})
}
