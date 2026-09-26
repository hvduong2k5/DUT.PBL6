# Customer Service MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-26
>
> OpenAPI: `customer-service-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Customer Service MVP đã được người dùng duyệt cho phần Customer của `US-CS-01`: lấy context, tạo Ticket, nhận mã theo dõi, xem public conversation và gửi phản hồi. Hàng đợi, phân công/chuyển tiếp, internal note, omnichannel workspace, SLA operations và AI thuộc Giai đoạn 2 hoặc ngoài `web-user`.

Artefact này hoàn thành bước đề xuất API Contract từ requirement, UI, Mockoon và BFF thực tế. Chưa thực hiện Backend/Architecture/Security/Customer Service/Notification/Media/Product review, backend implementation, API Gateway deployment hoặc chuyển frontend khỏi Mockoon. Candidate chưa phải production contract.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_16_Customer_Service.md`: actor, rule, release scope và boundary.
- `docs/ui_web/CUSTOMER_SERVICE_SCOPE_TRACEABILITY.md`: phạm vi Web D2C và traceability.
- `docs/ui_web/CUSTOMER_SERVICE_DATA_API_MATRIX.md`: payload, validation, lỗi và mock scenario.
- UI đã duyệt tại `/support/request` và `/support/tickets/[ticketId]`.
- `mocks/mockoon/oma-customer-service-mvp.json`: executable examples trên port `4019`.
- `src/lib/support/*`, `src/services/support-service.ts` và BFF `/api/support/*`: allowlist, trusted Origin, idempotency forwarding và response sanitization thực tế.

## 3. Ranh giới Browser, BFF và Gateway

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → BFF | `/api/support/*` | Same-origin; không gửi owner, canonical priority, SLA deadline, queue hay assignee. |
| BFF → Gateway Candidate | `/api/v1/support/ticket-context`, `/api/v1/support/tickets[/*]` | Contract đề xuất trong OpenAPI. |
| BFF → Mockoon | `SUPPORT_UPSTREAM_URL`, port `4019` | Local UI development trước Backend. |
| Local scenario | `mockScenario`/`X-Mock-Scenario` | Chỉ development/test; không thuộc production contract. |

Ticket/Order ID không phải credential. Registered Customer ownership hoặc Guest grant bind đúng Ticket được kiểm tra lại ở mọi read/message operation. Missing và out-of-scope dùng cùng public response để giảm enumeration.

## 4. Capability đề xuất

| ID | Gateway operation | Kết quả |
| --- | --- | --- |
| `SUP-C01` | `GET /support/ticket-context` | Subject/priority options, attachment policy và Order đã được authorization. |
| `SUP-C02` | `POST /support/tickets` | Tạo Ticket idempotent cho Registered Customer hoặc Guest hợp lệ. |
| `SUP-C03` | `GET /support/tickets/{ticketId}` | Customer-safe Ticket và public conversation projection. |
| `SUP-C04` | `POST /support/tickets/{ticketId}/messages` | Gửi public Customer message idempotent cho Ticket đang mở trao đổi. |

Không public staff queue, assignment history, internal note, routing score, escalation rule, raw SLA policy, private employee data, provider payload, Audit console hay command sang domain liên quan.

## 5. Quyết định Candidate

1. Anonymous context không chứa Customer hoặc Order data; không suy diễn account/ownership từ email, phone hay mã Order dễ đoán.
2. Registered Customer có thể liên kết Order thuộc account; Guest chỉ liên kết khi có Order access grant riêng đã xác minh.
3. Guest create cần anti-abuse challenge. Sau create, backend có thể cấp HttpOnly Guest Ticket grant có TTL ngắn và bind đúng một Ticket.
4. `requestedPriority` chỉ là mong muốn của Customer; canonical priority, queue, SLA và escalation do server tính.
5. Create Ticket và add message dùng required `Idempotency-Key`; retry không nhân đôi Ticket/message.
6. Candidate nhận `attachmentIds` đã được Media service xác nhận; không tin filename/MIME/size do browser tự khai là tệp đã lưu hoặc an toàn.
7. Public messages tách khỏi internal notes. Customer operation không thể giả sender role, tạo staff reply hoặc gửi internal note.
8. Lưu message và phát Notification là hai kết quả khác nhau; lỗi provider không được làm mất message đã lưu hoặc đánh dấu đã giao sai.
9. Public assignee chỉ là display projection được duyệt; không public staff ID, contact, schedule hay assignment history.
10. `RESOLVED`/`CLOSED` không tự chứng minh Order/Payment/Shipment/Return đã hoàn tất. Ticket không sở hữu state của domain liên quan.
11. `canReply` là projection từ canonical status/policy; server vẫn kiểm tra lại khi nhận message.
12. Ticket detail không chứa Audit payload, internal SLA timestamps, routing/priority score, fraud signal hay provider credential.
13. Linked Order và product data là Customer-safe snapshot/read model; Ticket không trực tiếp sửa Order.
14. State enum trong contract là Customer projection, không phải staff transition API hoặc toàn bộ internal state machine.

## 6. Sai khác có chủ đích với local implementation

| Implementation/Mockoon local | API Candidate |
| --- | --- |
| Context local tại `/support/context` | Candidate đặt rõ mục đích tại `/support/ticket-context`. |
| Mutation gửi `idempotencyKey` trong JSON body | Dùng required `Idempotency-Key` header. |
| UI gửi attachment metadata demo; Mockoon không lưu binary | Candidate nhận confirmed `attachmentIds` từ Media flow được phê duyệt. |
| Create response trả `ticketPath` | Candidate không trả UI path; frontend tự tạo route từ `ticketId`. |
| Guest mock chỉ đổi context và nhập contact | Candidate cần anti-abuse challenge và Ticket-scoped Guest grant. |
| Mock priority/service promise/static assignee | Backend tính canonical priority/SLA/routing; chỉ trả customer-safe projection. |
| Mock conversation nằm trong một fixture | Production cần durable message model, pagination và Notification delivery semantics. |
| `mockScenario` và `X-Mock-Scenario` | Không tồn tại trong production contract. |

