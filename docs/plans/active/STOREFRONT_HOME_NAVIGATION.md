# Gói Huế — Home và điều hướng khách hàng

**Trạng thái** · Chờ duyệt bản đối chiếu UI/UX v2. Nhánh `codex/storefront-home-navigation` tách từ `dev` tại `b3957a8`. Checkpoint trước là `37053ca`. Commit chứa tài liệu này là checkpoint sửa sai khác với mẫu.

## Nguồn để triển khai

- Thư mục báo cáo local `E:/Goi-Hue-Brand-Report/mon-02`. Dùng `home-v02/index.html`, `home-v02/home.css` và `home-v02/home.js` cho thứ tự, nội dung trình bày, trạng thái và tương tác. Dùng `bo-ui-v01/ui.css` cùng SVG, hình sản phẩm và họa tiết đi kèm cho phần nhận diện.
- Khi làm trang sau, dùng nguyên thành phần tương ứng trong thư viện. Điều chỉnh bề rộng, khoảng cách và xuống dòng theo màn hình. Ghi rõ sai khác nếu mẫu mô phỏng có thao tác mà dịch vụ hiện tại chưa hỗ trợ.
- Giữ React/Next.js, routing, session, customer access, phân quyền B2B/nhân viên, cart API và các trang mua hàng hiện hữu.

## Đã triển khai

- Home dùng hai hero chuyển đổi được, danh mục, bộ lọc và sản phẩm, phối hộp quà và hai vùng câu chuyện theo mẫu. Giữ phần tra cứu lô, đăng ký tin và thành phần recommendation của ứng dụng phía dưới. SKU, giá và đường dẫn sản phẩm đến từ Home hiện hữu của DUT.PBL6.
- Header dùng menu bộ sưu tập có ảnh. Tìm kiếm hỗ trợ tiếng Việt có dấu/không dấu. Web mở dropdown hoặc hộp chi tiết. Điện thoại dùng thanh đáy 5 mục, bottom sheet và chi tiết toàn màn hình. Ở thanh trên cùng, mục Tài khoản dẫn tới các tiện ích của ứng dụng, thay vị trí liên kết thư viện dành cho người thiết kế.
- Kiểu chữ Playfair Display và Be Vietnam Pro phục vụ từ `public/fonts/goi-hue` kèm hai giấy phép OFL. Lỗi trước đó là Georgia trên máy thiếu glyph như ế, ị, ử. Khung dùng đúng SVG và tỉ lệ cắt 14 đơn vị của mẫu. Nút có viền trong 8 px, hero có viền 16 px. Giữ chữ thương hiệu và nhãn điều hướng trên một dòng.
- Phạm vi mã gồm `apps/web-user/src/components/storefront/`, header/footer/logo ở `components/layout/`, Home và layout trong `src/app/(storefront)/`, stylesheet storefront, hình và font. CSS mẫu được giới hạn phạm vi để các trang quản trị và thân trang con giữ giao diện hiện hữu.

## Tích hợp và điểm cần duyệt

- Thẻ Home giữ dữ liệu sản phẩm của ứng dụng. Hộp chi tiết tải quy cách từ product-detail-service. Giỏ nhanh đọc giỏ thật và cập nhật số lượng qua cart-service. Đặt hàng tiếp tục vào luồng checkout hiện tại.
- Phối hộp riêng có bốn màu, ba tổ hợp, thiệp và giá mẫu. API thêm vào giỏ nhận SKU và số lượng; chưa có trường cho tổ hợp, màu và thiệp. CTA mở bản tóm tắt và dẫn sang hỗ trợ đặt riêng. Bản mẫu Antigravity dùng giỏ giả trong phiên. Cần quyết định về hợp đồng dữ liệu nếu triển khai mua hộp phối riêng.
- Đã kiểm tra build, lint, typecheck và 230 test. Edge với phiên khách đã kiểm tra menu có ảnh, lọc/sắp xếp, tìm kiếm, đổi hero, xem chi tiết, đổi giấy/tổ hợp/thiệp và bottom sheet. Kiểm tra chiều rộng 1440, 768, 390 và 320 px. Luồng giao dịch có tài khoản và backend thật chưa kiểm thử đầu cuối.
- Ở desktop 1440 px, so trực tiếp với bản mẫu ở cùng font: hero 1280 × 520 px, ảnh hero rộng 747 px, thẻ danh mục đầu rộng 408 px. Ảnh đối chiếu local nằm tại `mon-02/DUT-PBL6-home-review/`.

## Bàn giao

Chạy trong `apps/web-user` bằng `npm ci`, `npm run build`, rồi `npm run start -- -H 127.0.0.1 -p 4321`. Duyệt Home và điều hướng ở cổng 4321 trước khi tiếp tục sang trang khác.
