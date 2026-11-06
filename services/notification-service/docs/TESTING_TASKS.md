# Công việc kiểm thử Notification Service

Cập nhật 2026-11-06. Branch `codex/notification-service`; source checkpoint `40461df629344e3e357f7c4af959db505364a567`. Tài liệu này theo dõi việc kiểm thử cần thực hiện; [VALIDATION](VALIDATION.md) giữ ma trận acceptance toàn service. Trạng thái dưới đây lấy từ checkpoint triển khai, không phải kết quả chạy lại trong lần cập nhật tài liệu này.

## Kết quả đã có

| Hạng mục | Trạng thái / phạm vi bằng chứng |
|---|---|
| Unit, HTTP và contract Go tests | PASS: `go test -race -count=1 -v ./...`; 5 top-level tests cùng các subcase. PostgreSQL test bị SKIP, không tính là DB PASS |
| Static checks | `go vet ./...` PASS |
| Build | `go build ./...` PASS với portable Go 1.27.3; chưa chứng minh runtime với DB thật |
| Schema/fixture | `python scripts/verify-contracts.py` PASS cho schema/fixture Order đã pin và command local hợp lệ/không hợp lệ |
| Required integration gate | Đã kiểm tra: thiếu DB URL và đặt `NOTIFICATION_REQUIRE_INTEGRATION=1` thì test FAIL đúng kỳ vọng |
| PostgreSQL thật | NOT RUN: chưa có DB test hoạt động; `TestPostgresIntake` SKIPPED |
| Kafka/Redis/provider/Email end-to-end | NOT RUN; các adapter/worker tương ứng chưa triển khai |

## Việc đã kiểm thử trong N1

- [x] Local mode, loopback listener, thiếu database URL, local token quá ngắn và từ chối production mode.
- [x] Intent giữ nguyên khi context thay đổi nhưng fingerprint phải đổi; audience khác tạo intent khác.
- [x] HTTP 202 cho request mới và 200 cho duplicate qua store stub; không diễn giải thành Email đã gửi.
- [x] HTTP 401 khi thiếu xác thực; 403 cho purpose không được cấp; đầu vào sai không gọi repository.
- [x] Từ chối unknown fields, JSON nối thêm, idempotency key sai, recipient sai và body vượt giới hạn.
- [x] HTTP 409 từ store conflict; HTTP 503 khi store lỗi; không lộ lỗi DB nhạy cảm trong response.
- [x] Liveness độc lập readiness; readiness lỗi trả 503; foreign request trả 404 qua stub.
- [x] Kiểm tra schema hash đã pin, fixture local và event subscription vẫn disabled.

Các kiểm thử dùng stub chỉ chứng minh mapping/authorization/validation ở tầng tương ứng. Atomicity, khóa và durable dedup vẫn cần PostgreSQL thật.

## T1 — Khôi phục môi trường kiểm thử

- [ ] Có Docker engine hoạt động hoặc PostgreSQL disposable được cấp riêng.
- [ ] Nếu cần sửa Docker/WSL của host, xác định phạm vi và phương án bảo toàn dữ liệu trước khi thao tác. Không factory reset, xóa volume hoặc sửa stack khác để chạy test.
- [ ] Kiểm tra port `15417` còn trống trước khi tạo stack riêng.
- [ ] Cấp `NOTIFICATION_TEST_PASSWORD` qua environment; không đưa giá trị vào Markdown, Git hoặc log.
- [ ] Khởi động riêng stack `pbl6-notification-test` bằng Compose của service; xác minh PostgreSQL healthy.

Blocker quan sát ở checkpoint triển khai: Docker Desktop không khởi động backend vì không remove/listen được `%LOCALAPPDATA%/Docker/run/dockerInference`; WSL cũng từng báo `HCS_E_SERVICE_NOT_AVAILABLE`. Đây là bằng chứng lần chạy trước; cần kiểm tra lại khi tiếp tục, không coi là trạng thái hiện thời đã xác minh.

## T2 — Chạy các test PostgreSQL đã viết

`tests/integration/intake_test.go` đã chuẩn bị các trường hợp sau, nhưng chưa chạy với DB thật:

