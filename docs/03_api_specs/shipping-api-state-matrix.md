# Shipping HTTP API State and Access Matrix

This matrix describes the currently implemented HTTP adapter in
[`shipping-service.openapi.yaml`](shipping-service.openapi.yaml). It is not a decision to
publish these routes through Kong.

| Route group | Authentication / intended caller | Success and empty state | Error and retry behavior | Exposure status |
|---|---|---|---|---|
| `/health/*` | No auth; platform probes | `200` when process/database is healthy | Readiness returns `503` when database is unavailable | Operational only |
| `/api/v1/internal/*` | Shared `SHIPPING_INTERNAL_TOKEN`; trusted service caller | Quote `200`; shipment creation `201`; actions `200` | `400` malformed input or missing/invalid `Idempotency-Key`, `401` missing/invalid token, `404` missing shipment, `409` transition/idempotency conflict, `422` validation, `502` carrier failure | Internal only; Order's canonical integration remains versioned gRPC |
| `/api/v1/admin/shipping/*` | Currently the same shared internal token; trusted internal tooling only | Shipment `200`; list returns `200` with `items: []` when empty | `401` invalid token, `404` missing shipment, `500` internal failure | Not employee-authenticated; employee permission/scope policy is not defined here |
| `/api/v1/orders/{orderID}/tracking` | Shared internal token; backend/BFF caller | `200`; no shipments returns `shipments: []`; only applied events and no internal notes | `400` blank order ID, `401` invalid token, `500` internal failure | Not a public customer route; no Kong route is implied |
| Fake carrier webhook | HMAC-SHA256 in `X-Shipping-Signature` using exact body | `200` accepted or duplicate callback | `400` malformed body/payload, `401` invalid signature, `404` unknown shipment, `500` processing failure | Local/test carrier only |
| GHN webhook | Constant-time comparison of `X-GHN-Webhook-Secret` | `200` accepted or duplicate callback | `404` when disabled, `400` malformed payload, `401` invalid secret, `503` shipment not committed yet (retry), `500` unexpected processing failure | Provider callback only; endpoint must be reachable over HTTPS in deployment |

## Common error shape

All HTTP errors return `{ "code": "...", "message": "...", "requestId": "..." }` and
also include the matching `X-Request-Id` response header. Messages are safe summaries and
must not include provider credentials, raw webhook bodies, or internal exception details.

Shipment creation, cancellation, and handover require a 16-128 character printable ASCII
`Idempotency-Key`. Repeated cancellation and handover requests return the current shipment
state without repeating the state transition; create replays use the persisted key.

## Explicitly not covered

- Customer-facing shipment tracking behind customer authentication and order ownership checks.
- Employee/admin APIs with employee identity, permission, and resource-scope authorization.
- Any new Kong route, public hostname, or gateway policy.
- The separate gRPC v1/v2 contracts and Kafka event schemas.
