# Hướng Dẫn Triển Khai Keycloak Độc Lập Với Coolify

Tài liệu này hướng dẫn chi tiết cách triển khai **Keycloak Standalone** (phiên bản 24+) trên nền tảng **Coolify**, sử dụng **máy chủ PostgreSQL có sẵn** của bạn và hỗ trợ HTTPS tự động qua Reverse Proxy Traefik.

---

## 1. Yêu Cầu Chuẩn Bị (Prerequisites)

1. **Máy chủ PostgreSQL có sẵn:**
   - Đã tạo sẵn database tên là `keycloak_db`:
     ```sql
     CREATE DATABASE keycloak_db;
     ```
   - Đảm bảo máy chủ PostgreSQL mở port `5432` và cho phép địa chỉ IP của server Coolify kết nối vào (kiểm tra firewall và file `pg_hba.conf`).
2. **Tên miền (Domain / Subdomain):**
   - Đã trỏ bản ghi DNS (A record) của subdomain về địa chỉ IP server Coolify (ví dụ: `keycloak.omama.vn` hoặc `auth.omama.vn`).
3. **Coolify:** Đang chạy phiên bản Coolify v4+.

---

## 2. Hướng Dẫn Deploy Trên Coolify (Docker Compose)

### Bước 1: Tạo Resource mới trên Coolify Dashboard
1. Đăng nhập vào Coolify Dashboard.
2. Chọn **Project** và **Environment** của bạn (ví dụ: `Production`).
3. Bấm nút **+ New Resource** -> Chọn **Docker Compose**.
4. Chọn Server/Destination muốn deploy.

---

### Bước 2: Dán nội dung Docker Compose vào Coolify
Trong tab **Docker Compose**, dán nội dung cấu hình sau (hoặc dùng file [`docker-compose.yml`](./docker-compose.yml)):

```yaml
version: '3.8'

services:
  keycloak:
    image: quay.io/keycloak/keycloak:24.0.5
    container_name: omafood-keycloak-standalone
    restart: unless-stopped
    command: start --import-realm
    environment:
      # Tài khoản Admin tối cao Keycloak
      KEYCLOAK_ADMIN: ${KEYCLOAK_ADMIN}
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_ADMIN_PASSWORD}

      # Kết nối đến PostgreSQL server có sẵn của bạn
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://${POSTGRES_HOST}:${POSTGRES_PORT}/${KEYCLOAK_DB_NAME}
      KC_DB_USERNAME: ${POSTGRES_USER}
      KC_DB_PASSWORD: ${POSTGRES_PASSWORD}

      # Cấu hình Reverse Proxy Traefik của Coolify (BẮT BUỘC để không bị lỗi HTTPS)
      KC_PROXY_HEADERS: xforwarded
      KC_HOSTNAME: ${KC_HOSTNAME}
      KC_HTTP_ENABLED: "true"

      # Health & Metrics
      KC_HEALTH_ENABLED: "true"
      KC_METRICS_ENABLED: "true"
    volumes:
      - /app/realm.json:/opt/keycloak/data/import/realm.json:ro
    networks:
      - coolify
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/127.0.0.1/8080; echo -e 'GET /health/ready HTTP/1.1\\r\\nHost: localhost\\r\\nConnection: close\\r\\n\\r\\n' >&3; grep -q 'HTTP/1.1 200 OK' <&3"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s

networks:
  coolify:
    external: true
```

> **Lưu ý quan trọng cho Coolify:**
> - `KC_PROXY_HEADERS=xforwarded`: Bắt buộc để Keycloak hiểu các header `X-Forwarded-For`, `X-Forwarded-Proto` do Traefik gửi tới.
> - `KC_HTTP_ENABLED=true`: Cho phép Traefik giao tiếp với container qua HTTP nội bộ trên port 8080, trong khi người dùng truy cập bên ngoài bằng HTTPS an toàn.

---

### Bước 3: Cấu hình Tên Miền (Domains / FQDN)
1. Trong phần cài đặt của Application trên Coolify, tìm mục **Domains** (hoặc **FQDN**).
2. Điền domain đầy đủ có tiền tố HTTPS:
   ```
   https://keycloak.omama.vn
   ```
3. Port nội bộ để Coolify ánh xạ vào là: **`8080`**.
4. Coolify sẽ tự động cấu hình Traefik và xin chứng chỉ SSL Let's Encrypt miễn phí.

---

