# Order Service full regression ngày24 tháng10 năm2026

Kết luận: phần Order Service đã triển khai và các provider connectivity checks đều PASS. **Chưa nghiệm thu toàn bộ thiết kế Order Core3.2 hoặc chuỗi thanh toán SePay cập nhật PAID**, vì target worker/Payment runtime chưa tích hợp. Kết quả máy đọc được ở [results](07_full_regression_2026_10_24.results.json).

## Môi trường và phạm vi

- Isolated Docker Compose project `codex-order-audit-20261024`, DB runtime/simulator riêng, mỗi Go test thêm isolated schemas. PostgreSQL16, Redis7, pinned Apache Kafka, Go1.22.12 Linux/amd64, race/CGO. API/simulator image build lại từ working tree hiện tại, không dùng binary cũ.
- HTTP API/simulator và Kong3.9 chỉ mở loopback. Core nghiệp vụ và persistence dùng implementation hiện tại; Catalog/Inventory/Profile/Fulfillment/payment sandbox adapters trong integration tests vẫn là simulator.
- Provider thật được gọi là **SePay Test mode**, không phải Live: token API Access riêng, read-only API requests; không tạo/chuyển/hoàn tiền, không đổi cấu hình Dashboard SePay.
- Worker nhận webhook là Cloudflare sandbox capture độc lập đã được cấu hình trước đó, lưu D1 mã hóa. Không đưa API token vào Worker vì Worker chưa gọi provider API. Secrets/private payload snapshots nằm ngoài Git, không có trong report.

## Kết quả đã chạy

| Nhóm kiểm thử | Kết quả | Evidence thực sự |
| --- | --- | --- |
| Build API và simulator | PASS | Build lại bằng Dockerfile service |
| go vet và full Go race |130 passed test events,0 failed,0 skipped | Legacy domain7, Core60, persistence21, cohort1, integration41 |
| Postman lint |12 requests scanned,0 errors/warnings | Collection MS04 hiện có |
| Postman HTTP regression |21 executed requests,36 assertions,0 failed | Checkout, operation polling, order/payment simulator, ownership và unsigned callback |
| Worker unit tests |4 passed,0 failed | HMAC/raw bytes/replay/window/encryption/malformed/body limit/storage failure |
| Worker HTTPS probe |7 checks PASS | Health, signed capture/replay, wrong/stale auth, POST only, no public receipt reads |
| Actual SePay API |10 HTTP checks PASS | Account list/detail, transaction list/2details/filter/pagination, absent UUID404 và missing token401 |
| Actual SePay sender evidence |2 non-probe receipts matched2 API transactions | Stored signature/hash/AESGCM valid; reference/receiver/amount/direction/content/bank time khớp, webhook_success=1 |
| Kong runtime |4 checks PASS | Authenticated cart proxied, spoofed identity401, unsigned legacy callback401; target SePay local route404 được xác nhận là chưa active |
| Runtime contracts |3 response schemas PASS | Order, CheckoutOperation, Cart theo sandbox3.1 |
| Actual legacy outbox |15 payloads/5 event types PASS | Audit, cancelled, fulfillment_ready, paid, placed; schema definitions8 |
| Static target contracts | PASS |3 specs,8 v3 schemas,9 event examples,32 invalid event cases rejected;4 valid/2 invalid OpenAPI examples |
| Protobuf | PASS | Compile12 files including imports; regenerate7 Go files không drift; không có baseline descriptor comparison |
| Format/docs | PASS | gofmt gate,22 docs/275 local links, Kong YAML |

Go test event counts bao gồm subtests và fuzz seeds, không phải130 ca độc lập. Không chạy load test, fuzz campaign hoặc production failover trong lượt này. Runtime event count phản ánh các fixtures đã phát trong lần kiểm tra, không khẳng định mọi event schema đều đã phát thực tế.

## Vấn đề phát hiện và xử lý

1. Runtime contract checker lấy schemas target3.2 để validate sandbox3.1, báo thiếu items_digest. Sửa selector, ghi cohort vào kết quả; target required fields vẫn giữ nguyên. Thêm explicit Compose project và output directory, tránh chạm stack/evidence khác. [ADR008](../../02_architecture/adr/order-service/adr_008_verification_contract_bound_to_cohort.md).
2. Linux gofmt gate phát hiện line-ending/format residue ở3 legacy Go files. Đã normalize bằng gofmt; whitespace-insensitive content vẫn khớp HEAD, không đổi hành vi. Proto generated files không drift.
3. **Cả2 giao dịch SePay hiện chưa có mã OMP hợp lệ trong nội dung**, trường code rỗng. Không tìm thấy token OMP đúng12 ký tự để test auto-match Order. Đây là thiếu fixture cho happy path nhận diện, không phải lỗi webhook delivery. Cần mô phỏng thêm nội dung `OMP7K9M2N4P6R8T` rồi kiểm code; mã minh họa không đại diện Order có thật.

## Các bước chưa thể nghiệm thu

- Target checkout/worker pipeline và authenticated Payment receiver chưa nối public gateway. Cloudflare capture vẫn chỉ ACK notification, không claim inbox/P1 hoàn chỉnh.
- SePay API Access đã chứng minh đọc được dữ liệu; chưa được gọi từ Payment adapter trong transaction/order workflow.
- Chưa có một Order thật trong target cohort được gắn payment intent, nhận giao dịch SePay tương ứng, xác minh API và chuyển PAID. Không suy PAID từ webhook_success hoặc ACK200.
- Không nghiệm thu refund provider, live banking, rotation/retention/restore hoặc production activation từ kết quả này.

Đợt test này không tự động triển khai các phần roadmap còn thiếu. Bước triển khai tiếp theo cần hoàn thiện Worker/Inventory theo roadmap và Payment ingestion/verification; mỗi thay đổi runtime phải có ADR khi gặp gap.

## Chạy lại

```text
docker compose -p <isolated-project> -f services/order-service/docker-compose.test.yml run --rm -T tests sh -c 'go vet ./... && go test -race -json -count=1 -timeout 8m ./...'
postman collection lint "postman/collections/MS-04 Order Regression"
postman collection run "postman/collections/MS-04 Order Regression" --delay-request 100 --timeout 60000 --no-report-events
python services/order-service/scripts/verify-contracts.py --fresh-fixture --compose-project <isolated-project> --evidence-dir <output-dir>
python packages/contracts/verify_order_design.py --compile-proto --output <output-file>
node --test services/order-service/tools/sepay-sandbox/worker.test.mjs
node services/order-service/tools/sepay-sandbox/probe.mjs
node services/order-service/tools/sepay-sandbox/api-check.mjs <summary-file>
```

Postman/runtime checker yêu cầu API và simulator đã chạy trên ports theo compose. API detail/list kiểm trên tối đa100 giao dịch, detail kiểm2 records trong lượt này. Correlation dùng private D1 export và API snapshots cùng đợt; thao tác decrypt là offline, không disable timestamp window của public endpoint. Raw Go/Postman/build logs lưu ở temporary evidence directory của đợt kiểm thử; JSON summary trong repository đã loại secrets/PII.
