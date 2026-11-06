package persistence

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	d "github.com/omamx/fulfillment-service/internal/domain"
)

func (s *Store) ClaimCleanup(ctx context.Context) (f d.File, found bool, e error) {
	var id string
	e = s.DB.QueryRow(ctx, `WITH pick AS(SELECT file_id FROM fulfillment.media_files WHERE status IN ('CREATED','EXPIRED','REJECTED','DELETE_PENDING') AND attached_at IS NULL AND legal_hold=false AND (cleanup_lease_until IS NULL OR cleanup_lease_until<clock_timestamp()) AND session_expires_at<clock_timestamp()-interval '24 hours' ORDER BY session_expires_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE fulfillment.media_files f SET status='DELETE_PENDING',version=version+1,cleanup_lease_until=clock_timestamp()+interval '1 minute' FROM pick WHERE f.file_id=pick.file_id RETURNING f.file_id::text`).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return f, false, nil
	}
	if e != nil {
		return f, false, e
	}
	f, e = s.File(ctx, id)
	return f, e == nil, e
}
func (s *Store) Deleted(ctx context.Context, f d.File) error {
	_, e := s.DB.Exec(ctx, `UPDATE fulfillment.media_files SET status='DELETED',version=version+1 WHERE file_id=$1 AND version=$2 AND status='DELETE_PENDING' AND attached_at IS NULL AND legal_hold=false`, f.ID, f.Version)
	return e
}
