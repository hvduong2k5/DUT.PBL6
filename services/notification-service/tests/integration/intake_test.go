package integration

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/notification-service/internal/adapters/postgres"
	"github.com/omamx/notification-service/internal/domain"
	"github.com/omamx/notification-service/migrations"
	"os"
	"sync"
	"testing"
)

func TestPostgresIntake(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("NOTIFICATION_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("dedicated test database required")
		}
		t.Skip("NOTIFICATION_TEST_DATABASE_URL absent; real PostgreSQL NOT RUN")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := "notification_test_" + uuid.NewString()[:8]
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	if e = migrations.Apply(ctx, p); e != nil {
		t.Fatal(e)
	}
	if e = migrations.Apply(ctx, p); e != nil {
		t.Fatal(e)
	}
	if e = migrations.Check(ctx, p); e != nil {
		t.Fatal(e)
	}
	s := postgres.Store{Pool: p}
	c := domain.Command{SourceEvent: domain.SourceEvent{ID: uuid.NewString(), Source: "order-service"}, Purpose: "ORDER_CONFIRMATION", Audience: "BUYER", RecipientRef: "opaque", ContextRef: "version1"}
	var wg sync.WaitGroup
	results := make(chan domain.Result, 50)
	failures := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := s.Accept(ctx, "order-service", "same-key", c)
			if e != nil {
				failures <- e
			} else {
				results <- r
			}
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for e := range failures {
		t.Fatal(e)
	}
	first := ""
	original := 0
	for r := range results {
		if first == "" {
			first = r.ID
		}
		if r.ID != first {
			t.Fatal("duplicate request")
		}
		if !r.Duplicate {
			original++
		}
	}
	if original != 1 {
		t.Fatalf("originals=%d", original)
	}
	alias, e := s.Accept(ctx, "order-service", "another-key", c)
	if e != nil || alias.ID != first || !alias.Duplicate {
		t.Fatalf("business dedup: %+v %v", alias, e)
	}
	changed := c
	changed.ContextRef = "version2"
	if _, e = s.Accept(ctx, "order-service", "same-key", changed); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("missing key conflict", e)
	}
	if _, e = s.Accept(ctx, "order-service", "third-key", changed); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("missing intent conflict", e)
	}
	if _, e = s.Get(ctx, "other-service", first); !errors.Is(e, domain.ErrNotFound) {
		t.Fatal("foreign scope exposed", e)
	}
	var requests, audits int
	if e = p.QueryRow(ctx, "SELECT count(*) FROM notification_requests").Scan(&requests); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, "SELECT count(*) FROM audit_outbox").Scan(&audits); e != nil {
		t.Fatal(e)
	}
	if requests != 1 || audits != 1 {
		t.Fatal("non-atomic duplicate", requests, audits)
	}
	// Failure of the durable audit obligation must roll back the entire intake.
	_, e = p.Exec(ctx, `CREATE FUNCTION reject_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit failure'; END $$;
	CREATE TRIGGER reject_audit BEFORE INSERT ON audit_outbox FOR EACH ROW EXECUTE FUNCTION reject_audit()`)
	if e != nil {
		t.Fatal(e)
	}
	rollbackCommand := c
	rollbackCommand.SourceEvent.ID = uuid.NewString()
	if _, e = s.Accept(ctx, "order-service", "rollback-key", rollbackCommand); e == nil {
		t.Fatal("audit failure was ignored")
	}
	if e = p.QueryRow(ctx, "SELECT count(*) FROM notification_requests").Scan(&requests); e != nil || requests != 1 {
		t.Fatal("partial request committed", requests, e)
	}
	if _, e = p.Exec(ctx, "DROP TRIGGER reject_audit ON audit_outbox; DROP FUNCTION reject_audit()"); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, "UPDATE notification_schema_migrations SET checksum='tampered'"); e != nil {
		t.Fatal(e)
	}
	if migrations.Apply(ctx, p) == nil {
		t.Fatal("checksum drift ignored")
	}
}
