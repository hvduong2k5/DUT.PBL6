# Keycloak Service — Triển Khai Keycloak Độc Lập

Keycloak 24 (Quarkus) container hóa độc lập, tích hợp sẵn giao diện đăng nhập thuần tiếng Việt Ô Mạ (`themes/omama`), nạp tự động realm template `omama` với cấu hình bảo mật hiện đại và tài khoản khởi tạo Super Admin toàn quyền.

```
services/keycloak-service/
├── Dockerfile                  # Image Keycloak tối ưu (Quarkus pre-build, nạp theme & realm)
├── Dockerfile.coolify          # File build triển khai Git-based trên Coolify
├── .dockerignore               # Danh sách loại trừ khi build Docker image
├── .env.example                # Biến môi trường mẫu cho Production / Coolify
├── KEYCLOAK_SERVICE_AGENT_RULES.md # Quy chuẩn phát triển Keycloak Service
├── realm-export.template.json  # Realm template omama (token 15 phút, tiếng Việt, Super Admin)
├── order-user-profile-attribute.json
└── themes/omama/login/         # Giao diện đăng nhập Ô Mạ (tiếng Việt độc quyền)
```

---

## 1. Điểm Nổi Bật Về Kiến Trúc

- **Không dùng Docker Compose:** Dịch vụ triển khai trực tiếp bằng Dockerfile kết nối vào tài nguyên PostgreSQL độc lập có sẵn (trên Coolify hoặc máy chủ cơ sở dữ liệu dùng chung).
- **Quarkus Pre-build (Optimized Build):** Dockerfile sử dụng multi-stage build chạy `kc.sh build` ngay lúc build image, giúp container khởi động tức thì trong 2-3 giây ở môi trường production (`start --optimized`).
- **Giao diện thuần Tiếng Việt:** Chỉ hỗ trợ tiếng Việt (`locales=vi`, `defaultLocale=vi`, `internationalizationEnabled=false`), tạm ẩn hoàn toàn tiếng Anh và thanh chuyển đổi ngôn ngữ.
- **Tài khoản khởi tạo Super Admin:** Có sẵn tài khoản `admin` sở hữu vai trò `realm-admin` (Keycloak Admin Console) và toàn bộ quyền `permission_grants` (`#GLOBAL`) liên microservices.
- **Bảo mật Token chuẩn IAM:**
  - `accessTokenLifespan = 900` (15 phút): Giảm 67% số lần refresh token, tối ưu hóa hiệu năng toàn hệ thống.
  - Refresh Token Rotation: `revokeRefreshToken = true`, `refreshTokenMaxReuse = 0`.
  - Cấu trúc Lean JWT: Loại bỏ PII nhạy cảm khỏi Access Token, định danh bằng `user_id`, `principal_type`, và phân quyền qua mảng `permission_grants`.

---

## 2. Môi Trường Local Development (Dev)

### Bước 1: Build Docker image cục bộ
Đứng tại thư mục `services/keycloak-service`:
```powershell
docker build -t oma-keycloak:24.0.5 .
```

### Bước 2: Chuẩn bị cơ sở dữ liệu PostgreSQL
Nếu bạn chưa có sẵn PostgreSQL trên máy, có thể chạy nhanh một container PostgreSQL dev:
```powershell
docker run -d --name keycloak-postgres-dev `
  -p 5434:5432 `
  -e POSTGRES_DB=keycloak_db `
  -e POSTGRES_USER=keycloak `
  -e POSTGRES_PASSWORD=password123 `
  postgres:16-alpine
```

### Bước 3: Khởi động Keycloak container
Chạy Keycloak kết nối tới PostgreSQL vừa tạo:
```powershell
docker run -d --name omafood-keycloak-dev `
  -p 8080:8080 `
  -e KEYCLOAK_ADMIN=admin `
  -e KEYCLOAK_ADMIN_PASSWORD=admin `
  -e KC_DB=postgres `
  -e KC_DB_URL=jdbc:postgresql://host.docker.internal:5434/keycloak_db `
  -e KC_DB_USERNAME=keycloak `
  -e KC_DB_PASSWORD=password123 `
  -e KC_HOSTNAME=http://localhost:8080 `
  -e KC_HOSTNAME_STRICT=false `
  oma-keycloak:24.0.5 start-dev --import-realm
```

*(Trên Linux, thay `host.docker.internal` bằng IP bridge của Docker hoặc IP máy host).*

### Bước 4: Kiểm tra trạng thái
```powershell
docker logs -f omafood-keycloak-dev
```
Chờ đến khi xuất hiện thông báo: `Keycloak 24.0.5 on JVM (powered by Quarkus) started in ...`.