### Bước 4: Khai báo Biến Môi Trường (Environment Variables)
Chuyển sang tab **Environment Variables** trên Coolify và thêm các biến sau:

| Tên biến | Ví dụ giá trị | Ghi chú |
| :--- | :--- | :--- |
| `POSTGRES_HOST` | `103.xxx.xxx.xxx` (hoặc IP server) | IP máy chủ PostgreSQL của bạn |
| `POSTGRES_PORT` | `5432` | Cổng PostgreSQL |
| `KEYCLOAK_DB_NAME` | `keycloak_db` | Tên database Keycloak |
| `POSTGRES_USER` | `postgres` | Username DB |
| `POSTGRES_PASSWORD` | `MatKhauCuaBan123!` | Password DB |
| `KEYCLOAK_ADMIN` | `admin` | Tài khoản Admin Master |
| `KEYCLOAK_ADMIN_PASSWORD` | `MatKhauAdminKeycloak123!` | Mật khẩu Admin Master |
| `KC_HOSTNAME` | `https://keycloak.omama.vn` | URL công khai của Keycloak |

---

### Bước 5: Nạp File Cấu Hình Realm (`realm.json`)
Bạn có 2 cách để nạp file [`realm-export.template.json`](./realm-export.template.json):

#### Cách A: Mount File trực tiếp trên Coolify (Khuyên dùng)
1. Trong Coolify, chuyển sang tab **Storages** (hoặc **Persistent Storage**).
2. Thêm một File Mount:
   - **Source path on Host:** `/data/coolify/keycloak/realm.json`
   - **Destination path in Container:** `/opt/keycloak/data/import/realm.json`
   - **Nội dung:** Copy toàn bộ nội dung file [`realm-export.template.json`](./realm-export.template.json) dán vào đây.

#### Cách B: Import qua giao diện Admin UI (Đơn giản nhất nếu không rành mount volume)
1. Bỏ dòng `volumes` trong docker-compose và nhấn **Deploy**.
2. Sau khi Keycloak khởi động xong, truy cập `https://keycloak.omama.vn/admin`.
3. Đăng nhập bằng `KEYCLOAK_ADMIN` và `KEYCLOAK_ADMIN_PASSWORD`.
4. Nhấn vào dropdown menu chọn realm (góc trên bên trái, đang hiển thị **master**) -> Chọn **Create Realm**.
5. Tại mục **Resource file**, bấm **Browse...** và chọn file [`realm-export.template.json`](./realm-export.template.json) từ máy tính của bạn.
6. Bấm **Create**. Toàn bộ cấu hình Realm `omama`, Client `frontend-web`, Client `identity-service-admin` và các Role sẽ được tạo tự động 100%!

---

### Bước 6: Bấm Deploy
Nhấn nút **Deploy** trên Coolify Dashboard.
Xem tab **Logs**: Khi xuất hiện dòng:
```
[io.quarkus] (main) Keycloak 24.0.5 on JVM (powered by Quarkus ...) started in X.XXXs. Listening on: http://0.0.0.0:8080
```
=> Keycloak đã khởi chạy thành công và sẵn sàng hoạt động tại `https://keycloak.omama.vn`.

---

## 3. Các Bước Cấu Hình Bắt Buộc Sau Khi Deploy

### 1. Lấy Client Secret của `identity-service-admin`
1. Đăng nhập vào: `https://keycloak.omama.vn/admin`
2. Chuyển sang realm **`omama`**.
3. Vào menu **Clients** -> Chọn **`identity-service-admin`**.
4. Chuyển sang tab **Credentials**.
5. Copy chuỗi tại ô **Client Secret**.
6. Dán secret này vào file `.env` của `identity-service`:
   ```properties
   KEYCLOAK_CLIENT_SECRET=xxx_chuoi_secret_vua_copy_xxx
   ```

### 2. Gán quyền Quản Trị User cho Service Account (Quan trọng)
1. Vẫn trong client **`identity-service-admin`**, chọn tab **Service account roles**.
2. Bấm nút **Assign role**.
3. Tại ô dropdown bộ lọc, đổi từ *Filter by roles* sang **Filter by clients**.
4. Tìm client **`realm-management`** và tích chọn 3 quyền:
   - `manage-users` (Để Backend có quyền tạo user mới, suspend tài khoản)
   - `view-users` (Để tìm kiếm user theo email/sđt)
   - `manage-realm` (Để đồng bộ Role giữa Backend và Keycloak)
5. Bấm **Assign**.

