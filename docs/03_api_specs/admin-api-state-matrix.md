# Internal Employee API State Matrix

Updated: 2026-11-07

This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.

## State semantics

- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.
- **Empty:** successful `200` response with an empty collection and zero totals, never `404`.
- **Unauthorized:** `401` when credentials/session/grant are missing, invalid or expired.
- **Forbidden:** `403` when identity is known but capability, permission, ownership or scope is insufficient.
- **Error:** `4xx` is actionable client/business failure; `5xx` is server/dependency failure. All errors use the shared `Error` schema.
- **Success:** only documented `2xx` codes are compatible. Mutations requiring asynchronous processing use `202`.

Employee APIs require authentication, effective permission and server-side data-scope enforcement. A hidden menu is not authorization.

Procurement receiving handoff (contracted gateway, authenticated BFF and inspection form): receipt inspection
returns 400 INVALID_ARGUMENT for unsupported/inexact physical quantities (minimum 0.01, exact KG-to-gram
or integral G/GRAM/UNIT/PIECE, int32 bound). The transaction must leave receipt/PO/outbox unchanged on
failure. UI must preserve the entered inspection and show the validation error, not report success.
On success, Procurement inspection is recorded, but Inventory stock remains QUARANTINE pending its
own QC/disposition acceptance. Procurement acceptance does not imply sellable stock or finance settlement.

The Order principal-kind gate retains Employee access to direct admin detail and cancellation-request routes; permission and warehouse checks remain mandatory. Guest, payment-grant and Customer principals cannot access the admin detail projection.

## Procurement Web integration — 2026-11-28

Contract version 2.11.0 adds explicit Web `/api/admin/procurement/*` and Identity
`/api/v1/admin/procurement-api/*` facades for the same 37 approved upstream operations.
Both are disabled by default. Web supports OIDC HttpOnly cookies and existing Identity
BFF sessions, not legacy/mock authentication. Browser bearer/role/internal-token headers
are ignored; Identity binds the verified token to an active employee in its DB and relays
that JWT, never a shared internal token. Procurement remains responsible for signed
permission-and-scope checks, receipt warehouse ownership and no-self-approval.

| UI surface | Success / empty | Loading | Error / auth |
| --- | --- | --- | --- |
| `/admin/procurement` supplier/material/PO tabs | Real 20-row offset page; detail on explicit selection. Empty page shows no-data state, not an invented global total. | Pending list/detail states; stale requests aborted when selection changes. | 401 asks for sign-in, 403 shows missing permission/scope, other failures offer reload with requestId. No mock fallback. |
| Receipt lookup | Explicit receipt UUID; no fabricated receipt-list endpoint. | Pending lookup; a newer lookup cancels the old one. | Warehouse access denied remains 403; 404 is not empty success. |
| Supplier/material/offer forms | Create/edit/status/preference and offer-version/toggle forms use real API results. Immutable supplier/material code and base unit are read-only. | Submission disables controls/double-submit. | Client field errors preserve values; backend 400/401/403 remain actionable; 409 requires closing/reloading current data, never automatically overwrites a version. |
| Contract forms | Create a new metadata revision, activate DRAFT or terminate ACTIVE with a reason; MANAGE GLOBAL even for reads. No file URL/upload/signature. | Related contract pagination/detail and command pending states. | Exact updated_at retained; overlap/stale 409 preserves input. No silent activation or changing issued terms. |
| PO workflow forms | Create/draft edit, submit, independent approve/reject, record send/supplier response, new revision, cancellation and close-remainder. Archived revisions are read-only. | Submission pending, no automatic retries; GET current PO after success before showing subsequent controls. | Creator cannot approve/reject own PO. Changing issued terms uses revision. Preserve version/reason; stale/invalid-state 409 needs reload. Send action records an operator action, not an automated email. |
| Receiving/variance forms | Explicit PO-line quantities and optional verified lot dates; inspection requires exact accepted+rejected sum, notes for rejection/over-receipt. Resolve only OPEN variances with the receipt version and note. | Submission locks inputs; inspection does not assume full acceptance. | Signed warehouse scope checked upstream. Procurement acceptance does not release Inventory QUARANTINE or imply financial settlement. |
| Mutation retry and failure | Original upstream result/status, unchanged Idempotency-Key/JSON. Only synchronous 200/201 with a resource UUID completes the client command. | No automatic retry; uncertain attempts retain key+hash in employee/method/path-scoped session storage, not payload or token. | Same-origin checks; absent key 400, non-JSON 415, >64 KiB 413. Unknown outcome/5xx/malformed success keeps the old key; changed payload is blocked until the original command is reconciled. Storage failure is fail-closed before sending. |
| Disabled / unavailable integration | No fabricated empty dataset. | Not applicable when disabled. | Authenticated disabled route 404; absent configuration/timeout 503; invalid/oversized (>1 MiB)/redirect response 502. |

