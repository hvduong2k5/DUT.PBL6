# Traceability gRPC transport — 0.3.0-draft

RunID `traceability-grpc-20261127-local`. Amendment27/11/2026 thay transport nội bộ0.2.2, giữ nghiệp vụ và schema DB. [Report08](08_scoped_backend_2026-11-26.md) là checkpoint REST nội bộ lịch sử; dùng [gRPC guide](../../03_api_specs/traceability-workflow-grpc.md), proto và phiếu0.3.0 cho công việc hiện tại.

Đã fetch `origin/dev@457f8615`; runtime Traceability và các proto Catalog/Identity/Traceability baseline không có thay đổi so với nền worktree. Runtime được bàn giao bằng commit triển khai riêng trên cùng nhánh; tài liệu/interface là source snapshot riêng để ghim các phiếu owner. Chưa tạo PR, merge hoặc deploy trong lượt này. Không sửa worktree chính hoặc source implementation của owner khác.

## Thay đổi đã thực hiện

| Hướng gọi | Bản0.3.0 |
| --- | --- |
| Traceability → Inventory | Giữ BatchSource gRPC/TLS + candidate warehouse9 |
| Traceability → Catalog | CatalogTraceabilitySource.GetSkuBinding gRPC/TLS, không gọi REST Product–SKU/exact-SKU nữa |
| Traceability → Identity | StaffAuthorization.GetCurrentStaffGrants gRPC/TLS; credential service riêng + employee JWT metadata, local JWT verification và current-grant intersection |
| Order → Traceability | BatchLinkResolver.ResolveBatchLinks gRPC/TLS, bỏ internal REST proposal chưa phát hành |
| Web/BFF/admin/customer/public | Giữ REST/Origin/CSRF/no-store theo contract hiện có |
| OIDC / Kafka | OIDC discovery/JWKS HTTPS theo protocol; Kafka/outbox chỉ thông báo sau local commit, không đổi schema/topic |

Ba proto mới là additive shared integration targets trong `packages/proto/`; method/field cũ trong catalog.proto/identity.proto không thay đổi. Generate bốn service-local Go contracts từ3file mới và Inventory candidate, không hand-edit generated code. Fulfillment packing14 và Inventory warehouse9 vẫn là phần việc provider theo candidate đã chốt.

Identity client phân biệt STAFF_UNAUTHENTICATED/FORBIDDEN với SERVICE_AUTHENTICATION_FAILED/PERMISSION_DENIED bằng ErrorInfo; missing/unknown auth detail, RPC chưa implement, timeout, oversize/mismatched response đều fail closed503. Không fallback /users/me hoặc token stale. Catalog dùng cùng lookup khi bootstrap và verify Product/SKU, giữ version semantics, hidden/no-price; wrong service auth là dependency/config error.

Resolver nhận1–100bindings, request256KiB/response512KiB, budget1.5s. TLS, dedicated Order credential, requestId correlation, enum/presence, exact echo/order/duplicates; không peer query, không internal ID/evidence. Internal HTTP route cũ404. HTTP và gRPC có lifecycle chung: lỗi/startup/shutdown của listener dừng cả hai trước đóng DB/worker, forced Stop khi hết deadline.

## Phạm vi kiểm thử

PostgreSQL16 riêng tại loopback15433, schema/LOGIN tạm theo test; Catalog/Identity là provider mô phỏng nhưng sử dụng TCP, TLS certificate verification, protobuf framing và signed JWT thật bằng test keys. Resolver tests kết hợp workflow với PostgreSQL thật và TLS RPC; không gọi provider thật của nhóm. Source fixture sandbox chỉ dùng DB/mode tách biệt.

Các bài mới kiểm current-grant intersection/revoke, customer/inactive/subject mismatch, service-vs-staff auth errors, response limits, hidden/no-price binding shape, NOT_FOUND/UNIMPLEMENTED/deadline mapping, resolver100/101/caps/duplicates/null URL/old/corrupt/DB down và dual-listener drain. Provider-side Catalog interceptor và Identity IAM query thực tế chưa do các test fixture này xác nhận.

| Kiểm tra đã chạy | Kết quả | Giới hạn |
| --- | --- | --- |
| Go suite với PostgreSQL riêng, signed JWT và TLS gRPC fixtures |213test/subtest PASS,3Kafka SKIP;12package PASS,9package không có test; exit0 | Không là provider thật hoặc full E2E |
| go vet / go build | PASS exit0 | Không bật deployment |
| REST response ↔ OpenAPI |7response thực PASS | Resolver đã chuyển sang proto/TLS tests, không còn REST capture |
| Redocly |7file PASS exit0 | Warning baseline không đồng nghĩa stable promotion |
| Full MVP contract gate | PASS:47OpenAPI,52event schemas,17proto files/codegen,25registry entries,20negative guards | Static compatibility only; runtime acceptance vẫn PENDING |
| Codegen Go | Bốn contract khớp generator PASS | Không hand-edit generated code |
| Java proto compatibility |3proto mới generate PASS với protoc3.25.3 + grpc-java1.62.2 | Không phải full Java service build |

Native Gradle generateProto local chưa hoàn tất: Catalog offline thiếu Spring Boot plugin cache; Identity thiếu gradle-wrapper.jar. Đã kiểm bằng đúng compiler/plugin versions từ build.gradle theo artifacts checksum tại Maven Central, không sửa build/source của owner. Owner vẫn chạy build/test thật trong môi trường service của mình.

