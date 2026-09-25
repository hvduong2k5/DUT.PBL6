# Payment MVP — Scope & Traceability

## Mục tiêu vòng triển khai

Feature nối Order vừa được Checkout tạo với trải nghiệm thanh toán dành cho Guest/Customer. UI cho khách xem hướng dẫn chuyển khoản/QR mô phỏng hoặc nghĩa vụ COD, theo dõi trạng thái đã được backend xác minh và xử lý hết hạn/thất bại mà không tự đánh dấu Order là `PAID`.

## Phạm vi MVP Web D2C

| User story | Phạm vi UI | Trạng thái |
| --- | --- | --- |
| `US-PAY-01` | Khởi tạo/xem lần thử chuyển khoản hoặc QR; chờ, kiểm tra lại, thành công, thất bại, hết hạn, cần đối soát | Thực hiện |
| `US-PAY-02` | Hiển thị COD khi Checkout đã chọn COD; ghi rõ chưa thu tiền và không gắn `PAID` | Thực hiện |
| `US-PAY-03` | Guest/Customer đã được xác minh quyền sở hữu xem trạng thái Payment của đúng Order | Thực hiện |
| `US-PAY-04/05` | Nhân viên xem Payment và đối soát | Ngoài `web-user`; cần portal, RBAC và Audit riêng |

Màn hình chính dùng D2C-007 làm visual reference tại route `/payment/[orderId]`. Checkout `/checkout` chuyển tới route này sau khi server tạo Order thành công.

## Ngoài phạm vi

- Kết nối ngân hàng/cổng thanh toán thật, webhook thật hoặc tự động gạch nợ.
- QR có thể quét/chuyển tiền thật; local UI chỉ hiển thị QR-style demo và tài khoản giả.
- Tự xác nhận `PAID` từ thao tác “Kiểm tra trạng thái” hoặc browser return URL.
- Nhân viên điều chỉnh Payment, đối soát, hoàn tiền hoặc đánh dấu Paid thủ công.
- Quản lý vòng đời Order, fulfillment, shipping, notification và báo cáo tài chính.
- Tạo trang xác nhận/tracking Order D2C-008; thuộc EPIC 08.
- Lưu hoặc public thông tin nhạy cảm của tài khoản/thẻ.

## Quyết định nghiệp vụ và bảo mật

1. Payment status và Order status là hai trạng thái khác nhau. Chỉ kết quả đáng tin cậy từ backend/provider mới có thể tạo tác động `PAID`.
2. `BANK_TRANSFER` tạo Payment attempt có amount/reference do server cung cấp; browser không gửi hoặc sửa số tiền.
3. `COD` ghi nhận nghĩa vụ thu tiền, Order có thể được tiếp nhận theo policy nhưng Payment vẫn chưa thành công cho tới xác nhận thu tiền hợp lệ.
4. Guest không truyền token sở hữu qua URL. Checkout BFF cấp cookie ký `HttpOnly`, `SameSite=Lax`, giới hạn path `/api/payments`; Payment BFF kiểm tra chữ ký và `orderId`.
5. Cookie local chỉ chứa định danh Order, method, amount và thời hạn; không chứa người nhận, địa chỉ hay danh sách hàng.
6. `X-Mock-Scenario` chỉ dùng development. Production từ chối mock query.
7. Các nút sao chép chỉ sao chép fixture demo. UI phải luôn cảnh báo “không chuyển tiền thật”.
8. Countdown biểu diễn thời hạn Payment attempt/local reservation window; hết giờ không tự tạo attempt mới hoặc tự thay đổi Order.
9. Retry attempt dùng idempotency key; backend thật phải kiểm tra Order/reservation còn hợp lệ.

## Traceability UI

| UI/state | Requirement | Mock capability |
| --- | --- | --- |
| Checkout chuyển sang Payment | `FR-PAY-01/02` | Signed access cookie + `/payment/[orderId]` |
| Hướng dẫn QR/chuyển khoản demo | `US-PAY-01`, `FR-PAY-03` | `PAY-C01` |
| Chờ xác minh | `US-PAY-01/03` | `PAY-C02` mặc định |
| Thành công đã xác minh | `FR-PAY-04/05` | `payment-confirmed` |
| Thất bại/hết hạn | `FR-PAY-07` | `payment-failed`, `payment-expired` |
| Sai số tiền/tham chiếu | `FR-PAY-06/13` | `payment-mismatch` |
| COD chưa thu | `US-PAY-02`, `FR-PAY-08` | Method lấy từ signed Checkout context |
| Kiểm tra trạng thái/loading/error | `US-PAY-03` | `payment-slow`, `payment-error` |
| Tạo attempt mới khi còn hợp lệ | `FR-PAY-07` | `PAY-C03`, `retry-not-eligible` |
| Không có quyền/not found | `FR-PAY-10` | BFF ownership check, `payment-not-found` |

## Dependency cần Backend/Architecture/Security xác nhận

- EPIC 06/08: nguồn Order snapshot, trạng thái Order và điều kiện còn hiệu lực.
- EPIC 09: reservation TTL/release và điều kiện retry.
- Provider/Bank: QR payload, webhook signature, deduplication và trusted confirmation.
- Guest ownership: token/cookie production, vòng đời và khả năng mở lại trên thiết bị khác.
- COD: eligibility, trạng thái Order khởi đầu và nguồn xác nhận thu tiền.
- Security/Finance: dữ liệu nào được public, retention, audit và reconciliation handoff.
