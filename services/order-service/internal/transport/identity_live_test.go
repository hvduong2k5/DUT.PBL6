package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

func TestKeycloakLiveCart(t *testing.T) {
	issuer := os.Getenv("KEYCLOAK_TEST_ISSUER")
	if issuer == "" {
		t.Skip("Real Keycloak/Redis fixture not configured")
	}
	if !strings.HasPrefix(issuer, "http://127.0.0.1:") || !strings.Contains(issuer, "/realms/order-auth-test-") {
		t.Fatal("requires isolated local realm")
	}
	c := app.Config{Mode: "sandbox", AuthMode: "keycloak", OIDCIssuer: issuer, OIDCAudience: "order-api", OIDCAllowLocalHTTP: true, Profile: "http://unused-profile.test", ProfileToken: "unused-profile-credential"}
	v, e := app.NewCustomerIdentity(c)
	if e != nil {
		t.Fatal(e)
	}
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("KEYCLOAK_TEST_REDIS")})
	defer rdb.Close()
	ctx := context.Background()
	for _, id := range []string{"A", "B"} {
		principal := os.Getenv("KEYCLOAK_TEST_PRINCIPAL_" + id)
		p, e := v.Authenticate(ctx, "Bearer "+os.Getenv("ORDER_TEST_TOKEN_"+id))
		if e != nil || p.ID != principal {
			t.Fatal("real token principal mismatch", id, e)
		}
		key := "cart:" + (domain.Principal{ID: principal, Role: "CUSTOMER"}).Scope()
		body, _ := json.Marshal(domain.Cart{Revision: 1, Items: []domain.Item{{SKU: "private-" + id}}})
		if e := rdb.Set(ctx, key, body, 0).Err(); e != nil {
			t.Fatal(e)
		}
		defer rdb.Del(ctx, key)
	}
	s := &app.Service{C: c, Identity: v, Redis: rdb}
	server := httptest.NewServer(Handler(s))
	defer server.Close()
	for _, id := range []string{"A", "B"} {
		r, _ := http.NewRequest("GET", server.URL+"/api/v1/cart", nil)
		r.Header.Set("Authorization", "Bearer "+os.Getenv("ORDER_TEST_TOKEN_"+id))
		r.Header.Set("X-User-ID", os.Getenv("KEYCLOAK_TEST_PRINCIPAL_A"))
		r.Header.Set("X-User-Role", "ADMIN")
		response, e := server.Client().Do(r)
		if e != nil {
			t.Fatal(e)
		}
		var cart domain.Cart
		e = json.NewDecoder(response.Body).Decode(&cart)
		response.Body.Close()
		if e != nil || response.StatusCode != 200 || len(cart.Items) != 1 || cart.Items[0].SKU != "private-"+id {
			t.Fatal("HTTP cart isolation failed", id, response.StatusCode, e)
		}
	}
	parts := strings.Split(os.Getenv("ORDER_TEST_TOKEN_A"), ".")
	if len(parts) != 3 || len(parts[2]) == 0 {
		t.Fatal("fixture token is not a JWT")
	}
	replacement := "A"
	if strings.HasPrefix(parts[2], replacement) {
		replacement = "B"
	}
	parts[2] = replacement + parts[2][1:]
	for _, auth := range []string{"", "Bearer invalid", "Bearer " + strings.Join(parts, "."), "Bearer " + os.Getenv("KEYCLOAK_TEST_NO_ORDER_AUDIENCE_TOKEN")} {
		r, _ := http.NewRequest("GET", server.URL+"/api/v1/cart", nil)
		r.Header.Set("Authorization", auth)
		response, e := server.Client().Do(r)
		if e != nil {
			t.Fatal(e)
		}
		response.Body.Close()
		if response.StatusCode != 401 {
			t.Fatal("invalid token accepted", response.StatusCode)
		}
	}
	response, e := server.Client().Post(server.URL+"/api/v1/guest-sessions", "application/json", nil)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("guest enabled")
	}
}
