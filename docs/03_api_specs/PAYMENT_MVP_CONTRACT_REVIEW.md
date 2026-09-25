# Payment MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-26
>
> OpenAPI: `payment-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Payment MVP đã được người dùng duyệt cho `US-PAY-01~03`: chuyển khoản/QR, COD và xem trạng thái Payment. Artefact này hoàn thành bước đề xuất API Contract từ requirement, UI, Mockoon và BFF thực tế.

Chưa thực hiện Backend/Architecture/Order/Finance/Security/Product review, provider integration, webhook/statement ingestion, staff reconciliation, API Gateway deployment hoặc chuyển frontend khỏi Mockoon. Candidate chưa phải production contract.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_07_Payment.md`: rule Payment và acceptance criteria.
- `docs/ui_web/PAYMENT_SCOPE_TRACEABILITY.md`: scope Customer MVP và boundary với Staff/Order.
- `docs/ui_web/PAYMENT_DATA_API_MATRIX.md`: payload, trạng thái, lỗi và mock scenario.
- UI `/payment/[orderId]` đã duyệt và integration Checkout → Payment.
- `mocks/mockoon/oma-payment-mvp.json`: executable examples trên port `4016`.
- `src/lib/payment/*`, `src/services/payment-service.ts`, BFF `/api/payments/*`: validation, signed access, response sanitization và retry thực tế.

## 3. Ranh giới Browser, BFF và Gateway

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → BFF | `/api/payments/orders/{orderId}[/*]` | Same-origin, không gửi amount/currency/method, không đọc signed HttpOnly access. |
| BFF → Gateway Candidate | `/api/v1/payments/orders/{orderId}[/*]` | Contract đề xuất trong OpenAPI. |
| BFF → Mockoon | `PAYMENT_UPSTREAM_URL`, port `4016` | Local UI development trước Backend. |
| Local scenario | `mockScenario`/`X-Mock-Scenario` | Chỉ development/test; không thuộc production contract. |

Order ID không phải credential. Registered Customer phải được scope theo identity; Guest dùng capability ngắn hạn, bind đúng một Order. Not-found và unauthorized ownership dùng cùng public response để tránh enumeration.

## 4. Capability đề xuất

| ID | Gateway operation | Kết quả |
| --- | --- | --- |
| `PAY-C01` | `GET /payments/orders/{orderId}` | Lấy method, amount, instructions và trạng thái từ trusted server state. |
| `PAY-C02` | `GET /payments/orders/{orderId}/status` | Đọc trạng thái đã được Payment domain chấp nhận; bản thân request không tạo success. |
| `PAY-C03` | `POST /payments/orders/{orderId}/attempts` | Tạo bank-transfer attempt mới nếu server đánh giá Order còn eligible. |

Không public health route, mock scenario, webhook/provider callback, raw provider payload, staff reconciliation command hoặc refund trong contract Customer này.

## 5. Quyết định Candidate

1. Method, amount, currency, Order reference và eligibility luôn lấy từ server; browser không được gửi hoặc override.
2. `SUCCEEDED/PAID` chỉ được ghi sau bằng chứng hợp lệ từ nguồn tin cậy và khớp Order reference, amount, currency.
3. Browser return, reload, polling và nút “kiểm tra trạng thái” không phải bằng chứng thanh toán.
4. Notification/provider event phải idempotent; duplicate hợp lệ không tạo side effect thứ hai.
5. Sai amount/currency/reference chuyển `REQUIRES_RECONCILIATION`, không tự `PAID`.
6. COD tạo `COD_PENDING_COLLECTION/PAYMENT_ON_DELIVERY`; không có QR và không `PAID` trước khi thu tiền hợp lệ.
7. Retry chỉ dành cho `BANK_TRANSFER`, Order chưa paid, không có giao dịch mơ hồ và còn đáp ứng Order/reservation policy.
8. POST retry dùng `Idempotency-Key` header và không có business request body.
9. Bank instructions/QR do server hoặc provider sinh; browser không tự ghép payload QR.
10. Public response không chứa provider secret, webhook signature, raw bank payload, customer financial data, reconciliation note hoặc audit internals.

## 6. Sai khác có chủ đích với local implementation

| Implementation/Mockoon local | API Candidate |
| --- | --- |
| Retry gửi `idempotencyKey` trong JSON body | Dùng required `Idempotency-Key` header; không có body. |
| Signed cookie chứa Order/method/amount để bảo vệ mock stateless | Production resolve lại canonical Order/Payment state; capability chỉ định danh và authorize. |
| QR được vẽ bằng CSS, `qrPayloadType=DEMO_ONLY` | Gateway trả `qrCodeImageUrl` do server/provider kiểm soát. |
| Demo account được mask và không thể chuyển tiền | Production trả official merchant receiving account theo Security/Finance approval. |
| Local luôn trả `expiresAt`, kể cả COD | Candidate dùng `expiresAt=null` cho COD/non-expiring state. |
| `mockScenario` và `X-Mock-Scenario` | Không tồn tại trong production contract. |
| Mock fixture dùng Order/payment ID và thời gian cố định | Không phải production semantic. |

