# Keycloak Service — Triển khai Keycloak độc lập

Keycloak 24 + PostgreSQL riêng (`keycloak_db`), kèm theme đăng nhập Ô Mạ (`themes/omama`) và realm template `omama`. Identity Service chỉ là client của Keycloak (OIDC + Admin API).

```
services/keycloak-service/
├── docker-compose.yml
├── .env.example
├── KEYCLOAK_SERVICE_AGENT_RULES.md
├── realm-export.template.json
├── order-user-profile-attribute.json
└── themes/omama/login/
```

## 1. Chạy bằng Docker Compose

```powershell
cd services/keycloak-service
copy .env.example .env      # sửa mật khẩu/KC_HOSTNAME
docker network create coolify   # chỉ lần đầu nếu máy chưa có network này
docker compose up -d
```
- Keycloak: `http://localhost:8080` (`KEYCLOAK_PORT`), Admin Console: `/admin`.
- Container `omafood-keycloak-standalone` nằm trong network `coolify` để Identity Service gọi nội bộ qua `http://omafood-keycloak-standalone:8080`.

## 2. Deploy trên Coolify
1. **+ New Resource → Docker Compose**, trỏ tới `services/keycloak-service/docker-compose.yml` (Base Directory = `services/keycloak-service`).
2. Domain: `https://keycloak.omama.vn`, port nội bộ `8080`.
3. Khai báo biến môi trường theo `.env.example` (`KEYCLOAK_ADMIN`, `KEYCLOAK_ADMIN_PASSWORD`, `KEYCLOAK_DB_PASSWORD`, `KC_HOSTNAME`, ...).
4. Deploy; chờ log `Keycloak 24.0.5 ... started`.

`KC_PROXY_HEADERS=xforwarded`, `KC_HTTP_ENABLED=true`, `KC_HOSTNAME=https://...` là bắt buộc phía sau Traefik.

## 3. Realm template
- `command: start --import-realm` import `realm-export.template.json` **chỉ khi realm `omama` chưa tồn tại**.
- Cấu hình token bảo mật trong template:
  - `accessTokenLifespan = 300` (5 phút)
  - `revokeRefreshToken = true`, `refreshTokenMaxReuse = 0` (Refresh Token Rotation)
- **Realm đang chạy không tự cập nhật.** Áp dụng thủ công: Admin Console → Realm `omama` → *Realm settings → Tokens* (Access Token Lifespan = 5 phút; *Revoke Refresh Token* = On), hoặc:
  ```bash
  kcadm.sh config credentials --server http://localhost:8080 --realm master --user admin
  kcadm.sh update realms/omama -s accessTokenLifespan=300 -s revokeRefreshToken=true -s refreshTokenMaxReuse=0
  ```
- Template không chứa secret thật; sau import phải lấy secret client (mục 4).

## 4. Cấu hình bắt buộc sau khi deploy
1. **Client Secret `identity-service-admin`:** Clients → `identity-service-admin` → Credentials → copy vào `KEYCLOAK_CLIENT_SECRET` của identity-service.
2. **Service account roles:** client `identity-service-admin` → Service account roles → Assign role → Filter by clients → `realm-management`: `manage-users`, `view-users`, `manage-realm`.
3. **Client `frontend-web`:** Valid redirect URIs (`https://api.omama.vn/api/v1/public/auth/callback`, `http://localhost:8016/api/v1/public/auth/callback`, và URI của mobile/web nếu dùng OIDC trực tiếp), Web origins theo domain frontend.

## 5. Theme
- Đặt tên theme: `omama` (đã gán `loginTheme` trong realm template).
- Link "quay lại"/"tiếp tục với tư cách khách" suy ra từ `redirect_uri` + `state` của OAuth client, không cấu hình domain frontend trong Keycloak.
- Không dùng `?html` trong `.ftl` (Keycloak 24 auto-escape).

## 6. Troubleshooting
| Triệu chứng | Nguyên nhân / Xử lý |
|---|---|
| `ERR_TOO_MANY_REDIRECTS` | Thiếu `KC_PROXY_HEADERS=xforwarded`, `KC_HTTP_ENABLED=true`, `KC_HOSTNAME=https://...` |
| Keycloak không kết nối DB | Compose dùng DB riêng `keycloak-postgres`; kiểm tra `KEYCLOAK_DB_PASSWORD` trùng khi volume đã khởi tạo bằng mật khẩu cũ |
| `Invalid parameter: redirect_uri` | Thêm URI vào Valid redirect URIs của client |
| `An internal server error has occurred` ở trang login | Lỗi FreeMarker trong theme (ví dụ `?html`); xem log container |
| Token cũ vẫn dùng được sau khi thu hồi quyền | Realm đang chạy chưa áp `accessTokenLifespan=300`/rotation (mục 3) |
