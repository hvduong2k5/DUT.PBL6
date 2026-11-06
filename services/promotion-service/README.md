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

The service listens on `localhost:8017`. Readiness checks PostgreSQL at `GET /health/ready`;
liveness is `GET /health/live`. PostgreSQL is exposed on `localhost:5447`.

## Configuration

- `PROMOTION_HTTP_ADDRESS` (default `:8017`)
- `PROMOTION_GRPC_ADDRESS` (reserved for the contract server)
- `PROMOTION_DATABASE_URL` (local default targets port `5447`)
- `PROMOTION_INTERNAL_TOKEN` (required)

Database migrations run at startup and can also be applied explicitly with `go run ./cmd/migrate`.

## Current data foundation

The first migration establishes promotion and coupon definitions, order-scoped voucher holds,
finalized redemption usage, command idempotency records and a transactional outbox. Coupon
redemption limits are checked in a transaction while the coupon row is locked; a cached counter
is never the authority for availability. The gRPC adapter is pending because the checked-in Go
protobuf output predates the current Promotion proto and omits the Order 3.2 hold messages/RPCs.
