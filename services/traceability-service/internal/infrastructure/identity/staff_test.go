package identity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

func TestStaffRequiresLiveGlobalGrant(t *testing.T) {
	const id = "22222222-2222-4222-8222-222222222222"
	const issuer = "https://identity.example/realms/omama"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, audience, scope string
		status                int
		canApprove            bool
		want                  error
	}{
		{"global grant", "omama-api", "GLOBAL", 200, true, nil},
		{"grant revoked", "omama-api", "DEPARTMENT", 200, false, nil},
		{"disabled staff", "omama-api", "GLOBAL", 403, false, domain.ErrForbidden},
		{"Identity offline", "omama-api", "GLOBAL", 503, false, domain.ErrUnavailable},
		{"wrong audience", "other-api", "GLOBAL", 200, false, domain.ErrUnauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			payload, err := json.Marshal(map[string]any{
				"iss": issuer, "aud": tc.audience, "sub": "staff-1", "user_id": id, "typ": "Bearer",
				"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
				"permission_grants": []string{domain.Approve + "#GLOBAL", domain.Publish + "#WAREHOUSE#A"},
			})
			if err != nil {
				t.Fatal(err)
			}
			signed, err := signer.Sign(payload)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := signed.CompactSerialize()
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/users/me" || r.Header.Get("Authorization") != "Bearer "+raw {
					t.Error("incorrect Identity request")
				}
				w.WriteHeader(tc.status)
				if tc.status == 200 {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
						"principalId": id, "idpSubject": "staff-1", "principalType": "EMPLOYEE", "provisioningStatus": "ACTIVE",
						"permissionGrants": []map[string]string{{"permissionCode": domain.Approve, "scopeType": tc.scope}},
					}})
				}
			}))
			defer server.Close()
			staff := Staff{Tokens: oidc.NewVerifier(issuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "omama-api"}), IdentityURL: server.URL, HTTP: server.Client()}
			principal, err := staff.Verify(context.Background(), raw)
			if !errors.Is(err, tc.want) || principal.Can(domain.Approve) != tc.canApprove || principal.Can(domain.Publish) {
				t.Fatalf("unexpected authorization result: %+v, %v", principal, err)
			}
		})
	}
}
