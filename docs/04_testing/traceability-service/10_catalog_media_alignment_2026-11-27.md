# Đồng bộ Catalog và Media theo plan Dương — 0.4.0-draft

RunID: `traceability-catalog-media-20261127-local`, ngày 27/11/2026. Đã fetch và merge `dev@ac5395c5` (PR #175 của Dương) vào worktree riêng `codex/traceability-order-flow`; không sửa worktree chính, không sửa implementation Java/NestJS của owner. Theo chỉ đạo mới, làm cả Catalog và Media trong phần Traceability.

## Code và contract hiện hành

- Gỡ proposal CatalogTraceabilitySource/GetSkuBinding chưa phát hành; dùng canonical CatalogService.GetTraceabilityBinding(product_id, sku_code). Bootstrap SKU gọi GetProductVariant với channel UNSPECIFIED trước. Giữ nguyên version_fingerprint vào catalog_version; format khớp REST verifier cũ, không sửa frozen snapshot/hash/QR.
- CatalogAccess vẫn kiểm x-internal-token và CATALOG_INTERNAL_TOKEN. Traceability giữ TRACEABILITY_CATALOG_TOKEN; không bỏ credential theo ví dụ cũ trong plan. Baseline port9005 thay ví dụ9090; không yêu cầu port9105/Bearer RPC mới.
- Generate bảy service-local proto files từ Catalog/Media/common canonical, Identity/resolver target và Inventory warehouse candidate. Không sửa canonical Catalog/Media method/field hoặc generated code bằng tay.
- Media flag bổ sung ValidateMediaReference trước submit/publish cho OCOP evidence document:<Media UUID>. READY/purpose/MIME/size/SHA-256 bắt buộc; không thêm binary/object key vào REST Traceability hoặc rewrite certificate schema.
- Migration000003 thêm media_bind_jobs. Publish atomically ghi public pointer/audit/outbox và binding obligations; lỗi enqueue rollback toàn bộ. Worker sau commit dùng lease15s, RPC2s, retry5s, giữ job qua restart; completed job immutable, publish replay không tạo hai job. Media bind outage không đảo publication đã commit.
- TLS là mặc định. Để kết nối provider dev hiện plaintext, Catalog/Media có explicit *_GRPC_PLAINTEXT chỉ cho loopback/private IP/DNS service tương ứng; không fallback khi TLS lỗi. Inventory/Identity/resolver giữ TLS. Media baseline chưa có auth interceptor: không coi token không được provider kiểm là bảo mật; chỉ expose trong mạng tin cậy.
- Identity current-grants RPC và Order resolver không có counterpart trong plan Dương; giữ là phần việc riêng đã giao. Không suy PR #175 hoàn tất các kết nối đó.

## Kiểm thử đã chạy

| Kiểm tra | Kết quả / giới hạn |
| --- | --- |
| Go suite với PostgreSQL16 riêng, signed JWT và TCP/TLS peer fixtures | **235 test/subtest PASS, 3 Kafka SKIP, 0 FAIL** |
| Ca mới Media | READY/invalid purpose/status/MIME/checksum/size; immutable fileId; RPC errors; HTTP state giữ nguyên khi Media down; publish/queue atomic rollback, replay, retries, exclusive lease/completed-job protection |
| Transport | Canonical Catalog bootstrap/verify/error/caps; private explicit plaintext policy, cấm public target/Identity/Inventory plaintext và cấm fallback tự động |
| go vet / go build | PASS |
| REST actual response ↔ OpenAPI | 7 captures PASS; Media HTTP guard kiểm state/status riêng |
| Redocly | 7 source specifications PASS exit0 |
| Generated Go | 7 proto files khớp generator PASS |
| MVP contract gate | PASS:46 OpenAPI,52 event schemas,17 proto/codegen,25 registry entries,20 negative guards; baseline exceptions do PR #175 đã ghi, không tự nới thêm |
| Docker tooling | Xem checkpoint đóng gói ở cuối báo cáo; tách với runtime provider acceptance |

Ba ca Kafka SKIP là TestKafkaRelayUnavailableThenAck, TestKafkaCrashWindowReplaysSameEvent, TestKafkaSameProfileOrder. Artifacts đã lọc trong `.sandbox/traceability-catalog-media-20261127-local/`; không chứa DSN/token. PostgreSQL test riêng loopback15433, không dùng database nhóm. Catalog/Media fixtures dùng đúng canonical wire nhưng không là provider Java/NestJS thật hoặc full E2E. Chưa kiểm UI upload/print/camera, provider thật, IAM/BFF hoặc Order exact-proof integration trong lượt này.

## Cách bật và rollback

1. Dừng/upgrade mọi writer theo runbook scope trước. Dùng migrator trên DB đúng mode để `go run ./cmd/migrate -runtime-role <existing restricted role>`; migration000003 bắt buộc cho binary mới ngay cả khi Media flag false. Không sửa checksum000001/000002; cấp lại runtime grants cho media_bind_jobs, API không dùng schema owner.
2. Cấu hình Catalog GRPC_ADDR=địa chỉ thực9005, token khớp Catalog, CA hoặc explicit private plaintext. Media GRPC_ADDR=địa chỉ thực9007 và CA hoặc explicit private plaintext. Không publish peer host/public ports. Các env đầy đủ trong .env.example và guide0.4.0.
3. Upload/complete OCOP file qua Media REST hiện có. Đối chiếu READY/checksum với provider thật rồi bật TRACEABILITY_MEDIA_ENABLED. Cờ mặc định false; không bịa reference cho legacy. Frozen reference không tương thích cần draft mới và independent review, không backfill snapshot cũ.
4. Theo dõi `SELECT count(*), min(created_at) FROM media_bind_jobs WHERE bound_at IS NULL;`. Job retry qua restart; cờ tắt dừng worker và giữ pending. Drain jobs trước rollback schema, không drop nghĩa vụ gắn file để làm sạch test. Replay sau peer ACK có thể gọi Bind lặp; provider phải idempotent.
5. OCOP_CERTIFICATE hiện PUBLIC_READ ngay từ upload theo plan/provider, không đợi QR publish. Chỉ upload bản scan được phép công khai; evidence permission không thu hồi object public. Không dùng PublishMedia vì hiện chuyển file sang bucket Catalog. File dùng chung nhiều lô phải vẫn được Media bảo vệ retention dù baseline chỉ có một bound_entity.

Không có PR mới/merge vào dev/release hoặc deploy production trong lượt này. Review trên nhánh riêng, trưởng nhóm duyệt release theo quy trình.

## Đóng gói theo rule dev mới

AGENTS.md từ PR #175 yêu cầu bỏ Compose riêng từng service và có Dockerfile/Dockerfile.coolify. Runner scripts/dev.ps1 chuyển sang Docker CLI, chỉ quản lý container codex.task=traceability-local, cổng loopback và volume riêng; không dừng/reset container test15433 hoặc stack khác. Dockerfile local nhận Linux artifact; Dockerfile.coolify build source; race tests dùng target test của Dockerfile. Retire Dockerfile.test/ba Compose files; báo cáo08/09 là runbook lịch sử, dùng hướng dẫn hiện hành trong README.

PowerShell runner đã parse PASS; Init có bước migrate/regrant khi dùng state cũ, giữ credential và data. Docker image/runtime smoke sẽ được ghi bằng kết quả thực sau kiểm đóng gói; không suy script parse thành stack chạy thật.
