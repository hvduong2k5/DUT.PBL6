# Keycloak Service — Triển khai Keycloak độc lập

Keycloak 24 + PostgreSQL riêng (`keycloak_db`), kèm theme đăng nhập Ô Mạ (`themes/omama`) và realm template `omama`. Identity Service chỉ là client của Keycloak (OIDC + Admin API).

```
services/keycloak-service/
├── docker-compose.dev.yml         # Compose chạy môi trường Local Development (start-dev, zero-config)
├── docker-compose.yml             # Compose triển khai Production / Staging (Coolify, HTTPS, proxy headers)
├── .env.example                   # Biến môi trường mẫu cho Production
├── KEYCLOAK_SERVICE_AGENT_RULES.md# Quy chuẩn phát triển Keycloak Service
├── realm-export.template.json     # Realm template omama (Lean JWT, 5-phút token lifespan, token rotation)
├── order-user-profile-attribute.json
└── themes/omama/login/            # Giao diện đăng nhập Ô Mạ
```

---

## 1. Môi Trường Local Development (Dev Environment)

File `docker-compose.dev.yml` được thiết kế riêng cho lập trình viên phát triển cục bộ:
- Chạy Keycloak ở chế độ `start-dev --import-realm` (tự động nạp realm `omama`, không đòi hỏi HTTPS hay proxy headers).
- **Không yêu cầu** tạo mạng external `coolify`.
- Dùng PostgreSQL riêng trên port **`5434:5432`** (tránh xung đột với port `5432` mặc định hoặc `5433` của `identity-service`).
- Keycloak lắng nghe trên port **`8080:8080`**.
- Mật khẩu mặc định sẵn sàng: `admin` / `admin` (Keycloak) và `keycloak` / `password123` (Postgres).

### Khởi động cụm Dev:
```powershell
cd services/keycloak-service
docker compose -f docker-compose.dev.yml up -d
```

### Kiểm tra trạng thái:
```powershell
docker compose -f docker-compose.dev.yml ps
docker compose -f docker-compose.dev.yml logs -f keycloak
```
Chờ đến khi thấy thông báo: `Keycloak 24.0.5 on JVM ... started in ...`.

