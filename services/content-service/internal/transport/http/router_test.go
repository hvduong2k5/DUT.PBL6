package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	transport "github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/middleware"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/tests/testhelper"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test identity injection never exists in the production executable.
type verifier struct {
	identity domain.Identity
	err      error
}

func (v verifier) Verify(*http.Request) (domain.Identity, error) { return v.identity, v.err }
func router(repo domain.Repository, v middleware.Verifier, ready func(context.Context) error) http.Handler {
	log := zerolog.New(io.Discard)
	return transport.NewRouter(transport.RouterConfig{Handler: handlers.New(usecase.New(repo), log), Verifier: v, Ready: ready, Logger: log})
}
func request(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	return out
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
		require.Equal(t, 401, out.Code)
		require.Contains(t, out.Body.String(), "UNAUTHENTICATED")
	}
	for _, v := range []verifier{{err: errors.New("bad token")}, {identity: domain.Identity{Role: "ADMIN"}}} {
		out := request(router(testhelper.NewMemory(), v, nil), "GET", "/api/v1/admin/articles", "")
		require.Equal(t, 401, out.Code)
	}
	out := request(router(testhelper.NewMemory(), verifier{identity: testhelper.Actor("CUSTOMER")}, nil), "GET", "/api/v1/admin/articles", "")
	require.Equal(t, 403, out.Code)
}
func TestHTTPWorkflowAndErrors(t *testing.T) {
	repo := testhelper.NewMemory()
	r := router(repo, verifier{identity: testhelper.Actor("ADMIN")}, func(context.Context) error { return nil })
	raw, _ := json.Marshal(testhelper.Content("http-story"))
	out := request(r, "POST", "/api/v1/admin/articles", string(raw))
	require.Equal(t, 201, out.Code, out.Body.String())
	require.NotEmpty(t, out.Header().Get("ETag"))
	require.NotEmpty(t, out.Header().Get("Location"))
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	require.Equal(t, 404, request(r, "GET", "/api/v1/articles/http-story", "").Code)
	require.Equal(t, 409, request(r, "POST", "/api/v1/admin/articles", string(raw)).Code)
	require.Equal(t, 404, request(r, "GET", "/api/v1/admin/articles/"+uuid.NewString(), "").Code)
	require.Equal(t, 400, request(r, "GET", "/api/v1/admin/articles/bad-id", "").Code)
	require.Equal(t, 400, request(r, "POST", "/api/v1/admin/articles", `{"title":`).Code)
	require.Equal(t, 400, request(r, "POST", "/api/v1/admin/articles", `{"unknown":true}`).Code)
	require.Equal(t, 400, request(r, "POST", "/api/v1/admin/articles", string(raw)+` {}`).Code)
	require.Equal(t, 422, request(r, "POST", "/api/v1/admin/articles", `{}`).Code)
	require.Equal(t, 409, request(r, "POST", path+"/publish", `{"version":1}`).Code)
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
	require.Equal(t, 409, request(r, "PATCH", path, string(body)).Code)
	body, _ = json.Marshal(map[string]any{"version": a.Version})
	out = request(r, "POST", path+"/hide", string(body))
	require.Equal(t, 200, out.Code)
	require.Equal(t, 404, request(r, "GET", "/api/v1/articles/http-story", "").Code)
	for _, path := range []string{"/api/v1/articles?page=x", "/api/v1/articles?page=1&page=2"} {
		require.Equal(t, 400, request(r, "GET", path, "").Code)
	}
	for _, path := range []string{"/api/v1/articles?page=0", "/api/v1/articles?limit=101", "/api/v1/articles?status=DRAFT"} {
		require.Equal(t, 422, request(r, "GET", path, "").Code)
	}
	require.Equal(t, 200, request(r, "GET", "/api/v1/admin/articles?status=DRAFT", "").Code)
}
func TestApprovalRequiresApproverAndRejectReason(t *testing.T) {
	repo := testhelper.NewMemory()
	admin := router(repo, verifier{identity: testhelper.Actor("ADMIN")}, nil)
	editor := router(repo, verifier{identity: testhelper.Actor("CONTENT_EDITOR")}, nil)
	raw, _ := json.Marshal(testhelper.Content("review"))
	out := request(admin, "POST", "/api/v1/admin/articles", string(raw))
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	require.Equal(t, 200, request(editor, "POST", path+"/submit", `{"version":1}`).Code)
	for _, action := range []string{"approve", "reject"} {
		require.Equal(t, 403, request(editor, "POST", path+"/"+action, `{"version":2,"reason":"sources"}`).Code)
	}
	require.Equal(t, 422, request(admin, "POST", path+"/reject", `{"version":2}`).Code)
	out = request(admin, "POST", path+"/reject", `{"version":2,"reason":"sources required"}`)
	require.Equal(t, 200, out.Code)
	require.Contains(t, out.Body.String(), "sources required")
}

func TestPublishRolesAndStates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		verifier middleware.Verifier
		want     int
	}{
		{"admin", verifier{identity: testhelper.Actor("ADMIN")}, 200},
		{"approver", verifier{identity: testhelper.Actor("CONTENT_APPROVER")}, 200},
		{"editor", verifier{identity: testhelper.Actor("CONTENT_EDITOR")}, 403},
		{"other role", verifier{identity: testhelper.Actor("CUSTOMER")}, 403},
		{"missing verifier", nil, 401},
		{"invalid token", verifier{err: errors.New("invalid token")}, 401},
		{"invalid identity", verifier{identity: domain.Identity{Role: "ADMIN"}}, 401},
	} {
		for _, state := range []domain.Status{domain.Draft, domain.Pending, domain.Approved, domain.Rejected, domain.Published} {
			t.Run(tc.name+"/"+string(state), func(t *testing.T) {
				ctx := context.Background()
				repo := testhelper.NewMemory()
				s := usecase.New(repo)
				admin := testhelper.Actor("ADMIN")
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
func TestHealthAndInternalErrorSanitization(t *testing.T) {
	r := router(brokenRepo{testhelper.NewMemory()}, nil, func(context.Context) error { return errors.New("offline") })
	require.Equal(t, 200, request(r, "GET", "/livez", "").Code)
	for _, path := range []string{"/readyz", "/healthz"} {
		out := request(r, "GET", path, "")
		require.Equal(t, 503, out.Code)
		require.Contains(t, out.Body.String(), `"database":false`)
	}
	out := request(r, "GET", "/api/v1/articles/test", "")
	require.Equal(t, 500, out.Code)
	require.NotContains(t, out.Body.String(), "DSN")
	require.NotContains(t, out.Body.String(), "stack trace")
	r = router(testhelper.NewMemory(), nil, func(context.Context) error { return nil })
	for _, path := range []string{"/readyz", "/healthz"} {
		require.Equal(t, 200, request(r, "GET", path, "").Code)
	}
}
