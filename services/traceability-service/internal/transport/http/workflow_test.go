package httptransport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/dependency"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
)

const editorID = "22222222-2222-4222-8222-222222222222"
const reviewerID = "33333333-3333-4333-8333-333333333333"

func workflowServer(t *testing.T) (http.Handler, *application.Workflow, *postgres.Profiles, func(string, ...string) string) {
	t.Helper()
	_, pool := testutil.Database(t)
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store, _ := postgres.NewProfiles(pool, "sandbox")
	store.Clock = clock
	fixture, err := dependency.NewFixture("sandbox", []domain.DependencySnapshot{{Binding: draftInput().Binding, BatchCode: "SYNTHETIC-LOT", CatalogVersion: "1", BatchVersion: "1", ProductionDate: "2026-10-01", ExpiryDate: "2027-01-01", BestBefore: time.Date(2027, 1, 1, 17, 0, 0, 0, time.UTC)}}, clock)
	if err != nil {
		t.Fatal(err)
	}
	drafts := &application.Drafts{Store: store, Dependencies: fixture, Clock: clock}
	workflow := &application.Workflow{Store: store, Dependencies: fixture, Clock: clock}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := identity.NewSandboxVerifier("sandbox", map[string]ed25519.PublicKey{"test": public}, clock)
	if err != nil {
		t.Fatal(err)
	}
	token := func(actor string, permissions ...string) string {
		header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "test"})
		claims, _ := json.Marshal(map[string]any{"iss": identity.SandboxIssuer, "aud": identity.SandboxAudience, "sub": actor, "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "permissions": permissions})
		data := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
		return data + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(data)))
	}
	return NewHandler(HandlerConfig{Identity: verifier, Drafts: drafts, Workflow: workflow}), workflow, store, token
}
func completeDraftInput() domain.CreateDraftRequest {
	request := draftInput()
	end := "2026-12-31"
	stars := 4
	request.Content.Producer = "Synthetic workshop"
	request.Content.WorkshopLocation = "Hue"
	request.Content.Process = "Synthetic process"
	request.Content.Ingredients = []domain.Ingredient{{Name: "Synthetic material", Origin: "Hue"}}
	request.Content.Certificates = []domain.Certificate{{ID: "66666666-6666-4666-8666-666666666666", Type: "OCOP", Number: "SYNTHETIC-OCOP", Issuer: "Synthetic authority", Scope: "PRODUCT", ScopeID: request.ProductID, IssuedOn: "2026-01-01", ValidUntil: &end, Stars: &stars, VerificationMethod: "MANUAL", VerifiedAt: time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC), EvidenceIDs: []string{request.Content.Evidence[0].ID}}}
	return request
}

