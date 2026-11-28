package http_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	transport "github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/middleware"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/tests/testhelper"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// Test identity injection never exists in the production executable.
type verifier struct {
	identity domain.Identity
	err      error
}

func (v verifier) Verify(*http.Request) (domain.Identity, error) { return v.identity, v.err }

func router(repo domain.Repository, v middleware.Verifier, ready func(context.Context) error) http.Handler {
	log := zerolog.New(io.Discard)
	var store domain.IdempotencyStore
	if s, ok := repo.(domain.IdempotencyStore); ok {
		store = s
	}
	return transport.NewRouter(transport.RouterConfig{Handler: handlers.New(usecase.New(repo), log, handlers.DefaultMaxBody), Verifier: v, Idempotency: store, Ready: ready, Logger: log})
}
func request(r http.Handler, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for n := 0; n+1 < len(headers); n += 2 {
		req.Header.Set(headers[n], headers[n+1])
	}
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	return out
}

// envelope asserts the flat contract error body and that its requestId matches
// the X-Request-ID response header.
func envelope(t *testing.T, out *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, out.Code, out.Body.String())
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &raw))
	require.NotContains(t, raw, "error", "nested error object is not the contract")
	var e handlers.ErrorBody
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &e))
	require.Equal(t, code, e.Code)
	require.NotEmpty(t, e.Message)
	require.NotEmpty(t, e.RequestID)
	require.Equal(t, out.Header().Get("X-Request-ID"), e.RequestID)
	require.Equal(t, "no-store", out.Header().Get("Cache-Control"))
}

func TestAdminFailClosedAndForgedHeaders(t *testing.T) {
	r := router(testhelper.NewMemory(), nil, nil)
	for _, method := range []string{"GET", "POST"} {
		req := httptest.NewRequest(method, "/api/v1/admin/articles", nil)
		req.Header.Set("X-User-ID", uuid.NewString())
		req.Header.Set("X-User-Role", "ADMIN")
		req.Header.Set("X-Internal-Token", "anything")
		req.Header.Set("Authorization", "Bearer fake")
		out := httptest.NewRecorder()
		r.ServeHTTP(out, req)
		envelope(t, out, 401, "UNAUTHENTICATED")
	}
	for _, v := range []verifier{{err: errors.New("bad token")}, {identity: domain.Identity{Grants: testhelper.Full().Grants}}} {
		out := request(router(testhelper.NewMemory(), v, nil), "GET", "/api/v1/admin/articles", "")
		envelope(t, out, 401, "UNAUTHENTICATED")
	}
	// A verified principal without grants (a customer, or an admin role holder
	// with no CONTENT_* grant) is forbidden, not unauthenticated.
	out := request(router(testhelper.NewMemory(), verifier{identity: testhelper.Customer()}, nil), "GET", "/api/v1/admin/articles", "")
	envelope(t, out, 403, "FORBIDDEN")
	out = request(router(testhelper.NewMemory(), verifier{err: domain.ErrForbidden}, nil), "GET", "/api/v1/admin/articles", "")
	envelope(t, out, 403, "FORBIDDEN")
}

