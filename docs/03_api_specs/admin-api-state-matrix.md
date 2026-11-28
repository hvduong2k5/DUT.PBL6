# Internal Employee API State Matrix

Transport amendment2026-11-27: target0.3.0 uses gRPC for internal business reads/resolver/current grants; Web/BFF REST and UI states below remain unchanged. See [gRPC guide](traceability-workflow-grpc.md) and [report09](../04_testing/traceability-service/09_grpc_transport_2026-11-27.md). Do not use the superseded internal REST proposals from0.2.2.

## Traceability workflow target — 2026-11-21 (draft; backend implemented, UI integration pending)

Backend checkpoint 2026-11-26: scoped v2, context/lookup and workflow are implemented behind explicit flags, with local PostgreSQL and signed-HTTP verification. See [implementation and rollout evidence](../04_testing/traceability-service/08_scoped_backend_2026-11-26.md). Cookie BFF, real IAM/Inventory/Catalog integration and print UI acceptance remain pending; this does not promote stable or mark UI states verified.

Review target0.2.2 (2026-11-25): [BFF adapters](traceability-workflow-bff-draft.openapi.yaml) add explicit cookie transports; cookie mutations require validated Origin, and v2 requires ACTIVE EMPLOYEE. [Batch context](traceability-workflow-draft.openapi.yaml) resolves warehouse-authorized Product/SKU before draft creation; it does not depend on an unimplemented Inventory SKU detail read. Scoped permissions keep exact grant scopeId instead of role-derived ALL. Missing provider context returns an error in the traceability panel without undoing a successful receipt.

Updated 2026-11-25: scoped staff operations use [full admin v2 target](traceability-scoped-admin-draft.openapi.yaml) and [service spec](traceability-workflow-service-spec.md). Legacy admin v1 retains its GLOBAL-only capability and wire errors; scoped staff must not fallback to v1. Mutation header v2 is Idempotency-Key; errors are code/message/requestId. New evidence-reference omission on edit preserves stored references by ID before validation.

See [design](traceability-workflow-design-2026-11-21.md) and [draft OpenAPI](traceability-workflow-draft.openapi.yaml). This supplement does not promote generated stable rows or GLOBAL-only baseline to WAREHOUSE support.

| Flow | Success / empty | Loading | Error | Auth / scope |
| --- | --- | --- | --- | --- |
| Receipt → profile lookup | Receipt stays APPLIED; lookup 200 items/total=1 opens existing profile, 0 allows create | Disable duplicate lookup/create while pending | Lookup 503 retries independently; never repeat receipt to recover profile | 401 reauthenticate; 403 keep receipt success and hand off to authorized staff; warehouse verified server-side |
| Create/edit/submit | Existing workflow; exact binding, stable idempotency key | Pending command disables repeat submission | 409 binding race reads existing profile; version/source conflict requires reread; dependency failure preserves receipt | Separate permissions; GLOBAL or matching WAREHOUSE; scoped replay and list never disclose another warehouse |
| Download / print | Current eligible public profile and scoped read_internal; same qr_url on reprint | No export until latest eligibility check completes | Never published/410 blocks new valid-label export; 503 blocks export and allows retry | 401/403 blocks staff export; public QR itself remains anonymous |


Updated: 2026-11-21

This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.

## State semantics

- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.
- **Empty:** successful `200` response with an empty collection and zero totals, never `404`.
- **Unauthorized:** `401` when credentials/session/grant are missing, invalid or expired.
- **Forbidden:** `403` when identity is known but capability, permission, ownership or scope is insufficient.
- **Error:** `4xx` is actionable client/business failure; `5xx` is server/dependency failure. All errors use the shared `Error` schema.
- **Success:** only documented `2xx` codes are compatible. Mutations requiring asynchronous processing use `202`.

Employee APIs require authentication, effective permission and server-side data-scope enforcement. A hidden menu is not authorization.

The Order principal-kind gate retains Employee access to direct admin detail and cancellation-request routes; permission and warehouse checks remain mandatory. Guest, payment-grant and Customer principals cannot access the admin detail projection.

