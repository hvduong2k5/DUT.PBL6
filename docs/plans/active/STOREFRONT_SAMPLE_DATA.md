# Dữ liệu mẫu từ SQL cho duyệt UI

Phạm vi gồm trích sản phẩm/danh mục công khai, API đọc catalogue/chi tiết/gợi ý và chuyển nguồn mock/real/auto qua cấu hình server. Giữ file SQL nguồn nguyên vẹn. Không đưa dữ liệu khách hàng, mật khẩu, phiên, đơn hàng hoặc consent vào fixture.

## Quyết định

STOREFRONT_DATA_MODE mặc định real; local preview chạy auto. mock luôn đọc fixture, real giữ API hiện hữu, auto fallback khi API đọc lỗi mạng/5xx hoặc kho thật rỗng. Giữ 400/401/403, sản phẩm không tồn tại trong kho thật, và kết quả lọc rỗng. Khi một request dùng fixture, phản hồi gắn X-Storefront-Data-Source: mock và UI thông báo dữ liệu mẫu.

SQL thiếu SKU và chứng nhận. SKU mẫu dùng tiền tố DEMO-, khối lượng lấy từ tên nếu có; không gán chứng nhận. Ảnh SQL đều placeholder; chỉ dùng ảnh minh họa Gói Huế khi có ảnh phù hợp nhóm, giữ tên và mô tả gốc. Không gán ảnh mè xửng cho bánh in.

## Các bước

1. Trích file JSON chỉ từ product/category, ghi hash nguồn.
2. Ánh xạ sang contract catalogue và recommendation hiện hữu, lọc/sắp xếp/phân trang thống nhất.
3. Bọc API đọc bằng lựa chọn nguồn, giữ một nguồn trong từng phản hồi.
4. Test hồi quy chế độ, kho rỗng, lọc rỗng, lỗi quyền và slug không tồn tại.
5. Build và duyệt local, cập nhật bàn giao, commit/push nhánh hiện tại.

Đã hoàn tất. 5 sản phẩm / 4 danh mục trích từ SQL, hash nguồn B2A0B91BC3FBDF652CE063E1AEC400B34AADB420E7CA945A1C298B92AFDA2DB0. Xuất lại JSON cho kết quả byte-identical; SQL gốc giữ nguyên.

TypeScript và production build đạt. 60 file test / 235 test đạt, gồm các hồi quy real/mock/auto, 400/401/403, lỗi upstream, kho rỗng và lọc rỗng. Edge gọi trực tiếp API local ở auto, không intercept request: danh sách 5 hàng, bộ lọc, detail theo slug và rail home/product_detail/cart với loại trừ sản phẩm hiện tại đều đạt. Ảnh preview ở mon-02/DUT-PBL6-home-review/sample-data.

Phần sửa gồm JSON công khai, script xuất, bộ chọn nguồn và route đọc catalog/recommendations, nhãn dữ liệu mẫu, guard SKU DEMO ở cart và tracking, script khởi chạy và env example. Consent/đơn hàng không được giả lập.

Checkpoint là commit chứa tài liệu này trên nhánh codex/storefront-home-navigation. Bước tiếp theo người dùng duyệt dữ liệu mẫu trước khi tiếp tục UI khác.

## Cách chạy và chuyển nguồn

Trong apps/web-user, build một lần với NEXT_PUBLIC_RECOMMENDATIONS_ENABLED=true. Sau đó chọn một trong các lệnh sau. Dừng tiến trình preview đang chạy trước khi đổi chế độ, vì cùng cổng 4321.

~~~powershell
$env:NEXT_PUBLIC_RECOMMENDATIONS_ENABLED="true"
npm run build
npm run preview:data -- auto
~~~

Dùng npm run preview:data -- mock để luôn đọc JSON, hoặc npm run preview:data -- real để chỉ gọi dịch vụ. Đổi chế độ chỉ cần khởi động lại tiến trình và tải lại trang trình duyệt. Ngoài preview script, cấu hình server STOREFRONT_DATA_MODE có ba giá trị tương tự và mặc định real. Không cần chỉnh mã component hoặc file .env cá nhân.

Muốn xuất lại JSON từ bản SQL mới, dùng python scripts/export-goihue-sample.py "D:/Dyland/Downloads/goihue_backup.sql". Chỉ đọc COPY records của product/category; không chạy SQL hoặc import vào database.

File SQL thiếu SKU và khối lượng cho hộp quà, nên SKU là định danh DEMO riêng; 0g là khối lượng chưa khai báo. Ảnh theo nhóm là ảnh minh họa trong thư viện đã có. Bánh in giữ placeholder. Tên danh mục “Mức” và giá 74.998đ được giữ đúng SQL. Không xác nhận có chứng nhận OCOP.

Phạm vi fallback là API đọc sản phẩm, danh mục và gợi ý. Consent, sự kiện và đặt hàng tiếp tục dùng luồng thật. Sản phẩm DEMO không gửi vào giỏ thật. Các rail catalog-preview được loại khỏi đo lường và provenance, tránh giả kết quả recommend-engine.