func TestHTTPWorkflowAndErrors(t *testing.T) {
	repo := testhelper.NewMemory()
	r := router(repo, verifier{identity: testhelper.Full()}, func(context.Context) error { return nil })
	raw, _ := json.Marshal(testhelper.Content("http-story"))
	out := request(r, "POST", "/api/v1/admin/articles", string(raw))
	require.Equal(t, 201, out.Code, out.Body.String())
	require.NotEmpty(t, out.Header().Get("ETag"))
	require.NotEmpty(t, out.Header().Get("Location"))
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	envelope(t, request(r, "GET", "/api/v1/articles/http-story", ""), 404, "NOT_FOUND")
	envelope(t, request(r, "POST", "/api/v1/admin/articles", string(raw)), 409, "DUPLICATE_SLUG")
	envelope(t, request(r, "GET", "/api/v1/admin/articles/"+uuid.NewString(), ""), 404, "NOT_FOUND")
	envelope(t, request(r, "GET", "/api/v1/admin/articles/bad-id", ""), 400, "BAD_REQUEST")
	envelope(t, request(r, "POST", "/api/v1/admin/articles", `{"title":`), 400, "BAD_REQUEST")
	envelope(t, request(r, "POST", "/api/v1/admin/articles", `{"unknown":true}`), 400, "BAD_REQUEST")
	envelope(t, request(r, "POST", "/api/v1/admin/articles", string(raw)+` {}`), 400, "BAD_REQUEST")
	envelope(t, request(r, "POST", "/api/v1/admin/articles", `{}`), 422, "VALIDATION_ERROR")
	envelope(t, request(r, "POST", path+"/publish", `{"version":1}`), 409, "INVALID_STATE")
	envelope(t, request(r, "POST", path+"/approve", `{"version":1}`), 409, "INVALID_STATE")
	envelope(t, request(r, "DELETE", path, ""), 405, "METHOD_NOT_ALLOWED")
	envelope(t, request(r, "GET", "/nope", ""), 404, "NOT_FOUND")
	for _, action := range []string{"submit", "approve", "publish"} {
		body, _ := json.Marshal(map[string]any{"version": a.Version})
		out = request(r, "POST", path+"/"+action, string(body))
		require.Equal(t, 200, out.Code, out.Body.String())
		require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	}
	body, _ := json.Marshal(map[string]any{"version": a.Version, "body": "private-new-draft", "slug": "new-http-story"})
	out = request(r, "PATCH", path, string(body))
	require.Equal(t, 200, out.Code, out.Body.String())
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	pub := request(r, "GET", "/api/v1/articles/http-story", "")
	require.Equal(t, 200, pub.Code)
	require.NotContains(t, pub.Body.String(), "private-new-draft")
	require.NotContains(t, pub.Body.String(), "created_by")
	require.NotContains(t, pub.Body.String(), "current_revision")
	list := request(r, "GET", "/api/v1/articles?q=original&category=heritage&page=1&limit=1", "")
	require.Equal(t, 200, list.Code)
	require.Contains(t, list.Body.String(), `"total":1`)
	require.NotContains(t, list.Body.String(), "private-new-draft")
	require.Equal(t, 404, request(r, "GET", "/api/v1/articles/new-http-story", "").Code)
	envelope(t, request(r, "PATCH", path, string(body)), 409, "VERSION_CONFLICT")
	body, _ = json.Marshal(map[string]any{"version": a.Version})
	out = request(r, "POST", path+"/hide", string(body))
	require.Equal(t, 200, out.Code)
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	require.False(t, a.Visible)
	require.Equal(t, 404, request(r, "GET", "/api/v1/articles/http-story", "").Code)
	// Hidden content can be restored without a second review.
	body, _ = json.Marshal(map[string]any{"version": a.Version})
	envelope(t, request(r, "POST", path+"/hide", string(body)), 409, "INVALID_STATE")
	out = request(r, "POST", path+"/restore", string(body))
	require.Equal(t, 200, out.Code, out.Body.String())
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	require.True(t, a.Visible)
	require.Equal(t, 200, request(r, "GET", "/api/v1/articles/http-story", "").Code)
	var actions []string
	for _, e := range a.Audit {
		actions = append(actions, e.Action)
	}
	require.Equal(t, []string{"create", "submit", "approve", "publish", "edit", "hide", "restore"}, actions)
	for _, path := range []string{"/api/v1/articles?page=x", "/api/v1/articles?page=1&page=2"} {
		envelope(t, request(r, "GET", path, ""), 400, "BAD_REQUEST")
	}
	for _, path := range []string{"/api/v1/articles?page=0", "/api/v1/articles?limit=101", "/api/v1/articles?status=DRAFT"} {
		envelope(t, request(r, "GET", path, ""), 422, "VALIDATION_ERROR")
	}
	require.Equal(t, 200, request(r, "GET", "/api/v1/admin/articles?status=DRAFT", "").Code)
}

func TestApprovalRequiresApproverAndRejectReason(t *testing.T) {
	repo := testhelper.NewMemory()
	admin := router(repo, verifier{identity: testhelper.Full()}, nil)
	editor := router(repo, verifier{identity: testhelper.Editor()}, nil)
	raw, _ := json.Marshal(testhelper.Content("review"))
	out := request(admin, "POST", "/api/v1/admin/articles", string(raw))
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	require.Equal(t, 200, request(editor, "POST", path+"/submit", `{"version":1}`).Code)
	for _, action := range []string{"approve", "reject"} {
		envelope(t, request(editor, "POST", path+"/"+action, `{"version":2,"reason":"sources"}`), 403, "FORBIDDEN")
	}
	envelope(t, request(admin, "POST", path+"/reject", `{"version":2}`), 422, "VALIDATION_ERROR")
	out = request(admin, "POST", path+"/reject", `{"version":2,"reason":"sources required"}`)
	require.Equal(t, 200, out.Code)
	require.Contains(t, out.Body.String(), "sources required")
}

