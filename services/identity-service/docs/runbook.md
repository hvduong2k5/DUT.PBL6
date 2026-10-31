# Identity Service — Runbook

## Tổng quan
Runbook này mô tả cách xử lý các sự cố vận hành thường gặp của identity-service trong môi trường production/staging.

**Service**: identity-service (MS-16)  
**Port**: 8016 (HTTP REST), 9016 (gRPC)  
**Database**: PostgreSQL — `identity_db`  
**Dependencies**: Keycloak, Kafka, PostgreSQL, Redis (nếu có cache)

---

## 1. Health & Readiness Checks

### Endpoint kiểm tra
```bash
# Liveness (service có đang sống không)
curl http://localhost:8016/actuator/health/liveness

# Readiness (service có sẵn sàng nhận traffic không)
curl http://localhost:8016/actuator/health/readiness

# Chi tiết tất cả health indicator
curl http://localhost:8016/actuator/health
```

### Kết quả mong đợi
```json
{
  "status": "UP",
  "components": {
    "db": { "status": "UP" },
    "kafka": { "status": "UP" }
  }
}
```

### Khi nào readiness là DOWN
- PostgreSQL không kết nối được.
- Flyway migration đang chạy hoặc thất bại.
- Kafka broker không có connection nào healthy.

---

## 2. Xử lý sự cố Keycloak không khả dụng

### Triệu chứng
- Endpoint `POST /api/v1/public/registration/customer` trả lỗi `500`.
- Log xuất hiện: `Failed to create user in Keycloak`.
- `registration_operations.operation_status = FAILED` nhưng `identity_principals.provisioning_status = PENDING`.

### Nguyên nhân
- Keycloak đang restart, deploy, hoặc mạng nội bộ bị gián đoạn.

### Xử lý
1. **Không panic**: Dữ liệu `PENDING` trong DB vẫn còn, không bị mất.
2. Kiểm tra Keycloak health: `curl http://keycloak:8080/health/ready`
3. Sau khi Keycloak phục hồi, Reconciliation Job sẽ tự động retry các operation ở trạng thái `FAILED` hoặc `PENDING`.
4. Kiểm tra kết quả: Query DB xác nhận operation đã `COMPLETED`.

```sql
-- Tìm các principal cần reconcile
SELECT p.principal_id, p.provisioning_status, o.operation_status, o.error_code
FROM identity_principals p
JOIN registration_operations o ON o.principal_id = p.principal_id
WHERE p.provisioning_status = 'PENDING'
  AND o.operation_status IN ('FAILED', 'PENDING')
  AND o.created_at < NOW() - INTERVAL '10 minutes';
```

5. **Không tự động xóa user Keycloak** làm compensation mặc định; chỉ xóa nếu business policy cho phép.

---

## 3. Xử lý sự cố JWKS Unknown Key ID

### Triệu chứng
- Gateway trả `401 Unauthorized` với reason `Unknown key ID`.
- Log tại Gateway: `No key found for kid: <some-kid>`.

### Nguyên nhân
- Keycloak đã rotate signing key nhưng Gateway vẫn dùng JWKS cache cũ.

### Xử lý
1. Buộc Gateway refresh JWKS cache (cấu hình theo service tương ứng).
2. Nếu vẫn lỗi, kiểm tra Keycloak JWKS endpoint: `curl https://keycloak.omama.vn/realms/omama/protocol/openid-connect/certs`
3. Xác nhận kid mới có trong response.
4. Kiểm tra JWKS cache TTL của Gateway và điều chỉnh nếu quá dài.

### Phòng ngừa
- Cấu hình Gateway cache JWKS với TTL <= 10 phút.
- Kích hoạt `on-unknown-kid: refresh` thay vì `reject` trong cấu hình Gateway.

---

## 4. Xử lý sự cố Kafka Outbox Publisher bị tồn đọng

### Triệu chứng
- Cột `outbox_events.delivery_status = 'PENDING'` tích tụ nhiều trong DB.
- Metric `outbox_lag` tăng cao.
- Consumer downstream (Profile Service) không nhận được event `UserRegistered`.

### Nguyên nhân
- Kafka broker unavailable.
- Outbox Publisher bị crash hoặc bị locked.
- Network timeout liên tục.

### Xử lý

**Bước 1**: Kiểm tra trạng thái Kafka
```bash
# Kiểm tra broker
kafka-broker-api-versions.sh --bootstrap-server kafka:9092

# Kiểm tra lag của consumer group
kafka-consumer-groups.sh --bootstrap-server kafka:9092 --describe --group identity-outbox-publisher
```

**Bước 2**: Kiểm tra các event tồn đọng
```sql
-- Đếm event chưa publish
SELECT delivery_status, COUNT(*), MIN(created_at) as oldest
FROM outbox_events
WHERE delivery_status IN ('PENDING', 'PROCESSING')
GROUP BY delivery_status;

-- Kiểm tra event bị stuck ở PROCESSING (publisher bị crash)
SELECT * FROM outbox_events
WHERE delivery_status = 'PROCESSING'
  AND locked_until < NOW()
ORDER BY created_at ASC
LIMIT 20;
```

