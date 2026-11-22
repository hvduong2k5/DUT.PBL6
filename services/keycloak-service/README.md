# Keycloak Service — Triển Khai Keycloak Độc Lập

Keycloak 24 (Quarkus) container hóa độc lập, tích hợp sẵn giao diện đăng nhập thuần tiếng Việt Ô Mạ (`themes/omama`), nạp tự động realm template `omama` với cấu hình bảo mật hiện đại và tài khoản khởi tạo Super Admin cho môi trường demo / phát triển.

```
services/keycloak-service/
├── Dockerfile                  # Image Keycloak tối ưu (Quarkus pre-build, nạp theme & realm)
├── Dockerfile.coolify          # File build triển khai Git-based trên Coolify (đồng bộ với Dockerfile)
├── .dockerignore               # Danh sách loại trừ khi build Docker image
├── .env.example                # Biến môi trường mẫu cho Production / Coolify
├── KEYCLOAK_SERVICE_AGENT_RULES.md # Quy chuẩn phát triển Keycloak Service
├── realm-export.template.json  # Realm template omama (token 15 phút, tiếng Việt, tài khoản demo)
├── order-user-profile-attribute.json
└── themes/omama/login/         # Giao diện đăng nhập Ô Mạ (tiếng Việt độc quyền)
```

---

## 1. Điểm Nổi Bật Về Kiến Trúc

- **Không dùng Docker Compose:** Dịch vụ triển khai trực tiếp bằng Dockerfile kết nối vào tài nguyên PostgreSQL độc lập có sẵn (trên Coolify hoặc máy chủ cơ sở dữ liệu dùng chung).
- **Hỗ trợ song song `Dockerfile` và `Dockerfile.coolify`:** Cả 2 file có nội dung chỉ thị build giống hệt nhau, đảm bảo Coolify build thành công dù cấu hình trỏ tới file nào.
- **Quarkus Pre-build (Optimized Build):** Dockerfile sử dụng multi-stage build chạy `kc.sh build` ngay lúc build image, giúp container khởi động tức thì trong 2-3 giây ở môi trường production (`start --optimized`).
- **Tối ưu hóa Hostname & Backchannel:**
  - `KC_HOSTNAME=https://keycloak.omama.vn`: Cung cấp public base URL cho OIDC discovery và issuer.
  - `KC_HOSTNAME_STRICT_BACKCHANNEL=false`: Cho phép các backend microservices trong mạng nội bộ Coolify gọi trực tiếp Keycloak qua địa chỉ Docker nội bộ (ví dụ `http://keycloak:8080`) mà không bị lỗi lệch URL hay bị ép redirect về domain public.
- **Giao diện thuần Tiếng Việt chuẩn Keycloak:**
  - `internationalizationEnabled: true`, `supportedLocales: ["vi"]`, `defaultLocale: "vi"`.
  - Giúp `DefaultLocaleSelectorProvider` của Keycloak nhận diện đúng tiếng Việt cho mọi thông báo hệ thống/lỗi, đồng thời tự động ẩn thanh chuyển đổi ngôn ngữ (do chỉ có 1 locale hợp lệ).
