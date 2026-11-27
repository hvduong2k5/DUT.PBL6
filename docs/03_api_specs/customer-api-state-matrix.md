# Customer API State Matrix

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
| `GET /api/v1/products/variants/{sku}` | EPIC_04 | stable | 200 | N/A | 400/— | — | 403 | 404 | — | — | 500/503 |
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

## F-53 guest checkout trust boundary (B-G0)

All guest feature flags default OFF. OIDC access-token renewal keeps the Customer identity and forwards the renewed credential through the configured commerce topology. Refresh-only cookies still count as Customer session material; failed renewal never falls back to a guest. Customer login presence takes precedence over any guest cookie, including invalid/expired login sessions. Guest fixed capabilities cover cart, checkout and exact own-order reads only; account/profile/history, employee and payment writes retain customer/employee authorization. Payment grants cover exact-order payment reads only and never use guest-session validation. No fallback after either validator rejects a token.

| Operation | Success / loading | Empty | Errors | Authentication / authorization |
| --- | --- | --- | --- | --- |
| `POST /api/v1/guest-sessions` | 201 / disable duplicate action | No session for plain page views or an empty cart read | 400 input overrides, 429 quota, 503 persistence | Anonymous; 403 when issuance flag off |
| `DELETE /api/v1/guest-sessions/current` | 204 / revoke pending | N/A | 503 | 401 invalid session; 403 wrong principal |
| `POST /api/v1/orders/{order_id}/guest-payment-access` | 201 / mint pending | N/A | 400 key, 404 ownership, 409 replay, 503 | Active owning guest session only |
| `DELETE /api/v1/orders/{order_id}/guest-payment-access` | 204 / revoke pending | N/A | 400 key, 404 ownership, 503 | Active owning guest session only |

AUTH-11..16 must pass before enabling cart/COD. Customer step-up and cross-customer isolation remain mandatory. The rc.2 pick, deployment, Kong restart after recreation, live COD/SePay/merge proofs require Muse audit and Bao approval; local checks alone are not release acceptance.


## F-53 guest cart and COD (B-G1)

Guest cart mutations require `GUEST_CART_MUTATION_ENABLED` plus the storefront guest gate; disabled mutations return 403 before lazy issuance. Mutations are limited to 60/session/minute (429). Customer account surfaces keep their existing authorization.

Guest COD quote/accept requires `GUEST_COD_CHECKOUT_ENABLED`, uses an inline address and does not fetch Profile. Loading disables duplicate submission, empty cart stays 200, stale revision is 409, invalid/unserviceable address is 422, and dependency failure is 503. A transaction serializes admission on the session and limits unpaid pending orders plus in-flight checkout to 3; the fourth returns 429 `GUEST_PENDING_ORDER_LIMIT` without creating an operation. Idempotent replay remains available at quota. Guest checkout shows the authoritative order code and phone-OTP tracking guidance. Cross-session order reads return 404; account history and cancellation remain customer-only for session guests.

The simulator validates province/ward pairs against the canonical administrative dataset shared with profile-service; there is no hardcoded pair. Real Catalog COD retains the `dev` default collection policy `cod-collection-1.0`; `ORDER_COD_POLICY_REFERENCE` optionally overrides it, and malformed values return 422 `COD_NOT_AVAILABLE`. Guest COD still requires its independent feature flag. The server snapshots the final collect amount. `CONFIRMED_COD` remains `UNPAID`, with no payment deadline or bank receiver; Finance settlement remains required for completion. `PROOF-COD-01` remains a live post-audit acceptance requirement.


## F-53 guest VietQR (B-G2)

`GUEST_VIETQR_CHECKOUT_ENABLED` is independent of COD. The Web lazily exchanges the active guest session for an exact-order payment read grant, stores it in an httpOnly cookie scoped to that order's payment BFF routes, and forwards only the grant on polling. The session token cannot read payment; another order's grant returns 401. Any present invalid Customer credential returns an error without switching to the guest grant.

| Operation | Success/loading | Failure/auth |
| --- | --- | --- |
| Web `POST /api/payments/orders/{orderId}/guest-access` | 204 / disable duplicate action; token never exposed | 400 input, 401 missing/invalid credential, 403 flag/principal, 404 ownership, 409 replay, 503 dependency |
| `GET /api/v1/orders/{order_id}/payment` | 200 / display current state and bounded polling | 401 grant, 403 principal, 404 owner, 503 dependency |
| `GET /api/v1/orders/{order_id}/payment-instructions` | 200 / display QR only while payable | 401 grant, 403 principal, 404 owner, 409 expired, 503 dependency |

SePay verification remains HMAC-SHA256 over timestamp + raw payload (five-minute window), durable deduplicated observation, canonical API row verification and exactly-once allocation/finalization. An observation alone cannot settle. Late money stays in reconciliation and cannot auto-fulfil. `PROOF-VQR-01` requires the separately approved live transfer and duplicate replay; local TEST statements do not satisfy that live proof.


## F-53 cart merge after login/registration (B-G3)

| Operation | Success/loading/empty | Errors/auth |
| --- | --- | --- |
| `POST /api/v1/cart` | 200 persisted outcome; one Customer revision; empty source allowed | 400 body/key, 401 invalid/expired/revoked source, 403 Customer/flag, 409 key/source conflict, 503 Catalog/store; all changes rollback together |
| `GET /api/v1/cart?merge_key=...` | 200 original persisted merge result even after source revocation | 400 key, 401 Customer session, 403 wrong principal, 404 another Customer/key, 503 store |
| Web `POST /api/cart/merge` | 200 merged result and clear guest cookie; 204 when no guest cookie or rollout off | Login stays successful on merge error; preserve source cookie and show retry in cart |

The source credential stays in an httpOnly cookie and is sent to Order in a dedicated body field through the existing authenticated Customer proxy. A stable server-derived merge key makes retries/tabs idempotent. Outcomes show merged / adjusted / dropped with per-SKU reasons. Failure never breaks the login session. Converted carts use SQL authority and the same scope locks for mutation and checkout cleanup; flag-off cannot restore stale Redis carts. `PROOF-MERGE-01` still requires the audited live login flow and DB ledger evidence.
