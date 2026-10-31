# Order Management MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-26
>
> OpenAPI: `order-management-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Order Management MVP đã được người dùng duyệt cho `US-ORD-01`, `US-ORD-02` và `US-ORD-04`: xác nhận Order, lịch sử Customer, chi tiết/timeline, tra cứu Guest và tự hủy Order còn đủ điều kiện. Artefact này hoàn thành bước đề xuất API Contract từ requirement, UI, Mockoon và BFF thực tế.

Chưa thực hiện Backend/Architecture/Security/Payment/Inventory/Shipping/Product review, backend implementation, API Gateway deployment hoặc chuyển frontend khỏi Mockoon. Candidate chưa phải production contract.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_08_Order_Management.md`: rule, acceptance criteria và ranh giới Customer/Staff.
- `docs/ui_web/ORDER_MANAGEMENT_SCOPE_TRACEABILITY.md`: scope Customer MVP và traceability.
- `docs/ui_web/ORDER_MANAGEMENT_DATA_API_MATRIX.md`: payload, validation, lỗi và mock scenario.
- UI đã duyệt tại `/orders/[orderId]/confirmation`, `/account/orders`, `/orders/[orderId]` và `/track-order`.
- `mocks/mockoon/oma-order-management-mvp.json`: executable examples trên port `4017`.
- `src/lib/orders/*`, `src/services/order-service.ts` và BFF `/api/orders/*`: allowlist, validation, trusted Origin, cookie forwarding và response sanitization thực tế.

## 3. Ranh giới Browser, BFF và Gateway

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → BFF | `/api/orders[/*]` | Same-origin; không gửi Customer ID; mutation được kiểm tra Origin. |
| BFF → Gateway Candidate | `/api/v1/customers/me/orders`, `/api/v1/orders/*` | Contract đề xuất trong OpenAPI. |
| BFF → Mockoon | `ORDER_UPSTREAM_URL`, port `4017` | Local UI development trước Backend. |
| Local scenario | `mockScenario`/`X-Mock-Scenario` | Chỉ development/test; không thuộc production contract. |

Order ID và Order number không phải credential. Customer list được scope hoàn toàn từ identity. Detail/cancellation yêu cầu Customer ownership hoặc Guest grant ngắn hạn bind đúng một Order. Not-found và ngoài ownership dùng cùng public response để giảm enumeration.

## 4. Capability đề xuất

| ID | Gateway operation | Kết quả |
| --- | --- | --- |
| `ORD-C01` | `GET /customers/me/orders` | Danh sách Order của Customer hiện tại, search/filter/page có giới hạn. |
| `ORD-C02` | `GET /orders/{orderId}` | Immutable snapshot, Order/Payment/Shipping projection, timeline và cancellation policy. |
| `ORD-C03` | `POST /orders/{orderId}/cancellations` | Hủy idempotent sau khi server kiểm tra lại eligibility. |
| `ORD-C04` | `POST /orders/guest-access/challenges` | Tạo challenge trung tính, không xác nhận Order/contact tồn tại. |
| `ORD-C05` | `POST /orders/guest-access/challenges/{challengeId}/verify` | Xác minh OTP và cấp Guest grant bind một Order. |

Không public health route, mock scenario, Customer ID filter, staff command, raw provider/carrier payload, internal timeline/audit, notification delivery hoặc return/refund approval trong contract Customer này.

## 5. Quyết định Candidate