- **Tài khoản khởi tạo Super Admin (Demo / Development theo Baseline 7-B):**
  - Khởi tạo sẵn user `admin` (`admin@omama.vn` / mật khẩu `admin`) với vai trò `realm-admin` trên Keycloak Admin Console. Quyền hạn nghiệp vụ không hardcode trong template mà được cấp phát động thông qua Identity Service (migration `V21` liên kết principal và gán role `ADMIN` #GLOBAL).

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

---

## 3. Triển Khai Production Trên Coolify (Có Sẵn Resource PostgreSQL)

> [!NOTE]
> **Hiện trạng triển khai:** Trên Coolify, Keycloak có thể đang chạy dưới dạng prebuilt-image kết hợp resource PostgreSQL riêng. Phương án Git-based qua Dockerfile này là giải pháp chuẩn hóa mới giúp đóng gói sẵn theme tiếng Việt Ô Mạ và tối ưu hóa thời gian khởi động của Quarkus.

### Bước 1: Tạo database `keycloak_db` trong PostgreSQL resource
Truy cập giao diện quản lý PostgreSQL trên Coolify hoặc qua DBeaver/psql kết nối vào database server:
```sql
CREATE DATABASE keycloak_db;
```

### Bước 2: Tạo Application trên Coolify
1. Trong Project trên Coolify, nhấn **+ New Resource** → chọn **Git Repository**.
2. Chọn repo dự án: `https://github.com/hvduong2k5/DUT.PBL6`.
3. Cấu hình cơ bản (General):
   - **Build Pack:** `Dockerfile`
   - **Base Directory:** `services/keycloak-service`
   - **Dockerfile:** `Dockerfile` (hoặc `Dockerfile.coolify`)
   - **Port:** `8080`
   - **Domains:** Domain public của bạn, ví dụ `https://keycloak.omama.vn` hoặc `https://auth.omama.vn`.

### Bước 3: Khai báo Biến môi trường Runtime (Environment Variables)
Sao chép các biến sau vào mục **Environment Variables** của Application trên Coolify:

| Tên biến | Giá trị mẫu | Giải thích |
|---|---|---|
| `KEYCLOAK_ADMIN` | `admin` | Tài khoản Super Admin của Keycloak master realm |
| `KEYCLOAK_ADMIN_PASSWORD` | `<mat_khau_manh>` | Mật khẩu tài khoản Super Admin master realm |
| `KC_DB_URL` | `jdbc:postgresql://<ten_postgres_resource>:5432/keycloak_db` | URL kết nối database trên mạng nội bộ Coolify |
| `KC_DB_USERNAME` | `<user_postgres>` | Username đăng nhập PostgreSQL |
| `KC_DB_PASSWORD` | `<password_postgres>` | Password đăng nhập PostgreSQL |
| `KC_HOSTNAME` | `https://keycloak.omama.vn` | Domain public truy cập Keycloak |
| `KC_PROXY_HEADERS` | `xforwarded` | Bắt buộc khi chạy sau reverse proxy Traefik của Coolify |
| `KC_HTTP_ENABLED` | `true` | Bật HTTP listener cho kết nối nội bộ từ reverse proxy |
| `KC_HOSTNAME_STRICT_BACKCHANNEL` | `false` | Cho phép backend services gọi Keycloak qua mạng nội bộ |

> [!IMPORTANT]
> - Các biến build-time (`KC_DB=postgres`, `KC_HEALTH_ENABLED=true`, `KC_METRICS_ENABLED=true`) đã được đóng gói cố định trong image, không cần và không thể ghi đè lúc runtime.
> - Đảm bảo Application Keycloak và PostgreSQL resource nằm trên cùng một Docker Network của Coolify.

---

## 4. Tài Khoản Super Admin & Phân Quyền Theo Baseline 7-B

Theo kiến trúc chuẩn của hệ thống (Baseline 7-B), **Keycloak chỉ đóng vai trò IdP xác thực**, không quản lý hay viết tay danh sách quyền hạn nghiệp vụ (`permission_grants`) trong template. Toàn bộ quyền được quản lý và cấp phát từ cơ sở dữ liệu của **Identity Service** thông qua `PermissionSyncService`:

| Thông tin | Giá trị |
|---|---|
| **Tên đăng nhập (Username)** | `admin` |
| **Mật khẩu khởi tạo** | `admin` (`temporary: false`) |
| **Email** | `admin@omama.vn` (đã xác thực) |
| **Vai trò Keycloak (Realm Management)** | `realm-admin` (Toàn quyền quản trị realm `omama` trên Keycloak Admin Console) |
| **Vai trò Hệ thống (Realm Role)** | `ADMIN` |
| **Định danh nội bộ (`app_user_id` / `sub`)** | `a0000000-0000-0000-0000-000000000002` |
| **Loại chủ thể (`principal_type`)** | `EMPLOYEE` |
| **Đặc quyền dịch vụ (`permission_grants`)** | **Không viết tay trong template.** Được Identity Service (`identity_db`) bootstrap tự động qua migration `V21`, gán vai trò `ADMIN` với phạm vi `#GLOBAL`. Khi đăng nhập hoặc chạy đồng bộ, `PermissionSyncService` tự động nạp đầy đủ các quyền thực tế (bao gồm cả `traceability.*` của V20). |

> [!WARNING]
> Tài khoản này chỉ dành cho mục đích demo / kiểm thử nội bộ. Khi vận hành chính thức, khuyến nghị đổi mật khẩu tài khoản này trên Keycloak Admin Console hoặc quản lý cấp phát tài khoản nhân sự thông qua Identity Service.

---

## 5. Cập Nhật Realm Đang Chạy & Kịch Bản Khôi Phục (Disaster Recovery)

> [!CAUTION]
> **TUYỆT ĐỐI KHÔNG XÓA DATABASE TRÊN PRODUCTION:**  
> Cờ `--import-realm` chỉ nạp template khi realm `omama` chưa từng tồn tại trong CSDL. Xóa database trên production sẽ làm mất toàn bộ dữ liệu người dùng thật, token, và cấu hình đang hoạt động. Thao tác xóa DB chỉ áp dụng cho môi trường dev tạm thời.

### Cách cập nhật cấu hình cho Realm đang hoạt động trên Production:
1. **Cách 1: Qua Keycloak Admin Console (Khuyến nghị):**
   - Đăng nhập `https://keycloak.omama.vn/admin` → chọn Realm `omama`.
   - Vào **Realm settings** → **Localization** hoặc **Tokens** để điều chỉnh cấu hình trực tiếp. Thay đổi có hiệu lực ngay lập tức.
2. **Cách 2: Qua Keycloak Admin CLI (`kcadm.sh`):**
   - Thực thi lệnh trực tiếp trong container:
     ```bash
     /opt/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080 --realm master --user admin --password <password>
     /opt/keycloak/bin/kcadm.sh update realms/omama -s accessTokenLifespan=900 -s internationalizationEnabled=true
     ```

### Quy trình Sao lưu & Phục hồi (Disaster Recovery):
* **Sao lưu (Backup):**
  ```bash
  pg_dump -h <host_db> -U <user> -d keycloak_db > keycloak_backup_$(date +%F).sql
  ```
* **Phục hồi (Restore):**
  ```bash
  psql -h <host_db> -U <user> -d keycloak_db < keycloak_backup_YYYY-MM-DD.sql
  ```

---

## 6. Lấy Client Secret Cho `identity-service`

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

## 7. Bảng Khắc Phục Sự Cố (Troubleshooting)

| Triệu chứng | Nguyên nhân thường gặp | Cách xử lý |
|---|---|---|
| `ERR_TOO_MANY_REDIRECTS` | Thiếu proxy headers sau reverse proxy Traefik của Coolify | Kiểm tra biến môi trường: `KC_PROXY_HEADERS=xforwarded`, `KC_HTTP_ENABLED=true`, `KC_HOSTNAME=https://<domain_cua_ban>` |
| Keycloak không kết nối được PostgreSQL | Tên host hoặc thông tin kết nối DB không chính xác | Kiểm tra lại `KC_DB_URL`. Đảm bảo container Keycloak và PostgreSQL cùng mạng Docker nội bộ. |
| `Invalid parameter: redirect_uri` | Client frontend chưa khai báo redirect URI hợp lệ | Vào Admin Console -> Realm `omama` -> Clients -> `frontend-web` -> Thêm URL callback vào **Valid redirect URIs** |
| `An internal server error has occurred` ở trang login | Lỗi cú pháp FreeMarker trong theme `.ftl` | Xem log container bằng `docker logs omafood-keycloak-standalone`. Loại bỏ các filter cũ như `?html`. |
| Cập nhật `realm-export.template.json` nhưng không thấy áp dụng trên production | Cờ `--import-realm` bỏ qua import nếu realm đã tồn tại | **Không xóa database**. Cập nhật trực tiếp qua Admin Console hoặc lệnh `kcadm.sh update realms/omama` như hướng dẫn ở mục 5. |
