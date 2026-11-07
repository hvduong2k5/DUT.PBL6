# MS-06 Care Service - Delivery Plan and Traceability

> Status: **Execution checklist**  
> Updated: 2026-10-11
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

### Current implementation checkpoint - 2026-10-11

Implemented in the first vertical slice:

- Care-specific OpenAPI for Media and registered-customer Ticket MVP;
- runnable Go service skeleton with PostgreSQL, Kafka outbox, S3-compatible object storage, Prometheus, and OpenTelemetry wiring;
- initial platform/media/ticket migration and rollback file;
- internal Media Module with direct upload, verification, scanner port, access URL, and ownership/purpose checks;
- customer ticket create/list/detail/message APIs with transactional attachment and outbox writes;
- staff queue/detail, self-claim, internal-note, in-app public-reply, and state-transition APIs with permission enforcement, queue filters, cursor pagination, optimistic concurrency, idempotency, separated data, SLA updates, histories, metadata-only outbox where appropriate, and audit;
- duplicate-safe near-breach and breached SLA workers with configurable polling/window and metadata-only outbox events;
- immutable upload-session expiry and retry-safe orphan-media cleanup;
- Return schema/domain foundation with separate case, immutable line-level decision,
  returned-goods, and refund-obligation lifecycles; no external Return API is exposed;
- unit tests, isolated-schema PostgreSQL migration/repository integration tests, `go vet`, build, JSON schema parse, and OpenAPI lint.

Local Compose verification now covers PostgreSQL, Kafka topic initialization, SeaweedFS S3 compatibility, migration startup, service readiness, direct upload and orphan cleanup, staff queue/detail authorization and audit, concurrent self-claim, internal-note isolation, staff in-app public reply, the ticket state machine, and SLA alert polling. Media cleanup verification proves an expired READY orphan becomes `DELETED`, its object returns `404`, API access returns `409`, attached media remains accessible, and replay/complete cannot extend the original upload deadline. Self-claim proves one `200`/one `409` for competing agents, replay safety across different request IDs, and exactly one assignment, claim audit, assigned event, and idempotency result. Internal-note verification proves `403` without permission, `201` then `200` on replay, `409` for a changed payload, no customer DTO leakage, metadata-only audit, and no note event in outbox. Staff reply verification covers permission and assigned-owner scope, idempotent replay, customer visibility, first-response completion, and content-free audit/outbox metadata. Transition verification covers permission and assigned-staff scope, invalid edges, replay and changed-payload conflicts, `WAITING_CUSTOMER` pause/resume, resolve/reopen/close, seven atomic audit/outbox/idempotency records, and successful publication of all seven Kafka events. SLA verification proves one near-breach and one breached event, publication through Kafka, metadata-only payloads, no duplicates across repeated polls, and exclusion of paused, completed, resolved, and closed milestones. The host race test remains unavailable because its installed C compiler lacks 64-bit race support; normal unit tests, vet, and build pass.

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
- [x] Add an advisory-locked, checksum-validating up-migration runner for deployment jobs.
- [x] Test clean migration and rollback on an isolated PostgreSQL schema.
- [ ] Test migration from a populated previous production version when such a version exists.

### `CARE-120` - Request security and API conventions

- [x] Parse trusted gateway identity context after validating the server-side internal token.
- [x] Implement permission middleware plus customer resource ownership checks for implemented routes.
- [x] Add request ID middleware.
- [x] Propagate a distinct correlation ID across HTTP, audit, outbox, and CloudEvents.
- [x] Add standard error envelope and stable error codes.
- [x] Add body-size limits, timeouts, panic recovery, and safe logging.
- [x] Implement transactional idempotency request hashing and resource-response replay.

### `CARE-130` - Kafka reliability

- [x] Implement transactional outbox writer in business transactions.
- [x] Implement outbox publisher with exponential retry and lag observability.
- [ ] Implement inbox deduplication and consumer transaction boundary.
- [ ] Define DLQ naming, retry bounds, and replay safeguards.
- [x] Propagate HTTP trace context and correlation ID through transactional outbox CloudEvents.

### `CARE-140` - Observability

- [x] Structured Zerolog fields without request/response bodies, tokens, signed URLs, or evidence data.
- [x] Prometheus HTTP, PostgreSQL pool, outbox, SLA worker, and media lifecycle/cleanup metrics.
- [ ] Add Kafka consumer-lag and Return/Review business metrics when those capabilities exist.
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
- [x] Implement S3-compatible presigned upload and download with separate internal and public endpoints.
- [x] Use non-guessable object keys independent of original file name.
- [x] Enforce required private bucket configuration and startup validation.
- [ ] Add Testcontainers S3-compatible storage integration tests.