Menu visibility uses the existing employee permission projection only as a UI hint; it
does not replace the signed token snapshot. Backend denials always win. Contract reads
still require PROCUREMENT_MANAGE#GLOBAL, not ordinary VIEW. Receiving-only employees
can look up receipts without being granted supplier or PO list access.

Forecast/replenishment planning is excluded: no Web/Identity/Kong supply-plan route
was added. Legacy private code is not a newly supported product surface. Supplier
contract files remain deferred until a verified immutable-file storage flow is agreed;
no binary upload, arbitrary document URL or unsigned download was introduced.
These tests are not browser/service E2E, deployment approval, Inventory QC release or
finance settlement evidence. All 25 existing write operations now have business forms;
there are no new REST shapes or added business permissions in this UI change.

Forms use exact decimal text (not binary floating point), safe-integer money/version
inputs and at most 100 lines within the 64 KiB facade cap. The browser retry journal
stores only a key/hash, not enough to reconstruct form contents. Keep an uncertain
form open. After reload, re-enter exactly the original inputs/version to retry; if
unavailable, seek operator reconciliation instead of clearing storage or creating again.
Different tabs/devices are not a global dedupe journal. Late responses cannot erase
another command's retry evidence. GET/refetch failure after a committed write disables
subsequent actions and must not prompt a new create. These constraints do not replace
the owning service's durable idempotency or authorization checks.

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

## Procurement service authorization, PO actions and receiving gateway (2.10.0)

Identity migration `V28__seed_procurement_permissions.sql` adds the five permissions
below to existing enabled, non-archived ADMIN roles. It creates no account, role or
scope assignment. Coordinate Shipping V23/V24 and Promotion V27 before applying
V28 to a shared environment; missing earlier seeds must not be bypassed with
Flyway out-of-order or repair. After migration, run PermissionSync for affected
employees (`forceRelogin=false`) and refresh/login: an existing token is unchanged.
During rollout an employee with a stale token still receives 403, not a role-based
bypass. This seed alone does not deploy a gateway route, BFF relay or Web screen.

Kong configuration now allowlists thirty-seven operations in `admin-stable.openapi.yaml`:
six supplier/material/offer GETs, PO create/detail, receipt create/detail and inspection.
PO submit/approve/reject/send/supplier-response/cancel/close are also exposed.
Revision create/history are exposed too. OPTIONS is preflight only.
PO list/draft editing and receipt variance resolution are exposed too. Other
supply-plan paths, health probes and arbitrary suffixes are
not exposed by these routes. Authorization is Bearer
JWT only; the contract's global cookie alternative does not apply to these operations.
Web session relay/UI is not implemented by this change.

