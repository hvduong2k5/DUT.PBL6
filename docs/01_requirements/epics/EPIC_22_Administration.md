# EPIC 22 - User / Role / Permission Administration

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-03: Kiểm Soát Phân Quyền Dựa Trên Vai Trò (RBAC Guard)
- **Mô tả**: *Là một Quản trị viên Kỹ thuật (ACT-07), Tôi muốn hệ thống thực thi ma trận phân quyền chi tiết trên từng API endpoint, Để ngăn chặn hành vi truy cập trái phép và bảo vệ an toàn dữ liệu doanh nghiệp.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Chặn người dùng thường truy cập tài nguyên của Quản trị viên
    Given Người dùng đăng nhập với vai trò "CUSTOMER"
    When Gửi request xem báo cáo doanh thu "GET /api/v1/analytics/revenue"
    Then API Gateway kiểm tra JWT Claims thấy không có quyền "ANALYTICS:VIEW"
    And Chặn request ngay tại tầng Gateway, không forward vào microservice nội bộ
    And Ghi nhật ký cảnh báo an ninh (Security Audit Log)
    And Trả về mã lỗi HTTP 403 Forbidden kèm thông báo "Bạn không có quyền truy cập chức năng này."

  Scenario: Cho phép Giám Đốc truy cập phân hệ Báo cáo Chiến lược
    Given Người dùng đăng nhập với vai trò "EXECUTIVE_ADMIN"
    When Gửi request "GET /api/v1/analytics/strategic-insights"
    Then API Gateway xác thực thành công vai trò và quyền hạn
    And Cho phép forward request vào Analytics Service và trả về HTTP 200 OK
  ```
- **Kỹ thuật**: Middleware `RBACAuthGuard` trên API Gateway & NestJS Decorators `@Roles('ADMIN', 'MANAGER')`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*