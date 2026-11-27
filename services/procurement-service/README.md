# Procurement Service (MS-08)

Procurement Service owns supplier records, raw material definitions, supplier-specific offers, Purchase Order approval, and internal goods-receipt/QC records. Goods receipt inspection updates PO receipt quantities, but does not mutate Inventory or trigger finance effects.

## Run locally

### Web workspace and session adapter (default-off)

The new work does **not** include forecasting/replenishment planning. The abandoned
read-plan projection was removed; legacy private supply-plan code is not exposed by
Kong or the new Web/Identity adapter.

`apps/web-user` now has `/admin/procurement` for supplier/material/PO pagination and
detail, explicit receipt/contract-ID lookup, and business forms for all 25 approved
write operations. It uses real APIs and existing employee login; there is no port-4030
mock, raw JSON request editor or browser-visible shared token.

The workspace supports supplier/material create/edit/status, supplier preferences,
offer version creation/toggling, contract metadata revisions/activation/termination,
PO create/draft edit/submit/approve/reject/send/supplier response/revise/cancel/close,
receipt creation/inspection and open-variance resolution. Related offer lists,
contract history and immutable PO revision snapshots are read from real endpoints.
Independent permissions remain independent; MANAGE/APPROVE do not silently grant VIEW.

Forms preserve exact decimal text and updated_at precision, use current resource
versions, restrict safe integer money input and allow at most 100 lines within the
64 KiB request cap. Receipt/lot dates and inspected quantities are entered explicitly,
never invented. No-self-approval and workflow hints mirror backend guards, which remain
authoritative for signed scope, policy and transactional validation.

Browser retries persist only an idempotency key and SHA-256 payload digest in session
storage, namespaced by employee/method/path. No tokens or business payloads are stored.
An uncertain response locks unchanged retry to the same key; changed data cannot
silently mint a new key. Keep the form until resolved. After reload, re-enter the exact
original data to retry; if it cannot be reconstructed (including version fields),
obtain operator reconciliation instead of clearing browser storage or creating again.
Separate tabs/devices are not one shared retry journal. Successful writes are followed
by a fresh GET before enabling subsequent resource actions; frozen replay results
are never treated as current state. Conflicts preserve input and require refresh.

Validation: Web `npm test`, `npm run typecheck`, and ESLint; from the repository root,
`python scripts/verify-procurement-admin-contract.py --web-forms` checks all 25 built
mutation requests against shared schemas as well as the existing HTTP/JWT fixtures.
These checks do not establish browser/live-service or deployment acceptance.

- Web: `ADMIN_PROCUREMENT_ENABLED=true` enables the page and BFF. Keep false until configured.
- `WEB_AUTH_MODE=identity-bff`: Web calls `IDENTITY_UPSTREAM_URL` (ending `/api/v1`)
  with only `OMAMA_SESSION`; Identity resolves it and forwards its verified JWT.
- Identity: set `IDENTITY_PROCUREMENT_API_ENABLED=true` and origin-only
  `IDENTITY_PROCUREMENT_API_URL` to Procurement or an approved gateway exposing
  `/api/v1/admin/procurement/*`. Do not include `/api` in this URL.
- `WEB_AUTH_MODE=oidc`: set Web's server-only `PROCUREMENT_ADMIN_UPSTREAM_URL`
  (ending `/api/v1`). The trusted access-token cookie is forwarded server-side only.
- Prefer HTTPS. Trusted private/local HTTP requires explicit opt-in via
  `PROCUREMENT_ADMIN_ALLOW_INSECURE_HTTP=true` (Web) and/or
  `IDENTITY_PROCUREMENT_API_ALLOW_INSECURE_HTTP=true` (Identity).
- Set `WEB_PUBLIC_URL` and Identity `FRONTEND_BASE_URL` to the same actual Web origin
  for cookie mutation CSRF checks. Do not use browser-supplied roles to authorize.
- Existing V28 seed and PermissionSync/refresh are still required for actual grants;
  no live IAM/Keycloak configuration was changed in this task.

Mutations preserve required Idempotency-Key and JSON body, do not retry automatically,
and reject cross-origin requests. Requests are capped at 64 KiB and JSON responses at
1 MiB. Errors, version conflicts and warehouse denials are not converted to empty pages.
Contracts/state details are in the shared Admin stable 2.11.0 specification and matrix.
No contract-file upload was added: verified immutable storage is still required first.

### Backend local startup

```powershell
Set-Location services/procurement-service
$env:PROCUREMENT_OIDC_ISSUER = "https://keycloak.omama.vn/realms/omama"
$env:PROCUREMENT_OIDC_AUDIENCE = "omama-api"
docker compose up --build
```

The HTTP API listens on `localhost:8008`; PostgreSQL is exposed on `localhost:5449`. The service applies pending migrations at startup. `GET /health/live` reports process liveness and `GET /health/ready` checks the database.