1. `customerId`, owner contact và internal source không được nhận từ browser; Customer identity quyết định list scope.
2. Product, SKU, price, recipient và address trong detail là immutable Order snapshot, không nối lại dữ liệu hiện tại từ Catalog/Profile.
3. Order, Payment và Shipping dùng status riêng; frontend không suy diễn `PAID`, `DELIVERED` hoặc refund success từ Order status.
4. Timeline là allowlist Customer-safe; không chứa staff, warehouse, Batch/Lot, fraud, reconciliation, carrier credential hoặc audit internals.
5. `canCancel`/`cancelBy` chỉ là projection để hiển thị; command luôn kiểm tra lại canonical Order, Payment, reservation và Shipping state.
6. Cancellation dùng required `Idempotency-Key`; cùng key và logical request trả kết quả cũ, khác payload trả conflict.
7. Cancellation transition, reservation-release obligation và refund handoff phải được ghi nhất quán; retry không lặp side effect.
8. `refundRequired=true` và `PENDING_HANDOFF` chỉ là nghĩa vụ hoàn tiền, không phải xác nhận đã hoàn tiền.
9. Guest challenge trả `202` trung tính và timing tương đương cho mọi combination đúng format; destination mask chỉ dựa trên input của caller.
10. OTP ngắn hạn, một lần, giới hạn attempt/resend; OTP và grant không được log, đưa vào URL hoặc cho browser JavaScript đọc.
11. Guest grant dùng HttpOnly/Secure/SameSite protection, TTL ngắn, revoke/rotate được và chỉ authorize đúng Order đã xác minh.
12. Public `404` không phân biệt Order không tồn tại với Order không thuộc caller.

## 6. Sai khác có chủ đích với local implementation

| Implementation/Mockoon local | API Candidate |
| --- | --- |
| Cancellation gửi `idempotencyKey` trong JSON body để demo executable | Dùng required `Idempotency-Key` header; body chỉ chứa reason/note. |
| List response kèm `customer.displayName/email` cho Account shell | Candidate bỏ Customer profile khỏi Order domain; UI lấy từ authenticated account context/Profile API. |
| BFF lọc/search/page trên fixture Mockoon | Backend/Gateway áp dụng filter và pagination trong Customer ownership scope. |
| Mockoon phát cookie demo và fixture OTP cố định | Production phát grant ký/mã hóa; OTP thật, ngẫu nhiên, một lần và không lưu plaintext. |
| Cookie local chưa có `Secure` để chạy HTTP localhost | Production bắt buộc transport/cookie policy theo Security review. |
| `mockScenario` và `X-Mock-Scenario` | Không tồn tại trong production contract. |
| Mock cancellation trả obligation nhưng không persist cross-service state | Backend cần transaction/outbox/saga và audit để chống side effect lặp. |

UI/BFF chưa đổi theo các khác biệt này vì contract vẫn **PROPOSED / NOT APPROVED**. Chỉ đồng bộ sau review.

## 7. State ownership và invariant

| Phần | Source of truth cần xác nhận | Invariant |
| --- | --- | --- |
| Order lifecycle | Order domain | Chỉ transition hợp lệ; history không sửa snapshot thương mại. |
| Payment status/refund | Payment/Finance domain | Order không tự suy diễn paid/refunded; refund handoff khác refund completion. |
| Reservation release | Inventory domain | Một cancellation chỉ tạo tối đa một logical release obligation. |
| Shipping status | Shipping/Fulfillment domain | Có Shipment/đã qua packing có thể làm cancellation mất eligibility. |
| Public timeline | Order read model | Chỉ event allowlist và wording an toàn cho Customer. |
| Guest ownership | Identity/Order access boundary | Grant chỉ bind một Order, TTL ngắn và không thể mở rộng scope. |

Canonical Order state machine, nhất là COD và `DELIVERED → COMPLETED`, phải do Backend/Architecture phê duyệt; enum trong candidate không tự định nghĩa quyền transition.

## 8. Open questions cần review

### Backend/Architecture

1. Canonical state machine, transition owner và optimistic concurrency/version cho Order là gì?
2. Order read model lấy Payment/Shipping projection qua synchronous query, replicated events hay composition tại Gateway?
3. Cancellation orchestration dùng local transaction, outbox/saga hay command choreography; compensating action khi dependency lỗi là gì?
4. Idempotency key scope theo actor/Order/operation, retention bao lâu và replay trả status/header nào?
5. Pagination dùng page hay cursor khi dữ liệu lớn; ordering ổn định và consistency khi có Order mới thế nào?
6. Timeline event allowlist/versioning do service nào sở hữu và xử lý event out-of-order ra sao?

