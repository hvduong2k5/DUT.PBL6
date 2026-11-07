# Procurement Service (MS-08)

Procurement Service owns supplier records, raw material definitions, supplier-specific offers, Purchase Order approval, and internal goods-receipt/QC records. Goods receipt inspection updates PO receipt quantities, but does not mutate Inventory or trigger finance effects.

## Run locally

```powershell
Set-Location services/procurement-service
$env:PROCUREMENT_INTERNAL_TOKEN = "procurement-local-development-secret"
docker compose up --build
```

The HTTP API listens on `localhost:8008`; PostgreSQL is exposed on `localhost:5449`. The service applies pending migrations at startup. `GET /health/live` reports process liveness and `GET /health/ready` checks the database.

All `/api/v1/admin/procurement/*` routes require these headers from the trusted API Gateway:

- `X-Internal-Token: <PROCUREMENT_INTERNAL_TOKEN>`
- `X-User-ID: <authenticated-user-id>`
- `X-User-Role: ADMIN | SUPPLY_MANAGER | PROCUREMENT_MANAGER`

Goods receipt endpoints also allow `WAREHOUSE_STAFF`. Warehouse Staff can create, view, and inspect goods receipts but cannot administer suppliers, materials, offers, or purchase orders.

Do not expose this service directly to public clients. The gateway must authenticate the user and set these trusted headers.

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
- `POST /api/v1/admin/procurement/supply-plans/{supplyPlanID}/submit` moves a draft to `PENDING_APPROVAL`; `POST .../{supplyPlanID}/approve` and `POST .../{supplyPlanID}/reject` are restricted to `ADMIN` or `PROCUREMENT_MANAGER`. The submitter and creator cannot approve or reject their own plan, and rejection requires a reason. `GET .../{supplyPlanID}/status-history` exposes the audit trail.
- `POST /api/v1/admin/procurement/supply-plans/{supplyPlanID}/purchase-order` accepts `supplier_id` and `expected_version`. It converts only an approved plan, uses positive proposed quantities from the approved snapshot, and snapshots current effective supplier offers into a draft PO. The conversion and PO creation are atomic and unique per plan; retrying with the same supplier returns the existing PO, while a different supplier conflicts.
- A plan line may include `manual_override_quantity` and a required `manual_override_reason`. Positive overrides must meet the plan MOQ and be a multiple of pack size; zero is allowed to explicitly omit a material. Snapshots preserve the algorithm's `recommended_order_quantity`, final `proposed_order_quantity`, reason and override warning. Add overrides while the plan is a draft so the approved revision contains the reviewed amounts.
- The preview endpoint resolves material code/name/unit from the active Procurement catalog. It returns a `DRAFT_PREVIEW`, or `INPUT_WARNING` when either source is stale/future-dated; previewing does not save, approve, or create a PO.
- Net requirement is `max(0, forecast + safety stock - on hand - open PO)`. A positive proposal is raised to MOQ and rounded up to a pack-size multiple. The response includes rounding excess, projected inventory and a suggested order-by date computed as horizon start minus lead time.
- These plans use caller-supplied snapshots. Live Analytics/Inventory connectors and manual overrides remain separate work. If a selected supplier's current minimum quantity exceeds the approved proposal or no effective offer exists, conversion is rejected; update and reapprove the plan before retrying.

## Goods receipt and quality inspection

- `POST /api/v1/admin/procurement/goods-receipts` creates a draft receipt against a `SENT`, `ACKNOWLEDGED`, or `PARTIALLY_RECEIVED` PO. Each line references a PO line and records received quantity plus optional supplier lot and paired manufacture/expiry dates. It requires an `Idempotency-Key` header (16–128 visible ASCII characters); retrying the same key and payload returns the original receipt, while reusing the key with changed data returns `409 CONFLICT`.
- `GET /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}` returns the receipt snapshot.
- `POST /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/inspect` records accepted/rejected quantities, optional humidity percentage, and inspection notes for every receipt line.
- `POST /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/variances/{varianceID}/resolve` lets `ADMIN` or `PROCUREMENT_MANAGER` resolve an open variance using `ACCEPT_VARIANCE` or `REJECT_VARIANCE` and a required note.

Accepted plus rejected quantity must equal the physical quantity received. Inspection is versioned and updates the receipt, PO received/accepted quantities, PO lifecycle, and both histories in one database transaction. It records open `OVER_RECEIVED` variances when physical quantity exceeds a PO line's remaining open quantity, and `QUALITY_REJECTED` variances for rejected quantities; the reason comes from required inspection notes. An under-delivery remains a valid partial receipt because the service has no supplier shipment notice proving that the delivery was complete. An open variance keeps the PO in `PARTIALLY_RECEIVED` and prevents closing the remaining quantity. Resolving the last variance recalculates the PO status; it reaches `RECEIVED` only when every line's accepted quantity meets the ordered quantity. Accepted quantity cannot exceed the PO quantity; rejected goods can be replaced under the same PO. Procurement does not create Inventory batches, increase stock, or publish `GoodsReceivedEvent`; those effects require the Inventory contract and idempotent integration to be aligned first.

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

Inventory's current `GetStockLevel` RPC can support a later stock lookup. The existing `GoodsReceivedEvent` schema and Inventory implementation need alignment on supplier/warehouse identifiers, quantity precision, manufacturing/expiry dates, inspection results, and idempotency before publishing that event or claiming stock was received.

## Configuration

- `PROCUREMENT_HTTP_ADDRESS` (default `:8008`)
- `PROCUREMENT_DATABASE_URL` (default `postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable`)
- `PROCUREMENT_INTERNAL_TOKEN` (required)
