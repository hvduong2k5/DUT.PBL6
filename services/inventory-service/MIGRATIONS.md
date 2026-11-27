# Inventory migration and demo jobs

Updated: 2026-11-21.

All environments use the same image and binaries. Only environment variables and
job commands differ. The server verifies schema read-only and never migrates or
seeds. No deployment command should run a destructive SQL seed.

## Commands

Set DB_URL to the intended Inventory database. Run from services/inventory-service:

```sh
go run ./cmd/migrate
# Optional demo job; default is disabled, including when DB_URL is absent.
INVENTORY_SEED_DEMO=true go run ./cmd/seed-demo
go run ./cmd/server
```

Both Dockerfiles package /app/inventory-migrate, /app/inventory-seed-demo, and
/app/inventory-service. On Docker/Coolify run migration and optional seed as
separate jobs, overriding the image ENTRYPOINT to the corresponding binary.
Wait for each job to exit successfully before starting the server. The Coolify
server entrypoint is a wrapper for secrets; passing arguments to that wrapper
does not select another job. Production leaves INVENTORY_SEED_DEMO unset/false.

## Existing manually migrated databases

Back up and stop stock writers for this one-time transition. Confirm the exact
original migrations 000001–000005 were already applied, then run:

```sh
go run ./cmd/migrate baseline
go run ./cmd/migrate
```

Baseline records checksums without replaying SQL. It checks key tables, the batch
version column, and the stock function; presence checks cannot prove historical
SQL contents. Baseline refuses a database already containing migration history.
Do not use it on a partial schema. A DB with manual 000006 must first have its
history reconciled by an operator; this baseline only attests 000001–000005.

000007 was an unreleased demo-only migration in earlier versions of PR #118. It
is removed from the schema set, its number remains reserved, and demo SQL now
belongs to internal/seed. New schema changes use 000008. Never renumber 000008 to
fill the gap. Migrations 000001–000006 are unchanged. If an earlier PR build was
used to record 000007 in a test DB, the new runner deliberately rejects that
unknown history entry. Inspect/back up that DB and reconcile its history before
switching builds; do not overwrite checksums or rerun old DDL. Existing demo data
is retained, and existing SKUs are skipped by the optional seed job.

## Runner guarantees

Embedded *.up.sql files execute lexically in one transaction protected by an
advisory lock (2026112101). inventory_schema_migrations tracks version, checksum,
and applied_at. SHA-256 normalizes CRLF to LF for Windows/Linux parity. Matching
checksums skip. Duplicate versions, history gaps, unknown versions, and checksum
mismatches fail. Any pending migration failure rolls back the whole batch.
Server Verify also probes required receipt/batch columns to catch partial schema
changes even if an operator recorded incorrect history. The migration job can
wait up to five minutes; the server only checks and never competes for DDL locks.

There is no automatic schema down command. For an approved rollback, stop stock
writers, lock 2026112101 transactionally, apply the latest matching down SQL, and
remove only its history row in that same transaction. 000008 refuses rollback
once canonical receipt evidence or nullable-date batches exist. Never delete
business records merely to force rollback.

## Optional demo data

INVENTORY_SEED_DEMO=true is required by the seed CLI for both up and down:

```sh
INVENTORY_SEED_DEMO=true go run ./cmd/seed-demo
INVENTORY_SEED_DEMO=true go run ./cmd/seed-demo down
```

A single VALUES list defines 40 OM-* finished-good SKUs and 2,600 physical units.
Inventory seed does not create Catalog products or images. Existing SKUs (even
stock 0 or a different warehouse) are skipped with notices and never replenished.
Concurrent jobs serialize. Stock snapshots intentionally emit two absolute
versions per newly seeded SKU (80 outbox events for a fresh seed). Snapshot
versions remain monotonic across a reviewed rollback and later reseed while the
outbox history is retained; do not purge that history during this demo lifecycle.

Dates are synthetic: manufacture = first seed date minus 7 days, expiry = first
seed date plus 90 days. Reruns do not refresh dates. After expiry, stock becomes
unsellable as intended; create a fresh disposable demo DB for a new demo cycle.
The ACTIVE state is appropriate with 90 days remaining (near-expiry is <=45 days).

Rollback requires stopping all stock writers. It refuses changed quantities,
status/version, extra batches, reservations, adjustments, receipts, or procurement
references. It emits a higher-version zero snapshot before deleting owned rows,
keeps outbox/audit history, and preserves all preexisting rows. The old
migrations/seed_sample_data.sql has been removed; use cmd/seed-demo instead.
infra/scripts/init-inventory-data.ps1 uses the same runner and accepts -SeedDemo.

Catalog handoff remains separate: Downloads/V12__seed_pbl6_demo.sql was supplied
to the Catalog owner. It is not merged by this PR and its migration number must
be rechecked there. Before a storefront demo, that owner must load the matching
40 OM-* SKU definitions and prices. Until then, Inventory stock alone cannot make
those products appear in the storefront. Nine demo weights and stock/dates remain
synthetic; images are deferred.

