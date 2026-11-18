package migrations

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GrantRuntime grants DML to a separately provisioned, unprivileged login.
// It never creates a role, accepts a password, or grants schema ownership.
func GrantRuntime(ctx context.Context, conn *pgx.Conn, role string) error {
	if role == "" {
		return errors.New("runtime role is required")
	}
	var safe bool
	err := conn.QueryRow(ctx, `SELECT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=current_schema() AND nspowner=r.oid)
	AND NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relowner=r.oid)
 FROM pg_roles r WHERE rolname=$1`, role).Scan(&safe)
	if err != nil || !safe {
		return errors.New("runtime role must be an existing unprivileged login without memberships or schema ownership")
	}
	var schema string
	if err = conn.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qrole := pgx.Identifier{role}.Sanitize()
	qschema := pgx.Identifier{schema}.Sanitize()
	statements := []string{
		"REVOKE ALL ON SCHEMA " + qschema + " FROM " + qrole,
		"REVOKE ALL ON ALL TABLES IN SCHEMA " + qschema + " FROM " + qrole,
		"GRANT USAGE ON SCHEMA " + qschema + " TO " + qrole,
		"GRANT SELECT ON ALL TABLES IN SCHEMA " + qschema + " TO " + qrole,
	}
	for _, table := range []string{"traceability_profiles", "traceability_revisions", "outbox_events"} {
		statements = append(statements, "GRANT INSERT, UPDATE ON "+pgx.Identifier{schema, table}.Sanitize()+" TO "+qrole)
	}
	for _, table := range []string{"review_decisions", "audit_entries", "idempotency_records", "eligibility_observations"} {
		statements = append(statements, "GRANT INSERT ON "+pgx.Identifier{schema, table}.Sanitize()+" TO "+qrole)
	}
	for _, statement := range statements {
		if _, err = tx.Exec(ctx, statement); err != nil {
			return err
		}
	}
	// PUBLIC privileges or ownership must not silently undo the separation.
	var create bool
	if err = tx.QueryRow(ctx, `SELECT has_schema_privilege($1,$2,'CREATE')`, role, schema).Scan(&create); err != nil {
		return err
	}
	if create {
		return errors.New("runtime role still has schema CREATE privilege; revoke inherited/PUBLIC access before granting")
	}
	return tx.Commit(ctx)
}
