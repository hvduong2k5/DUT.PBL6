# Xác thực và phân quyền nhân viên Traceability

**Ngày lập:** 13/11/2026 (UTC+7) · **Mốc:** `dev@239c8db` · **Trạng thái:** Chốt thiết kế ngày 13/11/2026

Identity đã có staff token với `user_id` và `permission_grants`. Traceability hiện xác thực JWT sandbox. Adapter dùng Identity/Keycloak và giữ các permission đã có trong domain Traceability.

| Thao tác | Permission |
| --- | --- |
| Xem hồ sơ nội bộ | `traceability.read_internal` |
| Xem tham chiếu bằng chứng | `traceability.read_evidence`, cùng quyền đọc hồ sơ |
| Tạo hoặc sửa hồ sơ | `traceability.create` hoặc `traceability.edit`, cùng quyền đọc nội bộ |
| Gửi duyệt | `traceability.submit` |
| Duyệt hoặc từ chối | `traceability.approve` |
| Publish | `traceability.publish` |
| Đình chỉ hoặc lưu trữ | `traceability.suspend` hoặc `traceability.archive` |

Traceability xác minh chữ ký RS256, issuer, audience và hạn token qua JWKS của Keycloak. Dùng audience `omama-api`, issuer realm `omama` theo môi trường và lấy `user_id` đã xác thực làm actor UUID. Mỗi yêu cầu admin đối chiếu `user_id`, `sub`, loại tài khoản nhân viên và trạng thái hoạt động qua `GET /api/v1/users/me`. Chỉ chấp nhận quyền GLOBAL có trong cả token lẫn quyền hiện hành do Identity trả về. Identity gián đoạn thì trả `503`. API public tra cứu QR cho phép truy cập ẩn danh.

P8 chỉ nhận grant `GLOBAL`. Người soạn có quyền đọc, tạo, sửa và gửi duyệt. Người duyệt có quyền đọc, duyệt và publish. Quyền xem bằng chứng, đình chỉ và lưu trữ được cấp riêng theo nhiệm vụ. Scope kho/phòng ban sẽ cần binding tương ứng trước khi hỗ trợ. Giữ kiểm tra người đóng góp không tự duyệt, kể cả khi có cả hai quyền. Ghi actor đã xác thực trong audit.

## Nghiệm thu

- Hai tài khoản thật chạy tạo, gửi duyệt, duyệt và publish qua Kong. Audit khớp `user_id` của từng người.
- Token thiếu, hết hạn, sai issuer/audience hoặc sai chữ ký trả `401`. Token hợp lệ thiếu permission hoặc ngoài scope trả `403`.
- Header actor/quyền giả không đổi kết quả. Người soạn có quyền duyệt vẫn bị chặn tự duyệt.
- Kiểm tra tài khoản bị vô hiệu hóa, quyền bị thu hồi, đổi signing key và người có quyền đọc nhưng thiếu quyền xem bằng chứng.

Nguồn đối chiếu [schema staff token](../../../../packages/contracts/iam/staff-token.v1.schema.json) và [permission Traceability](../../../../services/traceability-service/internal/domain/principal.go).
