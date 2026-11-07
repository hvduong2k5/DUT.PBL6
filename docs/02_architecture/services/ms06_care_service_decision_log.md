# MS-06 Care Service - Decision Log and Policy Defaults

> Status: **Implementation decision register**  
> Updated: 2026-10-10  
> Related design: [MS-06 Care Service Technical Design](./ms06_care_service_design.md)

## 1. Status meanings

| Status | Meaning |
| --- | --- |
| `ACCEPTED` | May be implemented by Care without another design decision. |
| `PROPOSED` | Recommended direction, but another service/team or product owner must ratify it before dependent implementation is released. |
| `DEFERRED` | Intentionally excluded from the current delivery scope. |
| `SUPERSEDED` | Replaced by a newer decision; retained for history. |

Policy values in this document are initial defaults, not magic constants. They must be represented by named configuration or policy records when the value can change without altering a domain invariant.

## 2. Architecture decisions

| ID | Status | Decision | Rationale and consequence |
| --- | --- | --- | --- |
| `CARE-ADR-001` | `ACCEPTED` | Use Go and Clean/Hexagonal architecture. | Matches the service ecosystem and keeps domain/use cases independent from HTTP, PostgreSQL, Kafka, and S3. |
| `CARE-ADR-002` | `ACCEPTED` | Use PostgreSQL 16 instead of MongoDB. | Transactions, constraints, money, state machines, audit, idempotency, inbox/outbox, and optimistic concurrency are primary requirements. Older MongoDB diagrams are superseded for MS-06. |
| `CARE-ADR-003` | `ACCEPTED` | Care owns a private `care_db`; no cross-service database access. | Preserves bounded contexts. External data is obtained by contract and stored only as IDs/minimum snapshots. |
| `CARE-ADR-004` | `ACCEPTED` | REST is the north-south API. Care initially acts as a gRPC client only. | No confirmed synchronous consumer currently requires a Care gRPC server. |
| `CARE-ADR-005` | `ACCEPTED` | Use Kafka with CloudEvents, transactional outbox, inbox deduplication, bounded retry, and DLQ. | Prevents dual-write loss and duplicate side effects. |
| `CARE-ADR-006` | `ACCEPTED` | Do not introduce Redis in MVP. | PostgreSQL is authoritative; caching is added only after measurement or a coordination requirement. |
| `CARE-ADR-007` | `ACCEPTED` | Do not introduce Temporal in MVP. | Explicit state machines plus inbox/outbox are sufficient for the first workflow. Re-evaluate only when retry/timer/compensation complexity proves material. |
| `CARE-ADR-008` | `ACCEPTED` | IDs are application-generated UUID v7-compatible values; timestamps are UTC. | Time-ordered IDs work well with PostgreSQL indexes and do not expose sequence volume. |
| `CARE-ADR-009` | `ACCEPTED` | Money is integer minor units plus ISO currency. | Floating-point money is forbidden. Care accepts authoritative refundable ceilings rather than reconstructing financial allocation. |
| `CARE-ADR-010` | `ACCEPTED` | Authorization is permission-and-scope based. | A role name alone does not prove access to a specific ticket, order, case, file, or packing video. |

## 3. Media decisions

