# MS-06 Care Service - Technical Design Baseline

> Status: **Accepted as implementation baseline for Care Service**  
> Updated: 2026-10-10  
> Scope: architecture and implementation direction; no runtime implementation is implied by this document alone.

## 1. Purpose and source of truth

This document is the technical baseline for `MS-06 care-service`. Implementation, API contracts, database migrations, event schemas, tests, deployment manifests, and runbooks must be checked against it.

Companion documents:

- [Decision log and policy defaults](./ms06_care_service_decision_log.md)
- [Delivery plan and traceability](./ms06_care_service_delivery_plan.md)

Reference requirements:

- [`EPIC_14_Return_Refund.md`](../../01_requirements/epics/EPIC_14_Return_Refund.md)
- [`EPIC_15_Review_Rating.md`](../../01_requirements/epics/EPIC_15_Review_Rating.md)
- [`EPIC_16_Customer_Service.md`](../../01_requirements/epics/EPIC_16_Customer_Service.md)
- [`service_boundary.md`](../service_boundary.md)
- [`bounded_context.md`](../bounded_context.md)
- [`domain_design.md`](../domain_design.md)

When documents conflict, use this order until the repository establishes a global policy:

1. Accepted ADR/decision recorded in the decision log.
2. Approved requirement and business rule.
3. Versioned OpenAPI, Protobuf, and event schema implemented and verified by tests.
4. This technical design.
5. Older high-level diagrams and descriptive documents.
6. Draft/generated frontend contracts.

Draft frontend contracts currently express the desired `fileId` direction, but they are not authoritative until reviewed and committed.

## 2. Service identity and boundaries

| Item | Value |
| --- | --- |
| Service | `MS-06 care-service` |
| Bounded context | `BC-06 Customer Care & Review` |
| Domain type | Supporting Domain |
| HTTP port | `8006` |
| Database | PostgreSQL 16, private database `care_db` |
| Public transport | REST through API Gateway |
| Internal transport | gRPC clients and Kafka |
| Event topic | `care.events.v1` |

Care Service owns:

- support tickets, public messages, internal notes, assignments, priorities, and SLA milestones;
- return cases, return lines, customer evidence, review decisions, returned-goods projections, and refund-obligation projections;
- verified reviews, review versions, moderation, official replies, and rating aggregates;
- metadata for files uploaded for Care purposes;
- Care audit history, idempotency records, inbox records, and outbox events.

Care Service does not own:

- authentication, accounts, roles, or permission definitions;
- order, payment, shipment, inventory, or packing-video state;
- the physical refund execution;
- warehouse inspection or stock disposition;
- notification delivery;
- raw packing video;
- binary file contents stored in object storage.

There are no cross-service database reads or foreign keys. External IDs are soft references accompanied by the minimum immutable snapshot required for audit and display.

## 3. Delivery scope

### 3.1 MVP

The first release includes:

1. Service foundation: configuration, PostgreSQL, migrations, authentication context, authorization, standard errors, idempotency, inbox/outbox, health, logs, metrics, and traces.
2. Internal Media Module: upload session, upload verification, metadata, authorization, and short-lived access URL.
3. Registered-customer Ticket MVP: create, read, public conversation, attachments, and basic state tracking.
4. Agent operations and SLA after the customer flow is stable.
5. Return/refund workflow after the required Order, Inventory, Fulfillment, and Finance contracts are ratified.

### 3.2 Phase 2

- verified reviews, media, moderation, official reply, and rating aggregation;
- richer agent queue, multi-channel conversation, and reporting;
- packing-video investigation flow;
- advanced return shipping and exchange scenarios.

### 3.3 Explicitly excluded from MVP

- AI chatbot, automatic AI triage, or AI-generated replies;
- Elasticsearch;
- Temporal or another workflow engine;
- a separate Media microservice;
- Guest ticket creation without a verified OTP/proof session;
- Care executing refunds or changing inventory;
- Redis unless a measured requirement justifies it;
- exchange/replacement-order orchestration.

