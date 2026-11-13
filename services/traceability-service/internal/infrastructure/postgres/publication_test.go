package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func approvedSetup(t *testing.T) (context.Context, *Profiles, domain.Profile, domain.DependencySnapshot) {
	t.Helper()
	ctx, store, p, snapshot := reviewSetup(t)
	meta := draftMetadata()
	meta.Key = "submit"
	if _, err := store.SubmitRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 1}, snapshot); err != nil {
		t.Fatal(err)
	}
	result, err := store.DecideRevision(ctx, reviewMeta("approve"), p.ID, 1, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Independent approval"})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, store, decodeProfile(t, result), snapshot
}

func TestPublishSourceHashAndDurableEvent(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	meta := reviewMeta("publish")
	changed := snapshot
	changed.BatchVersion = "corrected-2"
	if _, err := store.PublishRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, changed); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("source change accepted: %v", err)
	}
	now := store.Clock().Add(10 * time.Second)
	store.Clock = func() time.Time { return now }
	snapshot.CheckedAt = now
	result, err := store.PublishRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	published := decodeProfile(t, result)
	if published.Version != 4 || published.PublicRevision == nil || *published.PublicRevision != 1 || published.Revisions[0].Status != "PUBLISHED" || published.QRCode != p.QRCode {
		t.Fatal("invalid publication state")
	}
	record, err := store.Public(ctx, p.QRCode)
	if err != nil || record.Mode != "sandbox" || record.VerificationMethod != "MANUAL" || record.FoodSafetyCert != nil {
		t.Fatalf("public: %+v %v", record, err)
	}
	data, _ := json.Marshal(record)
	for _, secret := range []string{"private/", "contributors", "created_by", "actor_id", "scope_id", "dependency_snapshot"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("public leaked %s", secret)
		}
	}
	var id, topic, key, status string
	var payload []byte
	var version int64
	err = store.DB.QueryRow(ctx, `SELECT id::text,topic,partition_key,status,payload,aggregate_version FROM outbox_events`).Scan(&id, &topic, &key, &status, &payload, &version)
	if err != nil || topic != "traceability.events.v1" || key != p.ID || status != "PENDING" || version != 4 {
		t.Fatalf("outbox: %s %s %s %d %v", topic, key, status, version, err)
	}
	if !strings.Contains(string(payload), id) || strings.Contains(string(payload), "private/") {
		t.Fatal("event identity/privacy invalid")
	}
	replay, err := store.PublishRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, domain.DependencySnapshot{})
	if err != nil || !replay.Replayed {
		t.Fatal("publish replay failed", err)
	}
	var count int
	if err = store.DB.QueryRow(ctx, `SELECT count(*) FROM outbox_events`).Scan(&count); err != nil || count != 1 {
		t.Fatal("replay emitted duplicate event")
	}
}

