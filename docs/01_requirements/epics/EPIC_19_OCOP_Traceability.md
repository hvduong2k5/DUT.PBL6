# EPIC 19 - OCOP Traceability

## 1. User Stories
*(Copy các User Story của Epic này từ file Product Backlog sang đây để chi tiết hóa)*

## 2. Functional Requirements (FR)
*(Liệt kê các yêu cầu tính năng chi tiết, các API cần thiết)*

## 3. Acceptance Criteria (AC)
*(Viết các kịch bản Gherkin Given/When/Then, Validation Rules, Edge Cases)*


<!-- MIGRATED FROM backlog_software.md -->
### US-06: Truy Xuất Nguồn Gốc Làng Nghề Qua Mã QR Story
- **Mô tả**: *Là một Khách hàng (ACT-01/02), Tôi muốn quét mã QR in trên vỏ hộp bánh để xem video quy trình nấu kẹo thủ công, nguồn gốc nguyên liệu và chứng chỉ vệ sinh ATTP, Để tôi hoàn toàn tin tưởng vào chất lượng sản phẩm OCOP chính gốc.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 3 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Quét mã QR Story trên bao bì
    Given Khách hàng quét mã QR trên nắp hộp bánh Mè Xửng O Mạ
    When Trình duyệt mở trang đích "/qr-story/lo-san-xuat-2026-08-01"
    Then Trang hiển thị:
      | Thông tin hiển thị              | Chi tiết minh bạch                              |
      | Nghệ nhân phụ trách             | Nghệ nhân Trần Thị Lan (40 năm tuổi nghề)       |
      | Nguồn gốc nguyên liệu           | Mè vàng Quảng Điền, Đậu phộng Phù Mỹ, Mạch nha nếp |
      | Ngày sản xuất & Hạn dùng        | NSX: 01/08/2026 - HSD: 01/02/2027              |
      | Chứng nhận kiểm định ATTP       | Giấy chứng nhận OCOP 4 Sao số 128/QĐ-UBND       |
      | Video clip ngắn 30 giây         | Cận cảnh công đoạn ngào đường kéo chỉ truyền thống |
  ```
- **Kỹ thuật & API**: `GET /api/v1/catalog/qr-story/{batchCode}`.

---


## 4. UI/UX Reference
*(Link Figma hoặc mockup cho tính năng này)*