# Inventory migration job

Updated: 2026-11-21.

The same binaries and migration set are used in every environment. The server only
calls `migrations.Verify`: it never applies schema changes or inserts demo stock.
Run the migration job successfully before starting the new server release.

## New database

Set `DB_URL` for the intended Inventory database, then run from this service:

```sh
go run ./cmd/migrate
go run ./cmd/server
```

The images include `/app/inventory-migrate`. Run a separate job using the same
release image, overriding its entrypoint to `/app/inventory-migrate`; provide
`DB_URL`. Start the service only after that job exits successfully. This also
applies to the Coolify image; do not append arguments to its server entrypoint.

## Existing database with manually applied 000001–000005

Back up the database and confirm all five original migrations were applied.
Stop the old server during the transition. Baseline only records checksums; it
does not execute these SQL files or prove their historical contents. It checks
key legacy tables, the batch version column, and the stock projection function.

```sh
go run ./cmd/migrate baseline
go run ./cmd/migrate
```

In the release image, the equivalent commands are `/app/inventory-migrate baseline`
and `/app/inventory-migrate`. Baseline is one-time and refuses databases already
containing the migration history table. Do not baseline an incomplete schema.
If 000006 or 000007 were also applied manually, reconcile that database explicitly
before adopting this runner; this baseline covers only 000001–000005.

## Guarantees and operational limits

- All pending SQL and history writes share one transaction and advisory lock.
  Failure rolls back the entire pending batch. Concurrent migration jobs serialize.
- Files are embedded and run in lexical order. Versions must be unique.
- `inventory_schema_migrations(version, checksum, applied_at)` tracks SHA-256
  checksums with CRLF normalized to LF so Windows and Linux builds agree.
- Existing matching versions are skipped. Changed checksums, unknown versions,
  and history gaps fail. Restore the original SQL; never overwrite checksums to
  conceal an edited migration.
- Missing history on a legacy database stops with baseline instructions; it never
  reruns 000001 automatically. Server verification is read-only and fails when any
  packaged migration is missing or its checksum differs.
- **000007 still contains demo stock. Running this release's migration job in any
  environment applies that data.** Moving execution out of the server does not
  make this migration optional. Review the seed before running the job on real data.
- The runner has no automatic down command. For a reviewed manual rollback, stop
  the service, acquire the runner advisory transaction lock (2026112101), execute
  the matching down SQL, then delete only that version's history row in the same
  transaction. Roll back latest versions first. Seed rollback refuses used stock.
  Running this release's migration job again will reapply any removed version;
  the server refuses a database missing its required migrations.

## Integration tests

Use a disposable PostgreSQL 16 database and set `INVENTORY_TEST_DB_URL`:

```sh
go test ./migrations -v -count=1
go test ./...
```

Tests create isolated schemas and cover fresh install, legacy baseline, repeat
execution, checksum mismatch, missing schema, atomic rollback, stock preservation,
and seed rollback ownership. Database tests skip when the test URL is absent.