| ID | Status | Decision | Rationale and consequence |
| --- | --- | --- | --- |
| `CARE-MEDIA-001` | `ACCEPTED` | Media is an internal Care module, not a separate microservice. | Avoids network consistency and operational overhead while only Care is in scope. |
| `CARE-MEDIA-002` | `SUPERSEDED` | Use MinIO locally and S3-compatible private object storage in production. | Superseded by `CARE-MEDIA-010` after official MinIO Community container images became unavailable. |
| `CARE-MEDIA-003` | `ACCEPTED` | Frontend uploads binary directly using a short-lived presigned session. | Care avoids proxying large multipart bodies and duplicate bandwidth. |
| `CARE-MEDIA-004` | `ACCEPTED` | Business APIs accept immutable `file_id`, not binary bodies or arbitrary URLs. | Enables ownership, purpose, scan, retention, and provider-independent checks. |
| `CARE-MEDIA-005` | `ACCEPTED` | Private access URLs are generated after authorization and expire after 10 minutes by default. | URLs are capabilities and must not become long-lived identifiers. Config: `MEDIA_ACCESS_URL_TTL=10m`. |
| `CARE-MEDIA-006` | `ACCEPTED` | Upload sessions expire after 15 minutes; unattached uploads are cleanup candidates after 24 hours. | Limits leaked upload capability and orphan cost. Config: `MEDIA_UPLOAD_URL_TTL=15m`, `MEDIA_ORPHAN_TTL=24h`. |
| `CARE-MEDIA-007` | `ACCEPTED` | A file must be verified and have a clean scan result before attachment. | Client-declared MIME/size/checksum are not authoritative. In non-production, a clearly labeled no-op scanner may be used; production must fail closed. |
| `CARE-MEDIA-008` | `ACCEPTED` | Packing video is not a Care media record. | Fulfillment owns the video and issues its own short-lived access URL. Care stores only access audit/case reference, never the video URL. |
| `CARE-MEDIA-009` | `PROPOSED` | Retention is policy-driven; physical deletion is disabled until legal/product retention is ratified. | Prevents accidental evidence destruction. Cleanup may remove never-attached orphan objects only. |
| `CARE-MEDIA-010` | `ACCEPTED` | Use pinned SeaweedFS in S3-compatible mode for local development and private S3-compatible object storage in production. | Preserves the S3 contract and provider portability without relying on unavailable MinIO Community images. Internal storage and client-facing signing endpoints are configured separately. |
| `CARE-MEDIA-011` | `ACCEPTED` | The original upload session deadline is immutable and cannot be extended by idempotent replay. Never-attached records become `EXPIRED` at orphan TTL, then their object is removed before the record becomes `DELETED`. | Prevents upload capability renewal and makes cleanup retry-safe. Attached evidence is excluded; failed object deletion leaves an `EXPIRED` tombstone for the next worker poll. |

Initial upload limits:

| Purpose | Allowed files | Count per operation | Limit per file | Notes |
| --- | --- | ---: | ---: | --- |
| `RETURN_EVIDENCE` | JPEG, PNG, WebP; MP4 | 5 images and 1 video | image 10 MiB; video 100 MiB | Total operation limit 150 MiB. |
| `TICKET_ATTACHMENT` | JPEG, PNG, WebP, PDF; MP4 | 5 files | document/image 10 MiB; video 100 MiB | Executables and archives are rejected. |
| `REVIEW_MEDIA` | JPEG, PNG, WebP; MP4 | 5 images and 1 video | image 10 MiB; video 100 MiB | Phase 2; media can be moderated independently. |

These are policy defaults: `MEDIA_MAX_*` configuration. Storage-side policy and post-upload verification must both enforce them.

## 4. Ticket policy decisions

