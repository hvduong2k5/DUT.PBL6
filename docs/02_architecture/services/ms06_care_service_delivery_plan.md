# MS-06 Care Service - Delivery Plan and Traceability

> Status: **Execution checklist**  
> Updated: 2026-10-10  
> Design: [Technical baseline](./ms06_care_service_design.md)  
> Decisions: [Decision log](./ms06_care_service_decision_log.md)

## 1. How to use this document

Every implementation pull request should:

- name one or more work-item IDs from this file;
- name affected decision IDs;
- update the checklist and traceability when scope changes;
- include tests for the listed acceptance criteria;
- avoid declaring a phase complete while a required contract gate is open.

Checked items have implementation evidence in the branch. Unchecked items remain planned or require an external contract/verification gate.

### Current implementation checkpoint - 2026-10-10

Implemented in the first vertical slice:

- Care-specific OpenAPI for Media and registered-customer Ticket MVP;
- runnable Go service skeleton with PostgreSQL, Kafka outbox, MinIO/S3, Prometheus, and OpenTelemetry wiring;
- initial platform/media/ticket migration and rollback file;
- internal Media Module with direct upload, verification, scanner port, access URL, and ownership/purpose checks;
- customer ticket create/list/detail/message APIs with transactional attachment and outbox writes;
- unit tests, `go vet`, build, JSON schema parse, and OpenAPI lint.

Verification limitation: Docker Desktop was not running in the implementation environment, so PostgreSQL/MinIO/Kafka integration tests and migration execution remain open. The host race test was also unavailable because its installed C compiler lacks 64-bit race support; normal unit tests, vet, and build passed.

## 2. Contract and documentation gate

### `CARE-000` - Ratify the baseline

- [ ] Review `CARE-ADR-001` through `CARE-ADR-010`.
- [ ] Obtain product confirmation for configurable policy defaults.
- [ ] Obtain cross-team confirmation for all `PROPOSED` ownership decisions.
- [ ] Mark superseded MongoDB references or link them to `CARE-ADR-002`.
- [ ] Confirm API base path and port `8006` with Gateway/deployment owners.

Exit criteria:

- No implementation-critical ownership question is represented only in chat.
- Every unresolved cross-service item has an owner and target date.

### `CARE-001` - Normalize public contracts

- [x] Create a Care OpenAPI source of truth rather than implementing directly from the legacy monolithic section.
- [ ] Replace `evidence_media_urls` with `evidence_file_ids` when Return endpoints are added; the new Media/Ticket contract already establishes the `file_id` direction.
- [ ] Replace review `media_urls` with `media_file_ids`.
- [x] Add media upload-session and access endpoints.
- [x] Add `Idempotency-Key` to implemented retryable create/action operations.
- [x] Standardize error envelope, pagination, request ID, and conflict responses for the implemented contract.
- [ ] Align stable/draft frontend contracts with backend OpenAPI.

Exit criteria:

- OpenAPI lints successfully.
- Contract examples contain no arbitrary private evidence URL input.
- Customer and admin operations have explicit authentication and permission requirements.

### `CARE-002` - Ratify internal contracts

- [ ] Add or extend Order Protobuf for Care order context, ownership, return eligibility, and review eligibility.
- [ ] Confirm trusted identity propagation for Fulfillment `GetPackingVideoUrl`.
- [ ] Version `ReturnTicketApprovedEvent` with line-level and money data.
- [ ] Define returned-goods inspection request/result schemas and their owners.
- [ ] Define refund requested/processed/failed schemas and the canonical publisher.
- [ ] Define notification intents/events required by ticket and return workflows.
- [ ] Validate all event schemas and backward-compatibility rules.

Exit criteria:

- Provider teams accept request/response/event ownership.
- Care has consumer contract tests or fixtures for every dependency.

## 3. Foundation phase

### `CARE-100` - Service skeleton

- [x] Create `services/care-service` Go module using the repository's service conventions.
- [x] Add `cmd/server`, config, domain/application/infrastructure boundaries, and graceful shutdown.
- [x] Add Dockerfile, `.dockerignore`, local environment example, Make targets, and README.
- [x] Expose `/health/live`, `/health/ready`, and `/metrics`.