All `/api/v1/admin/procurement/*` routes validate an Identity-issued RS256 bearer access token. Configure `PROCUREMENT_OIDC_ISSUER` and `PROCUREMENT_OIDC_AUDIENCE`; `PROCUREMENT_OIDC_JWKS_URL` is optional and defaults to the Keycloak certs endpoint. HTTP is rejected except for an explicitly enabled loopback-only local test (`PROCUREMENT_OIDC_ALLOW_LOCAL_HTTP=true`). The token must contain `principal_type=EMPLOYEE`, `user_id`, and the signed `permission_grants` array. The service does not trust `X-User-ID`, `X-User-Role`, or `X-Internal-Token` as user authentication. A BFF/Gateway must forward the user's bearer token, keep direct upstream access private and expose only explicitly allowlisted paths.

The compact authorization vocabulary is:

- `PROCUREMENT_VIEW#GLOBAL`: read supplier/material/offer/PO/plan data and preview supply plans.
- `PROCUREMENT_MANAGE#GLOBAL`: manage suppliers, materials, offers, contracts (including contract reads), purchase orders, and supply plans, excluding approval actions.
- `PROCUREMENT_APPROVE#GLOBAL`: approve or reject purchase orders and supply plans. Existing no-self-approval checks remain in force.
- `PROCUREMENT_RECEIVE#WAREHOUSE#<warehouseCode>`: create and inspect receipts, and read receipts for that warehouse. A GLOBAL grant is accepted for operational admins.
- `PROCUREMENT_VARIANCE_RESOLVE#WAREHOUSE#<warehouseCode>`: resolve receipt variances for that warehouse. A GLOBAL grant is accepted for operational admins.

Grant format follows Identity's signed snapshot convention: `PERMISSION#GLOBAL` or `PERMISSION#WAREHOUSE#<scopeId>`. Realm roles are not used as authorization bypasses. Missing/invalid JWT returns 401; a valid employee token without the required permission or warehouse scope returns 403.

### Identity permission rollout

`services/identity-service/src/main/resources/db/migration/V28__seed_procurement_permissions.sql`
adds the five permissions above to existing enabled, non-archived ADMIN roles.
It does not create principals, roles or GLOBAL assignments, or overwrite existing
permission metadata. Non-admin staff need explicit role/permission assignments
through the existing IAM workflow. Procurement still checks permissions and scope,
not the ADMIN role name.

1. Coordinate the complete Identity migration chain before shared deployment:
   Shipping owns V23/V24, Promotion owns V27, Procurement owns V28. Check the
   deployed Flyway history first; do not deploy V28 ahead of pending lower seeds,
   edit applied migrations, or use repair/out-of-order as a shortcut.
2. Apply migrations through the normal Identity deployment, then reconcile each
   affected employee using `POST /api/v1/admin/iam/principals/{principalId}/sync-permissions?forceRelogin=false`
   with an authorized IAM operator. Verify effective permissions; handle 502 sync
   failures before declaring activation complete.
3. Refresh/login for a new signed token. Confirm EMPLOYEE, the exact configured
   issuer, `omama-api` audience, and the intended grants. Token claims cannot be
   changed by SQL. Receiving/variance staff use matching warehouse grants; a
   warehouse ADMIN assignment does not automatically gain GLOBAL scope.
4. Smoke-test missing token (401), insufficient/wrong-warehouse grant (403),
   authorized reads/actions, and maker-checker denial using real Identity tokens.
   Unit and isolated PostgreSQL tests do not replace this deployment acceptance.

The seed alone does not activate deployed access. The gateway configuration
below still needs deployment; Identity BFF relay and Web UI remain separate work.

Seed regression coverage lives in Identity's `ProcurementPermissionMigrationTest`
(actual PostgreSQL DDL and transactional rollback) and `PermissionSyncServiceTest`
(scoped Keycloak projection). Set `IDENTITY_TEST_JDBC_URL` to an **isolated test**
PostgreSQL JDBC URL, including test credentials, before running Identity's Gradle
test suite. The two database migration cases are explicitly skipped when that
variable is absent; a unit-only green result is not migration execution evidence.

### Kong integration: management reads, master creates, PO actions and receiving

The shared `admin-stable.openapi.yaml` version 2.11.0 documents thirty-seven upstream operations plus their Web/Identity facades:
six supplier/material/offer GETs, PO create/detail, receipt create/detail and receipt
inspection/variance resolution, PO list/create/detail/draft edit, lifecycle actions and revision create/history. Kong routes the exact contracted GET/POST/PUT/PATCH/OPTIONS path shapes to
`http://procurement-service:8008`, without stripping the path, retrying business
requests or injecting credentials. Procurement verifies the employee bearer JWT
and operation permission/scope; the gateway does
not exchange a cookie for a token. The new default-off Web/Identity adapter performs
that session-to-JWT relay as described above; it must be explicitly configured.

Supplier/material creation and supplier price-version creation are also exposed.
Supplier/material full edits, status changes, offer toggles and preferred supplier selection are also exposed.
Supplier contract list/detail/create/activate/terminate are also exposed (metadata only).
Supply-plan APIs are **not** exposed. Their legacy implementation remains private;
forecasting/replenishment planning is excluded from the current scope.
The Web workspace now supplies the approval/dispatch forms as well, but live-service
and browser end-to-end acceptance remain separate from this implementation evidence.

