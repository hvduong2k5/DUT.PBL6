# Return & Refund MVP — Scope & Traceability

## Mục tiêu vòng triển khai

Feature cung cấp luồng hậu mãi Web D2C sau giao hàng: Customer/Guest đã chứng minh quyền sở hữu xem eligibility theo từng Order line, tạo yêu cầu đổi/trả có phạm vi rõ ràng và theo dõi Return Case. UI không tự phê duyệt, không tự hoàn tiền, không tự nhập lại tồn và không coi đề xuất của Customer là quyết định nghiệp vụ.

## Vì sao đây là feature kế tiếp

- EPIC 09 Inventory và EPIC 10 Packing là staff workflow, không có screen trong `ui_web` D2C hiện tại.
- Phần Customer của EPIC 11 Shipping đã được chiếu trong Order detail/tracking của EPIC 08; nghiệp vụ tạo vận đơn và đồng bộ 3PL thuộc Backend/Staff.
- EPIC 12 Marketplace và EPIC 13 Offline Sales không phải luồng `web-user` kế tiếp.
- EPIC 14 khớp trực tiếp D2C-017 và D2C-018, là hai screen kế tiếp sau Order Management.

## Phạm vi MVP Web D2C

| User story | Phạm vi UI | Trạng thái |
| --- | --- | --- |
| `US-RET-01` | Hủy trực tiếp Order còn eligible đã nằm trong EPIC 08; UI hậu mãi chỉ hướng dẫn handoff khi không còn direct cancel | Tái sử dụng / ranh giới |
| `US-RET-02` | Chọn Order line, số lượng, lý do và phương án mong muốn để tạo Return Case | Thực hiện |
| `US-RET-03` | Chọn ảnh/video và hiển thị metadata/validation trước khi gửi; local mock không lưu binary | Thực hiện có giới hạn |
| `US-RET-04` | Customer xem projection trạng thái, quyết định và yêu cầu bổ sung; staff review ngoài `web-user` | Customer projection |
| `US-RET-05` | Customer xem kết quả duyệt/từ chối/một phần; thao tác Manager ngoài phạm vi | Read-only projection |
| `US-RET-06` | Customer xem refund obligation/status; không thực hiện Refund trong UI này | Read-only projection |
| `US-RET-07` | Packing Video access cho staff | Giai đoạn 2 / ngoài phạm vi |

## Route UI

- `/orders/[orderId]/after-sales/request`: D2C-017, tạo Return Case từ Order đã xác minh.
- `/after-sales/[caseId]`: D2C-018, xem chi tiết Case, timeline, evidence projection, quyết định và bổ sung nội dung.
- `/orders/[orderId]`: thêm lối vào hậu mãi; direct cancellation hiện hữu vẫn không bị trộn với Return Case.

Tên folder, route và code identifier dùng tiếng Anh; nội dung hiển thị dùng tiếng Việt.

## Quyết định nghiệp vụ và bảo mật

1. `orderId`, `caseId` và Order number không phải credential; quyền Customer/Guest được kiểm tra lại ở mọi operation.
2. Eligibility, `maxReturnQty`, policy window, reason và resolution option do server quyết định từ Order line/Case hiện có.
3. Request cho phép một phần Order; tổng số lượng đang có Case/đã trả không được vượt số mua.
4. Tạo Case dùng idempotency key; retry không tạo Case, media obligation hoặc Refund trùng.
5. Customer chọn phương án mong muốn (`REPLACEMENT`, `ORIGINAL_PAYMENT_REFUND`, `LOYALTY_CREDIT`), nhưng đây không phải quyết định được duyệt.
6. Hoàn sang loyalty chỉ được thực thi sau khi có consent hợp lệ và quyết định có quyền; UI local chỉ thu nhận lựa chọn mong muốn.
7. File picker local chỉ validate type/size/count và gửi metadata demo. Production phải dùng upload intent/pre-signed URL, malware scan và evidence confirmation; không tin `fileName/mediaType/sizeBytes` từ browser như bằng chứng đã lưu.
8. Evidence chỉ ở trạng thái usable sau khi storage/scan xác nhận; không hiển thị upload đang lỗi như bằng chứng hoàn tất.
9. Case detail là Customer-safe projection; không public staff note, approval threshold, fraud signal, Packing Video gốc, provider payload hoặc Audit internals.
10. Hội thoại tổng quát thuộc EPIC 16. Operation “bổ sung hồ sơ” chỉ chuyển Customer message/evidence note sang Case/Ticket boundary, không biến EPIC 14 thành chat platform.
11. `APPROVED` không đồng nghĩa hàng đã nhận hoặc tiền đã hoàn; `REFUND_PENDING` không đồng nghĩa refund success.
12. Pickup/courier projection không trao quyền cập nhật Shipment và không public số điện thoại cá nhân không cần thiết.

## Traceability UI

| UI/state | Requirement | Mock capability |
| --- | --- | --- |
| Eligibility và policy window | `US-RET-02`, `FR-RET-03/04` | `RET-C01`, default/ineligible/error/slow |
| Chọn line/số lượng một phần | `US-RET-02`, `BR-REFUND-02` | `RET-C01/C02` |
| Chọn resolution/reason/details | `US-RET-02` | `RET-C02` |
| File metadata validation | `US-RET-03` | UI validation + mock evidence metadata |
| Duplicate create | `US-RET-02`, idempotency | `RET-C02`, cùng key trả cùng Case |
| Case status/steps/decision | `US-RET-04/05` | `RET-C03`, scenario requested/reviewing/approved/rejected/refund-pending/refunded |
| Refund boundary | `US-RET-06`, `BR-REFUND-01~04` | Projection-only, không có Customer refund command |
| Bổ sung hồ sơ | `US-RET-03/04`, boundary EPIC 16 | `RET-C04` |
| Ownership/not found/rate limit | `BR-AUTH-04` | `case-not-found`, `case-error`, `supplement-rate-limited` |

## Ngoài phạm vi

- Staff queue, thẩm định, Manager approve/reject, Warehouse receiving/inspection.
- Thực thi Refund, loyalty ledger, Inventory disposition hoặc tạo Return Shipment thật.
- Packing Video gốc, provider/carrier raw payload, internal note và Audit console.
- Binary upload/storage, virus scan và signed download URL thật; local chỉ mô phỏng metadata.
- General Customer Service Ticket/chat; chỉ hiển thị projection và gửi bổ sung Case.
- Reopen/appeal closed Case, SLA administration và policy configuration.

## Dependency cần Backend/Architecture xác nhận

- Canonical Return Case state machine, partial decision và reopen/appeal semantics.
- Eligibility window/reason matrix theo SKU, condition, delivery event và Case đang mở.
- Media upload/scan lifecycle, retention, authorization và signed URL.
- EPIC 14 ↔ EPIC 16 ownership cho conversation/supplement.
- Return Shipment/pickup, Warehouse inspection và Inventory disposition handoff.
- Refund/loyalty consent, amount ceiling, idempotency và trusted completion event.
