# Product Discovery MVP — Scope & Traceability

## Mục tiêu vòng triển khai

Feature này hiện thực phần khám phá sản phẩm công khai của EPIC 03 trên UI Web, dùng Mockoon để FE kiểm chứng luồng trước khi đề xuất API Contract.

## Phạm vi MVP

| User story | Phạm vi UI | Trạng thái |
| --- | --- | --- |
| `US-DISC-01` | Xem catalog, duyệt theo danh mục, empty state | Thực hiện |
| `US-DISC-02` | Tìm kiếm theo từ khóa, không có kết quả, từ khóa không hợp lệ | Thực hiện |
| `US-DISC-03` | Lọc giá, khối lượng, loại sản phẩm, còn hàng; kết hợp/xóa/reset bộ lọc | Thực hiện |

Màn hình chính là `D2C-002 Danh mục & Tìm kiếm`, route `/products`. `D2C-001 Trang chủ & Khám phá` chỉ được dùng làm tham chiếu thị giác và điểm vào catalog trong vòng này.

## Ngoài phạm vi

- Gợi ý bán chạy và gợi ý chung (`US-DISC-04`) thuộc Phase 2.
- Câu chuyện thương hiệu/văn hóa (`US-DISC-05`) thuộc Phase 2.
- Trợ lý AI và cá nhân hóa thuộc Phase 3; không triển khai `D2C-003`.
- Chi tiết sản phẩm thuộc EPIC 04. Card ở catalog chỉ cung cấp thông tin tóm tắt/offer khớp điều kiện.
- Giỏ hàng/mua ngay thuộc EPIC 05.
- Bộ lọc rating được ẩn cho tới khi EPIC 15 cung cấp nguồn rating thật; không tạo dữ liệu đánh giá giả.

## Quyết định nghiệp vụ

1. Điều kiện giá, khối lượng, kiểu đóng gói và tồn kho áp dụng ở cấp SKU/offer. Mỗi product result trả về `matchedOffer` là SKU bán được phù hợp với bộ lọc hiện tại.
2. Catalog chỉ trả sản phẩm đã public/approved và SKU còn hiệu lực. Lô hết hạn không được tính là tồn kho khả dụng.
3. Giá dùng số nguyên VND; khối lượng dùng số nguyên gram.
4. Tìm kiếm bỏ khoảng trắng thừa, không phân biệt hoa thường. API thật được đề xuất hỗ trợ tìm kiếm tiếng Việt không phân biệt dấu; giới hạn từ khóa 2–100 ký tự.
5. Catalog cho phép không có từ khóa. Khi người dùng chủ động submit ô tìm kiếm rỗng, UI hiển thị validation thay vì gửi request vô nghĩa.
6. URL là nguồn trạng thái của query/filter/sort/page để có thể bookmark, chia sẻ và dùng nút Back/Forward.

## Traceability UI

| UI/state | Requirement | Mock capability |
| --- | --- | --- |
| Danh sách public catalog | `US-DISC-01` | `DISC-C01` |
| Chọn danh mục | `US-DISC-01` | `DISC-C01 category` |
| Tìm theo từ khóa | `US-DISC-02` | `DISC-C01 q` |
| Empty/no-result | `US-DISC-01/02` | Kết quả `items=[]` |
| Price/weight/type/stock filters | `US-DISC-03` | `DISC-C01` filters |
| Filter chips, remove, reset | `US-DISC-03` | URL search params |
| Sorting | Hỗ trợ khám phá | `DISC-C01 sort` |
| Pagination | Catalog growth | `DISC-C01 page/pageSize` |
| Loading/error/retry | UI resilience | `catalog-slow`, `catalog-error` |

## Dependency cần Backend/Architecture xác nhận sau UI review

- EPIC 04: trạng thái public/approved, product summary và SKU sellable.
- EPIC 09: cách xác định `isAvailable`, tồn kho khả dụng và lô hết hạn.
- EPIC 15: rating aggregate; cho tới lúc đó rating filter không xuất hiện.
- Search engine: cơ chế normalize dấu tiếng Việt và xếp hạng relevance.
- Pagination/sort stability khi catalog thay đổi.