### `CARE-110` - PostgreSQL and migrations

- [x] Create private `care_db` configuration.
- [x] Add versioned service-owned migrations and rollback file; the production migration job remains deployment work.
- [x] Add platform tables: idempotency, inbox, outbox, audit.
- [x] Keep pgx inside the PostgreSQL adapter and out of domain/use-case ports.
- [ ] Test clean migration, rollback policy, and migration from the previous version.

### `CARE-120` - Request security and API conventions

- [x] Parse trusted gateway identity context after validating the server-side internal token.
- [x] Implement permission middleware plus customer resource ownership checks for implemented routes.
- [x] Add request ID middleware.
- [ ] Propagate a distinct correlation ID across HTTP, outbox, and downstream calls.
- [x] Add standard error envelope and stable error codes.
- [x] Add body-size limits, timeouts, panic recovery, and safe logging.
- [x] Implement transactional idempotency request hashing and resource-response replay.

### `CARE-130` - Kafka reliability

- [x] Implement transactional outbox writer in business transactions.
- [x] Implement outbox publisher with exponential retry and lag observability.
- [ ] Implement inbox deduplication and consumer transaction boundary.
- [ ] Define DLQ naming, retry bounds, and replay safeguards.
- [ ] Propagate trace and correlation context.

### `CARE-140` - Observability

- [x] Structured Zerolog fields without request/response bodies, tokens, signed URLs, or evidence data.
- [ ] Prometheus HTTP, database, Kafka, worker, and business metrics.
- [ ] OpenTelemetry traces for HTTP, pgx, gRPC, Kafka, and storage calls.
- [x] Readiness checks for PostgreSQL and object storage without turning liveness into a dependency check.

Foundation definition of done:

- Unit tests, race test, lint, build, migration integration test, and container smoke test pass.
- Duplicate idempotency keys and Kafka redelivery cause no duplicate side effects.
- Logs and traces do not expose tokens, signed URLs, bank data, or evidence contents.

## 4. Internal Media Module phase

### `CARE-200` - Media persistence and policy

- [x] Add `media_files`, `media_references`, and scan-result migrations.
- [x] Model purpose, state, owner, provider locator, size, MIME, checksum, retention, and optimistic version.
- [x] Implement purpose- and MIME-specific limits from `CARE-MEDIA-*` decisions.
- [x] Prevent mutation of provider locator/checksum through the application repository after attachment.

### `CARE-210` - S3-compatible storage adapter

- [x] Define application-owned storage interface.
- [x] Implement MinIO/S3 presigned upload and download.
- [x] Use non-guessable object keys independent of original file name.
- [x] Enforce required private bucket configuration and startup validation.
- [ ] Add Testcontainers MinIO integration tests.

### `CARE-220` - Upload and access API

- [x] Create upload session with `file_id` and expiry.
- [x] Complete upload by storage HEAD/stat verification.
- [x] Detect/verify MIME, size, and checksum metadata.
- [x] Integrate scanner port and fail configuration closed when production uses the no-op scanner.
- [x] Issue short-lived access URL only after resource authorization.
- [x] Reject cross-owner, wrong-purpose, non-ready, scan-failed/infected, expired, and deleted files from attachment/access paths.

### `CARE-230` - Media workers

- [ ] Scan uploaded files asynchronously where required.
- [ ] Expire upload sessions.
- [ ] Delete never-attached orphan objects after policy TTL.
- [ ] Preserve attached evidence until retention/legal-hold policy allows deletion.
- [ ] Emit metrics for pending, rejected, scan failure, and cleanup failure.

Media definition of done:

- Binary never passes through a Care business endpoint.
- A business aggregate can attach only a ready, clean, same-owner, correct-purpose `file_id`.
- Signed access URLs expire and are never stored or fully logged.

## 5. Ticket delivery

### `CARE-300` - Registered-customer Ticket MVP

