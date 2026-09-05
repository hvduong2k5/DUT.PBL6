# EPIC 10 — PACKING & FULFILLMENT

## 1. Acceptance Criteria

### AC-02 — Packing Video
```gherkin
Given đơn hàng OM-001 đang ở trạng thái cần đóng gói
When Packing Staff hoàn thành đóng gói
And lưu video đóng gói
Then video được liên kết với OM-001
And người không có quyền không thể truy cập video
```

### AC-03 — Packing Investigation
```gherkin
Given khách hàng khiếu nại thiếu sản phẩm
When CSKH tra cứu OM-001
Then hệ thống hiển thị thông tin đóng gói
And cho phép xem bằng chứng video nếu CSKH có quyền
```