## Test commands

Use a disposable PostgreSQL 16 database, set INVENTORY_TEST_DB_URL, then:

```sh
python ../../scripts/test-inventory-procurement-contract.py
go test ./...
go test -race -count=1 ./...
```

CI provisions PostgreSQL, validates the canonical fixture with JSON Schema, runs
migrate twice, confirms default seed inserts nothing, runs explicit seed twice,
and executes race/integration tests. Tests create isolated schemas. DB tests
skip when INVENTORY_TEST_DB_URL is absent; CI always supplies it. Live Procurement
producer integration remains a deployment check because that service is absent
from this repository.

Monitoring: set INVENTORY_METRICS_ADDR to a private listener, scrape /metrics and
check /readyz. Group lag polls committed versus end offsets every 15 seconds;
-1 means no complete sample is available (including an uninitialized group).
Use broker ACLs allowing topic metadata and group offset reads. Schedule
SELECT inventory_prune_procurement_inbox(now()-interval '90 days') daily; this
only removes inbox rows backed by permanent receipt evidence. Replayed receipts
remain deduplicated after pruning. DLQ retention is 90 days; inspect/replay before
expiry, retaining original source/id/receipt_id and reconciling stock first.

## Review repair plan and implementation — 2026-11-21

1. Preserve event identity independently of inbox retention; normalize numeric hashes.
2. Strengthen baseline and Verify checks for required tables, functions and enabled triggers.
3. Align unit spelling/precision/range in the canonical JSON Schema and consumer;
   ignore other Procurement event types on the shared topic without DLQ pollution.
4. Add an audited QC release operation, then rerun regression and integration gates.

Implemented in 000009 (000001–000008 are not edited). The permanent
procurement_event_ledger retains source/event ID/hash even after inbox pruning.
It backfills retained inbox evidence from the older release; IDs whose inbox rows
were already deleted cannot be reconstructed. No migration can recover that lost
history. Before enabling pruning after an upgrade, reconcile the older deployment's
pruned period from retained producer/DLQ evidence. Legacy hashes remain readable;
retained original receipt payloads allow semantic comparison of numeric replays.
New hashes compare exact numeric values, so 2.5 and 2.50 are equivalent. Alias
inbox rows can now be pruned while their permanent identity proof stays intact.

### Privileged QC release job

The same image also packages /app/inventory-qc-release. This is an internal
operations command, not a staff REST API. Only a dedicated DB login that is a
member of inventory_qc_operator may execute it. A DBA provisions this NOLOGIN
role and grants the dedicated login only the needed schema usage, SELECT on
Inventory tables/history, UPDATE of batches/inventory_items, and INSERT on
inventory_qc_releases/outbox_events, plus required function execution. Do not
share the migration/superuser credential with staff or use it as a public API.
The migration deliberately does not create cluster-wide roles or grant membership.

```sql
-- DBA setup, once; replace the login with your actual restricted operations login.
CREATE ROLE inventory_qc_operator NOLOGIN;
GRANT inventory_qc_operator TO inventory_qc_job_login;
-- Grant object privileges in the intended Inventory schema as described above.
```

```sh
/app/inventory-qc-release --actor <employee-uuid> --warehouse <warehouse-uuid> \
  --batch <batch-uuid> --expected-version 1 \
  --manufactured 2026-11-20 --expires 2027-02-20 \
  --reason 'QC certificate reference' --idempotency-key qc-release-unique-key
```

Supply verified dates and an accountable employee UUID. DB-role membership is the
command's authorization boundary; actor is an audited operator-supplied identity,
not a signed staff session. The job rejects warehouse/version/state mismatch,
invalid dates and key reuse with different data. It locks item before batch,
updates QUARANTINE to ACTIVE/NEAR_EXPIRY without changing physical stock, records
request/result/reason/warehouse atomically, and emits normal versioned stock
snapshots. An identical retry returns the original result. The 000009 down refuses
to remove permanent event/QC evidence. API-driven QC can be added later with the
normal permission-and-scope contract workflow; it is not exposed by this command.

Concrete grants (run as DBA in the intended schema; replace public/login if needed):

```sql
GRANT USAGE ON SCHEMA public TO inventory_qc_operator;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO inventory_qc_operator;
GRANT UPDATE (mfg_date,exp_date,status) ON public.batches TO inventory_qc_operator;
GRANT UPDATE (inventory_version) ON public.inventory_items TO inventory_qc_operator;
GRANT INSERT ON public.inventory_qc_releases,public.outbox_events TO inventory_qc_operator;
GRANT EXECUTE ON FUNCTION public.inventory_emit_stock_snapshot(text),
 public.inventory_stock_snapshot_trigger(),public.inventory_batch_revision(),
 public.inventory_sellable_quantity(text,timestamptz) TO inventory_qc_operator;
```