Analytics MVP scoped mode: Dashboard and metric series require `ANALYTICS_DASHBOARD_VIEW#GLOBAL`; all Supply reads require `ANALYTICS_SUPPLY_VIEW#GLOBAL`. Bearer JWT takes the grant from `permission_grants`; Identity session takes structured `permissionGrants`. Flat `permissions`, role `ADMIN`, and WAREHOUSE/DEPARTMENT grants do not authorize these routes. Missing token/session → 401; authenticated actor without matching grant or ACTIVE EMPLOYEE state → 403. Pilot UUID allowlist mode remains dev-only.

## Provisional JWT consumer rollout (parts 3–4)

Direct Profile contract is version 2.0.0: staff JWT v1 with GLOBAL HR permission;
department is a filter. Legacy employee tokens cannot use `/users/me` grants as
a fallback. Order retains each permission's GLOBAL/WAREHOUSE scope. Both retain
the old signed snapshot until token/grant expiry; role ADMIN alone never bypasses.
Issuer tasks 1–2 are pending; see [handoff](../06_operations/profile-order-jwt-handoff.md).

The entries below supplement the existing UI-format rows, without rewriting
historical inferred examples. Producer responses and BFF adaptation are defined
in [consumer read OpenAPI](iam-consumer-read.openapi.yaml), referenced by draft.

| Flow | Success / empty | Loading | Auth / scope | Error |
|---|---|---|---|---|
| Order GET list/queue/detail projection | 200; empty items, next_cursor null; unknown provider state → UNKNOWN; absent SLA → unavailable | Pending request, load-more disabled while fetching | 401 invalid/missing token; 403 no ORDER_VIEW; 404 foreign detail; recipient null without same-warehouse sensitive permission | 400 malformed query/cursor; 502 invalid projection/oversized proxy body; 503 upstream unavailable; no mock fallback |
| Web `/api/admin/orders[/{id}]` | UI DTO, counts explicitly over loaded rows; cursor loads next page | Initial skeleton; load-more progress; detail loading | Opaque OMAMA_SESSION only; backend snapshot controls authority | Friendly retry state; no fabricated dates or SLA |
| Identity GET `/api/v1/admin/order-api/orders[/queue|/{id}]` | Fixed upstream projection and server bearer; no-store | Upstream bounded by 8 seconds and 1 MiB | Existing ACTIVE employee gate plus Order's JWT policy; no browser identity headers | Denial propagated; no redirects or arbitrary targets |
| IAM GET `/api/v1/admin/access/{employees,roles,permissions}[/{id}]` | Versioned envelope, real calculatedAt, known lifecycle; empty arrays/counts | UI skeleton / detail pending | Required permission checked by task 2 producer; accessReview controls review fields | Producer absent → Web 503; malformed/version mismatch → 502; marked missing resource → 404; no mock fallback |
| Web `/api/admin/access/...` | Validated UI DTO; unknown permission catalog codes preserved, not treated as grants | Retry and loading states | Cookie only; no live `/users/me` permission substitution | Mutations 405; no credential exposure |

Access Review shows **current configured access**, not an inventory of unexpired
JWT permissions. Missing task 2 lifecycle/projection fields are an unavailable
capability, never invented dates, revision, impact or status. Existing CRUD APIs
are not silently aliased to the new projections. Role mutations remain owned by
task 2. Common BFF errors include code, safe message, requestId and errors.
## Permission sync semantics

- The IAM database is the source of truth; Keycloak user attributes are a projection used to mint tokens.
- `GET .../effective-permissions` returns the *configured* grants recomputed from the database. An access token already issued is immutable and only reflects changes after the next refresh; the response includes `tokenNote` so the UI can label "configured" vs "in token".
- `POST .../sync-permissions` is the reconciliation path after a failed sync. Failures are audited as `IAM_PERMISSION_SYNC_FAILED` with the trigger and error code. A `502` means Keycloak rejected/failed the update and the call can be retried.
- Revocation paths (revoke assignment, reduce role permissions, disable role, suspend account) sync and then force re-login: Keycloak sessions are revoked and BFF sessions removed.

## Identity lifecycle

- A disabled role remains in list and detail with `isEnabled=false`. Assignment to it returns 409. Re-enabling the role allows a new assignment.
- Invitation acceptance requires the exact Keycloak issuer and subject bound to the pending principal. An expired invitation returns 410. Revoked and accepted invitations return 409.

