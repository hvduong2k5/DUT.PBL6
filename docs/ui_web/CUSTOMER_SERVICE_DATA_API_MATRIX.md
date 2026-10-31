# Customer Service MVP — Data & Mock API Matrix

## Browser/BFF API

Browser chỉ gọi Next.js BFF cùng origin. BFF allowlist method/path/body/query, kiểm tra trusted Origin cho mutation và gọi Mockoon port `4019`.

| Capability | Method | Browser BFF | Upstream mock |
| --- | --- | --- | --- |
| `SUP-C01` | GET | `/api/support/context` | `/api/v1/support/context` |
| `SUP-C02` | POST | `/api/support/tickets` | `/api/v1/support/tickets` |
| `SUP-C03` | GET | `/api/support/tickets/{ticketId}` | `/api/v1/support/tickets/{ticketId}` |
| `SUP-C04` | POST | `/api/support/tickets/{ticketId}/messages` | `/api/v1/support/tickets/{ticketId}/messages` |

## Context response

Context gồm `customerMode`, customer/contact projection, danh sách subject, priority option, Order đã được authorization và attachment policy. Browser không gửi owner/customerId, SLA deadline hay assignee.

Fixture mặc định dùng Customer `Nguyễn Văn Bảo Long`, Order `order-mock-20260926-001` / `OMA-260926-001` và subject `PRODUCT_ADVICE`.

## Create request local

```json
{
  "subjectCode": "PRODUCT_ADVICE",
  "title": "Tư vấn cách bảo quản bánh và thưởng trà",
  "message": "Tôi cần hướng dẫn bảo quản bánh Phục Linh và cách hãm Trà Sen.",
  "requestedPriority": "IMPORTANT",
  "linkedOrderId": "order-mock-20260926-001",
  "attachments": [{
    "clientReference": "attachment-1",
    "fileName": "khay-tra.jpg",
    "mediaType": "image/jpeg",
    "sizeBytes": 1468000
  }],
  "idempotencyKey": "support-ticket-550e8400-e29b-41d4-a716-446655440000"
}
```

Guest bổ sung `contact: { displayName, email, phone? }`; server không tự liên kết account/Order từ email hoặc Order number. Local BFF chuyển idempotency key trong body thành `Idempotency-Key` upstream.

Response `201` trả `ticketId`, `ticketNumber`, `status=NEW`, `ticketPath`, `createdAt` và customer-safe message.

## Ticket detail boundary

- Identity: `ticketId`, `ticketNumber`, `createdAt`, subject/title và customer-visible status.
- Requested priority và service promise là projection; không public internal priority score/escalation rule.
- Optional linked Order chỉ gồm ID/number/status label và product snapshot an toàn.
- Conversation chỉ có public messages với sender role, display name, time và customer-safe attachments.
- Internal note, queue, private assignee data, Audit log và provider payload không được trả về.

## Attachment validation local

| Rule | Giá trị |
| --- | --- |
| Loại | `image/jpeg`, `image/png`, `image/webp`, `application/pdf` |
| Số lượng | tối đa 5 file |
| Kích thước | tối đa 10 MiB/file |
| Nội dung request | metadata demo; không gửi binary vào Mockoon |

## Error semantics

| HTTP | Code | UI behavior |
| --- | --- | --- |
| 400/422 | `INVALID_REQUEST` / `VALIDATION_ERROR` | Field error; giữ form/message |
| 401 | `SUPPORT_ACCESS_REQUIRED` | Đăng nhập hoặc khôi phục Guest access |
| 403/404 | `ORDER_LINK_NOT_ALLOWED` / `TICKET_NOT_FOUND` | Không tiết lộ resource ngoài ownership |
| 409 | `DUPLICATE_TICKET` | Mở Ticket đã nhận nếu có ID an toàn |
| 413 | `ATTACHMENT_TOO_LARGE` | Bỏ file vượt policy |
| 429 | `SUPPORT_RATE_LIMITED` | Hiển thị retry time |
| 500 | `SUPPORT_ERROR` | Error state + retry |
| 503 | `SUPPORT_UPSTREAM_UNAVAILABLE` | Kiểm tra Mockoon/API Gateway + retry |

## Mock scenarios

| Scenario | Kết quả |
| --- | --- |
| mặc định | Context registered, tạo Ticket và Ticket đang xử lý |
| `context-guest`, `context-error`, `context-slow` | Guest/error/loading |
| `create-duplicate`, `create-order-forbidden`, `create-error`, `create-slow` | Create conflict/authorization/error/loading |
| `ticket-new`, `ticket-waiting`, `ticket-resolved`, `ticket-closed` | Customer-visible lifecycle |
| `ticket-not-found`, `ticket-error`, `ticket-slow` | Detail error/loading |
| `message-closed`, `message-rate-limited`, `message-error`, `message-slow` | Reply failure/loading |

## Dữ liệu không public

- Internal note, assignment history, queue/routing score, escalation rule và Audit payload.
- Full Customer profile/contact ngoài projection cần thiết.
- Storage key, long-lived signed URL, malware result chi tiết.
- Provider credential/webhook payload và service topology.
- Raw Order/Payment/Shipment/Return/Inventory data ngoài snapshot đã được cấp quyền.
