# ADR-007 — Cách ly xử lý IPN và ghi nhận thiếu bằng chứng thanh toán

- Ngày: 16/10/2026.
- Trạng thái: Accepted cho profile `mvp1` sandbox.
- Bổ sung ADR-002 và ADR-006 sau khi thử ngoại lệ với merchant sandbox thật.

## Vấn đề đã tái hiện

1. Worker xử lý IPN và gọi API đối soát dùng chung một goroutine. Khi provider query chậm, IPN đã lưu vào database nhưng chưa được xử lý; test giữ request query đang mở cho thấy đơn không chuyển PAID kịp dù IPN hợp lệ đã ACK 200.
2. Khi backend bị tắt, SePay nhận 503. Sau khi bật lại, API detail báo CAPTURED nhưng transactions rỗng. Trước sửa, đối soát không tạo receipt và cũng không tạo cảnh báo nghiệp vụ; đơn tiếp tục chờ rồi hết hạn.
3. Nếu biến cảnh báo thiếu transaction thành operational hold, IPN hợp lệ đến sau trong thời hạn lại không được phân bổ tiền. Test query warning → valid IPN đã tái hiện vấn đề này trước khi sửa.

## Quyết định

- Worker IPN và worker query chạy độc lập, dùng cùng cơ chế shutdown có chờ hoàn tất. Query vẫn có timeout tổng 90 giây, nhỏ hơn lease 2 phút; IPN không phải chờ query kết thúc.
- Khi query xác thực trả CAPTURED, không có transaction và chưa có receipt thành công tương ứng merchant/invoice, lưu observation mã nội bộ `QUERY_CAPTURED_WITHOUT_TRANSACTION`. Payload lưu mã hóa, deduplicate như inbox hiện có; không giả đây là callback do provider gửi.
- Observation được đưa vào hàng chờ MANUAL_REVIEW với lý do `PROVIDER_TRANSACTION_PROOF_MISSING`. Không tạo receipt, allocation hay PAID từ CAPTURED đơn thuần.
- **Observation thiếu bằng chứng không tự bật operational hold.** Đơn đang PENDING_PAYMENT vốn chưa được phép giao hàng; một IPN hợp lệ đến sau trong thời hạn vẫn phải xử lý bình thường. Các lỗi tài chính khác như payload sai, transaction mâu thuẫn hoặc thông báo bất lợi vẫn giữ quy tắc hold hiện có.
- Nếu chưa nhận được transaction đến khi hết hạn, timeout và trả hàng giữ chỗ vẫn hoạt động. Tiền đến sau được lưu vào đối soát; không phục hồi đơn đã hủy. Cảnh báo query cũ được giữ làm dấu vết cho người vận hành; chưa tự đóng sau một IPN mới.

## Đánh đổi

- Có thêm một goroutine nhưng không có thêm database, queue bên ngoài hay service mới.
- Cảnh báo giúp người vận hành thấy mất bằng chứng; không thay thế bằng chứng giao dịch thật. Tự phục hồi thành PAID vẫn bị giới hạn nếu sandbox không trả transaction và provider chưa gửi lại IPN hợp lệ.
- Không giữ hàng vô thời hạn chỉ vì CAPTURED thiếu chi tiết. Có thể phải xử lý khoản chuyển đến muộn và hoàn tiền thủ công sau khi xác minh được giao dịch.
- Vẫn cần vận hành kiểm tra các notification MANUAL_REVIEW; chưa có công cụ tự lấy lại payload thất bại từ dashboard merchant.

## Kiểm chứng

- `TestMVPSlowReconciliationDoesNotBlockIPN`: query bị giữ mở nhưng IPN vẫn được xử lý và đơn PAID.
- `TestMVPReconcileFailureDoesNotAdvanceOrInventMoney`: HTTP lỗi không tiến cursor; sai định danh không tạo tiền; CAPTURED thiếu transaction có cảnh báo; IPN hợp lệ đến sau vẫn hoàn tất.
- `TestMVPStockExpiryAndPaymentTimeoutRace`: nhận tiền khi mất giữ chỗ/đua với timeout không phát paid/ready sai.
- Bộ test cuối: 65 test chính, 29 ca con; không lỗi, không bỏ qua test nghiệp vụ; có `-race` và `go vet`.
- Bản backend sau sửa đã thanh toán qua IPN SePay sandbox thật thêm một lần, đạt PAID / CONFIRMED / COMMITTED.

Xem [báo cáo kiểm thử đầy đủ](../../../../04_testing/order-service/SEPAY_MVP_TEST_REPORT.md) và [bằng chứng máy đọc](../../../../04_testing/order-service/mvp1-expanded-verification.json).
