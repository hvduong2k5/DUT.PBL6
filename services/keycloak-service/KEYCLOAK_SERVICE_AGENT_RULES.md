# Agent Rules — Keycloak Service

Quy tắc áp dụng cho mọi thay đổi trong `services/keycloak-service`. AI agent và lập trình viên phải tuân thủ.

## 1. Cấu trúc
```
services/keycloak-service/
├── Dockerfile                  # Image Keycloak 24 tối ưu (Quarkus build, nạp theme & realm template)
├── Dockerfile.coolify          # Dockerfile triển khai Git-based trên Coolify
├── .dockerignore               # Loại bỏ file rác khi build container
├── .env.example                # Biến môi trường mẫu (không chứa secret thật)
├── README.md                   # Hướng dẫn chạy container ở dev và production (Coolify)
├── realm-export.template.json  # Realm JSON import lần đầu (token 15 phút, tiếng Việt, Super Admin)
├── order-user-profile-attribute.json
└── themes/omama/login/         # Theme đăng nhập / quên mật khẩu (Tri Kỷ - Tiếng Việt)
```
- Không đặt cấu hình Keycloak trong `services/identity-service/deploy/`. Identity Service chỉ là client (Admin API + OIDC).
- Database `keycloak_db` thuộc Keycloak; không dùng chung với `identity_db`.
- Không sử dụng Docker Compose trong `services/keycloak-service`; triển khai qua Dockerfile kết nối trực tiếp tài nguyên PostgreSQL độc lập.

## 2. Realm template & bảo mật token
- `realm-export.template.json` **không được chứa secret thật** (client secret sản xuất).
- `accessTokenLifespan` = `900` (15 phút) nhằm tối ưu hiệu năng và giảm tải token refresh (giảm 67% số lần refresh so với 5 phút), trong khi việc thu hồi quyền vẫn được bảo vệ an toàn bằng cơ chế session revocation / Force Re-login tức thì.
- Bắt buộc bật Refresh Token Rotation: `revokeRefreshToken = true`, `refreshTokenMaxReuse = 0`.
- Ngôn ngữ: Chỉ hỗ trợ tiếng Việt (`defaultLocale: vi`, `supportedLocales: ["vi"]`, `internationalizationEnabled: false`).
- Tài khoản quản trị khởi tạo: User `admin` sở hữu vai trò `ADMIN` (realm role), `realm-admin` (client `realm-management`), và toàn bộ quyền `permission_grants` (`#GLOBAL`) cho các microservices.
- `--import-realm` chỉ import khi realm chưa tồn tại. Realm đang chạy phải cập nhật thủ công (Admin Console → Realm settings → Tokens, hoặc `kcadm.sh update realms/omama`).
- Client public (`frontend-web`) bắt buộc PKCE `S256`, redirect URI liệt kê tường minh.

## 3. Theme (FreeMarker)
- Chỉ hỗ trợ tiếng Việt; cấu hình `locales=vi`, `defaultLocale=vi` trong `theme.properties`.
- Không dùng built-in `?html` (Keycloak 24 đã bật auto-escaping, sẽ gây lỗi "An internal server error has occurred").
- Không hardcode domain frontend. Link "quay lại"/"tiếp tục với tư cách khách" suy ra từ `redirect_uri` + `state` của OAuth client (xem `omama-login.js`).
- Kiểm tra trước khi commit: `node --check` cho JS, các thẻ `<#if>`/`<#list>` cân bằng.

## 4. Deploy
- Port lắng nghe mặc định: `8080`.
- Chế độ Production Coolify: Container kết nối trực tiếp đến PostgreSQL resource qua biến môi trường `KC_DB_URL`, `KC_DB_USERNAME`, `KC_DB_PASSWORD`.
- Bắt buộc khai báo `KC_PROXY_HEADERS=xforwarded` và `KC_HTTP_ENABLED=true` khi chạy sau reverse proxy (Traefik/Coolify).

## 5. Commit Message
- 100% tiếng Anh, theo `AGENTS.md`. Scope gợi ý: `keycloak`.