- [x] Add ticket, message, status history, SLA milestone, internal-note, assignment, and attachment schemas.
- [x] Implement create ticket with idempotency.
- [x] Implement customer's list/detail views with ownership checks.
- [x] Implement public customer message flow.
- [x] Attach only ready, clean, same-owner `TICKET_ATTACHMENT` file IDs.
- [x] Publish `TicketCreated` and customer-message events through outbox; later transitions must add their own event schema.

MVP acceptance scenarios:

- Same idempotency key and same request returns the same ticket.
- Same key and different request returns conflict.
- Customer A cannot read or attach media to Customer B's ticket.
- Customer DTO never includes internal notes.
- An unverified order reference is rejected without leaking whether another customer's order exists.

### `CARE-310` - Agent queue and assignment

- [ ] Add assignments, internal notes, priority/category, and optimistic version.
- [ ] Implement queue filters and cursor pagination.
- [ ] Implement claim/assign/reassign with conflict handling.
- [ ] Implement allowed state transitions and reason requirements.
- [ ] Implement staff public reply separately from internal note.
- [ ] Audit assignment, priority, transition, and sensitive-context access.

### `CARE-320` - SLA

- [ ] Store first-response and resolution milestones independently.
- [ ] Implement pause/resume rules from `CARE-TICKET-005`.
- [ ] Add near-breach and breached workers without duplicate alerts.
- [ ] Delegate customer/staff alert delivery to Notification.
- [ ] Preserve milestone and breach history after resolution/closure.

Ticket phase exit criteria:

- `US-CS-01` is production-ready after `CARE-300`.
- `US-CS-02` through `US-CS-05` are not marked complete until `CARE-310/320` and dependent contracts pass.
- Guest and AI requirements remain explicitly deferred, not partially exposed.

## 6. Return and refund delivery

Contract gate: do not start externally integrated refund execution before `CARE-RETURN-012` and `CARE-RETURN-013` are ratified.

### `CARE-400` - Return case intake

- [ ] Add case, lines, evidence references, status history, and decision schemas.
- [ ] Query Order for ownership, immutable lines, eligibility, quantity, timestamps, and refundable ceiling.
- [ ] Enforce one non-conflicting active return quantity per order line.
- [ ] Create return with `evidence_file_ids` and idempotency.
- [ ] Store timestamped minimum source snapshot.
- [ ] Implement customer list/detail/supplement APIs.

### `CARE-410` - Review and decision workflow

- [ ] Implement `NEEDS_INFORMATION` and `REVIEWING` transitions.
- [ ] Enforce Manager permission for approve/partial approve/reject.
- [ ] Store immutable decision scope, reasons, quantities, amount, currency, and return-required flag.
- [ ] Implement packing-video access through Fulfillment and audit it.
- [ ] Publish versioned decision event via outbox.

### `CARE-420` - Returned goods

- [ ] Track reverse-shipment reference without owning shipment state.
- [ ] Request/consume inspection according to ratified ownership.
- [ ] Deduplicate inspection events.
- [ ] Project Inventory disposition; never directly restock.
- [ ] Keep `return case`, `returned goods`, and `refund obligation` states independent.

### `CARE-430` - Refund projection

- [ ] Create one refund obligation per approved decision scope.
- [ ] Enforce amount not exceeding the latest authoritative ceiling and reservation rule.
- [ ] Publish refund request/approved event according to the ratified saga.
- [ ] Consume success/failure idempotently.
- [ ] Mark Care refund projection successful only after canonical result.
- [ ] Support controlled retry without a second financial obligation.
- [ ] Reconcile pending/failed obligations and expose operational metrics.

Return definition of done:

- Partial quantities and partial amounts are traceable by order line.
- Approval never implies goods received or refund completed.
- Goods received never implies stock sellable.
- Redelivery/concurrency tests demonstrate no duplicate case, decision, inspection, or refund obligation.
- All financial actions have actor, reason, before/after state, event ID, and correlation ID.

## 7. Review delivery - Phase 2

### `CARE-500` - Verified review

