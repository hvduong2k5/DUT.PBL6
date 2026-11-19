package migrations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPromotionVersionIsTwelveAndF42GapIsNotRenumbered(t *testing.T) {
	plan, err := migrationPlan(migrationFiles)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, step := range plan {
		if step.name == "000012_promotion_checkout.up.sql" && step.version == 12 {
			found = true
		}
	}
	if !found {
		t.Fatal("Promotion must retain version 12 before and after F-42 is merged")
	}
	// A later merge introduces these filenames. Discovery must include them
	// without changing the persistent identity of the Promotion migration.
	merged := fstest.MapFS{}
	for _, step := range plan {
		merged[step.name] = &fstest.MapFile{Data: []byte(step.ddl)}
	}
	for _, name := range []string{"000009_case_decisions.up.sql", "000010_refund_authority.up.sql", "000011_evidence_vault.up.sql"} {
		if _, exists := merged[name]; !exists {
			merged[name] = &fstest.MapFile{Data: []byte("SELECT 1;")}
		}
	}
	plan, err = migrationPlan(merged)
	if err != nil || len(plan) < 12 {
		t.Fatal("F-42 merge lost migration entries", err)
	}
	for index, step := range plan {
		if step.version != index+1 {
			t.Fatal("migration versions must follow filenames", step)
		}
	}
}

func TestMigrationPlanRejectsDuplicateVersion(t *testing.T) {
	_, err := migrationPlan(fstest.MapFS{
		"000009_case_decisions.up.sql":     {Data: []byte("SELECT 1;")},
		"000009_promotion_checkout.up.sql": {Data: []byte("SELECT 2;")},
	})
	if err == nil {
		t.Fatal("duplicate migration IDs must fail, not silently skip DDL")
	}
}

type migrationTestDB struct{ tx *migrationTestTx }

func (d migrationTestDB) Begin(context.Context) (pgx.Tx, error) { return d.tx, nil }

type migrationTestTx struct {
	pgx.Tx
	versions map[int]bool
	executed []string
	commits  int
	failDDL  bool
}

func (tx *migrationTestTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if sql == PromotionCheckout && tx.failDDL {
		return pgconn.CommandTag{}, errors.New("synthetic DDL failure")
	}
	tx.executed = append(tx.executed, sql)
	if strings.HasPrefix(sql, "INSERT INTO order_migrations") {
		tx.versions[args[0].(int)] = true
	}
	return pgconn.NewCommandTag("OK"), nil
}

type migrationRow bool

func (r migrationRow) Scan(dest ...any) error { *dest[0].(*bool) = bool(r); return nil }
func (tx *migrationTestTx) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	return migrationRow(tx.versions[args[0].(int)])
}
func (tx *migrationTestTx) Commit(context.Context) error   { tx.commits++; return nil }
func (tx *migrationTestTx) Rollback(context.Context) error { return nil }

func TestApplyRunsPromotionAfterPreviouslyAppliedF42Versions(t *testing.T) {
	tx := &migrationTestTx{versions: map[int]bool{}}
	for version := 1; version <= 11; version++ {
		tx.versions[version] = true
	}
	if err := Apply(context.Background(), migrationTestDB{tx}); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, sql := range tx.executed {
		if sql == PromotionCheckout {
			count++
		}
	}
	if count != 1 || !tx.versions[12] || tx.commits != 1 {
		t.Fatal("version 9-11 history incorrectly skipped Promotion DDL", count, tx.versions)
	}
	tx.executed = nil
	if err := Apply(context.Background(), migrationTestDB{tx}); err != nil {
		t.Fatal(err)
	}
	for _, sql := range tx.executed {
		if sql == PromotionCheckout {
			t.Fatal("already applied version 12 ran twice")
		}
	}
}

func TestApplyDoesNotRecordFailedPromotionDDL(t *testing.T) {
	tx := &migrationTestTx{versions: map[int]bool{}, failDDL: true}
	for version := 1; version <= 11; version++ {
		tx.versions[version] = true
	}
	if err := Apply(context.Background(), migrationTestDB{tx}); err == nil {
		t.Fatal("DDL failure was swallowed")
	}
	if tx.versions[12] || tx.commits != 0 {
		t.Fatal("failed DDL was marked applied")
	}
}
