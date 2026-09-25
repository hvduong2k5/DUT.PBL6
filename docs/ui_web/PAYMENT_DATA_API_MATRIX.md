# Payment MVP — Data & Mock API Matrix

## Browser/BFF API

Browser chỉ gọi Next.js BFF cùng origin. BFF xác minh signed access cookie và gọi Mockoon port `4016`; khi Backend sẵn sàng chỉ thay upstream.

| Capability | Method | Browser BFF | Upstream mock |
| --- | --- | --- | --- |
| `PAY-C01` | GET | `/api/payments/orders/{orderId}` | `/api/v1/payments/orders/{orderId}` |
| `PAY-C02` | GET | `/api/payments/orders/{orderId}/status` | `/api/v1/payments/orders/{orderId}/status` |
| `PAY-C03` | POST | `/api/payments/orders/{orderId}/attempts` | `/api/v1/payments/orders/{orderId}/attempts` |

## Payment summary public response

```json
{
  "orderId": "order-mock-20260925-001",
  "orderNumber": "OMA-260925-001",
  "method": "BANK_TRANSFER",
  "amountVnd": 665000,
  "currency": "VND",
  "paymentId": "pay-demo-001",
  "status": "PENDING",
  "orderPaymentState": "PENDING_PAYMENT",
  "expiresAt": "2026-09-26T10:15:00Z",
  "instructions": {
    "providerLabel": "Ngân hàng mô phỏng — không chuyển tiền thật",
    "beneficiary": "Ô MẠ DEMO",
    "accountNumberMasked": "0000 0000 0000",
    "transferReference": "OMA-DEMO-001",
    "qrPayloadType": "DEMO_ONLY"
  }
}
```

`orderId/orderNumber/method/amountVnd/expiresAt` được BFF lấy từ signed Checkout context, không tin dữ liệu browser và không lấy từ query. `instructions` là dữ liệu upstream đã được allowlist/sanitize.

Với COD, `instructions=null`, `status=COD_PENDING_COLLECTION` và `orderPaymentState=PAYMENT_ON_DELIVERY`; UI không hiển thị QR.

## Payment status

| Status | Ý nghĩa UI | Order implication |
| --- | --- | --- |
| `PENDING` | Chờ nguồn tin cậy xác minh | Không `PAID` |
| `SUCCEEDED` | Payment đã xác minh đúng amount/currency/reference | Backend có thể phát sự kiện cho EPIC 08 |
| `FAILED` | Attempt thất bại/hủy | Không `PAID`; retry nếu Order còn hợp lệ |
| `EXPIRED` | Attempt/reservation hết hạn | Không `PAID`; Backend quyết định retry/release |
| `REQUIRES_RECONCILIATION` | Có sai lệch cần nhân viên kiểm tra | Không tự `PAID` |
| `COD_PENDING_COLLECTION` | Nghĩa vụ thu COD | Không `PAID` trước khi thu hợp lệ |

## Request tạo lại attempt

```json
{
  "idempotencyKey": "payment-retry-550e8400-e29b-41d4-a716-446655440000"
}
```

BFF chỉ nhận key đúng format, không nhận amount/currency/orderId trong body.

## Error semantics

| HTTP | Code | UI behavior |
| --- | --- | --- |
| 400/422 | `INVALID_REQUEST` | Không gửi request; thông báo dữ liệu không hợp lệ |
| 401 | `PAYMENT_ACCESS_REQUIRED` | Không có/không hợp lệ signed access; quay lại Checkout |
| 404 | `PAYMENT_NOT_FOUND` | Không tìm thấy, hoặc signed access không thuộc Order trong URL; dùng cùng response để không tiết lộ Order |
| 409 | `PAYMENT_RETRY_NOT_ALLOWED` | Order/reservation không còn phù hợp để retry |
| 429 | `PAYMENT_RATE_LIMITED` | Hiển thị thời gian thử lại |
| 500 | `PAYMENT_ERROR` | Error state + retry |
| 503 | `PAYMENT_UPSTREAM_UNAVAILABLE` | Kiểm tra Mockoon/API Gateway + retry |

## Mock scenarios

Thêm `?mockScenario=<name>` vào URL Payment trong development.

| Scenario | Kết quả |
| --- | --- |
| mặc định | Bank transfer/COD pending theo method Checkout |
| `payment-confirmed` | `SUCCEEDED` |
| `payment-failed` | `FAILED` |
| `payment-expired` | `EXPIRED` |
| `payment-mismatch` | `REQUIRES_RECONCILIATION` |
| `payment-error` | `500` |
| `payment-slow` | Pending sau 3 giây |
| `payment-not-found` | `404` |
| `retry-not-eligible` | POST retry trả `409` |

## Dữ liệu không public

- Exact inventory/reservation internals, warehouse, Batch/Lot.
- Provider secret, webhook secret/signature, raw bank payload.
- Full bank account/card data của khách.
- Reconciliation notes, staff identity và audit internals.
- Cart context, cookie signature và upstream service topology.
