# Chạy và review Traceability Batch MVP

Nhánh `feat/traceability-service`. Đây là sandbox backend với PostgreSQL/broker thật,
source fixture và identity ký riêng; chưa phải tích hợp Catalog/Inventory/Identity
thật. Không commit hoặc gửi cho người khác `.env`, `.sandbox`, token hoặc mật khẩu.

## 1. Khởi tạo và chạy trên máy local

Cần Docker Desktop đang chạy, Go >=1.25 và PowerShell. Từ thư mục
`services/traceability-service`, chạy:

```powershell
./scripts/dev.ps1 -Action Init
./scripts/dev.ps1 -Action Start
./scripts/dev.ps1 -Action Smoke
```

`Init` chỉ tạo mật khẩu `.env` nếu file chưa tồn tại, dựng DB riêng ở localhost15403,
migrate rõ ràng, tạo runtime role hạn chế và bộ dữ liệu tổng hợp. Không reset DB,
không đổi mật khẩu/khóa khi dùng lại cùng `.sandbox`. Khởi tạo dở sẽ báo lỗi và
giữ file để kiểm tra; không tự xóa hoặc quay về credential mặc định.

`Start` build và đợi API healthy trên localhost8003. Image chạy UID10001; chỉ public
key và fixture được mount vào API, private key ở host. Lần đầu cần mạng để tải image/
Go modules. Profile Compose mặc định chỉ chạy DB; API và broker bật riêng.

`Smoke` dùng editor và approver khác nhau: create→submit→approve→publish→QR200→
new revision/edit→QR vẫn giữ snapshot cũ→suspend→QR410. Báo cáo nằm ở
`.sandbox/smoke-report.json`; `qr_url` là trang Web cấu hình, public API là
`/api/v1/trace/{qr_code}`. Runbook không tuyên bố frontend Web đã tương thích.

Fixture đã Suspended/Archived sẽ được xác minh lại bằng history/QR410 khi chạy
Smoke lần sau (`existing_terminal_fixture_verified`), không restore hoặc bịa một
full run mới. Tạo môi trường/dataset riêng nếu cần thực hiện lại toàn flow từ đầu.

Token hết hạn sau 10 phút; tạo token mới bằng:

```powershell
./scripts/dev.ps1 -Action Tokens
```

Không dán nội dung token/DSN vào log hoặc PR. Script khôi phục các biến môi trường
của shell sau khi chạy; API không tự đọc file `.env` khi chạy native.

## 2. Bật relay và kiểm tra vận hành

```powershell
./scripts/dev.ps1 -Action Start -Relay
Invoke-RestMethod http://127.0.0.1:8003/readyz
Invoke-WebRequest http://127.0.0.1:8003/metrics
```

Broker riêng là Redpanda Kafka-compatible, host19093/internal `kafka:9092`. Script
tạo topic `traceability.events.v1` nếu chưa có. Relay tắt giữ Pending; bật lại sẽ
gửi backlog. `traceability_relay_enabled`, pending/failed/oldest-age và delivered/
error counters phản ánh khả năng thực tế. Failed head chặn riêng hồ sơ đó; không
cho event mới vượt qua để che lỗi. Consumer cần dedup event ID và so version;
version có thể có khoảng trống. Shared peer consumers/Audit vẫn là gate R2.

Worker expiry mặc định bật, scan100 hồ sơ/tick30s và dedup observation theo
revision/reason/cutoff. Nó không tự Suspended root. Public guard tự tính hạn ở mỗi
request, kể cả lúc worker/broker ngừng. `verified_at` là kiểm tra OCOP thủ công;
`source_verified_at` là snapshot nguồn, không phải thời điểm scan hay EXT-10 verify.

`VIDEO_ALLOWED_HOSTS` rỗng mặc định từ chối mọi video URL. Muốn thử video, cấu hình
danh sách hostname HTTPS chính xác trước khi chạy API; không chấp nhận scheme,
credential hoặc port trong allowlist. Service chỉ lưu metadata, không fetch/upload.

## 3. Chạy kiểm thử

```powershell
./scripts/dev.ps1 -Action Test
./scripts/dev.ps1 -Action Race
```

Test chạy với DB/broker riêng, schema `trace_test_*` ngẫu nhiên và runtime role
ngắn hạn. Test dọn đúng schema/role do nó tạo; không đụng bảng public của fixture
đang review. Race dùng Go1.25 Linux + GCC trong `Dockerfile.test`, không phụ thuộc
GCC Windows. Thiếu test DSN/broker khi chạy `go test` trực tiếp sẽ SKIP có nhãn
NOT_RUN; không coi một lần chạy bị skip là integration PASS.

Validation contract, bằng Python3.11 hoặc phiên bản phù hợp dependency đã pin:

```powershell
python -m venv .sandbox/spec-venv
./.sandbox/spec-venv/Scripts/python.exe -m pip install -r scripts/spec-requirements.txt
./.sandbox/spec-venv/Scripts/python.exe -m unittest discover -s scripts -p test_validate_specs.py -v
./.sandbox/spec-venv/Scripts/python.exe scripts/validate_specs.py
./scripts/dev.ps1 -Action Tokens
./.sandbox/spec-venv/Scripts/python.exe scripts/validate_specs.py --runtime
```

Lệnh runtime đọc token từ file và chỉ gửi tới loopback; kiểm tra create fixture và
admin response đã redacted. Broker sample là dữ liệu tổng hợp trong artifacts;
bốn event variants là synthetic schema checks, không giả thành peer roundtrip.

## 4. Dừng, backup và rollback

```powershell
./scripts/dev.ps1 -Action Stop
```

Chỉ dừng API; DB/broker/volume vẫn giữ. SIGTERM dừng worker và drain HTTP/tài nguyên
trong cùng shutdown budget. Quá deadline trả lỗi, không giả đã drain thành công;
outbox lease hết hạn có thể được gửi lại với cùng event ID.

Trước thay đổi migration, backup DB riêng bằng `pg_dump` qua container `db` và lưu
vào `.sandbox` (quyền truy cập hạn chế); sao lưu riêng cấu hình/credential ở nơi
được phép. `pg_dump` database không bao gồm password của role. Phục hồi thử vào DB
cô lập trước, không restore đè DB đang chạy. Không dùng `docker compose down -v`
như thao tác rollback. Down migration trong code chỉ dành schema test dùng một lần.

Rollback binary về image đã biết cần giữ schema tương thích. Startup từ chối
checksum/mode/version khác; không sửa checksum hay xóa dữ liệu để vượt guard.
Profile restore v1 không có API. Chưa mở production hoặc EXT-10.

## 5. Điểm reviewer cần xem

- Contributor được kế thừa khi clone/thu hồi, kể cả clone từ bản cũ khác working.
- Self-approval và ADMIN bypass bị chặn; approve/reject có căn cứ và CAS.
- Verify nguồn ngoài transaction, sau đó lock/CAS/recheck freshness khi commit.
- Publish pointer/audit/outbox nguyên tử; chỉnh Draft không sửa public revision.
- Public allowlist không có evidence/actor/internal ID; 404/410/no-store đúng.
- Retry/lease/max-attempt/crash-window giữ ID/payload; worker không tạo event vô hạn.
- Giới hạn R2 và các câu hỏi X-01..X-09 trong file quyết định; fixture không phải
  bằng chứng provider hay consumer thật đã tương thích.

Đối chiếu kết quả thực tế trong [báo cáo kiểm chứng](./01_verification.md).
