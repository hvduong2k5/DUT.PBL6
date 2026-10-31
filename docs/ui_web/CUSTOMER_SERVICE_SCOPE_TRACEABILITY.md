# Customer Service MVP — Scope & Traceability

## Mục tiêu vòng triển khai

Feature cung cấp đầu mối hỗ trợ chính thức cho Web D2C: Customer hoặc Guest nhập yêu cầu, tùy chọn liên kết Order đã được phép, nhận mã Ticket duy nhất, xem nội dung trao đổi công khai và gửi phản hồi bổ sung. Ticket giữ hội thoại và cam kết phản hồi; không sở hữu quyết định đổi/trả, Refund, Order, Payment hay Shipment.

## Vì sao đây là feature kế tiếp

- EPIC 14 Return & Refund và hai screen D2C-017/018 đã hoàn tất.
- EPIC 16 khớp trực tiếp hai screen kế tiếp D2C-019/020.
- `US-CS-01` là Must Have của MVP; `US-CS-02` đến `US-CS-06` là workspace nhân viên Giai đoạn 2 và ngoài `web-user`.

## Phạm vi MVP Web D2C

| User story | Phạm vi UI | Trạng thái |
| --- | --- | --- |
| `US-CS-01` | Tạo Ticket, nhận mã theo dõi, xem Ticket của mình và bổ sung trao đổi | Thực hiện |
| `US-CS-02` | Hàng đợi, lọc, nhận/chuyển Ticket | Giai đoạn 2 / staff-only |
| `US-CS-03` | Customer xem public conversation; internal note và omnichannel workspace ngoài phạm vi | Customer projection |
| `US-CS-04` | Chỉ liên kết snapshot Order đã được authorization; không tổng hợp workspace giao dịch | Giới hạn MVP |
| `US-CS-05/06` | SLA alert nội bộ và phối hợp tồn kho | Giai đoạn 2 / ngoài phạm vi |

## Route UI

- `/support/request`: D2C-019, tạo yêu cầu hỗ trợ.
- `/support/tickets/[ticketId]`: D2C-020, xem chi tiết và gửi phản hồi công khai.

Tên folder, route và code identifier dùng tiếng Anh; nội dung hiển thị dùng tiếng Việt. Header/Footer tái sử dụng component chung.

## Quyết định nghiệp vụ và bảo mật

1. `ticketId`, Ticket number, email và Order number không phải credential; quyền xem/gửi phản hồi được kiểm tra lại ở mọi operation.
2. Registered Customer chỉ được liên kết Order thuộc tài khoản. Guest chỉ liên kết sau verification riêng; mock mặc định không suy diễn quyền từ Order number.
3. Tạo Ticket và gửi message dùng idempotency key; retry không nhân đôi Ticket/message.
4. Priority khách chọn là mức độ mong muốn, không phải SLA/priority nội bộ cuối cùng.
5. UI không public internal note, queue, assignee identity nội bộ, escalation rule, fraud signal hay Audit payload.
6. Attachment local chỉ validate metadata. Production phải dùng upload intent/pre-signed URL, content sniffing, malware scan và attachment confirmation.
7. Ticket liên quan đổi/trả chỉ giữ hội thoại và liên kết Return Case; EPIC 14 vẫn sở hữu quyết định/refund state machine.
8. SLA hiển thị là customer-safe service promise, không để browser tự quyết định vi phạm SLA.
9. `RESOLVED` không tự đồng nghĩa nghiệp vụ liên quan đã hoàn tất; Customer có thể gửi phản hồi theo chính sách reopen.
10. Dữ liệu mock dùng Customer đã đăng nhập để bám Stitch; API shape vẫn hỗ trợ Guest contact.

## Traceability UI

| UI/state | Requirement | Mock capability |
| --- | --- | --- |
| Context khách, chủ đề, priority, Order được phép | `US-CS-01`, `BR-AUTH-04` | `SUP-C01` |
| Tạo Ticket + mã duy nhất | `US-CS-01`, `FR-15` | `SUP-C02` |
| Validation / duplicate retry | `US-CS-01` | UI validation + idempotency |
| Chi tiết và public conversation | `US-CS-01`, boundary `US-CS-03` | `SUP-C03` |
| Gửi phản hồi bổ sung | `US-CS-01` | `SUP-C04` |
| Ownership/not found/rate limit | `BR-AUTH-04` | các scenario lỗi tương ứng |

## Ngoài phạm vi

- Staff queue, assignment, transfer, internal note, escalation và SLA operations.
- Omnichannel ingestion/delivery status thật; Email/SMS/Push/Zalo thuộc EPIC 28.
- Thay đổi Order/Payment/Shipment/Return Case từ Ticket.
- Binary upload/storage, malware scan và signed download URL thật.
- AI summary/draft/knowledge search, CSAT persistence và hotline integration.

## Dependency cần Backend/Architecture xác nhận

- Guest verification/access-session model và thời hạn truy cập Ticket.
- Canonical Ticket state machine, reopen/close policy và customer-visible status.
- Subject/priority taxonomy, service promise và SLA ownership.
- Attachment upload/scan/retention/authorization lifecycle.
- Customer Service ↔ Return/Order/Notification boundary và event ownership.
- Idempotency retention, concurrency/versioning cho message và privacy/audit policy.