// Authorization is per permission: an actor holding only the reject grant cannot
// approve, and the ADMIN role name carries no authority at all.
func TestEachActionNeedsItsOwnPermission(t *testing.T) {
	for _, tc := range []struct {
		action string
		grant  domain.Permission
	}{
		{"submit", domain.PermSubmit}, {"approve", domain.PermApprove}, {"reject", domain.PermReject},
		{"publish", domain.PermPublish}, {"hide", domain.PermHide}, {"restore", domain.PermPublish},
	} {
		t.Run(tc.action, func(t *testing.T) {
			repo := testhelper.NewMemory()
			a, err := usecase.New(repo).Create(context.Background(), testhelper.Content("per-action"), testhelper.Full())
			require.NoError(t, err)
			path := "/api/v1/admin/articles/" + a.ID.String() + "/" + tc.action
			others := []domain.Permission{domain.PermView, domain.PermCreate, domain.PermEdit, domain.PermSubmit, domain.PermApprove, domain.PermReject, domain.PermPublish, domain.PermHide}
			var without []domain.Permission
			for _, p := range others {
				if p != tc.grant {
					without = append(without, p)
				}
			}
			out := request(router(repo, verifier{identity: testhelper.Actor(without...)}, nil), "POST", path, `{"version":1,"reason":"r"}`)
			envelope(t, out, 403, "FORBIDDEN")
		})
	}
}

func TestPublishPermissionsAndStates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		verifier middleware.Verifier
		want     int
	}{
		{"full access", verifier{identity: testhelper.Full()}, 200},
		{"approver", verifier{identity: testhelper.Approver()}, 200},
		{"editor", verifier{identity: testhelper.Editor()}, 403},
		{"no grants", verifier{identity: testhelper.Customer()}, 403},
		{"missing verifier", nil, 401},
		{"invalid token", verifier{err: errors.New("invalid token")}, 401},
		{"invalid identity", verifier{identity: domain.Identity{Grants: testhelper.Full().Grants}}, 401},
	} {
		for _, state := range []domain.Status{domain.Draft, domain.Pending, domain.Approved, domain.Rejected, domain.Published} {
			t.Run(tc.name+"/"+string(state), func(t *testing.T) {
				ctx := context.Background()
				repo := testhelper.NewMemory()
				s := usecase.New(repo)
				admin := testhelper.Full()
				a, err := s.Create(ctx, testhelper.Content("publish-rbac"), admin)
				require.NoError(t, err)
				steps := map[domain.Status][]string{
					domain.Pending:   {"submit"},
					domain.Approved:  {"submit", "approve"},
					domain.Rejected:  {"submit", "reject"},
					domain.Published: {"submit", "approve", "publish"},
				}
				for _, action := range steps[state] {
					a, err = s.Transition(ctx, a.ID, a.Version, action, "review reason", admin)
					require.NoError(t, err)
				}
				before, err := repo.Get(ctx, a.ID)
				require.NoError(t, err)
				body, err := json.Marshal(map[string]any{"version": a.Version})
				require.NoError(t, err)
				path := "/api/v1/admin/articles/" + a.ID.String() + "/publish"
				req := httptest.NewRequest("POST", path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", admin.ID.String())
				req.Header.Set("X-User-Role", "ADMIN")
				req.Header.Set("Authorization", "Bearer forged")
				r := router(repo, tc.verifier, nil)
				out := httptest.NewRecorder()
				r.ServeHTTP(out, req)
				want := tc.want
				if want == 200 && state != domain.Approved {
					want = 409
				}
				require.Equal(t, want, out.Code, out.Body.String())
				after, err := repo.Get(ctx, a.ID)
				require.NoError(t, err)
				if want != 200 {
					require.Equal(t, before, after, "denied publication must preserve the entire article")
					if want == 409 {
						require.Contains(t, out.Body.String(), "INVALID_STATE")
					}
					return
				}
				require.Equal(t, before.Version+1, after.Version)
				require.Equal(t, domain.Published, after.Current().Status)
				require.Equal(t, &after.CurrentRevisionID, after.PublishedRevisionID)
				require.Len(t, after.Revisions, len(before.Revisions))
				out = request(r, "POST", path, string(body))
				require.Equal(t, 409, out.Code)
				require.Contains(t, out.Body.String(), "VERSION_CONFLICT")
			})
		}
	}
}

