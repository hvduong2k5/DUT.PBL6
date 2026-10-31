# MS-15 — Báo cáo hardening và kiểm chứng 08/10/2026

## Kết quả thực tế

| Kiểm tra | Kết quả | Bằng chứng |
| --- | --- | --- |
| Docker build bản cuối | PASS | Image `profile-hardening-api`, Go 1.22, runtime Alpine/non-root |
| `go vet ./...` | PASS, exit 0 | Chạy trước full race suite, không có diagnostic |
| `go test -race -count=1 -v ./...` | PASS, 18 top-level test và 63 subtest, không có failure/race | [go-test-race-output.txt](go-test-race-output.txt) |
| gRPC provider thật | PASS | `TestRealGRPCProvider`: generated client → TCP server đã register → PostgreSQL |
| Kafka + outbox worker thật | PASS, exit 0 | [kafka-test-output.txt](kafka-test-output.txt), `TestRealKafkaOutboxDelivery` chạy riêng với Compose connection settings |
| Postman CLI 1.70.0 collection lint | 38 requests, 0 errors, 0 warnings | Collection v3 đã lưu trong `postman/collections/` |
| Postman backend regression | **38 requests / 57 assertions / 0 failures**, exit 0 | [postman-cli-output.txt](postman-cli-output.txt), [postman-run-result.json](postman-run-result.json) |
| Postman OpenAPI lint local | No issues found, exit 0 | [openapi-lint-output.txt](openapi-lint-output.txt), [openapi-lint-result.json](openapi-lint-result.json) |
| PostgreSQL checkout benchmark | 3.454 iterations; **1.201.831 ns/op (~1,20 ms/op)**, 3.817 B/op, 75 allocs/op | [postgres-benchmark-output.txt](postgres-benchmark-output.txt) |

Postman chạy local với `--no-report-events`, không push workspace hoặc upload run. CLI có thông báo cần login để lấy thông tin/governance cloud; local spec lint vẫn hoàn tất exit 0. Không tuyên bố đã chạy governance ruleset của team.

Full Go suite mặc định skip `TestRealKafkaOutboxDelivery` nếu không có connection settings; ca này đã chạy riêng với **DB/Kafka/API worker thật**, không tính skip là pass. Testcontainers tạo PostgreSQL/Redis riêng và dùng các migration version-control thay vì DDL copy.

## Regression đã kiểm tra

- Anonymous/query customer ID và identity headers thiếu internal token bị từ chối.
- Auth subject ánh xạ tới profile ID khác nhau; subject không có profile trả 404.
- Customer không đọc được HR data; HR role truy cập đúng endpoint.
- Địa chỉ khách khác không đọc/sửa được; địa chỉ đã xóa không đọc được.
- First address default, second address non-default, switch đúng một default, retry switch không tăng version, delete default chọn replacement và delete cuối trả `[]`.
- 100 concurrent switch requests đều phải thành công; thêm 20 concurrent first creates và concurrent switches có kiểm tra từng error.
- Address stale update/delete bị từ chối; thiếu If-Match trả 428.
- Ward/province mismatch bị từ chối; tên địa giới giả từ client được thay bằng canonical master data.
- Profile CAS increment/version conflict; không đổi email qua Profile API; reject phone sai, tên/người nhận rỗng, unknown field.
- Outbox insert failure làm rollback profile update, address create và verified claim persistence.
- Guest claim không có verifier trả 503; không ghi claim. Claim GET chỉ trả owner.
- Consistency known/mismatch/unknown, fuzzy JSON field naming và tên đường chứa số.
- gRPC auth metadata, required customer ID, cross-customer NotFound, default lookup, profile/list RPC và deleted address.
- Cache L1 expiry/deep copy và invalidation cross-pod qua Redis.
- Kafka CloudEvent `vn.omama.profile.updated.v1`: specversion, event UUID, partition key, customer/version payload và persisted publication ACK.

## Sai lệch phát hiện trong quá trình kiểm thử

1. Collection cũ dùng `/profile/me`, `expected_version`, field consistency/fuzzy khác code, và chấp nhận nhiều status thay vì assert hành vi cụ thể. Đã cập nhật contract và assertions.
2. Hai “gRPC/Kafka tests” cũ chỉ là HTTP simulation/fixture validation. Đã thay bằng actual gRPC/Kafka tests; HTTP simulation route bị bỏ và Postman kiểm tra 405.
3. Dataset migration gán đường Mai Lão Bạng cho `WARD-TL-013`; ca “known match” được sửa theo dataset thực. Không dùng assertion để hợp thức hóa geography chưa được xác minh ngoài dataset.
4. `UNSPECIFIED` dài 11 ký tự nhưng cột gender là VARCHAR(10). Migration 000003 mở rộng VARCHAR(16).
5. Một unit test có sẵn cho `2 Tháng 9` thất bại do bỏ nhầm số của tên đường; đã sửa parser.

## Chạy lại

Từ repo root, Docker đang chạy và Postman CLI có trên PATH:

```powershell
$env:PROFILE_INTERNAL_TOKEN = '<temporary-local-test-token>'
./services/profile-service/scripts/test-postman.ps1 -StartStack
```

Go tools trong thư mục `services/profile-service`:

```sh
go vet ./...
go test -race -count=1 -v ./...
```

Cho Kafka end-to-end: provision topic `profile.events.v1` một partition trên stack test, sau đó đặt `TEST_PROFILE_DATABASE_URL` tới DB test và `TEST_PROFILE_KAFKA_BROKER` tới broker test; chạy `go test -race -count=1 -v -run TestRealKafkaOutboxDelivery ./tests/integration`. Broker test quảng bá `host.docker.internal:19092` để client trong Docker truy cập được.

Benchmark: đặt `TEST_PROFILE_DATABASE_URL` tới DB test đã migrate, chạy `go test -run '^$' -bench BenchmarkCheckoutPostgres -benchtime=3s ./tests/benchmark`. Không trỏ những lệnh này vào production.

## Giới hạn và tích hợp còn lại

- Guest claim đã được sửa để fail closed; **chưa có OTP/Order ownership verifier adapter**, nên flow claim thành công chưa được test đầu-cuối với provider thật. Không có SMS gửi ra ngoài.
- Caller nội bộ cần secret token cấu hình; browser không được nắm token. Production TLS/mTLS/secret rotation do deployment và caller phối hợp, chưa xác minh trên hạ tầng production.
- ETag là strong version token của backend trực tiếp. Contract `/customers/me` candidate vẫn chưa được phê duyệt hoặc thay thế frontend contract trong đợt này.
- Chưa xác minh production P99/load SLA. Benchmark trên Docker đo DB lookup, không đại diện latency percentile của toàn bộ checkout/network. Số 421,9 ns/op lịch sử từ mock không còn dùng làm bằng chứng SLA.
- Vault dùng mock ở stack local; không kiểm thử Vault production. MS-18 audit consumer và integration frontend/Identity/Order chưa nằm trong stack này.
- Các vùng/đường trong migration là dataset hiện có; kiểm thử nhất quán nội bộ không chứng minh catalog đầy đủ/chính xác cho toàn quốc.
- Không sửa Kong Gateway. Các file candidate đã staged trước task được giữ nguyên. Stack/container chỉ phục vụ task đã được dọn sau kiểm thử; Docker Desktop và các container development có sẵn được giữ nguyên.

Thiết kế mới: [06_hardening_and_contract.md](../../02_architecture/services/profile-service/06_hardening_and_contract.md). API trực tiếp: [profile-service.openapi.yaml](../../03_api_specs/profile-service.openapi.yaml).
