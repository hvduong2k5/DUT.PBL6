# Order Management MVP — Data & Mock API Matrix

## Browser/BFF API

Browser chỉ gọi Next.js BFF cùng origin. BFF allowlist method/path/query/body, kiểm tra trusted Origin cho mutation và gọi Mockoon port `4017`.

| Capability | Method | Browser BFF | Upstream mock |
| --- | --- | --- | --- |
| `ORD-C01` | GET | `/api/orders` | `/api/v1/customers/me/orders` |
| `ORD-C02` | GET | `/api/orders/{orderId}` | `/api/v1/orders/{orderId}` |
| `ORD-C03` | POST | `/api/orders/{orderId}/cancellations` | `/api/v1/orders/{orderId}/cancellations` |
| `ORD-C04` | POST | `/api/orders/guest-access/challenges` | `/api/v1/orders/guest-access/challenges` |
| `ORD-C05` | POST | `/api/orders/guest-access/challenges/{challengeId}/verify` | `/api/v1/orders/guest-access/challenges/{challengeId}/verify` |

## Query danh sách

| Query | Rule |
| --- | --- |
| `q` | Optional, 2–80 ký tự; tìm theo mã Order hoặc snapshot product name |
| `status` | Optional enum hoặc `ALL` |
| `year` | Optional `2020..current year` |
| `page` | Integer `1..1000`, mặc định `1` |
| `pageSize` | Integer `1..20`, mặc định `10` |
| `mockScenario` | Development only; BFF từ chối trong production |

Browser không được gửi `customerId`, owner/contact filter, internal status/source hoặc arbitrary sort.

## Order list item

```json
{
  "orderId": "order-mock-20260926-001",
  "orderNumber": "OMA-260926-001",
  "placedAt": "2026-09-26T03:05:00Z",
  "status": "PROCESSING",
  "statusLabel": "Đang chuẩn bị",
  "paymentStatus": "PAID",
  "shippingStatus": "NOT_SHIPPED",
  "totalVnd": 665000,
  "currency": "VND",
  "itemCount": 3,
  "previewItems": [{ "name": "Bánh Ngũ Sắc Cung Đình", "quantity": 2, "imageUrl": "https://...", "imageAlt": "..." }],
  "canCancel": true,
  "cancelPolicyMessage": "Có thể tự hủy trước khi bắt đầu đóng gói."
}
```

## Order detail boundary

- Identity: `orderId`, `orderNumber`, `source`, `placedAt`.
- Separate statuses: `status`, `payment.status`, `shipping.status`.
- Immutable snapshot: lines, SKU label, unit price, quantity, subtotal, shipping, discount, total.
- Recipient snapshot: full name, masked phone/email for Guest, shipping address and delivery note.
- Public timeline: code, label, description, timestamp, current/completed flags; không có internal actor/source payload.
- Cancellation policy: `canCancel`, `cancelBy`, Customer-safe message và allowed reason codes.

## Cancellation request

```json
{
  "reasonCode": "CHANGED_MIND",
  "note": "Tôi muốn thay đổi sản phẩm.",
  "idempotencyKey": "order-cancel-550e8400-e29b-41d4-a716-446655440000"
}
```

Local BFF nhận key trong body để UI/Mockoon executable. API Candidate sau implement sẽ đề xuất chuyển key sang `Idempotency-Key` header.

Response phải phân biệt:

- `refundRequired=false`: Order unpaid được cancel, yêu cầu release reservation.
- `refundRequired=true`: Order paid được cancel theo policy và tạo refund obligation; không được ghi “đã hoàn tiền”.
- `409 ORDER_CANCELLATION_NOT_ALLOWED`: policy đã thay đổi hoặc Order đi quá mốc.

## Guest challenge

### Request challenge

```json
{ "orderNumber": "OMA-260926-001", "contact": "0914288668" }
```

Response `202` luôn trung tính cho combination đúng format. Mock local trả `challengeId` và hint; production không được tiết lộ contact có khớp Order hay không.

### Verify challenge

```json
{ "otp": "789214" }
```

Verify thành công trả `orderId`, `orderPath`, expiry và set Guest Order access cookie HttpOnly. OTP không được log hoặc đưa vào URL.

## Error semantics

| HTTP | Code | UI behavior |
| --- | --- | --- |
| 400/422 | `INVALID_REQUEST` / `VALIDATION_ERROR` | Field error; không gửi request sai |
| 401 | `ORDER_ACCESS_REQUIRED` | Đăng nhập hoặc tra cứu Guest lại |
| 404 | `ORDER_NOT_FOUND` | Không tìm thấy hoặc ngoài owner scope; không phân biệt |
| 409 | `ORDER_CANCELLATION_NOT_ALLOWED` | Reload detail, hiển thị policy/hỗ trợ |
| 410 | `GUEST_CHALLENGE_EXPIRED` | Tạo challenge mới |
| 429 | `ORDER_RATE_LIMITED` | Hiển thị retry time |
| 500 | `ORDER_ERROR` | Error state + retry |
| 503 | `ORDER_UPSTREAM_UNAVAILABLE` | Kiểm tra Mockoon/API Gateway + retry |

## Mock scenarios

| Scenario | Kết quả |
| --- | --- |
| mặc định | 4 Order; detail processing còn eligible để cancel |
| `orders-empty` | Danh sách rỗng |
| `orders-error`, `orders-slow` | List error/loading |
| `order-pending-payment`, `order-shipped`, `order-delivered`, `order-cancelled` | Detail state |
| `order-not-found`, `order-error` | Detail error states |
| `cancel-unpaid`, `cancel-paid` | Cancel success không/có refund obligation |
| `cancel-not-eligible`, `cancel-error`, `cancel-slow` | Cancel conflict/error/loading |
| `guest-invalid` | Challenge trả neutral accepted |
| `guest-otp-invalid`, `guest-otp-expired`, `guest-rate-limited` | Verify failure |

## Dữ liệu không public

- Customer ID/owner lookup logic, full Guest verification evidence và OTP.
- Exact inventory/reservation allocation, warehouse, Batch/Lot và cost/margin.
- Provider/payment raw payload, reconciliation/fraud note.
- Staff identity, internal transition reason, audit payload và SLA/queue data.
- Carrier credential, internal tracking integration và service topology.