| ID | Status | Decision | Rationale and consequence |
| --- | --- | --- | --- |
| `CARE-TICKET-001` | `ACCEPTED` | MVP accepts registered customers only. | Guest tickets are unsafe until OTP/proof-session ownership is defined. Guest support remains a requirement gap, not silently implemented. |
| `CARE-TICKET-002` | `ACCEPTED` | Use states `NEW`, `TRIAGED`, `ASSIGNED`, `IN_PROGRESS`, `WAITING_CUSTOMER`, `WAITING_INTERNAL`, `RESOLVED`, `CLOSED`. | Separates intake, ownership, work, waiting, resolution, and final closure. |
| `CARE-TICKET-003` | `ACCEPTED` | Public messages and internal notes are different entities, tables, repository APIs, and DTOs. | A discriminator column alone creates an unacceptable leakage risk. |
| `CARE-TICKET-004` | `ACCEPTED` | First-response target is 4 elapsed hours; resolution target is 48 elapsed hours for normal priority. | Simple MVP clock; configuration may vary by priority/category. No claim of business-hours support until a calendar is implemented. |
| `CARE-TICKET-005` | `ACCEPTED` | `WAITING_CUSTOMER` pauses resolution SLA; `WAITING_INTERNAL` does not. First-response SLA never resets. | Internal handoff must not hide delays. All pause/resume timestamps are audited. |
| `CARE-TICKET-006` | `ACCEPTED` | A resolved ticket may reopen for 7 days; a closed ticket creates a linked follow-up instead. | Keeps final closure stable while preserving continuity. Config: `TICKET_REOPEN_WINDOW=168h`. |
| `CARE-TICKET-007` | `ACCEPTED` | Assignment and transitions use optimistic concurrency. | Competing agents receive a conflict rather than silently overwriting ownership/state. |
| `CARE-TICKET-008` | `PROPOSED` | Initial categories are `ORDER`, `PAYMENT`, `SHIPPING`, `PRODUCT_QUALITY`, `RETURN_REFUND`, `ACCOUNT`, `OTHER`; priorities are `LOW`, `NORMAL`, `HIGH`, `URGENT`. | Product/operations must ratify queue ownership and escalation mapping before Agent Queue release. |
| `CARE-TICKET-009` | `DEFERRED` | Multi-channel identity merge and outbound channel delivery. | Phase 2; MVP supports in-app/web API conversation and delegates notifications. |
| `CARE-TICKET-010` | `DEFERRED` | AI triage, summaries, drafts, and knowledge assistant. | No AI path may bypass authorization or human approval; implement after Phase 2 governance. |
| `CARE-TICKET-011` | `ACCEPTED` | Staff queue and detail reads require `care.tickets.read`; reading the complete staff detail is audited as `TICKET_CONTEXT_VIEWED`. | Keeps read access independent from future mutation permissions and records access to internal notes and sensitive ticket context. Gateway-provided permissions are trusted only after the internal token is validated. |
| `CARE-TICKET-012` | `ACCEPTED` | Self-claim uses `care.tickets.claim`, a required reason, idempotency, and `expected_version`; server identity is always the assignee. MVP self-claim may move either `NEW` or `TRIAGED` directly to `ASSIGNED`. | Prevents a caller from supplying an arbitrary staff identity and makes concurrent claims deterministic. The direct `NEW` claim is safe because category, default priority, and SLA are established at creation. Assign/reassign to another staff member or queue remains gated on a ratified staff-directory and queue-taxonomy contract. |
| `CARE-TICKET-013` | `ACCEPTED` | Creating an internal note requires `care.tickets.notes.write`; the note is persisted separately, audited by ID/author only, and its content is not emitted through outbox/notification events. | Enforces the public/internal data boundary and minimizes leakage through logs, audit payloads, Kafka, and customer DTOs. |
| `CARE-TICKET-014` | `ACCEPTED` | State changes require `care.tickets.transition`, `expected_version`, idempotency, and a reason. Except for unassigned `NEW -> TRIAGED`, the actor must be the assigned staff member. | Prevents a general permission from bypassing dynamic ticket ownership. Transitions update history, audit, outbox, lifecycle timestamps, and resolution-SLA pause/completion in one transaction; Manager override remains a future distinct permission. |
| `CARE-TICKET-015` | `ACCEPTED` | A staff public reply requires `care.tickets.reply`, assignment to the actor, and idempotency. Care stores an in-app `STORED` message, completes first-response SLA, and emits metadata only. | Customer ticket detail can show the reply immediately without leaking message content through audit/Kafka. Email/push delivery and later delivery-status changes remain blocked on the Notification Service contract. |
| `CARE-TICKET-016` | `ACCEPTED` | Care polls active, unpaused SLA milestones and emits one near-breach event within the configurable `SLA_NEAR_BREACH_WINDOW` and one breached event after the deadline. Alert timestamps and outbox rows are committed atomically with row locks. | Multiple service instances cannot emit duplicate logical alerts. Events contain routing metadata only; Notification owns eventual staff delivery. Completed, paused, resolved, and closed milestones are not alerted. |

## 5. Return and refund policy decisions