### 3. Cấu hình CORS (Web Origins) cho Frontend Web & Mobile
1. Vào menu **Clients** -> Chọn **`frontend-web`**.
2. Cuộn xuống mục **Access settings**:
   - **Valid redirect URIs:**
     - `https://omama.vn/*`
     - `http://localhost:3000/*` (Cho môi trường phát triển local)
   - **Web origins:**
     - `https://omama.vn`
     - `http://localhost:3000`
     - `+` (Cho phép tất cả redirect URIs hợp lệ)
3. Bấm **Save**.

---

## 4. Khắc Phục Sự Cố Thường Gặp Trên Coolify (Troubleshooting)

### 1. Lỗi vòng lặp HTTPS (Redirect Loop: `ERR_TOO_MANY_REDIRECTS`)
- **Nguyên nhân:** Traefik của Coolify nhận HTTPS từ người dùng nhưng chuyển tiếp về Keycloak bằng HTTP, làm Keycloak tưởng kết nối chưa an toàn nên gửi redirect ngược lại HTTPS.
- **Xử lý:** Đảm bảo 2 biến môi trường sau đã được khai báo chính xác:
  ```properties
  KC_PROXY_HEADERS=xforwarded
  KC_HOSTNAME=https://keycloak.omama.vn
  KC_HTTP_ENABLED=true
  ```

### 2. Lỗi `Connection to localhost:5432 refused` hoặc không kết nối được PostgreSQL
- **Nguyên nhân:** 
  - Biến `POSTGRES_HOST` đặt là `localhost` hoặc `127.0.0.1` (trong container Docker, `localhost` trỏ vào chính nó chứ không phải máy chủ bên ngoài).
- **Xử lý:**
  - Nếu PostgreSQL nằm trên cùng VPS với Coolify: Dùng IP nội bộ của Docker gateway (ví dụ `172.17.0.1`) hoặc IP public của VPS.
  - Nếu PostgreSQL nằm trên server khác: Dùng IP public của server đó và đảm bảo firewall đã `ufw allow 5432/tcp`.

### 3. Lỗi `Invalid parameter: redirect_uri` khi login từ Web Frontend
- **Nguyên nhân:** Domain của Frontend chưa được thêm vào danh sách `Valid redirect URIs` của client `frontend-web`.
- **Xử lý:** Vào Client `frontend-web` trên Keycloak Admin Console, thêm `https://omama.vn/*` và `http://localhost:3000/*` vào mục **Valid redirect URIs**.

1. **Máy chủ PostgreSQL có sẵn:**
   - Đã tạo sẵn database tên là `keycloak_db`:
     ```sql
     CREATE DATABASE keycloak_db;
     ```
   - Đảm bảo máy chủ PostgreSQL mở port `5432` và cho phép địa chỉ IP của server Coolify kết nối vào (kiểm tra firewall và file `pg_hba.conf`).
2. **Tên miền (Domain / Subdomain):**
   - Đã trỏ bản ghi DNS (A record) của subdomain về địa chỉ IP server Coolify (ví dụ: `keycloak.omama.vn` hoặc `auth.omama.vn`).
3. **Coolify:** Đang chạy phiên bản Coolify v4+.

---

## 2. Hướng Dẫn Deploy Trên Coolify (Docker Compose)

### Bước 1: Tạo Resource mới trên Coolify Dashboard
1. Đăng nhập vào Coolify Dashboard.
2. Chọn **Project** và **Environment** của bạn (ví dụ: `Production`).
3. Bấm nút **+ New Resource** -> Chọn **Docker Compose**.
4. Chọn Server/Destination muốn deploy.

---

### Bước 2: Dán nội dung Docker Compose vào Coolify
Trong tab **Docker Compose**, dán nội dung cấu hình sau (hoặc dùng file [`docker-compose.yml`](./docker-compose.yml)):