**Bước 3**: Giải phóng lock nếu publisher bị crash (locked_until đã quá hạn)
```sql
-- Reset các event bị stuck về PENDING để Publisher retry
UPDATE outbox_events
SET delivery_status = 'PENDING',
    locked_by = NULL,
    locked_until = NULL
WHERE delivery_status = 'PROCESSING'
  AND locked_until < NOW();
```

**Bước 4**: Restart Outbox Publisher Service / Pod nếu cần.

---

## 5. Xử lý sự cố Database không kết nối được

### Triệu chứng
- Service khởi động thất bại.
- Readiness probe fail.
- Log: `Connection refused` hoặc `Too many connections`.

### Xử lý

**Kiểm tra connection pool**:
```bash
# Xem số connection hiện tại đến identity_db
psql -h postgres-host -U postgres -c "
SELECT count(*), state FROM pg_stat_activity
WHERE datname = 'identity_db'
GROUP BY state;
"
```

**Kiểm tra Flyway migration**:
```bash
# Nếu service không khởi động được do migration thất bại
psql -h postgres-host -U identity_user -d identity_db -c "
SELECT version, description, success, installed_on
FROM flyway_schema_history
ORDER BY installed_rank DESC LIMIT 10;
"
```

**Nếu migration thất bại**: Không xóa hoặc sửa V1-V5. Phân tích lỗi, tạo V6 để fix nếu cần.

---

## 6. Xử lý sự cố Reconciliation Backlog

### Khi nào cần reconcile thủ công
- Keycloak và PostgreSQL bị mất đồng bộ sau sự cố dài.
- `identity_principals.provisioning_status = PENDING` nhưng user thực tế đã tồn tại trong Keycloak.

### Quy trình kiểm tra
```sql
-- Tìm principal PENDING lâu hơn 1 giờ (có thể đã có trong Keycloak nhưng chưa update)
SELECT principal_id, idp_issuer, created_at
FROM identity_principals
WHERE provisioning_status = 'PENDING'
  AND created_at < NOW() - INTERVAL '1 hour';
```

1. Với mỗi `principal_id`, gọi Keycloak Admin API tìm user theo `app_registration_operation_id` attribute.
2. Nếu user tồn tại trong Keycloak: cập nhật `idp_subject`, set `provisioning_status = ACTIVE`, ghi outbox event.
3. Nếu user không tồn tại: đánh dấu operation là `FAILED`, để retry workflow hoặc xử lý thủ công tùy policy.

---

## 7. Security Runbook

### Token bị rò rỉ hoặc nghi ngờ bị chiếm đoạt

1. Vào Keycloak Admin Console → Users → tìm user → Sessions → Logout All Sessions.
2. Nếu cần vô hiệu hóa tài khoản ngay lập tức: Set `enabled = false` trong Keycloak.
3. Ghi incident vào `identity_audit_log` với `action_type = SECURITY_INCIDENT`.
4. Thông báo user thay đổi mật khẩu qua luồng Keycloak.

### Brute force nghi ngờ

- Keycloak có tích hợp Brute Force Detection. Kiểm tra cấu hình tại: Realm Settings → Security Defenses → Brute Force Detection.
- Kiểm tra log tại Keycloak Events (`Admin Events`, `Login Events`).

### Secret rotation (client-secret của identity-service-admin)

1. Vào Keycloak Admin → Clients → `identity-service-admin` → Credentials → Regenerate secret.
2. Cập nhật secret mới vào Secret Manager (không commit vào Git).
3. Restart identity-service để nhận config mới.
4. Xác nhận Keycloak Admin API hoạt động bình thường qua health check.

---

## 8. Backup & Restore

### Backup identity_db
```bash
# Full backup
pg_dump -h postgres-host -U identity_user identity_db \
  --format=custom -f /backup/identity_db_$(date +%Y%m%d_%H%M%S).pgdump

# Backup chỉ schema (cho recovery testing)
pg_dump -h postgres-host -U identity_user identity_db \
  --schema-only -f /backup/identity_db_schema_$(date +%Y%m%d).sql
```

### Restore
```bash
pg_restore -h postgres-host -U identity_user -d identity_db_new \
  /backup/identity_db_20261011.pgdump
```

**Sau khi restore**: Chạy Flyway migrate để đảm bảo schema đồng bộ với code hiện tại.

### Backup Keycloak Realm
```bash
# Export realm (không bao gồm password hash vì Keycloak không export credential dạng plain)
/opt/keycloak/bin/kc.sh export --realm omama --file /backup/realm-omama-$(date +%Y%m%d).json
```

---

## 9. Logs và Tracing

### Structured logging
- Mọi log phải có `correlation_id` (trace ID từ Gateway hoặc gRPC header).
- **Không log**: JWT token, password, OTP, client secret, raw fingerprint.
- Log level `INFO` cho registration success/failure, `WARN` cho idempotency conflict, `ERROR` cho unexpected exception.

### Tìm log theo correlation ID
```bash
# Ví dụ với ELK/Loki
grep '"correlation_id":"abc-123"' /var/log/identity-service/app.log
```

### Metrics quan trọng cần monitor
| Metric | Cảnh báo khi |
|---|---|
| `outbox_events_pending_count` | > 1000 trong 5 phút |
| `registration_failed_total` | Tăng đột biến |
| `keycloak_admin_api_errors_total` | > 10 trong 1 phút |
| `grpc_check_permission_p99_latency_ms` | > 500ms |
| `db_connection_pool_active` | > 80% pool size |