### Security/Identity/Gateway

7. BFF truyền Customer identity tới Gateway bằng session, access token hay signed internal context nào?
8. Guest grant do BFF, Identity hay Order service phát; ký/mã hóa, TTL, rotation, revocation và cross-device policy thế nào?
9. Challenge enumeration protection gồm timing equalization, dummy work, IP/contact/device rate limit và abuse monitoring nào?
10. OTP provider, expiry, attempt/resend limit, hashing, replay protection và log redaction được quy định ra sao?
11. CSRF/Origin, SameSite, Secure, cookie path/domain và anti-bot control áp dụng ở lớp nào?
12. Recipient snapshot được mask khác nhau cho Customer và Guest hay cùng response; cache-control/PII retention thế nào?

### Order/Inventory/Shipping/Payment

13. Mốc cuối tự hủy là status/deadline nào; packing, Shipment creation và carrier handoff ảnh hưởng ra sao?
14. Cancellation paid/COD/unpaid có matrix riêng nào; trường hợp payment đang mơ hồ/reconciliation xử lý thế nào?
15. Reservation release và refund handoff được coi là accepted hay completed ở response; event xác nhận sau đó đi đâu?
16. Nếu Order cancel thành công nhưng Inventory/Payment tạm lỗi, retry/recovery và Customer wording là gì?
17. Shipping failed, delivery failed, expired và completed có transition/hậu mãi nào nằm ngoài cancellation?
18. Return/exchange/refund của EPIC 14 nhận handoff bằng ID/event nào mà không nhập nhằng với Order cancellation?

### Product/Frontend

19. Customer được tự hủy đến deadline cố định hay đến operational milestone; có cần countdown không?
20. Lý do nào bắt buộc note, có cho thay đổi recipient trước hủy hay luôn đi qua Checkout mới?
21. Guest có được xem toàn bộ recipient snapshot hay phải mask mạnh hơn Customer?
22. Trạng thái/timeline copy nào được đảm bảo ổn định cho UI và localization/versioning ra sao?

## 9. Checklist review theo team

- **Backend/Architecture:** resource shape, state machine, read model, consistency, idempotency, concurrency và event ordering.
- **Security/Identity/Gateway:** Customer propagation, Guest grant, OTP, enumeration, CSRF, cookie, rate limit và PII logging.
- **Order/Inventory/Shipping:** cancellation matrix, milestone, reservation release, Shipment boundary và failure recovery.
- **Payment/Finance:** paid/COD/reconciliation state, refund obligation, duplicate command và completion event.
- **Product/UX:** cancellation deadline/reason, Guest data visibility, public wording và hậu mãi handoff.
- **Frontend:** sau Approved mới đổi idempotency header, account context, types và `ORDER_UPSTREAM_URL` sang Gateway.

## 10. Điều kiện chuyển sang Approved

- Các team liên quan trả lời open question và ghi quyết định vào contract/review log.
- Canonical Order/Payment/Shipping state ownership và cancellation matrix được phê duyệt.
- Guest/Customer authorization, OTP, enumeration, CSRF, rate limit, cookie và PII policy đạt Security review.
- Idempotency, concurrency, reservation release và refund handoff có failure/recovery semantics rõ ràng.
- Contract tests bao phủ list ownership/filter/page, immutable snapshot, Customer/Guest detail, invalid/expired/rate-limited challenge, not-found anti-enumeration, unpaid/paid cancellation, ineligible cancellation, concurrent cancellation, idempotent replay và dependency failure.
- OpenAPI resolve/lint không còn lỗi blocking.
- Backend, Mockoon và frontend được đồng bộ theo contract đã phê duyệt; E2E chạy qua API Gateway.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