- [ ] Migration trên schema mới thành công; chạy lại migration không làm thay đổi dữ liệu.
- [ ] 50 request đồng thời cùng key/body tạo đúng một request, cùng ID và chỉ một lần được coi là original.
- [ ] Key khác nhưng cùng business intent trả request cũ.
- [ ] Cùng key với context thay đổi trả conflict; key mới nhưng cùng intent có fingerprint khác cũng conflict.
- [ ] Producer khác không đọc được request, trả not found.
- [ ] Chỉ có một request và một audit obligation sau duplicate requests.
- [ ] Audit insert thất bại làm rollback toàn bộ intake; không còn request ghi dở.
- [ ] Migration checksum bị sửa phải bị từ chối.
- [ ] Sau test chỉ cleanup schema random do test tạo, không xóa database/volume chung.

## T3 — Bổ sung nghiệm thu N1 còn thiếu

- [ ] Chạy binary thật với migration explicit và database riêng; POST rồi GET đúng persisted request.
- [ ] Restart process rồi retry cùng key: giữ nguyên request ID và không tạo audit/request trùng.
- [ ] DB outage/recovery: readiness đổi đúng trạng thái; không trả 202 khi chưa commit; liveness vẫn phản ánh process.
- [ ] Fault injection cho DB commit outcome không rõ: retry cùng key không tạo request mới sau commit đã thành công.
- [ ] Kiểm tra payload mới sau rollback không có idempotency record mồ côi; lỗi không làm transaction bị giữ lâu.
- [ ] Graceful shutdown khi đang có request: không mất committed request, không treo pool/process.
- [ ] Kiểm tra `/metrics` không chứa contact, context payload, token hoặc request ID trong labels.
- [ ] Kiểm tra token sai/thiếu, header identity giả, nhiều Authorization headers, producer/purpose bị cấm và request ngoài scope.
- [ ] Ghi command, exit code, PASS/FAIL/SKIP, toolchain và source SHA vào evidence Markdown trong `docs/verification/`.

N1 chỉ được ACCEPTED sau khi các yêu cầu DB/runtime tương ứng đã chạy và lỗi đã xử lý. Không mở N2 chỉ vì unit tests PASS.

## Lệnh chạy

Chạy từ `services/notification-service`. Cấp các biến environment ở máy chạy; placeholder dưới đây không phải credentials.

```powershell
# Unit/race/vet/build; DB integration SKIP nếu chưa có URL.
./scripts/verify.ps1

# Nếu dùng portable Go đã tải trên máy hiện tại:
./scripts/verify.ps1 -Go 'C:\Users\HP\.codex\tmp\notification-go-1.27.3\go\bin\go.exe'

# Chỉ sau khi đã cấp disposable password và Docker hoạt động:
docker compose -f deploy/local/compose.yaml up -d --wait

# Cấp NOTIFICATION_TEST_DATABASE_URL đến DB test trước lệnh này.
# -Integration bắt buộc có URL và không chấp nhận việc thiếu DB.
./scripts/verify.ps1 -Integration

python scripts/verify-contracts.py
```

Compose dùng database/user `notification_test`, host `127.0.0.1:15417`, project/volume riêng. Test dùng schema random trong DB được chỉ định. Không trỏ test URL vào production hoặc DB có dữ liệu thật. Python schema check cần `jsonschema`; môi trường checkpoint dùng 4.26.0. Nếu thiếu Go trong PATH, dùng tham số `-Go` cả khi chạy `-Integration`.

## T4 — Kiểm thử các phase tiếp theo

Chỉ triển khai khi phase tương ứng mở và N1 đã đạt DoD; chi tiết acceptance ở VALIDATION:

- [ ] N2: lease/fencing, crash trước/sau dispatch, stable send key, UNKNOWN/reconciliation, retry budget/expiry, template safety, callback duplicate/out-of-order, suppression và audit.
- [ ] N3: đúng recipient/audience/cohort của Order; OTP theo Identity owner; invoice/artifact đúng quyền; owner contract và auth integration thật.
- [ ] N4: PostgreSQL/Kafka restart, backlog/backpressure, privacy/key rotation/retention, runbook và scope-approved provider acceptance.
- [ ] N5–N7: SMS/Zalo fallback/ordering, Push endpoint ownership/consent, Internal Alert scope/escalation theo entry plan của từng phase.

Không ghi “delivered” từ HTTP 202 hoặc provider acceptance; không ghi “read” nếu thiếu bằng chứng đọc. Không đưa OTP, contact thật, raw callback hoặc credentials vào evidence.

## Bước tiếp theo

T1 → T2 → T3 → xác nhận N1 DoD → mở detailed plan N2. Toàn bộ test code/evidence vẫn ở thư mục Notification; không chỉnh service khác hoặc shared infrastructure ngoài phạm vi được giao.