type brokenRepo struct{ domain.Repository }

func (b brokenRepo) GetPublished(context.Context, string) (*domain.Article, error) {
	return nil, errors.New("private database DSN and stack trace")
}

type panicRepo struct{ domain.Repository }

func (panicRepo) GetPublished(context.Context, string) (*domain.Article, error) {
	panic("boom: private details")
}
func TestHealthAndInternalErrorSanitization(t *testing.T) {
	r := router(brokenRepo{testhelper.NewMemory()}, nil, func(context.Context) error { return errors.New("offline") })
	require.Equal(t, 200, request(r, "GET", "/livez", "").Code)
	for _, path := range []string{"/readyz", "/healthz"} {
		out := request(r, "GET", path, "")
		require.Equal(t, 503, out.Code)
		require.Contains(t, out.Body.String(), `"database":false`)
		var e handlers.ErrorBody
		require.NoError(t, json.Unmarshal(out.Body.Bytes(), &e))
		require.Equal(t, "NOT_READY", e.Code)
		require.Equal(t, out.Header().Get("X-Request-ID"), e.RequestID)
	}
	out := request(r, "GET", "/api/v1/articles/test", "")
	envelope(t, out, 500, "INTERNAL_ERROR")
	require.NotContains(t, out.Body.String(), "DSN")
	require.NotContains(t, out.Body.String(), "stack trace")
	out = request(router(panicRepo{testhelper.NewMemory()}, nil, nil), "GET", "/api/v1/articles/test", "")
	envelope(t, out, 500, "INTERNAL_ERROR")
	require.NotContains(t, out.Body.String(), "boom")
	r = router(testhelper.NewMemory(), nil, func(context.Context) error { return nil })
	for _, path := range []string{"/readyz", "/healthz"} {
		require.Equal(t, 200, request(r, "GET", path, "").Code)
	}
}

func TestPublicCachingAndAdminNoStore(t *testing.T) {
	repo := testhelper.NewMemory()
	r := router(repo, verifier{identity: testhelper.Full()}, nil)
	raw, _ := json.Marshal(testhelper.Content("cached"))
	out := request(r, "POST", "/api/v1/admin/articles", string(raw))
	require.Equal(t, "no-store", out.Header().Get("Cache-Control"))
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	for _, action := range []string{"submit", "approve", "publish"} {
		body, _ := json.Marshal(map[string]any{"version": a.Version})
		out = request(r, "POST", path+"/"+action, string(body))
		require.Equal(t, 200, out.Code)
		require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	}
	require.Equal(t, "no-store", request(r, "GET", path, "").Header().Get("Cache-Control"))
	for _, url := range []string{"/api/v1/articles/cached", "/api/v1/articles"} {
		first := request(r, "GET", url, "")
		require.Equal(t, 200, first.Code)
		require.Equal(t, "public, max-age=60, s-maxage=300", first.Header().Get("Cache-Control"))
		etag := first.Header().Get("ETag")
		require.NotEmpty(t, etag)
		second := request(r, "GET", url, "", "If-None-Match", etag)
		require.Equal(t, 304, second.Code)
		require.Empty(t, second.Body.String())
		require.Equal(t, etag, second.Header().Get("ETag"))
		require.Equal(t, 200, request(r, "GET", url, "", "If-None-Match", `"stale"`).Code)
	}
	// Errors on public routes must never be cached.
	require.Equal(t, "no-store", request(r, "GET", "/api/v1/articles/missing", "").Header().Get("Cache-Control"))
}