## 4. Technology baseline

| Concern | Selection | Notes |
| --- | --- | --- |
| Language | Go | Use the repository-supported stable Go version. |
| Architecture | Clean/Hexagonal | Domain and use cases do not import transport or infrastructure packages. |
| HTTP | `go-chi/chi/v5` | Standard `net/http` middleware model. |
| Validation | `go-playground/validator/v10` plus domain validation | Transport validation never replaces invariants. |
| Database driver | `pgx/v5` | Native PostgreSQL pool and transactions. |
| SQL generation | `sqlc`, if consistent with the Profile Service baseline | Raw SQL remains reviewable. |
| Migration | Repository-standard migration tool | Migrations are owned by Care Service. |
| Messaging | `segmentio/kafka-go` | CloudEvents envelope, synchronous acknowledgements, bounded retries. |
| RPC | `grpc-go` and Protobuf | Care starts as a gRPC client; add a server only for a confirmed consumer. |
| Object storage | SeaweedFS S3-compatible local, S3-compatible production | Private bucket by default; configure internal and public signing endpoints separately. |
| Logging | Zerolog | Structured logs, PII-safe fields. |
| Metrics | Prometheus | Business and infrastructure metrics. |
| Tracing | OpenTelemetry | OTLP export; traces and metrics are required. |
| Tests | Testify and Testcontainers | PostgreSQL, Kafka, and S3-compatible storage integration tests. |

PostgreSQL replaces the MongoDB suggestion in older architecture documents. The reason is the service's need for transactional state transitions, unique and partial indexes, monetary constraints, optimistic concurrency, audit history, idempotency, and transactional outbox processing.

## 5. Target package layout

```text
services/care-service/
├── cmd/server/
├── config/
├── internal/
│   ├── domain/
│   ├── usecase/
│   ├── repository/
│   ├── transport/http/
│   ├── transport/grpcclient/
│   ├── infrastructure/postgres/
│   ├── infrastructure/kafka/
│   ├── infrastructure/storage/
│   ├── infrastructure/telemetry/
│   ├── infrastructure/auth/
│   ├── media/
│   ├── tickets/
│   ├── returns/
│   ├── reviews/
│   └── worker/
├── migrations/
├── tests/
├── Dockerfile
└── README.md
```

Dependency direction:

```text
HTTP/Kafka handler -> application use case -> domain port <- infrastructure adapter
```

Use cases depend on interfaces owned by the domain/application layer, not concrete PostgreSQL, Kafka, S3, or gRPC clients.

## 6. Internal Media Module

### 6.1 Ownership model

Media is an internal Care module, not a standalone microservice. Care owns metadata and business authorization; S3-compatible object storage owns binary objects.

```text
Care PostgreSQL: file ID, owner, purpose, object key, checksum, status, retention
S3-compatible object storage: image, video, PDF binary
```

Packing video is excluded. Care obtains a short-lived packing-video URL from Fulfillment using `GetPackingVideoUrl` after checking the actor's Care permission.

### 6.2 Upload flow

```text
1. Client -> Care: request upload session with purpose, name, MIME, size, checksum
2. Care: authorize, validate policy, create PENDING_UPLOAD media record
3. Care -> client: file_id and short-lived presigned upload URL
4. Client -> S3-compatible storage: upload binary directly
5. Client -> Care: complete upload session
6. Care -> storage: HEAD/stat object and verify size/checksum/type
7. Scanner: mark CLEAN or INFECTED/FAILED
8. Care: READY files may be attached to a ticket, return, or review
```

Business APIs receive `file_id`; they never accept arbitrary evidence URLs as authoritative input.

### 6.3 Media states

