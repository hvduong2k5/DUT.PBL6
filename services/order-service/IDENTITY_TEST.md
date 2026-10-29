# Ghép Order với Identity/Keycloak — lát cắt khách đăng nhập

## Thay đổi này làm được gì?

Order có chế độ `ORDER_AUTH_MODE=keycloak`: kiểm access token RS256 từ Keycloak,
issuer, audience, hạn dùng, thời điểm bắt đầu dùng/được cấp và role `CUSTOMER`.
Token thiếu `user_id` bị từ chối. Không dùng `sub` làm principal hoặc customer ID.

Luồng ID: **Keycloak sub → claim user_id (principal của Identity) → Profile trả customer ID**.
Giỏ được phân vùng theo principal; địa chỉ/checkout tiếp tục dùng bước Resolve của
Profile và kiểm `response.user_id` khớp principal. Trong chế độ này không fallback
sang Profile simulator nếu thiếu cấu hình Profile.

Chế độ sandbox cũ vẫn dùng khi `ORDER_AUTH_MODE` không đặt hoặc bằng `sandbox`, để
giữ các test cũ. Khi chọn `keycloak`, HS256 sandbox bị từ chối; API cấp guest session
bị đóng. Staff/payment admin chưa được ánh xạ IAM trong lát cắt này.
`APP_MODE=sandbox` vẫn bắt buộc vì các dependency nghiệp vụ khác chưa nghiệm thu;
auth thật không biến cả hệ thống thành production-ready.

## 1. Chuẩn bị phía Identity

Áp dụng nhánh companion `codex/identity-order-claims` lên nhánh Identity. Nhánh đó:

- Truyền principal UUID do server tạo vào Keycloak attribute `app_user_id` khi đăng ký.
- Gán realm role `CUSTOMER`, chỉ báo provisioning thành công khi bước này thành công.
- Thêm client scope `order-customer`: audience `order-api`, claim `user_id` lấy từ
  `app_user_id`; giữ nguyên audience `omama-api` hiện tại.

Realm đã tồn tại **không tự thay đổi** khi sửa file template. Nhóm Identity phải áp
dụng các mapper vào realm hiện tại và gắn scope vào client Web/Mobile dùng để test.
Giữ scope `roles` chuẩn để token có `realm_access.roles`.

**Bắt buộc bảo vệ `app_user_id`:** cấu hình User Profile để chỉ admin/service account
được sửa attribute này, người dùng không được sửa qua Account Console hoặc API.
Giá trị phải trùng principal đã lưu trong DB Identity. Không dùng email làm khóa ánh
xạ, không cấp ID tùy ý từ FE. Tài khoản cũ cần backfill từ mapping Identity có thật,
không sinh một UUID mới khác principal. Xem hướng dẫn ở nhánh Identity.

Token access dùng cho Order phải có:

```json
{
  "iss": "https://KEYCLOAK_HOST/realms/omama",
  "aud": ["omama-api", "order-api"],
  "sub": "keycloak-subject",
  "user_id": "UUID-PRINCIPAL-NOI-BO",
  "typ": "Bearer",
  "realm_access": {"roles": ["CUSTOMER"]},
  "iat": 0,
  "exp": 0
}
```

Đây là ví dụ hình dạng; `iat`/`exp` phải là thời gian thật, không dùng số 0 để test.
Lấy access token qua đăng nhập Authorization Code + PKCE hiện có. Không bật password
grant hoặc tạo endpoint login mới chỉ để vượt bước này. Không commit token.

## 2. Cấu hình Order

Giữ cấu hình DB/Redis/dependency hiện có và bổ sung:

```dotenv
APP_MODE=sandbox
ORDER_AUTH_MODE=keycloak
ORDER_OIDC_ISSUER=https://KEYCLOAK_HOST/realms/omama
ORDER_OIDC_AUDIENCE=order-api
PROFILE_URL=http://PROFILE_INTERNAL_HOST:PORT
PROFILE_INTERNAL_TOKEN=<credential riêng của Order gọi Profile>
```

Issuer phải đúng chính xác `iss` Keycloak đang cấp, không lấy hostname ví dụ trong
tài liệu làm cấu hình triển khai. JWKS mặc định ở `{issuer}/protocol/openid-connect/certs`.
Có thể đặt `ORDER_OIDC_JWKS_URL` nếu mạng nội bộ cần URL khác; URL này là cấu hình
server tin cậy, không đọc từ header token. HTTPS bắt buộc; chỉ khi chạy sandbox trên
loopback mới được dùng HTTP với `ORDER_OIDC_ALLOW_LOCAL_HTTP=true`.

