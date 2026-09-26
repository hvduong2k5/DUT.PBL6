# Return & Refund MVP — API Contract Candidate Review

> Trạng thái: **PROPOSED / NOT APPROVED**
>
> Phiên bản: `0.1.0-candidate`
>
> Ngày đề xuất: 2026-09-26
>
> OpenAPI: `return-refund-mvp.openapi.yaml`

## 1. Điểm dừng workflow

UI Return & Refund MVP đã được người dùng duyệt cho phần Customer của `US-RET-02/03` và Customer-safe projection của `US-RET-04~06`: kiểm tra eligibility, tạo Return Case, theo dõi trạng thái/quyết định/pickup/refund và bổ sung hồ sơ. Direct cancellation của `US-RET-01` đã nằm trong Order Management; staff review/approval/inspection/refund execution nằm ngoài `web-user`.

Artefact này hoàn thành bước đề xuất API Contract từ requirement, UI, Mockoon và BFF thực tế. Chưa thực hiện Backend/Architecture/Security/Customer Service/Payment/Inventory/Shipping/Product review, backend implementation, Media integration, API Gateway deployment hoặc chuyển frontend khỏi Mockoon. Candidate chưa phải production contract.

## 2. Nguồn hình thành contract

- `docs/01_requirements/epics/EPIC_14_Return_Refund.md`: state/rule, actor và cross-domain boundary.
- `docs/ui_web/RETURN_REFUND_SCOPE_TRACEABILITY.md`: phạm vi Web D2C và traceability.
- `docs/ui_web/RETURN_REFUND_DATA_API_MATRIX.md`: payload, validation, lỗi và mock scenario.
- UI đã duyệt tại `/orders/[orderId]/after-sales/request` và `/after-sales/[caseId]`.
- `mocks/mockoon/oma-return-refund-mvp.json`: executable examples trên port `4018`.
- `src/lib/returns/*`, `src/services/return-service.ts` và BFF `/api/returns/*`: route/body allowlist, trusted Origin, idempotency forwarding và response sanitization thực tế.

## 3. Ranh giới Browser, BFF và Gateway

| Lớp | Path/config | Trách nhiệm |
| --- | --- | --- |
| Browser → BFF | `/api/returns[/*]` | Same-origin, file metadata demo, không gửi owner/refund amount/eligibility override. |
| BFF → Gateway Candidate | `/api/v1/orders/{orderId}/return-eligibility`, `/api/v1/return-cases[/*]` | Contract đề xuất trong OpenAPI. |
| BFF → Mockoon | `RETURN_UPSTREAM_URL`, port `4018` | Local UI development trước Backend. |
| Local scenario | `mockScenario`/`X-Mock-Scenario` | Chỉ development/test; không thuộc production contract. |

Order/Case ID không phải credential. Customer identity hoặc Guest grant bind Order được kiểm tra lại ở mọi operation. Missing và out-of-scope dùng cùng public response để giảm enumeration.

## 4. Capability đề xuất

| ID | Gateway operation | Kết quả |
| --- | --- | --- |
| `RET-C01` | `GET /orders/{orderId}/return-eligibility` | Eligibility theo line, remaining quantity, reason/resolution policy và pickup snapshot. |
| `RET-C02` | `POST /return-cases` | Tạo Case idempotent từ line scope và confirmed evidence reference. |
| `RET-C03` | `GET /return-cases/{caseId}` | Customer-safe Case, decision, evidence, pickup, Refund và communications projection. |
| `RET-C04` | `POST /return-cases/{caseId}/supplements` | Bổ sung nội dung có giới hạn; không phải general chat/staff command. |

Không public media upload binary, staff queue, assessment note, approve/reject command, Warehouse inspection, Inventory disposition, Refund execution, Return Shipment command, Packing Video source, provider payload hoặc Audit console trong Customer contract này.

## 5. Quyết định Candidate

