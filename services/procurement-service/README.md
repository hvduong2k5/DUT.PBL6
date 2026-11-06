# Procurement Service (MS-08)

Initial service slice for supplier and raw-material master data. It owns supplier records, raw material definitions, and supplier-specific offers. Purchase orders, goods receipt publication, and downstream inventory/finance effects are not part of this slice.

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

Supplier types: `FARMER`, `COOPERATIVE`, `OCOP_BUSINESS`, `COMPANY`. A supplier-material offer stores a positive integer minor-unit price, ISO-style 3-letter currency, unit, decimal minimum quantity, and lead time.

## Integration boundary

Inventory's current `GetStockLevel` RPC can support a later stock lookup. Goods receipt must not mutate Inventory data directly. The existing `GoodsReceivedEvent` schema and Inventory implementation need alignment on supplier/warehouse identifiers, quantity precision, manufacturing/expiry dates, and inspection results before publishing that event or claiming stock was received.

## Configuration

- `PROCUREMENT_HTTP_ADDRESS` (default `:8008`)
- `PROCUREMENT_DATABASE_URL` (default `postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable`)
- `PROCUREMENT_INTERNAL_TOKEN` (required)
