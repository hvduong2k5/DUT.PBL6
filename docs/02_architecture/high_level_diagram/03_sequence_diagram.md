# BỘ SƠ ĐỒ TUẦN TỰ LUỒNG NGHIỆP VỤ & SAGA ĐỀN BÙ (SEQUENCE DIAGRAMS)

## 1. Vị Trí & Vai Trò Trong Tài Liệu HLD
- **Cấp độ kiến trúc:** Luồng tương tác nghiệp vụ động (Sequence Diagrams).
- **Vị trí trong HLD:** Đặt tại **Chương 4 (Mục 4.2 - Luồng Nghiệp Vụ Lõi Critical Path)** kết hợp **Chương 7 (Story Walkthrough)** — Trả lời câu hỏi: *Từng thông điệp được gửi đi theo thứ tự thời gian như thế nào? Bước nào bắt buộc phải chờ (sync gRPC < 20ms), bước nào chuyển giao bất đồng bộ (async Kafka), và khi xảy ra lỗi/timeout thì ai kích hoạt đền bù?*
- **Chiến lược phân rã (Modular De-cluttering):** Thay vì gộp chung toàn bộ từ lúc khách bấm nút, thanh toán, đóng gói, giao hàng, kiểm toán và đền bù vào một sơ đồ khổng lồ gây chồng chéo các khối `par`, `alt`, `rect`, tài liệu phân rã thành **4 sơ đồ tuần tự chuyên biệt, mạch lạc và độc lập**:
  - **Sơ đồ 3.1 (Critical Path Checkout):** Pha đồng bộ thời gian thực (< 150ms) từ lúc khách bấm "Đặt mua" đến khi nhận mã VietQR.
  - **Sơ đồ 3.2 (Thanh Toán & Hậu Kỳ Bất Đồng Bộ):** Pha nhận tiền VietQR qua Webhook và kích hoạt chuỗi cung ứng đóng gói seal, giao hàng 3PL qua Kafka.
  - **Sơ đồ 3.3 (Kịch Bản Đền Bù Phân Tán - Centralized Compensation):** Pha xử lý ngoại lệ khi quá hạn 15 phút hoặc lỗi mạng, minh chứng luật Mục 3.4.
  - **Sơ đồ 3.4 (Hành Trình Đơn Sàn Marketplace Inbound Saga):** Pha nhập đơn từ Shopee/TikTok qua ACL, khóa tồn dùng chung và xử lý khi hết hàng.

---

## 2. Quy Chuẩn Ký Hiệu Áp Dụng (Tuân Thủ Tuyệt Đối `regulation.md`)

| Ký Hiệu | Bản Chất Kỹ Thuật | Căn Cứ Tại `regulation.md` |
| :---: | :--- | :--- |
| `->>` | **gRPC Synchronous Command:** Đường liền, mũi tên đặc. Lệnh trên đường găng (Critical Path), chặn luồng chờ phản hồi. | Dòng 5: `->> (đường liền, mũi tên đặc)` |
| `-->>` | **Phản hồi gRPC:** Đường đứt, mũi tên đặc. Kết quả xác nhận hoặc mã lỗi từ Provider. | Dòng 6: `-->> (đường đứt, mũi tên đặc)` |
| `-)` | **Kafka Domain Event:** Đường liền, mũi tên hở. Sự kiện bất đồng bộ phát tán (Fire-and-Forget). | Dòng 7: `-) (đường liền, mũi tên hở)` |
| `box "External Systems"` | **Phân vùng hệ thống ngoài:** Đóng khung riêng các đối tác ngoại vi (Cổng Ngân hàng VietQR, Sàn Shopee). | Dòng 8: `box "External Systems" để tách vùng` |
| `rect rgb(255,230,230)` | **Audit Plane (Bất khả xâm):** Tô nền đỏ nhạt bao bọc toàn bộ tương tác kiểm toán. | Dòng 9: `rect rgb(255,230,230) ... end` |
| `-x` hoặc `--x` | **Lỗi / Timeout / Đền bù:** Đường nét kèm dấu gạch chéo X đỏ, biểu thị sự cố mạng hoặc quá hạn thanh toán. | Dòng 10: `-x hoặc --x (dấu X built-in)` |