1. Eligibility tính từ canonical Order/delivery/line quantity/policy/active Case; `maxReturnQty` được kiểm tra lại khi tạo Case.
2. Case có thể chỉ chứa một phần line/quantity; tổng requested/active/returned không vượt lượng đã mua còn lại.
3. `preferredResolution` là mong muốn Customer, không phải quyết định có quyền và không tạo side effect tài chính/hàng hóa.
4. Tạo Case và supplement dùng required `Idempotency-Key`; cùng key/logical request trả kết quả cũ, khác payload trả conflict.
5. Candidate nhận `evidenceIds` đã được Media service xác nhận; không tin filename/MIME/size do browser tự khai là bằng chứng đã lưu.
6. Evidence chỉ usable khi ownership, upload completion, content sniffing và malware scan đạt; signed URL có TTL/quyền phù hợp.
7. `APPROVED` không đồng nghĩa hàng trả đã nhận, replacement đã giao hoặc Refund đã hoàn tất.
8. `REFUND_PENDING` là nghĩa vụ/attempt đang chờ. Chỉ trusted Payment result mới cho phép projection `REFUNDED`/`COMPLETED`.
9. Refund amount/method lấy từ quyết định có quyền và Payment gốc; browser không gửi amount được duyệt.
10. Loyalty credit cần consent có bằng chứng và không được đồng thời hoàn cùng giá trị về phương thức gốc.
11. Case không trực tiếp sửa Order/Payment/Inventory/Shipping; cross-domain command/event phải idempotent, audit được và có recovery semantics.
12. Supplement chỉ bổ sung context sang Case/Ticket boundary; không approve/reject/refund/update Shipment/evidence.
13. Public Case loại bỏ internal note, approval threshold, fraud signal, staff identity không cần thiết, provider payload, storage key và warehouse disposition.
14. Guest grant bind đúng Order và Customer-visible Cases, TTL ngắn, revoke/rotate được; Case ID không mở rộng scope.

## 6. Sai khác có chủ đích với local implementation

| Implementation/Mockoon local | API Candidate |
| --- | --- |
| Create gửi `idempotencyKey` trong JSON body | Dùng required `Idempotency-Key` header. |
| UI gửi `evidence` metadata demo; Mockoon không lưu binary | Candidate nhận confirmed `evidenceIds` từ Media flow đã được phê duyệt. |
| Create response trả `casePath` để redirect local | Candidate không trả UI path; frontend tự tạo route từ `caseId`. |
| Mock detail trả communications cùng Case fixture | Production projection/composition phải giữ EPIC 16 là owner của hội thoại/Ticket. |
| Mock Case/Refund/Pickup là static projection | Backend lấy canonical state hoặc read model có version/event consistency. |
| Local cookie chưa có `Secure` để chạy HTTP localhost | Production bắt buộc transport/cookie policy theo Security review. |
| `mockScenario` và `X-Mock-Scenario` | Không tồn tại trong production contract. |

UI/BFF chưa đổi theo các khác biệt này vì contract vẫn **PROPOSED / NOT APPROVED**. Chỉ đồng bộ sau review.

## 7. State ownership và invariant

| Phần | Source of truth cần xác nhận | Invariant |
| --- | --- | --- |
| Return Case lifecycle/decision | Return domain | Chỉ actor có quyền transition; final decision giữ history/Audit. |
| Order/delivery snapshot | Order/Shipping | Case liên kết snapshot lúc tạo và current projection, không sửa Order trực tiếp. |
| Evidence binary/scan | Media/Object Storage | Case chỉ tham chiếu evidence caller-owned đã confirmed/scan. |
| Conversation/Ticket | Customer Service | Supplement không biến Return domain thành chat platform. |
| Return Shipment/pickup | Shipping | Approval không tự coi pickup/return delivery hoàn tất. |
| Returned-goods disposition | Warehouse/Inventory | Chỉ inspection hợp lệ mới tạo Inventory movement, đúng một lần. |
| Refund/loyalty | Payment/Finance/Loyalty | Amount không vượt approved/refundable ceiling; completion từ nguồn tin cậy. |

State enum trong candidate là Customer projection, không tự cấp quyền transition và không thay canonical state machine của từng domain.

## 8. Open questions cần review

### Backend/Architecture

