# EPIC 25 - Customer Analytics & DSS

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-22: Tự Động Phân Khúc Khách Hàng Theo Ma Trận RFM
- **Mô tả**: *Là một Giám đốc Marketing (ACT-06), Tôi muốn hệ thống tự động phân loại toàn bộ khách hàng thành 4 nhóm (Champions VIP, Loyal, Potential, At Risk) dựa trên mô hình RFM, Để tôi triển khai các chiến dịch chăm sóc khách hàng cá nhân hóa.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Chạy tiến trình phân loại khách hàng RFM định kỳ
    Given Dữ liệu mua sắm của 5.000 khách hàng trong Data Mart
    When Hệ thống chạy Cronjob tính toán điểm Recency, Frequency, Monetary lúc 00:00 hàng ngày
    Then Toàn bộ khách hàng được gán nhãn phân khúc:
      - Nhóm Champions (VIP): Điểm R(4-5), F(4-5), M(4-5)
      - Nhóm Loyal (Thân thiết): Điểm R(3-5), F(3-4), M(3-4)
      - Nhóm Potential (Tiềm năng): Điểm R(4-5), F(1-2), M(1-3)
      - Nhóm At Risk (Nguy cơ rời bỏ): Điểm R(1-2), F(3-5), M(3-5)
    And Dashboard hiển thị biểu đồ phân bổ khách hàng hình phễu và bản đồ nhiệt Heatmap
    And Cho phép xuất danh sách email/SĐT của nhóm "At Risk" để tạo chiến dịch kích cầu
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/rfm-segments`, `POST /api/v1/analytics/rfm/export`.

---

<!-- MIGRATED FROM backlog_software.md -->
### US-23: Phân Tích Tốc Độ Bán & Dự Báo Cạn Kho (Sales Velocity & Lead Time Forecast)
- **Mô tả**: *Là một Quản lý Sản xuất Làng nghề (ACT-04/06), Tôi muốn biết tốc độ tiêu thụ hàng ngày và số ngày dự kiến cạn kho của từng dòng mè xửng, Để chủ động lên kế hoạch nhập nguyên liệu mè vừng, đậu phộng, đường mạch nha trước các mùa cao điểm.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Dự báo thời điểm cạn kho dựa trên tốc độ tiêu thụ
    Given Dữ liệu bán hàng 14 ngày qua của SKU "Mè Xửng Dẻo Hộp 500g"
    When Tốc độ bán đạt 65 hộp/ngày và số lượng tồn kho còn lại là 260 hộp
    Then Hệ thống tính toán "Số ngày tồn kho còn lại" = 260 / 65 = 4 ngày
    And Thời gian cần để nhập nguyên liệu và ngào mẻ bánh mới (Lead Time) là 5 ngày
    And Hệ thống bật cảnh báo khẩn cấp màu đỏ:
      "CẢNH BÁO CẠN KHO: Mè Xửng Dẻo 500g sẽ hết hàng sau 4 ngày nữa. Cần nhập 300kg đường mạch nha và 150kg mè vàng trước ngày 02/09."
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/inventory-forecast`.

---

<!-- MIGRATED FROM backlog_software.md -->
### US-24: Động Cơ Đề Xuất Chiến Lược Kinh Doanh Thông Minh (Strategic Insights Engine)
- **Mô tả**: *Là một Nhà Quản trị Doanh nghiệp (ACT-06), Tôi muốn hệ thống tự động phân tích quy luật và đưa ra các khuyến nghị chiến lược hành động (Gợi ý gói Combo tăng AOV, Chiến dịch Marketing mùa vụ Cố Đô/Lễ Tết, Tái kích hoạt khách hàng cũ), Để doanh nghiệp tối đa hóa doanh thu và nâng cao sức cạnh tranh.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Hệ thống tự động sinh 3 khuyến nghị chiến lược hành động
    Given Dữ liệu tổng hợp từ giao dịch SQL và nhật ký NoSQL MongoDB
    When Quản trị viên mở trang "/admin/strategic-insights"
    Then Hệ thống hiển thị 3 thẻ khuyến nghị thông minh:
      1. Khuyến nghị Combo (Market Basket): "Phát hiện 68% khách mua Mè Xửng Giòn mua kèm Trà Cung Đình -> Gợi ý tạo Bundle 'Thưởng Trà Xứ Huế' chiết khấu 10% để tăng AOV thêm 20%."
      2. Khuyến nghị Mùa vụ: "Còn 30 ngày đến Lễ Hội Festival Huế -> Đề xuất tăng 40% ngân sách quảng cáo cho từ khóa 'Đặc sản Huế làm quà' và chuẩn bị 2.000 hộp quà gỗ."
      3. Khuyến nghị Tái kích hoạt: "Có 52 khách hàng VIP chưa mua lại sau 60 ngày -> Đề xuất gửi Voucher giảm 15% kèm tin nhắn Zalo ZNS cá nhân hóa."
    And Cung cấp nút "Kích hoạt chiến dịch ngay" và nút "Xuất Báo cáo Chiến lược PDF / Excel"
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/strategic-insights/recommendations`, `POST /api/v1/analytics/reports/export-pdf`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*