---

## 3. Sơ Đồ 3.1: Critical Path Checkout & Khóa Tồn Kho gRPC (Thời Gian Thực < 150ms)
*Mục đích: Đặc tả giai đoạn sống còn trên đường găng, chỉ có các cuộc gọi gRPC đồng bộ có Circuit Breaker (Timeout 2.0s).*

```mermaid
sequenceDiagram
    autonumber
    actor Customer as Khách Hàng (D2C)
    participant Gateway as API Gateway (Kong)
    participant OrderSaga as MS-04: Order Saga Orchestrator
    participant Inventory as MS-01: Inventory Service
    participant Catalog as MS-05: Catalog Service
    participant Promotion as MS-07: Promotion Service

    Note over Customer,Promotion: PHA ĐỒNG BỘ THỜI GIAN THỰC (CRITICAL PATH - LATENCY < 150ms)
    Customer->>Gateway: POST /api/v1/checkout (JWT Token, Cart Items, Address)
    Gateway->>Gateway: Validate JWT qua Cached JWKS (0.05ms) + Sanitize Headers
    Gateway->>OrderSaga: Forward Request kèm X-User-Id: CUST-9876, traceparent
    OrderSaga->>OrderSaga: Local DB Tx: Tạo Order DRAFT + Saga State IN_PROGRESS

    par Thẩm Tra Đồng Thời Qua gRPC (Critical Path Timeout 2.0s)
        OrderSaga->>Inventory: ReserveStock(order_id, items, ttl=15m)
        Inventory->>Inventory: DB Tx (SELECT FOR UPDATE): Khóa tồn 15m trong stock_reservations
        Inventory-->>OrderSaga: Response: RESERVATION_STATUS_SUCCESS
    and
        OrderSaga->>Catalog: ValidatePriceAndSKU(items)
        Catalog-->>OrderSaga: Response: VALID (Đúng giá niêm yết kẹo OCOP)
    and
        OrderSaga->>Promotion: ValidateVoucher(voucher_code, user_id)
        Promotion-->>OrderSaga: Response: VALID (Đủ điều kiện giảm 10%)
    end

    OrderSaga->>OrderSaga: Local DB Tx: Update status = PENDING_PAYMENT, Sinh VietQR động
    OrderSaga-->>Gateway: Trả về Order Summary + Mã VietQR (345.000 VND)
    Gateway-->>Customer: Hiển thị màn hình Quét mã VietQR (Đồng hồ đếm ngược 15:00)
```

---

## 4. Sơ Đồ 3.2: Thanh Toán VietQR & Chuỗi Cung Ứng Hậu Kỳ Bất Đồng Bộ (Happy Path)
*Mục đích: Đặc tả giai đoạn sau khi nhận được tiền từ ngân hàng, toàn bộ quy trình đóng gói seal, giao hàng 3PL và kiểm toán chạy qua Kafka.*

