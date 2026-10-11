package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/repository"
	transport "github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/tests/testhelper"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func database(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("CONTENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CONTENT_TEST_DATABASE_URL unset: real PostgreSQL integration NOT RUN")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	require.NoError(t, base.Ping(ctx))
	schema := "content_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = base.Exec(ctx, `CREATE SCHEMA `+pgx.Identifier{schema}.Sanitize())
	require.NoError(t, err)
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 5
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
		_, err := base.Exec(context.Background(), `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`)
		require.NoError(t, err)
		base.Close()
	})
	return pool
}
func migrate(t *testing.T, p *pgxpool.Pool, direction string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_content."+direction+".sql"))
	require.NoError(t, err)
	_, err = p.Exec(context.Background(), string(raw))
	require.NoError(t, err)
}
func TestMigrationUpDownUpAndConstraints(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	require.Error(t, postgres.Ready(ctx, p))
	migrate(t, p, "up")
	require.NoError(t, postgres.Ready(ctx, p))
	var count int
	require.NoError(t, p.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema()`).Scan(&count))
	require.Equal(t, 4, count)
	migrate(t, p, "down")
	require.Error(t, postgres.Ready(ctx, p))
	migrate(t, p, "up")
	require.NoError(t, postgres.Ready(ctx, p))
	s := usecase.New(repository.New(p))
	actor := testhelper.Actor("ADMIN")
	a, err := s.Create(ctx, testhelper.Content("constraint"), actor)
	require.NoError(t, err)
	b, err := s.Create(ctx, testhelper.Content("other"), actor)
	require.NoError(t, err)
	_, err = p.Exec(ctx, `UPDATE articles SET published_revision_id=$2 WHERE id=$1`, a.ID, b.CurrentRevisionID)
	require.Error(t, err, "cross-article publication pointer must fail")
	_, err = p.Exec(ctx, `UPDATE article_revisions SET body='silently overwritten' WHERE id=$1`, a.CurrentRevisionID)
	require.Error(t, err, "snapshot mutation must fail")
	_, err = p.Exec(ctx, `INSERT INTO approval_decisions(id,article_id,revision_id,status,reason,actor_id,created_at) VALUES($1,$2,$3,'REJECTED','',$4,now())`, uuid.New(), a.ID, a.CurrentRevisionID, actor.ID)
	require.Error(t, err)
}
func TestPostgresEditorialTransactionsAndPublication(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	migrate(t, p, "up")
	s := usecase.New(repository.New(p))
	actor := testhelper.Actor("ADMIN")
	a, err := s.Create(ctx, testhelper.Content("db-story"), actor)
	require.NoError(t, err)
	require.Len(t, a.Slugs, 1)
	_, err = s.Create(ctx, testhelper.Content("db-story"), actor)
	require.ErrorIs(t, err, domain.ErrSlug)
	var count int
	require.NoError(t, p.QueryRow(ctx, `SELECT count(*) FROM articles`).Scan(&count))
	require.Equal(t, 1, count, "duplicate create must roll back article and revision")
	_, err = s.Get(ctx, uuid.New(), actor)
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = s.Update(ctx, uuid.New(), 1, usecase.Patch{Title: ptr("x")}, actor)
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = s.GetPublished(ctx, "db-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	a, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{Title: ptr("Reviewed title")}, actor)
	require.NoError(t, err)
	require.Len(t, a.Revisions, 2)
	_, err = s.Update(ctx, a.ID, 1, usecase.Patch{Title: ptr("stale")}, actor)
	require.ErrorIs(t, err, domain.ErrConflict)
	a, err = s.Transition(ctx, a.ID, a.Version, "submit", "", actor)
	require.NoError(t, err)
	_, err = s.Transition(ctx, a.ID, a.Version, "approve", "", testhelper.Actor("CONTENT_EDITOR"))
	require.ErrorIs(t, err, domain.ErrForbidden)
	_, err = s.Transition(ctx, a.ID, a.Version, "publish", "", actor)
	require.ErrorIs(t, err, domain.ErrState)
	a, err = s.Transition(ctx, a.ID, a.Version, "reject", "Need evidence", actor)
	require.NoError(t, err)
	require.Equal(t, "Need evidence", a.Decisions[0].Reason)
	for _, action := range []string{"submit", "approve", "publish"} {
		a, err = s.Transition(ctx, a.ID, a.Version, action, "", actor)
		require.NoError(t, err)
	}
	publishedID := *a.PublishedRevisionID
	other, err := s.Create(ctx, testhelper.Content("reserved"), actor)
	require.NoError(t, err)
	_, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{Slug: ptr(other.Current().Content.Slug)}, actor)
	require.ErrorIs(t, err, domain.ErrSlug)
	same, err := s.Get(ctx, a.ID, actor)
	require.NoError(t, err)
	require.Equal(t, a.Version, same.Version)
	require.Len(t, same.Revisions, len(a.Revisions), "slug failure must roll back new revision")
	a, err = s.Update(ctx, a.ID, a.Version, usecase.Patch{Body: ptr("Private draft body"), Slug: ptr("replacement")}, actor)
	require.NoError(t, err)
	pub, err := s.GetPublished(ctx, "db-story")
	require.NoError(t, err)
	require.Equal(t, "Original story", pub.Revision(publishedID).Content.Body)
	_, err = s.GetPublished(ctx, "replacement")
	require.ErrorIs(t, err, domain.ErrNotFound)
	items, total, err := s.List(ctx, domain.Filter{Page: 1, Limit: 1, Public: true, Query: "reviewed", Category: "heritage"}, domain.Identity{})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "Reviewed title", items[0].Revision(publishedID).Content.Title)
	require.Empty(t, items[0].Revision(publishedID).Content.Body, "lists must not load large article bodies")
	a, err = s.Transition(ctx, a.ID, a.Version, "hide", "", actor)
	require.NoError(t, err)
	_, err = s.GetPublished(ctx, "db-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	for _, action := range []string{"submit", "approve", "publish"} {
		a, err = s.Transition(ctx, a.ID, a.Version, action, "", actor)
		require.NoError(t, err)
	}
	_, err = s.GetPublished(ctx, "db-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	pub, err = s.GetPublished(ctx, "replacement")
	require.NoError(t, err)
	require.Equal(t, "Private draft body", pub.Revision(*pub.PublishedRevisionID).Content.Body)
	a, err = s.Get(ctx, a.ID, actor)
	require.NoError(t, err)
	require.Equal(t, "Original story", a.Revision(publishedID).Content.Body)
	require.Len(t, a.Slugs, 2)
}
func ptr(s string) *string { return &s }
func TestConcurrentVersionAndSlugRaces(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	migrate(t, p, "up")
	s := usecase.New(repository.New(p))
	actor := testhelper.Actor("ADMIN")
	a, err := s.Create(ctx, testhelper.Content("race"), actor)
	require.NoError(t, err)
	run := func(fn func() error, want error) {
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); <-start; results <- fn() }()
		}
		close(start)
		wg.Wait()
		close(results)
		success, conflict := 0, 0
		for e := range results {
			if e == nil {
				success++
			} else if errors.Is(e, want) {
				conflict++
			} else {
				t.Fatalf("unexpected race error: %v", e)
			}
		}
		require.Equal(t, 1, success)
		require.Equal(t, 1, conflict)
	}
	run(func() error { _, err := s.Update(ctx, a.ID, 1, usecase.Patch{Title: ptr("winner")}, actor); return err }, domain.ErrConflict)
	run(func() error { _, err := s.Create(ctx, testhelper.Content("unique-race"), actor); return err }, domain.ErrSlug)
	a, err = s.Get(ctx, a.ID, actor)
	require.NoError(t, err)
	require.EqualValues(t, 2, a.Version)
	require.Len(t, a.Revisions, 2)
}

type verifiedTestIdentity struct{ actor domain.Identity }

func (v verifiedTestIdentity) Verify(*http.Request) (domain.Identity, error) { return v.actor, nil }
func TestRealHTTPWithPostgres(t *testing.T) {
	p := database(t)
	migrate(t, p, "up")
	log := zerolog.New(io.Discard)
	h := handlers.New(usecase.New(repository.New(p)), log)
	server := httptest.NewServer(transport.NewRouter(transport.RouterConfig{Handler: h, Logger: log, Verifier: verifiedTestIdentity{testhelper.Actor("ADMIN")}, Ready: func(ctx context.Context) error { return postgres.Ready(ctx, p) }}))
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	call := func(method, path string, body any, want int) []byte {
		raw, _ := json.Marshal(body)
		req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, want, resp.StatusCode, string(data))
		return data
	}
	for _, path := range []string{"/livez", "/readyz", "/healthz"} {
		call("GET", path, nil, 200)
	}
	raw := call("POST", "/api/v1/admin/articles", testhelper.Content("network"), 201)
	var a handlers.AdminResponse
	require.NoError(t, json.Unmarshal(raw, &a))
	path := "/api/v1/admin/articles/" + a.ID.String()
	call("GET", "/api/v1/articles/network", nil, 404)
	for _, action := range []string{"submit", "approve", "publish"} {
		raw = call("POST", path+"/"+action, map[string]any{"version": a.Version}, 200)
		require.NoError(t, json.Unmarshal(raw, &a))
	}
	call("GET", "/api/v1/articles/network", nil, 200)
	call("PATCH", path, map[string]any{"version": a.Version, "body": "unapproved HTTP draft"}, 200)
	raw = call("GET", "/api/v1/articles/network", nil, 200)
	require.NotContains(t, string(raw), "unapproved HTTP draft")
	call("GET", "/api/v1/articles?q=original&limit=1", nil, 200)
	call("GET", path, nil, 200)
	// Same repository, real network, default executable auth boundary.
	closed := httptest.NewServer(transport.NewRouter(transport.RouterConfig{Handler: h, Logger: log}))
	defer closed.Close()
	req, err := http.NewRequest("GET", closed.URL+"/api/v1/admin/articles", nil)
	require.NoError(t, err)
	req.Header.Set("X-User-ID", uuid.NewString())
	req.Header.Set("X-User-Role", "ADMIN")
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 401, resp.StatusCode)
}