- [ ] Consume/query authoritative review eligibility.
- [ ] Enforce one review per customer/order line and rating 1-5.
- [ ] Create public privacy-safe DTOs.
- [ ] Implement edit window and immutable versions.
- [ ] Attach only clean `REVIEW_MEDIA` file IDs.

### `CARE-510` - Moderation and reply

- [ ] Hide/restore review or individual media with reason and permission.
- [ ] Use optimistic version for moderation.
- [ ] Add one versioned official reply.
- [ ] Add negative-review queue without automatic ticket creation.
- [ ] Allow idempotent creation of a linked ticket by authorized staff.

### `CARE-520` - Rating aggregate

- [ ] Maintain published-only sum/count/distribution by product and SKU.
- [ ] Update on create/edit/hide/restore without double counting.
- [ ] Provide a rebuild job from review source records.
- [ ] Publish review events for Catalog/Analytics through outbox.

Review definition of done:

- Verified status cannot be forged by request payload.
- Hidden content does not affect public ratings.
- Aggregate rebuild equals incremental aggregate.
- Public response contains no order/contact/private customer data.

## 8. Hardening and operations

### `CARE-600` - Security and resilience

- [ ] Threat model IDOR, SSRF, malicious upload, presigned URL leakage, event forgery, replay, privilege escalation, and internal-note leakage.
- [ ] Add rate limits at Gateway and service safeguards where required.
- [ ] Add timeouts, bounded retries, and circuit-breaking policy for gRPC/storage calls.
- [ ] Add database backup/restore verification and migration rollback procedure.
- [ ] Verify production scanner and private-bucket policies.

### `CARE-610` - Deployment

- [x] Add local Compose dependencies for PostgreSQL, Kafka, and MinIO. Observability backends remain deployment work.
- [ ] Add Kubernetes Deployment, Service, ConfigMap/Secret references, probes, resources, HPA, and disruption policy.
- [ ] Add dashboards and alerts for API errors, latency, SLA, outbox, consumer lag, media scan, refund pending/failure.

### `CARE-620` - Runbook

- [ ] Document outbox backlog recovery.
- [ ] Document inbox/DLQ inspection and safe replay.
- [ ] Document stuck upload/scan cleanup.
- [ ] Document refund reconciliation without re-execution.
- [ ] Document data export/redaction/deletion versus retention/legal hold.
- [ ] Document dependency degradation behavior.

## 9. Requirement traceability

| Requirement | Planned work | Contract/data | Key verification |
| --- | --- | --- | --- |
| `US-CS-01`, `FR-CS-01~04` | `CARE-300` | Ticket OpenAPI; ticket/message/media tables | Idempotent create; ownership; safe order linking; no internal-note leak |
| `US-CS-02`, `FR-CS-05~06` | `CARE-310` | Admin queue/actions | Concurrent claim conflict; assignment audit |
| `US-CS-03`, `FR-CS-07~10` | `CARE-310` | Message/note separation; Notification contract | Public/internal isolation; outbound retry dedupe |
| `US-CS-04`, `FR-CS-11~14` | `CARE-002`, `CARE-310` | Order/Profile/Fulfillment clients | Timestamped snapshot; PII masking; packing-video scope audit |
| `US-CS-05`, `FR-CS-15~17` | `CARE-320` | SLA milestones; Notification event | Pause rules; one alert per milestone; history preserved |
| `US-CS-06`, `FR-CS-18~20` | Deferred Phase 2 | Inventory expiry projection | Care never changes stock or selling eligibility |
| `US-AI-02`, `US-AI-06` | Deferred | None in MVP | No AI endpoint/permission path exists |
| `US-RET-01`, `FR-RET-01~02` | Primarily Order; later Care escalation | Cancellation ownership contract | Care does not duplicate simple cancellation |
| `US-RET-02`, `FR-RET-03~04` | `CARE-400` | Order eligibility; return tables | Owner/window/line/quantity/active-case constraints |
| `US-RET-03`, `FR-RET-05~06` | `CARE-200~230`, `CARE-400` | Media session and evidence refs | Ready/clean/owner/purpose checks; immutable evidence |
| `US-RET-04`, `FR-RET-07~08` | `CARE-002`, `CARE-410` | Context clients; request-information API | Permission and minimum snapshot; no approval by support agent |
| `US-RET-05`, `FR-RET-09~11` | `CARE-410~420` | Versioned decision/inspection events | Manager-only decision; separate lifecycles; quarantine default |
| `US-RET-06`, `FR-RET-12~16`, `FR-RET-20~22` | `CARE-430` | Refund contract; obligation/history/outbox | Ceiling; original method; no duplicate execution; audit/reconcile |
| `US-RET-07`, `FR-RET-17~18` | `CARE-410` | Fulfillment gRPC | Exact order/package, permission, TTL, access audit |
| `US-REV-01`, `FR-REV-01~06` | `CARE-500` | Review eligibility; reviews/versions | Verified ownership; 1-5; uniqueness; edit window |
| `US-REV-02`, `FR-REV-07~08` | `CARE-200~230`, `CARE-500/510` | Review media refs | Upload limits; independent media moderation |
| `US-REV-03`, `FR-REV-09` | `CARE-500` | Public review DTO | Verified label; no transaction/PII leak |
| `US-REV-04`, `FR-REV-11~12` | `CARE-510` | Negative queue/link | Rating 1-2 queue; no accidental duplicate ticket |
| `US-REV-05`, `FR-REV-13~16` | `CARE-510` | Moderation/reply/version/audit | Permission; reason; optimistic conflict; history |
| `US-REV-06`, `FR-REV-10`, `FR-REV-17~19` | `CARE-520` | Rating aggregates/events | Published-only; rebuild consistency; zero versus no data |

