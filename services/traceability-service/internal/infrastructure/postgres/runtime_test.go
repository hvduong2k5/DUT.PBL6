package postgres

import (
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRuntimeLOGINAndSchemaReadiness(t *testing.T) {
	ctx, dsn, admin := testutil.RuntimeDatabase(t)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("runtime login failed")
	}
	t.Cleanup(pool.Close)
	if err = CheckRuntime(ctx, pool, "sandbox"); err != nil {
		t.Fatal(err)
	}
	if err = CheckRuntime(ctx, pool, "integration"); err == nil {
		t.Fatal("mode mismatch accepted")
	}
	if err = CheckRuntime(ctx, admin, "sandbox"); err == nil {
		t.Fatal("migrator accepted as runtime")
	}
	if err = CheckRuntime(ctx, nil, "sandbox"); err == nil {
		t.Fatal("missing database accepted")
	}
	_, err = admin.Exec(ctx, `UPDATE schema_migrations SET checksum='tampered'`)
	if err != nil {
		t.Fatal(err)
	}
	if err = CheckRuntime(ctx, pool, "sandbox"); err == nil {
		t.Fatal("schema corruption accepted")
	}
}

func TestRuntimeLOGINCanPublishAndObserveExpiry(t *testing.T) {
	ctx, dsn, admin := testutil.RuntimeDatabase(t)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("runtime connection failed")
	}
	t.Cleanup(pool.Close)
	store, _ := NewProfiles(pool, "sandbox")
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	store.Clock = func() time.Time { return now }
	request := createRequest()
	request.Content = completeContent(now)
	created, err := store.CreateDraft(ctx, draftMetadata(), request)
	if err != nil {
		t.Fatal(err)
	}
	p := decodeProfile(t, created)
	snapshot := domain.DependencySnapshot{Binding: request.Binding, BatchCode: "SYNTHETIC-LOT", ProductionDate: "2026-10-01", ExpiryDate: "2027-01-01", BestBefore: time.Date(2027, 1, 1, 17, 0, 0, 0, time.UTC), CatalogVersion: "1", BatchVersion: "1", CheckedAt: now}
	if _, err = store.SubmitRevision(ctx, reviewMeta("submit"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 1}, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DecideRevision(ctx, reviewMeta("approve"), p.ID, 1, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Independent review"}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.PublishRevision(ctx, reviewMeta("publish"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err != nil {
		t.Fatal(err)
	}
	store.Clock = func() time.Time { return time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC) }
	if _, count, err := store.SweepExpiry(ctx, "", 10); err != nil || count != 1 {
		t.Fatal("runtime expiry privileges insufficient", err)
	}
	var events int
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM outbox_events`).Scan(&events); err != nil || events != 2 {
		t.Fatal("runtime workflow events missing")
	}
}