1. Canonical Return Case state machine, version/optimistic concurrency và reopen/appeal flow là gì?
2. Eligibility policy engine thuộc Return hay Product/Order; policy version được snapshot vào Case thế nào?
3. Cross-domain orchestration dùng transaction/outbox/saga hay choreography; recovery khi Shipping/Inventory/Payment lỗi ra sao?
4. Idempotency scope/retention/replay status cho create Case, supplement, pickup, inspection và Refund là gì?
5. Case detail được compose synchronous hay read model từ event; stale/current state được biểu diễn thế nào?
6. Partial approval/rejection theo line/quantity được model thế nào để không vượt purchased/remaining quantity?

### Media/Security/Privacy

7. Media upload intent/confirm API, storage owner, max count/size/aggregate, MIME sniffing và malware scan contract là gì?
8. Evidence state machine, quarantine, retry, deletion/retention/legal hold và signed URL TTL thế nào?
9. Ai được xem Packing Video/Customer evidence; access có audit và watermark/download restriction nào?
10. Guest grant, CSRF/Origin/SameSite/Secure, anti-enumeration, rate limit và bot/abuse protection áp dụng ra sao?
11. Recipient/address/communications mask khác nhau giữa Registered Customer và Guest thế nào?

### Customer Service/Product

12. EPIC 14 hay EPIC 16 sở hữu supplement/message ID, notification và Customer-visible thread projection?
13. Case final có cho bổ sung không; reopen/appeal cần Ticket mới hay transition Case hiện tại?
14. Policy window/reason matrix theo SKU, loại lỗi, delivery event, hàng thực phẩm và quà cá nhân hóa là gì?
15. Customer wording nào ổn định; SLA chỉ là estimate hay commitment và do domain nào tính?

### Shipping/Warehouse/Inventory

16. Pickup/return Shipment được tạo khi approve hay sau Customer confirmation; split package xử lý thế nào?
17. Warehouse inspection model condition, quantity, Batch/Lot, disposition và evidence thế nào?
18. Inventory movement chỉ được tạo ở mốc nào; duplicate/out-of-order inspection event xử lý ra sao?
19. Replacement Shipment liên kết Case/Order line ra sao mà không làm sai Order gốc?

### Payment/Finance/Loyalty

20. Refundable ceiling gồm item, shipping, discount, promotion allocation, prior Refund và fee thế nào?
21. Ai có quyền approve/execute Refund, dual control/threshold và Audit được áp dụng ra sao?
22. Original-payment Refund pending/failed/late success được reconcile thế nào để không hoàn trùng?
23. Loyalty consent format/version/revocation và cross-ledger idempotency được xử lý ra sao?

## 9. Checklist review theo team

- **Backend/Architecture:** resource shape, state machine, policy version, read model, consistency, idempotency và recovery.
- **Security/Media:** Customer/Guest ownership, upload/scan, signed access, CSRF, enumeration, rate limit, PII và evidence retention.
- **Customer Service/Product:** Ticket/supplement ownership, Customer wording, SLA, reopen/appeal và policy matrix.
- **Shipping/Warehouse/Inventory:** pickup/return Shipment, inspection, disposition, movement và replacement boundary.
- **Payment/Finance/Loyalty:** refundable ceiling, approval authority, trusted completion, reconciliation và consent.
- **Frontend:** sau Approved mới đổi metadata sang Media flow, idempotency header, redirect path và `RETURN_UPSTREAM_URL` sang Gateway.

## 10. Điều kiện chuyển sang Approved

- Các team liên quan trả lời open question và ghi quyết định vào contract/review log.
- Case state machine, eligibility/policy version và line-level partial decision được phê duyệt.
- Customer/Guest authorization, media upload/scan/access, CSRF, enumeration, rate limit, PII và retention đạt Security review.
- Pickup, inspection, Inventory disposition, replacement và Refund orchestration có idempotency/failure/recovery semantics rõ ràng.
- Contract tests bao phủ ownership, eligible/ineligible/expired, partial quantity, active duplicate Case, evidence processing/rejected, requested/reviewing/approved/rejected, pickup, refund pending/completed/failed, supplement open/final, concurrent create, idempotent replay và dependency failure.
- OpenAPI resolve/lint không còn lỗi blocking.
- Backend, Media flow, Mockoon và frontend được đồng bộ theo contract đã phê duyệt; E2E chạy qua API Gateway.

Cho đến khi hoàn tất, version giữ `0.1.0-candidate` và trạng thái **PROPOSED / NOT APPROVED**.