Master creates require migration `000020_master_data_create_results`, signed
EMPLOYEE with `PROCUREMENT_MANAGE#GLOBAL`, and a 16-128 visible ASCII
`Idempotency-Key`. Actor/key is shared across supplier/material/offer creates,
separate from PO/receiving keys. Matching normalized payload returns the frozen
original 201 result; changed kind, parent or payload returns 409. Decimal quantity
text remains significant. Results have no TTL; resource/window/result writes
commit together, and failures consume no key. GET current state after replay.
Offer start date is mandatory (YYYY-MM-DD, UTC midnight); optional end date is
inclusive and stored as next-day exclusive midnight. Both parents must be active,
unit must match material base unit, and active price ranges cannot overlap.
Creating a new version closes an earlier active open-ended range atomically.
Existing PO price snapshots are unchanged. Master creation does not provision
Inventory SKU/unit mappings, emit receiving events or release quarantine.

Master mutations require `000021_master_data_mutation_results`, the same MANAGE
GLOBAL employee gate, an Idempotency-Key and `expected_updated_at` copied exactly
from the current supplier/material/offer, preserving fractional seconds. These six
actions share an actor/key namespace, separate from create/PO/receiving keys.
Matching replay returns the original frozen 200 snapshot; changed action/resource/
parent/timestamp/payload is 409. Stale observations are 409. Always GET current
state after replay; do not use a historical create/edit result as the current token.
Every update, including no-op or implicit cleanup, advances updated_at at least
one microsecond, even if the clock moves backward. Supplier/material codes and
material base_unit are immutable; full edits preserve status/creation/preference.
Omitted optional edit fields are cleared. Offer toggles preserve price terms/version.
Enabling checks active parents, matching base unit and non-overlapping active
ranges; active future/expired prices need not be effective now. Setting a preference
requires active parents and an active effective price in the material's base unit.
`supplier_id` is required; an empty string clears it, including on inactive material.
Suspending a supplier clears all its preferences; inactivating a material clears
its preference. Disabling the last eligible price clears that pair's preference.
Reactivation never restores preferences. Cleanup invalidates affected material
timestamps and commits atomically with the mutation and actor/action/result journal.
Low-volume master mutation commands and price creates serialize before row locks.
No TTL/pruning; failed transactions consume no keys. Existing PO/receipt snapshots,
Inventory stock/events, QC and finance are unchanged. Private HTTP callers of the
six actions must also adopt the new header/timestamp contract before rollout.

Collections return `items: []` when empty. Supplier/material pages contain `limit`
and `offset`, not a fabricated global total. Offers include all versions (including
inactive/future/expired); clients must not equate `active` with currently effective.
All service HTTP errors now use the shared flat `{code,message,requestId}` envelope,
including private routes; migrate any caller still reading `error.code`. Admin
responses use `Cache-Control: no-store`; the response request ID is also returned
in `X-Request-Id`. It is diagnostic only, not trusted user identity.

The base Compose publishes PostgreSQL and HTTP on loopback only. To connect Kong
without putting the database on its shared network:

```powershell
# Use the existing network shared with Kong; Coolify may use a different name.
$env:PROCUREMENT_GATEWAY_NETWORK = "omamx-network"
docker compose -f docker-compose.yml -f docker-compose.gateway.yml config
docker compose -f docker-compose.yml -f docker-compose.gateway.yml up -d --build
```

Set the exact realm issuer and real JWKS endpoint (HTTPS), configure audience
`omama-api`, and ensure the gateway and service share the selected network. Do not
publish the database or use a browser-visible internal token. Loading the updated
Kong configuration and PermissionSync/token refresh are explicit rollout steps;
none of these commands has been run against a shared/live deployment here.

From the repository root, run `python scripts/verify-procurement-admin-contract.py`
with the MVP Python dependencies. It validates actual Go HTTP/JWT response fixtures,
all thirty-seven operation examples, denied methods/paths and network wiring. Separately
validate the real Kong 3.9.3 parser with:

```powershell
docker run --rm -e KONG_DATABASE=off -v "${PWD}/services/api-gateway/config/kong.yml:/config/kong.yml:ro" kong:3.9.3 kong config parse /config/kong.yml
```

For real router-selection coverage, CI also starts a disposable Kong with debug
headers enabled **only in that test**, admin listener off, and the Procurement
hostname mapped to the container's loopback (no application or live upstream).
`verify-procurement-admin-contract.py --gateway-url http://127.0.0.1:<test-port>`
checks the selected route for thirty-seven operations, 167 rejected method combinations and eleven
non-allowlisted paths. Expected upstream connection failures are not service
failures in this routing-only test. Never enable Kong debug headers on the live
gateway for this purpose. See `.github/workflows/contracts.yml` for the test setup.

These are local contract/routing checks, not acceptance with live Identity tokens,
deployed DNS, TLS or a Web session. Perform authorized/401/403 smoke checks after deployment.

## Supplier and material API

