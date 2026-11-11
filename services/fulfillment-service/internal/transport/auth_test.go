package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"testing"
	"time"
)

func signClaims(t *testing.T, key *rsa.PrivateKey, claims Claims) string {
	t.Helper()
	token, e := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if e != nil {
		t.Fatal(e)
	}
	return token
}

func TestStaffGrantsAuthorization(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	id := "e5f9ea69-bdc3-4ee7-9076-96c327335410"
	base := func(grants []string) Claims {
		return Claims{UserID: id, PermissionGrants: grants, RegisteredClaims: jwt.RegisteredClaims{Issuer: "https://issuer.example", Subject: "verified-sub", Audience: jwt.ClaimStrings{"omama-api"}, IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}}
	}
	parity := []string{
		"FULFILLMENT_TASK_READ#GLOBAL", "FULFILLMENT_TASK_READ_HISTORY#GLOBAL",
		"FULFILLMENT_TASK_ASSIGN_SELF#GLOBAL", "FULFILLMENT_TASK_WORK#GLOBAL",
		"FULFILLMENT_TASK_SEAL#GLOBAL", "FULFILLMENT_TASK_BLOCK#GLOBAL",
		"FULFILLMENT_TASK_RESOLVE_BLOCK#GLOBAL", "FULFILLMENT_MEDIA_UPLOAD#GLOBAL",
		"FULFILLMENT_MEDIA_READ_METADATA#GLOBAL", "FULFILLMENT_MEDIA_VIEW#GLOBAL",
	}
	auth := Auth{Key: &key.PublicKey, Issuer: "https://issuer.example"}

	t.Run("global parity grants", func(t *testing.T) {
		p, e := auth.Verify("Bearer "+signClaims(t, key, base(parity)), "omama-api")
		if e != nil {
			t.Fatal(e)
		}
		if !p.Global || !p.Can("FULFILLMENT_TASK_SEAL", "any-warehouse") || p.Can("FULFILLMENT_TASK_ASSIGN_OTHER", "any-warehouse") {
			t.Fatal("global parity mapping incorrect")
		}
	})
	t.Run("warehouse scoped grants", func(t *testing.T) {
		p, e := auth.Verify("Bearer "+signClaims(t, key, base([]string{"FULFILLMENT_TASK_READ#WAREHOUSE#w1", "FULFILLMENT_TASK_SEAL#WAREHOUSE#w1"})), "omama-api")
		if e != nil {
			t.Fatal(e)
		}
		if p.Global || !p.Can("FULFILLMENT_TASK_SEAL", "w1") || p.Can("FULFILLMENT_TASK_SEAL", "other") || p.Can("FULFILLMENT_TASK_WORK", "w1") {
			t.Fatal("warehouse scope mapping incorrect")
		}
	})
	t.Run("scopes collapse like the previous role mapping", func(t *testing.T) {
		p, e := auth.Verify("Bearer "+signClaims(t, key, base([]string{"FULFILLMENT_TASK_READ#WAREHOUSE#w1", "FULFILLMENT_TASK_SEAL#WAREHOUSE#w2"})), "omama-api")
		if e != nil {
			t.Fatal(e)
		}
		if !p.Can("FULFILLMENT_TASK_SEAL", "w1") || !p.Can("FULFILLMENT_TASK_READ", "w2") {
			t.Fatal("grant scopes were kept per-permission instead of collapsing")
		}
	})
	t.Run("assign other only when granted", func(t *testing.T) {
		p, e := auth.Verify("Bearer "+signClaims(t, key, base([]string{"FULFILLMENT_TASK_ASSIGN_OTHER#GLOBAL"})), "omama-api")
		if e != nil || !p.Can("FULFILLMENT_TASK_ASSIGN_OTHER", "w1") {
			t.Fatal("explicit assign-other grant rejected")
		}
	})
	t.Run("non fulfillment grants do not open staff access", func(t *testing.T) {
		if _, e := auth.Verify("Bearer "+signClaims(t, key, base([]string{"ORDER_VIEW#GLOBAL", "CATALOG_VIEW#GLOBAL"})), "omama-api"); e == nil {
			t.Fatal("foreign grants allowed")
		}
	})
	t.Run("empty grants claim is staff without fulfillment access", func(t *testing.T) {
		if _, e := auth.Verify("Bearer "+signClaims(t, key, base([]string{})), "omama-api"); e == nil {
			t.Fatal("empty grants allowed")
		}
	})
	t.Run("missing grants claim is not a staff token", func(t *testing.T) {
		if _, e := auth.Verify("Bearer "+signClaims(t, key, base(nil)), "omama-api"); e == nil {
			t.Fatal("claimless token allowed")
		}
	})
	t.Run("forged legacy claims are ignored", func(t *testing.T) {
		forged := jwt.MapClaims{
			"user_id": id, "sub": "verified-sub", "iss": "https://issuer.example",
			"aud": "omama-api", "iat": jwt.NewNumericDate(time.Now().Add(-time.Minute)).Unix(),
			"exp":          jwt.NewNumericDate(time.Now().Add(time.Minute)).Unix(),
			"permissions":  []string{"FULFILLMENT_TASK_SEAL"},
			"warehouse_ids": []string{"w1"},
		}
		token, e := jwt.NewWithClaims(jwt.SigningMethodRS256, forged).SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := auth.Verify("Bearer "+token, "omama-api"); e == nil {
			t.Fatal("forged flat claims allowed")
		}
	})
	for _, bad := range []string{
		"FULFILLMENT_TASK_SEAL", "FULFILLMENT_TASK_SEAL#GLOBAL#extra",
		"FULFILLMENT_TASK_SEAL#BOGUS", "FULFILLMENT_TASK_SEAL#WAREHOUSE",
		"FULFILLMENT_TASK_SEAL#WAREHOUSE#", "FULFILLMENT_TASK_SEAL#WAREHOUSE# w1",
		"#GLOBAL", "", "FULFILLMENT_TASK_SEAL#warehouse#w1",
	} {
		t.Run("malformed grant "+bad, func(t *testing.T) {
			if _, e := auth.Verify("Bearer "+signClaims(t, key, base([]string{bad})), "omama-api"); e == nil {
				t.Fatal("malformed grant allowed")
			}
		})
	}
}

func TestInternalReadCredentialIsSeparateFromStaff(t *testing.T) {
	auth := Auth{InternalTokens: map[string]string{"order-service": strings.Repeat("o", 40)}}
	p, e := auth.Verify("Bearer "+strings.Repeat("o", 40), "fulfillment-internal")
	if e != nil || p.Service != "order-service" || p.Can("FULFILLMENT_TASK_SEAL", "w1") {
		t.Fatal("internal read became staff")
	}
	for _, header := range []string{"", "Bearer wrong", "Bearer " + strings.Repeat("i", 40)} {
		if _, e = auth.Verify(header, "fulfillment-internal"); e == nil {
			t.Fatal("wrong peer allowed")
		}
	}
	if _, e = auth.Verify("Bearer "+strings.Repeat("o", 40), "omama-api"); e == nil {
		t.Fatal("internal credential entered staff API")
	}
}
