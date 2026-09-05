# 48. NON-FUNCTIONAL REQUIREMENTS

## NFR-01 — Performance
Các thao tác phổ biến như xem sản phẩm, tìm kiếm và thêm giỏ hàng phải có thời gian phản hồi phù hợp với trải nghiệm người dùng.

## NFR-02 — Mobile First
Website phải tối ưu cho thiết bị di động và tương thích với nền tảng ứng dụng di động (Native Mobile App iOS/Android).

## NFR-03 — Availability
Hệ thống phải đảm bảo khả dụng cao trong các giai đoạn cao điểm như Tết và các mùa quà tặng.

## NFR-04 — Security
Dữ liệu Customer, Employee, Order, Payment, Supplier (Nhà cung cấp) và Audit phải được bảo vệ khỏi truy cập trái phép.

## NFR-05 — Authorization
Người dùng chỉ được truy cập dữ liệu và chức năng thuộc quyền được cấp.

## NFR-06 — Data Integrity
Thông tin đơn hàng, thanh toán và tồn kho phải đảm bảo tính nhất quán (Consistency). Đặc biệt, hệ thống phải có cơ chế xử lý Race-condition và chống bán vượt (Over-selling) khi trừ tồn kho đồng thời từ nhiều kênh (Website, Marketplace, Offline).

## NFR-07 — Audit Integrity
Audit Log phải có khả năng phát hiện việc sửa đổi hoặc xóa trái phép.

### Technical Security Constraint
Cơ chế Audit Integrity có thể sử dụng:
```text
HMAC-SHA256
+
prev_hash
+
hash chain
```
để phát hiện thay đổi trái phép trong chuỗi Audit.

## NFR-08 — Privacy
Hệ thống phải bảo vệ dữ liệu cá nhân và chỉ sử dụng dữ liệu theo mục đích được phép.

## NFR-09 — Media Security
Ảnh/video nội bộ như Packing Video phải được kiểm soát quyền truy cập.

## NFR-10 — Scalability
Hệ thống phải có khả năng mở rộng khi số lượng Customer, Order, Media, giao dịch Offline tăng cao và xử lý tốt tải trọng của các truy vấn dữ liệu lớn phục vụ báo cáo phân tích (DSS/Analytics).

## NFR-11 — Observability
Hệ thống phải có khả năng theo dõi lỗi, hiệu năng, trạng thái xử lý nghiệp vụ, cũng như giám sát tình trạng sức khỏe (Health Check & Timeout) của các hệ thống ngoại vi tích hợp (Marketplace, Payment Provider, Logistics).

---


<!-- MIGRATED FROM backlog_software.md -->
### US-25: Giám Sát Hiệu Năng Tập Trung (Prometheus, Grafana & Jaeger Tracing)
- **Mô tả**: *Là một Kỹ sư Vận hành (ACT-07), Tôi muốn theo dõi các chỉ số tải CPU, RAM, RPS, Response Time P95 và truy vết Trace-ID xuyên suốt các Microservices trên Grafana, Để phát hiện và khắc phục điểm nghẽn hệ thống tức thì.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Truy vết lỗi phân tán bằng Trace-ID trên Jaeger
    Given Khách hàng gặp lỗi khi thanh toán đơn hàng
    When Kỹ sư tra cứu "Trace-ID: tr_8899aabb" trên Dashboard Jaeger
    Then Hệ thống hiển thị toàn bộ cây tiến trình (Span Waterfall) qua các service:
      Gateway (20ms) -> Order Service (45ms) -> Payment Service (120ms - ERROR 500 Bank Timeout)
    And Giúp kỹ sư xác định chính xác nguyên nhân lỗi do cổng ngân hàng phản hồi chậm
  ```

---

<!-- MIGRATED FROM backlog_software.md -->
### US-26: Tự Động Hóa Pipeline CI/CD Đa Dịch Vụ Với GitHub Actions
- **Mô tả**: *Là một Lập trình viên trong nhóm (ACT-07), Tôi muốn mỗi lần tạo Pull Request vào nhánh `develop` thì hệ thống sẽ tự động chạy kiểm thử Lint, Unit Test, Contract Test và Build Docker Image, Để đảm bảo mã nguồn luôn đạt chuẩn chất lượng trước khi bàn giao.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Pipeline CI/CD chạy tự động khi có Pull Request
    Given Lập trình viên push mã nguồn và tạo PR vào nhánh "develop"
    When GitHub Actions kích hoạt workflow ".github/workflows/ci.yml"
    Then Tự động chạy song song:
      1. Kiểm tra Linting & TypeScript compilation
      2. Chạy bộ Unit Test (Jest/PyTest) yêu cầu Code Coverage >= 75%
      3. Chạy Pact Contract Test kiểm tra hợp đồng API giữa Gateway và các Service
      4. Đóng gói Docker Images đa tầng (Multi-stage build)
    And Nếu tất cả Passed (Xanh) -> Cho phép Merge PR
    And Nếu có bất kỳ bước nào Failed (Đỏ) -> Khóa nút Merge và gửi cảnh báo lỗi vào Discord/Telegram nhóm
  ```

---