func TestIdempotencyKey(t *testing.T) {
	repo := testhelper.NewMemory()
	r := router(repo, verifier{identity: testhelper.Full()}, nil)
	raw, _ := json.Marshal(testhelper.Content("idem"))
	first := request(r, "POST", "/api/v1/admin/articles", string(raw), "Idempotency-Key", "key-1")
	require.Equal(t, 201, first.Code, first.Body.String())
	replay := request(r, "POST", "/api/v1/admin/articles", string(raw), "Idempotency-Key", "key-1")
	require.Equal(t, 201, replay.Code)
	require.Equal(t, "true", replay.Header().Get("Idempotent-Replayed"))
	require.Equal(t, first.Body.String(), replay.Body.String())
	require.Equal(t, first.Header().Get("Location"), replay.Header().Get("Location"))
	items, total, err := repo.List(context.Background(), domain.Filter{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total, "a replay must not create a second article")
	require.Len(t, items, 1)
	other, _ := json.Marshal(testhelper.Content("idem-other"))
	envelope(t, request(r, "POST", "/api/v1/admin/articles", string(other), "Idempotency-Key", "key-1"), 409, "IDEMPOTENCY_KEY_REUSED")
	// Keys are scoped to the principal: another user's same key is independent.
	stranger := router(repo, verifier{identity: testhelper.Full()}, nil)
	require.Equal(t, 201, request(stranger, "POST", "/api/v1/admin/articles", string(other), "Idempotency-Key", "key-1").Code)
	// Without a key the request is not deduplicated.
	envelope(t, request(r, "POST", "/api/v1/admin/articles", string(raw)), 409, "DUPLICATE_SLUG")
	envelope(t, request(r, "POST", "/api/v1/admin/articles", string(raw), "Idempotency-Key", "bad key with spaces"), 400, "BAD_REQUEST")
	// A failed first attempt is replayed too, so the client sees one outcome.
	failing := request(r, "POST", "/api/v1/admin/articles", `{}`, "Idempotency-Key", "key-422")
	envelope(t, failing, 422, "VALIDATION_ERROR")
	again := request(r, "POST", "/api/v1/admin/articles", `{}`, "Idempotency-Key", "key-422")
	require.Equal(t, 422, again.Code)
	require.Equal(t, "true", again.Header().Get("Idempotent-Replayed"))
	// Actions are covered as well.
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &a))
	submit := "/api/v1/admin/articles/" + a.ID.String() + "/submit"
	one := request(r, "POST", submit, `{"version":1}`, "Idempotency-Key", "submit-1")
	two := request(r, "POST", submit, `{"version":1}`, "Idempotency-Key", "submit-1")
	require.Equal(t, 200, one.Code)
	require.Equal(t, 200, two.Code, "the retry must replay 200, not fail on the advanced version")
	require.Equal(t, one.Body.String(), two.Body.String())
	got, err := repo.Get(context.Background(), a.ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, got.Version)
}

