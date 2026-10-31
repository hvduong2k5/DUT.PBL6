package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
)

func completeContent(now time.Time) domain.Content {
	end := "2026-12-31"
	stars := 4
	proof := "55555555-5555-4555-8555-555555555555"
	return domain.Content{ProductName: "Synthetic product", Producer: "Synthetic producer", WorkshopLocation: "Hue", Process: "Synthetic process", Ingredients: []domain.Ingredient{{Name: "Material", Origin: "Synthetic origin"}}, Evidence: []domain.Evidence{{ID: proof, Reference: "private/synthetic-proof", Kind: "CERTIFICATE", Description: "Synthetic OCOP proof"}}, Certificates: []domain.Certificate{{ID: "66666666-6666-4666-8666-666666666666", Type: "OCOP", Number: "SYNTHETIC-OCOP", Issuer: "Synthetic authority", Scope: "PRODUCT", ScopeID: createRequest().ProductID, IssuedOn: "2026-01-01", ValidUntil: &end, Stars: &stars, VerificationMethod: "MANUAL", VerifiedAt: now.Add(-time.Hour), EvidenceIDs: []string{proof}}}}
}
func reviewSetup(t *testing.T) (context.Context, *Profiles, domain.Profile, domain.DependencySnapshot) {
	t.Helper()
	ctx, pool := testutil.Database(t)
	store, _ := NewProfiles(pool, "sandbox")
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	store.Clock = func() time.Time { return now }
	request := createRequest()
	request.Content = completeContent(now)
	result, err := store.CreateDraft(ctx, draftMetadata(), request)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, store, decodeProfile(t, result), domain.DependencySnapshot{Binding: request.Binding, BatchCode: "SYNTHETIC-LOT", ProductionDate: "2026-10-01", ExpiryDate: "2027-01-01", BestBefore: time.Date(2027, 1, 1, 17, 0, 0, 0, time.UTC), CatalogVersion: "1", BatchVersion: "1", CheckedAt: now}
}
func reviewMeta(key string) domain.CommandMetadata {
	return domain.CommandMetadata{PrincipalID: "33333333-3333-4333-8333-333333333333", Key: key, RequestID: "review-request"}
}

func TestSubmitFreezeIndependentApproval(t *testing.T) {
	ctx, store, p, snapshot := reviewSetup(t)
	meta := draftMetadata()
	meta.Key = "submit-1"
	request := domain.RevisionCommand{ExpectedVersion: 1}
	submitted, err := store.SubmitRevision(ctx, meta, p.ID, 1, request, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	after := decodeProfile(t, submitted)
	if after.Version != 2 || after.Revisions[0].Status != "IN_REVIEW" || after.Revisions[0].DependencySnapshot.BatchVersion != "1" {
		t.Fatal("submit did not freeze snapshot")
	}
	replay, err := store.SubmitRevision(ctx, meta, p.ID, 1, request, domain.DependencySnapshot{})
	if err != nil || !replay.Replayed {
		t.Fatalf("submit replay: %v", err)
	}
	meta.Key = "self-approve"
	for _, action := range []string{domain.ActionApprove, domain.ActionReject} {
		if _, err = store.DecideRevision(ctx, meta, p.ID, 1, action, domain.RevisionCommand{ExpectedVersion: 2, Reason: "self decision"}); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("self decision: %v", err)
		}
	}
	approved, err := store.DecideRevision(ctx, reviewMeta("approve-1"), p.ID, 1, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Independent evidence checked"})
	if err != nil {
		t.Fatal(err)
	}
	after = decodeProfile(t, approved)
	if after.Version != 3 || after.Revisions[0].Status != "APPROVED" || after.PublicRevision != nil || after.EverPublished {
		t.Fatal("approval published data")
	}
	var decisions, events int
	if err = store.DB.QueryRow(ctx, `SELECT (SELECT count(*) FROM review_decisions),(SELECT count(*) FROM outbox_events)`).Scan(&decisions, &events); err != nil || decisions != 1 || events != 0 {
		t.Fatal("invalid approval side effects")
	}
	if _, err = store.DB.Exec(ctx, `UPDATE traceability_revisions SET content='{}' WHERE profile_id=$1`, p.ID); err == nil {
		t.Fatal("approved content mutable")
	}
}

func TestCloneInheritsWithdrawnContributors(t *testing.T) {
	ctx, store, p, _ := reviewSetup(t)
	if _, err := store.NewRevision(ctx, reviewMeta("no-reason"), p.ID, domain.NewRevisionRequest{ExpectedVersion: 1, FromRevision: 1}); err == nil {
		t.Fatal("working draft withdrawn without reason")
	}
	cloned, err := store.NewRevision(ctx, reviewMeta("clone-2"), p.ID, domain.NewRevisionRequest{ExpectedVersion: 1, FromRevision: 1, Reason: "Replace working draft"})
	if err != nil {
		t.Fatal(err)
	}
	current := decodeProfile(t, cloned)
	if current.Revisions[0].Status != "ARCHIVED" || current.Revisions[1].DependencySnapshot != nil || len(current.Revisions[1].Contributors) != 2 {
		t.Fatal("clone did not preserve history")
	}
	third := reviewMeta("clone-3")
	third.PrincipalID = "77777777-7777-4777-8777-777777777777"
	cloned, err = store.NewRevision(ctx, third, p.ID, domain.NewRevisionRequest{ExpectedVersion: 2, FromRevision: 1, Reason: "Clone old source while withdrawing new working copy"})
	if err != nil {
		t.Fatal(err)
	}
	current = decodeProfile(t, cloned)
	if len(current.Revisions[2].Contributors) != 3 || current.QRCode != p.QRCode {
		t.Fatal("withdrawn contributors lost or QR changed")
	}
	var source int
	var withdrawn string
	if err = store.DB.QueryRow(ctx, `SELECT (after_state->>'source_revision')::int,(after_state->'withdrawn_revisions')::text FROM audit_entries WHERE action='revision.create' AND revision_number=3`).Scan(&source, &withdrawn); err != nil || source != 1 || withdrawn != "[2]" {
		t.Fatal("clone/withdrawal lineage missing from immutable audit", err)
	}
	snapshot := domain.DependencySnapshot{Binding: p.Binding, BatchCode: "LOT", ProductionDate: "2026-01-01", ExpiryDate: "2027-01-01", BestBefore: store.Clock().Add(time.Hour), CatalogVersion: "1", BatchVersion: "1", CheckedAt: store.Clock()}
	third.Key = "submit-3"
	if _, err = store.SubmitRevision(ctx, third, p.ID, 3, domain.RevisionCommand{ExpectedVersion: 3}, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, actor := range current.Revisions[2].Contributors {
		meta := reviewMeta("approve-inherited")
		meta.PrincipalID = actor
		if _, err = store.DecideRevision(ctx, meta, p.ID, 3, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 4, Reason: "Attempt inherited bypass"}); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("inherited contributor approved: %v", err)
		}
	}
}

