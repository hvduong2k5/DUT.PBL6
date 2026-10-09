# ADR-004 — Identity subject, ownership và verified guest claims

- Ngày: 08/10/2026
- Trạng thái: Implemented in Profile Service; ingress và OTP/Order adapter là dependency tích hợp.

## Bối cảnh

Fallback query customer ID cho phép caller tự chọn identity. Address/claim GET không giới hạn owner. User ID bị dùng làm profile PK. Guest claim được ghi VERIFIED mà chưa có bằng chứng.

## Quyết định

Resolve `user_id → customer_profiles.id`. HTTP chỉ nhận forwarded identity từ caller nội bộ có token cấu hình; gRPC dùng cùng authentication metadata. Không có query fallback. Ownership được enforce khi đọc/ghi và sai owner trả 404.

Claim usecase bắt buộc verifier kiểm tra proof và đơn guest. Không có verifier thì fail closed với 503, không tạo claim hoặc event. Chưa triển khai gửi SMS/OTP hoặc gọi Order API vì repo chưa có provider/contract chạy thực cho hai dependency này.

## Hệ quả

Caller server cần gửi token và identity; không expose token ở client. Production cần network/TLS protection và quản lý/rotation secret. Deployment K8s tham chiếu Secret, không chứa token. Flow guest claim tạm unavailable thay vì cho phép nhận đơn chưa xác minh. Giữ interface để triển khai adapter sau, không giả OTP thành công.
