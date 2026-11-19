package postgres

import "context"

const expiredHoldBatchSize = 500

// ExpireDueHolds advances expired HELD resources using PostgreSQL's clock.
// Row locks serialize this reaper with hold finalization and release commands.
func (r *VoucherRepository) ExpireDueHolds(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `WITH due AS (
		SELECT id
		FROM voucher_holds
		WHERE state='HELD' AND expires_at<=clock_timestamp()
		ORDER BY expires_at,id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	)
	UPDATE voucher_holds h
	SET state='EXPIRED',resource_version=h.resource_version+1,updated_at=clock_timestamp()
	FROM due
	WHERE h.id=due.id AND h.state='HELD'`, expiredHoldBatchSize)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
