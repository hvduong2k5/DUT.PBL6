# Internal Employee API State Matrix

### Company dispatcher Web UI (opt-in)

- `/admin/shipping/dispatch` requires SHIPMENT_ASSIGN for navigation and uses BFF GET/POST `/api/admin/shipping/company/shipments/{shipmentId}/assignment` plus GET `.../assignment/history`. Same SHIPPING_COMPANY_BFF_ENABLED flag, OIDC bearer or identity-bff session relay; no mock 4030 fallback. Existing management page is preserved and links to this screen when enabled.
- Input is explicit shipment UUID and Identity driver principalId (not Keycloak sub); no fabricated employee names or directory autocomplete. Backend verifies active employee and GLOBAL grants. POST also requires USER_VIEW#GLOBAL. UI does not assign IAM roles or bypass the provider's before-handover restriction.
- Success: current assignment/reassignment snapshot and immutable ordered audit. Empty: successful history with items []/total 0 and missing current assignment means initial expected_version 0. A current 404 alone never authorizes version 0 for an unknown shipment. Loading disables editing/submission. Selected UUID must match loaded state; changing it requires reload.
- Retry: identical payload retains key after network error; 409 invalidates loaded state and requires refresh. Missing input/UUID/reason is blocked; reason <=500 characters without controls. POST actor is always from authenticated backend principal, never UI body. Errors: 400 malformed/override, 401 sign-in, 403 permission/directory access, 404 missing/non-INTERNAL/unassigned, 409 version/lifecycle/key conflict, 413/415 payload/media, 422 ineligible target/no-op/validation, 502/503 invalid/unavailable upstream. No falsely successful UI state when save fails.
- No browser/live account acceptance yet; live dispatch E2E remains pending. Dispatcher UI does not confirm warehouse handover, delivery, COD or refunds.

### Opt-in company delivery Web adapter

- `/api/admin/shipping/company/shipments` GET, `/{shipmentId}` GET, `/{shipmentId}/accept` POST and `/{shipmentId}/progress` POST map only to the corresponding Shipping v2 provider. `SHIPPING_COMPANY_BFF_ENABLED=false` by default. OIDC bearer-cookie mode sends server JWT direct; identity-bff forwards only OMAMA_SESSION to Identity `/api/v1/admin/shipping-api/**`, whose existing session filter refreshes/resolves the JWT. No mock fallback or session-as-JWT substitution.
- Workbench `/admin/work/delivery` switches to company UI when enabled. SHIPMENT_EXECUTE is a UI navigation gate; signed JWT grants and resource ownership remain enforced by Shipping. Lists, detail, recipient, acceptance and progress come from real provider APIs. Existing legacy UI remains unchanged when disabled.
- Success/empty: provider data, no fabricated assignee. Loading disables actions and selection; reason required for progress. Warehouse handover cannot be triggered by driver. Retry retains key for unchanged payload; refresh after 409. Errors: 400 input, 401 sign-in/expired or revoked session, 403 grant/ownership/origin, 404 missing/disabled relay, 409 conflict, 413 >4 KiB, 415 media type, 422 validation, 502 malformed/redirect upstream, 503 unavailable or missing relay configuration. Errors are safe, requestId-bearing and no-store. No JWT/internal token reaches browser responses.
- Mutations require explicit same-origin Origin; BFF forwards only server cookie or bearer according to auth mode, trusted Origin and idempotency key. No actor/role/internal-token headers or arbitrary upstream selection. No verified photo, COD settlement or refund claim. Live Identity/JWT and browser E2E acceptance remain pending; do not enable until relay/provider configuration and IAM resync are accepted.

### Identity Shipping session relay