Kết quả lưu trong artifact RunID dưới `.sandbox/traceability-grpc-20261127-local/` (go-test.jsonl, summary.json và lint logs). Artifact không chứa credential/JWT/DSN. Các response REST capture bây giờ có7case; resolver dùng proto/TLS tests riêng, không kiểm lại internal REST đã bỏ.

## Cấu hình và chạy lại

Trong thư mục service, giữ env DB/runtime/public URL, OIDC issuer và worker hiện có. Thay URL business HTTP bằng:

| Env | Nội dung |
| --- | --- |
| TRACEABILITY_CATALOG_ADDR / CA_FILE / TOKEN | gRPC target, trusted server CA, dedicated Traceability credential |
| TRACEABILITY_IDENTITY_ADDR / CA_FILE / TOKEN | gRPC target, trusted server CA, dedicated credential khác Catalog/Inventory/Order |
| TRACEABILITY_INVENTORY_ADDR / CA_FILE / TOKEN | Giữ gRPC/TLS cũ, provider trả warehouse9 |
| GRPC_ADDR | Listener resolver, sandbox default127.0.0.1:9003; không dùng chung HTTP8003 |
| GRPC_TLS_CERT_FILE / KEY_FILE | Bắt buộc khi bật resolver ở integration; SAN khớp DNS service, key readable chỉ cho service UID |
| TRACEABILITY_SCOPED_API_ENABLED / CURSOR_HMAC_KEY | Scoped v2 flag và HMAC≥32bytes, mặc định false |
| TRACEABILITY_ORDER_RESOLVER_ENABLED / TOKEN | Resolver flag độc lập và credential Order riêng≥32bytes, mặc định false |

Bản integration mới yêu cầu cấu hình đủ3peer TLS/credentials ngay cả khi hai feature flags false; flags không là chế độ tương thích với staff provider cũ. Chuẩn bị RPC Catalog/Identity trước staff cutover; RPC thiếu sẽ fail closed503. Không hỗ trợ TRACEABILITY_CATALOG_URL/TRACEABILITY_IDENTITY_URL fallback. Integration luôn gRPC/TLS; sandbox loopback/isolated-container explicit mới có plaintext. Shared source credentials phải khác nhau. Traceability chỉ chuyển employee JWT vào x-staff-authorization tới Identity; không đưa vào request message/URL/log hoặc gửi sang Catalog/Inventory/Order resolver.

Migration000002/backfill giữ nguyên runbook report08: nâng mọi writer, bật require-warehouse-writes một chiều rồi mới scoped traffic; legacy metadata chỉ backfill bằng nguồn verified, giữ public hash/QR. Tuy nhiên cấu hình business peer trong runbook cũ phải dùng env/RPC mới ở bảng trên. Resolver read không phụ thuộc việc provider Catalog/Identity đã implement RPC vì không gọi chúng; process integration vẫn cần peer TLS config và OIDC discovery như wiring hiện hành.

Compose integration base mount3peer CAs và giữ resolver tắt. Muốn bật resolver có TLS dùng override (không thêm host/public port):

```powershell
docker compose -f compose.integration.yaml -f compose.resolver-tls.yaml up -d --build
```

Thiết lập các host CA/cert/key paths và .env trước; mount read-only/create_host_path=false. Hai PEM resolver cần quyền đọc cho UID10001, không copy private key vào image/repository. Cấu hình gRPC target/CA của Order khớp service DNS9003. Sandbox compose chỉ publish host loopback8003/9003; giữ flags false nếu chưa kiểm.

Tái lập local tests với TRACEABILITY_TEST_DATABASE_URL trỏ DB dùng riêng `om_traceability_db`, TRACEABILITY_TEST_ARTIFACT_DIR trỏ thư mục RunID:

```powershell
go test -json -count=1 -p 2 ./...
go vet ./...
go build ./...
python scripts/validate_scoped_runtime.py .sandbox/traceability-grpc-20261127-local
python scripts/generate_scoped_batch_client.py --check
```

Python spec/codegen requirements đã pin trong scripts; Go plugins v1.36.6/v1.5.1. Thiếu test DSN sẽ SKIP PostgreSQL và không là bằng chứng DB. Kafka tests vẫn cần broker riêng; nếu không chạy phải ghi SKIP, không suy từ outbox DB tests thành broker acceptance.

## Gửi lại cho owner

Phát lại6file Markdown0.3.0 cùng source SHA mới, ghi rõ thay phiếu0.2.2 về transport. Catalog bổ sung GetSkuBinding + dedicated method guard; Identity bổ sung current staff grants RPC + dual-credential/ErrorInfo, giữ BFF/assignments; Order thay resolver client và map enum/presence đúng REST DTO. Inventory/Fulfillment giữ field9/14; Web giữ REST, không viết gRPC browser client. Owner đã làm logic theo bản cũ tái sử dụng logic và thay/thêm adapter, không cần trao đổi xin ACCEPT trước code.

Provider thật, cookie BFF, Web QR decode/print/camera và Order/Fulfillment exact-proof E2E còn pending. Không mở production để thay nghiệm thu local hoặc tự promote stable. Review PR vào dev rồi trưởng nhóm đưa lên release theo quy trình đã thống nhất.
