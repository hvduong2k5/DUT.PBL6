# EPIC 18 — B2B SALES & WHOLESALE

## 1. Acceptance Criteria

### AC-09 — B2B Quotation
```gherkin
Given Khách B2B gửi yêu cầu báo giá
When Sales Manager duyệt và áp dụng chiết khấu sỉ
Then Khách B2B nhận được thông báo
And Khách B2B có thể Checkout với mức giá đã duyệt
```


<!-- MIGRATED FROM backlog_software.md -->
### US-09: Yêu Cầu Báo Giá & Đặt Hàng Quà Tặng Doanh Nghiệp B2B
- **Mô tả**: *Là một Khách hàng Doanh nghiệp (ACT-03), Tôi muốn gửi yêu cầu đặt set quà tặng mè xửng số lượng lớn (> 50 hộp), đính kèm file logo công ty và nhận bảng dự toán chiết khấu sỉ tự động, Để tôi nhanh chóng trình duyệt ngân sách công ty.*
- **Độ ưu tiên**: Should Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Gửi yêu cầu báo giá B2B thành công
    Given Người dùng truy cập trang "/qua-tang-doanh-nghiep"
    When Chọn loại set quà "Hộp Gỗ Di Sản Hoàng Cung", số lượng "150 hộp"
    And Tải lên file logo công ty dạng vector ".AI / .PNG"
    And Nhập thông tin doanh nghiệp, mã số thuế và yêu cầu xuất hóa đơn VAT
    And Bấm "Gửi yêu cầu báo giá"
    Then Hệ thống tự động tính toán mức chiết khấu sỉ 18%
    And Gửi email báo giá sơ bộ kèm file PDF dự toán cho khách hàng
    And Tạo 1 Ticket chăm sóc B2B ưu tiên trên Web Admin
  ```
- **Kỹ thuật & API**: `POST /api/v1/catalog/b2b-quotes`.

---