# Admin Shipping & Delivery — Data/API Matrix

**Cập nhật:** 2026-10-03  
**Nguồn phạm vi:** `docs/01_requirements/epics/EPIC_11_Shipping_Delivery.md`  
**Màn hình:** `/shipping` (quản lý) và `/work/delivery` (công việc cá nhân) trong `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Danh sách Shipment và trạng thái giao vận | `US-SHIP-01`, `FR-SHIP-01~04` | `GET /api/admin/shipping/shipments` | `GET /admin/shipping/shipments` | Read-only |
| KPI đang hoạt động, chờ bàn giao, đang vận chuyển và ngoại lệ | `US-SHIP-01`, `US-SHIP-06` | list projection | cùng list route | Đã có |
| Lọc trạng thái, provider, nguồn Order và ngoại lệ | `US-SHIP-01`, `US-SHIP-06` | Lọc trên projection đã tải | cùng list route | Đã có |
| Chi tiết người nhận, Package, phí, COD và bàn giao | `US-SHIP-01` | `GET /api/admin/shipping/shipments/:shipmentId` | cùng path không có `/api` | Popup read-only |
| Tracking giữ trạng thái gốc và trạng thái ánh xạ | `US-SHIP-06`, `FR-SHIP-15~18` | detail projection | detail route | Đã có |
| Nhận biết sự kiện lặp/đến trễ/chưa ánh xạ | `US-SHIP-06` | detail projection | detail route | Đã có dữ liệu trình bày |
| Hàng đợi Shipment được phân công | `US-SHIP-03`, `FR-SHIP-08` | `GET /api/admin/shipping/workbench/shipments` | `GET /admin/shipping/workbench/shipments` | Giai đoạn 2, BFF lọc đúng employee |
| Chi tiết công việc, người nhận, Package và COD | `US-SHIP-03`, `FR-SHIP-08` | `GET /api/admin/shipping/workbench/shipments/:shipmentId` | cùng path không có `/api` | Read-only Workbench |

## Permission và dữ liệu nhạy cảm

- `SHIPMENT_VIEW`: quyền nền để đọc Shipment; không tự cấp quyền vào một loại màn hình.
- `SHIPMENT_MANAGEMENT_VIEW`: mở `/shipping`; fixture `SALES_MANAGER` có quyền này với scope `ALL`.
- Chỉ có một role nghiệp vụ `DELIVERY`; khả năng làm việc hoặc quản lý được quyết định bằng Permission và Scope.
- `DELIVERY_WORKBENCH_VIEW`: mở `/work/delivery`; profile mock `DELIVERY_WORKBENCH` trả role `DELIVERY` với scope `ASSIGNED_ONLY`.
- `SHIPMENT_SENSITIVE_VIEW`: xem số điện thoại và địa chỉ người nhận.
- Delivery Workbench bắt buộc đồng thời có `DELIVERY_WORKBENCH_VIEW`, `SHIPMENT_SENSITIVE_VIEW`, scope Shipment và quan hệ phân công đúng employee.
- Profile `PACKING_WORKBENCH` và `WAREHOUSE_STAFF` vẫn có quyền Shipment phục vụ luồng tạo/bàn giao trong tương lai, nhưng không vì vậy mà được vào trang quản lý hoặc Delivery Workbench.
- Mock `SALES_MANAGER` có quyền quản lý, scope `ALL`, nhưng không mặc nhiên xem dữ liệu người nhận.
- `SYSTEM_ADMIN` không tự động có quyền Shipping.
- BFF che dữ liệu người nhận trên trang quản lý nếu thiếu quyền nhạy cảm; ở Workbench, BFF từ chối cả request nếu thiếu quyền hoặc Shipment không được phân công.

## Ranh giới nghiệp vụ

- Shipment status và Order status được hiển thị thành hai cột/trường riêng. Tạo vận đơn không đồng nghĩa Order đã `SHIPPED`.
- COD chỉ là nghĩa vụ/trạng thái tham chiếu; Shipping không tự xác nhận Payment đã thu tiền.
- Tracking lưu mã/trạng thái gốc, trạng thái nội bộ ánh xạ, thời điểm sự kiện, thời điểm nhận và kết quả xử lý.
- Phí Checkout, phí provider dự kiến và phí thực tế được giữ riêng; frontend không tự đối soát.
- `3PL Demo A/B` chỉ là fixture trung lập, không phải quyết định chọn provider production.
- Các mã trạng thái hiện dùng vòng đời đề xuất trong Epic và chưa phải state machine chính thức.

## Mutation tạm gác

Chưa triển khai tạo/hủy vận đơn, in nhãn, xác nhận bàn giao hoặc điều chỉnh tracking vì còn phải chốt:

1. Provider/dịch vụ được bật trong MVP và cơ chế chọn provider.
2. Một Order có một hay nhiều Package/Shipment và có giao một phần hay không.
3. Mốc chính thức để Order chuyển `SHIPPED`.
4. Vòng đời Shipment và bảng ánh xạ riêng của từng provider.
5. Hủy vận đơn, giao lại, số lần giao và chuyển hoàn.
6. Bằng chứng giao thành công/thất bại và nguồn xác nhận COD.
7. Chính sách bảo vệ dữ liệu cá nhân trên nhãn, tracking và log.
8. Quy trình, kỳ và tiêu chí đối soát phí vận chuyển.

Đã triển khai phần hàng đợi và chi tiết được phân công của `US-SHIP-03`. Mutation cập nhật kết quả giao của `US-SHIP-04` và báo cáo `US-SHIP-05` vẫn thuộc Giai đoạn 2 và chưa mở vì các chính sách nêu trên chưa chốt.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
$env:ADMIN_MOCK_PROFILE="DELIVERY_WORKBENCH"
npm run dev
```

Mở `http://localhost:3001/work/delivery` với profile `DELIVERY_WORKBENCH`. Session trả role `DELIVERY`, nhưng không truy cập được `/shipping` vì thiếu `SHIPMENT_MANAGEMENT_VIEW`.

Dùng `SALES_MANAGER` để kiểm tra `/shipping` và xác nhận bị từ chối `/work/delivery`. `SYSTEM_ADMIN` không mặc nhiên truy cập cả hai vùng nghiệp vụ.
