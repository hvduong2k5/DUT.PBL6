# Order Management MVP — Scope & Traceability

## Mục tiêu vòng triển khai

Feature cung cấp trải nghiệm Order dành cho Guest/Customer sau Checkout: xem lịch sử, xem snapshot và timeline của đúng Order, tra cứu Guest bằng bằng chứng sở hữu, đồng thời gửi yêu cầu tự hủy khi Order còn đủ điều kiện. Order status, Payment status và Shipping status luôn được trình bày riêng.

## Phạm vi MVP Web D2C

| User story | Phạm vi UI | Trạng thái |
| --- | --- | --- |
| `US-ORD-01` | Registered Customer xem danh sách Order và chi tiết snapshot lịch sử | Thực hiện |
| `US-ORD-02` | Guest/Customer xem trạng thái hiện tại và public timeline | Thực hiện |
| `US-ORD-04` | Guest/Customer tự hủy khi backend xác nhận còn eligible | Thực hiện |
| `US-ORD-03` | Gửi thông báo thay đổi trạng thái | Giai đoạn 2; thuộc EPIC 28 |
| `US-ORD-05/06` | Staff Order queue, hành động vận hành và SLA | Ngoài `web-user`; cần staff portal, RBAC và Audit |

## Route UI

- `/account/orders`: D2C-014, lịch sử của Registered Customer.
- `/orders/[orderId]/confirmation`: D2C-008, xác nhận ngắn sau Payment và dẫn sang tracking.
- `/orders/[orderId]`: D2C-015, snapshot, trạng thái Order/Payment/Shipping và public timeline.
- `/track-order`: D2C-016, tạo challenge và xác minh OTP trước khi mở Guest Order.
- Hủy Order dùng confirmation dialog trong trang chi tiết. D2C-017 là luồng đổi trả/hậu mãi sau giao hàng nên thuộc EPIC 14, không được dùng để mở rộng scope hiện tại.
- D2C-008 tái sử dụng cùng `ORD-C02`; không tạo thêm resource/lifecycle trùng với Order detail.

## Quyết định nghiệp vụ và bảo mật

1. Order list chỉ lấy từ current Customer context; browser không gửi `customerId`.
2. `orderId/orderNumber` không phải credential. Guest phải hoàn tất challenge bằng contact đã dùng đặt hàng và OTP trước khi xem dữ liệu.
3. Request challenge trả response trung tính để không xác nhận Order/contact có tồn tại.
4. Guest access dùng cookie `HttpOnly`, `SameSite=Lax`, TTL ngắn và bind đúng một Order; không đưa access token vào URL.
5. Chi tiết hiển thị immutable Order snapshot, không nối lại tên/giá/địa chỉ hiện tại từ Catalog hoặc Address Book.
6. Order, Payment và Shipping có trạng thái độc lập; không suy diễn `PAID` từ Order status hoặc browser action.
7. Public timeline chỉ chứa mô tả an toàn cho Customer; không public staff identity, warehouse, Batch/Lot, fraud/reconciliation hoặc audit internals.
8. `canCancel` và policy message do server quyết định tại thời điểm đọc, nhưng server phải kiểm tra lại khi nhận lệnh hủy.
9. Hủy dùng idempotency key. Request gửi lặp không được nhả reservation hoặc tạo refund obligation nhiều lần.
10. Hủy Order đã paid không đồng nghĩa đã refund; response chỉ ghi nhận refund obligation/handoff sang EPIC 14.
11. Order đã đi quá mốc tự hủy trả conflict và hướng dẫn hỗ trợ/đổi trả phù hợp.
12. Mock mutation giữ state trên UI trong phiên hiện tại; backend thật phải persist state và audit transition.

## Traceability UI

| UI/state | Requirement | Mock capability |
| --- | --- | --- |
| Danh sách, search/filter/year, empty | `US-ORD-01`, `FR-ORD-05` | `ORD-C01`, `orders-empty/error/slow` |
| Order snapshot sản phẩm/giá/giao nhận | `US-ORD-01`, `FR-ORD-06` | `ORD-C02` |
| Tách Order/Payment/Shipping status | `US-ORD-02`, `FR-ORD-02/08` | Detail projection và scenario theo trạng thái |
| Public status timeline | `US-ORD-02`, `BR-ORDER-05` | `ORD-C02` |
| Guest challenge/OTP | `US-ORD-02`, `FR-ORD-08` | `ORD-C04/C05` |
| Guest invalid/expired/rate limit | Ownership protection | `guest-invalid`, `guest-otp-invalid`, `guest-otp-expired`, `guest-rate-limited` |
| Tự hủy eligible | `US-ORD-04`, `FR-ORD-11/12` | `ORD-C03`, `cancel-unpaid`, `cancel-paid` |
| Không còn eligible | `US-ORD-04`, `FR-ORD-14` | `cancel-not-eligible` |
| Duplicate cancel | `US-ORD-04`, idempotency | Cùng key trả cùng kết quả; không lặp side effect |

## Ngoài phạm vi

- Staff list/queue/SLA, chỉnh priority hoặc chuyển trạng thái thủ công.
- Provider/payment verification, inventory allocation, packing, shipment creation và carrier integration.
- Notification delivery, bản đồ/realtime GPS và camera đóng gói thật.
- Return/exchange/refund approval; chỉ hiển thị handoff khi cancellation cần hoàn tiền.
- Marketplace/Offline/B2B Order và linked Guest Order workflow.
- Direct edit Order snapshot, recipient hoặc commercial total sau khi tạo.

## Dependency cần Backend/Architecture xác nhận

- Canonical Order state machine, đặc biệt nhánh COD và mốc `DELIVERED → COMPLETED`.
- Customer/Guest ownership, challenge TTL, OTP provider, rate limit và cách mở lại trên thiết bị khác.
- Cancellation matrix, deadline, required reason và handling Order đã paid/đã tạo Shipment.
- Reservation release, refund obligation và idempotent cross-service orchestration.
- Public timeline event allowlist và Customer-safe wording.