```yaml
version: '3.8'

services:
  keycloak:
    image: quay.io/keycloak/keycloak:24.0.5
    container_name: omafood-keycloak-standalone
    restart: unless-stopped
    command: start --optimized --import-realm
    environment:
      # Tài khoản Admin tối cao Keycloak
      KEYCLOAK_ADMIN: ${KEYCLOAK_ADMIN}
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_ADMIN_PASSWORD}

      # Kết nối đến PostgreSQL server có sẵn của bạn
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://${POSTGRES_HOST}:${POSTGRES_PORT}/${KEYCLOAK_DB_NAME}
      KC_DB_USERNAME: ${POSTGRES_USER}
      KC_DB_PASSWORD: ${POSTGRES_PASSWORD}

      # Cấu hình Reverse Proxy Traefik của Coolify (BẮT BUỘC để không bị lỗi HTTPS)
      KC_PROXY_HEADERS: xforwarded
      KC_HOSTNAME: ${KC_HOSTNAME}
      KC_HTTP_ENABLED: "true"

      # Health & Metrics
      KC_HEALTH_ENABLED: "true"
      KC_METRICS_ENABLED: "true"
    volumes:
      - /app/realm.json:/opt/keycloak/data/import/realm.json:ro
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/127.0.0.1/8080; echo -e 'GET /health/ready HTTP/1.1\\r\\nHost: localhost\\r\\nConnection: close\\r\\n\\r\\n' >&3; grep -q 'HTTP/1.1 200 OK' <&3"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
```

> **Lưu ý quan trọng cho Coolify:**
> - `KC_PROXY_HEADERS=xforwarded`: Bắt buộc để Keycloak hiểu các header `X-Forwarded-For`, `X-Forwarded-Proto` do Traefik gửi tới.
> - `KC_HTTP_ENABLED=true`: Cho phép Traefik giao tiếp với container qua HTTP nội bộ trên port 8080, trong khi người dùng truy cập bên ngoài bằng HTTPS an toàn.

---

### Bước 3: Cấu hình Tên Miền (Domains / FQDN)
1. Trong phần cài đặt của Application trên Coolify, tìm mục **Domains** (hoặc **FQDN**).
2. Điền domain đầy đủ có tiền tố HTTPS:
   ```
   https://keycloak.omama.vn
   ```
3. Port nội bộ để Coolify ánh xạ vào là: **`8080`**.
4. Coolify sẽ tự động cấu hình Traefik và xin chứng chỉ SSL Let's Encrypt miễn phí.

---

### Bước 4: Khai báo Biến Môi Trường (Environment Variables)
Chuyển sang tab **Environment Variables** trên Coolify và thêm các biến sau:

| Tên biến | Ví dụ giá trị | Ghi chú |
| :--- | :--- | :--- |
| `POSTGRES_HOST` | `103.xxx.xxx.xxx` (hoặc IP server) | IP máy chủ PostgreSQL của bạn |
| `POSTGRES_PORT` | `5432` | Cổng PostgreSQL |
| `KEYCLOAK_DB_NAME` | `keycloak_db` | Tên database Keycloak |
| `POSTGRES_USER` | `postgres` | Username DB |
| `POSTGRES_PASSWORD` | `MatKhauCuaBan123!` | Password DB |
| `KEYCLOAK_ADMIN` | `admin` | Tài khoản Admin Master |
| `KEYCLOAK_ADMIN_PASSWORD` | `MatKhauAdminKeycloak123!` | Mật khẩu Admin Master |
| `KC_HOSTNAME` | `https://keycloak.omama.vn` | URL công khai của Keycloak |

---

### Bước 5: Nạp File Cấu Hình Realm (`realm.json`)
Bạn có 2 cách để nạp file [`realm-export.template.json`](./realm-export.template.json):

#### Cách A: Mount File trực tiếp trên Coolify (Khuyên dùng)
1. Trong Coolify, chuyển sang tab **Storages** (hoặc **Persistent Storage**).
2. Thêm một File Mount:
   - **Source path on Host:** `/data/coolify/keycloak/realm.json`
   - **Destination path in Container:** `/opt/keycloak/data/import/realm.json`
   - **Nội dung:** Copy toàn bộ nội dung file [`realm-export.template.json`](./realm-export.template.json) dán vào đây.

#### Cách B: Import qua giao diện Admin UI (Đơn giản nhất nếu không rành mount volume)
1. Bỏ dòng `volumes` trong docker-compose và nhấn **Deploy**.
2. Sau khi Keycloak khởi động xong, truy cập `https://keycloak.omama.vn/admin`.
3. Đăng nhập bằng `KEYCLOAK_ADMIN` và `KEYCLOAK_ADMIN_PASSWORD`.
4. Nhấn vào dropdown menu chọn realm (góc trên bên trái, đang hiển thị **master**) -> Chọn **Create Realm**.
5. Tại mục **Resource file**, bấm **Browse...** và chọn file [`realm-export.template.json`](./realm-export.template.json) từ máy tính của bạn.
6. Bấm **Create**. Toàn bộ cấu hình Realm `omama`, Client `frontend-web`, Client `identity-service-admin` và các Role sẽ được tạo tự động 100%!