Đã có [Compose override](docker-compose.identity.yml). Từ root, sau khi đặt các biến
môi trường trên, chạy:

```powershell
docker compose -f services/order-service/docker-compose.test.yml -f services/order-service/docker-compose.identity.yml up -d --build api
```

Override chỉ đổi auth và Profile của API; các provider khác vẫn theo stack test.
Nó không tự triển khai Keycloak/Identity/Profile. Hostname trong biến phải truy cập
được từ container. Không dùng `localhost` của máy host để gọi service từ container.

JWKS cache tối đa 5 phút, thử refresh cho key mới tối đa một lần mỗi 15 giây,
HTTP timeout 2 giây, không theo redirect. Hết cache mà JWKS lỗi thì từ chối token;
key còn trong cache hợp lệ vẫn dùng đến hạn. Với khóa bị thu hồi khẩn cấp cần restart
Order để xóa cache. JWT offline không kiểm trạng thái tài khoản trong DB Identity
trên từng request: khóa tài khoản/chỉnh role có độ trễ đến lúc token hết hạn; cần
cơ chế thu hồi/introspection riêng nếu yêu cầu hiệu lực tức thì.

`ORDER_JWT_SECRET` vẫn phải cấu hình theo yêu cầu hiện có vì còn ký cursor; nó không
được dùng để xác thực access token trong chế độ Keycloak.

## 3. Kiểm thử local tự động

Trong thư mục `services/order-service`:

```sh
go test ./internal/app ./internal/transport
go vet ./...
go test ./...
```

Test mới dùng RSA ký thật, HTTP JWKS trên loopback và Redis protocol test server:

- Đúng issuer/audience và role, ánh xạ principal khác Keycloak subject.
- Hai token đọc đúng hai giỏ riêng qua HTTP Order, bỏ qua header mạo danh.
- Ownership đơn A từ chối principal B qua hàm kiểm quyền hiện có.
- Sai chữ ký, audience, issuer; token hết hạn/chưa có hiệu lực; thiếu claim;
  ID token; role nhân viên; token sandbox đều bị từ chối.
- Cache/đổi khóa/khóa bị gỡ/JWKS lỗi; Profile trả sai chủ hoặc thiếu cấu hình.
- Guest session không được cấp trong chế độ Keycloak.

Đây là kiểm thử Order với dependency thử nghiệm, **chưa chứng minh realm triển khai
thật đã có đúng mapper hoặc toàn bộ luồng đăng ký → Profile đã hoàn chỉnh**.
Các test PostgreSQL/Kafka có điều kiện sẽ skip khi chưa đặt biến môi trường tương ứng.

Kết quả local ngày 29/10/2026: `go test -count=1 ./...` và `go vet ./...` PASS
với Go 1.27.3 trên Windows. HTTP cart A/B và các test Keycloak mới thực sự chạy.
Không đặt biến DB/Kafka cho integration suite; Docker daemon không chạy nên chưa
có kết quả stack thật hoặc smoke test bằng token của realm triển khai.

## 4. Smoke test với token thật

Chuẩn bị hai khách A/B đã đăng ký, có Profile thật. Đặt `ORDER_TEST_TOKEN_A` và
`ORDER_TEST_TOKEN_B` trong môi trường phiên shell riêng, không ghi vào file Git.
Sau đó chạy từ root:

```powershell
python services/order-service/scripts/test-identity.py
```

Script mặc định gọi `http://127.0.0.1:18004`; đổi bằng `ORDER_TEST_URL`. Nó kiểm hai
token gọi cart được, thiếu/sửa token bị từ chối và guest đóng. Nếu có đơn thật của A,
đặt thêm `ORDER_TEST_ORDER_A` để kiểm A đọc được và B bị chặn. Script chỉ đọc, không
in token/nội dung giỏ/đơn. Nếu chưa có đơn thì bước ownership đơn được ghi SKIP;
không coi hai giỏ rỗng giống nhau là bằng chứng ownership.

Chỉ ghi nghiệm thu Identity thật sau khi smoke test qua và kiểm `app_user_id` không
thể bị khách tự sửa. Checkout vẫn cần Catalog/Shipping/Inventory và các adapter khác.
