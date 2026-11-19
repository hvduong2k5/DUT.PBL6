# Return & Refund MVP — Data & Mock API Matrix

## Browser/BFF API

Browser chỉ gọi Next.js BFF cùng origin. BFF allowlist method/path/body/query, kiểm tra trusted Origin cho mutation và gọi Mockoon port `4018`.

| Capability | Method | Browser BFF | Upstream mock |
| --- | --- | --- | --- |
| `RET-C01` | GET | `/api/returns/orders/{orderId}/eligibility` | `/api/v1/orders/{orderId}/return-eligibility` |
| `RET-C02` | POST | `/api/returns` | `/api/v1/return-cases` |
| `RET-C03` | GET | `/api/returns/{caseId}` | `/api/v1/return-cases/{caseId}` |
| `RET-C04` | POST | `/api/returns/{caseId}/supplements` | `/api/v1/return-cases/{caseId}/supplements` |

## Eligibility response

```json
{
  "orderId": "order-mock-20260926-001",
  "orderNumber": "OMA-260926-001",
  "deliveredAt": "2026-09-28T07:30:00Z",
  "policyWindowEndsAt": "2026-10-05T07:30:00Z",
  "eligible": true,
  "policyMessage": "Yêu cầu đổi/trả trong 7 ngày kể từ khi giao.",
  "lines": [{
    "lineId": "line-order-gift",
    "productName": "Hộp Quà Cố Đô Ngự Thiện",
    "skuLabel": "Hộp quà 4 thức",
    "purchasedQty": 2,
    "alreadyClaimedQty": 0,
    "maxReturnQty": 2,
    "unitPriceVnd": 345000,
    "eligible": true
  }],
  "reasonOptions": [{ "code": "DAMAGED_IN_TRANSIT", "label": "Hư hại trong vận chuyển" }],
  "resolutionOptions": [{ "code": "REPLACEMENT", "label": "Đổi sản phẩm cùng loại" }],
  "pickupSnapshot": {
    "recipientName": "Nguyễn Văn An",
    "phoneDisplay": "0914 *** 668",
    "addressDisplay": "Tầng 3, 84 Nguyễn Huệ, Phú Nhuận, Huế"
  }
}
```

`maxReturnQty` đã trừ quantity thuộc Case đang hoạt động/đã xử lý. Browser không được gửi owner, purchased price, refund amount hay eligibility override.

## Create Case request local

```json
{
  "orderId": "order-mock-20260926-001",
  "lines": [{ "lineId": "line-order-gift", "quantity": 1 }],
  "reasonCode": "DAMAGED_IN_TRANSIT",
  "preferredResolution": "REPLACEMENT",
  "details": "Góc hộp bị móp khi nhận hàng.",
  "pickupNote": "Thu hồi sau 14h.",
  "evidence": [{
    "clientReference": "evidence-1",
    "fileName": "hop-qua-bi-mop.jpg",
    "mediaType": "image/jpeg",
    "sizeBytes": 2480000
  }],
  "idempotencyKey": "return-case-550e8400-e29b-41d4-a716-446655440000"
}
```

Local BFF nhận idempotency key trong body để UI/Mockoon executable rồi chuyển thành `Idempotency-Key` upstream. API Candidate sau implement sẽ đề xuất upload-intent/confirmed evidence riêng và key chỉ ở header.

Response `201` trả `caseId`, `caseNumber`, `status=RETURN_REQUESTED`, `casePath`, `createdAt` và Customer-safe message.

## Case detail boundary

- Identity: `caseId`, `caseNumber`, `orderId`, `orderNumber`, `createdAt`.
- Status/steps: `RETURN_REQUESTED`, `EVIDENCE_REQUIRED`, `REVIEWING`, `APPROVED`, `REJECTED`, `RETURN_IN_TRANSIT`, `RECEIVED`, `REFUND_PENDING`, `REFUNDED`, `CLOSED`.
- Requested lines: immutable product/SKU/price snapshot, requested quantity, requested/approved resolution.
- Customer-safe decision: label, summary, decidedAt; không có internal note/actor threshold.
- Evidence: safe label, media type, state (`READY`, `PROCESSING`, `REJECTED`) và preview kind; local không có binary URL.
- Pickup/Return Shipment projection: carrier label, window, tracking code/status; có thể null.
- Refund projection: required, method, amount, status; `PENDING` khác `COMPLETED`.
- Communications projection: sender role, display name, message, occurredAt; nguồn canonical thuộc EPIC 16.

## Supplement request

```json
{
  "message": "Tôi bổ sung thời gian nhận hàng lúc 15:45.",
  "idempotencyKey": "return-supplement-550e8400-e29b-41d4-a716-446655440000"
}
```

Không dùng operation này để approve/reject/refund/update Shipment. Local mock trả message Customer-safe đã được ghi nhận.

## File validation local

| Rule | Giá trị |
| --- | --- |
| Loại | `image/jpeg`, `image/png`, `image/webp`, `video/mp4` |
| Số lượng | tối đa 5 file |
| Kích thước | tối đa 15 MiB/file |
| Nội dung request | metadata demo; không gửi binary vào Mockoon |

Production phải dùng upload intent, giới hạn kích thước ở edge/storage, malware scan, content sniffing, authorization và evidence confirmation.

## Error semantics

| HTTP | Code | UI behavior |
| --- | --- | --- |
| 400/422 | `INVALID_REQUEST` / `VALIDATION_ERROR` | Field error; giữ form |
| 401 | `RETURN_ACCESS_REQUIRED` | Đăng nhập hoặc xác minh Guest Order lại |
| 404 | `RETURN_CASE_NOT_FOUND` | Không tồn tại hoặc ngoài ownership; không phân biệt |
| 409 | `RETURN_NOT_ELIGIBLE` / `ACTIVE_CASE_EXISTS` | Reload eligibility hoặc mở Case hiện có |
| 413 | `EVIDENCE_TOO_LARGE` | Bỏ file vượt policy |
| 429 | `RETURN_RATE_LIMITED` | Hiển thị retry time |
| 500 | `RETURN_ERROR` | Error state + retry |
| 503 | `RETURN_UPSTREAM_UNAVAILABLE` | Kiểm tra Mockoon/API Gateway + retry |

## Mock scenarios

| Scenario | Kết quả |
| --- | --- |
| mặc định | Eligibility hợp lệ, tạo Case và Case approved/pickup scheduled |
| `return-ineligible`, `return-error`, `return-slow` | Eligibility ineligible/error/loading |
| `create-existing`, `create-error`, `create-slow` | Duplicate/error/loading khi tạo |
| `case-requested`, `case-reviewing`, `case-approved`, `case-rejected`, `case-refund-pending`, `case-refunded` | Case lifecycle projection |
| `case-not-found`, `case-error`, `case-slow` | Detail error/loading |
| `supplement-rate-limited`, `supplement-error`, `supplement-slow` | Supplement failure/loading |

## Dữ liệu không public

- Staff identity nội bộ, approval threshold, internal assessment/note và Audit payload.
- Full Customer contact ngoài snapshot đã mask phù hợp.
- Packing Video gốc, storage key, signed URL lâu dài, malware result chi tiết.
- Payment provider/refund raw payload, fraud/reconciliation data và financial credentials.
- Inventory disposition nội bộ, warehouse location, Batch/Lot và cost/margin.
- Carrier credential, webhook payload và service topology.
