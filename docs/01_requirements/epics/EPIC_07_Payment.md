# EPIC 07 - Payment

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-12: Sinh Mã VietQR Động Chuẩn NAPAS Tích Hợp Số Tiền
- **Mô tả**: *Là một Khách hàng (ACT-02), Tôi muốn màn hình hiển thị mã VietQR chứa đầy đủ số tài khoản, số tiền và nội dung chuyển khoản duy nhất, Để tôi chỉ cần quét mã trên App ngân hàng mà không cần nhập tay.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Sinh mã VietQR động
    Given Đơn hàng "OM-2026-8899" có tổng tiền thanh toán là 185.000 VNĐ
    When Khách hàng chọn phương thức "Thanh toán QR Napas"
    Then "Payment Service" sinh chuỗi mã VietQR chuẩn EMVCo chứa:
      | Trường dữ liệu          | Giá trị đặc tả                               |
      | Ngân hàng thụ hưởng     | Vietcombank (Chi nhánh Huế)                  |
      | Số tài khoản            | 0123456789 (Tài khoản Mè Xửng O Mạ)          |
      | Số tiền                 | 185000                                       |
      | Nội dung chuyển khoản   | OM20268899                                   |
    And Hiển thị hình ảnh mã QR và đồng hồ đếm ngược 15:00 trên màn hình
  ```
- **Kỹ thuật & API**: `POST /api/v1/payments/vietqr/generate`.

---

<!-- MIGRATED FROM backlog_software.md -->
### US-13: Tiếp Nhận Webhook Thanh Toán & Xử Lý Chống Trùng Lặp (Idempotent Consumer)
- **Mô tả**: *Là một Quản trị viên Doanh nghiệp (ACT-06), Tôi muốn hệ thống tự động xác nhận khi khách đã chuyển tiền qua Webhook ngân hàng và tuyệt đối không cộng tiền trùng lặp, Để tài chính luôn chuẩn xác 100%.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Nhận Webhook thanh toán thành công từ Ngân hàng
    Given Khách hàng đã quét mã QR và chuyển khoản thành công
    When Cổng thanh toán (EXT-01) gửi Webhook HTTP POST tới "/api/v1/payments/webhook"
    And Header chứa chữ ký số HMAC-SHA256 hợp lệ
    Then "Payment Service" kiểm tra Idempotency Key "payment:trans:{transId}" trong Redis:
      1. Khóa chưa tồn tại -> Lưu khóa vào Redis với TTL 24h
      2. Ghi nhận giao dịch thành công vào "om_payment_db.payments"
      3. Phát sự kiện "PaymentSuccessEvent" lên RabbitMQ
      4. "Order Service" chuyển đơn hàng sang trạng thái "PAID"
      5. Trả về HTTP 200 OK cho Ngân hàng trong vòng < 500ms

  Scenario: Ngân hàng gửi lại Webhook trùng lặp (Duplicate Webhook)
    Given Webhook của giao dịch "TRANS_12345" được gửi lại lần 2
    When "Payment Service" kiểm tra thấy Idempotency Key đã tồn tại trong Redis
    Then Lập tức phản hồi HTTP 200 OK
    And Bỏ qua bước xử lý cộng tiền lần hai, ngăn chặn thất thoát
  ```
- **Kỹ thuật & API**: `POST /api/v1/payments/webhook`, HMAC-SHA256 Signature Verification, Redis Distributed Lock.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*