### `CARE-220` - Upload and access API

- [x] Create upload session with `file_id` and expiry.
- [x] Complete upload by storage HEAD/stat verification.
- [x] Detect/verify MIME, size, and checksum metadata.
- [x] Integrate scanner port and fail configuration closed when production uses the no-op scanner.
- [x] Issue short-lived access URL only after resource authorization.
- [x] Reject cross-owner, wrong-purpose, non-ready, scan-failed/infected, expired, and deleted files from attachment/access paths.

### `CARE-230` - Media workers

- [ ] Scan uploaded files asynchronously where required.
- [x] Expire upload sessions without allowing idempotent replay to extend the original deadline.
- [x] Delete never-attached orphan objects after policy TTL with retry-safe tombstones.
- [ ] Preserve attached evidence until retention/legal-hold policy allows deletion.
- [x] Emit metrics for media lifecycle/scan status and cleanup success/failure.

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

- [x] Add assignments, internal notes, priority/category, and optimistic version to the data model.
- [x] Implement permission-protected queue filters and cursor pagination.
- [x] Implement staff detail with public messages and internal notes returned in separate collections.
- [x] Audit staff access to complete ticket context.
- [x] Implement idempotent self-claim with `expected_version`, assignment/status history, audit, and outbox writes.
- [x] Implement idempotent internal-note writes with a distinct permission and metadata-only audit.
- [x] Implement assigned-agent in-app public replies with first-response SLA completion and metadata-only audit/outbox.
- [ ] Integrate external email/push delivery and retry after the Notification Service contract is ratified.
- [ ] Implement assignment/reassignment to another staff member or queue after target contracts are ratified.
- [x] Implement allowed state transitions, assigned-staff scope, optimistic concurrency, and reason requirements.
- [ ] Audit assignment, priority, and transition mutations.

### `CARE-320` - SLA

- [x] Store and complete first-response and resolution milestones independently.
- [x] Implement resolution-SLA pause/resume rules from `CARE-TICKET-005` as part of state transitions.
- [x] Add near-breach and breached workers without duplicate alerts.
- [ ] Delegate customer/staff alert delivery to Notification.
- [x] Preserve milestone and breach history after resolution/closure.

Ticket phase exit criteria:

- `US-CS-01` is production-ready after `CARE-300`.
- `US-CS-02` through `US-CS-05` are not marked complete until `CARE-310/320` and dependent contracts pass.
- Guest and AI requirements remain explicitly deferred, not partially exposed.

## 6. Return and refund delivery

Contract gate: do not start externally integrated refund execution before `CARE-RETURN-012` and `CARE-RETURN-013` are ratified.

### `CARE-400` - Return case intake

- [x] Add case, lines, evidence references, immutable decision scope, status history, returned-goods projection, and refund-obligation schemas.
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

- [x] Threat model IDOR, SSRF, malicious upload, presigned URL leakage, event forgery, replay, privilege escalation, internal-note leakage, schema rollout, and dependency failure.
- [x] Add a bounded per-actor token-bucket safeguard inside Care, excluding health/metrics and emitting `429`, `Retry-After`, and a rejection metric.
- [ ] Configure the distributed/public-edge rate limit at Gateway; the in-process limit is per replica and cannot replace it.
- [ ] Add timeouts, bounded retries, and circuit-breaking policy for gRPC/storage calls.
- [ ] Add database backup/restore verification and migration rollback procedure.
- [ ] Verify production scanner and private-bucket policies.

### `CARE-610` - Deployment

- [x] Add local Compose dependencies for PostgreSQL, Kafka, and SeaweedFS S3-compatible storage. Observability backends remain deployment work.
- [x] Add a staging Kubernetes baseline with Deployment, Service, ConfigMap/Secret references, probes, resources, HPA, and disruption policy.
- [x] Add a pre-rollout Kubernetes migration Job using the same application image.
- [ ] Add environment overlays, digest-pinned registry image, network policy, and Gateway/Ingress after deployment values are provided.
- [x] Add baseline dashboard and alerts for API errors/latency, PostgreSQL saturation, ticket backlog, SLA, outbox, and media cleanup.
- [ ] Add consumer-lag, production scanner, and refund pending/failure panels after those capabilities exist.

### `CARE-620` - Runbook

- [x] Document outbox backlog recovery.
- [ ] Document inbox/DLQ inspection and safe replay.
- [x] Document stuck upload/scan cleanup for implemented states; production scanner recovery remains open.
- [ ] Document refund reconciliation without re-execution.
- [ ] Document data export/redaction/deletion versus retention/legal hold.
- [x] Document dependency degradation behavior.
- [x] Document migration rollout/rollback safety and backup/restore verification criteria.

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