```text
PENDING_UPLOAD -> UPLOADED -> SCANNING -> READY -> ATTACHED
       |             |           |
       +-------------+-----------+-> REJECTED

unattached PENDING_UPLOAD/UPLOADED/SCANNING/READY/REJECTED -> EXPIRED -> DELETED
```

`ATTACHED` means the file has an immutable business reference. A file can have multiple authorized references only when its purpose policy explicitly permits it.
The original upload deadline cannot be renewed through idempotent replay. A cleanup worker
first commits the `EXPIRED` tombstone, then removes the object and marks the record `DELETED`;
failed object deletion remains retryable without ever selecting `ATTACHED` evidence.

### 6.4 Planned endpoints

```http
POST   /api/v1/media/upload-sessions
POST   /api/v1/media/upload-sessions/{file_id}/complete
GET    /api/v1/media/{file_id}/access
DELETE /api/v1/media/{file_id}
```

The delete endpoint removes an unattached upload or requests deletion. It must not physically remove evidence under retention or legal hold.

### 6.5 Access URLs

Private objects remain private. After authorization, Care returns a signed download URL with a short TTL. Do not persist signed URLs, log their full query string, or return raw bucket/object credentials.

Cloudinary may be added later behind the storage interface. Even then, the public business contract remains `file_id`; provider asset IDs and delivery URLs remain adapter details.

## 7. Ticket domain

### 7.1 Aggregate and entities

- `SupportTicket` aggregate root;
- `TicketMessage` for customer-visible conversation;
- `TicketInternalNote` stored separately from public messages;
- `TicketAssignment` and assignment history;
- `TicketSLAMilestone` for first response and resolution;
- media references using Care `file_id`.

### 7.2 State machine

```text
NEW -> TRIAGED -> ASSIGNED -> IN_PROGRESS
                            -> WAITING_CUSTOMER
                            -> WAITING_INTERNAL
WAITING_* -> IN_PROGRESS
IN_PROGRESS/WAITING_* -> RESOLVED -> CLOSED
RESOLVED -> IN_PROGRESS (reopen within policy window)
```

Every transition has an actor, reason, timestamp, previous state, new state, and optimistic version.

### 7.3 Core invariants

- A customer sees only tickets they own.
- Internal notes are never returned through customer DTOs or notification payloads.
- Assignment and transition permissions are evaluated independently.
- Linking an order requires proof of ownership from the Order contract.
- Snapshots have `captured_at` and never impersonate current source state.
- Create/message/action requests are idempotent.
- Concurrent assignment or status updates use compare-and-swap/version checks.

## 8. Return and refund domain

### 8.1 Separate lifecycles

Do not use one status for the entire workflow.

Return case:

```text
REQUESTED -> NEEDS_INFORMATION -> REVIEWING -> APPROVED | PARTIALLY_APPROVED | REJECTED
APPROVED/PARTIALLY_APPROVED -> IN_FULFILLMENT -> COMPLETED
REJECTED/COMPLETED -> CLOSED
```

Returned goods:

```text
NOT_REQUIRED | AWAITING_CUSTOMER_SHIPMENT -> IN_TRANSIT -> RECEIVED -> INSPECTED
```

Refund obligation:

```text
NOT_REQUIRED | PENDING_APPROVAL -> APPROVED -> REQUESTED -> PROCESSING
PROCESSING -> SUCCEEDED | FAILED
FAILED -> REQUESTED (controlled retry)
```

Warehouse disposition is owned by Inventory and projected into Care:

```text
QUARANTINE | DISPOSE | RETURN_TO_VENDOR | RESTOCKABLE
```

Returned food is never made sellable automatically. Inventory must explicitly decide disposition; the default is quarantine.

### 8.2 Core invariants

