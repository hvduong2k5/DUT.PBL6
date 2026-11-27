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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	grpctransport "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/transport/grpc"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const warehouseA = "77777777-7777-4777-8777-777777777777"
const warehouseB = "88888888-8888-4888-8888-888888888888"
const batchA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const batchB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
const scopedPath = "/api/v2/admin/trace/records"
const orderCredential = "synthetic-order-credential-32-bytes-minimum"

// Simulated Inventory/Catalog only. PostgreSQL, runtime grants, transactions,
// signed identity, strict HTTP parsing and domain workflow are exercised live.
type scopedSources struct {
	batches      map[string]domain.DependencySnapshot
	clock        func() time.Time
	offline      bool
	catalogCalls int
}

func (s *scopedSources) Batch(ctx context.Context, namespace, batch, sku string) (domain.DependencySnapshot, error) {
	if s.offline {
		return domain.DependencySnapshot{}, domain.ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return domain.DependencySnapshot{}, err
	}
	item, ok := s.batches[batch]
	if !ok {
		return domain.DependencySnapshot{}, domain.ErrNotFound
	}
	if item.SourceNamespace != namespace || item.SKUCode != sku {
		return domain.DependencySnapshot{}, domain.ErrConflict
	}
	item.CheckedAt = s.clock()
	return item, nil
}
func (s *scopedSources) Catalog(ctx context.Context, sku string) (string, string, error) {
	s.catalogCalls++
	for _, item := range s.batches {
		if item.SKUCode == sku {
			return item.ProductID, item.CatalogVersion, nil
		}
	}
	return "", "", domain.ErrNotFound
}
func (s *scopedSources) Snapshot(ctx context.Context, b domain.Binding) (domain.DependencySnapshot, error) {
	item, err := s.Batch(ctx, b.SourceNamespace, b.BatchID, b.SKUCode)
	if err == nil && item.Binding != b {
		return item, domain.ErrConflict
	}
	return item, err
}

type scopedHarness struct {
	workflow *application.Workflow
	rpc      traceabilityv1.BatchLinkResolverClient
	handler  http.Handler
	store    *postgres.Profiles
	admin    *pgxpool.Pool
	sources  *scopedSources
	token    func(string, string, ...string) string
}

