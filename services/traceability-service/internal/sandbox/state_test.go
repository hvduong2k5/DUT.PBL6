package sandbox_test

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/dependency"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/sandbox"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	httptransport "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/transport/http"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSeedReuseAndSmokeWithRuntimeLOGIN(t *testing.T) {
	ctx, admin := testutil.Database(t)
	var schema string
	if err := admin.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	parsedURL, err := url.Parse(os.Getenv("TRACEABILITY_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal("invalid test configuration")
	}
	// Keep all initialization/migration/grants inside this test's disposable schema.
	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	input := parsedURL.String()
	dir := filepath.Join(t.TempDir(), "sandbox")
	now := time.Now().UTC()
	state, err := sandbox.Initialize(ctx, "sandbox", input, dir, "http://localhost:3000", now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP OWNED BY "+pgx.Identifier{state.Role}.Sanitize()); err != nil {
			t.Error("runtime role grant cleanup failed")
		}
		if _, err := admin.Exec(ctx, "DROP ROLE "+pgx.Identifier{state.Role}.Sanitize()); err != nil {
			t.Error("runtime role cleanup failed")
		}
	})
	keyBefore, err := os.ReadFile(filepath.Join(dir, "private.pem"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeBefore, err := os.ReadFile(filepath.Join(dir, "runtime-url.txt"))
	if err != nil {
		t.Fatal(err)
	}
	reused, err := sandbox.Initialize(ctx, "sandbox", input, dir, "http://localhost:3000", now.Add(time.Second))
	if err != nil || reused.ID != state.ID || reused.Role != state.Role {
		t.Fatal("seed reuse failed", err)
	}
	keyAfter, _ := os.ReadFile(filepath.Join(dir, "private.pem"))
	runtimeAfter, _ := os.ReadFile(filepath.Join(dir, "runtime-url.txt"))
	if string(keyBefore) != string(keyAfter) || string(runtimeBefore) != string(runtimeAfter) {
		t.Fatal("seed reuse rotated credentials")
	}
	runtime, err := pgxpool.New(ctx, strings.TrimSpace(string(runtimeAfter)))
	if err != nil {
		t.Fatal("runtime login failed")
	}
	t.Cleanup(runtime.Close)
	if err = postgres.CheckRuntime(ctx, runtime, "sandbox"); err != nil {
		t.Fatal(err)
	}
	publicData, _ := os.ReadFile(filepath.Join(dir, "public.pem"))
	block, _ := pem.Decode(publicData)
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := identity.NewSandboxVerifier("sandbox", map[string]ed25519.PublicKey{"sandbox-local": parsed.(ed25519.PublicKey)}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	fixtureData, _ := os.ReadFile(filepath.Join(dir, "batches.json"))
	var sources struct {
		Batches []domain.DependencySnapshot `json:"batches"`
	}
	if jsonutil.Decode(fixtureData, &sources) != nil {
		t.Fatal("seed fixture invalid")
	}
	fixture, err := dependency.NewFixture("sandbox", sources.Batches, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := postgres.NewProfiles(runtime, "sandbox")
	drafts := &application.Drafts{Store: store, Dependencies: fixture, Clock: time.Now, PublicWebURL: state.WebURL}
	workflow := &application.Workflow{Store: store, Dependencies: fixture, Clock: time.Now, PublicWebURL: state.WebURL}
	server := httptest.NewServer(httptransport.NewHandler(httptransport.HandlerConfig{Identity: verifier, Drafts: drafts, Workflow: workflow}))
	t.Cleanup(server.Close)
	report, err := sandbox.Smoke(ctx, "sandbox", dir, server.URL)
	if err != nil || report.Outcome != "full_flow_passed" || report.QRURL != state.WebURL+"/trace/"+report.QRCode {
		t.Fatal("runtime smoke failed", err)
	}
	replay, err := sandbox.Smoke(ctx, "sandbox", dir, server.URL)
	if err != nil || replay.Outcome != "existing_terminal_fixture_verified" || replay.ProfileID != report.ProfileID || replay.QRCode != report.QRCode {
		t.Fatal("safe smoke rerun failed", err)
	}
}

func TestSandboxToolsRejectOtherModesAndTargets(t *testing.T) {
	ctx := t.Context()
	dir := filepath.Join(t.TempDir(), "sandbox")
	if _, err := sandbox.Initialize(ctx, "integration", "postgres://localhost/om_traceability_db", dir, "http://localhost:3000", time.Now()); err == nil {
		t.Fatal("integration initialized sandbox")
	}
	if _, err := sandbox.Initialize(ctx, "sandbox", "postgres://localhost/peer_database", dir, "http://localhost:3000", time.Now()); err == nil {
		t.Fatal("peer database accepted")
	}
	if err := sandbox.RefreshTokens("integration", dir, time.Now()); err == nil {
		t.Fatal("integration token minted")
	}
	if _, err := sandbox.Smoke(ctx, "sandbox", dir, "https://remote.example"); err == nil {
		t.Fatal("tokens sent to remote API")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("rejected initializer wrote files")
	}
}
