# Order–Care Integration: Contract and Runtime Flows

> **Status:** Cross-service integration proposal for Order and Care review. This document describes the intended ownership and end-to-end behavior; it is not, by itself, a ratified API or event contract.
>
> **Purpose:** Give the Order and Care teams one shared description of the contracts Order needs to expose and how the completed Order–Care flows should behave.

## 1. Service ownership

| Concern | Owner | Responsibility |
| --- | --- | --- |
| Order identity, ownership, order lines, lifecycle and payment allocation | Order | Return authoritative order facts, validate commands, and own order state transitions. |
| Support ticket, return case, evidence references and staff decision | Care | Own case lifecycle, gather evidence, apply support policy, and record the immutable decision. |
| Refund obligation and order/payment projection | Order / Payment | Validate an approved Care decision, enforce the refundable limit, coordinate refund processing, and publish/query its canonical outcome. |
| Refund execution/accounting | Payment/Finance contract owner | Execute or account for the external financial effect according to the contract ratified by those owners. Care must not execute a refund. |
| Shipment and tracking | Shipping | Own outbound or reverse-shipment state and carrier checkpoints. |
| Goods inspection and stock disposition | Inventory | Own inspection result, quarantine and any stock disposition. Care must not update inventory directly. |

Order is authoritative for the order and its payment facts. Care is authoritative for whether a support/return case is approved. Approval in Care does not itself cancel an order, prove a shipment stopped, change inventory, or prove that money was refunded.

## 2. Contracts Order needs to provide

### 2.1 Purpose-bound order context for Care

Provide a service-to-service read capability that Care can use to verify and review an order. It should return only the fields needed for the requested purpose, not the raw commercial snapshot or a general-purpose order export.

At minimum, the contract should define:

- authenticated Care service identity and the permission/audience required;
- how Order verifies that the authenticated customer is the owner of the order;
- immutable `order_id`, `order_line_id`, snapshot ID/schema version, and source timestamp;
- order/payment facts needed to evaluate the Care request, including purchased line quantities and authoritative return eligibility/time where Order owns that policy;
- allocated receipt/payment reference, currency, and the authoritative refundable ceiling or a capability to obtain it;
- the privacy/redaction rules for recipient, contact, product and payment data;
- behavior for unavailable, ineligible, unowned, stale, or unknown-version data.

Care owns its return cases, so Order should not be assumed to know every Care case or calculate the number of units already reserved by open Care cases. Care must enforce its own case/quantity rules. At refund acceptance, Order/Payment must atomically enforce the remaining financial limit so concurrent or replayed decisions cannot over-refund.

### 2.2 Cancellation review handoff and decision application

Order should provide a durable way to record a cancellation review request and deliver it to Care. Care needs a trusted way to return a decision bound to the order, request/case, scope, policy version and decision version. The contract must support idempotent retries and lookup after a timeout.

Order remains responsible for checking current eligibility and coordinating the actual cancellation with Fulfillment and any required inventory compensation. A Care approval is not proof that fulfillment stopped or stock was reversed. Cancellation approval and refund approval are separate decisions and obligations.

### 2.3 Approved refund acceptance and outcome

Order/Payment should accept a refund obligation only after verifying the authoritative Care approval. The request/decision must bind at least the case and decision IDs, order/receipt scope, amount, currency, policy/version and actor/evidence reference required by policy.

Order/Payment should:

- enforce the refundable ceiling under a lock or equivalent atomic guard;
- persist one durable obligation with a stable idempotency key;
- return a stable obligation/reference and support status lookup/recovery;
- publish or expose a canonical success/failure/unknown outcome for Care to project;
- never interpret request acceptance or approval as proof of completed bank settlement.

The teams must ratify which component invokes the external refund provider and which component publishes the final result. Care only approves and tracks; it does not move money.

### 2.4 Care completion decision barrier

Define a request/query contract so Order can obtain a current Care decision before completing an order. The decision should bind the order, decision key, expected case version and policy version, and support querying the same decision after timeout.

Care must serialize case creation with its completion decision: a case that wins the race blocks completion according to policy; a decision that wins first is handled by the agreed post-completion policy. Order then rechecks its own settlement, delivery, hold, refund-obligation and other lifecycle guards before committing `COMPLETED`. A stale or unknown Care decision must not be treated as allow.

## 3. Intended completed runtime flows

### 3.1 Return/refund