func TestConcurrentApproveRejectAndAtomicDecision(t *testing.T) {
	ctx, store, p, snapshot := reviewSetup(t)
	meta := draftMetadata()
	meta.Key = "submit"
	if _, err := store.SubmitRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 1}, snapshot); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, action := range []string{domain.ActionApprove, domain.ActionReject} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			_, err := store.DecideRevision(ctx, reviewMeta(action), p.ID, 1, action, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Independent decision"})
			results <- err
		}(action)
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
		t.Fatalf("decision race: %d %d", success, conflict)
	}
	var decisions, audits, keys int
	if err := store.DB.QueryRow(ctx, `SELECT (SELECT count(*) FROM review_decisions),(SELECT count(*) FROM audit_entries),(SELECT count(*) FROM idempotency_records)`).Scan(&decisions, &audits, &keys); err != nil || decisions != 1 || audits != 3 || keys != 3 {
		t.Fatalf("race atomicity: %d %d %d %v", decisions, audits, keys, err)
	}
}

func TestReviewAuditFailureAndExpiry(t *testing.T) {
	ctx, store, p, snapshot := reviewSetup(t)
	meta := draftMetadata()
	meta.Key = "submit"
	if _, err := store.SubmitRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 1}, snapshot); err != nil {
		t.Fatal(err)
	}
	originalID := store.ID
	var existingAudit string
	if err := store.DB.QueryRow(ctx, `SELECT id::text FROM audit_entries LIMIT 1`).Scan(&existingAudit); err != nil {
		t.Fatal(err)
	}
	calls := 0
	store.ID = func() (string, error) {
		calls++
		if calls == 2 {
			return existingAudit, nil
		}
		return originalID()
	}
	if _, err := store.DecideRevision(ctx, reviewMeta("audit-failure"), p.ID, 1, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Review complete"}); err == nil {
		t.Fatal("audit failure committed decision")
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.Version != 2 || current.Revisions[0].Status != "IN_REVIEW" {
		t.Fatal("partial approval persisted")
	}
	var count int
	if err = store.DB.QueryRow(ctx, `SELECT count(*) FROM review_decisions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial decision persisted")
	}
	store.ID = originalID
	store.Clock = func() time.Time { return time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC) }
	if _, err = store.DecideRevision(ctx, reviewMeta("expired-approve"), p.ID, 1, domain.ActionApprove, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Too late"}); err == nil {
		t.Fatal("expired certificate approved")
	}
	if _, err = store.DecideRevision(ctx, reviewMeta("expired-reject"), p.ID, 1, domain.ActionReject, domain.RevisionCommand{ExpectedVersion: 2, Reason: "Expired certificate"}); err != nil {
		t.Fatal("cannot reject expired certificate", err)
	}
}

func TestSubmitRejectsChangedOrStaleSource(t *testing.T) {
	ctx, store, p, snapshot := reviewSetup(t)
	meta := draftMetadata()
	meta.Key = "submit"
	bad := snapshot
	bad.BatchID = "wrong-batch"
	if _, err := store.SubmitRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 1}, bad); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
	bad = snapshot
	bad.CheckedAt = bad.CheckedAt.Add(-31 * time.Second)
	if _, err := store.SubmitRevision(ctx, meta, p.ID, 1, domain.RevisionCommand{ExpectedVersion: 1}, bad); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatal(err)
	}
	current, err := store.Get(ctx, p.ID)
	if err != nil || current.Version != 1 || current.Revisions[0].Status != "DRAFT" {
		t.Fatal("failed submission changed state")
	}
}
