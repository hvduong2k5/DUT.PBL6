# SRE OPERATIONS RUNBOOK: MS-15 PROFILE SERVICE
## HỆ THỐNG THƯƠNG MẠI ĐIỆN TỬ ĐẶC SẢN OCOP HUẾ (MÈ XỬNG O MẠ)

Tài liệu hướng dẫn trực ca (On-Call Runbook) xử lý các sự cố khẩn cấp liên quan tới dịch vụ `profile-service`.

---

## 1. Thông Tin Tổng Quan Về Dịch Vụ
- **Tên dịch vụ:** `profile-service` (MS-15)
- **Cụm Kubernetes:** `omamx-ecommerce`
- **Port:** HTTP `8080` (D2C REST API), gRPC `50051` (Nội bộ liên-service)
- **Critical Path Phụ Trách:** Cung cấp thông tin địa chỉ giao hàng (`GetDeliveryAddress`) cho `MS-04 order-service` trong luồng Checkout ($P99 \le 5\text{ms}$).
- **Hạ Tầng Phụ Thuộc:** PostgreSQL 16 (`profile_db`), Redis 7 (`profile_cache`), HashiCorp Vault (Transit Engine), Kafka (`profile.events.v1`).

---

## 2. Quy Trình Ứng Phó Các Sự Cố Trọng Yếu (Incident Playbooks)

### Sự Cố 1: `ProfileServiceCheckoutSLAViolated` (P99 Latency > 5ms)
- **Mức độ:** `CRITICAL` (Trực tiếp ảnh hưởng tỷ lệ chuyển đổi đơn hàng kẹo mè xửng trên Web/Mobile).
- **Triệu chứng:** Prometheus Alert bắn kênh `#alerts-backend-p0`: `histogram_quantile(0.99) > 0.005` trong hơn 2 phút.
- **Các bước điều tra:**
  1. Kiểm tra tỷ lệ Cache Hit L1/L2:
     ```promql
     sum(rate(profile_cache_operations_total{layer="l1_ram", status="hit"}[5m])) / sum(rate(profile_cache_operations_total{layer="l1_ram"}[5m]))
     ```
     Nếu tỷ lệ Hit < 85%, kiểm tra xem có hiện tượng Cache Stampede hoặc Pod vừa khởi động lại hàng loạt hay không.
  2. Kiểm tra độ trễ kết nối CSDL PostgreSQL:
     ```bash
     kubectl exec -it deployment/profile-service -n omamx-ecommerce -- wget -qO- http://localhost:8080/readyz
     ```
  3. Kiểm tra số lượng kết nối DB pool:
     ```sql
     SELECT count(*), state FROM pg_stat_activity WHERE datname = 'profile_db' GROUP BY state;
     ```
- **Hành động khắc phục:**
  - Nếu kết nối DB bị nghẽn do thợ xưởng hoặc batch export: Scale tạm thời số connection pool `DB_MAX_CONNS=100` hoặc kích hoạt HPA scale pod.
  - Khởi động lại Redis nếu Redis latency tăng đột biến: `kubectl rollout restart deployment/profile-redis -n omamx-ecommerce`.

---

### Sự Cố 2: `ProfileServiceVaultUnwrapErrors` (Vault Mất Kết Nối / Lỗi Giải Mã CCCD)
- **Mức độ:** `WARNING` (Chỉ ảnh hưởng màn hình HR xem hồ sơ thợ xưởng, **hoàn toàn không ảnh hưởng luồng Checkout của khách**).
- **Triệu chứng:** HR báo lỗi 500 khi click xem số CCCD của thợ nấu kẹo.
- **Các bước điều tra:**
  1. Kiểm tra trạng thái cụm Vault:
     ```bash
     vault status -address=http://vault.security.svc.cluster.local:8200
     ```
  2. Kiểm tra xem Vault có bị Sealed không (`Sealed: true`):
     Nếu Vault bị Sealed, liên hệ Security Officer để Unseal với Shamir Keys.
  3. Kiểm tra AppRole Token hết hạn:
     Kiểm tra logs của `profile-service`:
     ```bash
     kubectl logs -l app=profile-service -n omamx-ecommerce | grep "vault"
     ```
- **Hành động khắc phục:**
  - Cập nhật K8s Secret `profile-vault-secret` với token mới nếu token bị thu hồi.
  - Nhờ cơ chế In-memory Short-lived DEK cache, các phiên làm việc đang mở của HR trong vòng 5 phút vẫn tiếp tục xem được CCCD mà không bị ngắt quãng.

---

### Sự Cố 3: `ProfileServiceOutboxLagHigh` (Dồn Ứ Sự Kiện Outbox > 100 Tin)
- **Mức độ:** `WARNING`
- **Triệu chứng:** Các sự kiện thay đổi hồ sơ hoặc địa chỉ mặc định không được đẩy kịp thời sang Kafka topic `profile.events.v1`.
- **Các bước điều tra:**
  1. Kiểm tra kết nối tới Kafka Broker:
     ```bash
     kubectl logs -l app=profile-service -n omamx-ecommerce | grep "failed writing message to kafka"
     ```
  2. Kiểm tra số lượng bản ghi tồn đọng trong DB:
     ```sql
     SELECT COUNT(*), min(created_at) FROM outbox_events WHERE published_at IS NULL;
     ```
- **Hành động khắc phục:**
  - Nếu Kafka Broker bị treo: Khởi động lại Kafka broker pod.
  - Tăng tốc độ giải phóng outbox: Tạm thời điều chỉnh biến môi trường `OUTBOX_POLL_INTERVAL_MS=100` và `OUTBOX_BATCH_SIZE=200`.

---

### Sự Cố 4: Cảnh Báo Hết Hạn Chứng Chỉ VSATTP OCOP (`StaffComplianceWarningEvent`)
- **Mức độ:** `NOTICE` (Tuân thủ kiểm soát chất lượng OCOP 4 sao).
- **Kịch bản:** Worker phát hiện thợ xưởng có chứng chỉ an toàn thực phẩm hết hạn trong 7 ngày tới.
- **Hành động:**
  - Kiểm tra danh sách nhân sự được cảnh báo qua API:
    ```bash
    curl -H "X-User-Role: HR_MANAGER" http://localhost:8080/api/v1/admin/employees?status=ACTIVE
    ```
  - Thông báo cho Trưởng phòng nhân sự và Quản đốc xưởng Hương Thủy bố trí lịch khám sức khỏe và cấp mới giấy chứng nhận cho thợ nấu kẹo.
