# Procurement Service (MS-08)

Procurement Service owns supplier records, raw material definitions, supplier-specific offers, Purchase Order approval, and internal goods-receipt/QC records. Goods receipt inspection updates PO receipt quantities, but does not mutate Inventory or trigger finance effects.

## Run locally

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

### Kong integration: management reads and idempotent receiving

The shared `admin-stable.openapi.yaml` version 2.2.0 documents eleven operations:
six supplier/material/offer GETs, PO create/detail, receipt create/detail and receipt
inspection. Kong routes the exact contracted GET/POST/OPTIONS path shapes to
`http://procurement-service:8008`, without stripping the path, retrying business
requests or injecting credentials. Procurement verifies the employee bearer JWT
and operation permission/scope; the gateway does
not exchange a cookie for a token. A browser's Identity session alone is not enough
until the separate BFF relay is implemented.

PO approval/send/revisions/cancel/close, variance resolution, plan/contract APIs and
master-data mutations are **not** exposed. Their implementation remains private;
full contracts and retry/idempotency acceptance are needed before gateway expansion.
In particular, creating a PO does not yet provide the whole approval/dispatch UI.
This is not a complete Procurement Web/BFF integration.

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
all eleven operation examples, denied methods/paths and network wiring. Separately
validate the real Kong 3.9.3 parser with:

```powershell
docker run --rm -e KONG_DATABASE=off -v "${PWD}/services/api-gateway/config/kong.yml:/config/kong.yml:ro" kong:3.9.3 kong config parse /config/kong.yml
```

For real router-selection coverage, CI also starts a disposable Kong with debug
headers enabled **only in that test**, admin listener off, and the Procurement
hostname mapped to the container's loopback (no application or live upstream).
`verify-procurement-admin-contract.py --gateway-url http://127.0.0.1:<test-port>`
checks the selected route for eleven operations, 55 rejected method combinations and eleven
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

Supplier types: `FARMER`, `COOPERATIVE`, `OCOP_BUSINESS`, `COMPANY`. Contracts use immutable revisions, effective dates, scope, payment terms, and an opaque `document_file_id` rather than an arbitrary URL. Contract endpoints require `ADMIN` or `PROCUREMENT_MANAGER`; activation rejects overlapping active agreements for the same supplier and scope. Offers are immutable price versions with effective dates, MOQ and lead time; overlapping ranges are rejected and a new version closes a previous open-ended version. A PO can only use an effective active offer for an active supplier/material, must meet the offer's minimum quantity, and snapshots the offer price and material name/code. All lines in one PO must use the same currency.

## Purchase Order workflow

- `GET /api/v1/admin/procurement/purchase-orders?limit=50&offset=0&supplier_id={uuid}&status=DRAFT`
- `POST /api/v1/admin/procurement/purchase-orders` creates a `DRAFT` from active supplier offers and requires an `Idempotency-Key` header (16–128 visible ASCII characters). The key is scoped to the authenticated user; retrying the same key and payload returns the original PO, while reusing it with a changed payload returns `409 CONFLICT`.
- `GET /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}` returns the PO, price snapshots, version, and current lifecycle metadata.
- `PUT /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}` replaces a draft's supplier/lines/notes/delivery date. Only drafts can be edited.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/submit` submits for approval.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/approve` approves a pending PO. The creator cannot approve their own PO.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/reject` rejects a pending PO and requires `{"reason":"..."}`.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/send` records that an approved PO was sent to the supplier. Repeating the request after it was sent returns the current PO without adding another transition.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/supplier-response` records a supplier response: `ACKNOWLEDGED`, `CHANGE_REQUESTED`, or `DECLINED`. Change requests and declines require a note.
- `GET /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/revisions` lists immutable snapshots of prior issued revisions.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/revisions` creates the next draft revision after `CHANGE_REQUESTED`; it requires a reason and revised lines. The supplier cannot change. The old revision is archived before the PO is updated, in one transaction.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/cancel` cancels an eligible PO and requires a reason.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/close` closes the outstanding quantity for a `SENT`, `ACKNOWLEDGED`, or `PARTIALLY_RECEIVED` PO and requires a reason. Quantities already received remain recorded; further receipts are rejected.

The lifecycle is `DRAFT → PENDING_APPROVAL → APPROVED → SENT`. After sending, the supplier may acknowledge, request changes, or decline. A change request can be revised into a new `DRAFT` revision and must pass approval again. The old issued revision is kept in `purchase_order_revision_snapshots`; the live PO row represents the latest revision. A sent request retry is idempotent while the PO remains in a post-send state and does not create another status-history entry. A partially received PO can be closed to cancel its remaining quantity; received quantities are preserved. Eligible pre-issue states may transition to `CANCELLED`. Every create/update/transition is written to `purchase_order_status_history`. Version-checked updates reject stale concurrent writes. Unit-price times decimal quantity is rounded half-up to the nearest minor currency unit. Decimal quantities support up to six fractional digits.

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
- `POST /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/variances/{varianceID}/resolve` requires `PROCUREMENT_VARIANCE_RESOLVE` with matching WAREHOUSE or GLOBAL scope to resolve an open variance using `ACCEPT_VARIANCE` or `REJECT_VARIANCE` and a required note.

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

Coverage: migration rerun, receipt-create idempotency, partial and fully rejected receipts,
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
