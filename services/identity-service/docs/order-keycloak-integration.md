# Identity → Order: principal trong access token

Nhánh companion Order: `codex/order-keycloak-auth`. Hai PR cần ghép vào từng nhánh
service trước khi thử chung; chỉ merge contract #34 không triển khai các thay đổi này.

## Code được cập nhật

- Registration truyền `principalId` do Identity sinh vào `KeycloakAdminClient`.
- Keycloak user được ghi attribute `app_user_id` bằng principal đó, không phải subject
  của Keycloak và không phải input người đăng ký.
- Sau tạo user, gán realm role `CUSTOMER`; lỗi gán role phải báo provisioning thất bại,
  không trả kết quả thành công. Nếu bước này lỗi sau khi user đã tạo, tài khoản có thể
  còn ở Keycloak: cần reconciliation hiện trạng trước khi retry, không tạo principal
  mới hay xóa account tự động. Thay đổi này chưa sửa toàn bộ saga đăng ký hiện có.
- Realm template có thêm client scope `order-customer`: thêm audience `order-api`
  và claim `user_id`. Audience `omama-api` cũ và các client khác được giữ nguyên.

## Realm đã chạy cần cập nhật cấu hình

File realm template chỉ tác dụng khi import; không tự cập nhật realm đang tồn tại.
Trên realm test, tạo scope `order-customer` với hai mapper trong template và thêm nó
vào Default client scopes của `frontend-web` (và Mobile client nếu có). Giữ scope
`roles` để role nằm ở `realm_access.roles`. Không gắn scope này vào service account
admin để giả lập người mua.

**Bắt buộc trước khi tin claim:** cấu hình User Profile cho `app_user_id` chỉ admin
được xem/sửa. File [attribute mẫu](../deploy/keycloak/order-user-profile-attribute.json)
là một phần tử để thêm vào danh sách `attributes` của User Profile hiện tại; không
ghi đè toàn bộ cấu hình bằng file này. Admin Console → Realm → User profile, hoặc
GET/merge/PUT API cấu hình profile qua quyền admin. Kiểm chứng người mua không thể
đổi attribute qua Account Console/API trước khi nghiệm thu. Không cấp quyền sửa
unmanaged attributes rộng cho khách.

Tài khoản cũ thiếu attribute cần backfill từ `identity_principals.principal_id` và
`idp_subject` đã có, kiểm khớp account và cấp role đúng. Không suy từ email và không
sinh ID mới. Khi mapper/role đổi, đăng nhập lại để lấy access token mới.

Order kiểm RS256/JWKS, issuer chính xác, audience `order-api`, typ `Bearer`, thời gian,
realm role CUSTOMER và UUID `user_id`. Role nhân viên/admin không được chuyển thành
quyền staff Order trong lần ghép này. Khách không có mapper/role sẽ nhận 401, không
có fallback sang `sub` hoặc token sandbox.

## Kiểm thử

Java 21, Gradle theo phiên bản của repo:

```sh
gradle test --tests '*KeycloakAdminClientTest'
```

Test kiểm đúng principal được ghi vào attribute, gán đúng role, từ chối principal
thiếu/rỗng và không báo thành công nếu role assignment lỗi. Đây là test client với
Keycloak Admin API giả lập, chưa phải nghiệm thu realm thật.

Kết quả local 29/10/2026: Java 21.0.9 + Gradle 9.2.0 compile main/test và chạy
`test --tests '*KeycloakAdminClientTest'` PASS, 3 test. Chưa chạy DB integration
suite hoặc đăng nhập trên Keycloak thật; chưa thay cấu hình realm triển khai.

Sau khi deploy cả hai nhánh, dùng hướng dẫn `services/order-service/IDENTITY_TEST.md`
ở nhánh Order: token khách A/B gọi cart, token bị sửa bị chặn, không cấp guest session;
có đơn của A thì kiểm B không đọc được. Registration → outbox → Profile cần chạy thật
để xác nhận ánh xạ customer cuối cùng, không lấy test client thay bằng chứng đó.
