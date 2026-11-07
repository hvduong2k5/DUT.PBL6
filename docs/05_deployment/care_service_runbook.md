# Care Service Operations Runbook

> Service: `MS-06 care-service`  
> Updated: 2026-10-11  
> Scope: implemented Media and Ticket capabilities only

This runbook covers failure diagnosis and recovery for the currently implemented
Care Service. Return/refund, external notification delivery, inbox consumers, DLQ,
and production malware scanning are not implemented yet; procedures for those
capabilities must be added with their code and contracts.

## 1. Safety rules

- Treat PostgreSQL as the source of truth for Care-owned state.
- Never edit a published outbox row or create a replacement event by hand.
- Never set a media record to `READY`, `ATTACHED`, or `DELETED` manually.
- Never expose `CARE_INTERNAL_TOKEN`, S3 credentials, signed URLs, message content,
  internal notes, or evidence data in tickets, dashboards, or copied logs.
- Take a database backup before any recovery action that mutates production data.
- Record operator, reason, affected IDs, query/result counts, and timestamps for
  every manual action.

## 2. Initial triage

1. Check liveness: `GET /health/live` must return `200` even when dependencies fail.
2. Check readiness: `GET /health/ready` reports PostgreSQL and object-storage state.
3. Check recent structured logs using `request_id`; do not search by message/evidence
   content.
4. Check `/metrics` for HTTP failures, outbox lag, SLA alerts, media state counts,
   and cleanup failures.
5. Confirm whether the incident affects writes, reads, event publication, uploads,
   or only background processing before restarting anything.

## 3. Outbox backlog recovery

The business write and outbox insert are atomic. An unpublished row means the
business change is committed but Kafka publication is pending. The worker retries
with bounded exponential delay and uses the outbox UUID as the CloudEvents event ID.
Consumers must deduplicate by this ID.

Read-only diagnosis:

```sql
SELECT COUNT(*) AS pending,
       MIN(created_at) AS oldest_created_at,
       MAX(attempts) AS max_attempts
FROM outbox_events
WHERE published_at IS NULL;

SELECT id, event_type, topic, partition_key, attempts,
       created_at, next_attempt_at, LEFT(COALESCE(last_error, ''), 300) AS last_error
FROM outbox_events
WHERE published_at IS NULL
ORDER BY created_at
LIMIT 100;
```

Recovery:

1. Verify Kafka broker and topic availability from the service network.
2. Verify `KAFKA_BROKERS` and `KAFKA_CARE_TOPIC`; do not change event types or
   partition keys during recovery.
3. Restore Kafka connectivity and leave the worker running. Eligible rows retry
   automatically.
4. If delayed `next_attempt_at` values make recovery unacceptably slow, an authorized
   database operator may advance only pending rows after the broker is healthy:

```sql
BEGIN;
UPDATE outbox_events
SET next_attempt_at = NOW()
WHERE published_at IS NULL
  AND next_attempt_at > NOW();
COMMIT;
```

5. Confirm pending count drains, `published_at` is populated, and consumers observe
   no duplicate side effects.

Do not reset `attempts`, clear `last_error`, set `published_at`, or republish payloads
from a separate script. There is currently no DLQ; persistent poison-event handling
is a release gap tracked by `CARE-130`.

## 4. Stuck upload and media cleanup

Lifecycle interpretation:

- `PENDING_UPLOAD`: session exists; object may not exist yet.
- `UPLOADED` / `SCANNING`: reserved for asynchronous scanning work.
- `READY`: clean unattached object that may be referenced before expiry.
- `ATTACHED`: referenced by a business aggregate and excluded from orphan cleanup.
- `REJECTED`: verification or scanning rejected the object.
- `EXPIRED`: orphan selected for deletion; deletion may be retrying.
- `DELETED`: storage removal succeeded and tombstone was committed.

Read-only diagnosis:

```sql
SELECT status, scan_status, COUNT(*)
FROM media_files
GROUP BY status, scan_status
ORDER BY status, scan_status;

SELECT id, owner_id, purpose, status, scan_status, object_key,
       created_at, expires_at, updated_at
FROM media_files
WHERE status IN ('PENDING_UPLOAD', 'UPLOADED', 'SCANNING', 'READY', 'REJECTED', 'EXPIRED')
  AND expires_at <= NOW()
ORDER BY expires_at
LIMIT 100;
```

Recovery:

1. Check `care_media_cleanup_objects_total{outcome="failed"}` and storage errors.
2. Restore object-storage connectivity and credentials.
3. Leave rows in `EXPIRED`; the cleanup worker retries them automatically.
4. Confirm successful retries increment the `deleted` counter and set database state
   to `DELETED` with `deleted_at`.
5. For an `ATTACHED` item, stop. It is intentionally excluded until retention and
   legal-hold policy `CARE-MEDIA-009` is ratified.

Do not manually delete an attached object, reuse an old signed URL, extend
`expires_at`, or mark an object clean. A `SCANNING` record in the current build is a
signal for investigation because production asynchronous scanning is not released.

## 5. SLA alert worker

SLA alert markers and outbox events are committed in the same transaction. Paused,
completed, resolved, and closed milestones are excluded.

```sql
SELECT milestone_type,
       COUNT(*) FILTER (WHERE completed_at IS NULL AND paused_at IS NULL) AS active,
       COUNT(*) FILTER (WHERE near_breach_alerted_at IS NOT NULL) AS near_alerted,
       COUNT(*) FILTER (WHERE breach_alerted_at IS NOT NULL) AS breached_alerted
FROM ticket_sla_milestones
GROUP BY milestone_type;
```

If alerts stop, check PostgreSQL and outbox health first. Do not clear alert timestamps
to force delivery; that creates a second logical alert. Notification delivery is not
yet integrated, so Kafka publication currently proves intent publication only.

## 6. Dependency degradation

| Dependency | Expected behavior | Operator action |
| --- | --- | --- |
| PostgreSQL unavailable at startup | Process fails fast; readiness is unavailable. | Restore database, verify migrations, then restart. |
| PostgreSQL unavailable at runtime | Business calls fail safely; workers retain no in-memory source of truth. | Restore database and verify pending outbox/media/SLA work. |
| Object storage unavailable at startup | Process fails while ensuring the private bucket. | Restore endpoint/credentials/bucket policy, then restart. |
| Object storage unavailable at runtime | Readiness returns `503`; media operations fail; cleanup remains retryable. | Restore storage and verify `EXPIRED` rows drain. |
| Kafka unavailable | Business writes continue with pending outbox rows; publisher retries. | Restore broker/topic and monitor backlog drain. |
| OTLP collector unavailable | Core business processing continues; trace export may be lost. | Restore collector and validate exporter logs. |
| Order ownership dependency absent | Ticket requests with `order_id` fail closed. | Do not bypass ownership verification; wait for the ratified contract. |

## 7. Migration and rollback

Production rollout order:

1. Back up the Care database and verify the backup is readable.
2. Run the versioned `*.up.sql` files using a dedicated migration job.
3. Verify schema objects and constraints.
4. Deploy application instances only after the migration job succeeds.
5. Run readiness and a non-destructive API smoke test.

The application never mutates schema at startup. Rollback SQL is validated by the
isolated-schema integration test, but using a down migration in production deletes
data for `000001`; it is therefore a disaster-recovery operation, not a normal
application rollback. Prefer rolling the application forward or restoring a tested
backup. Run destructive down migrations only with incident/change approval and a
verified recovery point.

Local migration verification:

```powershell
$env:CARE_TEST_DATABASE_URL = "postgres://care_user:care_password@localhost:5438/care_db?sslmode=disable"
go test -count=1 -run '^TestIntegration' ./internal/repository/postgres
Remove-Item Env:CARE_TEST_DATABASE_URL
```

The test creates and drops only a random `care_it_*` schema in the configured test
database. Never point `CARE_TEST_DATABASE_URL` at production.

## 8. Backup and restore verification

At minimum, a production readiness exercise must prove:

1. a consistent PostgreSQL backup can be restored into an isolated environment;
2. migration version and row counts match the recovery point;
3. ticket/message/note separation remains intact;
4. pending outbox rows remain pending and published rows retain event IDs;
5. media metadata still points to the expected private bucket objects;
6. readiness and read-only smoke tests pass before traffic is enabled.

Object storage requires its own versioning/backup policy. Restoring database metadata
without matching objects is not a complete Care recovery.

## 9. Current procedural gaps

- Inbox/DLQ inspection and replay: consumer framework is not implemented.
- Refund reconciliation: refund workflow and ownership contract are not implemented.
- Attached-media deletion/export/redaction: retention and legal-hold policy is not
  ratified.
- Production malware recovery: scanner adapter and quarantine operations are not
  implemented.
- Notification delivery reconciliation: Notification Service contract is not
  ratified.

These gaps must remain visible in the delivery plan and must not be handled with
manual database state changes.