- `GET /api/v1/admin/procurement/suppliers?limit=50&offset=0`
- `POST /api/v1/admin/procurement/suppliers`
- `GET /api/v1/admin/procurement/suppliers/{supplierID}`
- `PUT /api/v1/admin/procurement/suppliers/{supplierID}`
- `PATCH /api/v1/admin/procurement/suppliers/{supplierID}/status`
- Create/update accepts optional `legal_identifier`; duplicate non-empty identifiers are rejected case-insensitively. Suppliers are suspended rather than hard-deleted so PO/receipt history remains valid.
- `GET /api/v1/admin/procurement/materials?limit=50&offset=0`
- `POST /api/v1/admin/procurement/materials`
- `GET /api/v1/admin/procurement/materials/{materialID}`
- `GET /api/v1/admin/procurement/materials/{materialID}/offers` compares supplier price versions for a material.
- `PUT /api/v1/admin/procurement/materials/{materialID}` updates code/name/unit/description while preserving status and preferred supplier.
- `PATCH /api/v1/admin/procurement/materials/{materialID}/status`
- `PUT /api/v1/admin/procurement/materials/{materialID}/preferred-supplier` accepts `{"supplier_id":"<uuid>"}`; an empty value clears the preference. The supplier must be active and have an effective offer for the material.
- `GET /api/v1/admin/procurement/suppliers/{supplierID}/offers`
- `POST /api/v1/admin/procurement/suppliers/{supplierID}/offers`
- `PATCH /api/v1/admin/procurement/suppliers/{supplierID}/offers/{offerID}/status` activates/deactivates a price version.
- `GET/POST /api/v1/admin/procurement/suppliers/{supplierID}/contracts`
- `GET /api/v1/admin/procurement/contracts/{contractID}`
- `POST /api/v1/admin/procurement/contracts/{contractID}/activate` and `/terminate`

Supplier types: `FARMER`, `COOPERATIVE`, `OCOP_BUSINESS`, `COMPANY`. Contracts use immutable revisions, effective dates, scope and payment terms. All contract operations, including reads of commercial terms, require verified EMPLOYEE and `PROCUREMENT_MANAGE#GLOBAL`, not a role name or VIEW-only grant. New public contract creation is metadata-only: `document_file_id` must be omitted/empty until verified attachment storage and download authorization exist; no URL or binary input. Legacy opaque references may be read but confer no verification or download authority. Activation rejects overlapping active agreements for the same supplier and case-insensitive scope. Offers are immutable price versions with effective dates, MOQ and lead time; overlapping ranges are rejected and a new version closes a previous open-ended version. A PO can only use an effective active offer for an active supplier/material, must meet the offer's minimum quantity, and snapshots the offer price and material name/code. All lines in one PO must use the same currency.

### Supplier contract commands (migration 000022)

Apply `000022_supplier_contract_commands` after 000020/000021 before the new binary.
The five gateway operations are GET/POST `suppliers/{supplierID}/contracts`, GET
`contracts/{contractID}` and POST `contracts/{contractID}/{activate,terminate}`.
Create/activate/terminate share an actor-scoped Idempotency-Key namespace, separate
from master/PO/receiving keys. Matching normalized retry returns frozen 201/200,
changed action/resource/parent/terms/timestamp/reason returns 409. GET current state
after replay. Failures do not consume keys; current MANAGE GLOBAL JWT is mandatory.
Create allocates the next DRAFT revision for the trimmed, case-sensitive supplier/
contract_number; a new key deliberately creates another revision, never edits terms.
Scope normalizes trim/uppercase. Dates use UTC midnight, optional inclusive end
becomes next-day exclusive. Supplier must be active. Activate requires the exact
`expected_updated_at`, DRAFT state, active supplier, non-expired end and no ACTIVE
range overlap for the supplier/scope, even across different contract numbers.
Future start is allowed, so ACTIVE alone does not mean effective now. This retains
MANAGE-based activation without adding a separate maker-checker approval policy.
Terminate requires current timestamp, ACTIVE state and a nonblank reason, audited
with the signed actor. It preserves terms/range; terminal contracts cannot reactivate.
Status changes advance updated_at monotonically. DB guards reject terms edits,
deletion and truncation; create a new revision to change terms. Contract/history/
actor/action/reason/result commit together, without TTL or independent pruning.
Rollback refuses completed commands or reason evidence. List includes all statuses/
revisions with limit/offset, contract_number ASC then revision DESC; no global total
or frozen cross-page view. Missing supplier is 404; known empty supplier has [].
No automatic EXPIRED worker, legal signing/email, file downloads, price/PO mutation,
Inventory receiving event or finance settlement is introduced. Private HTTP callers
must adopt the new key/timestamp/reason and metadata-only input requirements.

## Purchase Order workflow

