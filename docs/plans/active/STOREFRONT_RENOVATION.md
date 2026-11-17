# Trùng tu UI khách hàng

Phạm vi được duyệt gồm Home và toàn bộ giao diện khách hàng, điều hướng, gợi ý sản phẩm và đồng ý cá nhân hóa. Tiếp tục nhánh codex/storefront-home-navigation từ e5b8612. Backend, quyền truy cập, API giao dịch và luồng checkout dùng hiện hữu. Phong cách kế thừa font, màu và khung Gói Huế, tinh giản họa tiết và mật độ.

## Lộ trình hiện tại

1. Bỏ công cụ mock và chế độ giả lập. Chọn một hero, màn hình tự responsive.
2. Home ưu tiên rail Sản phẩm nổi bật từ recommend-engine. Khách chọn sản phẩm mở trang chi tiết hiện hữu. Danh mục nằm trong điều hướng và catalogue. Bỏ lưới sản phẩm tĩnh và phối hộp giả chưa có API đặt mua.
3. Rail ở chi tiết và giỏ đồng bộ cùng hệ thẻ, có trạng thái tải và thử lại. Cá nhân hóa có nhãn giải thích, chỉ chỉnh được cho khách đăng nhập và consent lấy thành công.
4. Đồng bộ kiểu chữ, form, bề mặt, khoảng cách và nút trên catalogue, chi tiết, giỏ, checkout, tài khoản, hỗ trợ, theo dõi đơn.
5. Build/lint/typecheck/test, kiểm tra trình duyệt 1440/390/320, thử gợi ý bằng API fixture và phân biệt với backend thật.
6. Commit và push checkpoint, chạy local cho duyệt.

## Tiêu chí nghiệm thu

Công cụ mock đã gỡ. Home có một luồng mua sắm, không gán SKU mẫu cho hàng thật. Khách vãng lai xem gợi ý. Tắt consent ngăn event cá nhân hóa theo provider hiện hữu. Fail API hiển thị trạng thái và thao tác phục hồi. Các trang dùng cùng font có đủ dấu tiếng Việt, vùng chạm 44px, không tràn màn hình. Không PR hoặc merge.

## Bàn giao

Đã triển khai phần giao diện khách hàng. Phạm vi intentionally modified gồm customer layout/Home, header/footer/mobile tabs, các metadata và nhãn thương hiệu khách hàng, catalog mobile filter, recommendation rail và CSS dùng chung cho customer routes.

Build production và TypeScript, ESLint và 230 test đã đạt. Edge với API fixture xác nhận khách vãng lai thấy rail Home, đi từ danh mục sang catalogue, tìm sản phẩm theo slug, consent PUT bật/tắt và khôi phục sau 503; không phát sinh JavaScript page error. Khảo sát các route Home, catalogue, chi tiết, giỏ, checkout, tài khoản, login, hỗ trợ, tra đơn ở 390 và 320px. Kiểm tra phát hiện checkout step tràn ngang và đã sửa theo bố cục xếp dọc. Họa tiết nền mobile giới hạn bề rộng. Bộ lọc catalogue mobile thu gọn mặc định.

Giới hạn kiểm chứng: backend recommendations live trả 503. Các ảnh desktop/mobile ở mon-02/DUT-PBL6-home-review/renovation dùng fixture rõ ràng, không phải dữ liệu lấy từ backend. Chưa xác nhận checkout, đăng nhập và event ingest end-to-end trên các dịch vụ thật. Các rail chi tiết/giỏ giữ placement/context và tracking từ mã hiện hữu.

Bản local build và chạy với NEXT_PUBLIC_RECOMMENDATIONS_ENABLED=true qua biến môi trường tiến trình. File env người dùng không được sửa. Dùng ANALYTICS_RECOMMENDATION_UPSTREAM_URL để cấu hình upstream khi thử thật.

Bước tiếp theo: duyệt toàn bộ diện mạo khách hàng ở cổng 4321, rồi kết nối dịch vụ thật để kiểm thử các luồng có dữ liệu. Commit chứa kế hoạch này là checkpoint trùng tu. Xem docs/CURRENT_STATE.md cho bàn giao.