func scopedServer(t *testing.T, enforce bool) scopedHarness {
	t.Helper()
	ctx, dsn, admin := testutil.RuntimeDatabase(t)
	if enforce {
		conn, err := admin.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		err = migrations.RequireWarehouseWrites(ctx, conn.Conn())
		conn.Release()
		if err != nil {
			t.Fatal(err)
		}
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store, _ := postgres.NewProfiles(pool, "sandbox")
	store.Clock = clock
	store.CursorKey = []byte(strings.Repeat("synthetic-cursor-key-", 3))
	source := &scopedSources{clock: clock, batches: map[string]domain.DependencySnapshot{}}
	for _, item := range []struct{ batch, warehouse, sku string }{{batchA, warehouseA, "SKU-1"}, {batchB, warehouseB, "SKU-2"}} {
		source.batches[item.batch] = domain.DependencySnapshot{Binding: domain.Binding{ProductID: draftInput().ProductID, SKUCode: item.sku, SourceNamespace: "inventory", BatchID: item.batch}, WarehouseID: item.warehouse, BatchCode: "SYNTHETIC-LOT", ProductionDate: "2026-10-01", ExpiryDate: "2027-01-01", BestBefore: time.Date(2027, 1, 1, 17, 0, 0, 0, time.UTC), CatalogVersion: "1", BatchVersion: "1"}
	}
	drafts := &application.Drafts{Store: store, Dependencies: source, Clock: clock, PublicWebURL: "https://shop.example.test"}
	workflow := &application.Workflow{Store: store, Dependencies: source, Clock: clock, PublicWebURL: drafts.PublicWebURL}
	scoped := &application.Scoped{Store: store, Sources: source, Drafts: drafts, Workflow: workflow, Namespace: "inventory"}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := identity.NewSandboxVerifier("sandbox", map[string]ed25519.PublicKey{"test": public}, clock)
	if err != nil {
		t.Fatal(err)
	}
	token := func(actor, warehouse string, permissions ...string) string {
		grants := []string{}
		for _, permission := range permissions {
			suffix := "#GLOBAL"
			if warehouse != "" {
				suffix = "#WAREHOUSE#" + warehouse
			}
			grants = append(grants, permission+suffix)
		}
		header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "test"})
		claims, _ := json.Marshal(map[string]any{"iss": identity.SandboxIssuer, "aud": identity.SandboxAudience, "sub": actor, "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "permission_grants": grants})
		raw := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
		return raw + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(raw)))
	}
	handler := NewHandler(HandlerConfig{Identity: verifier, Drafts: drafts, Workflow: workflow, Scoped: scoped})
	rpcConnection := testutil.GRPC(t, func(options ...grpc.ServerOption) (*grpc.Server, error) {
		return grpctransport.NewServer(&application.Resolver{Store: store, Clock: clock, PublicWebURL: drafts.PublicWebURL}, orderCredential, options...)
	})
	return scopedHarness{workflow: workflow, handler: handler, store: store, admin: admin, sources: source, token: token, rpc: traceabilityv1.NewBatchLinkResolverClient(rpcConnection)}
}
func sendScoped(h http.Handler, method, path, body, token, key string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	r.Header.Set("X-User-Role", "ADMIN")
	r.Header.Set("X-User-Id", reviewerID)
	r.Header.Set("X-Request-Id", "synthetic-scoped-request")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func scopedInput(batch, sku string) domain.CreateDraftRequest {
	r := completeDraftInput()
	r.SourceNamespace = "inventory"
	r.BatchID = batch
	r.SKUCode = sku
	return r
}
func requireResponse(t *testing.T, r *httptest.ResponseRecorder, status int) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("wanted %d got %d: %s", status, r.Code, r.Body.String())
	}
	if r.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("response was cacheable")
	}
}
func responseProfile(t *testing.T, r *httptest.ResponseRecorder) domain.Profile {
	t.Helper()
	var p domain.Profile
	if err := json.Unmarshal(r.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func captureScopedContract(t *testing.T, name string, r *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	folder := os.Getenv("TRACEABILITY_TEST_ARTIFACT_DIR")
	if folder == "" {
		return r
	}
	if !json.Valid(r.Body.Bytes()) {
		t.Fatal("contract response is not JSON")
	}
	document := map[string]any{"status": r.Code, "cacheControl": r.Header().Get("Cache-Control"), "requestId": r.Header().Get("X-Request-Id"), "body": json.RawMessage(r.Body.Bytes())}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(folder, name+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestScopedWarehouseReplayAndEvidence(t *testing.T) {
	h := scopedServer(t, true)
	maker := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create, domain.Edit)
	request := scopedInput(batchA, "SKU-1")
	body := encoded(request)
	key := "scoped-create-key-0001"
	response := sendScoped(h.handler, "POST", scopedPath, body, maker, key)
	captureScopedContract(t, "create", response)
	requireResponse(t, response, 201)
	p := responseProfile(t, response)
	if domain.Warehouse(p) != warehouseA || strings.Contains(response.Body.String(), "internal/synthetic-proof") {
		t.Fatal("warehouse binding or evidence redaction failed")
	}
	outsider := h.token(reviewerID, warehouseB, domain.ReadInternal, domain.Create, domain.Edit)
	requireResponse(t, sendScoped(h.handler, "GET", scopedPath+"/"+p.ID, "", outsider, ""), 403)
	requireResponse(t, sendScoped(h.handler, "POST", scopedPath, encoded(scopedInput(batchB, "SKU-2")), maker, "scoped-create-key-0002"), 403)
	requireResponse(t, sendScoped(h.handler, "POST", scopedPath, body, h.token(editorID, warehouseB, domain.ReadInternal, domain.Create), key), 403)
	h.sources.offline = true
	replay := sendScoped(h.handler, "POST", scopedPath, body, maker, key)
	requireResponse(t, replay, 201)
	if replay.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Fatal("replay required peer or mutated state")
	}
	h.sources.offline = false
	edit := domain.EditDraftRequest{ExpectedVersion: 1, Content: request.Content}
	edit.Content.Evidence = append([]domain.Evidence(nil), request.Content.Evidence...)
	edit.Content.Evidence[0].Reference = ""
	path := scopedPath + "/" + p.ID + "/revisions/1"
	editBody := encoded(edit)
	first := sendScoped(h.handler, "PATCH", path, editBody, maker, "scoped-edit-key-00001")
	requireResponse(t, first, 200)
	edit.ExpectedVersion = 2
	edit.Content.Evidence[0].Reference = "internal/updated-proof"
	requireResponse(t, sendScoped(h.handler, "PATCH", path, encoded(edit), maker, "scoped-edit-key-00002"), 200)
	replay = sendScoped(h.handler, "PATCH", path, editBody, maker, "scoped-edit-key-00001")
	requireResponse(t, replay, 200)
	if replay.Header().Get("X-Idempotency-Replayed") != "true" || responseProfile(t, replay).Version != 2 {
		t.Fatal("preserve marker changed replay fingerprint")
	}
	stored, err := h.store.Get(context.Background(), p.ID)
	if err != nil || stored.Version != 3 || stored.Revisions[0].Content.Evidence[0].Reference != "internal/updated-proof" {
		t.Fatal("replay overwrote current evidence", err)
	}
	edit.ExpectedVersion = 3
	edit.Content.Evidence[0].ID = "99999999-9999-4999-8999-999999999999"
	edit.Content.Evidence[0].Reference = ""
	requireResponse(t, sendScoped(h.handler, "PATCH", path, encoded(edit), maker, "scoped-edit-key-00003"), 400)
	// A GLOBAL v1 read still omits the new root metadata. WAREHOUSE tokens do not enter v1.
	legacy := sendDraft(h.handler, "GET", recordsPath+"/"+p.ID, "", h.token(editorID, "", domain.ReadInternal), "")
	requireResponse(t, legacy, 200)
	var raw map[string]any
	_ = json.Unmarshal(legacy.Body.Bytes(), &raw)
	if _, exists := raw["warehouse_id"]; exists {
		t.Fatal("legacy root wire changed")
	}
	requireResponse(t, sendDraft(h.handler, "GET", recordsPath+"/"+p.ID, "", maker, ""), 403)
}

func TestScopedContextLookupAndCursorIsolation(t *testing.T) {
	h := scopedServer(t, true)
	maker := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create)
	contextPath := "/api/v2/admin/trace/batches/" + batchB + "/context?sku_code=SKU-2"
	requireResponse(t, sendScoped(h.handler, "GET", contextPath, "", maker, ""), 403)
	if h.sources.catalogCalls != 0 {
		t.Fatal("Catalog queried before warehouse authorization")
	}
	contextPath = "/api/v2/admin/trace/batches/" + batchA + "/context?sku_code=SKU-1"
	requireResponse(t, captureScopedContract(t, "context", sendScoped(h.handler, "GET", contextPath, "", maker, "")), 200)
	lookup := scopedPath + "/by-batch?product_id=" + draftInput().ProductID + "&sku_code=SKU-1&source_namespace=inventory&batch_id=" + batchA
	empty := sendScoped(h.handler, "GET", lookup, "", maker, "")
	captureScopedContract(t, "lookup-empty", empty)
	requireResponse(t, empty, 200)
	if !strings.Contains(empty.Body.String(), `"total":0`) {
		t.Fatal("empty lookup failed")
	}
	for i := 0; i < 6; i++ {
		batch := fmt.Sprintf("%08x-aaaa-4aaa-8aaa-aaaaaaaaaaaa", i+1)
		warehouse, sku := warehouseA, "SKU-1"
		if i%2 != 0 {
			warehouse, sku = warehouseB, "SKU-2"
		}
		request := scopedInput(batch, sku)
		meta := domain.CommandMetadata{PrincipalID: editorID, WarehouseID: warehouse, Key: fmt.Sprintf("seed-%d", i), RequestID: "seed"}
		if _, err := h.store.CreateDraft(context.Background(), meta, request); err != nil {
			t.Fatal(err)
		}
	}
	first := sendScoped(h.handler, "GET", scopedPath+"?page_size=1", "", maker, "")
	captureScopedContract(t, "list", first)
	requireResponse(t, first, 200)
	var page struct {
		Items []domain.Profile `json:"items"`
		Next  string           `json:"next_cursor"`
		Total int              `json:"total"`
	}
	if json.Unmarshal(first.Body.Bytes(), &page) != nil || page.Total != 1 || len(page.Items) != 1 || domain.Warehouse(page.Items[0]) != warehouseA || page.Next == "" || len(page.Next) > 256 {
		t.Fatal("unscoped pagination", first.Body.String())
	}
	next := scopedPath + "?page_size=1&cursor=" + page.Next
	requireResponse(t, sendScoped(h.handler, "GET", next, "", maker, ""), 200)
	changed := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create, domain.ReadEvidence)
	invalid := sendScoped(h.handler, "GET", next, "", changed, "")
	requireResponse(t, invalid, 400)
	if !strings.Contains(invalid.Body.String(), "INVALID_CURSOR") {
		t.Fatal("scope cursor not bound to grants")
	}
	requireResponse(t, sendScoped(h.handler, "GET", next, "", h.token(reviewerID, warehouseA, domain.ReadInternal, domain.Create), ""), 400)
}

