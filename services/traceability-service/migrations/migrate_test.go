package migrations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Every test owns a disposable schema; no public/peer schema enters search_path.
func database(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()
	url := os.Getenv("TRACEABILITY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("NOT_RUN: TRACEABILITY_TEST_DATABASE_URL is required for real PostgreSQL tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal("test database connection failed")
	}
	suffix := make([]byte, 12)
	if _, err = rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	schema := "trace_test_" + hex.EncodeToString(suffix)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if !strings.HasPrefix(schema, "trace_test_") {
			t.Fatal("unsafe cleanup schema")
		}
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close(cleanup)
	})
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal("invalid test database URL")
	}
	cfg.RuntimeParams["search_path"] = schema
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal("schema connection failed")
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return ctx, conn
}

func exec(t *testing.T, ctx context.Context, c *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := c.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}
func rejected(t *testing.T, ctx context.Context, c *pgx.Conn, sql, code string) {
	t.Helper()
	_, err := c.Exec(ctx, sql)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != code {
		t.Fatalf("expected SQLSTATE %s, got %v", code, err)
	}
}

const profile = "11111111-1111-4111-8111-111111111111"
const author = "22222222-2222-4222-8222-222222222222"
const reviewer = "33333333-3333-4333-8333-333333333333"

func seed(t *testing.T, ctx context.Context, c *pgx.Conn) {
	t.Helper()
	if err := Apply(ctx, c, "sandbox"); err != nil {
		t.Fatal(err)
	}
	exec(t, ctx, c, `INSERT INTO traceability_profiles(id,mode,product_id,sku_code,source_namespace,batch_id,qr_code)
 VALUES('11111111-1111-4111-8111-111111111111','sandbox','44444444-4444-4444-8444-444444444444','SKU-1','fixture','batch-1','abcdefghijklmnopqrstuv')`)
	exec(t, ctx, c, `INSERT INTO traceability_revisions(profile_id,number,contributors,created_by)
 VALUES('11111111-1111-4111-8111-111111111111',1,ARRAY['22222222-2222-4222-8222-222222222222']::uuid[],'22222222-2222-4222-8222-222222222222')`)
}

