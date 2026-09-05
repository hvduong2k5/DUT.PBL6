# EPIC 24 - Finance & Business Analytics

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-21: Báo Cáo Thống Kê Doanh Thu Đa Chiều & Trực Quan Hóa Tương Tác
- **Mô tả**: *Là một Giám đốc Kinh doanh (ACT-06), Tôi muốn xem báo cáo thống kê doanh thu đa chiều theo khoảng thời gian tùy chọn (ngày, tuần, tháng, quý), theo kênh bán (Website D2C, Mobile App, Điểm bán sỉ B2B) và theo dòng sản phẩm, Để tôi đánh giá chính xác hiệu quả kinh doanh của doanh nghiệp.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 8 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Xem báo cáo doanh thu đa chiều và Drill-down chi tiết
    Given Giám đốc truy cập phân hệ "/admin/bi-analytics/revenue"
    When Chọn bộ lọc: Thời gian "Quý 3/2026", Kênh bán "Tất cả kênh", Tiêu chí "Dòng sản phẩm"
    Then Dashboard hiển thị biểu đồ tương tác ECharts:
      | Chỉ số hiển thị                 | Ý nghĩa quản trị                                |
      | Tổng doanh thu & Tăng trưởng    | Doanh số đạt được so với cùng kỳ quý trước      |
      | Giá trị đơn trung bình (AOV)    | Mức chi tiêu trung bình trên 1 đơn hàng         |
      | Tỷ lệ hoàn đơn (Return Rate)    | Tỷ lệ đơn bị hoàn/hủy để kiểm soát chất lượng   |
      | Cơ cấu doanh thu theo dòng bánh | Tỷ trọng đóng góp của Mè dẻo, Mè giòn, Hộp quà  |
    And Cho phép nhấp vào từng cột biểu đồ để Drill-down xem chi tiết danh sách đơn hàng
  ```
- **Kỹ thuật & API**: `GET /api/v1/analytics/reports/revenue-multidimensional`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*