func TestScopedWorkflowAndOrderResolver(t *testing.T) {
	h := scopedServer(t, true)
	maker := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create, domain.Edit, domain.Submit, domain.Approve)
	reviewer := h.token(reviewerID, warehouseA, domain.ReadInternal, domain.Approve, domain.Publish, domain.Suspend)
	created := sendScoped(h.handler, "POST", scopedPath, encoded(scopedInput(batchA, "SKU-1")), maker, "workflow-create-0001")
	requireResponse(t, created, 201)
	p := responseProfile(t, created)
	revision := scopedPath + "/" + p.ID + "/revisions/1"
	requireResponse(t, sendScoped(h.handler, "POST", revision+"/submit", `{"expected_version":1}`, maker, "workflow-submit-0001"), 200)
	requireResponse(t, sendScoped(h.handler, "POST", revision+"/approve", `{"expected_version":2,"reason":"Self review"}`, maker, "workflow-review-self"), 403)
	requireResponse(t, sendScoped(h.handler, "POST", revision+"/approve", `{"expected_version":2,"reason":"Independent review"}`, reviewer, "workflow-review-0001"), 200)
	snapshot := h.sources.batches[batchA]
	snapshot.BatchVersion = "2"
	h.sources.batches[batchA] = snapshot
	requireResponse(t, sendScoped(h.handler, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "workflow-publish-001"), 409)
	snapshot.BatchVersion = "1"
	snapshot.WarehouseID = warehouseB
	h.sources.batches[batchA] = snapshot
	mismatch := sendScoped(h.handler, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "workflow-publish-001")
	requireResponse(t, mismatch, 409)
	if !strings.Contains(mismatch.Body.String(), "WAREHOUSE_BINDING_CONFLICT") {
		t.Fatal("warehouse changed silently")
	}
	snapshot.WarehouseID = warehouseA
	h.sources.batches[batchA] = snapshot
	if _, err := h.admin.Exec(context.Background(), `CREATE FUNCTION reject_scoped_outbox_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic local outbox failure'; END $$; CREATE TRIGGER reject_scoped_outbox BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION reject_scoped_outbox_test()`); err != nil {
		t.Fatal(err)
	}
	requireResponse(t, sendScoped(h.handler, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "workflow-publish-001"), 503)
	afterFailure, err := h.store.Get(context.Background(), p.ID)
	if err != nil || afterFailure.Version != 3 || afterFailure.PublicRevision != nil {
		t.Fatal("outbox failure left a partial publication", err)
	}
	if _, err = h.admin.Exec(context.Background(), `DROP TRIGGER reject_scoped_outbox ON outbox_events; DROP FUNCTION reject_scoped_outbox_test()`); err != nil {
		t.Fatal(err)
	}
	published := sendScoped(h.handler, "POST", revision+"/publish", `{"expected_version":3}`, reviewer, "workflow-publish-001")
	captureScopedContract(t, "published", published)
	requireResponse(t, published, 200)
	requireResponse(t, captureScopedContract(t, "public", sendScoped(h.handler, "GET", "/api/v1/trace/"+p.QRCode, "", "", "")), 200)
	items := []*traceabilityv1.TraceabilityBatchBinding{{SourceNamespace: "inventory", BatchId: batchA, SkuCode: "SKU-1"}, {SourceNamespace: "inventory", BatchId: batchA, SkuCode: "SKU-1"}, {SourceNamespace: "inventory", BatchId: batchA, SkuCode: "wrong"}}
	h.sources.offline = true
	resolved, rpcErr := resolveRPC(h, orderCredential, items)
	if rpcErr != nil || resolved.GetTotal() != 3 || resolved.GetItems()[0].GetStatus() != traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_AVAILABLE || resolved.Items[0].QrUrl == nil || resolved.Items[1].GetStatus() != traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_AVAILABLE || resolved.Items[2].GetStatus() != traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE || resolved.Items[2].QrUrl != nil {
		t.Fatal("resolver order/duplicates/status/presence", rpcErr)
	}
	if resolved.Items[0].GetBinding().GetBatchId() != batchA || resolved.Items[1].GetBinding().GetSkuCode() != "SKU-1" {
		t.Fatal("binding echo lost")
	}
	requireResponse(t, sendScoped(h.handler, "POST", scopedPath+"/"+p.ID+"/suspend", `{"expected_version":4,"reason":"Manual withdrawal"}`, reviewer, "workflow-suspend-001"), 200)
	resolved, rpcErr = resolveRPC(h, orderCredential, items)
	if rpcErr != nil || resolved.GetItems()[0].GetStatus() != traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NO_LONGER_PUBLIC || resolved.Items[0].QrUrl == nil {
		t.Fatal("withdrawn public link lost", rpcErr)
	}
	requireResponse(t, sendScoped(h.handler, "GET", "/api/v1/trace/"+p.QRCode, "", "", ""), 410)
	if _, err := resolveRPC(h, maker, items); status.Code(err) != codes.Unauthenticated {
		t.Fatal("staff token accepted as service identity", err)
	}
	if _, err := resolveRPC(h, orderCredential, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatal("empty input accepted", err)
	}
	// The replaced internal HTTP proposal was never shipped and must stay absent.
	requireResponse(t, sendScoped(h.handler, "POST", "/api/v1/internal/traceability/batch-links/resolve", `{}`, orderCredential, ""), 404)
}