- Allowlisted `/api/v1/admin/shipping-api/**` operations mirror management reads (SHIPMENT_VIEW#GLOBAL), assignment/history (SHIPMENT_ASSIGN#GLOBAL; assignment POST also USER_VIEW#GLOBAL), and owned workbench/accept/progress (SHIPMENT_EXECUTE#GLOBAL). No role bypass, unknown path/query or internal API forwarding. Existing session filter resolves/refreshes cookie JWT; Identity validates ACTIVE EMPLOYEE and exact grants; Shipping repeats its owning-resource checks.
- `IDENTITY_SHIPPING_API_ENABLED=false`; configure origin-only `IDENTITY_SHIPPING_API_URL`, HTTPS preferred. Private HTTP requires explicit `IDENTITY_SHIPPING_API_ALLOW_INSECURE_HTTP=true` in a trusted network. Cookie mutations must match FRONTEND_BASE_URL Origin. Response <=1 MiB, action JSON <=4 KiB, no redirects/Set-Cookie/provider auth relay back. Success mirrors provider, errors use safe Error envelope; no silent 200 fallback.
- V24 seeds SHIPMENT_ASSIGN/SHIPMENT_EXECUTE and maps only enabled/non-archived ADMIN role bundles. It creates no principal/role assignment/scope. Sync affected holders via existing IAM sync-permissions endpoint with forceRelogin=false, then refresh JWT/session; SQL alone cannot update issued JWTs or Keycloak attributes. Dispatcher also needs existing USER_VIEW#GLOBAL. Do not auto-assign a GLOBAL role to a driver.

### Assigned company-driver reads (opt-in v2 provider API)

- GET `/api/v2/admin/shipping/workbench/shipments` and `.../{shipmentId}` require signed `SHIPMENT_EXECUTE#GLOBAL` plus live ACTIVE EMPLOYEE. The backend always filters by JWT user_id and current assignment; GLOBAL is not a cross-driver read grant. Legacy v1 workbench/UI remains separate.
- List success: 200 summaries without recipient PII, current assignment version/accepted flag and shipment revision. Empty: `items: [], total: 0`; offset beyond total returns empty items with actual total. Limit 1..100 (default 50), offset 0..10000. Detail success: 200 current assigned recipient/parcel and versions; no empty success.
- Loading: skeleton and disable actions until detail loaded; after reassign/409 refresh list/detail. 401 invalid JWT; 403 missing permission/inactive employee; 400 malformed UUID/pagination or unknown/duplicate queries; 404 disabled, unknown/unowned/non-INTERNAL detail (same response); 503 Identity unavailable; 500 storage failure. No client-supplied actor/driver override. List/total and detail ownership/data use repeatable snapshots. No internal tracking notes or broad management fallback.
- Default `SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED=false`; Kong/BFF/UI connection still pending. No frontend-only ownership filtering is accepted as the security boundary.

### Internal driver progress (opt-in provider API)

- `POST /api/v2/admin/shipping/shipments/{shipmentId}/progress`: signed `SHIPMENT_EXECUTE#GLOBAL`, live ACTIVE EMPLOYEE, current assigned driver and accepted assignment required. GLOBAL never bypasses resource ownership. Default flag is off; no Kong/UI connection yet.
- Success: 200 committed receipt, or immutable original receipt on exact retry. No empty success state. Loading: prevent double submit; preserve the key/payload for network retries. Refresh assignment/revision on 409 before starting a new action.
- 401 invalid JWT; 403 permission/employee/ownership denied; 404 missing shipment/assignment or disabled; 409 stale assignment/revision, missing acceptance, invalid lifecycle or key conflict; 422 invalid values/non-INTERNAL; 400 malformed/missing input; 413 body limit; 415 media type; 503 Identity outage; 500 storage/outbox failure (transaction rolls back).
- Driver starts only after warehouse handover, then IN_TRANSIT → OUT_FOR_DELIVERY → DELIVERED or DELIVERY_FAILED. Failed delivery can retry OUT_FOR_DELIVERY or RETURNING → RETURNED. No skips/same-state actions/terminal rewrites. Reason is a private employee attestation, not verified image evidence. Shipment/tracking/audit/proof/outbox are atomic; Kafka consumer acceptance requires separate E2E. No COD settlement or automatic refund is asserted.

### Internal driver acceptance (opt-in provider API)

- `POST /api/v2/admin/shipping/shipments/{shipmentId}/accept`: signed `SHIPMENT_EXECUTE#GLOBAL`, live ACTIVE EMPLOYEE and locked assignment ownership required; no role/internal-token bypass or USER_VIEW requirement. Identity grant provisioning is pending.
- Success: 200 receipt; exact idempotent replay returns original receipt. No empty success state. Loading: disable submit while in flight; retry with the same key and payload.
- Auth: 401 invalid/missing JWT; 403 missing permission, inactive employee or different assigned driver. 404 missing shipment/assignment or disabled feature; 409 stale version, lifecycle conflict or reused key; 422 invalid version/non-INTERNAL shipment; 400 malformed/missing input; 413 oversized body; 415 wrong media type; 503 Identity unavailable; 500 persistence failure.
- Uses `SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED` (default false); no Kong/UI connection yet. Acceptance is not pickup/handover and emits no delivery event. Actor overrides are rejected. Reassignment requires fresh acceptance; delivery progress is the separate opt-in operation above.

Updated: 2026-11-07

This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.

## State semantics

- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.
- **Empty:** successful `200` response with an empty collection and zero totals, never `404`.
- **Unauthorized:** `401` when credentials/session/grant are missing, invalid or expired.
- **Forbidden:** `403` when identity is known but capability, permission, ownership or scope is insufficient.
- **Error:** `4xx` is actionable client/business failure; `5xx` is server/dependency failure. All errors use the shared `Error` schema.
- **Success:** only documented `2xx` codes are compatible. Mutations requiring asynchronous processing use `202`.
- Shipping admin reads require the employee grant `SHIPMENT_VIEW#GLOBAL` at the Shipping service. Warehouse/department-scoped shipment reads are not yet supported by the current Shipping read model.
- Company fleet `INTERNAL` and its assignment API are separate opt-in flags, both disabled by default. Assignment provider operations are `GET/POST /api/v2/admin/shipping/shipments/{shipmentId}/assignment` and `GET .../assignment/history`; driver accept/progress operations are opt-in as documented above, while legacy UI workbench remains unconnected. Existing private quote/create/cancel flows reuse shipment persistence/idempotency; COD and print labels are unsupported. Public callback injection is not supported.
- Assignment authorization: signed `SHIPMENT_ASSIGN#GLOBAL` plus live Identity ACTIVE EMPLOYEE validation for every read/write. POST additionally requires `USER_VIEW#GLOBAL` and forwards the employee bearer to existing Identity `GET /api/v1/admin/users/{driverId}` to verify an ACTIVE EMPLOYEE target. Actor comes only from JWT `user_id`/`sub`; no shared-token or ADMIN-role bypass. Identity grant provisioning is a rollout prerequisite, not completed by this Shipping task. Assignment operations remain absent from Kong.
- Assignment states: disabled route 404; success/replay 200 with original snapshot; loading is client-side pending state; unassigned detail 404; empty history 200 with `items: []`, `total: 0`; 400 malformed request/missing version/idempotency key; 401 invalid JWT or rejected credential; 403 missing GLOBAL grant/directory authorization/inactive actor; 404 unknown shipment; 409 stale assignment version, invalid lifecycle, or key reuse; 413 body over 1 MiB; 415 wrong content type; 422 non-INTERNAL shipment, invalid fields, missing/inactive/customer driver, or no-op reassignment; 503 `IDENTITY_UNAVAILABLE` on timeout, redirect, malformed/mismatched Identity data or upstream failure. No write on denied/failed checks; no success fallback to UUID-only eligibility. Existing assignment reads remain available after handover; new assignments do not.
- Shipping accepts the shared Identity permission-code syntax (`[A-Za-z][A-Za-z0-9_.:-]{0,159}`), including unrelated lowercase Traceability grants in an ADMIN JWT. A valid employee token with these grants and exact `SHIPMENT_VIEW#GLOBAL` proceeds to the normal read states; without that exact grant it returns 403, not 401. Malformed grants remain 401; permission matching is case-sensitive.
- Shipping provider v2: `GET /api/v2/admin/shipping/shipments[/{shipmentId}]` is the explicit snake_case backend read model. Kong preserves the full v2 path; Shipping serves aliases with the same JWT middleware and provider handlers as v1. No regex look-around or URI rewrite is required. The existing camelCase UI v1 operations are unchanged and still require their BFF adapter. List supports page/pageSize/status/carrier/orderId; detail includes shipment/tracking_events. States: 200 non-empty/empty arrays, loading, 400 invalid parameters, 401 invalid JWT, 403 missing GLOBAL grant, 404 missing shipment, 500 backend error, 502/503 upstream unavailable. These routes do not expose company-driver workbench or internal commands.
- Shipping provider rollout: Kong exposes only GET shipment list/detail and POST GHN callback; no internal token injection. Identity migration V23 seeds `SHIPMENT_VIEW` for enabled, non-archived ADMIN roles. Sync affected principals (`forceRelogin=false`) and refresh JWTs after deployment. Missing/invalid JWT returns 401; insufficient GLOBAL grant returns 403; empty list returns 200 with `items: []`; unavailable upstream returns 502/503. Existing Web BFF on port 4030 and camelCase UI projections are not yet wired to the snake_case Shipping backend. Workbench remains unimplemented and is not registered at Kong.

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
| `POST /api/v1/admin/catalog/categories` | EPIC_04 | stable | 201 | N/A | 400/— | 401 | 403 | 404 | 409 | — | 500 |
| `GET /api/v1/admin/catalog/products` | EPIC_04 | stable | 200 | 200 (collection empty) | 400/— | 401 | 403 | — | — | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
| `POST /api/v1/admin/catalog/products` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | — | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PUT /api/v1/admin/catalog/products/{productId}` | EPIC_04 | stable (legacy) | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/transition` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/skus` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}/skus/{skuId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `GET /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 200 | 200 (empty list) | 400/— | 401 | 403 | 404 | — | — | 500/503 |
| `PUT /api/v1/admin/catalog/products/{productId}/skus/{skuId}/prices` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `POST /api/v1/admin/catalog/products/{productId}/images` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `PATCH /api/v1/admin/catalog/products/{productId}/images/{imageId}` | EPIC_04 | stable | 200 | N/A | 400/422 | 401 | 403 | 404 | 409 | — | 500/503 |
| `DELETE /api/v1/admin/catalog/products/{productId}/images/{imageId}` | EPIC_04 | stable | 200 | N/A | —/— | 401 | 403 | 404 | — | — | 500/503 |
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

### Giá theo kênh

- `GET .../skus/{skuId}/prices` trả mọi bản ghi giá (ACTIVE, SUPERSEDED, EXPIRED) của SKU.
- `PUT .../skus/{skuId}/prices` supersede giá ACTIVE cùng kênh rồi tạo bản ghi mới với `effectiveFrom = now`.
- Kênh: `D2C_WEB`, `MOBILE_APP`, `B2B_WHOLESALE`, `POS_QUAY`, `SHOPEE`, `TIKTOK`.
- VND dùng `amountUnits`, `amountNanos = 0`; chống chồng lấn khoảng hiệu lực bằng `tstzrange` + `&&`.

### Ảnh sản phẩm

- Lưu `objectKey` (MinIO/S3); response trả thêm `url = https://cdn.omama.vn/ + objectKey` (giữ nguyên nếu objectKey là URL tuyệt đối).
- `role`: `COVER` | `GALLERY` | `THUMBNAIL` (mặc định `GALLERY`).
- `mediaStatus`: `PENDING_SCAN` | `APPROVED` | `REJECTED` (mặc định `PENDING_SCAN`).
- Bất biến: tối đa một ảnh `COVER` + `APPROVED`. Khi thêm/sửa thành cover APPROVED, cover cũ tự hạ xuống `GALLERY`.
- `DELETE` chỉ xóa metadata; object trên storage do job dọn dẹp riêng xử lý.
- PATCH ảnh yêu cầu `expectedVersion` của ảnh.

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
