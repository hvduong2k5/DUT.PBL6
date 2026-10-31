# Admin Packing & Fulfillment — Data/API Matrix

**Cập nhật:** 2026-10-03  
**Nguồn phạm vi:** `docs/01_requirements/epics/EPIC_10_Packing.md`  
**Màn hình:** `/packing` (quản lý) và `/work/packing` (công việc cá nhân) trong `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Hàng đợi Packing Task | `US-PACK-01`, `FR-PACK-01~03` | `GET /api/admin/packing/tasks` | `GET /admin/packing/tasks` | Read-only |
| KPI Task hoạt động, khẩn/quá SLA, chưa phân công và bị chặn | `US-PACK-01`, `US-PACK-07` | list projection | cùng list route | Đã có |
| Lọc trạng thái, nguồn, phân công, SLA và ưu tiên | `FR-PACK-02` | Lọc trên projection đã tải | cùng list route | Đã có |
| Chi tiết snapshot SKU/quy cách/số lượng và Batch phân bổ | `US-PACK-02`, `FR-PACK-04~05` | `GET /api/admin/packing/tasks/:taskId` | cùng path không có `/api` | Popup read-only |
| Điều kiện Payment/Order/Inventory đầu vào | `US-PACK-01`, `FR-PACK-03` | detail projection | detail route | Đã có |
| Checklist và điều kiện còn thiếu để hoàn tất | `US-PACK-03`, `US-PACK-08` | detail projection | detail route | Chỉ đọc |
| Lịch sử Packing Task | `BR-ORDER-05`, `BR-AUDIT-01` | detail projection | detail route | Chỉ đọc |
| Hàng đợi cá nhân theo phân công | `US-PACK-01`, `BR-PACK-01` | `GET /api/admin/packing/workbench/tasks` | `GET /admin/packing/workbench/tasks` | Workbench, BFF lọc đúng employee |
| Chi tiết thao tác lấy SKU/Batch | `US-PACK-02`, `FR-PACK-04~05` | `GET /api/admin/packing/workbench/tasks/:taskId` | cùng path không có `/api` | Workbench |
| Xác nhận từng bước checklist | `US-PACK-03`, `FR-PACK-06~07` | `PATCH /api/admin/packing/workbench/tasks/:taskId/checklist/:itemId` | cùng path không có `/api` | Đã có mutation, revision + idempotency |
| Xác nhận hoàn tất đóng gói | `US-PACK-08`, `FR-PACK-17~18` | `POST /api/admin/packing/workbench/tasks/:taskId/complete` | cùng path không có `/api` | Đã có mutation và popup xác nhận |

## Quyền và actor

- Chỉ có một role nghiệp vụ `PACKING`. Không tạo `PACKING_STAFF` hoặc `PACKING_LEAD` thành role riêng.
- `PACKING_TASK_VIEW`: quyền nền để đọc Packing Task; không tự cấp quyền vào một loại màn hình.
- Profile mock `PACKING_MANAGEMENT`: role `PACKING`, có `PACKING_MANAGEMENT_VIEW` và scope `PACKING_TASK: ALL`.
- Profile mock `PACKING_WORKBENCH`: role `PACKING`, có `PACKING_WORKBENCH_VIEW` và scope `PACKING_TASK: ASSIGNED_ONLY`.
- `PACKING_CHECKLIST_UPDATE` và `PACKING_TASK_COMPLETE`: hai quyền thao tác độc lập của nhân viên thực hiện.
- BFF lọc hàng đợi cá nhân theo `employeeId` và từ chối cả detail/mutation nếu Task không được phân công cho người đang đăng nhập.
- `SYSTEM_ADMIN` không tự động có quyền nghiệp vụ Packing.
- Sidebar, page/BFF và upstream thật đều phải kiểm tra quyền; việc ẩn menu không được xem là biện pháp bảo mật.

## Ranh giới dữ liệu

- Product/SKU hiển thị từ snapshot Order, không đọc lại catalog hiện tại.
- Batch và nhãn vị trí, nếu upstream có, là dữ liệu phân bổ từ Inventory; Packing không tự chọn lại Batch hoặc sửa số dư.
- Trạng thái Payment, Order và Reservation trong điều kiện đầu vào chỉ để đọc.
- `slaState` và `priority` do upstream tính; frontend không tự tính công thức SLA.
- `completionReadiness` chỉ giải thích điều kiện đang thiếu, không phải quyền thực hiện mutation.
- Video/media thuộc `US-PACK-04~06` giai đoạn 2 nên chưa có trong trang và API hiện tại.

Các mã `READY`, `ASSIGNED`, `IN_PROGRESS`, `BLOCKED`, `COMPLETED`, `CANCELLED` đang là projection trình bày dựa trên vòng đời đề xuất của Epic. Đây chưa phải state machine chính thức.

## Mutation đã triển khai và phần còn tạm gác

Đã triển khai mutation checklist và hoàn tất cho **Task đang xử lý, đã được phân công cho chính nhân viên**. Chưa triển khai nhận việc/phân công, bắt đầu Task, đổi người, mở lại hoặc thay đổi Batch vì Epic còn yêu cầu chốt:

1. Một Order có một hay nhiều Task/kiện và có đóng gói một phần hay không.
2. Điều kiện vào hàng đợi cho Payment, COD, B2B và Marketplace.
3. Cơ chế tự nhận, chỉ định, tự động phân công và chuyển giao.
4. Bộ checklist theo SKU/kênh/bao bì và cách xác nhận SKU/Batch thực lấy.
5. Mốc xuất kho và cơ chế bù trừ khi mở lại/hủy Task.
6. Công thức SLA, giờ làm việc và cách xử lý Task bị chặn.
7. Ma trận trạng thái chính thức và quyền mở lại. Hai mutation hiện có đã gửi `expectedRevision` và `Idempotency-Key`.

Media/video của `US-PACK-04~06` vẫn thuộc Giai đoạn 2. Workbench MVP không hiển thị thumbnail cục bộ như bằng chứng đã upload và chưa thêm API media khi chính sách kỹ thuật chưa được chốt.

## Chạy local

```powershell
npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
$env:ADMIN_MOCK_PROFILE="PACKING_WORKBENCH"
npm run dev
```

Mở `http://localhost:3001/work/packing` với profile `PACKING_WORKBENCH`. Session vẫn trả role `PACKING`, nhưng không thấy và không truy cập được `/packing` vì thiếu `PACKING_MANAGEMENT_VIEW`.

Dùng profile `PACKING_MANAGEMENT` hoặc `SALES_MANAGER` để kiểm tra `/packing`. `PACKING_MANAGEMENT` vẫn mang role `PACKING` nhưng không mặc nhiên có quyền thao tác Workbench. Dùng `WAREHOUSE_STAFF` hoặc `SYSTEM_ADMIN` để kiểm tra từ chối cả hai vùng Packing.
