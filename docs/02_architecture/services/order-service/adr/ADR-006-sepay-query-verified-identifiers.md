# ADR-006 — Tra cứu SePay theo invoice đã kiểm chứng

- Ngày: 2026-10-16.
- Trạng thái: Accepted cho sandbox, sau kiểm chứng merchant thật.

## Bối cảnh và bằng chứng

Lần thử merchant sandbox thật xác nhận `/v1/order/detail/{order_invoice_number}` trả HTTP 200. Dùng `order_id` dạng `PAY...` hoặc resource `id` dạng UUID tại cùng endpoint trả HTTP 404. Kiểm thử giả lập trước đây đã mô phỏng tra cứu theo `order_id`, vì vậy chưa phát hiện sai khác này.

Đơn đã được bấm “Giả lập thanh toán” có `order_status=CAPTURED`, nhưng phản hồi detail quan sát được có `transactions=[]`. Không được suy ra một receipt chỉ từ trạng thái tổng hợp.

## Quyết định

- Tra cứu detail theo merchant invoice lấy từ danh sách của merchant đã xác thực. Kiểm tra **cả invoice và provider order_id** trong kết quả trước khi xử lý giao dịch.
- Timeout mỗi HTTP request là 15 giây, thay cho 2 giây; thử từ máy phát triển có request vượt 2 giây. Tổng một lượt đối soát giới hạn 90 giây, nhỏ hơn lease 2 phút. Không tiến cursor khi lỗi; lần chạy sau thử lại.
- Gửi Accept JSON và User-Agent xác định service. Không log header Basic Auth hoặc response thô.
- Chỉ tạo receipt từ transaction PAYMENT/APPROVED hợp lệ. Khi provider không trả transactions, không giả transaction từ CAPTURED và không tự đánh dấu PAID.
- IPN vẫn bắt buộc `X-Secret-Key`. Log IPN merchant cho thấy một request thật tới URL đúng nhưng chỉ có Content-Type, nhận 401. Cấu hình SePay phải chọn **Auth Type: Secret Key**, không phải “Không có”. Không bỏ xác thực để làm bài test thành công.

## Đánh đổi và giới hạn

Request có thể giữ worker lâu hơn; giới hạn tổng thời gian bảo vệ lease. Đối soát vẫn chạy theo trang, lỗi ở một detail có thể khiến trang đó phải chạy lại. Việc tách worker nhận IPN khỏi worker query và cải thiện phân trang cần đánh giá khi tăng tải.

Kết quả tại thời điểm ghi: đã mở được trang BANK_TRANSFER sandbox đúng invoice/số tiền và thực hiện mô phỏng thanh toán; **chưa chứng minh Order PAID qua IPN**, vì request thật thiếu header xác thực. Unit/race test SePay, vet và các integration test `TestMVP` đã qua sau sửa. Không thay thế bằng chứng 61 test cũ bằng kết quả chạy một tập con này.

Cập nhật 14:17 ngày 16/10/2026 (Việt Nam): sau khi merchant chọn Auth Type Secret Key, một đơn sandbox mới 135.000 VND đã nhận IPN thật HTTP 200, notification APPLIED, một receipt; Order PAID / payment CONFIRMED / stock COMMITTED. [Bằng chứng đã bỏ bí mật](../../../../04_testing/order-service/live-sepay-sandbox-verification.json). Kết quả này xác nhận luồng IPN thành công, chưa chứng minh phục hồi mất IPN trong trường hợp detail không có transactions.

## Tham chiếu

Phần worker IPN/query và cảnh báo thiếu transaction được bổ sung bởi [ADR-007](ADR-007-payment-recovery-exception-tests.md) sau vòng kiểm thử ngoại lệ; không còn dùng chung một worker cho hai đường xử lý.

- [SDK chính thức — retrieve theo invoice](https://developer.sepay.vn/vi/cong-thanh-toan/sdk/nodejs).
- [Quy định X-Secret-Key](https://developer.sepay.vn/vi/cong-thanh-toan/IPN).
- [ADR-005 — Thiết lập local](ADR-005-local-sepay-sandbox-validation.md).