---

### Bước 6: Bấm Deploy
Nhấn nút **Deploy** trên Coolify Dashboard.
Xem tab **Logs**: Khi xuất hiện dòng:
```
[io.quarkus] (main) Keycloak 24.0.5 on JVM (powered by Quarkus ...) started in X.XXXs. Listening on: http://0.0.0.0:8080
```
=> Keycloak đã khởi chạy thành công và sẵn sàng hoạt động tại `https://keycloak.omama.vn`.

---

## 3. Các Bước Cấu Hình Bắt Buộc Sau Khi Deploy

### 1. Lấy Client Secret của `identity-service-admin`
1. Đăng nhập vào: `https://keycloak.omama.vn/admin`
2. Chuyển sang realm **`omama`**.
3. Vào menu **Clients** -> Chọn **`identity-service-admin`**.
4. Chuyển sang tab **Credentials**.
5. Copy chuỗi tại ô **Client Secret**.
6. Dán secret này vào file `.env` của `identity-service`:
   ```properties
   KEYCLOAK_CLIENT_SECRET=xxx_chuoi_secret_vua_copy_xxx
   ```

### 2. Gán quyền Quản Trị User cho Service Account (Quan trọng)
1. Vẫn trong client **`identity-service-admin`**, chọn tab **Service account roles**.
2. Bấm nút **Assign role**.
3. Tại ô dropdown bộ lọc, đổi từ *Filter by roles* sang **Filter by clients**.
4. Tìm client **`realm-management`** và tích chọn 3 quyền:
   - `manage-users` (Để Backend có quyền tạo user mới, suspend tài khoản)
   - `view-users` (Để tìm kiếm user theo email/sđt)
   - `manage-realm` (Để đồng bộ Role giữa Backend và Keycloak)
5. Bấm **Assign**.

### 3. Cấu hình CORS (Web Origins) cho Frontend Web & Mobile
1. Vào menu **Clients** -> Chọn **`frontend-web`**.
2. Cuộn xuống mục **Access settings**:
   - **Valid redirect URIs:**
     - `https://omama.vn/*`
     - `http://localhost:3000/*` (Cho môi trường phát triển local)
   - **Web origins:**
     - `https://omama.vn`
     - `http://localhost:3000`
     - `+` (Cho phép tất cả redirect URIs hợp lệ)
3. Bấm **Save**.

---

## 4. Khắc Phục Sự Cố Thường Gặp Trên Coolify (Troubleshooting)

### 1. Lỗi vòng lặp HTTPS (Redirect Loop: `ERR_TOO_MANY_REDIRECTS`)
- **Nguyên nhân:** Traefik của Coolify nhận HTTPS từ người dùng nhưng chuyển tiếp về Keycloak bằng HTTP, làm Keycloak tưởng kết nối chưa an toàn nên gửi redirect ngược lại HTTPS.
- **Xử lý:** Đảm bảo 2 biến môi trường sau đã được khai báo chính xác:
  ```properties
  KC_PROXY_HEADERS=xforwarded
  KC_HOSTNAME=https://keycloak.omama.vn
  KC_HTTP_ENABLED=true
  ```

### 2. Lỗi `Connection to localhost:5432 refused` hoặc không kết nối được PostgreSQL
- **Nguyên nhân:** 
  - Biến `POSTGRES_HOST` đặt là `localhost` hoặc `127.0.0.1` (trong container Docker, `localhost` trỏ vào chính nó chứ không phải máy chủ bên ngoài).
- **Xử lý:**
  - Nếu PostgreSQL nằm trên cùng VPS với Coolify: Dùng IP nội bộ của Docker gateway (ví dụ `172.17.0.1`) hoặc IP public của VPS.
  - Nếu PostgreSQL nằm trên server khác: Dùng IP public của server đó và đảm bảo firewall đã `ufw allow 5432/tcp`.

### 3. Lỗi `Invalid parameter: redirect_uri` khi login từ Web Frontend
- **Nguyên nhân:** Domain của Frontend chưa được thêm vào danh sách `Valid redirect URIs` của client `frontend-web`.
- **Xử lý:** Vào Client `frontend-web` trên Keycloak Admin Console, thêm `https://omama.vn/*` và `http://localhost:3000/*` vào mục **Valid redirect URIs**.

