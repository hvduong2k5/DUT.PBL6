# MS-07 Promotion Service

Promotion Service owns campaigns, coupons, voucher holds and redemption records. This initial
foundation provides a Go service process, PostgreSQL migrations, liveness/readiness endpoints,
and a local PostgreSQL Compose stack. The voucher domain and transactional hold repository now
implement discount calculation, availability checks, hold creation, finalization and release.
The existing `packages/proto/promotion/v1/promotion.proto` remains the integration contract.

## Run locally

```powershell
Set-Location services/promotion-service
$env:PROMOTION_INTERNAL_TOKEN = "promotion-local-development-secret"
docker compose up --build
```

The service listens on HTTP `localhost:8017` and gRPC `localhost:9017`. Readiness checks
PostgreSQL at `GET /health/ready`; liveness is `GET /health/live`. PostgreSQL is exposed on
`localhost:5447`. gRPC calls require `authorization: Bearer <PROMOTION_INTERNAL_TOKEN>` metadata.

## Admin coupon API

- `GET /api/v1/admin/coupons?limit=50&offset=0`
- `POST /api/v1/admin/coupons` creates a coupon in `DRAFT` state.
- `PUT /api/v1/admin/coupons/{couponID}` edits a draft. Published coupons are immutable; create
  a new coupon if the offer must change.
- `POST /api/v1/admin/coupons/{couponID}/publish` publishes now or schedules for `starts_at`.
- `POST /api/v1/admin/coupons/{couponID}/pause` pauses an active/scheduled coupon and requires a reason.
- `GET /api/v1/promotions/active?limit=50&offset=0` lists public, active, in-window coupons with remaining total uses.

Admin calls require `X-Internal-Token`, `X-User-ID`, and `X-User-Role: SALES_MANAGER`. The
`X-User-*` values are trusted only from the authenticated API Gateway; do not expose this service
directly to public clients. A lifecycle worker advances scheduled coupons and ends expired ones;
all manual/automatic status changes are audited.

## Configuration

- `PROMOTION_HTTP_ADDRESS` (default `:8017`)
- `PROMOTION_GRPC_ADDRESS` (default `:9017`)
- `PROMOTION_DATABASE_URL` (local default targets port `5447`)
- `PROMOTION_INTERNAL_TOKEN` (required)

Database migrations run at startup and can also be applied explicitly with `go run ./cmd/migrate`.

## Current data foundation

The first migration establishes promotion and coupon definitions, order-scoped voucher holds,
finalized redemption usage, command idempotency records and a transactional outbox. Coupon
redemption limits are checked in a transaction while the coupon row is locked; a cached counter
is never the authority for availability. The gRPC adapter implements the current Promotion v1
source contract, including the feature-gated Order 3.2 voucher-hold RPCs.
