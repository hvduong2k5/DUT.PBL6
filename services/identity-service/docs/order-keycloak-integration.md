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
- Realm template có thêm client scope `order-customer`: audience `order-api`, claim
  `user_id`, mapper `realm_access.roles` và role scope mapping CUSTOMER. Audience
  `omama-api` cũ và các client khác được giữ nguyên.

## Realm đã chạy cần cập nhật cấu hình

File realm template chỉ tác dụng khi import; không tự cập nhật realm đang tồn tại.
Trên realm test, tạo scope `order-customer` với ba mapper trong template và thêm nó
vào Default client scopes của `frontend-web` (và Mobile client nếu có). Giữ scope
`roles` nếu client cũ cần; scope mới tự map `realm_access.roles`. Thêm role scope
mapping CUSTOMER theo `scopeMappings` trong template. Không gắn scope này vào service account
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

Kết quả unit local 29/10/2026: Java 21.0.9 + Gradle 9.2.0 compile main/test và chạy
`test --tests '*KeycloakAdminClientTest'` PASS, 3 test. Chưa chạy DB integration
suite trong lần unit này; chưa thay cấu hình realm triển khai.

Đã bổ sung `KeycloakLiveProvisioningTest` dùng Keycloak Docker thật, do script
`services/order-service/scripts/test-keycloak-live.py` ở nhánh Order điều phối.
Script tạo realm riêng, chạy Java adapter tạo account, lấy token qua PKCE rồi gọi
HTTP Order/Redis thật. Test fixture bật bằng `KEYCLOAK_TEST_URL`, không cache kết quả
Gradle khi bật để tránh báo xanh mà không tạo account cho realm mới.

Bài live phát hiện và sửa lỗi template: chỉ tham chiếu tên scope `roles` không đảm
bảo nó tồn tại khi import realm có danh sách clientScopes tùy chỉnh. Token đã có
audience/user_id nhưng thiếu realm_access.roles nên Order từ chối đúng. Mapper role
được thêm rõ vào scope order-customer, không bỏ kiểm role phía Order.

**Kết quả live 30/10/2026: PASS.** Client Java tạo hai account trên Keycloak 24.0.5,
principal và role được lưu đúng, đăng nhập PKCE cấp token được Order chấp nhận,
HTTP Order đọc đúng giỏ A/B từ Redis thật. Account API cho cập nhật hồ sơ thường
nhưng không đổi được `app_user_id`; đã kiểm lại bằng Admin API. Account API fixture
có audience/role mapper riêng để kiểm đúng quyền (không lấy lỗi 401 làm bằng chứng).
Đây chưa phải kiểm thử REST registration, saga/DB Identity hay consumer Profile.

Sau khi deploy cả hai nhánh, dùng hướng dẫn `services/order-service/IDENTITY_TEST.md`
ở nhánh Order: token khách A/B gọi cart, token bị sửa bị chặn, không cấp guest session;
có đơn của A thì kiểm B không đọc được. Registration → outbox → Profile cần chạy thật
để xác nhận ánh xạ customer cuối cùng, không lấy test client thay bằng chứng đó.
