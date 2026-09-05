# EPIC 28 - Omnichannel Notification System

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-17: Ứng Dụng Di Động Khách Hàng Flutter & Push Notification FCM
- **Mô tả**: *Là một Khách hàng thân thiết (ACT-02), Tôi muốn sử dụng App Mè Xửng O Mạ trên điện thoại để tra cứu tiến trình đơn hàng và nhận thông báo khi shipper chuẩn bị giao bánh, Để tôi không bị lỡ đơn.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Nhận Push Notification khi đơn hàng đang trên đường giao
    Given Khách hàng đã cài đặt App Mè Xửng O Mạ trên Android/iOS
    When Đơn vị vận chuyển 3PL cập nhật trạng thái "Đang giao hàng" (Out for Delivery)
    Then Firebase Cloud Messaging (FCM) đẩy thông báo đến điện thoại:
      "Bánh Mè Xửng O Mạ thơm ngon đang trên đường giao đến bạn! Vui lòng chú ý điện thoại."
    And Khách hàng bấm vào thông báo để mở trực tiếp trang Tra cứu lộ trình trên App
  ```

---

<!-- MIGRATED FROM backlog_software.md -->
### US-18: Cụm WebSocket Server Bắn Thông Báo Chuông Đơn Mới Đến Web Admin
- **Mô tả**: *Là một Quản lý Bán hàng (ACT-05), Tôi muốn Web Admin phát âm thanh chuông báo và hiển thị pop-up ngay khi có khách thanh toán thành công, Để tôi kịp thời điều phối đóng gói mà không cần nhấn F5 reload trang.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Web Admin nhận thông báo thời gian thực qua WebSocket
    Given Quản lý đang mở Web Admin Portal trên trình duyệt
    When Có khách hàng thanh toán thành công trên Web hoặc App di động
    Then Cụm WebSocket Socket.io Gateway nhận sự kiện từ RabbitMQ
    And Đẩy sự kiện "NEW_ORDER_PAID" đến toàn bộ Admin đang kết nối trong vòng < 100ms
    And Màn hình Admin hiện badge đỏ, pop-up đơn mới và phát âm thanh chuông báo hiệu
  ```
- **Kỹ thuật**: Socket.io Gateway kết nối Redis Pub/Sub Adapter (hỗ trợ scale đa node).

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*