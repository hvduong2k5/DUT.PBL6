package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/migrations"
	"github.com/redis/go-redis/v9"
)

func guestService(t *testing.T) *app.Service {
	t.Helper()
	dsn := os.Getenv("ORDER_CORE_TEST_DATABASE")
	if dsn == "" {
		t.Skip("ORDER_CORE_TEST_DATABASE required")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "guest_f53_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		db.Close()
		admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		admin.Close()
	})
	if e = migrations.Apply(ctx, db); e != nil {
		t.Fatal(e)
	}
	cipher, e := persistence.NewCipher("test", map[string][]byte{"test": bytes.Repeat([]byte{7}, 32)})
	if e != nil {
		t.Fatal(e)
	}
	store, e := persistence.NewStore(db, cipher, bytes.Repeat([]byte{8}, 32))
	if e != nil {
		t.Fatal(e)
	}
	rdb := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: rdb.Addr()})
	t.Cleanup(func() { client.Close() })
	return &app.Service{C: app.Config{AuthMode: "keycloak", GuestSessionIssuanceEnabled: true, GuestSessionSecret: strings.Repeat("g", 32), JWTSecret: strings.Repeat("s", 32), GuestIssuancePerHour: 10}, DB: db, Redis: client, GuestStore: &persistence.Payment{Store: store}}
}

func TestGuestTrustBoundaryAUTH11Through16(t *testing.T) {
	s := guestService(t)
	h := Handler(s)
	request := func(method, path, body, auth string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.53:1234"
		r.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	count := func() int {
		var n int
		if e := s.DB.QueryRow(context.Background(), "SELECT count(*) FROM guest_checkout_sessions").Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	for _, body := range []string{`{"role":"CUSTOMER"}`, `{"scope":"payment:write"}`, `{"session_id":"chosen"}`, `{"expires_in":99999}`, `{"iss":"idp"}`, `{"aud":"order-api"}`, `null`} {
		w := request("POST", "/api/v1/guest-sessions", body, "")
		if w.Code != 400 || count() != 0 {
			t.Fatalf("AUTH-12 %s: %d %s", body, w.Code, w.Body)
		}
	}
	s.C.GuestSessionIssuanceEnabled = false
	if w := request("POST", "/api/v1/guest-sessions", "{}", ""); w.Code != 403 || count() != 0 {
		t.Fatal("AUTH-14", w.Code, w.Body)
	}
	s.C.GuestSessionIssuanceEnabled = true
	w := request("POST", "/api/v1/guest-sessions", "{}", "")
	if w.Code != 201 || count() != 1 {
		t.Fatal("AUTH-11", w.Code, w.Body)
	}
	var session struct {
		Token string `json:"access_token"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &session); e != nil {
		t.Fatal(e)
	}
	if w = request("GET", "/api/v1/cart", "", "Bearer "+session.Token); w.Code != 200 {
		t.Fatal("guest cart with no Identity/Profile", w.Code, w.Body)
	}
	if w = request("GET", "/api/v1/orders", "", "Bearer "+session.Token); w.Code != 403 {
		t.Fatal("AUTH-13", w.Code, w.Body)
	}
	if w = request("POST", "/api/v1/admin/payments/connections", "{}", "Bearer "+session.Token); w.Code != 403 && w.Code != 404 {
		t.Fatal("guest reached staff path", w.Code, w.Body)
	}
	now := time.Now()
	sandbox := app.SignToken(s.C.GuestSessionSecret, app.Claims{Subject: uuid.NewString(), Role: "GUEST", Issuer: "order-sandbox", Audience: "order-api", Issued: now.Unix(), Expires: now.Add(time.Hour).Unix()})
	if _, e := s.AuthenticateGuestSession(context.Background(), "Bearer "+sandbox, true); e == nil {
		t.Fatal("AUTH-16 customer-shaped JWT accepted by guest validator")
	}
	if _, e := s.AuthenticateContext(context.Background(), "Bearer "+strings.Replace(session.Token, ".", ".x", 1)); e == nil {
		t.Fatal("AUTH-16 invalid guest fell back")
	}
	if _, e := (&app.Service{C: s.C}).AuthenticateContext(context.Background(), "Bearer "+sandbox); e == nil {
		t.Fatal("AUTH-16 sandbox accepted in Keycloak mode")
	}
	for i := 1; i < 10; i++ {
		if w = request("POST", "/api/v1/guest-sessions", "{}", ""); w.Code != 201 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if w = request("POST", "/api/v1/guest-sessions", "{}", ""); w.Code != 429 || count() != 10 {
		t.Fatal("AUTH-15", w.Code, count())
	}
}

func TestGuestQuotaUsesOnlyTrustedProxyChain(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/guest-sessions", nil)
	r.RemoteAddr = "10.0.0.2:80"
	r.Header.Set("X-Forwarded-For", "192.0.2.99, 198.51.100.5, 10.0.0.3")
	if got := guestClientIP(r, nil); got != "10.0.0.2" {
		t.Fatal("untrusted peer spoofed address", got)
	}
	if got := guestClientIP(r, []string{"10.0.0.0/24"}); got != "198.51.100.5" {
		t.Fatal("wrong trusted hop", got)
	}
}

func TestCustomerPaymentWriteDoesNotSwitchToGuestCookieSTEP07(t *testing.T) {
	s := guestService(t)
	// dev's finance write protocol is sandbox staff-only. A Customer JWT is
	// valid authentication but has no write authority, with or without a cookie.
	s.C.AuthMode = "sandbox"
	s.Payment = s.GuestStore
	now := time.Now()
	token := app.SignToken(s.C.JWTSecret, app.Claims{Subject: uuid.NewString(), Role: "CUSTOMER", Issuer: "order-sandbox", Audience: "order-api", Issued: now.Unix(), Expires: now.Add(time.Hour).Unix()})
	for _, cookie := range []string{"", "oma_guest_session=valid-guest-session"} {
		r := httptest.NewRequest("POST", "/api/v1/admin/payments/refunds/"+uuid.NewString()+"/start", strings.NewReader(`{"reason":"test"}`))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		Handler(s).ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("customer write switched credentials: %d %s", w.Code, w.Body)
		}
	}
}
