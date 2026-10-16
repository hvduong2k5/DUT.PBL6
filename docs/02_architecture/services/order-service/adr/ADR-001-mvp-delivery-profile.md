# ADR-001 — Đợt giao đầu của Order và ranh giới MVP 1.0

Ngày: 15/10/2026. Quyết định: accepted cho implementation profile `mvp1`.

## Bối cảnh

Backlog sản phẩm còn yêu cầu Guest, Return một phần, Case/CSKH và những chức năng vận hành khác. Người dùng chọn tiếp tục phạm vi gọn đã vạch ra. Vì vậy tên `mvp1` trong cấu hình chỉ đợt giao đầu của Order, không có nghĩa hoàn thành mọi story MVP 1.0.

## Quyết định

Giữ profile sandbox 3.1 phục vụ regression và dữ liệu cũ. Profile `mvp1` chỉ nhận checkout CUSTOMER + VIETQR, không tạo Guest session, không bật auto-completion hoặc provider refund tự động. Đơn cũ không bị chuyển phương thức hoặc xóa enum.

Chọn SePay Payment Gateway BANK_TRANSFER ở sandbox. Giữ chặn production startup vì bộ Inventory/Fulfillment/identity thực chưa được chứng minh. Không đổi cờ để cho phép nhận tiền thật qua dependency simulator.

## Hệ quả

Hai profile có contract riêng và tests riêng. Mã mới có thể được dev/test mà không làm sai báo cáo runtime 3.1. Trước production phải thực hiện integration gates; không quảng bá sandbox test thành provider acceptance.

## Đối chiếu yêu cầu

| Epic/story | Đợt đầu | Còn thiếu cho MVP 1.0 |
| --- | --- | --- |
| AUTH-01 | Không bật Guest | Guest checkout, quyền tra cứu |
| CHK-01..04, PAY-01/03 | Checkout/QR, tiến trình tiền-kho | Kiểm thực tế provider và dependencies |
| PAY-02 | Loại COD theo quyết định người dùng | Requirements chưa đồng bộ việc bỏ COD |
| PAY-04/05, ORD-05/06 | API quản trị/đối soát và hàng đợi hiện có | UI và nghiệm thu SLA |
| RET-01..06 | Hủy và hoàn toàn phần thủ công | Return Case, media, thẩm định, hoàn/trả một phần |
| SHIP-01/02/06 | Tiếp tục adapter simulator hiện tại | Nhà vận chuyển thật |
| NOTI-01 | Sự kiện nguồn vẫn lưu bền | Email xác nhận và kiểm tích hợp Notification |

## Kiểm chứng

Kết quả thực thi được ghi ở báo cáo triển khai; ADR này không tự xác nhận tests đã PASS.