## 10. Known contract conflicts to resolve

| Conflict | Current evidence | Required resolution |
| --- | --- | --- |
| MongoDB versus PostgreSQL | Older architecture says MongoDB. | `CARE-ADR-002` is the Care implementation baseline; update parent docs when approved. |
| Arbitrary media URL versus `file_id` | Legacy OpenAPI uses `media_urls`/`evidence_media_urls`; draft frontend contract uses file references. | Backend OpenAPI must adopt `file_id`; compatibility strategy must be explicit. |
| Review `DELIVERED` versus `COMPLETED` | Bounded-context and OpenAPI wording differ. | Order returns `review_eligible_at` and reason; Care does not infer raw status. |
| Return aggregate conflates goods/refund | Older state model uses `APPROVED -> RETURNED -> REFUNDED`. | Implement separate case, goods, and refund lifecycles. |
| Inspection publisher | HLD text and service table disagree. | Inventory should own inspection/disposition result; ratify event owner. |
| Refund owner/publisher | Requirements and HLD mention Payment, Order, and Finance differently. | Ratify `CARE-RETURN-012` before integrated refund release. |
| Return approved event detail | Existing schema has no lines/amounts. | Publish a new version; do not mutate v1 semantics silently. |
| Guest support | Requirement permits Guest; secure proof flow is undefined. | Registered-only MVP; design OTP/proof session before enabling Guest. |

## 11. Global Definition of Done

A feature is complete only when:

- requirement, decision, API/event contract, migration, code, and tests are traceable;
- authorization covers actor permission and resource ownership/scope;
- retry and concurrency behavior are tested;
- no cross-service database access exists;
- business write and outbox write are atomic;
- consumers are idempotent;
- metrics/logs/traces and stable error codes are present;
- PII, internal notes, evidence, tokens, and signed URLs are protected;
- migrations and rollback/forward-fix strategy are documented;
- OpenAPI/Protobuf/event schemas validate;
- unit, integration, contract, race, and smoke tests appropriate to the change pass;
- operational failure and replay procedures are documented when the feature introduces a worker or external dependency.

## 12. Suggested implementation order

```text
CARE-000 -> CARE-001/002
         -> CARE-100/110/120/130/140
         -> CARE-200/210/220/230
         -> CARE-300
         -> CARE-310/320
         -> CARE-400/410/420/430
         -> CARE-500/510/520
         -> CARE-600/610/620
```

External contract work can proceed in parallel with foundation work, but a feature cannot cross its contract gate until the owning teams ratify it.

