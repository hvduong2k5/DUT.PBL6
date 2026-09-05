# EPIC 08 - Order Management

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-10: Khởi Tạo Đơn Hàng & Điều Phối Giao Dịch Phân Tán (SAGA Orchestrator)
- **Mô tả**: *Là một Khách hàng (ACT-02), Tôi muốn đơn hàng của tôi được khởi tạo và kiểm tra tồn kho một cách an toàn và tự động, Để đảm bảo hàng được giữ cho tôi trong lúc tôi thanh toán.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Luồng khởi tạo đơn hàng thành công (Happy Path)
    Given Khách hàng bấm "Đặt hàng ngay" từ giỏ hàng
    When "Order Service" tiếp nhận request và kích hoạt tiến trình SAGA:
      1. Tạo bản ghi đơn hàng với trạng thái "PENDING_INVENTORY" trong "om_order_db"
      2. Gửi lệnh gRPC "ReserveStock" sang "Inventory Service" để tạm giữ hàng 15 phút
      3. Nhận phản hồi giữ kho thành công -> Chuyển trạng thái đơn sang "PENDING_PAYMENT"
      4. Gửi lệnh sang "Payment Service" để sinh dữ liệu VietQR động
    Then Trả về mã HTTP 201 Created kèm mã đơn hàng và mã QR thanh toán cho Client

  Scenario: Luồng bù trừ SAGA khi hết hàng trong kho (Compensating Flow)
    Given Khách hàng bấm "Đặt hàng ngay"
    When "Inventory Service" phát hiện sản phẩm đã hết tồn kho
    Then "Inventory Service" trả về lỗi "OUT_OF_STOCK"
    And "Order Service" kích hoạt đền bù: Chuyển trạng thái đơn sang "REJECTED_STOCK_UNAVAILABLE"
    And Trả về mã lỗi HTTP 400 Bad Request kèm thông báo "Rất tiếc, sản phẩm vừa hết hàng."
  ```
- **Kỹ thuật & API**: `POST /api/v1/orders`, SAGA State Machine Pattern, gRPC Client.

---

<!-- MIGRATED FROM backlog_software.md -->
### US-11: Tự Động Hủy Đơn Hàng Quá Hạn Thanh Toán (Order Timeout Handler)
- **Mô tả**: *Là một Thủ kho (ACT-04), Tôi muốn các đơn hàng không thanh toán sau 15 phút sẽ tự động bị hủy và nhả lại số lượng tồn kho, Để hàng hóa không bị giữ ảo và người khác có thể mua được.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đơn hàng quá 15 phút chưa thanh toán
    Given Đơn hàng ở trạng thái "PENDING_PAYMENT" đã quá thời gian hết hạn TTL 15 phút
    When Tiến trình Scheduled Worker kiểm tra thấy chưa nhận được Webhook thanh toán
    Then Chuyển trạng thái đơn hàng thành "EXPIRED"
    And Phát sự kiện "OrderExpiredEvent" lên RabbitMQ
    And "Inventory Service" nhận sự kiện và nhả lại số lượng tồn kho đã tạm giữ
    And Gửi thông báo đến khách hàng "Đơn hàng của bạn đã hết hạn thanh toán."
  ```
- **Kỹ thuật**: RabbitMQ Dead Letter Queue (DLQ) / Redis Keyspace Notification.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*