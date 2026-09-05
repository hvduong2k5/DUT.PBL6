# EPIC 11 - Shipping & Delivery

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-15: Tự Động Tạo Vận Đơn 3PL & Nhận Cập Nhật Lộ Trình Shipper Realtime
- **Mô tả**: *Là một Nhân viên Đóng gói (ACT-04), Tôi muốn bấm 1 nút để hệ thống tự động đẩy đơn sang nhà vận chuyển (GHN/GHTK) và in mã vận đơn, Để rút ngắn thời gian bàn giao cho shipper.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Đẩy vận đơn sang đơn vị 3PL thành công
    Given Đơn hàng đã ở trạng thái "PAID" và đã được đóng gói hút chân không xong
    When Nhân viên kho bấm "Tạo vận đơn giao hàng"
    Then "Shipping Service" gọi API GHN/GHTK Sandbox gửi thông tin người nhận và trọng lượng
    And Nhận về mã vận đơn "GHN_HUE_998822"
    And In phiếu giao hàng khổ A6 có mã vạch
    And Chuyển trạng thái đơn sang "SHIPPED"
  ```
- **Kỹ thuật & API**: `POST /api/v1/shipping/create-shipment`, `POST /api/v1/shipping/webhook-3pl`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*