- A return references immutable `order_line_id`, not only SKU.
- Requested quantity cannot exceed purchased quantity minus active/completed return quantities.
- Refund cannot exceed the authoritative refundable ceiling supplied by Order/Payment.
- Care stores the approved amount but does not calculate discount, tax, loyalty, or multi-capture allocation independently.
- Approval, returned goods, refund execution, payment, shipment, and order are distinct histories.
- Sales Manager permission is required for refund approval.
- Every financial decision has immutable audit evidence.
- Redelivery of commands/events must not create duplicate cases, inspections, stock changes, or refunds.

## 9. Review domain

Review is Phase 2.

### 9.1 Aggregate and entities

- `Review` aggregate root;
- `ReviewVersion` for customer edits;
- `ReviewMedia` for independently moderated media;
- `ReviewReply` for official response;
- `ReviewModerationDecision`;
- rebuildable `ProductRatingAggregate`.

### 9.2 Lifecycle

```text
ELIGIBLE_ORDER_LINE -> PUBLISHED -> HIDDEN_POLICY -> PUBLISHED (restored)
```

Media may remain hidden while the review text is public. A hidden review does not contribute to public rating aggregates.

### 9.3 Core invariants

- Eligibility comes from an Order-owned eligibility contract, not a client-supplied flag.
- Rating is an integer from 1 to 5.
- One review exists per customer and immutable order line.
- Edits create a version and do not reset the edit window.
- `verified` is derived and cannot be edited by customers or staff.
- Public DTOs never expose order ID, contact data, or full customer identity.
- Moderation and official replies use optimistic concurrency and audit history.

## 10. Persistence model

Planned table groups:

| Module | Tables |
| --- | --- |
| Platform | `idempotency_records`, `inbox_events`, `outbox_events`, `audit_logs` |
| Media | `media_files`, `media_references`, `media_scan_results` |
| Ticket | `support_tickets`, `ticket_messages`, `ticket_internal_notes`, `ticket_assignments`, `ticket_status_history`, `ticket_sla_milestones` |
| Return | `return_cases`, `return_case_lines`, `return_evidence`, `return_decisions`, `return_decision_lines`, `return_status_history`, `returned_goods_tracking`, `refund_obligations`, `refund_status_history` |
| Review | `reviews`, `review_versions`, `review_media`, `review_replies`, `review_moderation`, `product_rating_aggregates`, `review_eligibility_projection` |

Persistence rules:

- UUID v7-compatible IDs generated by the application;
- `TIMESTAMPTZ` in UTC;
- money stored as integer minor units plus ISO currency code, never floating point;
- external IDs stored as strings/UUIDs without cross-database foreign keys;
- database check constraints for finite states and rating range;
- unique/partial indexes for active-return and review uniqueness rules;
- `version` columns for optimistic concurrency;
- no hard deletion of audit, decisions, messages, or evidence references;
- JSONB only for bounded snapshots or provider-specific metadata, not as a substitute for the domain model.

## 11. Planned HTTP surface

Exact OpenAPI is produced during the contract phase. The target surface is:

Customer:

```http
POST /api/v1/media/upload-sessions
POST /api/v1/media/upload-sessions/{id}/complete
GET  /api/v1/media/{file_id}/access

POST /api/v1/tickets
GET  /api/v1/tickets
GET  /api/v1/tickets/{ticket_id}
POST /api/v1/tickets/{ticket_id}/messages

POST /api/v1/returns
GET  /api/v1/returns
GET  /api/v1/returns/{return_id}
POST /api/v1/returns/{return_id}/supplements

POST  /api/v1/reviews
GET   /api/v1/reviews/products/{product_id}
PATCH /api/v1/reviews/{review_id}
```

Staff/admin:

```http
GET  /api/v1/admin/tickets
GET  /api/v1/admin/tickets/{ticket_id}
POST /api/v1/admin/tickets/{ticket_id}/assign
POST /api/v1/admin/tickets/{ticket_id}/transitions
POST /api/v1/admin/tickets/{ticket_id}/messages
POST /api/v1/admin/tickets/{ticket_id}/internal-notes

GET  /api/v1/admin/returns
GET  /api/v1/admin/returns/{return_id}
POST /api/v1/admin/returns/{return_id}/request-information
POST /api/v1/admin/returns/{return_id}/decisions
POST /api/v1/admin/returns/{return_id}/packing-video-access

GET  /api/v1/admin/reviews
POST /api/v1/admin/reviews/{review_id}/moderation
POST /api/v1/admin/reviews/{review_id}/reply
```