| ID | Status | Decision | Rationale and consequence |
| --- | --- | --- | --- |
| `CARE-RETURN-001` | `ACCEPTED` | Simple self-service order cancellation remains in Order Service. Care handles return/refund cases and later escalated cancellation cases only. | Avoids duplicate ownership of the order lifecycle. |
| `CARE-RETURN-002` | `ACCEPTED` | Default return-request window is 48 hours from authoritative `return_eligible_at` supplied by Order. | Matches the current D2C contract and perishable-product context. Config: `RETURN_REQUEST_WINDOW=48h`. Product may change this policy without changing the data model. |
| `CARE-RETURN-003` | `ACCEPTED` | MVP supports return-and-refund and refund-without-return; exchange/resend is deferred. | Avoids replacement-order, inventory reservation, shipping, and price-difference saga in the first release. Existing `RESEND_NEW_BOX` event values must not be used by MVP. |
| `CARE-RETURN-004` | `ACCEPTED` | Refund-without-return requires Manager permission, reason, evidence, and explicit `return_required=false`. | Makes exceptions visible and auditable. No automatic value threshold is enabled by default. |
| `CARE-RETURN-005` | `PROPOSED` | Seller/fulfillment/shipping fault uses merchant-paid return shipping; customer-preference return uses customer-paid shipping when the policy permits it. | Shipping team/product must ratify label and reverse-shipment contracts. MVP may provide manual instructions until that integration exists. |
| `CARE-RETURN-006` | `ACCEPTED` | Returned food defaults to quarantine and is never automatically restocked. | Inventory owns inspection/disposition; food safety requires an explicit decision before any stock adjustment. |
| `CARE-RETURN-007` | `ACCEPTED` | If return is required, refund request begins only after a valid inspection result. If return is waived, it may begin after approval. | Keeps goods and money lifecycles independent while allowing evidence-only exceptions. |
| `CARE-RETURN-008` | `ACCEPTED` | Order/Payment supplies the authoritative refundable ceiling and allocation. | Care must not recreate discount, tax, coupon, loyalty, shipping, or multi-capture calculations. Approved amount cannot exceed that ceiling. |
| `CARE-RETURN-009` | `ACCEPTED` | Refund defaults to the original payment method. Loyalty refund is deferred until consent, conversion, and anti-double-refund contracts exist. | Reduces financial ambiguity and respects `BR-REFUND-03`. |
| `CARE-RETURN-010` | `ACCEPTED` | One actor with `care.return.approve`/Manager scope approves in MVP; dual approval is disabled. | Meets current business rule while preserving an extension point. Any later high-value threshold must be a policy plus a separate approval record. |
| `CARE-RETURN-011` | `ACCEPTED` | Rejected cases may be appealed/reopened within 7 days with new evidence; prior decisions remain immutable. Completed/refunded cases require a new linked appeal case. | Preserves audit history and prevents state rewrites. Config: `RETURN_APPEAL_WINDOW=168h`. |
| `CARE-RETURN-012` | `PROPOSED` | Care approves and tracks; Order orchestrates the cross-service refund flow; the existing Finance contract reports final refund outcome; Inventory owns inspection/disposition. | This best matches current HLD and schemas, but Order, Payment/Finance, and Inventory teams must ratify publisher/command ownership before implementation. Care never executes the refund. |
| `CARE-RETURN-013` | `PROPOSED` | Version the approved-return event before use. Include case ID, decision ID, order lines, quantities, approved amounts/currency, return-required flag, reason codes, actor, and timestamps. | The existing event is too coarse for partial return/refund and idempotent downstream work. |

## 6. Review policy decisions

