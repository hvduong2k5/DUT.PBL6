// Package sandbox creates synthetic review inputs and ephemeral local credentials.
// It is never imported by the runtime API or enabled in integration mode.
package sandbox

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	"github.com/jackc/pgx/v5"
)

type State struct {
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	Mode      string    `json:"mode"`
	Role      string    `json:"runtime_role"`
	WebURL    string    `json:"web_url"`
	CreatedAt time.Time `json:"created_at"`
}

func Load(dir string) (State, error) {
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		return State{}, errors.New("sandbox state missing; initialize a dedicated directory first")
	}
	var s State
	if jsonutil.Decode(data, &s) != nil || s.Version != 1 || s.Mode != "sandbox" || !domain.ValidUUID(s.ID) || !strings.HasPrefix(s.Role, "trace_runtime_") {
		return State{}, errors.New("invalid sandbox state")
	}
	return s, nil
}

// Initialize never rotates an existing runtime password or overwrites signing
// keys. The manifest is written last, so partial initialization is explicit.
func Initialize(ctx context.Context, mode, migratorURL, dir, webURL string, now time.Time) (State, error) {
	if mode != "sandbox" {
		return State{}, errors.New("explicit sandbox mode required")
	}
	parsed, err := url.Parse(migratorURL)
	if err != nil || migratorURL == "" || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Hostname() == "" || parsed.Path != "/om_traceability_db" {
		return State{}, errors.New("initializer requires the dedicated om_traceability_db migrator URI")
	}
	if _, err = os.Stat(dir); err == nil {
		state, err := Load(dir)
		if err != nil {
			return State{}, err
		}
		existing, err := os.ReadFile(filepath.Join(dir, "runtime-url.txt"))
		if err != nil {
			return State{}, errors.New("existing runtime credential missing")
		}
		runtimeURL, err := url.Parse(strings.TrimSpace(string(existing)))
		if err != nil || runtimeURL.User == nil || runtimeURL.Host != parsed.Host || runtimeURL.Path != parsed.Path || runtimeURL.RawQuery != parsed.RawQuery || runtimeURL.User.Username() != state.Role {
			return State{}, errors.New("existing sandbox targets another database; use its original configuration")
		}
		// Reuse is read-only except refreshing short-lived token files.
		conn, err := pgx.Connect(ctx, runtimeURL.String())
		if err != nil {
			return State{}, errors.New("existing runtime login failed")
		}
		defer conn.Close(ctx)
		if err = migrations.Verify(ctx, conn, "sandbox"); err != nil {
			return State{}, errors.New("existing database mode/schema does not match")
		}
		if err = RefreshTokens(mode, dir, now); err != nil {
			return State{}, err
		}
		return state, nil
	} else if !os.IsNotExist(err) {
		return State{}, errors.New("cannot inspect sandbox directory")
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		return State{}, errors.New("cannot reserve new sandbox directory")
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return State{}, err
	}
	pub, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return State{}, err
	}
	priv, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return State{}, err
	}
	id, err := domain.NewUUID()
	if err != nil {
		return State{}, err
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return State{}, err
	}
	state := State{Version: 1, ID: id, Mode: "sandbox", Role: "trace_runtime_" + strings.ReplaceAll(id, "-", "")[:12], WebURL: webURL, CreatedAt: now.UTC()}
	password := hex.EncodeToString(secret)
	runtime := *parsed
	runtime.User = url.UserPassword(state.Role, password)
	internal := runtime
	internal.Host = "db:5432"
	files := map[string][]byte{
		"public.pem":      pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}),
		"private.pem":     pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: priv}),
		"runtime-url.txt": []byte(runtime.String() + "\n"),
		"compose.env":     []byte("TRACEABILITY_RUNTIME_DATABASE_URL=" + internal.String() + "\nOUTBOX_RELAY_ENABLED=false\nEXPIRY_WORKER_ENABLED=true\n"),
	}
	binding := domain.Binding{ProductID: "44444444-4444-4444-8444-444444444444", SKUCode: "SYNTHETIC-SKU-001", SourceNamespace: "fixture", BatchID: "synthetic-batch-" + state.ID}
	local := now.In(time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	batch := domain.DependencySnapshot{Binding: binding, BatchCode: "SYNTHETIC-LOT-" + state.ID[:8], ProductionDate: today.Format("2006-01-02"), ExpiryDate: today.AddDate(0, 0, 90).Format("2006-01-02"), BestBefore: today.AddDate(0, 0, 91).UTC(), CatalogVersion: "synthetic-catalog-1", BatchVersion: "synthetic-batch-1"}
	source, _ := json.MarshalIndent(map[string]any{"batches": []domain.DependencySnapshot{batch}}, "", "  ")
	files["batches.json"] = source
	certID, err := domain.NewUUID()
	if err != nil {
		return State{}, err
	}
	proofID, err := domain.NewUUID()
	if err != nil {
		return State{}, err
	}
	until := today.AddDate(0, 0, 60).Format("2006-01-02")
	stars := 4
	request := domain.CreateDraftRequest{Binding: binding, Content: domain.Content{ProductName: "[SANDBOX] Synthetic OCOP product", Producer: "Synthetic workshop", WorkshopLocation: "Synthetic location, Hue", Process: "Synthetic review fixture; not externally verified", Ingredients: []domain.Ingredient{{Name: "Synthetic ingredient", Origin: "Synthetic origin"}}, Evidence: []domain.Evidence{{ID: proofID, Reference: "sandbox/evidence/" + state.ID, Kind: "CERTIFICATE", Description: "Synthetic OCOP evidence for local review"}}, Certificates: []domain.Certificate{{ID: certID, Type: "OCOP", Number: "SYNTHETIC-OCOP-" + state.ID[:8], Issuer: "Synthetic authority", Scope: "PRODUCT", ScopeID: binding.ProductID, IssuedOn: today.Format("2006-01-02"), ValidUntil: &until, Stars: &stars, VerificationMethod: "MANUAL", VerifiedAt: now.UTC(), EvidenceIDs: []string{proofID}}}}}
	request.Content = request.Content.Normalize()
	body, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return State{}, err
	}
	files["create.json"] = body
	for name, data := range files {
		if err = writeNew(filepath.Join(dir, name), data); err != nil {
			return State{}, errors.New("sandbox initialization incomplete; preserved files must be inspected before reusing this directory")
		}
		if name == "public.pem" || name == "batches.json" {
			if err = os.Chmod(filepath.Join(dir, name), 0644); err != nil {
				return State{}, errors.New("cannot set public fixture permissions")
			}
		}
	}
	conn, err := pgx.Connect(ctx, migratorURL)
	if err != nil {
		return State{}, errors.New("migrator database connection failed")
	}
	defer conn.Close(ctx)
	if err = migrations.Apply(ctx, conn, "sandbox"); err != nil {
		return State{}, errors.New("sandbox migration failed; existing data was not reset")
	}
	var statement string
	if err = conn.QueryRow(ctx, `SELECT format('CREATE ROLE %I LOGIN NOINHERIT PASSWORD %L',$1::text,$2::text)`, state.Role, password).Scan(&statement); err != nil {
		return State{}, errors.New("runtime role preparation failed")
	}
	if _, err = conn.Exec(ctx, statement); err != nil {
		return State{}, errors.New("runtime role creation failed; preserved credential files were not printed")
	}
	if err = migrations.GrantRuntime(ctx, conn, state.Role); err != nil {
		return State{}, errors.New("runtime privilege grant failed")
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return State{}, err
	}
	if err = writeNew(filepath.Join(dir, "state.json"), data); err != nil {
		return State{}, err
	}
	if err = RefreshTokens(mode, dir, now); err != nil {
		return State{}, err
	}
	return state, nil
}

