# Admin Product & SKU Catalog — Data/API Matrix

**Cập nhật:** 2026-10-02  
**Backlog:** `EPIC_04_Product_SKU.md`  
**Màn hình:** `/catalog/products`  
**Ứng dụng:** `apps/web-admin`  
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

## 2. Permission và ranh giới truy cập

- Sidebar, Product list và Product detail yêu cầu `PRODUCT_VIEW`.
- Mock `SALES_MANAGER` có `PRODUCT_VIEW` với scope `PRODUCT: ALL`; `CUSTOMER_SERVICE`, `B2B_VIEWER` và `SYSTEM_ADMIN` không tự nhận quyền nghiệp vụ Product.
- BFF luôn gọi projection phiên nhân viên và trả `403` khi thiếu `PRODUCT_VIEW`. Việc ẩn menu không thay thế authorization phía server.
- Lát cắt này chưa định nghĩa Permission mutation vì Epic 04 chưa chốt taxonomy quyền thao tác Product/SKU/Price/Status.

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

## 5. Mutation đang tạm gác

Chưa triển khai tạo/sửa Product, tạo/sửa SKU, cập nhật giá hoặc thay trạng thái bán cho đến khi chốt:

1. Trường bắt buộc Product/SKU và quy tắc mã SKU.
2. Quy trình duyệt/công khai và actor được phép duyệt.
3. Quan hệ giữa thông tin thực phẩm catalog và Batch thực tế.
4. Xử lý SKU hết tồn, Product/SKU tạm dừng và giỏ hàng hiện hữu.
5. Đơn vị tiền, thuế/phí, thời điểm hiệu lực và ảnh hưởng tới giỏ đang giữ.
6. Thứ tự ưu tiên giữa giá cơ bản, giá kênh, promotion và báo giá B2B.

## 6. Chạy local

```powershell
# apps/web-admin/.env.local
ADMIN_API_UPSTREAM_URL=http://127.0.0.1:4030/api/v1
ADMIN_MOCK_ROLE=SALES_MANAGER

npx @mockoon/cli start --disable-log-to-file --data .\mocks\mockoon\admin_extensions.json
cd apps\web-admin
npm run dev
```

Mở `http://localhost:3001/catalog/products`.