- `GET /api/v1/admin/procurement/purchase-orders?limit=50&offset=0&supplier_id={uuid}&status=DRAFT` requires VIEW GLOBAL; combines optional supplier and status filters with AND, orders by created_at descending then UUID ascending, and returns items/limit/offset with no fabricated total. Unknown suppliers/no matches yield an empty array; explicit zero UUID is invalid. Status is case-insensitive, limit 1..100 and offset nonnegative. Pagination is not a frozen cross-page snapshot.
- `POST /api/v1/admin/procurement/purchase-orders` creates a `DRAFT` from active supplier offers and requires an `Idempotency-Key` header (16–128 visible ASCII characters). The key is scoped to the authenticated user; retrying the same key and payload returns the original PO, while reusing it with a changed payload returns `409 CONFLICT`.
- `GET /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}` returns the PO, price snapshots, version, and current lifecycle metadata.
- `PUT /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}` requires MANAGE GLOBAL, Idempotency-Key and expected_version; fully replaces supplier/lines/notes/delivery date. Only drafts can be edited; omitted notes/date clear them. Preserves ID/number/creator/revision/supply-plan linkage and does not create an archive revision or approve/send the PO.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/submit` submits for approval; requires MANAGE GLOBAL.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/approve` approves a pending PO; requires APPROVE GLOBAL. The creator cannot approve their own PO.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/reject` rejects a pending PO; requires APPROVE GLOBAL and a nonblank `reason`. The creator cannot reject their own PO.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/send` records that an approved PO was sent; requires MANAGE GLOBAL. It does not deliver email or invoke supplier transport.

All four actions require `Idempotency-Key` (16–128 visible ASCII characters) and
`{"expected_version":1}` using the version returned by the latest PO read. Reject
also requires `reason`; other actions reject a reason or unknown body fields.
Existing private HTTP clients must adopt these requirements. Apply migration
`000015_purchase_order_action_idempotency` before deploying this binary.

The signed employee and key identify one command across all nine PO action endpoints. Its
hash binds PO ID, action, expected version and trimmed reason. Matching retries
return the immutable committed PO snapshot, even after later actions/revisions;
use GET for current state. Changed payload/resource/action with that key returns
409. A new key with a stale version or invalid state also returns 409, including
SEND on an already SENT PO. JWT and action permission checks still apply to replay.
PO changes, history and result evidence commit atomically; validation or rollback
does not consume the key. Evidence has no TTL and must not be independently pruned.
The down migration removes replay evidence, not business history; stop these
endpoints before rollback and do not claim replay guarantees after removing it.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/supplier-response` lets an employee with MANAGE GLOBAL record a supplier response: `ACKNOWLEDGED`, `CHANGE_REQUESTED`, or `DECLINED`. Change requests and declines require a nonblank note. This is not supplier login or automatic supplier notification.
- `GET /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/revisions` requires VIEW GLOBAL; lists prior archived revisions newest-first with limit 1..100 and nonnegative offset. The live revision is not included; a known PO without archives returns `items: []`, limit and offset, without a fabricated total.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/revisions` requires MANAGE GLOBAL and creates the next draft after `CHANGE_REQUESTED`; requires Idempotency-Key, expected_version, reason, supplier_id and revised lines. The supplier cannot change, and any existing receipt blocks revision. The old revision is archived before the PO is updated, in one transaction.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/cancel` cancels an eligible PO and requires a reason.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/close` closes the outstanding quantity for a `SENT`, `ACKNOWLEDGED`, or `PARTIALLY_RECEIVED` PO and requires a reason. Quantities already received remain recorded; further receipts are rejected.

Supplier response/cancel/close require MANAGE GLOBAL, Idempotency-Key and
expected_version, with the same frozen-result replay policy as submit/send.
Apply additive migration `000016_purchase_order_terminal_actions` after 000015.
Supplier response accepts `{expected_version, response, note?}`; cancel/close
accept `{expected_version, reason}`. The key binds the specific response and
trimmed note/reason; changing either conflicts. Existing private HTTP callers
must adopt the new fields/header. Cancellation only applies before issue (DRAFT,
PENDING_APPROVAL, APPROVED or REJECTED), not to a sent PO. Closure rejects open
receipt variances, rechecking them inside the locked PO transaction; it preserves
physical/accepted quantities and does not refund payments or reverse Inventory.
The 000016 down migration refuses rollback if new action evidence exists; do not
delete that evidence as a workaround.

### PO revision rollout and retry semantics

Apply `000017_purchase_order_revision_evidence` after 000015/000016 before this
binary. It extends the shared command journal and blocks ordinary UPDATE, DELETE
and TRUNCATE of archived PO revisions. It does not reconstruct missing historical
evidence or restrict database owners who deliberately disable/drop these guards.
The down migration refuses rollback if completed REVISION commands exist, retains
all archived data, and otherwise removes the new guards; never delete evidence to
make rollback succeed. No migration has been applied to a live DB by these tests.

Revision fingerprint binds PO, REVISION, expected_version, supplier, delivery date,
reason, notes and ordered lines. Notes/reason/quantities are trimmed; UUIDs/date
are parsed before hashing. Line order and quantity text remain significant (`8`
versus `8.0` conflicts). Retry reads durable evidence before looking up offers:
same actor/key/payload returns the original frozen DRAFT with the same line IDs,
prices and version, even if offers are removed or newer revisions exist. GET PO
detail supplies current state; GET revisions supplies archived terms, not the
current draft. Changed command/key scope conflicts are 409. DB errors remain 500.