All retryable create/action endpoints require `Idempotency-Key`. Customer identity and permissions come from trusted gateway/JWT context, never request-body role fields.

## 12. Integration contracts

### 12.1 Synchronous dependencies

Order must expose a Care-oriented query/eligibility contract containing:

- order ownership result;
- immutable order line ID;
- product/SKU snapshot and purchased quantity;
- delivered/completed/eligibility timestamp;
- active/returned quantities;
- paid amount and maximum refundable amount per line;
- currency and original payment reference;
- return/review eligibility result and machine-readable rejection reasons.

Fulfillment provides `GetPackingVideoUrl` with short TTL. Authorization identity should be passed in trusted metadata; Care must not assert a role supplied by the browser.

Profile may be queried only for the minimum customer display data permitted for the use case. Care should prefer identity claims or a snapshot supplied by the owning service when possible.

### 12.2 Asynchronous dependencies

Care publishes planned CloudEvents to `care.events.v1`, including:

- ticket created/status changed/assigned and metadata-only SLA near-breach/breached alerts;
- return requested/approved/rejected;
- returned-goods inspection requested when the integration contract assigns that command to Care;
- refund obligation requested;
- review submitted/updated/moderated.

Care consumes, subject to ratified schemas:

- order delivery/completion and eligibility-relevant events;
- shipment failure/delivery events;
- returned-goods inspection results from Inventory/Fulfillment;
- refund processed/failed results from the refund-executing service.

Every producer write uses a transactional outbox. Every consumer uses an inbox/processed-event record and idempotent state transition. Poison messages use bounded retries and DLQ; replay procedures belong in the runbook.

The existing `ReturnTicketApprovedEvent` lacks line-level approved quantities and refund ceiling/amount details. It must be versioned before implementation relies on it.

## 13. Security and privacy

- Authorize by permission and resource scope, not role name alone.
- Verify customer ownership for every ticket, return, review, and media reference.
- Use private object buckets and short-lived signed URLs.
- Enforce upload purpose, content type, actual detected type, size, checksum, scan result, and ownership.
- Do not log tokens, signed URL query strings, bank account details, full addresses, or evidence contents.
- Separate public messages from internal notes in storage, repository methods, DTOs, and tests.
- Record access to packing video and sensitive evidence.
- Encrypt sensitive provider/payment references where required by the platform policy.
- Retention and legal hold override user-requested deletion.

## 14. Reliability and observability

Required endpoints:

```http
GET /health/live
GET /health/ready
GET /metrics
```

Required telemetry fields:

- request ID, trace ID, correlation ID;
- actor ID where permitted;
- aggregate type and ID;
- event ID/type;
- operation, duration, result, and stable error code.

Required business metrics include:

- ticket backlog and SLA breaches;
- return decision latency;
- refund pending/failed counts;
- media upload/scan failures;
- outbox backlog and Kafka consumer lag;
- review moderation backlog when Phase 2 is enabled.

The SLA alert worker polls only incomplete, unpaused milestones on non-terminal tickets.
It atomically records `near_breach_alerted_at` or `breach_alerted_at` with a metadata-only
outbox event while holding a row lock. This makes repeated polls and concurrent service
instances duplicate-safe; Notification remains responsible for delivering staff alerts.

## 15. Conformance rule

Any implementation that changes a boundary, lifecycle, ownership rule, public contract, event responsibility, persistence technology, or security policy must update the decision log first. Code and contracts must reference the relevant decision/work-item IDs in pull-request descriptions and tests.

