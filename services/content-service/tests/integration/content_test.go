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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/migrate"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/repository"
	transport "github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/handlers"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/migrations"
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

// migrate applies or reverts through the same runner the executable image uses.
func applyMigrations(t *testing.T, p *pgxpool.Pool, direction string) {
	t.Helper()
	ctx := context.Background()
	switch direction {
	case "up":
		_, err := migrate.Up(ctx, p, migrations.FS)
		require.NoError(t, err)
	case "down":
		for {
			v, err := migrate.Down(ctx, p, migrations.FS)
			require.NoError(t, err)
			if v == 0 {
				return
			}
		}
	}
}
func tableCount(t *testing.T, p *pgxpool.Pool) int {
	t.Helper()
	var count int
	require.NoError(t, p.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema()`).Scan(&count))
	return count
}
func TestMigrationUpDownUpAndConstraints(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	require.Error(t, postgres.Ready(ctx, p))
	applyMigrations(t, p, "up")
	require.NoError(t, postgres.Ready(ctx, p))
	require.Equal(t, 7, tableCount(t, p), "6 service tables + schema_migrations")
	done, err := migrate.Up(ctx, p, migrations.FS)
	require.NoError(t, err)
	require.Empty(t, done, "a second run applies nothing")
	v, err := migrate.Down(ctx, p, migrations.FS)
	require.NoError(t, err)
	require.EqualValues(t, 2, v, "down reverts one step at a time")
	require.Error(t, postgres.Ready(ctx, p), "readiness needs the audit and idempotency tables")
	applyMigrations(t, p, "down")
	require.Equal(t, 1, tableCount(t, p), "only schema_migrations remains")
	require.Error(t, postgres.Ready(ctx, p))
	applyMigrations(t, p, "up")
	require.NoError(t, postgres.Ready(ctx, p))
	s := usecase.New(repository.New(p))
	actor := testhelper.Full()
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
	_, err = p.Exec(ctx, `UPDATE audit_events SET reason='rewritten' WHERE article_id=$1`, a.ID)
	require.Error(t, err, "audit trail is append-only")
	_, err = p.Exec(ctx, `DELETE FROM audit_events WHERE article_id=$1`, a.ID)
	require.Error(t, err, "audit trail is append-only")
	_, err = p.Exec(ctx, `INSERT INTO audit_events(id,article_id,revision_id,action,actor_id,created_at) VALUES($1,$2,$3,'erase',$4,now())`, uuid.New(), a.ID, a.CurrentRevisionID, actor.ID)
	require.Error(t, err, "unknown audit action must fail")
	// A published revision keeps its status for ever; a draft can still move.
	a, err = s.Transition(ctx, a.ID, a.Version, "submit", "", actor)
	require.NoError(t, err)
	a, err = s.Transition(ctx, a.ID, a.Version, "approve", "", actor)
	require.NoError(t, err)
	a, err = s.Transition(ctx, a.ID, a.Version, "publish", "", actor)
	require.NoError(t, err)
	_, err = p.Exec(ctx, `UPDATE article_revisions SET status='DRAFT' WHERE id=$1`, a.CurrentRevisionID)
	require.Error(t, err, "published status must be immutable")
}

// A database created by psql or the old compose initdb mount has no
// schema_migrations table; the runner must adopt it, not re-run 000001.
func TestMigrationAdoptsPsqlCreatedDatabase(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	raw, err := migrations.FS.ReadFile("000001_content.up.sql")
	require.NoError(t, err)
	_, err = p.Exec(ctx, string(raw))
	require.NoError(t, err)
	done, err := migrate.Up(ctx, p, migrations.FS)
	require.NoError(t, err)
	require.Equal(t, []int64{2}, done)
	require.NoError(t, postgres.Ready(ctx, p))
}
func TestDatabaseErrorMapping(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	applyMigrations(t, p, "up")
	repo := repository.New(p)
	actor := testhelper.Full()
	a, err := domain.NewArticle(testhelper.Content("mapping"), actor, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, a))
	// Same article id again: articles_pkey is a conflict, not a 500.
	dup := *a
	require.ErrorIs(t, repo.Create(ctx, &dup), domain.ErrConflict)
	// Same (article, revision_number): the unique constraint is a version conflict.
	_, err = repo.Mutate(ctx, a.ID, a.Version, func(x *domain.Article) error {
		r := x.Revisions[0]
		r.ID = uuid.New()
		x.Revisions = append(x.Revisions, r)
		x.CurrentRevisionID = r.ID
		return nil
	})
	require.ErrorIs(t, err, domain.ErrConflict)
	// Another article's slug stays DUPLICATE_SLUG.
	other, err := domain.NewArticle(testhelper.Content("mapping"), actor, time.Now().UTC())
	require.NoError(t, err)
	require.ErrorIs(t, repo.Create(ctx, other), domain.ErrSlug)
	// Nothing leaked from the failed attempts.
	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Len(t, got.Revisions, 1)
	require.EqualValues(t, 1, got.Version)
}
func TestPostgresIdempotencyStore(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	applyMigrations(t, p, "up")
	repo := repository.New(p)
	user, other := uuid.New(), uuid.New()
	rec, created, err := repo.Begin(ctx, user, "k", "hash-a")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "hash-a", rec.RequestHash)
	rec, created, err = repo.Begin(ctx, user, "k", "hash-a")
	require.NoError(t, err)
	require.False(t, created)
	require.False(t, rec.Completed, "a concurrent duplicate sees the key in progress")
	_, created, err = repo.Begin(ctx, other, "k", "hash-a")
	require.NoError(t, err)
	require.True(t, created, "keys are scoped to the principal")
	require.NoError(t, repo.Complete(ctx, user, "k", 201, map[string]string{"Location": "/x"}, []byte(`{"ok":true}`)))
	rec, created, err = repo.Begin(ctx, user, "k", "hash-a")
	require.NoError(t, err)
	require.False(t, created)
	require.True(t, rec.Completed)
	require.Equal(t, 201, rec.Status)
	require.Equal(t, "/x", rec.Headers["Location"])
	require.JSONEq(t, `{"ok":true}`, string(rec.Body))
	require.NoError(t, repo.Abandon(ctx, user, "k"))
	_, created, err = repo.Begin(ctx, user, "k", "hash-a")
	require.NoError(t, err)
	require.False(t, created, "a completed key is never abandoned")
	require.NoError(t, repo.Abandon(ctx, other, "k"))
	_, created, err = repo.Begin(ctx, other, "k", "hash-b")
	require.NoError(t, err)
	require.True(t, created, "an abandoned key can be reused")
	// A crashed request is taken over once its in-progress window is stale.
	_, err = p.Exec(ctx, `UPDATE idempotency_keys SET created_at=now() - interval '3 minutes' WHERE principal_id=$1`, other)
	require.NoError(t, err)
	_, created, err = repo.Begin(ctx, other, "k", "hash-b")
	require.NoError(t, err)
	require.True(t, created)
	_, created, err = repo.Begin(ctx, other, "k", "different")
	require.NoError(t, err)
	require.False(t, created)
}
func TestPostgresEditorialTransactionsAndPublication(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	applyMigrations(t, p, "up")
	s := usecase.New(repository.New(p))
	actor := testhelper.Full()
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
	_, err = s.Transition(ctx, a.ID, a.Version, "approve", "", testhelper.Editor())
	require.ErrorIs(t, err, domain.ErrForbidden)
	_, err = s.Transition(ctx, a.ID, a.Version, "publish", "", actor)
	require.ErrorIs(t, err, domain.ErrState)
	a, err = s.Transition(ctx, a.ID, a.Version, "reject", "Need evidence", actor)
	require.NoError(t, err)
	require.Equal(t, "Need evidence", a.Decisions[0].Reason)
	for _, action := range []string{"submit", "approve", "publish"} {
		if action == "publish" {
			before, getErr := s.Get(ctx, a.ID, actor)
			require.NoError(t, getErr)
			_, err = s.Transition(ctx, a.ID, a.Version, "publish", "", testhelper.Editor())
			require.ErrorIs(t, err, domain.ErrForbidden)
			after, getErr := s.Get(ctx, a.ID, actor)
			require.NoError(t, getErr)
			require.Equal(t, before, after, "denied publication must not write to PostgreSQL")
		}
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
	a, err = s.Transition(ctx, a.ID, a.Version, "hide", "outdated facts", actor)
	require.NoError(t, err)
	_, err = s.GetPublished(ctx, "db-story")
	require.ErrorIs(t, err, domain.ErrNotFound)
	a, err = s.Transition(ctx, a.ID, a.Version, "restore", "", actor)
	require.NoError(t, err)
	pub, err = s.GetPublished(ctx, "db-story")
	require.NoError(t, err)
	require.Equal(t, "Original story", pub.Revision(publishedID).Content.Body)
	a, err = s.Transition(ctx, a.ID, a.Version, "hide", "", actor)
	require.NoError(t, err)
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
	// Every action, including hide/restore, is in the persisted audit trail.
	var trail []string
	for _, e := range a.Events {
		trail = append(trail, e.Action)
		require.Equal(t, actor.ID, e.ActorID)
	}
	require.Equal(t, []string{"create", "edit", "submit", "reject", "submit", "approve", "publish", "edit", "hide", "restore", "hide", "submit", "approve", "publish"}, trail)
	for _, e := range a.Events {
		if e.Action == "hide" {
			require.Equal(t, publishedID, e.RevisionID)
			break
		}
	}
	var statuses int
	require.NoError(t, p.QueryRow(ctx, `SELECT count(*) FROM article_revisions WHERE article_id=$1 AND status='PUBLISHED'`, a.ID).Scan(&statuses))
	require.Equal(t, 2, statuses, "old published snapshots keep their status when a new one is published")
}
func ptr(s string) *string { return &s }
func TestConcurrentVersionAndSlugRaces(t *testing.T) {
	ctx := context.Background()
	p := database(t)
	applyMigrations(t, p, "up")
	s := usecase.New(repository.New(p))
	actor := testhelper.Full()
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
	applyMigrations(t, p, "up")
	log := zerolog.New(io.Discard)
	repo := repository.New(p)
	h := handlers.New(usecase.New(repo), log, handlers.DefaultMaxBody)
	server := httptest.NewServer(transport.NewRouter(transport.RouterConfig{Handler: h, Logger: log, Verifier: verifiedTestIdentity{testhelper.Full()}, Idempotency: repo, Ready: func(ctx context.Context) error { return postgres.Ready(ctx, p) }}))
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	call := func(method, path string, body any, want int, headers ...string) []byte {
		raw, _ := json.Marshal(body)
		req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		for n := 0; n+1 < len(headers); n += 2 {
			req.Header.Set(headers[n], headers[n+1])
		}
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
	raw := call("POST", "/api/v1/admin/articles", testhelper.Content("network"), 201, "Idempotency-Key", "net-1")
	replayed := call("POST", "/api/v1/admin/articles", testhelper.Content("network"), 201, "Idempotency-Key", "net-1")
	require.JSONEq(t, string(raw), string(replayed), "the PostgreSQL store replays the first response")
	call("POST", "/api/v1/admin/articles", testhelper.Content("network-2"), 409, "Idempotency-Key", "net-1")
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