Archive, live PO/lines, REVISION_CREATED history and result evidence commit
atomically. A failed command consumes no key or revision number. Each new revision
uses current effective offers, increments revision/version once, clears old
approval/send metadata and makes the signed reviser the new draft creator; they
cannot self-approve/reject. Approval and send are required again. The endpoint
does not automatically notify the supplier, create stock or trigger payments.
Private HTTP revision clients must now send the required key and expected_version.

Draft editing additionally requires `000018_purchase_order_draft_edit_commands`.
It shares the same ordered-lines/string-normalization fingerprint policy as
revision under action EDIT_DRAFT. Matching retry returns the frozen committed
draft before consulting offers, including after submit; different commands with
one key or stale versions return 409. Two different edits at one version cannot
overwrite each other. PO/lines/history/result commit together; failures consume
no key. The creator-only maker-checker policy is unchanged: editing retains the
original creator and does not introduce a separate editor-exclusion rule.
Supplier may change while draft if offers/units/MOQ/currency validate. Existing
private PUT clients must adopt the new header/version. Rollback refuses to erase
completed EDIT_DRAFT evidence. Previously misnumbered drafts are not rewritten by
this migration; the fix preserves revision on subsequent draft edits.

The lifecycle is `DRAFT → PENDING_APPROVAL → APPROVED → SENT`. After sending, the supplier may acknowledge, request changes, or decline. A change request can be revised into a new `DRAFT` revision and must pass approval again. The old issued revision is kept in `purchase_order_revision_snapshots`; the live PO row represents the latest revision. A matching action retry returns its committed result without adding another history entry. A partially received PO can be closed to cancel its remaining quantity; received quantities are preserved. Eligible pre-issue states may transition to `CANCELLED`. Every create/update/transition is written to `purchase_order_status_history`. Version-checked updates reject stale concurrent writes. Unit-price times decimal quantity is rounded half-up to the nearest minor currency unit. Decimal quantities support up to six fractional digits.

## Supply planning preview

- `POST /api/v1/admin/procurement/supply-plans/preview` accepts forecast and inventory snapshots, their source IDs/timestamps, a caller-provided freshness threshold and policy version, plus per-material forecast, on-hand, open-PO, safety-stock, MOQ, pack-size and lead-time values.
- `POST /api/v1/admin/procurement/supply-plans` saves the same calculation as a `DRAFT` plan with revision 1 and an immutable JSON snapshot. `GET /api/v1/admin/procurement/supply-plans/{supplyPlanID}` retrieves its current snapshot; `GET /api/v1/admin/procurement/supply-plans?limit=50&offset=0` lists plans.
- `POST /api/v1/admin/procurement/supply-plans/{supplyPlanID}/revisions` recalculates a draft plan from a new input snapshot and requires its current `expected_version`. Revision snapshots are append-only; `GET /api/v1/admin/procurement/supply-plans/{supplyPlanID}/revisions` returns the complete history. Concurrent or stale updates return `409 CONFLICT`.
- `POST /api/v1/admin/procurement/supply-plans/{supplyPlanID}/submit` moves a draft to `PENDING_APPROVAL`; `POST .../{supplyPlanID}/approve` and `POST .../{supplyPlanID}/reject` require `PROCUREMENT_APPROVE#GLOBAL`. The submitter and creator cannot approve or reject their own plan, and rejection requires a reason. `GET .../{supplyPlanID}/status-history` exposes the audit trail.
- `POST /api/v1/admin/procurement/supply-plans/{supplyPlanID}/purchase-order` accepts `supplier_id` and `expected_version`. It converts only an approved plan, uses positive proposed quantities from the approved snapshot, and snapshots current effective supplier offers into a draft PO. The conversion and PO creation are atomic and unique per plan; retrying with the same supplier returns the existing PO, while a different supplier conflicts.
- A plan line may include `manual_override_quantity` and a required `manual_override_reason`. Positive overrides must meet the plan MOQ and be a multiple of pack size; zero is allowed to explicitly omit a material. Snapshots preserve the algorithm's `recommended_order_quantity`, final `proposed_order_quantity`, reason and override warning. Add overrides while the plan is a draft so the approved revision contains the reviewed amounts.
- The preview endpoint resolves material code/name/unit from the active Procurement catalog. It returns a `DRAFT_PREVIEW`, or `INPUT_WARNING` when either source is stale/future-dated; previewing does not save, approve, or create a PO.
- Net requirement is `max(0, forecast + safety stock - on hand - open PO)`. A positive proposal is raised to MOQ and rounded up to a pack-size multiple. The response includes rounding excess, projected inventory and a suggested order-by date computed as horizon start minus lead time.
- These plans use caller-supplied snapshots. Live Analytics/Inventory connectors and manual overrides remain separate work. If a selected supplier's current minimum quantity exceeds the approved proposal or no effective offer exists, conversion is rejected; update and reapprove the plan before retrying.

## Goods receipt and quality inspection