1. The customer submits a return/refund request to Care with the order reference, requested lines/quantities, reason and Care `file_id` evidence references.
2. Care calls Order's purpose-bound context capability. Order verifies ownership and returns the permitted, minimal order/payment facts. Care rejects or fails closed if ownership or eligibility cannot be established.
3. Care creates the case and its line-level request in its own database. Care enforces uniqueness/conflict rules against its own active cases and stores a timestamped source snapshot for audit.
4. A Care agent reviews the case and records an immutable approve, partial-approve, reject or needs-information decision. The decision contains the permitted lines, quantities, amounts, currency, reason and policy version.
5. If goods must be returned, Care tracks the reverse shipment through Shipping and the inspection result through Inventory. Refund is not started until the required inspection result is valid. For an approved return waiver, the agreed policy may allow refund processing without returned goods.
6. Care submits the approved refund decision to Order/Payment. Order/Payment verifies the approval, atomically enforces the refundable ceiling and creates/reuses the durable refund obligation.
7. Payment/Finance processing produces the canonical outcome. Order/Payment exposes or publishes it; Care consumes/queries it idempotently and updates its refund projection. The Care case is not marked refunded merely because the decision was approved.

### 3.2 Cancellation requiring Care review

1. The customer submits a cancellation request through the Order-owned flow. Order persists a nonterminal review request and durably sends it to Care.
2. Care creates or links a case, obtains the required Order context, reviews evidence/policy, and records an immutable decision.
3. Care returns the trusted decision to Order. If rejected, Order records the outcome and leaves the order lifecycle unchanged. If approved, Order revalidates current state and coordinates the Fulfillment stop/cancel barrier and required compensation.
4. Order changes its lifecycle only after its required remote proofs are confirmed. Any refund is handled as a separate refund obligation; cancellation approval does not imply refund success.

Ordinary self-service cancellations that Order policy allows remain in Order and do not need Care review.

### 3.3 Order completion

1. Shipping reports delivery through the agreed authenticated event/API contract. Order records the shipment checkpoint; `DELIVERED` alone does not prove COD settlement or satisfy every completion condition.
2. When the completion window/policy permits, Order requests the current Care completion decision using an idempotent decision key and expected case version.
3. Care serializes that decision with case creation and replies allow/deny with a decision ID/version. An active case, unresolved refund obligation or other policy condition can deny or defer completion.
4. Order verifies the Care decision and rechecks its own delivery, settlement, hold and financial guards under its lifecycle control. Only Order commits the transition to `COMPLETED`.

## 4. Integration and reliability requirements

- Use trusted service identity and least-privilege permissions; never accept a caller-supplied customer ID as ownership proof without Order verification.
- All retryable commands require idempotency. Decisions and outcomes require stable IDs and query/recovery behavior after lost acknowledgements.
- Events use versioned schemas, stable event IDs, correlation IDs, and enough order/case/decision/resource references for deduplication; do not include unnecessary PII or raw evidence URLs.
- Use transactional outbox/inbox or an equivalent durable handoff for asynchronous work. Define retry, timeout, poison-message and DLQ handling.
- Distinguish business facts: shipment created is not shipment dispatched; delivered is not COD settled; Care approved is not refund completed; returned goods received is not inventory sellable.
- Contract tests must cover ownership denial without data leakage, stale/ineligible orders, partial quantities/amounts, duplicate commands/events, concurrent refund limits, timeout recovery, case-open/completion races, and Fulfillment cancellation races.

## 5. Contract decisions to ratify before production integration

1. **Transport:** Care's decision log calls for an Order Protobuf/gRPC context contract, while the Order Core candidate describes internal HTTP/OpenAPI purpose-bound views and commands. Select one authoritative transport/version and align both services; do not integrate against two incompatible drafts.
2. **Order context:** confirm the exact request identity, task/reference association, response fields, masking rules, return eligibility source and versioning.
3. **Cancellation:** confirm the durable Order-to-Care handoff, Care decision verification and exact Fulfillment proof required before Order changes state.
4. **Refund:** confirm the command owner, provider executor, canonical result publisher, states, and how Order/Payment reserves and releases refundable budget.
5. **Completion:** confirm who requests the Care decision, decision-key/version semantics, timeout query, and post-completion case policy.
6. **Events:** ratify names, topic, schemas, compatibility rules, DLQ/replay procedure, and ownership for return approval, refund outcome, case-open/close and inspection result.

Until these decisions are ratified, candidate contracts are design inputs rather than proof of a deployed integration. Care and Order should preserve fail-closed behavior for any capability not yet available.

## 6. Source documents

- Care technical baseline: `docs/02_architecture/services/ms06_care_service_design.md`
- Care decision log: `docs/02_architecture/services/ms06_care_service_decision_log.md`
- Care delivery plan: `docs/02_architecture/services/ms06_care_service_delivery_plan.md`
- Care API: `docs/03_api_specs/care-service.openapi.yaml`
- Order Core design: `docs/02_architecture/services/order-service/12_order_core_design.md`
- Order Core candidate API: `docs/03_api_specs/order-core.candidate.openapi.yaml`
- Order Payment staff candidate API: `docs/03_api_specs/order-payment-staff.candidate.openapi.yaml`