```mermaid
sequenceDiagram
    autonumber
    actor Customer as Khách Hàng (D2C)
    box rgb(240, 248, 255) "External Systems (Hệ Thống Ngoại Vi)"
        participant BankVietQR as Cổng VietQR / Ngân Hàng
    end
    participant Gateway as API Gateway (Kong)
    participant OrderSaga as MS-04: Order Saga Orchestrator
    participant Kafka as Apache Kafka (Unified Broker)
    participant Fulfillment as MS-02: Fulfillment Service
    participant Shipping as MS-12: Shipping Service
    participant Notif as MS-17: Notification Service
    participant Audit as MS-18: Audit Service

    Note over Customer,BankVietQR: Khách quét mã VietQR qua App Ngân hàng
    Customer->>BankVietQR: Chuyển khoản 345.000 VND
    BankVietQR->>Gateway: Webhook: POST /api/v1/payments/vietqr/callback (HMAC-SHA256)
    Gateway->>OrderSaga: Forward Webhook Payment
    OrderSaga->>OrderSaga: Thẩm định chữ ký HMAC khớp số tiền
    OrderSaga->>OrderSaga: Local DB Tx: Order status = PAID, Saga State = COMPLETED
    OrderSaga->>OrderSaga: Ghi Outbox: OrderPaidEvent (không chứa PII)
    OrderSaga-->>Gateway: HTTP 200 OK
    Gateway-->>BankVietQR: HTTP 200 OK (Xác nhận nhận tiền)
    OrderSaga-->>Customer: WebSocket Push: Đã nhận thanh toán thành công!

    Note over OrderSaga,Shipping: CHUỖI CUNG ỨNG BẤT ĐỒNG BỘ QUA KAFKA (FIRE-AND-FORGET)
    OrderSaga-)Kafka: Outbox Poller phát: OrderPaidEvent (topic: order.events.v1)

    par Các dịch vụ hạ nguồn tiêu thụ độc lập (Non-blocking)
        Kafka-)Fulfillment: Consume OrderPaidEvent -> Tạo Picking Task tại xưởng Huế
        Kafka-)Notif: Consume OrderPaidEvent -> Gửi tin nhắn Zalo ZNS / SMS xác nhận đơn
    end

    Note over Fulfillment,Shipping: Thợ xưởng nhặt kẹo, quét mã, quay video dán seal niêm phong
    Fulfillment-)Kafka: Outbox Poller phát: PackingCompletedEvent (có video seal URL)
    Kafka-)Shipping: Consume PackingCompletedEvent -> Tự động gọi API 3PL tạo vận đơn giao hàng

    rect rgb(255, 230, 230)
        Note over OrderSaga,Audit: AUDIT PLANE (BẤT KHẢ XÂM - HOÀN TOÀN TÁCH RỜI BUSINESS)
        OrderSaga-)Kafka: Bắn Audit Record sang topic: audit.events.v1 (Action: ORDER_PAID)
        Kafka-)Audit: Consume audit.events.v1
        Audit->>Audit: Tính toán Entity Hash Chain: Hash_N = SHA256(Hash_{N-1} + Payload)
        Audit->>Audit: Ghi vào CSDL Append-Only (Định kỳ neo Merkle Root lên AWS S3 WORM)
    end
```

---

## 5. Sơ Đồ 3.3: Ngoại Lệ Quá Hạn 15 Phút & Kịch Bản Đền Bù Phân Tán (Centralized Compensation)
*Mục đích: Làm sáng tỏ nguyên tắc bất di bất dịch của Mục 3.4 — Saga Orchestrator là thực thể duy nhất phát hiện sự cố và phát lệnh đền bù giải phóng tồn kho.*

```mermaid
sequenceDiagram
    autonumber
    actor Customer as Khách Hàng (D2C)
    participant OrderSaga as MS-04: Order Saga Orchestrator
    participant Inventory as MS-01: Inventory Service
    participant Kafka as Apache Kafka (Unified Broker)

    Note over Customer,Inventory: TÌNH HUỐNG: ĐỒNG HỒ 15:00 HẾT HẠN HOẶC KHÁCH BẤM HỦY ĐƠN
    Customer-xOrderSaga: Payment Timeout / User Cancelled (--x)
    OrderSaga->>OrderSaga: Saga Timeout Watcher quét thấy đơn PENDING_PAYMENT quá 15 phút
    OrderSaga->>OrderSaga: Đánh dấu Saga State = COMPENSATING

    Note over OrderSaga,Inventory: Saga Orchestrator độc quyền kích hoạt đền bù tập trung (Mục 3.4)
    OrderSaga->>Inventory: ReleaseReservation(order_id, reason="PAYMENT_TIMEOUT")
    Inventory->>Inventory: DB Tx: Xóa bản ghi stock_reservations, giải phóng tồn khả dụng
    Inventory-->>OrderSaga: Response: RELEASED_SUCCESS

    OrderSaga->>OrderSaga: Local DB Tx: Order status = CANCELLED_TIMEOUT, Saga = SAGA_COMPENSATED
    OrderSaga-)Kafka: Publish: OrderCancelledEvent (topic: order.events.v1)
    
    Note over OrderSaga,Inventory: Dữ liệu nhất quán 100%, không bị rò rỉ tồn kho mồ côi!
```

