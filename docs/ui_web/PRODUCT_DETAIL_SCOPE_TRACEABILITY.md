# Product Detail & SKU Selection MVP — Scope & Traceability

## Mục tiêu MVP

Hiện thực lát cắt customer-facing của EPIC 04 để Guest Customer và Registered Customer xem dữ liệu catalog đã được phép công khai, chọn đúng SKU và nhận đúng giá/thông tin thực phẩm trước khi chuyển sang luồng mua ở EPIC 05. Mockoon là executable example cho UI, không phải contract cuối cùng.

## Actor và user story

| Actor | User story | Phạm vi thực hiện |
| --- | --- | --- |
| Guest Customer, Registered Customer | `US-PROD-03` | Xem các SKU; chọn theo khối lượng, hương vị, quy cách; cập nhật giá và thông tin SKU; chặn SKU không khả dụng. |
| Guest Customer, Registered Customer | `US-PROD-04` | Xem thành phần, cảnh báo dị ứng, hướng dẫn bảo quản, chính sách ngày sản xuất và hạn sử dụng ở cấp catalog. |
| Guest Customer, Registered Customer | Phần đọc của `US-PROD-05` | Hiển thị giá bán VND của đúng SKU được chọn. |
| Guest Customer, Registered Customer | Phần đọc của `US-PROD-06` | Hiển thị availability public và không cho tiếp tục với SKU không mua được. |

## User story bị hoãn hoặc chỉ là dependency

- `US-PROD-01`, `US-PROD-02`, phần quản trị của `US-PROD-05` và `US-PROD-06`: UI Sales Manager không thuộc `apps/web-user`.
- `US-PROD-07`: Omnichannel Pricing thuộc Phase 2.
- Add-to-Cart, cart persistence và revalidation khi mua thuộc EPIC 05; Checkout thuộc EPIC 06.
- Exact inventory, reservation, Batch/Lot, ngày sản xuất/hạn dùng thực tế và chặn lô hết hạn thuộc EPIC 09.
- Traceability chi tiết thuộc EPIC 19; content/SEO nâng cao thuộc EPIC 20.

## Màn hình và route

- Product Detail: `/products/[slug]`.
- Product Discovery `/products`: card được nối bằng link ổn định tới Product Detail, không thay đổi semantics search/filter/sort/pagination.
- Dùng lại duy nhất `SiteHeader` và `SiteFooter`; không sao chép layout vào page.

## Happy path và UI state

1. Khách mở link sản phẩm từ Product Discovery.
2. UI tải product detail qua same-origin BFF `/api/catalog/products/{slug}`.
3. Gallery, breadcrumb, mô tả, thông tin thực phẩm và danh sách SKU public được hiển thị.
4. Với nhiều SKU, chưa có purchase selection ban đầu. UI hiển thị khoảng giá và yêu cầu chọn SKU.
5. Khách chọn một SKU khả dụng; giá, khối lượng, hương vị, quy cách, availability và ảnh SKU (nếu có) cập nhật theo lựa chọn.
6. CTA xác nhận SKU chỉ hiển thị integration notice cho EPIC 05, không gọi Cart API và không tạo cảm giác đã thêm hàng.

Các state bắt buộc:

- Loading skeleton có `aria-busy`.
- Error và upstream unavailable có retry.
- Invalid slug bị BFF từ chối với `400`.
- Product không tồn tại hoặc chưa được public trả `404` và UI có đường về `/products`.
- SKU unavailable vẫn hiển thị nhưng disabled, có lý do public phù hợp.
- Toàn bộ SKU unavailable: product vẫn hiển thị để đọc thông tin; CTA disabled.
- Thiếu thông tin optional: ẩn section tương ứng hoặc ghi rõ chưa công bố, không suy diễn dữ liệu.

## Quyết định nghiệp vụ

