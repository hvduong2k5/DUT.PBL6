# Customer Core UI Completion Matrix

**Cập nhật:** 2026-10-01  
**API nguồn:** `mocks/mockoon/mobile_pbl.json`  
**Ưu tiên:** Customer D2C trước Web Admin/RBAC

Tài liệu này theo dõi mức hoàn thiện giao diện customer dựa trên file Mockoon tổng hợp. Đây là ma trận coverage UI, không phải tài liệu review API contract.

## Trạng thái

- **Đã tích hợp:** UI đang gọi endpoint trong `mobile_pbl.json` qua Next.js BFF.
- **Có UI cũ:** màn hình đã tồn tại nhưng còn dùng Mockoon MVP riêng và cần chuyển sang nguồn customer core.
- **Chưa có UI:** endpoint có trong Mockoon tổng hợp nhưng chưa có trải nghiệm hoàn chỉnh trong web.
- **Không có UI trực tiếp:** callback/webhook do hệ thống ngoài gọi.

## Coverage

| Nhóm | Màn hình/customer flow | Endpoint trong `mobile_pbl.json` | Trạng thái hiện tại | Công việc tiếp theo |
|---|---|---|---|---|
| Authentication | Đăng ký, đăng nhập, làm mới phiên, Auth Guard và đăng xuất | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me` | **Đã tích hợp** | Xác minh số điện thoại chờ API hỗ trợ |
| Authentication extension | Yêu cầu, xác minh liên kết và đặt lại mật khẩu | `POST /auth/recovery-requests`, `/verify`, `/reset` trong `customer_extensions.json` | **Đã tích hợp** | Thay mock mở rộng bằng API backend khi contract chính thức có sẵn |
| Customer capability extension | Projection actor/capability cho Guest và Registered | `GET /customer/capabilities` trong `customer_extensions.json` | **Đã tích hợp** | B2B cần projection theo organization/contract; Marketplace và Offline tiếp tục được xem là channel/customer source, không mặc định là web role |
| Discovery | Danh mục | `GET /categories` | **Đã tích hợp** | Bổ sung thêm nhóm danh mục khi API có dữ liệu |
| Discovery | Danh sách, lọc, tìm kiếm và lọc chứng nhận OCOP 3–5 sao | `GET /products`, `GET /products/search` với `ocop_star` | **Đã tích hợp** | Khối lượng, loại sản phẩm và trạng thái tồn kho chờ API hỗ trợ bộ lọc tương ứng |
| Product | Chi tiết và chọn biến thể | `GET /products/:id_or_slug` | **Đã tích hợp** | Thông tin thực phẩm sẽ hiển thị khi API bổ sung trường dữ liệu |
| Cart | Xem, thêm, cập nhật, xóa món, dọn giỏ và cart badge toàn site | `GET/DELETE /cart`, `POST /cart/items`, `PUT/DELETE /cart/items/:item_id` | **Đã tích hợp** | Theo dõi khả năng hợp nhất cart Guest/Registered khi API bổ sung quy tắc |
| Checkout | Guest/Registered Checkout, địa chỉ hành chính hai cấp, chọn địa chỉ tài khoản, áp dụng voucher và nhận VietQR | `GET /profile`, `GET /profile/addresses`, `POST /checkout`; `GET /locations/checkout` trong `customer_extensions.json` | **Đã tích hợp** | UI chỉ dùng Tỉnh/Thành phố → Phường/Xã; địa chỉ tài khoản được sao chép thành snapshot của đơn |
| Orders | Danh sách và chi tiết đơn | `GET /orders`, `GET /orders/:order_id` | **Đã tích hợp** | Bổ sung tên sản phẩm snapshot khi API chi tiết cung cấp ngoài `sku_code` |
| Orders | Hủy và tracking | `POST /orders/:order_id/cancel`, `GET /orders/:order_id/tracking` | **Đã tích hợp** | Bổ sung ETA khi API tracking cung cấp |
| Guest Order extension | Cấp quyền ngay sau Guest Checkout và OTP tra cứu lại đơn | `POST /orders/guest-access/grants`, `POST /orders/guest-access/challenges`, `POST /orders/guest-access/challenges/:id/verify` trong `customer_extensions.json` | **Đã tích hợp** | Guest cookie do BFF ký, có hạn và ràng buộc đúng một `orderId`; backend thật cần giữ cùng invariant |
| Profile | Xem/cập nhật họ tên và email liên hệ | `GET/PUT /profile` | **Đã tích hợp** | Ngày sinh/giới tính/đổi số điện thoại và xác minh email chờ API hỗ trợ |
| Address | Danh sách, thêm, sửa và xóa địa chỉ theo hai cấp Tỉnh/Thành phố → Phường/Xã | `GET/POST /profile/addresses`, `PUT/DELETE /profile/addresses/:address_id`; `GET /locations/checkout` trong `customer_extensions.json` | **Đã tích hợp** | Trường `district` cũ chỉ còn là tương thích upstream và không xuất hiện trong projection/UI Customer |
| Promotion | Danh sách, kiểm tra và áp dụng voucher tại Checkout | `GET /promotions/vouchers`, `POST /promotions/validate` | **Đã tích hợp** | Bổ sung voucher cá nhân hóa khi API trả dữ liệu theo phiên thành viên |
| Loyalty | Điểm và hạng thành viên | `GET /loyalty/points` | **Đã tích hợp** | Bổ sung lịch sử/quy đổi khi API hỗ trợ |
| Reviews | Danh sách và gửi đánh giá | `GET /reviews/products/:product_id`, `POST /reviews` | **Đã tích hợp** | Sau này bổ sung media upload/eligibility khi API hỗ trợ |
| Return | Tạo và xem yêu cầu đổi trả | `POST /returns`, `GET /returns/:return_id` | **Đã tích hợp** | Cần Media Upload API để thay ô nhập URL bằng chứng; bổ sung ticket không có endpoint customer |
| Traceability | Hồ sơ nguồn gốc theo QR | `GET /trace/:qr_code` | **Đã tích hợp** | Bổ sung camera scan khi có yêu cầu và permission trình duyệt |
| Customer Support extension | Tạo, xem và phản hồi ticket hỗ trợ | `GET /support/context`, `POST /support/tickets`, `GET /support/tickets/:id`, `POST /support/tickets/:id/messages` trong `customer_extensions.json` | **Đã tích hợp** | Media upload/scan thật chờ API hạ tầng tệp |
| Payment integration | VietQR từ Checkout và callback | `POST /checkout`, `POST /payments/vietqr/callback` | **Đã tích hợp phần customer** | Callback không có UI trực tiếp; trạng thái được đọc qua Order |
| Marketplace/Logistics | Shopee, TikTok và GHN webhook | `POST /webhooks/...` | Không có UI trực tiếp | Không tạo customer screen |

## Thứ tự triển khai đề xuất

1. Chuyển Product Discovery/Detail sang customer core để tạo luồng duyệt sản phẩm thống nhất.
2. Chuyển Cart → Checkout → Order theo một vertical flow hoàn chỉnh.
3. Chuyển Profile/Address và Return.
4. Chuyển Authentication cuối đợt đồng bộ để tránh làm gián đoạn các màn hình MVP đang dùng auth mock riêng. **Đã hoàn tất.**

## Ngoài phạm vi file Mockoon này

Các màn hình customer trong backlog như Social Login, Guest Checkout, B2B, Content, Notification và Gifting không được coi là hoàn tất chỉ từ `mobile_pbl.json`. Password Recovery, Guest Order và Customer Support hiện dùng chung `customer_extensions.json`; các capability còn lại tiếp tục chờ API bổ sung.

## Quy ước mock mở rộng

`mobile_pbl.json` tiếp tục là nguồn Customer API gốc và không bị sửa. Mọi endpoint, payload hoặc permission projection chưa có trong file này được gom vào duy nhất `mocks/mockoon/customer_extensions.json`; không tạo thêm mock riêng theo từng MVP. File mở rộng hiện chứa Password Recovery, Guest Order, Customer Support và Customer capability projection, chạy tại `http://127.0.0.1:4020/api/v1`.

## Lưu ý Mockoon

Trong `mobile_pbl.json`, route động `GET /products/:id_or_slug` hiện đứng trước `GET /products/search`, khiến Mockoon có thể hiểu `search` là một slug. BFF catalog có fallback tìm kiếm cục bộ chỉ ở môi trường development để giao diện vẫn kiểm thử được; production vẫn sử dụng endpoint tìm kiếm đúng như API.
