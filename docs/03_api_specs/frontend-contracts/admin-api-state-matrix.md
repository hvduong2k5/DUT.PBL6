# Internal Employee API State Matrix

Updated: 2026-10-05

This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.

## State semantics

- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.
- **Empty:** successful `200` response with an empty collection and zero totals, never `404`.
- **Unauthorized:** `401` when credentials/session/grant are missing, invalid or expired.
- **Forbidden:** `403` when identity is known but capability, permission, ownership or scope is insufficient.
- **Error:** `4xx` is actionable client/business failure; `5xx` is server/dependency failure. All errors use the shared `Error` schema.
- **Success:** only documented `2xx` codes are compatible. Mutations requiring asynchronous processing use `202`.

Employee APIs require authentication, effective permission and server-side data-scope enforcement. A hidden menu is not authorization.

## Stable operations

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `GET /api/v1/admin/access/employees` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/access/employees/{employeeId}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/access/permissions` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/access/permissions/{permissionCode}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/access/roles` | EPIC_22 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/access/roles/{roleCode}` | EPIC_22 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests` | EPIC_18 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}` | EPIC_18 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/b2b/quote-requests/{requestId}/actions` | EPIC_18 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}/files` | EPIC_18 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/b2b/quote-requests/{requestId}/versions` | EPIC_18 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/admin/catalog/products` | EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
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
| `POST /api/v1/admin/access/employees/invitations` | EPIC_22 | draft | 202 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PATCH /api/v1/admin/access/employees/{employeeId}/status` | EPIC_22 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PUT /api/v1/admin/access/employees/{employeeId}/role-assignments` | EPIC_22 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/access/roles` | EPIC_22 | draft | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PATCH /api/v1/admin/access/roles/{roleCode}` | EPIC_22 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PUT /api/v1/admin/access/roles/{roleCode}/permissions` | EPIC_22 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products` | EPIC_04 | draft | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}` | EPIC_04 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/skus` | EPIC_04 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/orders/{orderId}/actions` | EPIC_08 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/packing/tasks/{taskId}/assignments` | EPIC_10 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/shipping/shipments/{shipmentId}/actions` | EPIC_11 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/payments/cases/{caseId}/actions` | EPIC_07, EPIC_24 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/returns/cases/{caseId}/actions` | EPIC_14 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/actions` | EPIC_16 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/messages` | EPIC_16 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/support/tickets/{ticketId}/internal-notes` | EPIC_16 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/inventory/approval-requests/{approvalRequestId}/decision` | EPIC_09 | planned | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |

## Acceptance rules

1. Every list endpoint has both non-empty and empty contract tests.
2. Every protected endpoint has separate `401` and `403` tests.
3. Every detail endpoint returns `404` without leaking whether an out-of-scope resource exists.
4. Every mutation validates `400/422`, transition or revision conflict `409`, and retry behavior.
5. Every endpoint has controlled latency and `500/503` UI tests.
6. Error responses preserve `code` and `requestId`; clients do not branch on translated `message`.