Schema migration `000014_receipt_inspection_idempotency` must precede deployment.
Inspection now requires `Idempotency-Key` (16–128 visible ASCII characters), even
on private HTTP calls. The signed inspector, receipt ID, key and normalized payload
identify a replay. Inspection strings are trimmed and lines sorted by ID; decimal
text remains significant. Same command returns the current receipt without another
inspection, PO quantity update, history entry or outbox event. Different actor,
key or payload after inspection returns 409; historical inspections without replay
evidence are not backfilled or assumed equivalent. Evidence has no TTL and is
stored atomically with all business/outbox writes. Validation/DB rollback consumes
no key. Existing internal application calls can still use the non-HTTP inspection
method, but cannot retroactively claim idempotent replay of historical inspections.

- `POST /api/v1/admin/procurement/goods-receipts` creates a draft receipt against a `SENT`, `ACKNOWLEDGED`, or `PARTIALLY_RECEIVED` PO. Each line references a PO line and records received quantity plus optional supplier lot and paired manufacture/expiry dates. It requires an `Idempotency-Key` header (16–128 visible ASCII characters); retrying the same key and payload returns the original receipt, while reusing the key with changed data returns `409 CONFLICT`.
- `GET /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}` returns the receipt snapshot.
- `POST /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/inspect` records accepted/rejected quantities, optional humidity percentage, and inspection notes for every receipt line.
- `POST /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/variances/{varianceID}/resolve` requires `PROCUREMENT_VARIANCE_RESOLVE` with matching stored WAREHOUSE or GLOBAL scope, Idempotency-Key and `{expected_version, decision, note}`. Version is the receipt's, not the PO's. A nonblank note and OPEN variance on an INSPECTED receipt are required; decision is `ACCEPT_VARIANCE` or `REJECT_VARIANCE`.

### Variance decision rollout and retry semantics

Apply `000019_receipt_variance_resolution_results` before this binary. Existing
private resolve clients must add the key and receipt expected_version. Signed
actor/key is unique across variance resolutions (separate from PO/create/inspect
keys); fingerprint binds receipt, variance, version, uppercased trimmed decision
and trimmed note. Same command replays the frozen receipt snapshot, including
after later resolutions/PO closure. GET receipt for current state. Current JWT
permission/warehouse checks still apply before replay. Changed command conflicts;
historically resolved variances without a matching journal cannot claim replay.
Failed transactions consume no key. Receipt/PO/history/result commit together;
evidence has no TTL. Down migration refuses rollback while evidence exists; never
delete completed decisions to bypass it.

The transaction locks receipt then PO, as inspection does, serializing decisions
across different receipts for the same PO. The final decision recomputes PO status
from existing accepted quantities only after all variances are resolved. New
commands cannot reopen CLOSED/cancelled POs. Neither decision changes physical,
accepted or rejected quantities, emits another GoodsReceivedEvent, releases
Inventory quarantine nor refunds money. It records a business decision only;
physical disposition/Inventory QC and finance actions remain separate workflows.
VARIANCE_RESOLVE alone does not grant receipt-read access: the UI also needs the
existing VIEW GLOBAL or matching RECEIVE grant to load receipt detail/version.

Accepted plus rejected quantity must equal the physical quantity received. Inspection is versioned and updates the receipt, PO received/accepted quantities, PO lifecycle, histories, and one `GoodsReceivedEvent` per receipt line in a transactional outbox. Event rows commit atomically with inspection, so a failed database transaction cannot publish stock that Procurement did not accept. Outbox delivery is at-least-once; consumers must deduplicate by stable CloudEvent ID. Inventory PR #118 receives `quantity_received` as physical QUARANTINE stock, including fully rejected lines; accepted/rejected quantities are Procurement evidence and do not authorize Inventory QC release. It records open `OVER_RECEIVED` variances when physical quantity exceeds a PO line's remaining open quantity, and `QUALITY_REJECTED` variances for rejected quantities; the reason comes from required inspection notes. An under-delivery remains a valid partial receipt because the service has no supplier shipment notice proving that the delivery was complete. An open variance keeps the PO in `PARTIALLY_RECEIVED` and prevents closing the remaining quantity. Resolving the last variance recalculates the PO status; it reaches `RECEIVED` only when every line's accepted quantity meets the ordered quantity. Accepted quantity cannot exceed the PO quantity; rejected goods can be replaced under the same PO.

Example create request:

```json
{
  "supplier_id": "5d22523b-14e4-41a8-a788-414eaf74c751",
  "expected_delivery_date": "2026-12-01",
  "notes": "Restock for December production",
  "lines": [
    { "material_id": "d5dc087b-ef5f-44b3-9b9f-f32690a72311", "quantity": "25.5" }
  ]
}
```

## Integration boundary

`GoodsReceivedEvent` is emitted to `procurement.events.v1` after receipt inspection commits. One immutable line ID is reused on outbox retries and the Kafka key is sku_code. The contract is aligned with Inventory PR #118 at 21775d667bdf1373dce76c33d27f35aba1c8f5b0, without merging that branch. Supplier UUID strings are valid opaque supplier codes; Inventory maps them separately. warehouse_id is the Procurement warehouse code and requires Inventory mapping. Inventory also requires each SKU to be provisioned with the correct warehouse and explicit GRAM/UNIT base unit.