| Read flow | Success / empty | Loading | Authentication / scope | Error |
|---|---|---|---|---|
| GET supplier contracts/detail | 200 contract/page, every revision/status; known supplier with none gives items []; limit/offset without total; legacy file reference is unverified metadata, no download capability | Detail/history skeleton; preserve pagination; no cross-page snapshot guarantee | EMPLOYEE with MANAGE GLOBAL even for reads of commercial terms; VIEW-only/warehouse grants and roles insufficient | 400 malformed UUID/page; 401 invalid JWT; 403 missing MANAGE GLOBAL; 404 absent supplier/contract; 500 safe dependency error, never fabricated empty page |
| POST supplier contracts | 201 frozen DRAFT revision; fresh key deliberately creates next revision for same supplier/number; metadata only, document_file_id omitted/empty | Retain key/normalized terms on retry; GET current state after replay | Same MANAGE GLOBAL gate; creator from signed identity, no client actor override | 400 invalid fields/dates/key or unsupported file/link; 401/403 auth; 404 missing supplier; 409 inactive supplier or changed actor/key command; 500 rolls back revision/history/result |
| POST contract activate | 200 frozen ACTIVE result; terms preserved; future start allowed, ACTIVE does not imply effective now; no new maker-checker policy | Retain key/exact contract expected_updated_at; refresh controls after replay | Same MANAGE GLOBAL gate, rechecked on replay | 400 invalid JSON/key/timestamp; 401/403; 404 missing contract; 409 stale state/timestamp, inactive supplier, expired end, overlapping ACTIVE supplier/scope interval across any numbers, or changed command; 500 atomic status/history/result rollback |
| POST contract terminate | 200 frozen TERMINATED result; trimmed reason and actor audited; terms/effective range preserved; no PO/finance/stock effect | Retain key/timestamp/reason; terminal contract cannot reactivate, new terms need new revision | Same MANAGE GLOBAL gate | 400 missing/invalid reason/key/timestamp; 401/403; 404 missing contract; 409 non-ACTIVE/stale/reused key; 500 atomic rollback, safe retry |
| PUT supplier/material detail | 200 frozen committed full edit; preserve status, creation and preference; omitted optional fields clear; codes/base_unit immutable | Retain key/body/exact expected_updated_at including fractions; disable duplicate save; GET current state after replay | EMPLOYEE with MANAGE GLOBAL; current auth required on replay, no role bypass | 400 invalid key/timestamp/body; 401/403 auth; 404 missing record; 409 stale timestamp, immutable field or changed actor/key command, duplicate supplier legal ID; 500 atomic rollback, safe retry |
| PATCH supplier/material status | 200 frozen result; suspension clears all supplier preferences, material inactivity clears its preference; reactivation never restores; affected material timestamps advance | Same retry policy; reload affected materials after cleanup, no optimistic success before response | Same MANAGE GLOBAL gate | 400 invalid state/timestamp/key/JSON; 401/403 auth; 404 absent; 409 stale/reused key; 500 rolls back status, preference cleanup and result |
| PUT material preferred-supplier | 200 frozen material; required supplier_id is nonzero UUID to select or empty string to clear; clearing inactive material allowed | Retain selection/key/material timestamp; refresh current material after replay | Same MANAGE GLOBAL gate | 400 missing/null/invalid supplier_id or timestamp/key; 401/403; 404 missing material/supplier; 409 stale/reused key, inactive parent or no active effective matching-unit price; 500 no partial selection |
| PATCH supplier offer status | 200 frozen offer; price terms/version unchanged; deactivating last eligible price clears matching preference; active does not imply currently effective | Use offer timestamp, not price version or parent timestamp; refresh current versions/material after completion | Same MANAGE GLOBAL gate | 400 missing active/key/timestamp; 401/403; 404 absent parent/offer; 409 stale/reused key, inactive parent/unit mismatch or overlap when enabling; 500 rolls back toggle/cleanup/result |
| POST suppliers/materials | 201 original committed resource, initially ACTIVE; matching retry returns frozen result even after later changes; GET current detail | Retain entered fields and same key on retry; disable duplicate submit | Signed EMPLOYEE with MANAGE GLOBAL; VIEW/warehouse grants and role alone do not authorize; auth rechecked on replay | 400 missing/invalid key, JSON or fields; 401 invalid JWT; 403 missing grant; 409 duplicate code/legal identifier or actor/key reused for another kind/payload; 500 dependency failure, no partial resource or consumed key |
| POST suppliers/{id}/offers | 201 immutable create result with version/effective range; closes earlier open-ended active version atomically; existing PO snapshots unchanged | Retain key and explicit start date; reload versions after replay | Same MANAGE GLOBAL gate | 400 invalid UUID/key/body/date/price/quantity; 401/403 auth; 404 missing parent; 409 inactive parent, unit mismatch, overlapping active range or changed key payload; 500 rollback includes previous window closing |
| GET suppliers/materials | 200 page with items, limit, offset; empty items array, never null; no global total is claimed | Retain filters; disable duplicate pagination requests | 401 invalid/missing JWT; 403 known non-employee or missing PROCUREMENT_VIEW#GLOBAL, including warehouse-only grants | 400 invalid limit/offset; 500 safe dependency error, not a fabricated empty result |
| GET supplier/material detail | 200 domain record; omitted optional fields remain absent | Detail skeleton | Same JWT/GLOBAL gate; ADMIN role alone is insufficient | 400 malformed UUID; 404 unknown record; 500 dependency failure |
| GET supplier/material offers | 200 items array; known parent without offers returns []; all price versions included, no pagination/effective filter | Price comparison pending | Same JWT/GLOBAL gate | 400 malformed UUID; 404 unknown parent; 500 dependency failure |
| POST purchase-orders | 201 new DRAFT or existing current PO on same creator/key/payload; no automatic approval/send | Disable duplicate submit; retain Idempotency-Key across retry | MANAGE GLOBAL; signed employee supplies creator | 400 invalid JSON/key/fields; 404 missing references; 409 changed payload or lifecycle/offer conflict; 500 dependency failure |
| GET purchase-orders/{id} | 200 current lifecycle and price snapshots for the current revision | Detail pending | VIEW GLOBAL | 400 invalid UUID; 404 missing PO; 500 dependency failure |
| GET purchase-orders | 200 items/limit/offset; empty array for no matches or unknown supplier; created_at DESC then UUID ASC; no global total | Retain supplier/status filters and pagination; no cross-page frozen snapshot | VIEW GLOBAL, not warehouse-only VIEW | 400 invalid/nonzero supplier UUID, status, limit or offset; 401/403 auth; 500 dependency failure, never fabricated empty data |
| PUT purchase-orders/{id} | 200 edited DRAFT or frozen committed edit; preserves revision/creator/plan linkage; no archive or approval | Retain Idempotency-Key/expected_version/full replacement body; GET current detail after replay | MANAGE GLOBAL; original creator retained, creator-only maker-checker unchanged | 400 invalid key/version/body/offer quantity; 401/403 auth; 404 missing PO/reference; 409 non-draft, stale version or key conflict; 500 failure rolls back PO/lines/history/result |
| POST purchase-orders/{id}/submit | 200 PENDING_APPROVAL or original committed action snapshot on matching replay | Retain key and expected_version; disable duplicate action | MANAGE GLOBAL | 400 invalid key/body/version; 401 invalid JWT; 403 missing grant; 404 missing PO; 409 stale version/state or reused key with changed command; 500 dependency failure |
| POST purchase-orders/{id}/approve | 200 APPROVED or original committed action snapshot | Retain key/version; refresh current PO after completion | APPROVE GLOBAL; creator cannot approve | Same errors as submit; self-approval is 409 |
| POST purchase-orders/{id}/reject | 200 REJECTED or original committed action snapshot | Retain key/version and reason | APPROVE GLOBAL; creator cannot reject | Same errors as submit; blank reason is 400; self-rejection is 409 |
| POST purchase-orders/{id}/send | 200 SENT or original committed action snapshot; no email/supplier transport is invoked | Retain key/version; do not create a new key on network retry | MANAGE GLOBAL | Same errors as submit; fresh key on already SENT PO is 409 |
| POST purchase-orders/{id}/supplier-response | 200 ACKNOWLEDGED, CHANGE_REQUESTED or SUPPLIER_DECLINED; matching retry returns frozen command result | Retain key, version, response and note; GET current state after replay | MANAGE GLOBAL; employee records supplier evidence, not supplier authentication | 400 missing key/version, invalid response or missing required note; 401/403 auth; 404 missing PO; 409 stale state/version or changed command; 500 dependency failure |
| POST purchase-orders/{id}/cancel | 200 CANCELLED with audited reason; frozen on replay | Retain key/version/reason | MANAGE GLOBAL | 400 invalid body/key/reason; 401/403 auth; 404 missing PO; 409 sent/terminal state, stale version or key conflict; 500 dependency failure |
| POST purchase-orders/{id}/close | 200 CLOSED with received and accepted quantities preserved; no refund or Inventory reversal | Retain key/version/reason; resolve open variances before retrying | MANAGE GLOBAL | Same errors as cancel; open receipt variance is 409; new receipts/inspections after closure fail |
| GET purchase-orders/{id}/revisions | 200 archived revisions descending, with limit/offset; empty items array for known PO without archives; live revision excluded | History skeleton; retain pagination; no fabricated total | VIEW GLOBAL; warehouse-only VIEW is insufficient | 400 invalid UUID/limit/offset; 401 invalid JWT; 403 insufficient grant; 404 unknown PO; 500 dependency failure, never fabricated empty history |
| POST purchase-orders/{id}/revisions | 200 new DRAFT or frozen committed revision result; old issued terms remain archived | Retain key/version/reason/lines; refresh current detail after replay; require approval again | MANAGE GLOBAL; signed reviser becomes creator and cannot approve/reject own new draft | 400 invalid key/version/body/reason or changed supplier; 401/403 auth; 404 missing PO/reference; 409 stale state/version, any receipt, or changed command; 500 DB/dependency failure with no partial archive or consumed key |
| POST goods-receipts | 201 draft/current receipt for same receiver/key/payload; draft has no Inventory effect | Retain key and user-entered data on retry | RECEIVE GLOBAL or matching request warehouse | 400 invalid JSON/key/fields; 403 foreign warehouse; 404 missing PO; 409 changed payload/state conflict; 500 dependency failure |
| GET goods-receipts/{id} | 200 snapshot, including inspection/variance evidence | Detail pending | VIEW GLOBAL or RECEIVE GLOBAL/matching stored warehouse; warehouse VIEW never bypasses | 400 invalid UUID; 403 foreign receipt; 404 unknown receipt; 500 dependency failure |
| POST goods-receipts/{id}/inspect | 200 inspection/current receipt; identical retry does not update PO/history/outbox again; physical stock remains Inventory QUARANTINE | Retain same key/payload; disable duplicate action | RECEIVE GLOBAL or matching stored warehouse; signed employee is inspector | 400 invalid key/JSON/quantity; 403 wrong warehouse; 404 missing receipt; 409 changed key/actor/payload or historical inspection without evidence; 500 dependency failure, safe retry with unchanged key |
| POST goods-receipts/{id}/variances/{varianceID}/resolve | 200 frozen committed receipt result; records decision only, no quantity/Inventory/finance effect | Retain key, receipt expected_version, decision and note; GET current detail after replay | VARIANCE_RESOLVE GLOBAL or matching stored receipt warehouse, rechecked for replay; separately needs existing VIEW/RECEIVE grant to load receipt detail | 400 invalid key/version/body/decision/note or uninspected receipt; 401 invalid JWT; 403 wrong permission/warehouse; 404 receipt/variance absent; 409 stale version, already resolved without matching evidence, changed command or terminal PO; 500 dependency failure with atomic rollback |