### Dừng và dọn dẹp dev container:
```powershell
docker stop omafood-keycloak-dev keycloak-postgres-dev
docker rm omafood-keycloak-dev keycloak-postgres-dev
```

---

## 3. Triển Khai Production Trên Coolify (Có Sẵn Resource PostgreSQL)

Trên Coolify, bạn đã có sẵn tài nguyên PostgreSQL chung (`postgres` resource). Ta sẽ triển khai Keycloak thành một Application độc lập kết nối đến resource database này.

### Bước 1: Tạo database `keycloak_db` trong PostgreSQL resource
Truy cập giao diện quản lý PostgreSQL trên Coolify hoặc qua DBeaver/psql kết nối vào database server:
```sql
CREATE DATABASE keycloak_db;
```

### Bước 2: Tạo Application trên Coolify
1. Trong Project trên Coolify, nhấn **+ New Resource** → chọn **Git Repository**.
2. Chọn repo dự án: `https://github.com/hvduong2k5/DUT.PBL6` (hoặc GitHub App tương ứng).
3. Cấu hình cơ bản (General):
   - **Build Pack:** `Dockerfile`
   - **Base Directory:** `services/keycloak-service`
   - **Dockerfile:** `Dockerfile` (hoặc `Dockerfile.coolify`)
   - **Port:** `8080`
   - **Domains:** Domain public của bạn, ví dụ `https://keycloak.omama.vn` hoặc `https://auth.omama.vn`.

### Bước 3: Khai báo Biến môi trường (Environment Variables)
Sao chép các biến sau vào mục **Environment Variables** của Application trên Coolify:

| Tên biến | Giá trị mẫu | Giải thích |
|---|---|---|
| `KEYCLOAK_ADMIN` | `admin` | Tài khoản Super Admin của Keycloak master realm |
| `KEYCLOAK_ADMIN_PASSWORD` | `<mat_khau_manh>` | Mật khẩu tài khoản Super Admin master realm |
| `KC_DB` | `postgres` | Loại hệ quản trị cơ sở dữ liệu |
| `KC_DB_URL` | `jdbc:postgresql://<ten_postgres_resource>:5432/keycloak_db` | URL kết nối database trên mạng nội bộ Coolify |
| `KC_DB_USERNAME` | `<user_postgres>` | Username đăng nhập PostgreSQL |
| `KC_DB_PASSWORD` | `<password_postgres>` | Password đăng nhập PostgreSQL |
| `KC_HOSTNAME` | `https://keycloak.omama.vn` | Domain public truy cập Keycloak (bắt buộc đúng domain HTTPS) |
| `KC_PROXY_HEADERS` | `xforwarded` | Bắt buộc khi chạy sau reverse proxy Traefik của Coolify |
| `KC_HTTP_ENABLED` | `true` | Bật HTTP listener cho kết nối nội bộ từ reverse proxy |
| `KC_HOSTNAME_BACKCHANNEL_DYNAMIC` | `true` | Hỗ trợ backend services gọi Keycloak qua mạng nội bộ |
| `KC_HEALTH_ENABLED` | `true` | Bật endpoint health check (`/health/ready`) |
| `KC_METRICS_ENABLED` | `true` | Bật endpoint metrics |

> [!IMPORTANT]
> - Đảm bảo Application Keycloak và PostgreSQL resource nằm trên cùng một Docker Network của Coolify để phân giải tên miền nội bộ `<ten_postgres_resource>`.
> - Biến `KC_PROXY_HEADERS=xforwarded` và `KC_HTTP_ENABLED=true` là điều kiện tiên quyết để tránh lỗi chuyển hướng lặp (`ERR_TOO_MANY_REDIRECTS`).

### Bước 4: Triển khai (Deploy)
Nhấn **Deploy** trên Coolify. Quá trình diễn ra:
1. Coolify clone source code và build image theo `Dockerfile`.
2. Lệnh `kc.sh build` tối ưu hóa Quarkus.
3. Container khởi chạy lệnh `start --optimized --import-realm`.
4. Khi khởi chạy lần đầu tiên trên database mới, cờ `--import-realm` tự động nạp cấu hình từ `realm.json` (tạo realm `omama`, nạp theme tiếng Việt, cấu hình client và tài khoản khởi tạo).
5. Từ lần khởi động thứ hai trở đi, Keycloak phát hiện realm `omama` đã tồn tại và sẽ bỏ qua import, vào trạng thái sẵn sàng phục vụ ngay.

---

## 4. Tài Khoản Khởi Tạo Super Admin

