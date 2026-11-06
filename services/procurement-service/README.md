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
- `GET /api/v1/admin/procurement/materials?limit=50&offset=0`
- `POST /api/v1/admin/procurement/materials`
- `GET /api/v1/admin/procurement/materials/{materialID}`
- `PATCH /api/v1/admin/procurement/materials/{materialID}/status`
- `GET /api/v1/admin/procurement/suppliers/{supplierID}/offers`
- `POST /api/v1/admin/procurement/suppliers/{supplierID}/offers`

Supplier types: `FARMER`, `COOPERATIVE`, `OCOP_BUSINESS`, `COMPANY`. A supplier-material offer stores a positive integer minor-unit price, 3-letter currency, unit, decimal minimum quantity, and lead time. A PO can only use an active offer for an active supplier/material, must meet the offer's minimum quantity, and snapshots the offer price and material name/code. All lines in one PO must use the same currency.

## Purchase Order workflow

- `GET /api/v1/admin/procurement/purchase-orders?limit=50&offset=0&supplier_id={uuid}&status=DRAFT`
- `POST /api/v1/admin/procurement/purchase-orders` creates a `DRAFT` from active supplier offers.
- `GET /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}` returns the PO, price snapshots, version, and current lifecycle metadata.
- `PUT /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}` replaces a draft's supplier/lines/notes/delivery date. Only drafts can be edited.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/submit` submits for approval.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/approve` approves a pending PO. The creator cannot approve their own PO.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/reject` rejects a pending PO and requires `{"reason":"..."}`.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/send` records that an approved PO was sent to the supplier.
- `POST /api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/cancel` cancels an eligible PO and requires a reason.

The lifecycle is `DRAFT → PENDING_APPROVAL → APPROVED → SENT`; a pending PO may instead be `REJECTED`, and eligible states may transition to `CANCELLED`. Every create/update/transition is written to `purchase_order_status_history`. Version-checked updates reject stale concurrent writes. Unit-price times decimal quantity is rounded half-up to the nearest minor currency unit. Decimal quantities support up to six fractional digits.

## Goods receipt and quality inspection

- `POST /api/v1/admin/procurement/goods-receipts` creates a draft receipt against a `SENT` or `PARTIALLY_RECEIVED` PO. Each line references a PO line and records received quantity plus optional supplier lot and paired manufacture/expiry dates.
- `GET /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}` returns the receipt snapshot.
- `POST /api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/inspect` records accepted/rejected quantities, optional humidity percentage, and inspection notes for every receipt line.

Accepted plus rejected quantity must equal the physical quantity received. Inspection is versioned and updates the receipt, PO received/accepted quantities, PO lifecycle, and both histories in one database transaction. An over-receipt is rejected. A PO reaches `RECEIVED` only when every line's accepted quantity meets the ordered quantity; otherwise it remains `PARTIALLY_RECEIVED`. A rejected quantity does not count toward fulfillment. Procurement does not create Inventory batches, increase stock, or publish `GoodsReceivedEvent`; those effects require the Inventory contract and idempotent integration to be aligned first.

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
