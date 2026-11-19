# Fulfillment implementation checkpoint — 30/10/2026

Branch: `feat/fulfillment-service`. [Runtime README](../../../services/fulfillment-service/README.md) · [Plan](01_acceptance_and_verification_plan.md). Đây là checkpoint code/test, không thay owning-provider acceptance của registry MVP.

## Phạm vi code

| Slice | Đã implement | Giới hạn |
| --- | --- | --- |
| S1 | Module Go,3 migrations có hash ledger,17 bảng schema Fulfillment, health/query | DB role hardening/restore acceptance production chưa nghiệm thu |
| S2 | Guard theo Order, accept/cancel/query, original command evidence, generation/tombstone/uniqueness, gRPC | Permit/snapshot/allocation providers thật vẫn pending |
| S3 | REST queue, assignment self/unassign/start, picks/checklist/blocks/seal/history/concurrency/auth | Assign-other fail closed vì employee membership authority chưa nối; chưa load benchmark |
| S4 | Upload session/verification job, private S3 version-pinned adapter, SHA256/ffprobe/ClamAV, access audit, orphan cleanup | Chưa nghiệm thu S3 bucket policy/versioning/real scanner deployment; case/customer delegation disabled |
| S5 | Package/seal + immutable revisions + history/audit/outbox atomic; accepted/packing-v2 events | Media integration test dùng metadata fixture; không phải video thật đã scan |
| S6 | Typed owning-peer gRPC adapters và candidate package-proof provider | Actual Order/Inventory/Shipping/Identity E2E chưa chạy; thiếu provider không fallback |
| S7 | Ready inbox/job/consumer, collision ledger, Kafka publisher lease/fencing, SLA dedup, cleanup retry, metrics/shutdown | HA/restore/fault matrix, full telemetry và production security acceptance pending |
| S8 | Unsupported capability flags/legacy/reopen bị chặn | Không bật extensions trước protocol acceptance |

## Kiểm tra đã chạy

```text
docker build -f services/fulfillment-service/Dockerfile.tools -t omamx/fulfillment-go-tools:local services/fulfillment-service
docker compose -p codex-fulfillment-test -f services/fulfillment-service/docker-compose.test.yml run --rm tests
python services/fulfillment-service/scripts/verify-runtime-contracts.py
docker build -t omamx/fulfillment-service:local services/fulfillment-service
```

Môi trường: Go1.22.12 Linux trong Docker, PostgreSQL16.15 và Kafka4.3.1 thật trên private Compose network, không publish port của DB/broker. Test suite dùng `go vet ./...`, `go test -race -count=1`; kết quả máy đọc ở [implementation-verification.json](implementation-verification.json). Payload thực từ HTTP và outbox/Kafka được lưu local trong `services/fulfillment-service/tests/results/` (ignored), contract checker không dùng ví dụ tĩnh để thay payload runtime.

**Kết quả lượt chạy cuối:**16 top-level tests,19 cases kể cả subtests PASS;0 failed,0 skipped test cases. Codegen9 files regenerate byte-identical. Docker image build PASS; chạy binary trong container biệt lập trả health ready với admission/seal mặc định tắt, sau đó dừng container. Actual payload contracts1 HTTP +3 events PASS. Các file Order đang sửa dở được đối chiếu SHA256 trước/sau và giữ nguyên.

Các nhóm được kiểm:12 vòng race accept/cancel, late accept/tombstone, generation mới sau cancel và cấm cancellation proof mới sau ever-accepted; replay/hash conflict; exact revisions; assignment/picks/checklist/multiple blocks; atomic seal rows; required media binding/retention hold; rejected media không có verified timestamp; JWT signature/audience/warehouse/body/header injection; typed gRPC ACL và UNIMPLEMENTED peer; inbox dedup/collision/quarantine; actual Kafka replay stable bytes/key và stale publisher ACK bị fence; SLA dedup không đổi task. ClamAV INSTREAM verdict tests dùng TCP protocol fixture (clean/malware/error), không phải scanner provider thật.

Actual HTTP Task và3 actual events (task.accepted candidate, packing.completed.v2, audit.record.v1) được validate bằng OpenAPI/JSON schema. Media fixture set trạng thái verified qua internal repository trong test để kiểm state/binding/event atomicity; không có production/public endpoint set VERIFIED hoặc inject caller proof.

## Các quyết định implementation bổ sung

- Shared proto không bị sửa trên nhánh hiện tại: service pin byte snapshot từ dev và candidate interfaces riêng, codegen pinned; owning providers phải review/promote candidate trước integrated rollout.
- Staff/internal JWT pin RSA public key qua file; dynamic JWKS rotation và real staff membership còn G07. Production listener bắt buộc TLS/JWT; Kafka/S3 ACL/SASL/TLS và deployment policy cần nghiệm thu riêng trước mở môi trường thật.
- Business precondition failure của staff mutation rollback và không lưu một successful command giả; admission APPLIED/REJECTED có durable registry. Sau success, replay giữ response version gốc; admission query/replay trả resource observation hiện tại kèm original command evidence.
- Cleanup chỉ xóa orphan unlinked và không hold sau expiry+grace; verified/attached evidence không bị orphan cleanup. Storage lifecycle của các staging versions cũ và legal-hold release cần policy vận hành được review.
- Default mutation/media flags tắt. Domain-test mode chỉ local isolated; missing capability không downgrade. Recovery queries/outbox giữ identities và obligations.

## Chưa chứng minh

Không coi retry cùng key là đã kiểm mọi network-level COMMIT ACK loss, PITR/HA hoặc toàn42 FUL-T cases. Chưa có full owning-service checkout→packing→shipping, real S3/scanner, staff provisioning, carrier, tiền thật, restore/load acceptance. Vì vậy chưa đóng G03–10/G12 hoặc tuyên bố Milestone B/C/D đạt; không bật production capability hoặc sửa shared registry acceptance chỉ từ suite xanh.