UI/BFF chưa đổi theo các khác biệt này vì contract vẫn **PROPOSED / NOT APPROVED**. Chỉ đồng bộ sau review.

## 7. Trạng thái và invariant

| Payment | Order projection | Invariant |
| --- | --- | --- |
| `PENDING` | `PENDING_PAYMENT` | Chưa có verified success. |
| `SUCCEEDED` | `PAID` | Chỉ sau trusted verification khớp đủ dữ liệu. |
| `FAILED` | `PENDING_PAYMENT` | Không paid; retry phụ thuộc server eligibility. |
| `EXPIRED` | `PAYMENT_EXPIRED` | Không paid; Order/Inventory quyết định release/retry. |
| `REQUIRES_RECONCILIATION` | `PAYMENT_REVIEW` | Không tự paid; staff flow nằm ngoài Customer contract. |
| `COD_PENDING_COLLECTION` | `PAYMENT_ON_DELIVERY` | Nghĩa vụ thu COD, chưa paid. |

Payment không sở hữu toàn bộ Order lifecycle. Order service phải định nghĩa canonical transition và event/command boundary; response này chỉ là projection phục vụ màn hình Payment.

## 8. Open questions cần review

### Backend/Architecture

1. Customer/Guest access token nằm ở BFF cookie, Gateway token hay Order access grant; TTL/revocation/rotation thế nào?
2. Có cần tách summary và status endpoint hay dùng một resource cùng ETag/conditional polling?
3. Retry concurrency, idempotency scope, retention và replay HTTP status được triển khai thế nào?
4. Payment ↔ Order consistency dùng transaction, outbox/saga hay event choreography; xử lý event out-of-order ra sao?
5. Sau timeout của retry request, client resolve outcome bằng replay hay endpoint lookup nào?

### Payment/Finance/Provider

6. Nguồn tin cậy gồm webhook ký số, statement feed hay manual reconciliation nào; thứ tự ưu tiên ra sao?
7. Signature verification, replay window, provider event ID và raw payload retention được quy định thế nào?
8. Amount/currency/reference matching có tolerance/fee rule nào hay yêu cầu exact match?
9. QR format/provider, URL TTL, account rotation và beneficiary display do service nào sở hữu?
10. Khi customer chuyển lặp hoặc chuyển sau expiry, state/reconciliation/refund handoff thế nào?
11. COD được đánh dấu collected bởi Shipping/Order event nào; cash variance đi đâu?

### Product/Order/Inventory

12. Payment instruction TTL và inventory reservation TTL có giống nhau không?
13. Retry có gia hạn reservation hay chỉ dùng TTL còn lại; giới hạn số attempt là bao nhiêu?
14. `FAILED/EXPIRED` có cho retry trực tiếp hay buộc tạo lại Checkout/Order?
15. Status/message nào được phép hiển thị cho Customer mà không làm lộ fraud/reconciliation logic?

### Security/Privacy/Gateway

16. CSRF/Origin, polling rate limit, anti-enumeration và bot protection áp dụng thế nào?
17. Official receiving account/QR có cần ký, cache-control hoặc CSP riêng không?
18. Log redaction, encryption, retention và access control cho financial event/audit là gì?
19. Staff reconciliation của `US-PAY-04/05` dùng RBAC/dual control/audit nào và có được phép trực tiếp set `PAID` không?

## 9. Checklist review theo team

- **Backend/Architecture:** resource shape, identity propagation, status ownership, consistency, idempotency và concurrency.
- **Payment/Finance:** provider verification, matching, reconciliation, QR/account source, COD collection và duplicate money handling.
- **Order/Inventory:** transition `PAID`, reservation expiry/retry và event contract.
- **Security/Gateway:** Guest capability, CSRF, anti-enumeration, rate limit, signature, secrets, PII/financial logging.
- **Product/UX:** countdown, retry policy, Customer-safe copy, COD semantics và handling mismatch/late payment.
- **Frontend:** sau Approved mới đổi request/header/types, QR production và `PAYMENT_UPSTREAM_URL` sang Gateway.

## 10. Điều kiện chuyển sang Approved

- Tất cả team liên quan xác nhận các open question và ghi quyết định vào contract/review log.
- Canonical Payment/Order state machine, source-of-truth và trusted verification được phê duyệt.
- Guest/Customer ownership, CSRF, anti-enumeration, rate limit, provider signature và audit đạt Security review.
- Contract tests bao phủ bank pending/success/fail/expiry/mismatch, COD, duplicate provider event, wrong amount/currency/reference, unauthorized Order, retry eligible/ineligible, concurrent retry, idempotent replay và dependency failure.
- OpenAPI resolve/lint không còn lỗi blocking.
- Backend, Mockoon và frontend được đồng bộ theo contract đã phê duyệt; E2E chạy qua API Gateway.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
