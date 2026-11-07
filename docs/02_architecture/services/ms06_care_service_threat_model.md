# MS-06 Care Service - Threat Model

> Status: **Implemented-scope review**  
> Updated: 2026-10-11  
> Scope: Media Module, customer/staff Ticket APIs, workers, PostgreSQL, Kafka outbox,
> S3-compatible storage, migration runner, and deployment baseline

## 1. Security objectives

Care Service must preserve:

- confidentiality of customer messages, internal notes, evidence metadata, and signed
  media access URLs;
- integrity of ticket ownership, assignment, transitions, SLA history, audit records,
  idempotency records, and outbox events;
- availability without accepting unsafe writes when an authoritative dependency is
  unavailable;
- traceability of staff access and mutations without copying sensitive content into
  logs, audit payloads, metrics, or Kafka events.

The service is not a public authentication boundary. API Gateway authenticates the
user and forwards a trusted identity plus permissions. Care remains responsible for
validating gateway trust, resource ownership, dynamic assignment scope, payloads,
state transitions, and concurrency.

## 2. Trust boundaries and protected assets

| Boundary | Trusted input | Untrusted or sensitive input |
| --- | --- | --- |
| Browser/client to Gateway | None | All public headers, JSON, IDs, filenames, MIME declarations |
| Gateway to Care | `X-User-ID` and `X-Permissions` only after internal-token verification | Request payloads and resource identifiers remain untrusted |
| Care to PostgreSQL | Service-owned connection and parameterized commands | Persisted customer/staff content remains sensitive |
| Care to object storage | Configured private endpoint, bucket, and credentials | Uploaded bytes, declared metadata, and object contents |
| Care to Kafka | Configured broker/topic | Event consumers and redelivery behavior require separate trust controls |
| Operations/deployment | Secret manager and immutable image are intended trust roots | Manifest placeholders and developer environment values are not production trust roots |

High-value assets include the gateway shared secret, database/S3 credentials,
customer and staff identity, internal notes, evidence objects, signed URLs, assignment
and decision authority, audit history, and unpublished outbox records.

## 3. Threats, current controls, and remaining actions

