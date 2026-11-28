package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWarehouseBackfillPreservesHistoryAndRejectsRuntime(t *testing.T) {
	ctx, dsn, admin := testutil.RuntimeDatabase(t)
	owner, _ := NewProfiles(admin, "sandbox")
	meta := draftMetadata()
	created, err := owner.CreateDraft(ctx, meta, createRequest())
	if err != nil {
		t.Fatal(err)
	}
	p := decodeProfile(t, created)
	const warehouse = "77777777-7777-4777-8777-777777777777"
	snapshot := domain.DependencySnapshot{Binding: p.Binding, WarehouseID: warehouse, BatchVersion: "2", CheckedAt: time.Now().UTC()}
	runtimePool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer runtimePool.Close()
	runtime, _ := NewProfiles(runtimePool, "sandbox")
	if _, err = runtime.BackfillWarehouse(ctx, p.ID, 1, snapshot); err == nil {
		t.Fatal("runtime login changed warehouse ownership")
	}
	var before []byte
	if err = admin.QueryRow(ctx, `SELECT to_jsonb(r) FROM traceability_revisions r WHERE profile_id=$1`, p.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	changed, err := owner.BackfillWarehouse(ctx, p.ID, 1, snapshot)
	if err != nil || !changed {
		t.Fatal("backfill failed", err)
	}
	if changed, err = owner.BackfillWarehouse(ctx, p.ID, 1, snapshot); err != nil || changed {
		t.Fatal("idempotent backfill mutated twice", err)
	}
	after, err := owner.Get(ctx, p.ID)
	if err != nil || after.Version != 2 || domain.Warehouse(after) != warehouse || after.QRCode != p.QRCode {
		t.Fatal("backfill identity or version", err)
	}
	var afterJSON []byte
	var audits int
	if err = admin.QueryRow(ctx, `SELECT to_jsonb(r) FROM traceability_revisions r WHERE profile_id=$1`, p.ID).Scan(&afterJSON); err != nil || string(before) != string(afterJSON) {
		t.Fatal("backfill rewrote revision history", err)
	}
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM audit_entries WHERE profile_id=$1 AND action='profile.warehouse_backfill' AND after_state->>'batch_version'='2'`, p.ID).Scan(&audits); err != nil || audits != 1 {
		t.Fatal("backfill audit missing", err)
	}
	if _, err = runtimePool.Exec(ctx, `UPDATE traceability_profiles SET warehouse_id='88888888-8888-4888-8888-888888888888',version=version+1 WHERE id=$1`, p.ID); err == nil {
		t.Fatal("runtime moved warehouse")
	}
	if _, err = admin.Exec(ctx, `UPDATE traceability_profiles SET warehouse_id='88888888-8888-4888-8888-888888888888',version=version+1 WHERE id=$1`, p.ID); err == nil {
		t.Fatal("owner moved an already bound warehouse")
	}
	conn, err := admin.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if err = migrations.VerifyWarehouseWrites(ctx, conn.Conn()); err == nil {
		t.Fatal("scope readiness accepted unenforced writers")
	}
	if err = migrations.RequireWarehouseWrites(ctx, conn.Conn()); err != nil {
		t.Fatal(err)
	}
	if err = migrations.VerifyWarehouseWrites(ctx, conn.Conn()); err != nil {
		t.Fatal(err)
	}
	request := createRequest()
	request.BatchID = "other-batch"
	meta.Key = "old-writer"
	if _, err = runtime.CreateDraft(context.Background(), meta, request); err == nil {
		t.Fatal("old writer created an unbound profile")
	}
	if _, err = admin.Exec(ctx, `UPDATE service_metadata SET warehouse_writes_required=false`); err == nil {
		t.Fatal("rollback disabled writer guard")
	}
}

func TestWarehouseBackfillKeepsLegacyPublicSnapshot(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	published, err := store.PublishRevision(ctx, reviewMeta("legacy-publish"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	p = decodeProfile(t, published)
	before, err := p.Revisions[0].DependencySnapshot.SourceHash()
	if err != nil {
		t.Fatal(err)
	}
	snapshot.WarehouseID = "77777777-7777-4777-8777-777777777777"
	if changed, err := store.BackfillWarehouse(ctx, p.ID, 4, snapshot); err != nil || !changed {
		t.Fatal("legacy published metadata backfill", err)
	}
	after, err := store.Get(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := after.Revisions[0].DependencySnapshot.SourceHash()
	if err != nil || before != hash || after.Revisions[0].DependencySnapshot.WarehouseID != "" || after.QRCode != p.QRCode || after.PublicRevision == nil || *after.PublicRevision != 1 {
		t.Fatal("backfill rewrote public source")
	}
	if _, err = store.Public(ctx, p.QRCode); err != nil {
		t.Fatal("legacy public eligibility was lost", err)
	}
}
