# Admin Product & SKU Catalog — Data/API Matrix

**Cập nhật:** 2026-11-06
**Backlog:** `EPIC_04_Product_SKU.md`
**Màn hình:** `/admin/catalog/products`
**Ứng dụng:** `apps/web-user`
**Mock Admin:** `mocks/mockoon/admin_extensions.json`, cổng `4030`

## 1. Phạm vi đã triển khai

| Khả năng | Nguồn Epic | Browser/BFF | Mock upstream | Trạng thái |
| --- | --- | --- | --- | --- |
| Danh sách Product | `US-PROD-01`, `US-PROD-06`, `FR-PROD-01`, `FR-PROD-08` | `GET /api/admin/catalog/products` | `GET /admin/catalog/products` | Đã tích hợp read-only |
| Status tabs, tìm kiếm và lọc danh mục | `US-PROD-01`, `US-PROD-06` | Xử lý trên projection đã tải | cùng list route | Đã tích hợp |
| Kiểm tra mức đầy đủ thông tin thực phẩm | `US-PROD-04`, `FR-PROD-02`, `FR-PROD-06` | list/detail projection | list/detail route | Đã tích hợp |
| Popup chi tiết Product | `US-PROD-01`, `US-PROD-04` | `GET /api/admin/catalog/products/:productId` | cùng path không có `/api` | Đã tích hợp |
| Danh sách Variant/SKU | `US-PROD-02`, `FR-PROD-03~04` | trong popup detail | detail route | Đã tích hợp read-only |
| Giá cơ bản theo SKU | `US-PROD-05`, `FR-PROD-07` | trong popup detail | detail route | Đã tích hợp read-only |
| Trạng thái bán Product/SKU | `US-PROD-06`, `FR-PROD-08` | list/detail | list/detail route | Đã tích hợp read-only |
| Cập nhật nội dung Product/SKU, giá, ảnh | `US-PROD-02`, `US-PROD-05`, `US-PROD-07` | `/api/admin/catalog/**` (PATCH/PUT/POST/DELETE) | backend `catalog-service` | Backend + BFF sẵn sàng; UI chưa nối mutation |

## 2. Permission và ranh giới truy cập

- Sidebar, Product list và Product detail yêu cầu `PRODUCT_VIEW` hoặc `CATALOG_VIEW`.
- Thao tác ghi yêu cầu `CATALOG_MANAGE`/`PRODUCT_MANAGE` hoặc role `ADMIN`.
- Mock `SALES_MANAGER` có `PRODUCT_VIEW` với scope `PRODUCT: ALL`; `CUSTOMER_SERVICE`, `B2B_VIEWER` và `SYSTEM_ADMIN` không tự nhận quyền nghiệp vụ Product.
- BFF luôn gọi projection phiên nhân viên và trả `403` khi thiếu quyền. Việc ẩn menu không thay thế authorization phía server.
- Taxonomy quyền theo backend: `CATALOG_VIEW` (đọc), `CATALOG_MANAGE` (ghi nội dung/SKU/giá/ảnh), `CATALOG_APPROVE` (duyệt vòng đời). Xem [`admin-api-state-matrix.md`](../03_api_specs/admin-api-state-matrix.md).

## 3. Ranh giới dữ liệu

- Product là thực thể catalog; SKU là đơn vị quản lý khối lượng, hương vị, quy cách và giá cơ bản.
- Giá trong màn hình là **giá cơ bản của SKU**, chưa diễn giải đã gồm thuế/phí hay thứ tự ưu tiên với promotion, B2B và giá đa kênh.
- `US-PROD-07` Omnichannel Pricing thuộc giai đoạn 2 và không có trong projection hiện tại.
- Không trả số lượng tồn, reservation, Batch/Lot, ngày sản xuất hoặc hạn sử dụng thực tế. Các dữ liệu này thuộc Epic 09.
- `manufacturingDatePolicy` và `shelfLifeDescription` là thông tin/chính sách catalog; không đại diện cho lô hàng thực tế sẽ giao.
- Ảnh chỉ là URL và alt text. Binary upload, scan và Cloudinary/object storage không thuộc route catalog read-only này.
- Nội dung marketing, revision bài viết và SEO thuộc Epic 20; Product detail chỉ giữ mô tả catalog cơ bản.

## 4. Trạng thái

Product và SKU hiện chỉ dùng các trạng thái bán đã được Epic nêu:

- `ON_SALE`: đang bán.
- `PAUSED`: tạm ngừng bán.
- `DISCONTINUED`: ngừng kinh doanh.

Không tự bổ sung `DRAFT`, `PENDING_APPROVAL` hoặc `PUBLISHED`, vì quy trình duyệt/công khai sản phẩm vẫn nằm trong Mục 8 cần Product Owner chốt. Product `ON_SALE` cũng không đảm bảo có thể mua nếu tồn kho hoặc Batch thực tế không hợp lệ.

## 5. Mutation đã có backend, UI chưa tích hợp

Backend `catalog-service` đã cung cấp đầy đủ API quản trị: PATCH Product (mọi nội dung gồm `story`), PATCH SKU, lịch sử/cập nhật giá, quản lý ảnh và transition vòng đời. BFF `/api/admin/catalog/**` đã forward được `POST/PUT/PATCH/DELETE` cùng query `q/status/page/pageSize`.

UI hiện vẫn là read-only. Khi nối mutation cần chốt thêm ở tầng UI:

1. Trường bắt buộc Product/SKU và quy tắc mã SKU (backend đã cố định `skuCode` bất biến).
2. Ai được duyệt/công khai (`CATALOG_APPROVE`).
3. Quan hệ giữa thông tin thực phẩm catalog và Batch thực tế.
4. Xử lý SKU hết tồn, Product/SKU tạm dừng và giỏ hàng hiện hữu.
5. Đơn vị tiền, thuế/phí, thời điểm hiệu lực và ảnh hưởng tới giỏ đang giữ.
6. Thứ tự ưu tiên giữa giá cơ bản, giá kênh, promotion và báo giá B2B.

Chi tiết contract và ví dụ: [`admin-api-state-matrix.md`](../03_api_specs/admin-api-state-matrix.md) và [`admin-draft-planned.openapi.yaml`](../03_api_specs/admin-draft-planned.openapi.yaml).

## 6. Chạy local

> `ADMIN_MOCK_PROFILE` chỉ chọn dữ liệu fixture upstream. Quyền truy cập thực tế lấy từ tài khoản `EMPLOYEE` và mảng `permissions` do Identity Service trả về; đổi profile Mockoon không cấp thêm quyền.

```powershell
# apps/web-user/.env.local
ADMIN_API_UPSTREAM_URL=http://127.0.0.1:4030/api/v1
ADMIN_MOCK_PROFILE=SALES_MANAGER

npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-user
npm run dev
```

Mở `http://localhost:3000/admin/catalog/products`.
