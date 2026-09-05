# EPIC 09 — INVENTORY & BATCH

## 1. Acceptance Criteria

### AC-01 — Inventory Concurrency
```gherkin
Given sản phẩm chỉ còn 1 đơn vị
When hai khách hàng cùng đặt sản phẩm
Then chỉ một đơn hàng được xác nhận tồn kho thành công
And không được tạo đơn vượt quá tồn kho thực tế
```

### AC-04 — Expiry Warning
```gherkin
Given Batch B001 có hạn sử dụng sắp đến
When hệ thống kiểm tra tồn kho định kỳ
Then Warehouse Staff nhận được cảnh báo
And Supply/Sales Staff có thể xem danh sách sản phẩm cần xử lý
```


<!-- MIGRATED FROM backlog_software.md -->
### US-14: Quản Lý Tồn Kho Theo Lô & Cảnh Báo Ngưỡng Tồn Tối Thiểu
- **Mô tả**: *Là một Thủ kho (ACT-04), Tôi muốn theo dõi số lượng tồn kho theo từng lô sản xuất và nhận cảnh báo khi số lượng chạm ngưỡng tối thiểu, Để tôi kịp thời báo xưởng ngào thêm mẻ mè xửng mới.*
- **Độ ưu tiên**: Must Have | **Điểm ước lượng**: 5 SP
- **Acceptance Criteria (Gherkin)**:
  ```gherkin
  Scenario: Kích hoạt cảnh báo khi tồn kho chạm ngưỡng tối thiểu
    Given Sản phẩm "Mè Xửng Mè Đen Hộp 250g" có cấu hình ngưỡng tối thiểu là 50 hộp
    When Số lượng tồn kho sau đơn hàng giảm xuống còn 48 hộp
    Then "Inventory Service" phát sự kiện "LowStockWarningEvent" lên RabbitMQ
    And Gửi thông báo đỏ lên Dashboard Web Admin của Thủ kho
    And Ghi log sự kiện để phân hệ BI phân tích nhu cầu nhập nguyên liệu
  ```
- **Kỹ thuật & API**: `GET /api/v1/inventory/stocks`, `POST /api/v1/inventory/adjustments`.

---