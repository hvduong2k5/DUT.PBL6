# Gói Huế — Home và điều hướng khách hàng

**Trạng thái:** Đã triển khai bước Home, chờ người dùng duyệt giao diện.
**Nhánh:** `codex/storefront-home-navigation`. Mốc gốc `origin/dev` tại `b3957a89e390d57585ca0e77f30cdd33b4a68da3`. Commit chứa tài liệu này là checkpoint của bước Home.

## Phạm vi và quyết định

- Giữ Next.js, React, router và các provider của `apps/web-user`. Phạm vi hiện tại gồm Home, nhận diện ở header/footer và điều hướng chung trên những trang đang dùng SiteHeader/SiteFooter.
- Đối chiếu `mon-02/bo-ui-v01/Quy-chuan-UI.md` và `mon-02/home-v02/index.html` ở thư mục báo cáo Gói Huế. Dùng hero trưng bày nâu, ảnh thư viện, nền giấy, khung giật cấp, cành trà cạnh câu chuyện và cổng thành ở chân trang. Ảnh thuộc bộ minh họa môn học.
- Desktop gom nhóm mua sắm vào **Bộ sưu tập** và thông tin dịch vụ vào **Khám phá**. Giữ Quà biếu ở đầu trang để truy cập nhanh. Dưới 768 px dùng thanh đáy và bảng danh mục mở từ đáy. Dùng quyền và trạng thái đăng nhập hiện có để quyết định các mục B2B, nhân viên, đơn hàng và giỏ.
- Home dùng đúng slug danh mục, SKU và đường dẫn có sẵn. Bấm sản phẩm để chọn quy cách ở trang chi tiết. Giữ tra cứu lô, gợi ý và tùy chọn cá nhân hóa. Form email tiếp tục là demo cục bộ.
- Khu vực thay đổi: `apps/web-user/src/app/(storefront)/` (Home và import CSS), `src/components/layout/`, `src/styles/storefront-heritage.css`, `public/images/goi-hue/`. Bước sau mới xét nội dung từng trang con.

## Kiểm tra và bàn giao

- Typecheck, build production, ESLint, 59 file Vitest với 230 test đạt.
- Kiểm tra trình duyệt Edge bằng phiên khách. Ở 1440 px: dropdown bám dưới nút, dùng được bàn phím và Escape. Ở 390 px: mở/đóng bottom sheet và từ sheet đi đúng `/products?category=tra-cung-dinh`. Kiểm tra chiều rộng 320, 390 và 768 px. Tra mã mẫu dẫn đến `/trace/QR-OMA-20261015-LOT08`.
- Ảnh kiểm tra ở thư mục báo cáo local `mon-02/DUT-PBL6-home-review/`. Kiểm tra này chỉ xác nhận UI và điều hướng. Luồng có dữ liệu trực tiếp, tài khoản khách/nhân viên đăng nhập, đơn hàng và B2B chưa kiểm tra end-to-end với backend.
- Chạy lại trong `apps/web-user` bằng `npm ci`, `npm run build`, `npm run start -- -H 127.0.0.1 -p 4321`; mở địa chỉ loopback cổng 4321.
- Bước tiếp theo: người dùng duyệt Home, menu desktop và bottom sheet trên màn hình hẹp. Sau khi duyệt, thống nhất trang kế tiếp trước khi mở rộng phạm vi.