Receipt inspection requires schema migration 000014 before deploying the new binary.
Idempotency evidence is stored in the same PostgreSQL transaction as receipt, PO,
history and outbox. Normalization trims inspection strings and sorts lines by ID;
decimal text remains significant. Evidence has no cache TTL. Replays return current
resource state, not an immutable historical HTTP response. No key is consumed by
validation failures or rolled-back transactions. Private inspection callers must
also supply the newly required header.

The four PO actions require migration 000015, Idempotency-Key and expected_version.
Reject additionally requires a nonblank reason; other actions do not accept reason.
Unlike create/inspect replay, PO action replay returns a frozen committed snapshot,
not current PO state; GET the PO before displaying current workflow controls.
Signed actor/key is shared across all nine PO action endpoints and binds resource, action,
expected version and trimmed reason. Changed command returns 409. Evidence and
PO/history commit atomically, with no TTL; failures do not consume the key.
Current JWT and operation permissions are required even for replay. Existing
private HTTP clients must adopt the new header/body requirements.

Supplier-response/cancel/close additionally require migration 000016, preserving
all 000015 evidence. Response body uses expected_version, response and optional
note (nonblank for CHANGE_REQUESTED/DECLINED); cancel/close use expected_version
and nonblank reason. Trimmed note and the exact response are bound to the key.
Closure rechecks open variances in the PO transaction; prior quantities are never
zeroed. These operations do not communicate with the supplier or reverse payments.
Revision create/history additionally require migration 000017 after 000015/000016.
Revision fingerprint includes supplier/date/trimmed notes/reason and ordered
lines; quantity text and line order remain significant. Replay reads the original
committed DRAFT before consulting current offers, even after later revisions.
Archive, PO/lines, history and command result commit together; failed commands
consume no revision number/key. Any receipt blocks revision, even a draft receipt.
Archived snapshots are protected against ordinary UPDATE/DELETE/TRUNCATE; database
owners can still deliberately disable those guards. Rollback refuses to discard
completed revision evidence. The read page contains only prior archived revisions,
newest-first, limit 1..100 and nonnegative offset; it is not the live current PO.

