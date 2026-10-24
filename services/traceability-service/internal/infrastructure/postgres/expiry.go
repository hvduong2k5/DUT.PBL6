package postgres

import (
	"context"
	"errors"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// SweepExpiry scans a bounded UUID page; the caller carries the cursor between
// ticks and wraps on an empty page, so observed first pages cannot starve later ones.
func (s *Profiles) SweepExpiry(ctx context.Context, after string, limit int) (string, int, error) {
	if (after != "" && !domain.ValidUUID(after)) || limit < 1 || limit > 100 {
		return "", 0, ErrInvalidCommand
	}
	var cursor any
	if after != "" {
		cursor = after
	}
	rows, err := s.DB.Query(ctx, `SELECT id::text FROM traceability_profiles WHERE status='ACTIVE' AND public_revision IS NOT NULL AND ($1::uuid IS NULL OR id>$1::uuid) ORDER BY id LIMIT $2`, cursor, limit)
	if err != nil {
		return after, 0, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return after, 0, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return after, 0, err
	}
	if len(ids) == 0 {
		return "", 0, nil
	}
	observed := 0
	for _, id := range ids {
		changed, err := s.observeExpiry(ctx, id)
		if err != nil {
			return after, observed, err
		}
		if changed {
			observed++
		}
		after = id
	}
	return after, observed, nil
}

func (s *Profiles) observeExpiry(ctx context.Context, id string) (bool, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var state string
	// If another command owns this profile, retry it on the next cursor cycle.
	err = tx.QueryRow(ctx, `SELECT status FROM traceability_profiles WHERE id=$1 FOR UPDATE SKIP LOCKED`, id).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if state != "ACTIVE" {
		return false, nil
	}
	p, err := readProfile(ctx, tx, id)
	if err != nil {
		return false, err
	}
	if p.PublicRevision == nil {
		return false, nil
	}
	revision, err := findRevision(&p, *p.PublicRevision)
	if err != nil {
		return false, err
	}
	now := s.Clock().UTC()
	reason, cutoff, due := revision.Content.ExpiryTransition(now)
	if !due {
		return false, nil
	}
	tag, err := tx.Exec(ctx, `INSERT INTO eligibility_observations(profile_id,public_revision,reason,observed_state,cutoff,observed_at)
 VALUES($1,$2,$3,'INELIGIBLE',$4,$5) ON CONFLICT DO NOTHING`, id, revision.Number, reason, cutoff, now)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET version=version+1,updated_at=$2 WHERE id=$1`, id, now); err != nil {
		return false, err
	}
	meta := domain.CommandMetadata{PrincipalID: "00000000-0000-4000-8000-000000000003", RequestID: "expiry-worker"}
	if err = s.audit(ctx, tx, meta, id, revision.Number, "certification.status.changed", p.Version, p.Version+1, now, reason); err != nil {
		return false, err
	}
	if err = s.event(ctx, tx, id, p.Version+1, revision.Number, "certification.status.changed", "ACTIVE", reason, now); err != nil {
		return false, err
	}
	// Root remains ACTIVE. A newly reviewed certificate revision can still publish.
	return true, tx.Commit(ctx)
}
