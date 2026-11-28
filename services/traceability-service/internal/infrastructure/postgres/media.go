package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// BindMediaJob leases one committed publication obligation. RPC calls never hold a DB transaction.
func (s *Profiles) BindMediaJob(ctx context.Context, bind func(context.Context, string, string) error) (bool, error) {
	var profile, file, token string
	err := s.DB.QueryRow(ctx, `UPDATE media_bind_jobs SET lease_token=gen_random_uuid(), lease_until=now()+interval '15 seconds', attempts=attempts+1
 WHERE (profile_id,revision,file_id)=(SELECT profile_id,revision,file_id FROM media_bind_jobs
 WHERE bound_at IS NULL AND retry_at<=now() AND (lease_until IS NULL OR lease_until<now())
 ORDER BY retry_at,profile_id,revision,file_id FOR UPDATE SKIP LOCKED LIMIT 1)
 RETURNING profile_id::text,file_id::text,lease_token::text`).Scan(&profile, &file, &token)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	callErr := bind(callCtx, profile, file)
	cancel()
	if callErr == nil {
		_, err = s.DB.Exec(ctx, `UPDATE media_bind_jobs SET bound_at=now(),lease_token=NULL,lease_until=NULL WHERE lease_token=$1 AND bound_at IS NULL`, token)
	} else {
		_, err = s.DB.Exec(ctx, `UPDATE media_bind_jobs SET retry_at=now()+interval '5 seconds',lease_token=NULL,lease_until=NULL WHERE lease_token=$1 AND bound_at IS NULL`, token)
	}
	return true, err
}