Draft PUT requires migration 000018 and uses EDIT_DRAFT in the same journal. Same
request replays before consulting offers even after submit; it cannot create a
new revision or silently overwrite a competing edit. Supplier/lines/notes/date
are full replacements; omitted optional values clear them. Revision and original
creator/supply-plan linkage are preserved. No new editor-exclusion approval policy
is introduced. Existing private PUT callers must add key/version. Previously lost
revision numbering is not reconstructed; no live data repair is performed.

Variance resolution requires migration 000019, mandatory key/receipt version,
normalized decision and note. Its signed actor/key namespace is separate from PO,
receipt creation and inspection. Matching replay returns the original frozen
receipt snapshot even after later decisions/closure; permissions remain mandatory.
Receipt/PO/history/result commit atomically and evidence has no TTL. Receipt -> PO
locking serializes decisions across receipts so the last resolution sees all
others. PO completion derives from accepted quantities without changing any
quantity or emitting another receipt event. No Inventory QC release or refund is
performed. No fresh resolution may reopen a closed PO. Rollback cannot discard
completed decisions; historical items without evidence return 409 rather than
fabricating replay results. Physical disposition remains separate.

All Procurement HTTP errors, including private operations, now use the shared flat
`Error` envelope (`code`, safe `message`, `requestId`), replacing the unpublished
legacy `{error:{code,message}}` shape. Internal consumers must migrate. Admin
responses are `Cache-Control: no-store`. A bounded safe `X-Request-Id` is preserved
for gateway correlation; otherwise a new UUID is generated. It never grants access.

