# EPIC 17 - Promotion & Loyalty

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-08: Áp Dụng Mã Ưu Đãi (Coupon / Voucher Engine)
- **Mô tả**: *Là một Khách hàng (ACT-02), Tôi muốn nhập mã giảm giá khi thanh toán, Để tôi được giảm trực tiếp trên tổng giá trị đơn hàng hoặc được miễn phí vận chuyển.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 3 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Áp mã giảm giá hợp lệ
    Given Đơn hàng có giá trị 300.000 VNĐ
    When Khách hàng nhập mã voucher "FESTIVALHUE10" (Giảm 10% cho đơn từ 200k, tối đa 50k)
    Then Hệ thống kiểm tra điều kiện hợp lệ (còn lượt dùng, trong thời hạn áp dụng)
    And Giảm trực tiếp 30.000 VNĐ vào tổng tiền thanh toán
    And Hiển thị số tiền phải trả còn lại là 270.000 VNĐ

  Scenario: Nhập mã voucher đã hết lượt sử dụng
    Given Voucher "CHAOXUAN" đã đạt giới hạn 500 lượt sử dụng
    When Khách hàng bấm "Áp dụng"
    Then Hệ thống từ chối và hiển thị thông báo "Mã giảm giá đã hết lượt sử dụng."
  ```
- **Kỹ thuật & API**: `POST /api/v1/orders/apply-coupon`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*