func TestPublishOutboxFailureRollsBackEverything(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	if _, err := store.DB.Exec(ctx, `CREATE FUNCTION fail_outbox_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic broker-independent outbox failure'; END $$; CREATE TRIGGER fail_outbox BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION fail_outbox_test()`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishRevision(ctx, reviewMeta("publish-failure"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err == nil {
		t.Fatal("outbox failure accepted")
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.Version != 3 || current.PublicRevision != nil || current.EverPublished || current.Revisions[0].Status != "APPROVED" {
		t.Fatal("partial publication persisted")
	}
	var audits, keys int
	if err = store.DB.QueryRow(ctx, `SELECT (SELECT count(*) FROM audit_entries),(SELECT count(*) FROM idempotency_records)`).Scan(&audits, &keys); err != nil || audits != 3 || keys != 3 {
		t.Fatal("outbox failure left audit/dedup")
	}
	if _, err = store.Public(ctx, p.QRCode); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("failed publish became visible")
	}
}

func TestConcurrentPublishAndImmutablePublicHistory(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, key := range []string{"publish-a", "publish-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, err := store.PublishRevision(ctx, reviewMeta(key), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot)
			results <- err
		}(key)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, domain.ErrConflict):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("publish race: %d %d", success, conflict)
	}
	clone, err := store.NewRevision(ctx, reviewMeta("new-revision"), p.ID, domain.NewRevisionRequest{ExpectedVersion: 4, FromRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	after := decodeProfile(t, clone)
	content := after.Revisions[1].Content
	content.ProductName = "Next public name"
	if _, err = store.EditDraft(ctx, reviewMeta("edit-next"), p.ID, 2, domain.EditDraftRequest{ExpectedVersion: 5, Content: content}); err != nil {
		t.Fatal(err)
	}
	before, err := store.Public(ctx, p.QRCode)
	if err != nil || before.ProductName == content.ProductName {
		t.Fatal("editing draft changed published snapshot")
	}
	if _, err = store.SubmitRevision(ctx, reviewMeta("submit-next"), p.ID, 2, domain.RevisionCommand{ExpectedVersion: 6}, snapshot); err != nil {
		t.Fatal(err)
	}
	independent := reviewMeta("approve-next")
	independent.PrincipalID = "77777777-7777-4777-8777-777777777777"
	if _, err = store.DecideRevision(ctx, independent, p.ID, 2, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 7, Reason: "Independent second review"}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.PublishRevision(ctx, reviewMeta("publish-next"), p.ID, 2, domain.RevisionCommand{ExpectedVersion: 8}, snapshot); err != nil {
		t.Fatal(err)
	}
	public, err := store.Public(ctx, p.QRCode)
	if err != nil || public.Revision != 2 || public.ProductName != content.ProductName {
		t.Fatal("new revision not published")
	}
	after, err = store.Get(ctx, p.ID)
	if err != nil || after.Revisions[0].Content.ProductName != before.ProductName || after.Revisions[0].Status != "ARCHIVED" {
		t.Fatal("old revision overwritten")
	}
	if _, err = store.SetStatus(ctx, reviewMeta("suspend"), p.ID, "SUSPENDED", domain.RevisionCommand{ExpectedVersion: 9, Reason: "Internal operator reason not distributed"}); err != nil {
		t.Fatal(err)
	}
	var gone *domain.GoneError
	if _, err = store.Public(ctx, p.QRCode); !errors.As(err, &gone) || gone.Reason != "PROFILE_SUSPENDED" {
		t.Fatal("suspended record public", err)
	}
	if _, err = store.NewRevision(ctx, reviewMeta("bypass"), p.ID, domain.NewRevisionRequest{ExpectedVersion: 10, FromRevision: 2}); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("new revision bypassed suspension")
	}
	if _, err = store.SetStatus(ctx, reviewMeta("archive"), p.ID, "ARCHIVED", domain.RevisionCommand{ExpectedVersion: 10, Reason: "Archive after suspension"}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Public(ctx, p.QRCode); !errors.As(err, &gone) || gone.Reason != "PROFILE_ARCHIVED" {
		t.Fatal(err)
	}
}

func TestPublicEligibilityWithoutWorker(t *testing.T) {
	ctx, store, p, snapshot := approvedSetup(t)
	if _, err := store.PublishRevision(ctx, reviewMeta("publish"), p.ID, 1, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err != nil {
		t.Fatal(err)
	}
	store.Clock = func() time.Time { return time.Date(2026, 12, 31, 16, 59, 59, 0, time.UTC) }
	if _, err := store.Public(ctx, p.QRCode); err != nil {
		t.Fatal("inclusive last day rejected", err)
	}
	store.Clock = func() time.Time { return time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC) }
	var gone *domain.GoneError
	if _, err := store.Public(ctx, p.QRCode); !errors.As(err, &gone) || gone.Reason != "CERTIFICATE_EXPIRED" {
		t.Fatal("read did not enforce expiry", err)
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.Status != "ACTIVE" || current.Version != 4 {
		t.Fatal("read silently changed root status")
	}
}
