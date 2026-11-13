package httptransport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

type dependencyFunc func(context.Context, domain.Binding) (domain.DependencySnapshot, error)

func (f dependencyFunc) Snapshot(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
	return f(ctx, b)
}

func draftServer(t *testing.T) (http.Handler, *application.Drafts, *postgres.Profiles, func(...string) string) {
	t.Helper()
	_, pool := testutil.Database(t)
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store, _ := postgres.NewProfiles(pool, "sandbox")
	store.Clock = clock
	fixture, err := dependency.NewFixture("sandbox", []domain.DependencySnapshot{{Binding: draftInput().Binding, BatchCode: "SYNTHETIC-LOT-1", CatalogVersion: "synthetic-1", BatchVersion: "synthetic-1"}}, clock)
	if err != nil {
		t.Fatal(err)
	}
	service := &application.Drafts{Store: store, Dependencies: fixture, Clock: clock}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := identity.NewSandboxVerifier("sandbox", map[string]ed25519.PublicKey{"test": public}, clock)
	if err != nil {
		t.Fatal(err)
	}
	token := func(permissions ...string) string {
		header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "test"})
		claims, _ := json.Marshal(map[string]any{"iss": identity.SandboxIssuer, "aud": identity.SandboxAudience, "sub": "22222222-2222-4222-8222-222222222222", "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "permissions": permissions})
		data := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
		return data + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(data)))
	}
	return NewHandler(HandlerConfig{Identity: verifier, Drafts: service}), service, store, token
}
func draftInput() domain.CreateDraftRequest {
	return domain.CreateDraftRequest{Binding: domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SKU-1", SourceNamespace: "fixture", BatchID: "batch-1"}, Content: domain.Content{ProductName: "Synthetic draft", Evidence: []domain.Evidence{{ID: "55555555-5555-4555-8555-555555555555", Reference: "internal/synthetic-proof", Kind: "CERTIFICATE", Description: "Synthetic proof"}}}}
}
func sendDraft(h http.Handler, method, path, body, token, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("X-Idempotency-Key", key)
	}
	// Client-supplied identity/roles must never be trusted.
	req.Header.Set("X-User-Role", "ADMIN")
	req.Header.Set("X-User-Id", "33333333-3333-4333-8333-333333333333")
	req.Header.Set("X-Request-Id", "spoofed-request-id")
	req.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
func encoded(value any) string { data, _ := json.Marshal(value); return string(data) }

const recordsPath = "/api/v1/admin/trace/records"

func TestSignedDraftHTTPAndReplayPermissions(t *testing.T) {
	h, service, store, token := draftServer(t)
	editor := token(domain.Create, domain.Edit, domain.ReadInternal)
	reader := token(domain.ReadInternal)
	body := encoded(draftInput())
	created := sendDraft(h, "POST", recordsPath, body, editor, "create-1")
	if created.Code != 201 {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	if strings.Contains(created.Body.String(), "internal/synthetic-proof") {
		t.Fatal("writer without read_evidence received reference")
	}
	var p domain.Profile
	if err := json.Unmarshal(created.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Revisions[0].CreatedBy != "22222222-2222-4222-8222-222222222222" {
		t.Fatal("trusted spoofed identity header")
	}
	raw, err := store.Get(context.Background(), p.ID)
	if err != nil || raw.Revisions[0].Content.Evidence[0].Reference != "internal/synthetic-proof" {
		t.Fatal("redaction mutated stored history")
	}
	if id := created.Header().Get("X-Request-Id"); len(id) != 32 || id == "spoofed-request-id" {
		t.Fatal("unsafe request id")
	}
	for _, permission := range [][]string{{domain.ReadInternal}, {domain.Create}, {"ADMIN"}} {
		denied := sendDraft(h, "POST", recordsPath, body, token(permission...), "create-1")
		expected := 403
		if permission[0] == "ADMIN" {
			expected = 401
		}
		if denied.Code != expected {
			t.Fatalf("replay permission bypass: %d %s", denied.Code, denied.Body.String())
		}
	}
	service.Dependencies = dependencyFunc(func(context.Context, domain.Binding) (domain.DependencySnapshot, error) {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	})
	replay := sendDraft(h, "POST", recordsPath, body, editor, "create-1")
	if replay.Code != 201 || replay.Header().Get("X-Idempotency-Replayed") != "true" || replay.Body.String() != created.Body.String() {
		t.Fatalf("replay while dependency down: %d %s", replay.Code, replay.Body.String())
	}
	if failed := sendDraft(h, "POST", recordsPath, body, editor, "new-create"); failed.Code != 503 {
		t.Fatalf("fake fallback: %d", failed.Code)
	}
	path := recordsPath + "/" + p.ID
	visible := sendDraft(h, "GET", path, "", token(domain.ReadInternal, domain.ReadEvidence), "")
	if visible.Code != 200 || !strings.Contains(visible.Body.String(), "internal/synthetic-proof") {
		t.Fatal("authorized evidence unavailable")
	}
	redacted := sendDraft(h, "GET", path, "", reader, "")
	if redacted.Code != 200 || strings.Contains(redacted.Body.String(), "internal/synthetic-proof") {
		t.Fatal("read evidence leak")
	}
	listed := sendDraft(h, "GET", recordsPath+"?page_size=1", "", reader, "")
	if listed.Code != 200 || strings.Contains(listed.Body.String(), "internal/synthetic-proof") {
		t.Fatal("list evidence leak")
	}
	for _, query := range []string{"page_size=0", "page_size=101", "cursor=malformed", "unknown=1", "page_size=1&page_size=2"} {
		if invalid := sendDraft(h, "GET", recordsPath+"?"+query, "", reader, ""); invalid.Code != 400 {
			t.Fatal("invalid list query accepted", query, invalid.Code)
		}
	}
	patch := encoded(domain.EditDraftRequest{ExpectedVersion: 1, Content: domain.Content{ProductName: "Changed draft"}})
	if denied := sendDraft(h, "PATCH", path+"/revisions/1", patch, reader, "edit-1"); denied.Code != 403 {
		t.Fatal("read token edited draft")
	}
	edited := sendDraft(h, "PATCH", path+"/revisions/1", patch, editor, "edit-1")
	if edited.Code != 200 {
		t.Fatalf("edit: %d %s", edited.Code, edited.Body.String())
	}
	stale := sendDraft(h, "PATCH", path+"/revisions/1", patch, editor, "edit-2")
	if stale.Code != 409 {
		t.Fatalf("stale edit: %d", stale.Code)
	}
	replay = sendDraft(h, "PATCH", path+"/revisions/1", patch, editor, "edit-1")
	if replay.Code != 200 || replay.Body.String() != edited.Body.String() {
		t.Fatal("edit replay changed response")
	}
	for _, response := range []*httptest.ResponseRecorder{created, replay, redacted, visible, stale} {
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store")
		}
	}
}

func TestDraftHTTPValidationAndSourceGuards(t *testing.T) {
	h, _, _, token := draftServer(t)
	editor := token(domain.Create, domain.Edit, domain.ReadInternal)
	for _, body := range []string{`null`, `{}`, `{"product_id":"x"}`, `{"unknown":1}`, encoded(draftInput()) + `{}`, strings.Replace(encoded(draftInput()), `"SKU-1"`, `"SKU-1","SKU_CODE":"SKU-2"`, 1), strings.Replace(encoded(draftInput()), `"content":{`, `"content":null,"old_content":{`, 1)} {
		if result := sendDraft(h, "POST", recordsPath, body, editor, "validation"); result.Code != 400 {
			t.Errorf("malformed request accepted: %d", result.Code)
		}
	}
	if response := sendDraft(h, "POST", recordsPath, strings.Replace(encoded(draftInput()), `"product_name":"Synthetic draft"`, `"product_name":null`, 1), editor, "null-primitive"); response.Code != 400 {
		t.Fatal("null primitive coerced into empty Draft value")
	}
	if result := sendDraft(h, "POST", recordsPath, encoded(draftInput()), editor, ""); result.Code != 400 {
		t.Fatalf("missing key: %d", result.Code)
	}
	request := draftInput()
	request.SKUCode = "WRONG-SKU"
	if result := sendDraft(h, "POST", recordsPath, encoded(request), editor, "mismatch"); result.Code != 409 {
		t.Fatalf("known mismatch: %d", result.Code)
	}
	request = draftInput()
	request.BatchID = "unknown-batch"
	if result := sendDraft(h, "POST", recordsPath, encoded(request), editor, "missing"); result.Code != 404 {
		t.Fatalf("unknown batch: %d", result.Code)
	}
	request = draftInput()
	request.SourceNamespace = "inventory"
	if result := sendDraft(h, "POST", recordsPath, encoded(request), editor, "no-real-adapter"); result.Code != 503 {
		t.Fatalf("fixture fallback: %d", result.Code)
	}
	if result := sendDraft(h, "POST", recordsPath, encoded(draftInput()), "", "spoofed"); result.Code != 401 {
		t.Fatal("header-only identity trusted")
	}
	if result := sendDraft(h, "POST", recordsPath, encoded(draftInput()), editor+"invalid", "bad-token"); result.Code != 401 {
		t.Fatal("bad signature accepted")
	}
	oversize := fmt.Sprintf(`{"product_name":%q}`, strings.Repeat("x", 256*1024))
	if result := sendDraft(h, "POST", recordsPath, oversize, editor, "oversize"); result.Code != 400 {
		t.Fatal("oversize request accepted")
	}
	created := sendDraft(h, "POST", recordsPath, encoded(draftInput()), editor, "create")
	if created.Code != 201 {
		t.Fatal(created.Code)
	}
	request = draftInput()
	request.Content.ProductName = "Different body"
	if result := sendDraft(h, "POST", recordsPath, encoded(request), editor, "create"); result.Code != 409 {
		t.Fatalf("key/body mismatch: %d", result.Code)
	}
}

func TestMissingVerifierFailsClosed(t *testing.T) {
	h := NewHandler(HandlerConfig{})
	if result := sendDraft(h, "POST", recordsPath, `{}`, "fake-token", "key"); result.Code != 401 {
		t.Fatal("missing verifier accepted identity")
	}
	if result := sendDraft(h, "GET", "/livez", "", "", ""); result.Code != 200 {
		t.Fatal("liveness requires identity")
	}
}