UI/BFF chưa đổi theo các khác biệt này vì contract vẫn **PROPOSED / NOT APPROVED**. Chỉ đồng bộ sau review.

## 7. State ownership và invariant

| Phần | Source of truth cần xác nhận | Invariant |
| --- | --- | --- |
| Ticket lifecycle/queue/assignment | Customer Service | Chỉ staff operation có quyền transition; Customer API chỉ đọc projection/gửi public message. |
| Conversation | Customer Service | Public message và internal note là hai visibility class riêng; không được rò rỉ qua projection. |
| SLA/service promise | Customer Service/Policy | Tính server-side theo lịch làm việc; Customer priority không override SLA. |
| Customer identity/Guest grant | Identity/Customer Service | Grant scope hẹp, TTL ngắn, revoke/rotate được; Ticket ID không cấp quyền. |
| Attachment binary/scan | Media/Object Storage | Ticket chỉ tham chiếu attachment caller-owned đã confirmed/scan. |
| Order/Payment/Shipment/Return | Domain tương ứng | Ticket chỉ tham chiếu snapshot và trigger workflow được phép; không ghi đè state nguồn. |
| Notification delivery | EPIC 28 | Message persisted khác delivery accepted/delivered/failed; retry không nhân đôi. |
| Audit | EPIC 23 | Assignment, escalation, close/reopen và sensitive access truy vết được nhưng không public. |

## 8. Open questions cần review

### Backend/Architecture

1. Canonical Ticket state machine, version/optimistic concurrency, reopen và auto-close policy là gì?
2. Conversation là aggregate trong Ticket hay resource riêng; pagination/cursor/order khi nhiều message thế nào?
3. Create duplicate detection ngoài idempotency có dùng fingerprint/time window không; false positive xử lý thế nào?
4. Idempotency scope/retention/replay status cho create/message và callback Notification là gì?
5. Linked domain context được snapshot, compose synchronous hay read model từ event; stale state biểu diễn ra sao?
6. Subject taxonomy, routing, priority và SLA calendar có version/config ownership ở đâu?

### Security/Privacy/Guest

7. Guest challenge dùng cơ chế nào; rate limit theo IP/device/contact và accessibility fallback ra sao?
8. Guest contact verification, Ticket access grant issuance, TTL, revoke/rotation và account-claim flow thế nào?
9. CSRF/Origin/SameSite/Secure, anti-enumeration và bot/abuse policy cho create/message là gì?
10. PII mask, encryption, consent, retention/deletion và data-subject request áp dụng thế nào?
11. Public assignee display được phép đến mức nào; trường hợp nhân viên cần ẩn danh ra sao?

### Media

12. Upload intent/confirm API, count/size/aggregate, MIME sniffing và malware scan contract là gì?
13. Attachment processing/rejected/quarantine, retry, retention/legal hold và signed access TTL thế nào?
14. Ai được xem/tải attachment; access có Audit, watermark hoặc download restriction nào?

### Customer Service/Product

15. Customer-visible status/wording, `canReply`, reopen và close semantics được chốt ra sao?
16. Customer requested priority có những option nào; urgent có cần bằng chứng/event date hay surcharge không?
17. SLA/service promise là estimate hay commitment; lịch làm việc, ngày lễ, pause/resume và breach wording thế nào?
18. Ticket liên quan Return Case do EPIC 14 hay EPIC 16 tạo/link; conversation owner và deep-link policy thế nào?
19. CSAT là resource nào, thời điểm mở survey, chống sửa/duplicate và anonymity thế nào?

### Notification/Operations

20. Message persisted, outbound delivery requested/accepted/delivered/failed được model và reconcile thế nào?
21. Retry provider/outbox bảo đảm at-least-once mà không gửi trùng nội dung cho Customer ra sao?
22. Queue assignment/escalation/transfer events nào cần Audit và Internal Alert?

## 9. Checklist review theo team

- **Backend/Architecture:** resource shape, state machine, message pagination, consistency, idempotency và domain handoff.
- **Security/Privacy:** Customer/Guest ownership, challenge, grant, CSRF, enumeration, rate limit, PII và retention.
- **Media:** upload/confirm/scan, authorization, signed access, quarantine và deletion.
- **Customer Service/Product:** taxonomy, priority, service promise/SLA, public wording, reopen/close và CSAT.
- **Notification:** durable message versus delivery lifecycle, outbox, retry, deduplication và provider failure.
- **Frontend:** sau Approved mới đổi attachment flow, idempotency header, context path, redirect và upstream sang Gateway.

## 10. Điều kiện chuyển sang Approved

- Các team liên quan trả lời open question và ghi quyết định vào contract/review log.
- Ticket state machine, public/internal visibility, message pagination, reopen/close và SLA policy được phê duyệt.
- Registered/Guest authorization, challenge/grant, CSRF, enumeration, abuse protection, PII và retention đạt Security review.
- Media upload/scan/access và Notification persistence/delivery có failure/retry/deduplication semantics rõ ràng.
- Contract tests bao phủ registered/guest, allowed/forbidden Order link, valid/invalid attachment, duplicate create, Ticket ownership, new/in-progress/waiting/resolved/closed, message open/closed, concurrent retry, pagination, rate limit và dependency failure.
- OpenAPI resolve/lint không còn lỗi blocking.
- Backend, Media/Notification flow, Mockoon và frontend được đồng bộ theo contract đã phê duyệt; E2E chạy qua API Gateway.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