Procurement's `/api/v1/admin/procurement/*` endpoints accept an Identity-issued
RS256 bearer JWT and require `principal_type=EMPLOYEE`, `user_id`, and the signed
`permission_grants` claim. Procurement no longer treats internal-token or user
role headers as user authentication. Its BFF/Gateway must forward the bearer
token and keep direct upstream access private. The service audience
is configurable; align it with Identity (currently expected to be `omama-api`).

| Procurement route family | Required grant | Scope/result |
|---|---|---|
| Supplier, material, offer, PO, and supply-plan reads; supply-plan preview | `PROCUREMENT_VIEW` | `GLOBAL` |
| Supplier, material, offer, contract reads and changes; PO and supply-plan changes excluding approvals | `PROCUREMENT_MANAGE` | `GLOBAL` |
| PO and supply-plan approve/reject actions | `PROCUREMENT_APPROVE` | `GLOBAL`; creator cannot approve their own submission |
| Goods receipt create/inspect; receipt read for warehouse staff | `PROCUREMENT_RECEIVE` | Matching `WAREHOUSE#<warehouseCode>`; operational admins may receive `GLOBAL` |
| Goods receipt variance resolution | `PROCUREMENT_VARIANCE_RESOLVE` | Matching `WAREHOUSE#<warehouseCode>`; operational admins may receive `GLOBAL` |

Missing/invalid credentials produce 401. A valid employee token missing the
required permission or matching warehouse scope produces 403. These operations
other than the thirty-seven operations above remain absent from frontend OpenAPI paths
and the Procurement Kong allowlist. The policy summary for those private operations
is retained in the draft OpenAPI metadata. A gateway read route does not create a
Web session adapter or make private mutations a stable frontend contract.
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