Physical quantity_received enters QUARANTINE, not sellable inventory. Do not substitute quantity_accepted into that field or drop fully rejected receipts: that would conceal physical custody. Optional inspection/lot/date fields remain evidence only; the consumer ignores them for quantity/QC and generates its own batch identity. Rejected-stock disposal/return and Inventory QC release need separate owning-service acceptance before production rollout.

The producer supplies stable unsampled receipt-local traceparent context. Quantities must be at least 0.01; KG converts exactly to whole grams, G/GRAM/UNIT/PIECE must be integral, and converted stock cannot exceed int32. Unsupported/inexact values fail inspection atomically with 400 INVALID_ARGUMENT; no rounding or partial receipt/PO/outbox commit is allowed. This does not validate Inventory's live warehouse/SKU/unit configuration.

Run `python scripts/verify-procurement-receipt-contract.py` from the repository root with the MVP Python dependencies and Go installed. Add `--inventory-ref 21775d66` to also validate actual producer bytes against the PR's schema, using only local Git objects. This is schema compatibility, not execution of the real Inventory consumer or a broker/DB E2E test.

Cutover: new events have traceparent, but persisted older outbox payloads are never rewritten by retries. Audit pending/published events before enabling the consumer; missing trace context or incompatible quantities require an operator-reviewed replay/correction plan. Do not regenerate IDs or blindly replay old receipts into stock. Inventory's new-group LastOffset policy also requires coordinated startup before new Procurement events, or an explicit retained-offset replay plan.

## Isolated producer-consumer verification

From the repository root on Windows with Docker Desktop and Go available:

```powershell
.\services\procurement-service\scripts\run-receiving-integration.ps1
```

The launcher creates a uniquely named disposable PostgreSQL 16/Kafka project on loopback-only
temporary ports. It archives Inventory PR #118 at `21775d667bdf1373dce76c33d27f35aba1c8f5b0`
into `.tmp`, copies only a test adapter there, and does not merge/edit that Inventory branch.
The pinned commit must exist in local Git objects. Procurement uses its real migrations,
application/repository, inspection transaction, outbox worker and Kafka producer. Inventory
uses the pinned real migrations and consumer. Each service has its own generated DB schema.

Coverage: migration rerun; concurrent submit/approve/reject/send and supplier
acknowledgement/change/decline/cancel/close, restart replay,
cross-PO key races and injected journal failure with PO/history rollback;
transactional variance denial and closing a partially received PO without losing quantities;
revision journal fault rollback of archive/PO/history, concurrent same-key and
cross-PO revision races, replay without offers, immutable archive guards,
two revision generations, pagination and receipt-based revision denial;
draft edit journal rollback, duplicate and competing commands, revision-preserving
edit/replay after submit, combined list filters and deterministic pagination;
variance journal fault rollback, duplicate/restart/frozen replay, concurrent
decisions across receipts with correct final PO status and no quantity/event changes;
master create journal fault rollback, frozen restart replay, concurrent same-key
and cross-kind creates, price-window/version races and rollback of prior window closing;
all six master mutation actions with journal/cleanup rollback, frozen restart replay,
stale/competing writes, immutable code/unit, price reactivation overlap and monotonic timestamps;
supplier contract journal faults, frozen restart replay, concurrent revision allocation,
activation overlap and termination races, reason audit, expiry/parent guards and DB immutability;
receipt-create idempotency, partial and fully rejected receipts,
injected outbox failure with receipt/PO/history rollback, real broker acknowledgement, receipt
provenance, exact 3500-gram physical stock in two undated QUARANTINE batches, zero sellable stock,
replayed-message dedupe, committed offsets and invalid-quantity DLQ without stock mutation.
This is not HTTP/JWT/Web acceptance, Inventory QC release or production deployment approval.
The fully rejected quantity remains physical quarantine evidence; its disposition is still a
separate business workflow.

The launcher removes only its own containers/network/anonymous volumes afterward; archived
source and synthetic published-event evidence remain in the printed `.tmp/receiving-e2e-*`
directory. `-KeepStack` retains that isolated project for diagnosis. No existing DB/volume,
live broker, deployment configuration or credentials are used.

## Configuration

- `PROCUREMENT_HTTP_ADDRESS` (default `:8008`)
- `PROCUREMENT_DATABASE_URL` (default `postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable`)
- `PROCUREMENT_OIDC_ISSUER` (required)
- `PROCUREMENT_OIDC_AUDIENCE` (optional; defaults to `omama-api`, which must be present in Identity-issued Procurement tokens)
- `PROCUREMENT_OIDC_JWKS_URL` (optional; defaults to `${PROCUREMENT_OIDC_ISSUER}/protocol/openid-connect/certs`)
- `PROCUREMENT_OIDC_ALLOW_LOCAL_HTTP` (optional; loopback-only development/testing)
- `KAFKA_BROKERS` (optional comma-separated brokers; when unset, committed outbox events remain pending until a broker is configured)
