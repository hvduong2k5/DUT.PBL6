# Admin Inventory & Warehouse Workbench — Data/API Matrix

**Cập nhật:** 2026-10-03  
**Backlog:** `EPIC_09_Inventory.md`  
**Màn hình:** `/inventory`, `/work/warehouse`  
**Ứng dụng:** `apps/web-admin`  
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## 1. Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Quản lý số dư theo SKU | `US-INV-01`, `FR-INV-01` | `GET /api/admin/inventory/skus` | `GET /admin/inventory/skus` | Read-only, có KPI/bộ lọc/bảng |
| Xem Batch, NSX/HSD và biến động | `US-INV-04~06` | `GET /api/admin/inventory/skus/:skuId` | cùng path không có `/api` | Popup trên trang quản lý |
| Ngữ cảnh Warehouse Workbench | `US-INV-01~06`, `US-INV-09` | `GET /api/admin/inventory/workbench` | `GET /admin/inventory/workbench` | SKU, Batch, số dư và biến động gần đây |
| Ghi nhận nhập kho | `US-INV-02`, `FR-INV-05~06`, `FR-INV-08` | `POST /api/admin/inventory/workbench/receipts` | cùng path không có `/api` | Batch hiện có hoặc Batch mới; có revision/idempotency |
| Ghi nhận xuất kho | `US-INV-03`, `FR-INV-07~08` | `POST /api/admin/inventory/workbench/issues` | cùng path không có `/api` | Order hoặc lý do ngoài Order đã phê duyệt |
| Hỏng, thất thoát, kiểm kê | `US-INV-09`, `FR-INV-18~19`, `FR-INV-22` | `POST /api/admin/inventory/workbench/adjustments` | cùng path không có `/api` | Mock an toàn trả chờ duyệt, chưa đổi số dư |

## 2. Role, Permission và Scope

Role không điều khiển giao diện trực tiếp. Hai profile local dưới đây cùng trả role `WAREHOUSE`:

- `WAREHOUSE_MANAGEMENT`: có `INVENTORY_VIEW`, `INVENTORY_MANAGEMENT_VIEW`, `BATCH_VIEW`; thấy `/inventory`, không thấy Workbench.
- `WAREHOUSE_WORKBENCH`: có `INVENTORY_VIEW`, `BATCH_VIEW`, `WAREHOUSE_WORKBENCH_VIEW` và ba Permission mutation; thấy `/work/warehouse`, không thấy trang quản lý.
- `WAREHOUSE_STAFF` là profile local cũ, không dùng làm chuẩn cho mô hình phân quyền mới.

Permission mutation độc lập:

- `INVENTORY_RECEIPT_CREATE`
- `INVENTORY_ISSUE_CREATE`
- `INVENTORY_ADJUSTMENT_CREATE`

BFF kiểm tra lại Permission và Scope `INVENTORY`; việc ẩn menu/nút không thay thế authorization phía server.

## 3. Quy tắc mutation

- Browser chỉ gửi nghiệp vụ biến động, không gửi hoặc sửa trực tiếp số dư đích.
- Mỗi lệnh có `expectedRevision` và `idempotencyKey`; BFF chuyển idempotency key vào header upstream.
- Nhập Batch mới bắt buộc mã Batch, NSX, HSD; HSD phải sau NSX.
- Xuất ngoài Order bắt buộc tham chiếu và lý do nghiệp vụ.
- Hỏng/thất thoát bắt buộc số lượng dương và lý do; kiểm kê gửi số lượng đếm thực tế không âm.
- API Inventory chịu trách nhiệm kiểm tra tồn âm, Batch đủ điều kiện, Order hợp lệ, tham chiếu trùng và xung đột revision.
- Kết quả mutation là `APPLIED` hoặc `PENDING_APPROVAL`. Frontend không tự suy ra ngưỡng phê duyệt.

## 4. Những gì chưa tự suy diễn

- Không thêm kho/kệ/bin, chuyển kho hoặc phân công work item vì mô hình một hay nhiều kho chưa chốt.
- Không tự tính `available` hoặc thời điểm trừ tồn vật lý.
- Không triển khai reservation/release hay FEFO vì các chính sách liên quan chưa chốt hoặc thuộc luồng khác.
- Không cho sửa/xóa Batch đã có giao dịch.
- Không triển khai hàng đợi phê duyệt; adjustment mock trả `PENDING_APPROVAL` để giữ số dư không đổi cho đến khi Epic 22/23 và policy được chốt.
- Không thêm cảnh báo tồn tối thiểu vì Backlog chưa có User Story chính thức.

## 5. Chạy local

```powershell
# Quản lý tồn kho
$env:ADMIN_MOCK_PROFILE="WAREHOUSE_MANAGEMENT"

# Hoặc Workbench thao tác kho
$env:ADMIN_MOCK_PROFILE="WAREHOUSE_WORKBENCH"

npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
npm run dev
```

Mở `http://localhost:3001/inventory` hoặc `http://localhost:3001/work/warehouse` theo profile.