## Stable operations

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `GET /api/v1/admin/users` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/users/{principalId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/users/{principalId}/password/reset` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/users/{principalId}/sessions/revoke` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/users/{principalId}/status` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/employee-invitations` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/admin/employee-invitations/{invitationId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `DELETE /api/v1/admin/employee-invitations/{invitationId}` | EPIC_22 | stable | 204 | N/A | —/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/public/auth/employee-invitations/{invitationId}/accept` | EPIC_22 | stable | 200 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500/502 |
| `GET /api/v1/admin/employees` | EPIC_22 | stable | 200 | 200 (empty array) | 400/— | 401 | 403 | — | — | — | 500 |
| `GET /api/v1/admin/role-assignments` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/role-assignments` | EPIC_22 | stable | 200/201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PATCH /api/v1/admin/role-assignments/{assignmentId}` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/role-assignments/{assignmentId}` | EPIC_22 | stable | 200/204 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/iam/principals/{principalId}/effective-permissions` | EPIC_22 | stable | 200 | 200 (empty array) | —/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/iam/principals/{principalId}/sync-permissions` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/502/503 |
| `GET /api/v1/admin/roles` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/roles` | EPIC_22 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/admin/roles/{roleId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/roles/{roleId}` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/roles/{roleId}` | EPIC_22 | stable | 200/204 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/roles/{roleId}/permissions` | EPIC_22 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PUT /api/v1/admin/roles/{roleId}/permissions` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/roles/{roleId}/permissions` | EPIC_22 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/roles/{roleId}/permissions/{permissionCode}` | EPIC_22 | stable | 204 | N/A | —/— | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests` | EPIC_18 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}` | EPIC_18 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/b2b/quote-requests/{requestId}/actions` | EPIC_18 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}/files` | EPIC_18 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}/versions` | EPIC_18 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/catalog/categories` | EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500 |
| `POST /api/v1/admin/catalog/categories` | EPIC_04 | stable | 201 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500 |
| `GET /api/v1/admin/catalog/categories/{categoryId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500 |
| `PUT /api/v1/admin/catalog/categories/{categoryId}` | EPIC_04 | stable | 200 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500 |
| `PATCH /api/v1/admin/catalog/categories/{categoryId}` | EPIC_04 | stable | 200 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500 |
| `DELETE /api/v1/admin/catalog/categories/{categoryId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | 409 | — | 500 |
| `GET /api/v1/admin/catalog/products` | EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/catalog/products` | EPIC_04 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PUT /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable (legacy) | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/transition` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/skus` | EPIC_04 | stable | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}/skus/{skuId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}/skus/{skuId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 200 | 200 (empty list) | 400/— | 401 | 403 | 404 | — | — | 500/503 |
| `PUT /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices/{priceId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/images` | EPIC_04 | stable | 201 | N/A | 400/422 | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}/images/{imageId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/images/{imageId}/review` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/catalog/products/{productId}/images/{imageId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/catalog/stock-projections` | EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/inventory/skus` | EPIC_09 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/inventory/skus/{skuId}` | EPIC_09 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/inventory/workbench` | EPIC_09 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/admin/inventory/workbench/adjustments` | EPIC_09 | stable | 202 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/workbench/issues` | EPIC_09 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/workbench/receipts` | EPIC_09 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/admin/orders` | EPIC_08 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/orders/{orderId}` | EPIC_08 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/packing/tasks` | EPIC_10 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/packing/tasks/{taskId}` | EPIC_10 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/packing/workbench/tasks` | EPIC_10 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/packing/workbench/tasks/{taskId}` | EPIC_10 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `PATCH /api/v1/admin/packing/workbench/tasks/{taskId}/checklist/{itemId}` | EPIC_10 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/packing/workbench/tasks/{taskId}/complete` | EPIC_10 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/payments/cases` | EPIC_07, EPIC_24 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/payments/cases/{caseId}` | EPIC_07, EPIC_24 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/returns/cases` | EPIC_14 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/returns/cases/{caseId}` | EPIC_14 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/session` | EPIC_22, EPIC_23 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/shipping/shipments` | EPIC_11 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/shipping/shipments/{shipmentId}` | EPIC_11 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/shipping/workbench/shipments` | EPIC_11 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/shipping/workbench/shipments/{shipmentId}` | EPIC_11 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/support/tickets` | EPIC_16 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/support/tickets/{ticketId}` | EPIC_16 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |

## Draft and planned operations

These rows are proposals. Backend may refine them, but must record the decision and update the OpenAPI contract before frontend integration.

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `POST /api/v1/admin/orders/{orderId}/actions` | EPIC_08 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/packing/tasks/{taskId}/assignments` | EPIC_10 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/shipping/shipments/{shipmentId}/actions` | EPIC_11 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/payments/cases/{caseId}/actions` | EPIC_07, EPIC_24 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/returns/cases/{caseId}/actions` | EPIC_14 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/actions` | EPIC_16 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/messages` | EPIC_16 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/internal-notes` | EPIC_16 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/approval-requests/{approvalRequestId}/decision` | EPIC_09 | planned | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |

## Catalog admin write contract

Nguồn chi tiết: [`admin-stable.openapi.yaml`](./admin-stable.openapi.yaml) cho tất cả thao tác catalog admin (đọc và ghi). Catalog admin chỉ dùng REST; gRPC chỉ internal.

### Envelope và quyền

- Mọi response dùng `ApiResponse`: `success`, `code`, `message`, `data`, `errors`, `requestId`, `timestamp`.
- `CATALOG_VIEW`: đọc danh sách/chi tiết/lịch sử giá.
- `CATALOG_MANAGE`: tạo/sửa Product, SKU, giá, ảnh.
- `CATALOG_APPROVE` (hoặc role `ADMIN`): `POST /products/{productId}/transition`.
- Actor audit lấy từ JWT `sub`; `X-Actor-Principal-Id` chỉ là fallback nội bộ, không cấp quyền.

### PATCH Product — toàn bộ nội dung

`PATCH /api/v1/admin/catalog/products/{productId}` cho phép sửa mọi nội dung:

| Field | Kiểu / giới hạn | Ghi chú |
| --- | --- | --- |
| `name` | string ≤ 500 | Không rỗng nếu gửi |
| `slug` | string ≤ 500 | Không rỗng; unique toàn catalog |
| `categoryId` | uuid | Danh mục phải tồn tại |
| `description` | string ≤ 5000 | Optional |
| `ingredients` | string ≤ 2000 | Optional |
| `storageGuide` | string ≤ 1000 | Optional |
| `ocopStar` | integer 3–5 | Optional |
| `ocopCertificateNo` | string ≤ 100 | Optional |
| `story` | string ≤ 5000 | Câu chuyện thương hiệu/làng nghề |
| `expectedVersion` | integer ≥ 0 | **Bắt buộc** |

Quy ước PATCH:

- Không gửi hoặc gửi `null` → giữ nguyên.
- Chuỗi rỗng `""` → xóa field optional (`description`, `ingredients`, `storageGuide`, `ocopCertificateNo`, `story`).
- `name`/`slug` nếu gửi lên thì không được rỗng.
- `expectedVersion` lệch revision hiện tại → `409 CATALOG_VERSION_CONFLICT`.
- `PUT /products/{productId}` là legacy (deprecated); client mới dùng PATCH.

Ví dụ cập nhật `story`:

```http
PATCH /api/v1/admin/catalog/products/6f1b1a2e-0f8e-4d5d-8f0b-9a8c7d6e5f40
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "story": "Câu chuyện làng nghề mè xửng Cồn Hến — hơn 200 năm gìn giữ hương vị Huế.",
  "expectedVersion": 3
}
```

Ví dụ xóa mô tả cũ và cập nhật OCOP:

```json
{
  "description": "",
  "ocopStar": 5,
  "ocopCertificateNo": "OCOP-HUE-2026-001",
  "expectedVersion": 4
}
```

### PATCH SKU

`PATCH /api/v1/admin/catalog/products/{productId}/skus/{skuId}`

| Field | Ghi chú |
| --- | --- |
| `variantName` | Không rỗng nếu gửi |
| `weightValue` | > 0 |
| `weightUnit` | `GRAM` / `KILOGRAM` |
| `flavor` | Chuỗi rỗng để xóa |
| `packagingType` | `HOP_GIAY_KRAFT`, `HOP_THIEC`, `GOI_HUT_CHAN_KHONG`, `HOP_QUA_TANG`, `DANG_ROI` |
| `shelfLifeDays` | > 0 |
| `saleStatus` | `ON_SALE` / `PAUSED` / `DISCONTINUED` (chấp nhận `ACTIVE` / `SUSPENDED` / `ARCHIVED`) |
| `expectedVersion` | **Bắt buộc** |

- `skuCode` bất biến, không đổi qua PATCH.
- PATCH SKU không đổi giá; giá dùng endpoint `/prices`.

### Giá theo kênh (Temporal Pricing - Database V11)

- `GET .../skus/{skuId}/prices`: Trả mọi bản ghi giá (ACTIVE, SUPERSEDED, EXPIRED) của SKU.
- `PUT .../skus/{skuId}/prices`: Cập nhật bảng giá áp dụng ngay (tự động supersede giá ACTIVE hiện hành tại `now` và tạo bản ghi mới từ `now -> infinity`). Hỗ trợ `expectedVariantVersion` để chống conflict đồng thời.
- `POST .../skus/{skuId}/prices`: Lên lịch hoặc tạo một slot giá mới theo khoảng `[effectiveFrom, effectiveTo)`. Bắt buộc `effectiveTo > effectiveFrom`. Chống chồng lấn khoảng hiệu lực bằng PostgreSQL Exclusion Constraint (`tstzrange` + `&&`); vi phạm trả về `409 Conflict`.
- `DELETE .../skus/{skuId}/prices/{priceId}`: Hủy slot giá đã lên lịch trong tương lai (`effectiveFrom > now`). Cấm xóa cứng bản ghi giá đã hoặc đang có hiệu lực.
- Kênh: `D2C_WEB`, `MOBILE_APP`, `B2B_WHOLESALE`, `POS_QUAY`, `SHOPEE`, `TIKTOK`.
- VND dùng `amountUnits`, `amountNanos = 0`.
- Outbox event: Chỉ phát sự kiện `vn.omama.catalog.product.price.changed.v1` và tăng `price_revision` khi giá thay đổi có hiệu lực ngay (`effectiveFrom <= now`). Giá tương lai không phát event thay đổi giá giỏ hàng.

### Ảnh sản phẩm

- Lưu `objectKey` (MinIO/S3); response trả thêm `url = https://cdn.omama.vn/ + objectKey` (giữ nguyên nếu objectKey là URL tuyệt đối).
- `role`: `COVER` | `GALLERY` | `THUMBNAIL` (mặc định `GALLERY`).
- Phân tách hai lớp trạng thái:
  - `scanStatus`: `PENDING` | `CLEAN` | `INFECTED` | `FAILED` (do Media Service kiểm tra an toàn mã độc cập nhật).
  - `mediaStatus`: `PENDING_SCAN` → `PENDING_REVIEW` → `APPROVED` | `REJECTED` (do Content Moderator duyệt).
- Ảnh chỉ được hiển thị công khai khi đồng thời: `scanStatus == CLEAN` VÀ `mediaStatus == APPROVED`.
- Đăng ký ảnh mới (`POST /images`): server tự gán `mediaStatus = PENDING_SCAN`, `scanStatus = PENDING`, `uploadedBy = actorId`.
- Sửa metadata (`PATCH /images/{imageId}`): chỉ sửa `altText`, `position`, `role`. Cấm sửa `mediaStatus` hoặc `scanStatus`. Không cho phép chuyển ảnh đã `APPROVED` thành `COVER` qua PATCH.
- Duyệt ảnh (`POST /images/{imageId}/review`): yêu cầu quyền `CATALOG_APPROVE`. Chỉ duyệt khi `scanStatus == CLEAN` và `mediaStatus == PENDING_REVIEW`. Tuân thủ Separation of Duties (người upload không được tự duyệt).
- Bất biến: tối đa một ảnh `COVER` + `APPROVED` + `CLEAN`. Khi duyệt một ảnh COVER thành APPROVED, cover cũ tự hạ xuống `GALLERY` trong cùng transaction khóa dòng Product.
- `DELETE` chỉ xóa metadata; object trên storage do job dọn dẹp riêng xử lý.
- Optimistic locking: PATCH và review ảnh đều yêu cầu `expectedVersion`.

### Vòng đời

- `approvalStatus`: `DRAFT` → `PENDING` → `APPROVED` / `REJECTED`.
- `listingStatus` (DB): `ACTIVE` / `SUSPENDED` / `ARCHIVED`; `saleStatus` (contract): `ON_SALE` / `PAUSED` / `DISCONTINUED`.
- Public chỉ trả sản phẩm `APPROVED` + `ACTIVE`.

| `transition` | Kết quả |
| --- | --- |
| `SUBMIT_FOR_REVIEW` | `DRAFT`/`REJECTED` → `PENDING` |
| `APPROVE` | `PENDING` → `APPROVED` |
| `REJECT` | `PENDING` → `REJECTED` |
| `SUSPEND` | `APPROVED` → tạm ngừng bán |
| `ARCHIVE` | Ngừng kinh doanh vĩnh viễn |

### Mã lỗi Catalog

| Code | HTTP | Mã nội bộ | Khi nào |
| --- | --- | --- | --- |
| `CATALOG_PRODUCT_NOT_FOUND` | 404 | 203 | `productId` không tồn tại |
| `CATALOG_VARIANT_NOT_FOUND` | 404 | 204 | `skuId` không tồn tại |
| `CATALOG_CATEGORY_NOT_FOUND` | 404 | 205 | `categoryId` không tồn tại |
| `CATALOG_SKU_DUPLICATE` | 409 | 207 | `skuCode` đã tồn tại |
| `CATALOG_PRODUCT_INVALID_STATE` | 422 | 208 | Transition không hợp lệ với trạng thái hiện tại |
| `CATALOG_SLUG_DUPLICATE` | 409 | 210 | `slug` đã được dùng |
| `CATALOG_VERSION_CONFLICT` | 409 | 211 | `expectedVersion` lệch revision hiện tại |
| `CATALOG_IMAGE_NOT_FOUND` | 404 | 212 | `imageId` không tồn tại hoặc không thuộc sản phẩm |
| `CATALOG_IMAGE_INVALID` | 400 | 213 | `role` / `mediaStatus` không hợp lệ |
| `CATALOG_VARIANT_INVALID` | 400 | 214 | `weightUnit` / `packagingType` / `saleStatus` không hợp lệ |
| `CATALOG_VARIANT_MISMATCH` | 400 | 202 | `skuId` không thuộc `productId` |

### BFF `apps/web-user`

- Route `/api/admin/catalog/[...path]` hỗ trợ `GET`, `POST`, `PUT`, `PATCH`, `DELETE`.
- Forward query `q`, `status`, `page`, `pageSize`, body JSON và header `Idempotency-Key`.
- Đọc yêu cầu `PRODUCT_VIEW`/`CATALOG_VIEW`; ghi yêu cầu `CATALOG_MANAGE`/`PRODUCT_MANAGE` hoặc role `ADMIN`.
- Lỗi upstream chuẩn hóa về `{ code, message, errors, requestId }`.

## Acceptance rules

1. Every list endpoint has both non-empty and empty contract tests.
2. Every protected endpoint has separate `401` and `403` tests.
3. Every detail endpoint returns `404` without leaking whether an out-of-scope resource exists.
4. Every mutation validates `400/422`, transition or revision conflict `409`, and retry behavior.
5. Every endpoint has controlled latency and `500/503` UI tests.
6. Error responses preserve `code` and `requestId`; clients do not branch on translated `message`.

`GET /api/v1/admin/roles` also accepts the `identity-service-admin` client-credentials token for audience `identity-roles-api`. This service permission covers the list operation only.

Inventory receipts require `INVENTORY_RECEIPT_CREATE` globally or for the SKU warehouse. The API checks the expected revision and replays the stored result for matching idempotent retries.

Fulfillment staff routes validate the `omama-api` audience by default. Missing or invalid credentials return 401. Valid credentials without an eligible staff identity or the required grant return 403.

## Private Inventory operations (contract consolidation — 2026-11-21)

These operations are documented in admin-stable.openapi.yaml with per-operation
private servers. They are not Employee Web routes and are not published by Kong.
The employee authentication rule above continues to apply to business APIs;
these monitoring endpoints use network isolation on INVENTORY_METRICS_ADDR,
without staff JWT/session authentication. Do not expose them via the public gateway.
No Customer/Employee UI state or business authorization policy changes here.

| Operation | Success | Failure | Loading / empty | Authentication |
|---|---|---|---|---|
| GET /readyz | 200 text/plain: consumer disabled or healthy | 503 shared Error: enabled consumer starting/blocked/stopped | N/A: operations probe, no UI or collection | No JWT on private listener; 401/403 N/A |
| GET /metrics | 200 Prometheus text: process counters and group lag | Listener/network failure is transport-level; no application error response declared | N/A: operations scrape, no UI or collection | No JWT on private listener; 401/403 N/A |

Readiness failures include stable code PROCUREMENT_CONSUMER_UNHEALTHY, safe
message, requestId and X-Request-Id. Lag -1 means unavailable/uninitialized.
Operation-level servers and security override the document's production gateway
and employee authentication defaults. The independent Inventory operations spec
was consolidated here; runtime URLs and response bodies are unchanged.

Transport amendment 0.4.0 (2026-11-27): use canonical Catalog/Media from PR #175 per [current guide](traceability-workflow-grpc.md). OCOP upload-session/complete precedes document:<fileId> evidence; Media-enabled submit/publish invalid file400/dependency503 keeps draft/approval and receipt unchanged. Publication replay remains successful while a later Media outage is retried by the durable bind worker. QR/public DTO schema is unchanged. See [report10](../04_testing/traceability-service/10_catalog_media_alignment_2026-11-27.md); UI/provider acceptance remains pending.

Inventory receipt provider includes batchId/batchCode/skuCode/warehouseId on fresh APPLIED results in the same idempotent transaction. Legacy replay may omit them: preserve APPLIED and resolve context separately, never submit another receipt to recover traceability binding.

Identity implementation checkpoint28/11: staff admin adapters require a known ACTIVE EMPLOYEE; scoped Traceability/Inventory BFF allowlists, no-store and bounded body/response are implemented. The dedicated TLS staff-grants RPC is enabled explicitly; legacy listener9016 remains unchanged. Provider tests pass on isolated PostgreSQL; live browser login/scope/Origin acceptance is still required.

### Inventory workbench provider verification — 2026-11-28

`GET /api/v1/admin/inventory/workbench` reads Inventory balances in a repeatable-read transaction. It intersects `INVENTORY_VIEW` and `WAREHOUSE_WORKBENCH_VIEW` per warehouse, including mixed GLOBAL/WAREHOUSE grants; unrelated grants cannot widen access. A real PostgreSQL receipt-to-workbench test covers positive balances, two warehouses, missing permission and crossed scopes. Responses are `no-store`. More than 100 SKUs or 1000 eligible batches returns 503, allowing the error/retry state without presenting incomplete stock. SKU labels remain codes; no Catalog product names or pending approvals are fabricated.

Identity staff transport aliases use `/api/v1/admin/trace-api/**` in accordance with the Identity namespace rule. The proxy forwards to the versioned Traceability `/api/v2/admin/trace/**` contract; the downstream business version is unchanged.

### Traceability Web workflow — 2026-11-28

Loading/empty/error/auth states đi qua [Web BFF draft](traceability-web-bff-draft.openapi.yaml). Identity session phải giữ permissionGrants; unrelated GLOBAL role không mở rộng WAREHOUSE trace grants. Draft chưa lưu chặn submit/status actions; conflict tạo trùng giữ nội dung đang nhập và đưa link mở profile đã có, không giả đã lưu. Receipt APPLIED vẫn được giữ nếu refresh/context/upload lỗi; retry dùng cùng key cho cùng intent. Scan mới cần lưu draft trước khi mở từ hồ sơ; stored evidence cần quyền read_evidence và kho hiện hành. QR chỉ dùng public DTO còn eligible, PNG/SVG/print recheck; portal in tem tách khỏi shell, xóa sau afterprint. Không restore SUSPENDED/ARCHIVED. [Report11](../04_testing/traceability-service/11_local_workflow_2026-11-28.md) ghi ca live/fixture cụ thể.

## Order Media Evidence (Media Service integration)

| Operation | Success / empty | Loading | Auth / scope | Error |
|---|---|---|---|---|
| GET `/api/v1/admin/media/{file_id}/download-url` | 200 `{ download_url, expires_at }` (15m presigned GET URL); empty N/A | Pending presigned URL fetch | 401 missing/invalid token; 403 insufficient payment/evidence staff permission | 503 `MEDIA_SERVICE_UNAVAILABLE` when Media gRPC is unreachable |
| `/api/v1/admin/payments/reconciliation-cases*`, `/api/v1/admin/payments/refunds*`, `/api/v1/admin/payments/evidence*` | 200/201/202 per operation; empty lists return `200` with empty `items` | Pending request / mutation in progress | 401 missing/invalid staff bearer; 403 missing `payment.*` permission or scope | 400/409 validation or idempotency/version conflict; 429 rate limited; 503 upstream unavailable |
