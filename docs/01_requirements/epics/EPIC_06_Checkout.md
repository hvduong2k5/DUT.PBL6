# EPIC 06 - Checkout

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-16: Trải Nghiệm Mua Sắm D2C Cố Đô Trên Website Next.js 14
- **Mô tả**: *Là một Khách hàng (ACT-01/02), Tôi muốn duyệt website với giao diện văn hóa Huế tinh tế, tốc độ phản hồi nhanh và thanh toán tiện lợi trong 3 bước, Để tôi có trải nghiệm mua sắm nông đặc sản cao cấp.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Khách hàng hoàn tất mua hàng 1-Click trên Web D2C
    Given Khách hàng duyệt trang chủ "/san-pham"
    When Bấm chọn "Mè Xửng Dẻo Hộp Quà Gỗ 500g" và nhấn "Mua Ngay"
    And Điền thông tin giao hàng gồm Tỉnh/Thành, Quận/Huyện, Số điện thoại
    And Quét mã VietQR thanh toán
    Then Màn hình tự động chuyển sang trang "/dat-hang-thanh-cong" trong vòng < 2 giây qua WebSocket
    And Khách hàng nhận được tin nhắn Zalo ZNS xác nhận đơn hàng
  ```

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*