| Threat | Current controls | Residual risk / required action |
| --- | --- | --- |
| Forged gateway identity headers | Every `/api/v1` route requires `X-Internal-Token`; comparison is constant-time; user ID must be a non-zero UUID. | Production network policy must allow API traffic only from Gateway. Add shared-secret rotation or workload identity/mTLS before broad internal access. |
| IDOR across customers | Customer repositories always scope ticket/media reads and writes by authenticated owner ID; missing ownership returns non-enumerating `404`/conflict behavior. | Add automated HTTP+PostgreSQL integration coverage for all customer resource combinations. |
| Staff privilege escalation | Each staff action has a distinct permission; claim fixes assignee to the actor; replies/transitions enforce current assignment and optimistic version. | Staff-directory/queue contracts are required before arbitrary reassignment. Manager override must be a separate permission and audited path. |
| Internal-note leakage | Notes use a separate table, repository operation, DTO collection, and permission; content is excluded from outbox and audit payloads. | Add response-schema regression tests whenever customer DTOs change. Notification must never consume internal notes. |
| Malicious or oversized upload | Binary bypasses Care HTTP; purpose/MIME/size/checksum/object HEAD checks run before readiness; attachment requires same owner, correct purpose, clean scan, READY state, and unexpired TTL. | `noop` scanning is staging-only and configuration fails in production. Implement production scanner, quarantine, timeout, retry, and incident procedure. |
| Arbitrary URL/SSRF | Business APIs accept immutable `file_id`, not caller-provided URLs. S3 internal/public endpoints come only from deployment configuration. | Restrict egress to approved dependencies and validate production endpoint/DNS policy. Never add URL-fetch behavior to evidence APIs. |
| Signed URL disclosure/reuse | URLs are generated only after authorization, short-lived, not stored, and not logged. Bucket is private and object keys are non-guessable. | Configure log/trace redaction at Gateway and observability backends; use storage access logs and rotate credentials after suspected leakage. |
| Replay or duplicate side effects | Mutations hash payloads under actor/operation/idempotency key; same request replays the resource, changed payload conflicts. Outbox IDs are stable and workers use row locks. | External consumers must deduplicate event IDs. Gateway rate limiting is still required against large key-cardinality attacks. |
| State-race overwrite | Staff claim/transition uses expected version and row locking; media attachment occurs in the business transaction; SLA workers use `SKIP LOCKED` and one-time markers. | Maintain concurrency integration tests for every future decision/refund operation. |
| SQL injection | Runtime values use pgx parameters; enum/state constraints also exist in PostgreSQL. | Review any future dynamic sort/filter identifiers against explicit allowlists. Database role must be least privilege after migrations. |
| HTTP resource exhaustion | JSON body is limited to 1 MiB, unknown/trailing fields are rejected, handler/server timeouts are bounded, and authenticated APIs have a per-actor, per-replica token-bucket safeguard with bounded idle-bucket cleanup. | Configure a distributed Gateway limit before trusted authentication and request concurrency controls. Revisit endpoint-specific limits for Return evidence metadata. |
| Event forgery or interception | Topics and event types are service-configured, payloads exclude note/message/evidence content, and outbox write is atomic with business state. | Production Kafka TLS/SASL, ACLs, schema compatibility checks, consumer inbox/DLQ, and replay authorization remain required. Local plaintext Kafka is not a production template. |
| Secret leakage | Secrets are absent from committed Kubernetes manifests and injected through Secret references; structured logs avoid tokens/URLs/body content. | Use an external secret manager, encryption at rest, rotation, scoped service accounts, and secret-scanning CI. Do not use `kubectl create secret --from-literal` in shared shell history without protected tooling. |
| Unsafe schema rollout | Dedicated runner embeds ordered up migrations, serializes with advisory lock, records immutable checksums, and fails closed on modification. Application startup never migrates. | Pin image by digest, restrict migration DB role/time window, back up first, and require approval for destructive down migrations. |
| Evidence deletion or legal-hold violation | Orphan cleanup excludes `ATTACHED`; failures leave retryable tombstones. | Attached retention/deletion remains blocked on `CARE-MEDIA-009`. No operator may manually delete attached evidence. |
| PII leakage through observability | Metrics use bounded status/outcome labels; HTTP logs record route rather than raw URL and exclude body/headers. Audit for notes/replies contains metadata only. | Establish central retention/redaction rules and test tracing integrations before enabling full production export. |
| Dependency failure causing unsafe acceptance | Readiness covers PostgreSQL/storage; missing Order ownership verification fails closed; outbox preserves committed events during Kafka failure. | Add bounded retry/circuit-breaker policies when gRPC dependencies exist. Define availability behavior in each ratified contract. |

## 4. Explicit production blockers

Production release is blocked until all applicable items are complete:

1. A real malware scanner and quarantine workflow replaces `SCANNER_MODE=noop`.
2. Kubernetes environment overlay supplies a digest-pinned image, real endpoints,
   external secrets, network policy, and Gateway routing.
3. Kafka uses authenticated encrypted transport and least-privilege topic ACLs.
4. Gateway strips caller-supplied trust headers, injects its own values, limits rates,
   and is the only permitted API caller at the network layer.
5. Database migration and runtime roles are separated and least privilege.
6. Backup/restore and migration recovery are exercised in a production-like environment.
7. Retention/legal-hold policy is ratified before attached evidence deletion exists.

Return/refund, Review, external notifications, guest support, and AI introduce new
threats and require an updated review before their APIs are released.

## 5. Verification checklist

- Attempt access to every customer resource as another customer.
- Attempt each staff mutation without permission, with permission but wrong assignment,
  with stale version, and with replayed/changed idempotency payload.
- Attempt upload completion with wrong owner, purpose, MIME, size, checksum, state,
  expiry, and scanner result.
- Verify customer DTOs, logs, metrics, audit rows, traces, and Kafka payloads contain no
  internal-note content, auth headers, signed URLs, or evidence bytes.
- Verify unavailable Order ownership dependency rejects `order_id` rather than accepting
  or revealing another customer's order.
- Verify two migration runners serialize and a modified applied migration fails closed.
- Verify Kafka outage grows a recoverable outbox backlog without rolling back already
  committed business state.
- Verify an object-storage outage causes readiness failure and leaves cleanup retryable.

Security findings must update this document, the delivery plan, affected decisions,
tests, and the operational runbook. Do not close a finding solely with a Gateway or
infrastructure assumption unless its configuration is versioned and verified.