// Real signature verification, claims and the full publication flow with no
// injected identity.
func signingKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	return key, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}
func token(t *testing.T, key *rsa.PrivateKey, mutate func(jwt.MapClaims)) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{"iss": "https://identity.test/realms/omama", "aud": "omama-api", "sub": "idp-subject",
		"user_id": uuid.NewString(), "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix()}
	if mutate != nil {
		mutate(claims)
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	require.NoError(t, err)
	return signed
}
func TestJWTVerifierEndToEnd(t *testing.T) {
	key, publicPEM := signingKey(t)
	v, err := middleware.NewJWTVerifier(publicPEM, "https://identity.test/realms/omama", "omama-api")
	require.NoError(t, err)
	repo := testhelper.NewMemory()
	r := router(repo, v, nil)
	grants := func(c jwt.MapClaims) {
		c["permission_grants"] = []string{"CONTENT_ARTICLE_VIEW#GLOBAL", "CONTENT_ARTICLE_CREATE#GLOBAL", "CONTENT_ARTICLE_EDIT#GLOBAL", "CONTENT_ARTICLE_SUBMIT#GLOBAL",
			"CONTENT_ARTICLE_APPROVE#GLOBAL", "CONTENT_ARTICLE_PUBLISH#GLOBAL", "ORDER_VIEW#WAREHOUSE#HUE"}
	}
	bearer := func(tok string) string { return "Bearer " + tok }
	staff := bearer(token(t, key, grants))
	raw, _ := json.Marshal(testhelper.Content("jwt-story"))
	out := request(r, "POST", "/api/v1/admin/articles", string(raw), "Authorization", staff)
	require.Equal(t, 201, out.Code, out.Body.String())
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	envelope(t, request(r, "GET", "/api/v1/articles/jwt-story", ""), 404, "NOT_FOUND")
	for _, action := range []string{"submit", "approve", "publish"} {
		body, _ := json.Marshal(map[string]any{"version": a.Version})
		out = request(r, "POST", path+"/"+action, string(body), "Authorization", staff)
		require.Equal(t, 200, out.Code, out.Body.String())
		require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	}
	require.Equal(t, 200, request(r, "GET", "/api/v1/articles/jwt-story", "").Code)

	other, other2 := signingKey(t)
	_ = other2
	expired := func(c jwt.MapClaims) { grants(c); c["exp"] = time.Now().Add(-time.Hour).Unix() }
	for name, tc := range map[string]struct {
		header string
		status int
		code   string
	}{
		"no header":      {"", 401, "UNAUTHENTICATED"},
		"garbage":        {"Bearer not-a-jwt", 401, "UNAUTHENTICATED"},
		"basic scheme":   {"Basic " + token(t, key, grants), 401, "UNAUTHENTICATED"},
		"wrong key":      {bearer(token(t, other, grants)), 401, "UNAUTHENTICATED"},
		"expired":        {bearer(token(t, key, expired)), 401, "UNAUTHENTICATED"},
		"wrong issuer":   {bearer(token(t, key, func(c jwt.MapClaims) { grants(c); c["iss"] = "https://evil.test" })), 401, "UNAUTHENTICATED"},
		"wrong audience": {bearer(token(t, key, func(c jwt.MapClaims) { grants(c); c["aud"] = "other-api" })), 401, "UNAUTHENTICATED"},
		"no expiry":      {bearer(token(t, key, func(c jwt.MapClaims) { grants(c); delete(c, "exp") })), 401, "UNAUTHENTICATED"},
		"bad user id":    {bearer(token(t, key, func(c jwt.MapClaims) { grants(c); c["user_id"] = "nope" })), 401, "UNAUTHENTICATED"},
		"malformed grant": {bearer(token(t, key, func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"CONTENT_ARTICLE_VIEW"}
		})), 401, "UNAUTHENTICATED"},
		"global grant with scope id": {bearer(token(t, key, func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"CONTENT_ARTICLE_VIEW#GLOBAL#X"}
		})), 401, "UNAUTHENTICATED"},
		"customer token without grants": {bearer(token(t, key, nil)), 403, "FORBIDDEN"},
		"service credential":            {bearer(token(t, key, func(c jwt.MapClaims) { grants(c); c["service"] = "order-service" })), 403, "FORBIDDEN"},
		"empty grants":                  {bearer(token(t, key, func(c jwt.MapClaims) { c["permission_grants"] = []string{} })), 403, "FORBIDDEN"},
		"only other service grants": {bearer(token(t, key, func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"ORDER_VIEW#GLOBAL", "ROLE_VIEW#GLOBAL"}
		})), 403, "FORBIDDEN"},
		"content grant on a non global scope": {bearer(token(t, key, func(c jwt.MapClaims) {
			c["permission_grants"] = []string{"CONTENT_ARTICLE_VIEW#CATEGORY#heritage"}
		})), 403, "FORBIDDEN"},
	} {
		t.Run(name, func(t *testing.T) {
			var out *httptest.ResponseRecorder
			if tc.header == "" {
				out = request(r, "GET", "/api/v1/admin/articles", "")
			} else {
				out = request(r, "GET", "/api/v1/admin/articles", "", "Authorization", tc.header)
			}
			envelope(t, out, tc.status, tc.code)
		})
	}
	// Holding VIEW only: reads work, writes are 403 per permission.
	viewOnly := bearer(token(t, key, func(c jwt.MapClaims) { c["permission_grants"] = []string{"CONTENT_ARTICLE_VIEW#GLOBAL"} }))
	require.Equal(t, 200, request(r, "GET", "/api/v1/admin/articles", "", "Authorization", viewOnly).Code)
	envelope(t, request(r, "POST", "/api/v1/admin/articles", string(raw), "Authorization", viewOnly), 403, "FORBIDDEN")

	_, err = middleware.NewJWTVerifier([]byte("not pem"), "iss", "aud")
	require.Error(t, err)
	_, err = middleware.NewJWTVerifier(publicPEM, "", "aud")
	require.Error(t, err)
}