func TestScopedStrictHeadersAndErrors(t *testing.T) {
	h := scopedServer(t, true)
	maker := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create)
	response := sendScoped(h.handler, "GET", scopedPath, "", "", "")
	captureScopedContract(t, "unauthenticated", response)
	requireResponse(t, response, 401)
	var envelope map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &envelope)
	if len(envelope) != 3 || envelope["code"] != "UNAUTHENTICATED" || envelope["requestId"] != response.Header().Get("X-Request-Id") {
		t.Fatal("v2 error envelope", envelope)
	}
	for _, key := range []string{"short", strings.Repeat("a", 129), "invalid spaces in key"} {
		requireResponse(t, sendScoped(h.handler, "POST", scopedPath, encoded(scopedInput(batchA, "SKU-1")), maker, key), 400)
	}
	for _, body := range []string{`{"product_id":"a","product_id":"b"}`, encoded(scopedInput(batchA, "SKU-1")) + ` {}`, `{"warehouse_id":"` + warehouseA + `"}`, strings.Repeat("x", 256*1024+1)} {
		requireResponse(t, sendScoped(h.handler, "POST", scopedPath, body, maker, "strict-header-key-001"), 400)
	}
	r := httptest.NewRequest("POST", scopedPath, strings.NewReader(encoded(scopedInput(batchA, "SKU-1"))))
	r.Header.Set("Authorization", "Bearer "+maker)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "strict-header-key-001")
	r.Header.Set("X-Idempotency-Key", "legacy")
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	requireResponse(t, w, 400)
}