func TestSignedHTTPReviewAndRevisionHistory(t *testing.T) {
	h, workflow, store, token := workflowServer(t)
	editor := token(editorID, domain.ReadInternal, domain.Create, domain.Edit, domain.Submit)
	reviewer := token(reviewerID, domain.ReadInternal, domain.ReadEvidence, domain.Approve, domain.Edit)
	created := sendDraft(h, "POST", recordsPath, encoded(completeDraftInput()), editor, "create")
	if created.Code != 201 {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	var p domain.Profile
	if err := json.Unmarshal(created.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	revisionPath := recordsPath + "/" + p.ID + "/revisions/1"
	submitted := sendDraft(h, "POST", revisionPath+"/submit", `{"expected_version":1}`, editor, "submit")
	if submitted.Code != 200 {
		t.Fatalf("submit %d %s", submitted.Code, submitted.Body.String())
	}
	edit := sendDraft(h, "PATCH", revisionPath, encoded(domain.EditDraftRequest{ExpectedVersion: 2, Content: completeDraftInput().Content}), editor, "edit-frozen")
	if edit.Code != 409 {
		t.Fatal("submitted content edited")
	}
	self := token(editorID, domain.ReadInternal, domain.Approve)
	if response := sendDraft(h, "POST", revisionPath+"/approve", `{"expected_version":2,"reason":"Self attempt"}`, self, "self-approve"); response.Code != 403 {
		t.Fatalf("self approval %d", response.Code)
	}
	if response := sendDraft(h, "POST", revisionPath+"/approve", `{"expected_version":2}`, reviewer, "missing-reason"); response.Code != 400 {
		t.Fatal("missing approval reason accepted")
	}
	approved := sendDraft(h, "POST", revisionPath+"/approve", `{"expected_version":2,"reason":"Independent evidence checked"}`, reviewer, "approve")
	if approved.Code != 200 {
		t.Fatalf("approve %d %s", approved.Code, approved.Body.String())
	}
	var after domain.Profile
	_ = json.Unmarshal(approved.Body.Bytes(), &after)
	if after.PublicRevision != nil || after.EverPublished || after.Revisions[0].Status != "APPROVED" {
		t.Fatal("approve implicitly published")
	}
	if public := sendDraft(h, "GET", "/api/v1/trace/"+after.QRCode, "", "", ""); public.Code != 404 {
		t.Fatal("unpublished public token visible")
	}
	replay := sendDraft(h, "POST", revisionPath+"/approve", `{"expected_version":2,"reason":"Independent evidence checked"}`, reviewer, "approve")
	if replay.Code != 200 || replay.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Fatal("decision replay failed")
	}
	if denied := sendDraft(h, "POST", revisionPath+"/approve", `{"expected_version":2,"reason":"Independent evidence checked"}`, token(reviewerID, domain.ReadInternal), "approve"); denied.Code != 403 {
		t.Fatal("replay bypassed revoked permission")
	}
	nextPath := recordsPath + "/" + p.ID + "/revisions"
	if missing := sendDraft(h, "POST", nextPath, `{"expected_version":3,"from_revision":1}`, editor, "withdraw-no-reason"); missing.Code != 400 {
		t.Fatal("approved withdrawal lacked reason")
	}
	cloned := sendDraft(h, "POST", nextPath, `{"expected_version":3,"from_revision":1,"reason":"Correct evidence in a new revision"}`, reviewer, "clone")
	if cloned.Code != 201 {
		t.Fatalf("clone %d %s", cloned.Code, cloned.Body.String())
	}
	after, err := store.Get(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revisions[0].Status != "ARCHIVED" || after.Revisions[1].Status != "DRAFT" || after.Revisions[1].DependencySnapshot != nil || len(after.Revisions[1].Contributors) != 2 || after.QRCode != p.QRCode {
		t.Fatal("clone history/identity changed")
	}
	submitted = sendDraft(h, "POST", nextPath+"/2/submit", `{"expected_version":4}`, editor, "submit-2")
	if submitted.Code != 200 {
		t.Fatalf("second submit %d %s", submitted.Code, submitted.Body.String())
	}
	if response := sendDraft(h, "POST", nextPath+"/2/approve", `{"expected_version":5,"reason":"Inherited contributor attempt"}`, reviewer, "approve-own-clone"); response.Code != 403 {
		t.Fatal("clone creator self-approved")
	}
	independent := token("77777777-7777-4777-8777-777777777777", domain.ReadInternal, domain.Approve)
	rejected := sendDraft(h, "POST", nextPath+"/2/reject", `{"expected_version":5,"reason":"Further evidence needed"}`, independent, "reject-2")
	if rejected.Code != 200 {
		t.Fatalf("reject %d %s", rejected.Code, rejected.Body.String())
	}
	workflow.Dependencies = dependencyFunc(func(context.Context, domain.Binding) (domain.DependencySnapshot, error) {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	})
	if replay = sendDraft(h, "POST", revisionPath+"/submit", `{"expected_version":1}`, editor, "submit"); replay.Code != 200 || replay.Header().Get("X-Idempotency-Replayed") != "true" || strings.Contains(replay.Body.String(), "internal/synthetic-proof") {
		t.Fatal("submit replay failed or leaked evidence")
	}
}

func TestSubmitNetworkVerificationOutsideTransactionAndCAS(t *testing.T) {
	h, workflow, store, token := workflowServer(t)
	editor := token(editorID, domain.ReadInternal, domain.Create, domain.Edit, domain.Submit)
	created := sendDraft(h, "POST", recordsPath, encoded(completeDraftInput()), editor, "create")
	if created.Code != 201 {
		t.Fatal(created.Code)
	}
	var p domain.Profile
	_ = json.Unmarshal(created.Body.Bytes(), &p)
	original := workflow.Dependencies
	workflow.Dependencies = dependencyFunc(func(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
		// A peer request must not hold the aggregate lock; this concurrent edit
		// acquires it and commits before the peer reply, so submit must detect CAS.
		current, err := store.Get(ctx, p.ID)
		if err != nil {
			return domain.DependencySnapshot{}, err
		}
		content := current.Revisions[0].Content
		content.ProductName = "Concurrent changed name"
		_, err = store.EditDraft(ctx, domain.CommandMetadata{PrincipalID: editorID, Key: "edit-during-verification", RequestID: "network-race"}, p.ID, 1, domain.EditDraftRequest{ExpectedVersion: 1, Content: content})
		if err != nil {
			return domain.DependencySnapshot{}, fmt.Errorf("edit blocked inside peer request: %w", err)
		}
		return original.Snapshot(ctx, b)
	})
	response := sendDraft(h, "POST", recordsPath+"/"+p.ID+"/revisions/1/submit", `{"expected_version":1}`, editor, "stale-submit")
	if response.Code != 409 {
		t.Fatalf("CAS after network call: %d %s", response.Code, response.Body.String())
	}
	current, err := store.Get(context.Background(), p.ID)
	if err != nil || current.Version != 2 || current.Revisions[0].Status != "DRAFT" {
		t.Fatal("stale source check committed")
	}
}

func TestIncompleteDraftCannotSubmit(t *testing.T) {
	h, _, _, token := workflowServer(t)
	editor := token(editorID, domain.ReadInternal, domain.Create, domain.Submit)
	created := sendDraft(h, "POST", recordsPath, encoded(draftInput()), editor, "create-incomplete")
	if created.Code != 201 {
		t.Fatal(created.Code)
	}
	var p domain.Profile
	_ = json.Unmarshal(created.Body.Bytes(), &p)
	rejected := sendDraft(h, "POST", recordsPath+"/"+p.ID+"/revisions/1/submit", `{"expected_version":1}`, editor, "submit-incomplete")
	if rejected.Code != 400 {
		t.Fatalf("incomplete draft submitted %d", rejected.Code)
	}
}

func TestDependencyTimeoutIs503WithoutSubmission(t *testing.T) {
	h, workflow, store, token := workflowServer(t)
	editor := token(editorID, domain.ReadInternal, domain.Create, domain.Submit)
	created := sendDraft(h, "POST", recordsPath, encoded(completeDraftInput()), editor, "create-timeout")
	if created.Code != 201 {
		t.Fatal(created.Code)
	}
	var profile domain.Profile
	_ = json.Unmarshal(created.Body.Bytes(), &profile)
	workflow.Dependencies = dependencyFunc(func(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
		<-ctx.Done()
		return domain.DependencySnapshot{}, ctx.Err()
	})
	started := time.Now()
	response := sendDraft(h, "POST", recordsPath+"/"+profile.ID+"/revisions/1/submit", `{"expected_version":1}`, editor, "timeout-submit")
	if response.Code != 503 || time.Since(started) > 4*time.Second {
		t.Fatal("provider deadline not enforced", response.Code)
	}
	current, err := store.Get(context.Background(), profile.ID)
	if err != nil || current.Version != 1 || current.Revisions[0].Status != "DRAFT" {
		t.Fatal("timed-out source check changed state")
	}
	var keys int
	if err = store.DB.QueryRow(context.Background(), `SELECT count(*) FROM idempotency_records`).Scan(&keys); err != nil || keys != 1 {
		t.Fatal("timed-out submission created a dedup result")
	}
}