---

## 6. Sơ Đồ 3.4: Hành Trình Nhập Đơn Sàn Marketplace Inbound Saga (Shopee / TikTok)
*Mục đích: Bắt lỗi mâu thuẫn kiến trúc đã phát hiện — minh chứng Channel Service chỉ làm ACL phát event, Order Saga mới là bên gọi ReserveStock.*

```mermaid
sequenceDiagram
    autonumber
    box rgb(240, 248, 255) "External Platforms"
        participant Shopee as Sàn Shopee Mall
    end
    participant Gateway as API Gateway (Kong)
    participant Channel as MS-13: Channel Service (ACL)
    participant Kafka as Apache Kafka
    participant OrderSaga as MS-04: Order Saga Orchestrator
    participant Inventory as MS-01: Inventory Service

    Note over Shopee,Inventory: ĐƠN HÀNG PHÁT SINH TỪ SHOPEE ĐỔ VỀ XƯỞNG O MẠ
    Shopee->>Gateway: POST /api/v1/webhooks/marketplace/shopee (Payload sàn)
    Gateway->>Channel: Forward Webhook (Verify Secret Signature)
    Channel->>Channel: ACL Normalize: Shopee Item 9928172 -> SKU nội bộ MX-GION-500G
    Channel->>Channel: Local DB Tx: Lưu marketplace_orders + Ghi Outbox Event
    Channel-->>Gateway: HTTP 200 OK (trong vòng 50ms)
    Gateway-->>Shopee: HTTP 200 OK

    Note over Channel,OrderSaga: Chuyển giao bất đồng bộ vào hệ thống lõi
    Channel-)Kafka: Publish: MarketplaceOrderImportedEvent (topic: channel.events.v1)
    Kafka-)OrderSaga: Consume: MarketplaceOrderImportedEvent
    OrderSaga->>OrderSaga: Local DB Tx: Tạo đơn IMPORTED_PENDING_STOCK, Saga = IN_PROGRESS

    alt Kho còn kẹo (Happy Path)
        OrderSaga->>Inventory: ReserveStock(order_id, items, ttl=15m)
        Inventory->>Inventory: DB Tx (SELECT FOR UPDATE): Khóa tồn kho dùng chung OK
        Inventory-->>OrderSaga: Response: RESERVATION_STATUS_SUCCESS
        OrderSaga->>OrderSaga: Order status = PAID / CONFIRMED, Saga = COMPLETED
        OrderSaga-)Kafka: Publish: OrderPaidEvent -> Chuyển thợ xưởng đóng gói
    else Kho hết kẹo dùng chung (Failure Path - Bán vượt tồn)
        OrderSaga->>Inventory: ReserveStock(order_id, items, ttl=15m)
        Inventory-->>OrderSaga: Response: INSUFFICIENT_STOCK (available_qty = 2 < 5)
        OrderSaga->>OrderSaga: Order status = RESERVATION_FAILED, Saga = FAILED_OUT_OF_STOCK
        OrderSaga-)Kafka: Publish: MarketplaceOrderStockFailedEvent
        Kafka-)Channel: Consume: MarketplaceOrderStockFailedEvent
        Channel->>Shopee: Partner Open API: Hủy đơn sàn hoặc báo CSKH xử lý ngoại lệ
    end
```

---

## 7. Tổng Kết So Sánh: Trực Quan Hóa Sau Phân Rã
- **Không còn tình trạng "ma trận mũi tên":** Mỗi sơ đồ chỉ biểu đạt đúng một phân đoạn nghiệp vụ rõ ràng với từ 4 đến 6 participants.
- **Tuân thủ triệt để `regulation.md`:** 
  - gRPC đồng bộ thể hiện bằng `->>` và `-->>`.
  - Kafka bất đồng bộ thể hiện bằng `-)`.
  - Webhook ngân hàng / sàn TMĐT đặt trong `box "External Systems"`.
  - Audit Plane được tô màu nền đỏ nhạt bằng `rect rgb(255, 230, 230)`.
  - Sự cố timeout và đền bù thể hiện bằng `-x` / `--x`.