Trong file template `realm-export.template.json`, tài khoản khởi tạo được thiết lập sẵn cho cả quản trị Keycloak và điều hành hệ thống:

| Thông tin | Giá trị |
|---|---|
| **Tên đăng nhập (Username)** | `admin` |
| **Mật khẩu khởi tạo** | `admin` |
| **Email** | `admin@omama.vn` (đã xác thực) |
| **Họ và tên** | Super Admin |
| **Vai trò Keycloak (Realm Management)** | `realm-admin` (Toàn quyền quản trị realm `omama` trên Keycloak Admin Console) |
| **Vai trò Hệ thống (Realm Role)** | `ADMIN` |
| **Định danh nội bộ (`app_user_id`)** | `a0000000-0000-0000-0000-000000000001` |
| **Loại chủ thể (`principal_type`)** | `EMPLOYEE` |
| **Đặc quyền dịch vụ (`permission_grants`)** | Toàn bộ 30 quyền hạn nghiệp vụ (`#GLOBAL`) của các microservices (Catalog, Order, Profile, Inventory, Fulfillment, Analytics, Payments, IAM) |

> [!TIP]
> Sau khi deploy thành công lần đầu trên môi trường production, hãy đăng nhập vào Admin Console và đổi mật khẩu tài khoản `admin` để đảm bảo an toàn thông tin.

---

## 5. Lấy Client Secret Cho `identity-service`

1. Truy cập Admin Console: `https://<domain-keycloak>/admin` (ví dụ `https://keycloak.omama.vn/admin`).
2. Đăng nhập bằng tài khoản `admin` (hoặc master admin).
3. Ở menu góc trên bên trái, chuyển Realm sang **`omama`**.
4. Vào menu bên trái chọn **Clients** → click chọn client **`identity-service-admin`**.
5. Chuyển sang tab **Credentials** → sao chép giá trị **Client Secret**.
6. Dán secret vừa sao chép vào cấu hình biến môi trường của `identity-service`:
   ```env
   KEYCLOAK_CLIENT_SECRET=<secret_vua_sao_chep>
   ```

---

## 6. Giao Diện Đăng Nhập Ô Mạ (Chỉ Tiếng Việt)

- Thư mục theme: `themes/omama/login/` (tự động nạp vào `/opt/keycloak/themes/` lúc build container).
- Cấu hình chỉ hỗ trợ tiếng Việt:
  - `realm-export.template.json`: `"internationalizationEnabled": false`, `"supportedLocales": ["vi"]`, `"defaultLocale": "vi"`.
  - `theme.properties`: `locales=vi`, `defaultLocale=vi`.
- Link "Quay lại" / "Tiếp tục với tư cách khách" được tính toán tự động dựa trên `redirect_uri` và `state` của OAuth client, không bị hardcode URL.
- **Lưu ý cú pháp:** Không sử dụng `?html` trong các file template `.ftl` (Keycloak 24 đã tự động escape; dùng thêm `?html` sẽ gây lỗi server 500).

---

## 7. Bảng Khắc Phục Sự Cố (Troubleshooting)

| Triệu chứng | Nguyên nhân thường gặp | Cách xử lý |
|---|---|---|
| `ERR_TOO_MANY_REDIRECTS` | Thiếu proxy headers sau reverse proxy Traefik của Coolify | Kiểm tra biến môi trường: `KC_PROXY_HEADERS=xforwarded`, `KC_HTTP_ENABLED=true`, `KC_HOSTNAME=https://<domain_cua_ban>` |
| Keycloak không kết nối được PostgreSQL | Tên host hoặc thông tin kết nối DB không chính xác | Kiểm tra lại `KC_DB_URL`. Đảm bảo container Keycloak và PostgreSQL cùng mạng Docker nội bộ. |
| `Invalid parameter: redirect_uri` | Client frontend chưa khai báo redirect URI hợp lệ | Vào Admin Console -> Realm `omama` -> Clients -> `frontend-web` -> Thêm URL callback vào **Valid redirect URIs** |
| `An internal server error has occurred` ở trang login | Lỗi cú pháp FreeMarker trong theme `.ftl` | Xem log container bằng `docker logs omafood-keycloak-standalone`. Loại bỏ các filter cũ như `?html`. |
| Cập nhật `realm-export.template.json` nhưng không thấy thay đổi | Cờ `--import-realm` bỏ qua import nếu realm `omama` đã có trong database | Xóa database và nạp lại từ đầu, hoặc cập nhật trực tiếp cấu hình qua Admin Console / CLI `kcadm.sh`. |
