package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

const WarehouseMaintenanceActor = "00000000-0000-4000-8000-000000000003"

// BackfillWarehouse is a maintenance port, never an HTTP command. PostgreSQL
// independently restricts NULL -> verified warehouse to the schema owner.
// It deliberately does not update a submitted/public revision or its hash.
func (s *Profiles) BackfillWarehouse(ctx context.Context, id string, expected int64, snapshot domain.DependencySnapshot) (bool, error) {
	if !domain.ValidUUID(id) || !domain.ValidUUID(snapshot.WarehouseID) || expected < 1 {
		return false, domain.ErrInvalidCommand
	}
	now := s.Clock().UTC()
	if snapshot.CheckedAt.After(now) || now.Sub(snapshot.CheckedAt) > 30*time.Second || snapshot.BatchVersion == "" {
		return false, domain.ErrUnavailable
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var version int64
	err = tx.QueryRow(ctx, `SELECT version FROM traceability_profiles WHERE id=$1 FOR UPDATE`, id).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrNotFound
	}
	if err != nil {
		return false, err
	}
	profile, err := readProfile(ctx, tx, id)
	if err != nil {
		return false, err
	}
	if profile.Binding != snapshot.Binding {
		return false, domain.ErrConflict
	}
	if profile.WarehouseID != nil {
		if *profile.WarehouseID != snapshot.WarehouseID {
			return false, domain.ErrWarehouseConflict
		}
		return false, tx.Commit(ctx)
	}
	if version != expected {
		return false, domain.ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET warehouse_id=$2,version=version+1,updated_at=$3 WHERE id=$1`, id, snapshot.WarehouseID, now); err != nil {
		return false, err
	}
	requestID, err := domain.NewUUID()
	if err != nil {
		return false, err
	}
	before := map[string]any{"version": version, "warehouse_id": nil}
	after := map[string]any{"version": version + 1, "warehouse_id": snapshot.WarehouseID, "source_namespace": snapshot.SourceNamespace, "batch_id": snapshot.BatchID, "batch_version": snapshot.BatchVersion, "source_verified_at": snapshot.CheckedAt}
	if err = s.auditStates(ctx, tx, domain.CommandMetadata{PrincipalID: WarehouseMaintenanceActor, RequestID: requestID}, id, 0, "profile.warehouse_backfill", before, after, now, "Authoritative warehouse metadata backfill"); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}
