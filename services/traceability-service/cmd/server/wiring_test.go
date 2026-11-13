package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
)

func TestAssembledSandboxUsesRuntimeLOGIN(t *testing.T) {
	ctx, dsn, admin := testutil.RuntimeDatabase(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "public.pem")
	fixturePath := filepath.Join(dir, "fixture.json")
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}), 0600); err != nil {
		t.Fatal(err)
	}
	binding := domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SYNTHETIC-SKU", SourceNamespace: "fixture", BatchID: "synthetic-batch"}
	data, _ := json.Marshal(map[string]any{"batches": []domain.DependencySnapshot{{Binding: binding, BatchCode: "SYNTHETIC-LOT", CatalogVersion: "1", BatchVersion: "1"}}})
	if err = os.WriteFile(fixturePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"SANDBOX_JWT_PUBLIC_KEY_FILE": keyPath, "SANDBOX_FIXTURE_FILE": fixturePath}
	getenv := func(key string) string { return env[key] }
	cfg := config.Config{Mode: "sandbox", DatabaseURL: dsn, PublicWebURL: "http://localhost:3000", VideoHosts: []string{"video.example"}}
	handler, closeResources, err := assemble(ctx, cfg, getenv, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeResources)
	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequest("GET", "/readyz", nil))
	if ready.Code != 200 {
		t.Fatalf("runtime readiness: %d", ready.Code)
	}
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "sandbox-local"})
	now := time.Now()
	claims, _ := json.Marshal(map[string]any{"iss": identity.SandboxIssuer, "aud": identity.SandboxAudience, "sub": "22222222-2222-4222-8222-222222222222", "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "permissions": []string{domain.Create, domain.ReadInternal}})
	token := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	token += "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(token)))
	video := "https://video.example/synthetic-video"
	body, _ := json.Marshal(domain.CreateDraftRequest{Binding: binding, Content: domain.Content{CraftingVideoURL: &video}})
	request := httptest.NewRequest("POST", "/api/v1/admin/trace/records", strings.NewReader(string(body)))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Idempotency-Key", "runtime-create")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 201 {
		t.Fatalf("assembled create: %d %s", response.Code, response.Body.String())
	}
	// The migrator observes the write performed by the distinct runtime LOGIN.
	var count int
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM traceability_profiles`).Scan(&count); err != nil || count != 1 {
		t.Fatal("runtime write missing")
	}
	// Also exercise a real loopback TCP listener, not only Recorder dispatch.
	network := httptest.NewServer(handler)
	t.Cleanup(network.Close)
	var created domain.Profile
	if err = json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.QRURL != "http://localhost:3000/trace/"+created.QRCode || created.Revisions[0].Content.CraftingVideoURL == nil {
		t.Fatal("runtime QR URL/video configuration missing")
	}
	get, err := http.NewRequest("GET", network.URL+"/api/v1/admin/trace/records/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	get.Header.Set("Authorization", "Bearer "+token)
	actual, err := network.Client().Do(get)
	if err != nil {
		t.Fatal("loopback HTTP failed")
	}
	defer actual.Body.Close()
	if actual.StatusCode != 200 || actual.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("loopback read failed")
	}
	metricsRequest := httptest.NewRecorder()
	handler.ServeHTTP(metricsRequest, httptest.NewRequest("GET", "/metrics", nil))
	if metricsRequest.Code != 200 || !strings.Contains(metricsRequest.Body.String(), "traceability_relay_enabled 0") || strings.Contains(metricsRequest.Body.String(), created.ID) {
		t.Fatal("metrics capability/cardinality invalid")
	}
	if _, err = admin.Exec(ctx, `UPDATE schema_migrations SET checksum='tampered'`); err != nil {
		t.Fatal(err)
	}
	notReady := httptest.NewRecorder()
	handler.ServeHTTP(notReady, httptest.NewRequest("GET", "/readyz", nil))
	if notReady.Code != 503 {
		t.Fatal("corrupt schema remained ready")
	}
	if _, _, err = assemble(ctx, cfg, getenv, nil); err == nil {
		t.Fatal("startup allowed corrupt schema")
	}
	cfg.Mode = "integration"
	if _, _, err = assemble(ctx, cfg, getenv, nil); err == nil {
		t.Fatal("integration used sandbox adapters")
	}
}

func TestBoundedConfigurationFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("abcd"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBounded(path, 3); err == nil {
		t.Fatal("oversized configuration accepted")
	}
	if data, err := readBounded(path, 4); err != nil || string(data) != "abcd" {
		t.Fatal("valid configuration rejected")
	}
	if _, err := readBounded(dir, 100); err == nil {
		t.Fatal("directory accepted")
	}
}