| ID | Status | Decision | Rationale and consequence |
| --- | --- | --- | --- |
| `CARE-REVIEW-001` | `ACCEPTED` | Review is Phase 2. | Ticket and return workflows have higher MVP priority. |
| `CARE-REVIEW-002` | `PROPOSED` | Order exposes `review_eligible_at` and eligibility reasons; Care does not couple to raw `DELIVERED` versus `COMPLETED` strings. | Resolves contradictory documents and centralizes order truth. Order team must ratify the query/event contract. |
| `CARE-REVIEW-003` | `ACCEPTED` | One review per customer and immutable `order_line_id`. A later purchase creates another eligible order line. | Provides a clear unique constraint and preserves verified-purchase evidence. |
| `CARE-REVIEW-004` | `ACCEPTED` | Review creation window is 30 days from `review_eligible_at`; edit window is 30 days from first publication. | Both values are configurable and edits do not reset the clock. |
| `CARE-REVIEW-005` | `ACCEPTED` | Rating is required; text is optional with maximum 2,000 Unicode characters. | Allows concise star ratings without forcing low-quality filler text. Content validation still applies. |
| `CARE-REVIEW-006` | `ACCEPTED` | Valid reviews publish after synchronous validation and clean-media checks; staff can hide/restore with reason. | Avoids a universal pre-moderation bottleneck. Flagged content may be held; no AI moderation in initial Phase 2. |
| `CARE-REVIEW-007` | `ACCEPTED` | Rating 1-2 is negative. It enters a Care queue but does not automatically create a ticket. | Avoids unwanted duplicate tickets; an authorized agent may create a linked ticket idempotently. |
| `CARE-REVIEW-008` | `ACCEPTED` | One current official reply per review; edits create versions and are audited. | Keeps the public conversation clear and traceable. |
| `CARE-REVIEW-009` | `ACCEPTED` | Public aggregate uses only `PUBLISHED` reviews; store sum and count, display average rounded to one decimal. | Hidden reviews must be removed from the public aggregate, and aggregates must be rebuildable. |
| `CARE-REVIEW-010` | `PROPOSED` | Public customer identity is a masked display name; avatar is shown only if the Profile privacy contract permits it. | Product/privacy must ratify exact masking and retention. Never expose order/contact data. |

## 7. Cross-service ownership proposal

This matrix is the recommended target and is not fully ratified until dependent teams accept their rows.

| Capability | Owner | Care responsibility |
| --- | --- | --- |
| Order ownership and line snapshot | Order | Query and store minimum timestamped snapshot. |
| Return/review eligibility | Order | Enforce returned result and reason; do not infer from browser data. |
| Return case and decision | Care | Authoritative owner. |
| Customer evidence metadata | Care | Authoritative owner; binary remains in object storage. |
| Packing video | Fulfillment | Request short-lived URL and audit access. |
| Reverse shipment | Shipping | Track returned projection/reference. |
| Inspection and disposition | Inventory | Request/consume result; never update stock directly. |
| Refund orchestration | Order (proposed) | Publish approved obligation and track correlation. |
| Refund execution | Finance/Payment contract owner (must be ratified) | Consume success/failure result only. |
| Financial ledger | Finance | Store only Care-side projection/reference. |
| Notification delivery | Notification | Publish notification intent/event; do not block transaction on delivery. |
| Identity and permissions | Identity/API Gateway | Enforce trusted claims and resource scope. |

## 8. Decisions still requiring ratification

Release work must not silently guess these contracts:

1. Exact owner and schema for refund command, execution, and final result (`CARE-RETURN-012`).
2. Reverse-shipment API/events and shipping-fee policy (`CARE-RETURN-005`).
3. Order Care-context/eligibility Protobuf contract (`CARE-REVIEW-002`, `CARE-RETURN-008`).
4. Versioned return-approved and inspection event schemas (`CARE-RETURN-013`).
5. Production evidence retention/legal-hold duration (`CARE-MEDIA-009`).
6. Agent queue category/priority/escalation mapping (`CARE-TICKET-008`).
7. Public review identity masking (`CARE-REVIEW-010`).

Until ratified, implement behind ports or feature flags and do not expose a misleading production-complete workflow.

## 9. Change procedure

For a changed decision:

1. Add a new row or mark the old row `SUPERSEDED`; do not erase history.
2. Record affected APIs, events, migrations, tests, and operational policy.
3. Update the technical design and delivery traceability.
4. Version external contracts when compatibility changes.
5. Add a migration/backfill/rollback plan before implementation.

