package httpapi

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJWTStaffAuthenticatorValidatesIdentityTokenAndMapsCareGrants(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const keyID = "care-test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testJWKS(keyID, &privateKey.PublicKey)))
	}))
	defer server.Close()

	authenticator, err := NewJWTStaffAuthenticator(server.URL+"/realms/omama", "omama-api", server.URL+"/jwks")
	require.NoError(t, err)
	actorID := uuid.New()
	token := signedCareTestToken(t, privateKey, keyID, map[string]any{
		"iss": server.URL + "/realms/omama", "sub": uuid.NewString(), "aud": []string{"order-api", "omama-api"},
		"exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Add(-time.Second).Unix(), "typ": "Bearer",
		"user_id": actorID.String(), "principal_type": "EMPLOYEE",
		"permission_grants": []string{"TICKET_QUEUE_VIEW#GLOBAL", "TICKET_REPLY#GLOBAL", "ORDER_VIEW#GLOBAL"},
	})

	actor, err := authenticator.Authenticate(context.Background(), "Bearer "+token)
	require.NoError(t, err)
	require.Equal(t, actorID, actor.ID)
	require.Equal(t, "EMPLOYEE", actor.PrincipalType)
	require.Contains(t, actor.Permissions, "TICKET_QUEUE_VIEW")
	require.Contains(t, actor.Permissions, "TICKET_REPLY")
	require.NotContains(t, actor.Permissions, "ORDER_VIEW")
}

func TestJWTStaffAuthenticatorDoesNotAuthorizeNonGlobalOrCustomerGrants(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const keyID = "care-test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testJWKS(keyID, &privateKey.PublicKey)))
	}))
	defer server.Close()
	authenticator, err := NewJWTStaffAuthenticator(server.URL+"/realms/omama", "omama-api", server.URL+"/jwks")
	require.NoError(t, err)

	claims := map[string]any{
		"iss": server.URL + "/realms/omama", "sub": uuid.NewString(), "aud": "omama-api",
		"exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Add(-time.Second).Unix(), "typ": "Bearer",
		"user_id": uuid.NewString(), "principal_type": "EMPLOYEE",
		"permission_grants": []string{"TICKET_QUEUE_VIEW#WAREHOUSE#WH-HCM-01"},
	}
	token := signedCareTestToken(t, privateKey, keyID, claims)
	actor, err := authenticator.Authenticate(context.Background(), "Bearer "+token)
	require.NoError(t, err)
	require.Empty(t, actor.Permissions)

	claims["principal_type"] = "CUSTOMER"
	claims["permission_grants"] = []string{"TICKET_QUEUE_VIEW#GLOBAL"}
	token = signedCareTestToken(t, privateKey, keyID, claims)
	_, err = authenticator.Authenticate(context.Background(), "Bearer "+token)
	require.ErrorIs(t, err, errStaffUnauthenticated)
}

func TestJWTStaffAuthenticatorRejectsWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const keyID = "care-test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(testJWKS(keyID, &privateKey.PublicKey)))
	}))
	defer server.Close()
	authenticator, err := NewJWTStaffAuthenticator(server.URL+"/realms/omama", "omama-api", server.URL+"/jwks")
	require.NoError(t, err)
	token := signedCareTestToken(t, privateKey, keyID, map[string]any{
		"iss": server.URL + "/realms/omama", "sub": uuid.NewString(), "aud": "order-api",
		"exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Add(-time.Second).Unix(), "typ": "Bearer",
		"user_id": uuid.NewString(), "principal_type": "EMPLOYEE", "permission_grants": []string{"TICKET_QUEUE_VIEW#GLOBAL"},
	})
	_, err = authenticator.Authenticate(context.Background(), "Bearer "+token)
	require.ErrorIs(t, err, errStaffUnauthenticated)
}

func signedCareTestToken(t *testing.T, key *rsa.PrivateKey, keyID string, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": keyID, "typ": "JWT"})
	require.NoError(t, err)
	body, err := json.Marshal(claims)
	require.NoError(t, err)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func testJWKS(keyID string, key *rsa.PublicKey) string {
	modulus := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	exponent := big.NewInt(int64(key.E)).Bytes()
	return `{"keys":[{"kty":"RSA","kid":"` + keyID + `","use":"sig","alg":"RS256","n":"` + modulus + `","e":"` + base64.RawURLEncoding.EncodeToString(exponent) + `"}]}`
}

func TestCanonicalCarePermissionAliases(t *testing.T) {
	for input, expected := range map[string]string{
		" care.tickets.read ": "TICKET_QUEUE_VIEW",
		"care.tickets.claim":  "TICKET_CLAIM",
		"ticket_reply":        "TICKET_REPLY",
	} {
		require.Equal(t, expected, canonicalCarePermission(input))
	}
	require.Equal(t, "TICKET_QUEUE_VIEW", canonicalCarePermission("ticket_queue_view"))
}
