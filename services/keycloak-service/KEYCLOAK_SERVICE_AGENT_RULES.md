# Agent Rules — Keycloak Service

Quy tắc áp dụng cho mọi thay đổi trong `services/keycloak-service`. AI agent và lập trình viên phải tuân thủ.

## 1. Cấu trúc
```
services/keycloak-service/
├── docker-compose.yml          # Keycloak + PostgreSQL riêng của Keycloak
├── .env.example                # Biến môi trường mẫu (không chứa secret thật)
├── README.md                   # Hướng dẫn deploy Keycloak đơn lẻ
├── realm-export.template.json  # Realm JSON import lần đầu
├── order-user-profile-attribute.json
└── themes/omama/login/         # Theme đăng nhập / quên mật khẩu (Tri Kỷ)
```
- Không đặt cấu hình Keycloak trong `services/identity-service/deploy/`. Identity Service chỉ là client (Admin API + OIDC).
- Database `keycloak_db` thuộc Keycloak; không dùng chung với `identity_db`.

## 2. Realm template & bảo mật token
- `realm-export.template.json` **không được chứa secret thật** (client secret, mật khẩu).
- `accessTokenLifespan` = `300` (5 phút). Chỉ được tăng tối đa `900` khi có quyết định rõ ràng.
- Bắt buộc bật Refresh Token Rotation: `revokeRefreshToken = true`, `refreshTokenMaxReuse = 0`.
- `--import-realm` chỉ import khi realm chưa tồn tại. Realm đang chạy phải cập nhật thủ công (Admin Console → Realm settings → Tokens, hoặc `kcadm.sh update realms/omama`).
- Client public (`frontend-web`) bắt buộc PKCE `S256`, redirect URI liệt kê tường minh.

## 3. Theme (FreeMarker)
- Không dùng built-in `?html` (Keycloak 24 đã bật auto-escaping, sẽ gây lỗi "An internal server error has occurred").
- Không hardcode domain frontend. Link "quay lại"/"tiếp tục với tư cách khách" suy ra từ `redirect_uri` + `state` của OAuth client (xem `omama-login.js`).
- Mọi message key phải có đủ ở `messages_vi.properties` và `messages_en.properties`.
- Kiểm tra trước khi commit: `node --check` cho JS, các thẻ `<#if>`/`<#list>` cân bằng.

## 4. Deploy
- Container name `omafood-keycloak-standalone`, port `8080`, network `coolify` được Identity Service tham chiếu (`KEYCLOAK_SERVER_URL`). Không đổi nếu chưa cập nhật Identity Service.
- Kiểm tra cú pháp compose: `docker compose -f services/keycloak-service/docker-compose.yml config`.

## 5. Commit Message
- 100% tiếng Anh, theo `AGENTS.md`. Scope gợi ý: `keycloak`.