### Truy cập Admin Console:
- URL: [http://localhost:8080/admin](http://localhost:8080/admin)
- Tài khoản: `admin`
- Mật khẩu: `admin`
- Realm quản lý nghiệp vụ: **`omama`** (đã được tự động import từ `realm-export.template.json`).

### Lấy Client Secret cho `identity-service`:
1. Đăng nhập [http://localhost:8080/admin](http://localhost:8080/admin) -> chọn Realm **`omama`** ở góc trên bên trái.
2. Menu bên trái chọn **Clients** -> click vào **`identity-service-admin`**.
3. Chọn tab **Credentials** -> copy giá trị **Client Secret**.
4. Dán secret vừa copy vào file `.env` của `identity-service` (`KEYCLOAK_CLIENT_SECRET=<secret_vua_copy>`).
5. Kiểm tra tab **Service account roles** -> đảm bảo đã có role `realm-management` (`manage-users`, `view-users`).

### Dừng môi trường Dev:
```powershell
# Dừng container, giữ nguyên dữ liệu CSDL:
docker compose -f docker-compose.dev.yml down

# Dừng container và xóa trắng volume để import lại template mới nhất:
docker compose -f docker-compose.dev.yml down -v
```

---

## 2. Triển Khai Production / Coolify

Dùng cho môi trường Production hoặc Staging kết nối mạng `coolify` phía sau Traefik reverse proxy:

```powershell
cd services/keycloak-service
copy .env.example .env          # Cấu hình mật khẩu mạnh và domain KC_HOSTNAME
docker network create coolify   # Tạo mạng nếu máy chủ chưa có
docker compose up -d
```

### Các bước cấu hình trên Coolify:
1. **+ New Resource → Docker Compose**, trỏ tới `services/keycloak-service/docker-compose.yml` (Base Directory: `services/keycloak-service`).
2. Domain: `https://keycloak.omama.vn`, port nội bộ `8080`.
3. Khai báo các biến môi trường theo `.env.example` (`KEYCLOAK_ADMIN`, `KEYCLOAK_ADMIN_PASSWORD`, `KEYCLOAK_DB_PASSWORD`, `KC_HOSTNAME=https://keycloak.omama.vn`, ...).
4. Deploy; kiểm tra log cho đến khi `Keycloak 24.0.5 ... started`.

> [!IMPORTANT]
> `KC_PROXY_HEADERS=xforwarded`, `KC_HTTP_ENABLED=true`, `KC_HOSTNAME=https://keycloak.omama.vn` là bắt buộc phía sau Traefik.

---

## 3. Cấu Hình Bảo Mật Token & Realm Template

Trong `realm-export.template.json`, cấu hình token tuân thủ nghiêm ngặt chuẩn kiến trúc IAM hiện đại:
- **Thời lượng Access Token chuẩn:** `accessTokenLifespan = 900` (15 phút). Chu kỳ này tối ưu tải cho hệ thống (giảm tần suất refresh token từ 12 lần/giờ xuống 4 lần/giờ/người dùng), trong khi việc thu hồi đặc quyền khi nhân viên bị hạ quyền/thu hồi vai trò vẫn được đảm bảo an toàn tuyệt đối nhờ cơ chế Force Re-login / Revoke session tức thì của Identity Service.
- **Refresh Token Rotation:** `revokeRefreshToken = true`, `refreshTokenMaxReuse = 0` (mỗi refresh token chỉ được dùng 1 lần duy nhất, ngăn chặn token replay attack).
- **Cấu trúc Lean JWT (Thu gọn tối đa):**
  - Đã loại bỏ các claim PII cồng kềnh khỏi Access Token (`email`, `name`, `given_name`, `family_name`, `preferred_username`).
  - Quyền hạn nghiệp vụ trả về chuẩn mảng `permission_grants` (ví dụ `["ORDER_VIEW#GLOBAL", "ORDER_SENSITIVE_VIEW#WAREHOUSE#WH-HCM-01"]`).
  - Phân loại chủ thể qua `principal_type` (`EMPLOYEE`, `CUSTOMER`) và định danh `user_id`.

> [!NOTE]
> `command: start --import-realm` (hoặc `start-dev --import-realm`) chỉ import template **khi realm `omama` chưa tồn tại trong CSDL**.  
> Nếu realm đã tồn tại, thay đổi cấu hình token thủ công qua Admin Console hoặc CLI:
> ```bash
> kcadm.sh config credentials --server http://localhost:8080 --realm master --user admin
> kcadm.sh update realms/omama -s accessTokenLifespan=900 -s revokeRefreshToken=true -s refreshTokenMaxReuse=0
> ```

---

## 4. Theme Đăng Nhập Ô Mạ

- Thư mục theme: `themes/omama/login/` (đã được cấu hình mặc định `loginTheme: "omama"` trong realm).
- Link "Quay lại" / "Tiếp tục với tư cách khách" được tính toán linh hoạt dựa trên `redirect_uri` + `state` của OAuth client, không hardcode domain.
- **Lưu ý cú pháp:** Không sử dụng `?html` trong các file template `.ftl` (Keycloak 24 đã tự động escape chống XSS; việc dùng thêm `?html` sẽ gây lỗi server 500).

---

## 5. Bảng Khắc Phục Sự Cố (Troubleshooting)

| Triệu chứng | Nguyên nhân | Cách xử lý |
|---|---|---|
| `ERR_TOO_MANY_REDIRECTS` | Thiếu proxy headers sau reverse proxy Traefik | Kiểm tra `KC_PROXY_HEADERS=xforwarded`, `KC_HTTP_ENABLED=true`, `KC_HOSTNAME=https://...` |
| Keycloak không kết nối DB | Volume DB cũ lưu mật khẩu khác với `.env` | Chạy `docker compose down -v` để xóa volume dev hoặc cập nhật lại mật khẩu trong `.env` |
| `Invalid parameter: redirect_uri` | Client chưa khai báo redirect URI hợp lệ | Vào Admin Console -> Realm `omama` -> Clients -> Thêm URI vào **Valid redirect URIs** |
| `An internal server error has occurred` ở trang login | Lỗi cú pháp FreeMarker trong theme `.ftl` | Kiểm tra log container: `docker compose logs -f keycloak`, loại bỏ các filter không tương thích như `?html` |
| Token cũ vẫn dùng được sau khi đổi quyền | Realm đang chạy chưa áp thời lượng 15 phút / rotation | Cập nhật lại `accessTokenLifespan=900` và bật *Revoke Refresh Token* trong *Realm settings -> Tokens* |

### Identity role-list service credential

On an existing realm, add the `identity-roles-api` audience mapper directly
to `identity-service-admin` and disable standard, implicit and direct access
flows for that client. For its service account, keep `realm-management`
`manage-users` and `view-users` and remove the direct `manage-realm` mapping.
The dedicated audience authorizes only `GET /api/v1/admin/roles` in Identity.
After applying the realm change, validate role listing and the Identity
provisioning flow with fresh service tokens.
