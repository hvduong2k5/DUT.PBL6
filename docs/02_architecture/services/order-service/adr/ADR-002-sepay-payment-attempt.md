# ADR-002 — SePay checkout và khoản thu độc lập

Ngày: 15/10/2026. Quyết định: accepted cho sandbox.

## Quyết định

Một Order có một payment attempt bền vững, mã hóa đơn merchant cố định. API trả danh sách trường form **có thứ tự**, chữ ký tạo trên server và action URL của SePay; frontend giữ thứ tự khi tạo form POST. Tải lại không tạo invoice mới. Chặn cấp form khi đơn hết hạn, có hold hoặc đã nhận tiền.

IPN yêu cầu cấu hình SePay auth type SECRET_KEY và so sánh `X-Secret-Key` bằng constant-time. Không áp dụng chữ ký simulator lên IPN SePay. Không dùng redirect success để cập nhật tiền. Ghi payload đã mã hóa trước xử lý để sự kiện chưa hiểu vẫn có thể review. Khoản thu hợp lệ được chuẩn hóa và chống trùng theo provider/môi trường + transaction.id (khóa tài nguyên dùng chung giữa IPN và query).

Đối chiếu định kỳ đọc danh sách đơn từ SePay rồi chi tiết bằng provider order_id, gồm đơn đã hủy và hết hạn. Tài liệu API và SDK không thống nhất định danh query detail: API ghi provider order_id, SDK ghi invoice. Chọn API order_id từ kết quả danh sách; không đoán lookup-by-invoice. Phân trang và chống trùng không thay thế sao kê ngân hàng cho khoản provider không trả về.

Đọc số tiền thập phân bằng chuỗi, chỉ chấp nhận số nguyên VND hoặc phần lẻ toàn số 0. Không dùng float cho ledger. Thông báo thẻ/void/không hiểu vào review, không tự chốt kho hay sửa PAID. Timeout query không tạo attempt mới và không giải phóng nghĩa vụ.

## Trade-off

Form trên trang SePay giảm UI phải xây nhưng cần internet/sandbox merchant. Một attempt không ngăn khách chuyển hai lần. Chưa hứa ngân hàng chặn QR hết hạn. Tài liệu đang có sai khác phải được kiểm merchant sandbox trước mở production.

## Nguồn

- [SePay IPN](https://developer.sepay.vn/vi/cong-thanh-toan/IPN)
- [Sandbox](https://developer.sepay.vn/vi/cong-thanh-toan/sandbox)
- [SDK và thứ tự form](https://developer.sepay.vn/vi/cong-thanh-toan/sdk/nodejs)
- [Order detail](https://developer.sepay.vn/vi/cong-thanh-toan/API/don-hang/chi-tiet-don-hang)
- [SDK source](https://github.com/sepayvn/sepay-pg-node)