func TestScopedConcurrentCreateKeepsOneBatchQR(t *testing.T) {
	h := scopedServer(t, true)
	token := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create)
	responses := make(chan *httptest.ResponseRecorder, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			responses <- sendScoped(h.handler, "POST", scopedPath, encoded(scopedInput(batchA, "SKU-1")), token, fmt.Sprintf("concurrent-create-%02d", index))
		}(i)
	}
	wg.Wait()
	close(responses)
	counts := map[int]int{}
	for response := range responses {
		counts[response.Code]++
	}
	if counts[201] != 1 || counts[409] != 1 {
		t.Fatal("duplicate batch create", counts)
	}
	var profiles, audits, commands int
	err := h.admin.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM traceability_profiles),(SELECT count(*) FROM audit_entries),(SELECT count(*) FROM idempotency_records)`).Scan(&profiles, &audits, &commands)
	if err != nil || profiles != 1 || audits != 1 || commands != 1 {
		t.Fatal("race left partial writes", profiles, audits, commands, err)
	}
}

func resolveRPC(h scopedHarness, token string, items []*traceabilityv1.TraceabilityBatchBinding) (*traceabilityv1.ResolveBatchLinksResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ctx = grpcmetadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token, "x-request-id", "synthetic-resolver-request")
	return h.rpc.ResolveBatchLinks(ctx, &traceabilityv1.ResolveBatchLinksRequest{Items: items})
}
func TestResolverCorruptRowsAndDatabaseFailureStayDistinct(t *testing.T) {
	h := scopedServer(t, true)
	token := h.token(editorID, warehouseA, domain.ReadInternal, domain.Create)
	created := sendScoped(h.handler, "POST", scopedPath, encoded(scopedInput(batchA, "SKU-1")), token, "resolver-draft-key-01")
	requireResponse(t, created, 201)
	binding := &traceabilityv1.TraceabilityBatchBinding{SourceNamespace: "inventory", BatchId: batchA, SkuCode: "SKU-1"}
	input := []*traceabilityv1.TraceabilityBatchBinding{binding}
	check := func(expected traceabilityv1.BatchLinkStatus) {
		t.Helper()
		r, err := resolveRPC(h, orderCredential, input)
		if err != nil || len(r.GetItems()) != 1 || r.GetItems()[0].GetStatus() != expected || r.Items[0].QrUrl != nil {
			t.Fatal("invalid resolver state/presence", err)
		}
	}
	check(traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE)
	many := make([]*traceabilityv1.TraceabilityBatchBinding, 100)
	for i := range many {
		many[i] = binding
	}
	if r, err := resolveRPC(h, orderCredential, many); err != nil || len(r.GetItems()) != 100 || r.GetTotal() != 100 {
		t.Fatal("valid 100-binding request rejected", err)
	}
	if _, err := h.admin.Exec(context.Background(), `UPDATE traceability_revisions SET content='{"ingredients":"corrupt"}'::jsonb`); err != nil {
		t.Fatal(err)
	}
	check(traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_NOT_AVAILABLE)
	tx, err := h.admin.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	p := responseProfile(t, created)
	for _, statement := range []string{
		`UPDATE traceability_revisions SET status='IN_REVIEW',dependency_snapshot='{}' WHERE profile_id=$1`,
		`INSERT INTO review_decisions(id,profile_id,revision_number,actor_id,decision,reason,profile_version) VALUES('cccccccc-cccc-4ccc-8ccc-cccccccccccc',$1,1,'33333333-3333-4333-8333-333333333333','APPROVE','Synthetic historical corruption fixture',2)`,
		`UPDATE traceability_revisions SET status='APPROVED' WHERE profile_id=$1`,
		`UPDATE traceability_revisions SET status='PUBLISHED' WHERE profile_id=$1`,
		`UPDATE traceability_profiles SET public_revision=1,ever_published=true,version=2 WHERE id=$1`,
	} {
		if _, err = tx.Exec(context.Background(), statement, p.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	check(traceabilityv1.BatchLinkStatus_BATCH_LINK_STATUS_UNAVAILABLE)
	if _, err = resolveRPC(h, orderCredential, append(many, binding)); status.Code(err) != codes.InvalidArgument {
		t.Fatal("101 bindings accepted", err)
	}
	huge := []*traceabilityv1.TraceabilityBatchBinding{{SourceNamespace: "inventory", BatchId: batchA, SkuCode: strings.Repeat("x", 256*1024+1)}}
	if _, err = resolveRPC(h, orderCredential, huge); status.Code(err) != codes.ResourceExhausted {
		t.Fatal("gRPC byte cap not enforced", err)
	}
	h.store.DB.Close()
	if _, err = resolveRPC(h, orderCredential, input); status.Code(err) != codes.Unavailable {
		t.Fatal("DB failure reported absent", err)
	}
	requireResponse(t, sendScoped(h.handler, "GET", scopedPath, "", token, ""), 503)
	requireResponse(t, sendScoped(h.handler, "GET", scopedPath+"/"+p.ID, "", token, ""), 503)
}

func TestLegacyWriterAndV2IdempotencyNamespaces(t *testing.T) {
	h := scopedServer(t, true)
	global := h.token(editorID, "", domain.ReadInternal, domain.Create, domain.Edit)
	input := scopedInput(batchA, "SKU-1")
	created := sendDraft(h.handler, "POST", recordsPath, encoded(input), global, "shared-create-000001")
	requireResponse(t, created, 201)
	p := responseProfile(t, created)
	// A v2 retry cannot replay an old v1 response even with the same actor/key/body.
	requireResponse(t, sendScoped(h.handler, "POST", scopedPath, encoded(input), global, "shared-create-000001"), 409)
	path := recordsPath + "/" + p.ID + "/revisions/1"
	edit := domain.EditDraftRequest{ExpectedVersion: 1, Content: input.Content}
	requireResponse(t, sendDraft(h.handler, "PATCH", path, encoded(edit), global, "shared-edit-0000001"), 200)
	edit.ExpectedVersion = 2
	edit.Content.ProductName = "Second API version"
	updated := sendScoped(h.handler, "PATCH", scopedPath+"/"+p.ID+"/revisions/1", encoded(edit), global, "shared-edit-0000001")
	requireResponse(t, updated, 200)
	after := responseProfile(t, updated)
	if after.Version != 3 || domain.Warehouse(after) != warehouseA {
		t.Fatal("legacy writer lost warehouse or versioned keys collided")
	}
}

func TestLegacyNullWarehouseIsVisibleOnlyToGlobal(t *testing.T) {
	h := scopedServer(t, false)
	owner, _ := postgres.NewProfiles(h.admin, "sandbox")
	result, err := owner.CreateDraft(context.Background(), domain.CommandMetadata{PrincipalID: editorID, Key: "legacy-null", RequestID: "legacy"}, scopedInput(batchA, "SKU-1"))
	if err != nil {
		t.Fatal(err)
	}
	var p domain.Profile
	_ = json.Unmarshal(result.Body, &p)
	requireResponse(t, sendScoped(h.handler, "GET", scopedPath+"/"+p.ID, "", h.token(editorID, warehouseA, domain.ReadInternal), ""), 403)
	global := sendScoped(h.handler, "GET", scopedPath+"/"+p.ID, "", h.token(editorID, "", domain.ReadInternal), "")
	requireResponse(t, global, 200)
	if !strings.Contains(global.Body.String(), `"warehouse_id":null`) {
		t.Fatal("legacy warehouse defaulted or missing")
	}
}