func TestMigrationRoundtrip(t *testing.T) {
	ctx, c := database(t)
	for range 2 {
		if err := Apply(ctx, c, "sandbox"); err != nil {
			t.Fatal(err)
		}
		if err := Apply(ctx, c, "sandbox"); err != nil {
			t.Fatalf("idempotent apply: %v", err)
		}
		if err := Apply(ctx, c, "integration"); err == nil {
			t.Fatal("mode mismatch accepted")
		}
		if err := DownForTest(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
}
func TestMigrationChecksum(t *testing.T) {
	ctx, c := database(t)
	if err := Apply(ctx, c, "sandbox"); err != nil {
		t.Fatal(err)
	}
	exec(t, ctx, c, `UPDATE schema_migrations SET checksum='tampered'`)
	if err := Apply(ctx, c, "sandbox"); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("checksum guard: %v", err)
	}
}
func TestBindingAndHistory(t *testing.T) {
	ctx, c := database(t)
	seed(t, ctx, c)
	rejected(t, ctx, c, `UPDATE traceability_profiles SET batch_id='other',version=2`, "23514")
	rejected(t, ctx, c, `UPDATE traceability_profiles SET version=3`, "23514")
	rejected(t, ctx, c, `INSERT INTO traceability_profiles(id,mode,product_id,sku_code,source_namespace,batch_id,qr_code)
 SELECT '55555555-5555-4555-8555-555555555555',mode,product_id,sku_code,source_namespace,batch_id,'zyxwvutsrqponmlkjihgfe' FROM traceability_profiles`, "23505")
	rejected(t, ctx, c, `DELETE FROM traceability_profiles`, "23514")
	rejected(t, ctx, c, `DELETE FROM traceability_revisions`, "23514")
	rejected(t, ctx, c, `UPDATE service_metadata SET mode='integration'`, "23514")
	rejected(t, ctx, c, `INSERT INTO traceability_revisions(profile_id,number,contributors,created_by) SELECT profile_id,2,contributors,created_by FROM traceability_revisions`, "23505")
	exec(t, ctx, c, `UPDATE traceability_revisions SET status='IN_REVIEW',dependency_snapshot='{"source_version":"1"}'`)
	rejected(t, ctx, c, `UPDATE traceability_revisions SET content='{"changed":true}'`, "23514")
	rejected(t, ctx, c, `UPDATE traceability_revisions SET contributors=ARRAY['33333333-3333-4333-8333-333333333333']::uuid[]`, "23514")
	rejected(t, ctx, c, `INSERT INTO review_decisions(id,profile_id,revision_number,actor_id,decision,reason,profile_version)
 VALUES('66666666-6666-4666-8666-666666666666','11111111-1111-4111-8111-111111111111',1,'22222222-2222-4222-8222-222222222222','APPROVE','checked',2)`, "23514")
}

func TestApprovalNeedsDecision(t *testing.T) {
	ctx, c := database(t)
	seed(t, ctx, c)
	exec(t, ctx, c, `UPDATE traceability_revisions SET status='IN_REVIEW',dependency_snapshot='{}'`)
	rejected(t, ctx, c, `UPDATE traceability_revisions SET status='APPROVED'`, "23514")
	tx, err := c.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO review_decisions(id,profile_id,revision_number,actor_id,decision,reason,profile_version)
 VALUES('66666666-6666-4666-8666-666666666666',$1,1,$2,'APPROVE','independent review',2)`, profile, reviewer); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status='APPROVED'`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	rejected(t, ctx, c, `UPDATE review_decisions SET reason='rewritten'`, "23514")
	rejected(t, ctx, c, `UPDATE traceability_revisions SET status='PUBLISHED'`, "23514")
	tx, err = c.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status='PUBLISHED'`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET public_revision=1,ever_published=true,version=2`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	exec(t, ctx, c, `UPDATE traceability_profiles SET status='SUSPENDED',version=3`)
	rejected(t, ctx, c, `UPDATE traceability_profiles SET status='ACTIVE',version=4`, "23514")
}

func TestRollbackAndCAS(t *testing.T) {
	ctx, c := database(t)
	seed(t, ctx, c)
	tx, err := c.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET version=2`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_entries(id,profile_id,actor_id,action,before_state,after_state,request_id) VALUES('77777777-7777-4777-8777-777777777777',$1,$2,'edit','{}','{}','test')`, profile, author); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var count, version int
	if err = c.QueryRow(ctx, `SELECT count(*) FROM audit_entries`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("audit rollback: %d %v", count, err)
	}
	if err = c.QueryRow(ctx, `SELECT version FROM traceability_profiles`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("version rollback: %d %v", version, err)
	}
	other, err := pgx.ConnectConfig(ctx, c.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close(ctx)
	type result struct {
		n   int64
		err error
	}
	results := make(chan result, 2)
	for _, conn := range []*pgx.Conn{c, other} {
		go func(conn *pgx.Conn) {
			tag, err := conn.Exec(ctx, `UPDATE traceability_profiles SET version=version+1 WHERE id=$1 AND version=1`, profile)
			results <- result{tag.RowsAffected(), err}
		}(conn)
	}
	var total int64
	for range 2 {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		total += r.n
	}
	if total != 1 {
		t.Fatalf("concurrent CAS updated %d profiles", total)
	}
}

func TestRuntimeRole(t *testing.T) {
	ctx, c := database(t)
	seed(t, ctx, c)
	var schema string
	if err := c.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	role := schema + "_runtime"
	qrole := pgx.Identifier{role}.Sanitize()
	exec(t, ctx, c, "CREATE ROLE "+qrole+" LOGIN NOINHERIT")
	defer func() {
		exec(t, ctx, c, "RESET ROLE")
		exec(t, ctx, c, "DROP OWNED BY "+qrole)
		exec(t, ctx, c, "DROP ROLE "+qrole)
	}()
	if err := GrantRuntime(ctx, c, role); err != nil {
		t.Fatal(err)
	}
	if err := GrantRuntime(ctx, c, "trace_migrator"); err == nil {
		t.Fatal("privileged migrator accepted")
	}
	exec(t, ctx, c, "SET ROLE "+qrole)
	exec(t, ctx, c, `SELECT * FROM traceability_profiles`)
	exec(t, ctx, c, `UPDATE traceability_profiles SET version=2`)
	rejected(t, ctx, c, `CREATE TABLE should_not_exist(id int)`, "42501")
	rejected(t, ctx, c, `TRUNCATE traceability_profiles CASCADE`, "42501")
	rejected(t, ctx, c, `DELETE FROM traceability_revisions`, "42501")
	rejected(t, ctx, c, `UPDATE schema_migrations SET checksum='bad'`, "42501")
	rejected(t, ctx, c, `ALTER TABLE traceability_profiles DISABLE TRIGGER ALL`, "42501")
}

func TestDownGuard(t *testing.T) {
	ctx, c := database(t)
	exec(t, ctx, c, `SET search_path=pg_catalog`)
	if err := DownForTest(ctx, c); err == nil {
		t.Fatal("down allowed outside disposable schema")
	}
}

func TestOutboxFailureRollsBackCommand(t *testing.T) {
	ctx, c := database(t)
	seed(t, ctx, c)
	eventSQL := `INSERT INTO outbox_events(id,profile_id,aggregate_version,event_type,topic,partition_key,payload)
 VALUES('88888888-8888-4888-8888-888888888888',$1::uuid,2,'profile.changed','traceability.events',$1::uuid::text,'{"version":2}')`
	exec(t, ctx, c, eventSQL, profile)
	tx, err := c.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for _, statement := range []string{
		`UPDATE traceability_profiles SET version=2`,
		`INSERT INTO audit_entries(id,profile_id,actor_id,action,before_state,after_state,request_id) VALUES('77777777-7777-4777-8777-777777777777','11111111-1111-4111-8111-111111111111','22222222-2222-4222-8222-222222222222','edit','{}','{}','rollback-test')`,
		`INSERT INTO idempotency_records(principal_id,action,key,fingerprint,status_code,response,resource_id) VALUES('22222222-2222-4222-8222-222222222222','edit','test-key',repeat('a',64),200,'{}','11111111-1111-4111-8111-111111111111')`,
	} {
		if _, err = tx.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.Exec(ctx, eventSQL, profile); err == nil {
		t.Fatal("duplicate outbox event accepted")
	}
	if err = tx.Commit(ctx); err == nil {
		t.Fatal("failed command committed")
	}
	var version, audits, keys int
	if err = c.QueryRow(ctx, `SELECT version,(SELECT count(*) FROM audit_entries),(SELECT count(*) FROM idempotency_records) FROM traceability_profiles`).Scan(&version, &audits, &keys); err != nil {
		t.Fatal(err)
	}
	if version != 1 || audits != 0 || keys != 0 {
		t.Fatalf("partial commit: version=%d audits=%d keys=%d", version, audits, keys)
	}
	rejected(t, ctx, c, `UPDATE outbox_events SET payload='{"tampered":true}'`, "23514")
	exec(t, ctx, c, `UPDATE outbox_events SET status='PUBLISHED',published_at=now(),attempts=1`)
	rejected(t, ctx, c, `UPDATE outbox_events SET status='PENDING',published_at=NULL`, "23514")
}
