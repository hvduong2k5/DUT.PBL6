package postgres

import (
	"context"
	"errors"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckRuntime uses effective PostgreSQL privileges, not just the DSN username.
func CheckRuntime(ctx context.Context, pool *pgxpool.Pool, mode string) error {
	if pool == nil {
		return errors.New("runtime database missing")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if err = migrations.Verify(ctx, conn.Conn(), mode); err != nil {
		return err
	}
	var safe bool
	err = conn.QueryRow(ctx, `SELECT NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid)
 AND NOT has_schema_privilege(current_user,current_schema(),'CREATE')
 AND NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relowner=r.oid)
 FROM pg_roles r WHERE rolname=current_user`).Scan(&safe)
	if err != nil {
		return err
	}
	if !safe {
		return errors.New("runtime database role has elevated privileges")
	}
	for _, table := range []string{"traceability_profiles", "traceability_revisions", "review_decisions", "audit_entries", "outbox_events", "idempotency_records", "eligibility_observations", "schema_migrations", "service_metadata"} {
		var selectOK, insertOK, updateOK, deleteOK, truncateOK bool
		err = conn.QueryRow(ctx, `SELECT has_table_privilege(current_user,$1,'SELECT'),has_table_privilege(current_user,$1,'INSERT'),has_table_privilege(current_user,$1,'UPDATE'),has_table_privilege(current_user,$1,'DELETE'),has_table_privilege(current_user,$1,'TRUNCATE')`, table).Scan(&selectOK, &insertOK, &updateOK, &deleteOK, &truncateOK)
		if err != nil {
			return err
		}
		mutable := table == "traceability_profiles" || table == "traceability_revisions" || table == "outbox_events"
		insertable := table != "schema_migrations" && table != "service_metadata"
		if !selectOK || insertOK != insertable || updateOK != mutable || deleteOK || truncateOK {
			return errors.New("runtime database privileges do not match service policy")
		}
	}
	return nil
}
