# Customer API State Matrix

Transport amendment2026-11-27: target0.3.0 uses gRPC for internal business reads/resolver/current grants; Web/BFF REST and UI states below remain unchanged. See [gRPC guide](traceability-workflow-grpc.md) and [report09](../04_testing/traceability-service/09_grpc_transport_2026-11-27.md). Do not use the superseded internal REST proposals from0.2.2.

## Order batch traceability target — 2026-11-21 (draft; resolver implemented, Order/UI integration pending)

Backend checkpoint 2026-11-26: the Traceability-owned internal resolver is implemented and locally tested. The customer Order subresource, sealed packing proof and Web mapping remain peer deliverables. See [verification scope](../04_testing/traceability-service/08_scoped_backend_2026-11-26.md); no full customer E2E or production verification is claimed.

Review target0.2.2: every line-batch row includes authoritative sku_code. Current Order manifests allow one line per SKU; the UI bridges exact typed SKU and verifies quantities when detail lacks canonical IDs. Duplicate/mismatched mapping becomes UNAVAILABLE, never a guessed link. More than1000rows/512KiB is root UNAVAILABLE with empty items/total0; unresolved chunks within the5second outer budget stay UNAVAILABLE. This does not erase or invalidate the order detail itself.

Updated 2026-11-25: see [Order handoff](traceability-workflow-peer-handoff.md#4-order-ord-trc-01) for canonical line IDs and guest session capability. NOT_APPLICABLE applies to cancellation before packing; sealed proof remains historical evidence after return/cancellation. Missing legacy proof is UNAVAILABLE, never reconstructed from current Inventory allocations.

See [design](traceability-workflow-design-2026-11-21.md) and [draft subresource](traceability-workflow-draft.openapi.yaml). Existing order detail and generated stable examples remain unchanged.

| Flow | Success / empty | Loading | Error | Auth |
| --- | --- | --- | --- | --- |
| Order traceability | READY: one row per sealed line/batch with quantity; PACKING_PENDING/NOT_APPLICABLE: 200 empty items and total=0 | Separate skeleton; order detail stays usable | Whole proof unavailable: 200 UNAVAILABLE empty; resolver item failure: UNAVAILABLE row; order DB unavailable: 503; retry only traceability read | Same detail ownership and existing guest capability; 401 absent/invalid, 403 known insufficient capability/ownership, 404 nonexistent order |
| Batch links | AVAILABLE opens public URL; NOT_AVAILABLE has no fabricated link; NO_LONGER_PUBLIC links only to safe public 410 | Disable link until data available | Never substitute sample/latest SKU record; no stale eligibility badge | Draft/internal evidence and customer/order PII never appear in public record |
| Public QR / footer lookup | Anonymous 200 public profile; manual entry uses /trace and /trace/{code} | Public-record loading state | 404 unknown; 410 withdrawn/expired; 503 temporary failure; no old valid profile fallback | No login required; customer history still requires its own order authorization |


Updated: 2026-11-14

This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.

## State semantics

- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.
- **Empty:** successful `200` response with an empty collection and zero totals, never `404`.
- **Unauthorized:** `401` when credentials/session/grant are missing, invalid or expired.
- **Forbidden:** `403` when identity is known but capability, permission, ownership or scope is insufficient.
- **Error:** `4xx` is actionable client/business failure; `5xx` is server/dependency failure. All errors use the shared `Error` schema.
- **Success:** only documented `2xx` codes are compatible. Mutations requiring asynchronous processing use `202`.

Customer APIs distinguish Guest, Registered Customer and verified B2B Customer capabilities. Ownership/grants must be checked server-side.

## Stable operations

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `GET /api/v1/users/me` | EPIC_01 | stable | 200 | N/A | —/— | 401 | — | 404 | — | — | 500/503 |
| `PUT /api/v1/users/me/password` | EPIC_01 | stable | 200 | N/A | 400/422 | 401 | — | 404 | — | — | 500/503 |
| `GET /api/v1/public/auth/health` | EPIC_01 | stable | 200 | N/A | — | — | — | — | — | — | 503 |
| `GET /api/v1/public/auth/login` | EPIC_01 | stable | 200 | N/A | 400/— | — | — | — | — | — | 500/503 |
| `GET /api/v1/public/auth/callback` | EPIC_01 | stable | 302 | N/A | 400/— | — | — | — | — | — | 500/503 |
| `GET /api/v1/public/auth/session` | EPIC_01 | stable | 200 | N/A | —/— | 401 | — | — | — | — | 500/503 |
| `POST /api/v1/public/auth/tokens/refresh` | EPIC_01 | stable | 200 | N/A | —/— | 401 | — | — | — | — | 500/503 |
| `POST /api/v1/public/auth/logout` | EPIC_01 | stable | 200 | N/A | —/— | 401 | — | — | — | — | 500/503 |
| `POST /api/v1/public/customer-registrations` | EPIC_01 | stable | 200 | N/A | 400/422 | — | — | — | 409 | — | 500/503 |
| `GET /api/v1/b2b/context` | EPIC_18 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/b2b/quote-requests` | EPIC_18 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/b2b/quotes/{quoteId}` | EPIC_18 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/b2b/quotes/{quoteId}/accept` | EPIC_18 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/cart` | EPIC_05 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `DELETE /api/v1/cart` | EPIC_05 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/cart/items` | EPIC_05 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PUT /api/v1/cart/items/{item_id}` | EPIC_05 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/cart/items/{item_id}` | EPIC_05 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/categories` | EPIC_03 | stable | 200 | 200 (collection empty) | 400/— | — | — | — | — | — | 500/503 |
| `POST /api/v1/checkout` | EPIC_06 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/customer/capabilities` | EPIC_01, EPIC_18 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/locations/checkout` | EPIC_02, EPIC_06 | stable | 200 | N/A | 400/— | — | — | — | — | — | 500/503 |
| `GET /api/v1/loyalty/points` | EPIC_17 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/orders` | EPIC_08 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/orders/{order_id}` | EPIC_08 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/orders/{order_id}/cancel` | EPIC_08 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/orders/{order_id}/tracking` | EPIC_08 | stable | 200 | 200 (collection empty) | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/orders/guest-access/challenges` | EPIC_08 | stable | 202 | N/A | 400/422 | — | — | — | 409 | 429 | 500/503 |
| `POST /api/v1/orders/guest-access/challenges/{challengeId}/verify` | EPIC_08 | stable | 200 | N/A | 400/422 | — | — | 404 | 409 | 429 | 500/503 |
| `POST /api/v1/orders/guest-access/grants` | EPIC_08 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/payments/vietqr/callback` | EPIC_07 | stable | 200 | N/A | 400/422 | 401 | — | — | 409 | — | 500/503 |
| `GET /api/v1/products` | EPIC_03, EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | — | — | — | — | — | 500/503 |
| `GET /api/v1/products/{id_or_slug}` | EPIC_03, EPIC_04 | stable | 200 | N/A | —/— | — | — | 404 | — | — | 500/503 |
| `GET /api/v1/products/search` | EPIC_03, EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | — | — | — | — | — | 500/503 |
| `GET /api/v1/profile` | EPIC_02 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `PUT /api/v1/profile` | EPIC_02 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/profile/addresses` | EPIC_02 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/profile/addresses` | EPIC_02 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PUT /api/v1/profile/addresses/{address_id}` | EPIC_02 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/profile/addresses/{address_id}` | EPIC_02 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/promotions/validate` | EPIC_17 | stable | 200 | N/A | 400/422 | — | — | — | 409 | — | 500/503 |
| `GET /api/v1/promotions/vouchers` | EPIC_17 | stable | 200 | 200 (collection empty) | 400/— | — | — | — | — | — | 500/503 |
| `POST /api/v1/returns` | EPIC_14 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/returns/{return_id}` | EPIC_14 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/reviews` | EPIC_15 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/reviews/products/{product_id}` | EPIC_15 | stable | 200 | 200 (collection empty) | —/— | — | — | 404 | — | — | 500/503 |
| `GET /api/v1/support/context` | EPIC_16 | stable | 200 | N/A | 400/— | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/support/tickets` | EPIC_16 | stable | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `GET /api/v1/support/tickets/{ticketId}` | EPIC_16 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/support/tickets/{ticketId}/messages` | EPIC_16 | stable | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | 429 | 500/503 |
| `GET /api/v1/trace/{qr_code}` | EPIC_19 | stable | 200 | N/A | —/— | — | — | 404 | — | — | 500/503 |
| `GET /api/v2/trace/{qr_code}` | EPIC_19 | P8 | 200 (nullable fields and verification method) | N/A | —/— | — | — | 404 unknown, 410 unpublished after publication | — | — | 503 |
| `POST /api/v1/webhooks/logistics/ghn` | EPIC_11, EPIC_12 | stable | 200 | N/A | 400/422 | 401 | — | — | 409 | — | 500/503 |
| `POST /api/v1/webhooks/marketplace/shopee` | EPIC_11, EPIC_12 | stable | 200 | N/A | 400/422 | 401 | — | — | 409 | — | 500/503 |
| `POST /api/v1/webhooks/marketplace/tiktok` | EPIC_11, EPIC_12 | stable | 200 | N/A | 400/422 | 401 | — | — | 409 | — | 500/503 |

## Draft and planned operations

These rows are proposals. Backend may refine them, but must record the decision and update the OpenAPI contract before frontend integration.

| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `POST /api/v1/media/upload-sessions` | EPIC_14, EPIC_16, EPIC_18 | draft | 201 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/media/upload-sessions/{uploadSessionId}/complete` | EPIC_14, EPIC_16, EPIC_18 | draft | 202 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/orders/{orderId}/returns/eligibility` | EPIC_08 | draft | 200 | 200 (collection empty) | 400/422 | 401 | 403 | 404 | — | — | 500/503 |
| `GET /api/v1/returns` | EPIC_14 | draft | 200 | 200 (collection empty) | 400/422 | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/returns/{returnId}/supplements` | EPIC_14 | draft | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/support/tickets` | EPIC_16 | draft | 200 | 200 (collection empty) | 400/422 | 401 | 403 | — | — | — | 500/503 |
| `POST /api/v1/b2b/quote-requests/{requestId}/messages` | EPIC_18 | planned | 201 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/loyalty/transactions` | EPIC_17 | planned | 200 | 200 (collection empty) | 400/422 | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/notifications` | EPIC_28 | planned | 200 | 200 (collection empty) | 400/422 | 401 | 403 | — | — | — | 500/503 |
| `PATCH /api/v1/notifications/{notificationId}/read` | EPIC_28 | planned | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/notification-preferences` | EPIC_28 | planned | 200 | N/A | 400/422 | 401 | 403 | — | — | — | 500/503 |
| `PUT /api/v1/notification-preferences` | EPIC_28 | planned | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `POST /api/v1/ai/shopping/conversations` | US-AI-01 | draft | 201 | N/A | 400/422 | 401 | 403 | — | — | 429 | 500/503 |
| `GET /api/v1/ai/shopping/conversations/{conversation_id}` | US-AI-01 | draft | 200 | N/A | —/— | 401 | 403 | 404 | — | 429 | 500/503 |
| `POST /api/v1/ai/shopping/conversations/{conversation_id}/messages` | US-AI-01 | draft | 200 SSE | N/A | —/422 | 401 | 403 | 404 | — | 429 | 500/503 |
| `POST /api/v1/ai/shopping/feedback` | US-AI-01, US-AI-04 | draft | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | 429 | 500/503 |
| `GET /api/v1/ai/recommendations` | US-AI-04 | draft | 200 | 200 (items empty) | —/422 | 401 | 403 | — | — | 429 | 500/503 |
| `GET /api/v1/ai/consent` | US-AI-01 | draft | 200 | N/A | —/— | 401 | — | — | — | 429 | 500/503 |
| `PUT /api/v1/ai/consent` | US-AI-01 | draft | 200 | N/A | —/422 | 401 | — | — | — | 429 | 500/503 |

## Acceptance rules

1. Every list endpoint has both non-empty and empty contract tests.
2. Every protected endpoint has separate `401` and `403` tests.
3. Every detail endpoint returns `404` without leaking whether an out-of-scope resource exists.
4. Every mutation validates `400/422`, transition or revision conflict `409`, and retry behavior.
5. Every endpoint has controlled latency and `500/503` UI tests.
6. Error responses preserve `code` and `requestId`; clients do not branch on translated `message`.

Web and Mobile show the loyalty wallet entry only when the capability projection includes `LOYALTY_VIEW`. Direct navigation remains subject to the same capability check.
