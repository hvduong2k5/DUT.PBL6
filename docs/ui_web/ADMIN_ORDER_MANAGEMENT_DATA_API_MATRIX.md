# Admin Order Management — Data/API Matrix

**Cập nhật:** 2026-10-02  
**Nguồn phạm vi:** `docs/01_requirements/epics/EPIC_08_Order_Management.md`  
**Màn hình:** `/orders` trong `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Hàng đợi đơn hàng và lọc theo trạng thái | `US-ORD-05` | `GET /api/admin/orders` | `GET /admin/orders` | Read-only |
| Tìm theo mã đơn/khách hàng/công đoạn | `US-ORD-05` | Lọc trên projection đã tải | cùng list route | Đã có |
| Lọc theo nguồn, thanh toán và SLA | `US-ORD-05~06` | Lọc trên projection đã tải | cùng list route | Đã có |
| Đánh dấu sắp quá hạn, quá hạn và bị chặn | `US-ORD-06` | list projection | cùng list route | Đã có |
| Xem snapshot đơn và dòng hàng | `US-ORD-05` | `GET /api/admin/orders/:orderId` | cùng path không có `/api` | Popup read-only |
| Xem trạng thái Payment, Reservation, Packing và Shipping riêng biệt | `US-ORD-05` | detail projection | detail route | Đã có |
| Xem lịch sử trạng thái Order | `US-ORD-05` | detail projection | detail route | Đã có |

Danh sách fixture hiện chỉ dùng nguồn `WEB_D2C` và `B2B`, là các luồng đã có trong dự án. `MARKETPLACE` và `OFFLINE` vẫn nằm trong kiểu dữ liệu nguồn để không khóa thiết kế, nhưng chưa được đưa vào fixture hoặc tạo thêm nghiệp vụ riêng.

## Permission và dữ liệu nhạy cảm

- `ORDER_VIEW`: mở menu, danh sách và popup chi tiết đơn.
- `ORDER_SENSITIVE_VIEW`: xem số điện thoại, email, địa chỉ và ghi chú giao hàng của người nhận.
- `SALES_MANAGER` trong mock có cả hai Permission và scope `ORDER: ALL`.
- `CUSTOMER_SERVICE` và `WAREHOUSE_STAFF` có `ORDER_VIEW`, scope `ASSIGNED_ONLY`, nhưng không có `ORDER_SENSITIVE_VIEW`; BFF thay các trường nhạy cảm bằng nội dung đã ẩn.
- `B2B_VIEWER` và `SYSTEM_ADMIN` không tự động có quyền Order. Role quản trị hệ thống không đồng nghĩa với toàn quyền nghiệp vụ.
- Việc ẩn menu chỉ hỗ trợ trải nghiệm; BFF luôn kiểm tra lại Permission trước khi gọi upstream.

Mock hiện chưa thực hiện lọc dữ liệu thật theo `ASSIGNED_ONLY`; đây là nhãn scope để chứng minh mô hình quyền. Backend thật phải áp dụng scope tại truy vấn và không được trả toàn bộ dữ liệu rồi mới lọc ở browser.

## Ranh giới trạng thái

Order dùng các trạng thái cơ sở được Epic 08 nêu: `PENDING_PAYMENT`, `PAID`, `CONFIRMED`, `PROCESSING`, `PACKED`, `SHIPPED`, `DELIVERED`, `COMPLETED`, `DELIVERY_FAILED`, `EXPIRED`, `CANCELLED`.

Các trạng thái sau được trả dưới dạng projection liên quan và không được frontend suy diễn thành trạng thái Order:

- Payment: trạng thái và tham chiếu thanh toán.
- Reservation: tình trạng giữ tồn kho.
- Packing: tình trạng công việc đóng gói.
- Shipping: tình trạng bàn giao/giao vận và tracking.

`slaState` (`ON_TRACK`, `AT_RISK`, `OVERDUE`, `COMPLETE`) chỉ là trạng thái trình bày do upstream tính. Frontend không tự tính thời hạn hoặc xác định công đoạn chịu trách nhiệm.

## Phần cố ý chưa triển khai

Không có nút chuyển trạng thái, xác nhận, đóng gói, giao hàng, hủy đơn hoặc hoàn tiền. Các mutation chờ chốt tối thiểu:

1. Ma trận chuyển trạng thái đầy đủ, gồm cả COD và các trạng thái lỗi.
2. Actor/Permission được phép thực hiện từng transition và quy tắc phê duyệt.
3. Chính sách hủy theo từng giai đoạn và quan hệ hủy đơn với hoàn tiền.
4. Xử lý payment đến muộn, payment mismatch và timeout.
5. Điều kiện chuyển `DELIVERED` sang `COMPLETED`.
6. Công thức SLA, pause/resume và công đoạn chịu trách nhiệm.
7. Quy tắc chỉnh sửa snapshot/địa chỉ sau khi đặt hàng.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
$env:ADMIN_MOCK_ROLE="SALES_MANAGER"
npm run dev
```

Mở `http://localhost:3001/orders`. Đổi `ADMIN_MOCK_ROLE` thành `WAREHOUSE_STAFF` hoặc `CUSTOMER_SERVICE` để kiểm tra dữ liệu người nhận bị che; dùng `VIEW_ONLY` hoặc `SYSTEM_ADMIN` để kiểm tra `403`/menu không hiển thị.