1. **Nhiều SKU:** không auto-select recommended/rẻ nhất. UI có thể tính khoảng giá để xem, nhưng `selectedSkuId` vẫn rỗng cho đến thao tác rõ ràng của khách. Display price không đồng nghĩa confirmed purchase selection.
2. **Một SKU:** auto-select khi product chỉ có đúng một SKU public và SKU đó khả dụng. Nếu SKU duy nhất unavailable thì không select.
3. **Dữ liệu thay đổi khi chọn SKU:** `priceVnd`, `weightGrams`, `flavor`, `packageType`, `isAvailable`, `unavailableReason` và `imageUrl` khi SKU có ảnh riêng.
4. **Unavailable:** SKU được giữ trong danh sách để khách hiểu lựa chọn nhưng control bị disabled. Chỉ dùng reason public như `Tạm hết hàng` hoặc `Tạm ngừng bán`; không lộ số lượng, kho, lô hoặc nguyên nhân vận hành nội bộ.
5. **Product hết availability:** không trả 404 nếu product vẫn public; trang chi tiết còn hữu ích cho việc đọc. Product unpublished/discontinued không public dùng cùng semantics `404` để không lộ sự tồn tại nội bộ.
6. **Food date:** catalog chỉ công khai policy, ví dụ `Ngày sản xuất được in trên bao bì` và `45 ngày kể từ ngày sản xuất`. Không trả ngày cụ thể của Batch/Lot.
7. **Price:** số nguyên VND, là giá bán Website đang hiển thị của SKU; không hiển thị giá gạch ngang, promotion hoặc giá kênh khác. Thuế/phí có bao gồm hay không vẫn cần Backend/Business xác nhận.
8. **Gallery và breadcrumb:** thuộc trang chi tiết và có trong Stitch nên triển khai. Heritage story chỉ dùng khi chính là `longDescription` đã approved của Product; không xây content feature riêng.
9. **OCOP:** chỉ hiển thị summary certification đã được catalog cho phép công khai (ví dụ số sao). Không triển khai QR, hồ sơ truy xuất hay thông tin lô.
10. **Cart integration:** CTA mang nhãn rõ `Giỏ hàng sắp ra mắt`. Sau khi có SKU hợp lệ, thao tác chỉ xác nhận SKU đã sẵn sàng cho EPIC 05 và giải thích chưa thêm vào giỏ; không gửi network request.

## Ngoài phạm vi

- Rating/review và nội dung “đã mua hàng”.
- Related products, bundle/combo, recommendation và cross-sell.
- Promotion, old price, countdown, loyalty.
- Shipping promise theo địa điểm, quantity/stock maximum và Buy Now.
- QR traceability, OCOP dossier, Batch/Lot cụ thể.
- Cart, checkout, payment, order hoặc inventory reservation.

Những khối này xuất hiện trong Stitch chỉ là visual reference; không được dùng dữ liệu giả để làm chúng trông như đã hoạt động.

## Dependency

| Dependency | Dữ liệu/quyết định cần |
| --- | --- |
| EPIC 03 | Link từ discovery và consistency của product identity/slug. |
| EPIC 05/06 | Sau này nhận `skuId`, kiểm tra lại giá và availability trước Cart/Checkout. |
| EPIC 09 | Nguồn tổng hợp `isAvailable`; exact inventory và Batch không public. |
| EPIC 19 | Quyền sở hữu traceability/chứng nhận chi tiết. |
| Backend/Architecture | Public projection, 404 semantics, cache/revalidation và error schema. |

## Mapping requirement → UI state → mock capability

| Requirement | UI state | Mock capability/scenario |
| --- | --- | --- |
| `US-PROD-03`, `FR-PROD-05` | Nhiều SKU, chưa chọn, validation khi xác nhận | `DETAIL-C01` default |
| `US-PROD-03`, `BR-PROD-02` | Chọn SKU cập nhật giá/quy cách | `DETAIL-C01` default |
| `US-PROD-03`, `FR-PROD-09` | SKU unavailable disabled | default + `all-unavailable` |
| `US-PROD-04`, `FR-PROD-06` | Food information và shelf-life policy | default + `missing-optional-food-info` |
| `US-PROD-05`, `FR-PROD-07` | Giá SKU bằng integer VND | `DETAIL-C01` default/single SKU |
| `US-PROD-06`, `FR-PROD-08` | Product public nhưng không thể mua | `all-unavailable` |
| Public/approved boundary | Not found/unpublished không bị phân biệt | `product-not-found`, slug không có trong mock |
| UI resilience | Loading, error, retry, upstream unavailable | `product-slow`, `product-error`, BFF `503` |

## Câu hỏi cần Backend/Architecture/Business xác nhận

1. `isAvailable` là projection tổng hợp từ catalog status và EPIC 09 theo SLA/cache nào; thời điểm Cart phải revalidate ra sao?
2. Bộ public unavailable reason chuẩn hóa gồm những code nào và text do Backend hay Frontend sở hữu?
3. Giá Website trả về đã bao gồm thuế/phí nào, có thời điểm hiệu lực/currency bắt buộc hay không?
4. Trường ngày sản xuất public nên là policy text, nhãn “xem trên bao bì”, hay schema có cấu trúc; ai phê duyệt nội dung?
5. Product discontinued nhưng vẫn cần SEO/history có còn public read-only hay dùng `404` như unpublished?
6. SKU image override và thứ tự gallery có được quản lý ở cấp SKU trong MVP không?
7. Dữ liệu OCOP summary nào đã được EPIC 19 phê duyệt để Product Catalog được phép public?
