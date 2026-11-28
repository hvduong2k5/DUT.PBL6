package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
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
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
)

func TestScopedAssemblyRequiresWriterGuardAndServesLoopback(t *testing.T) {
	ctx, dsn, admin := testutil.RuntimeDatabase(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	keyPath, fixturePath := filepath.Join(folder, "public.pem"), filepath.Join(folder, "batches.json")
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encodedKey}), 0600); err != nil {
		t.Fatal(err)
	}
	const warehouse = "77777777-7777-4777-8777-777777777777"
	const batch = "33333333-3333-4333-8333-333333333333"
	snapshot := domain.DependencySnapshot{Binding: domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SYNTHETIC-SKU", SourceNamespace: "fixture", BatchID: batch}, WarehouseID: warehouse, BatchCode: "SYNTHETIC-LOT", CatalogVersion: "1", BatchVersion: "1", ProductionDate: "2026-11-01", ExpiryDate: "2027-11-01"}
	data, _ := json.Marshal(map[string]any{"batches": []domain.DependencySnapshot{snapshot}})
	if err = os.WriteFile(fixturePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"SANDBOX_JWT_PUBLIC_KEY_FILE": keyPath, "SANDBOX_FIXTURE_FILE": fixturePath, "EXPIRY_WORKER_ENABLED": "false"}
	getenv := func(k string) string { return env[k] }
	cfg := config.Config{Mode: "sandbox", DatabaseURL: dsn, PublicWebURL: "http://localhost:3000", CursorHMACKey: strings.Repeat("synthetic-key-", 3), OrderResolverEnabled: true, OrderResolverToken: strings.Repeat("synthetic-order-", 3)}
	// Resolver can be released first without opening warehouse-scoped staff API.
	disabled, resolverService, closeDisabled, err := assembleApplication(ctx, cfg, getenv, nil)
	if err != nil {
		t.Fatal("independent resolver assembly failed", err)
	}
	absent := httptest.NewRecorder()
	disabled.ServeHTTP(absent, httptest.NewRequest("GET", "/api/v2/admin/trace/records", nil))
	if absent.Code != 404 {
		t.Fatal("disabled API was exposed")
	}
	resolver := httptest.NewRecorder()
	disabled.ServeHTTP(resolver, httptest.NewRequest("POST", "/api/v1/internal/traceability/batch-links/resolve", nil))
	if resolver.Code != 404 || resolverService == nil {
		t.Fatal("internal resolver was not assembled independently")
	}
	closeDisabled()
	cfg.ScopedAPIEnabled = true
	if _, closeUnexpected, err := assemble(ctx, cfg, getenv, nil); err == nil {
		closeUnexpected()
		t.Fatal("scoped API started before writer enforcement")
	}
	conn, err := admin.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = migrations.RequireWarehouseWrites(ctx, conn.Conn())
	conn.Release()
	if err != nil {
		t.Fatal(err)
	}
	handler, closeHandler, err := assemble(ctx, cfg, getenv, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeHandler)
	network := httptest.NewServer(handler)
	t.Cleanup(network.Close)
	now := time.Now()
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "sandbox-local"})
	claims, _ := json.Marshal(map[string]any{"iss": identity.SandboxIssuer, "aud": identity.SandboxAudience, "sub": "22222222-2222-4222-8222-222222222222", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "permission_grants": []string{domain.ReadInternal + "#WAREHOUSE#" + warehouse}})
	raw := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	raw += "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(raw)))
	request, _ := http.NewRequest("GET", network.URL+"/api/v2/admin/trace/batches/"+batch+"/context?sku_code=SYNTHETIC-SKU", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	response, err := network.Client().Do(request)
	if err != nil {
		t.Fatal("loopback HTTP failed")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 || response.Header.Get("Cache-Control") != "no-store" || !strings.Contains(string(body), warehouse) {
		t.Fatal("scoped assembly lost its warehouse capability", response.StatusCode, err)
	}
	ready, err := network.Client().Get(network.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer ready.Body.Close()
	if ready.StatusCode != 200 {
		t.Fatal("enforced runtime schema was not ready")
	}
}
