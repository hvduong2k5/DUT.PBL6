package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
)

func createRequest() domain.CreateDraftRequest {
	return domain.CreateDraftRequest{Binding: domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SKU-1", SourceNamespace: "fixture", BatchID: "batch-1"}}
}
func draftMetadata() domain.CommandMetadata {
	return domain.CommandMetadata{PrincipalID: testActor, Key: "draft-1", RequestID: "test-request-1"}
}
func decodeProfile(t *testing.T, result domain.CommandResult) domain.Profile {
	t.Helper()
	var p domain.Profile
	if err := json.Unmarshal(result.Body, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDraftCreateReadEditReplay(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store, err := NewProfiles(pool, "sandbox")
	if err != nil {
		t.Fatal(err)
	}
	meta := draftMetadata()
	result, err := store.CreateDraft(ctx, meta, createRequest())
	if err != nil {
		t.Fatal(err)
	}
	p := decodeProfile(t, result)
	if result.Status != 201 || p.Version != 1 || p.Status != "ACTIVE" || p.PublicRevision != nil || p.EverPublished || len(p.Revisions) != 1 || p.Revisions[0].Status != "DRAFT" {
		t.Fatalf("invalid initial state: %+v", p)
	}
	raw, err := base64.RawURLEncoding.DecodeString(p.QRCode)
	if err != nil || len(raw) < 16 {
		t.Fatal("QR lacks 128 bits")
	}
	fetched, err := store.Get(ctx, p.ID)
	if err != nil || fetched.QRCode != p.QRCode {
		t.Fatalf("get: %+v %v", fetched, err)
	}
	replay, err := store.CreateDraft(ctx, meta, createRequest())
	if err != nil || !replay.Replayed || string(replay.Body) != string(result.Body) {
		t.Fatalf("create replay: %v", err)
	}
	meta.Key = "edit-1"
	meta.PrincipalID = "33333333-3333-4333-8333-333333333333"
	edit := domain.EditDraftRequest{ExpectedVersion: 1, Content: domain.Content{ProductName: "Updated draft"}}
	updated, err := store.EditDraft(ctx, meta, p.ID, 1, edit)
	if err != nil {
		t.Fatal(err)
	}
	after := decodeProfile(t, updated)
	if after.Version != 2 || after.QRCode != p.QRCode || after.Revisions[0].Content.ProductName != "Updated draft" || len(after.Revisions[0].Contributors) != 2 {
		t.Fatalf("edit state: %+v", after)
	}
	replay, err = store.EditDraft(ctx, meta, p.ID, 1, edit)
	if err != nil || !replay.Replayed || string(replay.Body) != string(updated.Body) {
		t.Fatalf("edit replay: %v", err)
	}
	edit.Content.ProductName = "Other body"
	if _, err = store.EditDraft(ctx, meta, p.ID, 1, edit); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("mismatch: %v", err)
	}
	meta.Key = "edit-stale"
	if _, err = store.EditDraft(ctx, meta, p.ID, 1, edit); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale version: %v", err)
	}
	var audits, keys int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM audit_entries),(SELECT count(*) FROM idempotency_records)`).Scan(&audits, &keys); err != nil || audits != 2 || keys != 2 {
		t.Fatalf("audit/dedup counts %d %d %v", audits, keys, err)
	}
}

func TestDraftConcurrentEditsAndBinding(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store, _ := NewProfiles(pool, "sandbox")
	created, err := store.CreateDraft(ctx, draftMetadata(), createRequest())
	if err != nil {
		t.Fatal(err)
	}
	p := decodeProfile(t, created)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, key := range []string{"edit-a", "edit-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			meta := draftMetadata()
			meta.Key = key
			_, err := store.EditDraft(ctx, meta, p.ID, 1, domain.EditDraftRequest{ExpectedVersion: 1, Content: domain.Content{ProductName: key}})
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
		t.Fatalf("concurrent edits success=%d conflict=%d", success, conflict)
	}
	meta := draftMetadata()
	meta.Key = "duplicate-binding"
	if _, err = store.CreateDraft(ctx, meta, createRequest()); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate binding: %v", err)
	}
	_, err = pool.Exec(ctx, `UPDATE traceability_revisions SET status='IN_REVIEW',dependency_snapshot='{}' WHERE profile_id=$1`, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta.Key = "frozen-edit"
	if _, err = store.EditDraft(ctx, meta, p.ID, 1, domain.EditDraftRequest{ExpectedVersion: 2}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("frozen edit: %v", err)
	}
}

func TestDraftQRCollisionAndAtomicAudit(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store, _ := NewProfiles(pool, "sandbox")
	first, err := store.CreateDraft(ctx, draftMetadata(), createRequest())
	if err != nil {
		t.Fatal(err)
	}
	p := decodeProfile(t, first)
	originalQR := store.QR
	attempts := 0
	store.QR = func() (string, error) {
		attempts++
		if attempts == 1 {
			return p.QRCode, nil
		}
		return originalQR()
	}
	request := createRequest()
	request.BatchID = "batch-2"
	meta := draftMetadata()
	meta.Key = "second-batch"
	second, err := store.CreateDraft(ctx, meta, request)
	if err != nil {
		t.Fatal(err)
	}
	p2 := decodeProfile(t, second)
	if attempts != 2 || p2.QRCode == p.QRCode || p2.BatchID != "batch-2" {
		t.Fatal("collision rebound batch or was not retried")
	}
	store.QR = func() (string, error) { return p.QRCode, nil }
	request.BatchID = "batch-3"
	meta.Key = "exhausted-qr"
	if _, err = store.CreateDraft(ctx, meta, request); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("unbounded collision: %v", err)
	}
	store.QR = originalQR
	// A deterministic audit ID collision must roll back the just-created profile.
	var existingID string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM audit_entries LIMIT 1`).Scan(&existingID); err != nil {
		t.Fatal(err)
	}
	originalID := store.ID
	ids := 0
	store.ID = func() (string, error) {
		ids++
		if ids == 2 {
			return existingID, nil
		}
		return originalID()
	}
	meta.Key = "audit-failure"
	if _, err = store.CreateDraft(ctx, meta, request); err == nil {
		t.Fatal("audit failure did not abort command")
	}
	var profiles, keys int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM traceability_profiles),(SELECT count(*) FROM idempotency_records)`).Scan(&profiles, &keys); err != nil || profiles != 2 || keys != 2 {
		t.Fatalf("partial create: %d %d %v", profiles, keys, err)
	}
}

func TestDraftInvalidAndNotFound(t *testing.T) {
	ctx, pool := testutil.Database(t)
	store, _ := NewProfiles(pool, "sandbox")
	meta := draftMetadata()
	if _, err := NewProfiles(nil, "sandbox"); err == nil {
		t.Fatal("nil DB accepted")
	}
	if _, err := NewProfiles(pool, "production"); err == nil {
		t.Fatal("production enabled")
	}
	if _, err := store.Get(ctx, "bad-id"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, testProfile); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := store.EditDraft(ctx, meta, testProfile, 1, domain.EditDraftRequest{ExpectedVersion: 1}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	bad := createRequest()
	bad.ProductID = "bad"
	if _, err := store.CreateDraft(ctx, meta, bad); err == nil {
		t.Fatal("invalid binding accepted")
	}
	bad = createRequest()
	bad.Content.Evidence = []domain.Evidence{{ID: testProfile}}
	if _, err := store.CreateDraft(ctx, meta, bad); err == nil {
		t.Fatal("invalid content accepted")
	}
	meta.RequestID = ""
	if _, err := store.CreateDraft(ctx, meta, createRequest()); !errors.Is(err, ErrInvalidCommand) {
		t.Fatal(err)
	}
	if _, err := store.CreateDraft(context.Background(), meta, createRequest()); err == nil {
		t.Fatal("invalid request metadata accepted")
	}
}
