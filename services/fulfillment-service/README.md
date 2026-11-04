# MS-02 Fulfillment Service

Go1.22, PostgreSQL, REST8002/gRPC9002, transactional inbox/outbox, private S3-compatible media. Xem [luồng Order–Inventory–Fulfillment–Shipping, quyền và cách bật MVP](../../packages/contracts/mvp/fulfillment-integration.md) để tích hợp trên `dev`. [HLD/LLD](../../docs/02_architecture/services/fulfillment-service/README.md) và [evidence ban đầu](../../docs/04_testing/fulfillment-service/03_implementation_verification.md) ghi lại nền thiết kế; evidence cũ không thay thế lần chạy E2E hiện tại.

## Phần đã có code

- Service-owned hash-checked migrations;17 tables gồm16 bảng foundation và inbox-conflict ledger; cleanup retry lease bổ sung.
- Order guard khóa absence, accept/cancel terminal, high-watermark, one-ever-accepted task, durable command identities/original outcomes; exact source/package revisions.
- Staff REST: warehouse queue/cursor, assignment self/unassign/start, picks set-total, checklist, multi-source block guards, seal, history và command query. Same key/hash trả kết quả gốc; If-Match cho task mutation.
- JWT RS256 bằng trusted RSA public key, issuer/audience/time; profile tích hợp gọi Identity xác nhận ACTIVE EMPLOYEE và ADMIN GLOBAL/WAREHOUSE hiện hành. Internal gRPC dùng credential riêng và method ACL. Browser identity headers bị bỏ qua; thiếu public key thì protected APIs từ chối.
- Ready consumer v3 → durable inbox/job → permit request identity persisted trước dispatch; collision ledger không ghi đè original event. Outbox Kafka RequireAll/lease epoch/head ordering, stable payload/id/time/key, stale ACK không ghi đè lease mới.
- Media session/completion/verification worker, SHA256 thực, version-pinned object reads, ffprobe duration/container/video check, ClamAV INSTREAM, content-addressed final key, private access URL sau local audit. Seal cần VERIFIED evidence đúng task/package khi policy yêu cầu.
- SLA breach audit dedup theo task/deadline revision; orphan cleanup chỉ unlinked/expired/non-held objects, retry lease và exact deletion; liveness/readiness và outbox-age metric; graceful shutdown.

## Chạy local

```powershell
docker compose -f services/fulfillment-service/docker-compose.yml up --build -d
Invoke-RestMethod http://localhost:18002/health/ready
```

Compose này chỉ mở read/health skeleton với gates mặc định tắt; không seed task giả và không gọi tiền/carrier. PostgreSQL private, migration job chạy trước API, chỉ ports API bind localhost. Muốn gọi staff API, cung cấp issuer và RSA PEM public key được mount trong `local-keys/`, cấu hình `FULFILLMENT_JWT_PUBLIC_KEY=/keys/public.pem`. Private keys không đưa vào repo/image.

Profile tích hợp dùng `FULFILLMENT_STAFF_AUDIENCE=omama-api`, `user_id` UUID và `IDENTITY_URL`; không yêu cầu thêm permission/warehouse claim vào token hiện tại. Quyền staff lấy từ Identity `/api/v1/users/me`, Customer bị chặn kể cả có role ADMIN. Ba liên kết Order/Fulfillment/Inventory dùng token-file riêng, tối thiểu 32 byte. Legacy JWT claim profile chỉ dành cho kiểm thử nền khi không bật integrated-v2. Runtime pin public key qua file, rotation bằng config/deploy.

Xem [.env.example](.env.example) cho biến cấu hình khi chạy binary/Compose override. Base Compose không tự bật các biến peer/media từ shell. Để tích hợp, thêm override khai báo đúng env/endpoints/keys/brokers trong private network của team; không trỏ provider vào random service root.

## Capability profile

`FULFILLMENT_CANDIDATE_PEERS_ENABLED` bật typed gRPC adapters theo candidate packing contracts; cần Order permit/snapshot và Inventory allocation/validation providers thật. Missing/UNIMPLEMENTED/mismatched proof →fail closed, không fallback legacy hoặc simulator.

Admission/seal thực yêu cầu integrated-v2 + media prerequisites. `FULFILLMENT_DOMAIN_TEST_MODE=true` chỉ được dùng với `FULFILLMENT_ENV=local` trong môi trường isolated; không chứng minh E2E mua hàng. Unit/integration tests inject typed proof fixtures trực tiếp, không có HTTP endpoint nạp proof tùy ý.

`FULFILLMENT_MEDIA_UPLOAD_ENABLED` và `...ACCESS_ENABLED` điều khiển nhận yêu cầu mới. Recovery tiếp tục khi storage/scanner đã cấu hình dù upload flag tắt. Storage cần bucket versioning/private prefix ACL; upload key là staging, final key theo file/SHA256. Bài kiểm tra `scripts/test-web-identity.py --fulfillment` dùng RustFS versioning, MP4, ffprobe và engine ClamAV thực với database chữ ký fixture; production cần database chữ ký và chính sách media riêng.

Reassign sang nhân viên khác từ chối `MEMBERSHIP_PROVIDER_UNAVAILABLE` tới khi target membership authority có adapter. Customer/Care delegated video, legacy SubmitPackingEvidence, reopen, multi-package, paid cancellation/reversal vẫn disabled theo gates. Gỡ một block không tạo CANCELLED proof cho Inventory.

## Verification

```powershell
./services/fulfillment-service/scripts/test.ps1
```

Script build pinned toolchain, tạo PostgreSQL/Kafka trong Compose project riêng không publish port, chạy `go vet` và `go test -race`, kiểm actual HTTP/event payload bằng OpenAPI/JSON schemas, rồi dừng đúng project. Dependencies Python xem `docs/04_testing/fulfillment-service/requirements-design.txt`. Tests skip real infra nếu chạy Go trực tiếp mà thiếu test env; script trên cấp env để chúng thực sự chạy.

Regenerate protobuf (không sửa shared wire):

```powershell
docker run --rm -v "${PWD}:/workspace" omamx/fulfillment-go-tools:local sh scripts/generate-proto.sh
```

Snapshot shared proto pin `dev@a24492b` trong `contracts/SOURCE.json`; packing ports dùng bản canonical `packages/proto/packing/v1/packing.proto`, giữ namespace candidate để không đổi wire. Production mode yêu cầu TLS và JWT configuration; local cleartext chỉ cho private test network. E2E owning services có bài chạy riêng ở trên; backup/restore, HA/performance, production media policy và carrier/payment thật vẫn cần nghiệm thu riêng. Hai trường `integration` trong DTO staff hiện còn PENDING; xem trạng thái vận chuyển thực qua Order/Shipping tới khi bổ sung projection xác nhận cho màn staff.