func writeNew(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func RefreshTokens(mode, dir string, now time.Time) error {
	if mode != "sandbox" {
		return errors.New("token minting is sandbox only")
	}
	if _, err := Load(dir); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(dir, "private.pem"))
	if err != nil {
		return errors.New("sandbox signing key missing")
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" {
		return errors.New("invalid sandbox signing key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return errors.New("invalid sandbox signing key")
	}
	private, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return errors.New("sandbox signing key must be Ed25519")
	}
	roles := []struct {
		file, actor string
		permissions []string
	}{{"editor.jwt", "22222222-2222-4222-8222-222222222222", []string{domain.ReadInternal, domain.Create, domain.Edit, domain.Submit}}, {"approver.jwt", "33333333-3333-4333-8333-333333333333", []string{domain.ReadInternal, domain.ReadEvidence, domain.Approve, domain.Publish, domain.Suspend, domain.Archive}}}
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "sandbox-local"})
	for _, role := range roles {
		claims, _ := json.Marshal(map[string]any{"iss": identity.SandboxIssuer, "aud": identity.SandboxAudience, "sub": role.actor, "iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(), "permissions": role.permissions})
		token := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
		token += "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(token)))
		path := filepath.Join(dir, role.file)
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			return errors.New("token target is not a regular file")
		}
		if err = os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
			return errors.New("cannot save sandbox token")
		}
	}
	return nil
}
