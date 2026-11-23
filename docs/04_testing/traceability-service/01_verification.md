# Traceability — bằng chứng kiểm chứng

Ngày: 2026-10-17. Branch `feat/traceability-service`, thay đổi workspace chưa commit.
Checkpoint: P7 review candidate — signed workflow/public QR, relay/worker, Docker
smoke, Linux race, seed và runbook đã kiểm chứng; **chưa chốt completion audit R1;
R2 chưa đạt**.

## Môi trường và lệnh

- Host Windows, Go `go1.27.0 windows/amd64`; module khai báo Go 1.25.0.
- PostgreSQL 16 qua Compose riêng `services/traceability-service/compose.yaml`.
  DB `om_traceability_db`, localhost `15403`, schema test ngẫu nhiên mỗi test.
- Broker Kafka-compatible thật: `docker.redpanda.com/redpandadata/redpanda:v24.3.18`,
  container `traceability-service-kafka-1`, host loopback `19093`, internal `kafka:9092`.
  Không phải Kafka service/consumer của peer.
- `TRACEABILITY_TEST_DATABASE_URL` được đặt từ mật khẩu dev ở `.env` ignored;
  không ghi DSN/credential vào báo cáo. Test chạy với quyền migrator test,
  kiểm tra runtime bằng `SET ROLE` sang login hạn chế được tạo/xóa trong test.
- Mạng Compose dùng bridge riêng, cổng host chỉ bind `127.0.0.1`.
  Bỏ `internal: true` sau khi xác nhận cổng host không truy cập được trên Docker
  hiện tại; không thay đổi stack/database của peer.

Các lệnh đã chạy thành công từ thư mục service, với test DSN được đặt:

```powershell
go test -count=1 '-coverpkg=./...' '-coverprofile=coverage.out' ./...
go tool cover '-func=coverage.out'
go vet ./...
go build ./...
```

Lần chạy P6 mới nhất đặt cả `TRACEABILITY_TEST_DATABASE_URL` và
`TRACEABILITY_TEST_KAFKA_BROKERS=127.0.0.1:19093`; các Kafka tests không bị skip.

Kết quả mới nhất: tests/build/vet PASS. Coverage aggregate xuyên package **79,0%** với
`-coverpkg=./...`, tính cả application/adapter được gọi từ HTTP integration tests.
Lần đo cũ 69,3% dùng instrumentation mặc định theo package trên checkpoint cũ;
không so trực tiếp hai số như một mức tăng thuần do code/test.
Coverage hiện đạt ngưỡng cho code hiện có. Các checkpoint cũ là 79,6%/79,7% trước
khi thêm helper/lifecycle/strict-null checks; không dùng chúng thay kết quả mới.
Có bằng chứng broker delivery/expiry worker như bên dưới; peer vẫn chưa tích hợp.

## PostgreSQL tests đã PASS

| Symbol trong `migrations/migrate_test.go` | Bằng chứng |
| --- | --- |
| `TestMigrationRoundtrip` | Up/idempotent up/down/up trên schema trống; từ chối đổi mode |
| `TestMigrationChecksum` | Từ chối migration checksum sai |
| `TestBindingAndHistory` | Unique Batch binding, version tăng đúng, binding bất biến, một working revision, nội dung đã submit bất biến, contributor không bị xóa, self-approval bị chặn |
| `TestApprovalNeedsDecision` | Không Approved thiếu decision; decision/state commit cùng nhau; pointer publish nhất quán; không restore Suspended |
| `TestRollbackAndCAS` | Rollback version/audit; hai CAS đồng thời chỉ một cập nhật thành công |
| `TestRuntimeRole` | Runtime có DML cần thiết; không CREATE schema object/TRUNCATE/DELETE revision/sửa checksum/tắt trigger; migrator không được dùng làm runtime |
| `TestDownGuard` | Down bị từ chối ngoài schema test |
| `TestOutboxFailureRollsBackCommand` | Duplicate outbox làm rollback version/audit/idempotency; event payload bất biến; Published không reset thành Pending |

Đây là kiểm tra schema/transaction SQL; có thêm transaction runner và repository
tests bên dưới. Bảng migration dùng SET ROLE; kiểm tra LOGIN độc lập và HTTP Draft
được bổ sung ở checkpoint mới bên dưới. Không gọi phần này là tích hợp peer.

## Transaction runner và repository đã PASS

Nguồn: `internal/infrastructure/postgres/commands_test.go` và `profiles_test.go`,
chạy trên PostgreSQL thật bằng DSN test, không phải mock.

- `TestCommandsReplayAndIsolation`: cùng key/body replay đúng status/JSON,
  body khác bị conflict, namespace principal/action riêng.
- `TestCommandsConcurrentRetry`: 8 yêu cầu đồng thời, kể cả UUID khác hoa/thường,
  chỉ thực thi callback một lần và replay 7 lần.
- `TestCommandsRollback`: lỗi nghiệp vụ, outcome không hợp lệ và lỗi constraint
  lúc commit đều rollback profile/dedup; retry sau rollback thực thi được.
- `TestCommandsValidation`/`TestFingerprint`: kiểm tra metadata, fingerprint
  có action/resource và không phụ thuộc thứ tự map.
- `TestDraftCreateReadEditReplay`: tạo Draft thiếu nội dung, đọc snapshot nhất quán,
  sửa tăng version, thêm contributor, QR giữ nguyên, replay không ghi thêm audit.
- `TestDraftConcurrentEditsAndBinding`: hai edit cùng version chỉ một thành công;
  duplicate Batch binding và sửa revision đã submit đều bị chặn.
- `TestDraftQRCollisionAndAtomicAudit`: retry khi QR trùng có giới hạn; không tái
  gán Batch; lỗi audit rollback hồ sơ và dedup mới.
- `TestDraftInvalidAndNotFound`: validation và not-found ở repository.

Unit tests `internal/domain/model_test.go` đã PASS cho Draft/submission completeness,
certificate scope/manual/evidence/stars, ngày hết hạn inclusive tại UTC+7,
review_due đúng instant, revoked/future issue/verification, URL video allowlist,
redaction không sửa dữ liệu gốc, snapshot freshness và Batch depleted.
Các kiểm tra domain này chưa chứng minh workflow submit/public HTTP đã tồn tại.

Transaction runner không tự xác thực principal; application phải authorize trước
mỗi lệnh, kể cả replay. Repository đã được gắn endpoint Draft qua application.
Khóa theo idempotency
chỉ giữ trong local transaction, không bao network call tới peer.

## Identity, application và HTTP đã PASS

- `internal/infrastructure/identity/sandbox_test.go`: Ed25519 signature; fixed
  sandbox issuer/audience; exp/iat/nbf; lifetime <=15 phút; từ chối alg khác,
  key lạ, jku, role ADMIN, permission lạ/trùng, JSON claims trùng, token malformed;
  constructor không thể bật verifier sandbox trong integration/production.
- `internal/jsonutil/strict_test.go`: từ chối unknown fields, JSON trùng tên/case/
  escape alias, trailing JSON, non-object, invalid UTF-8, nesting quá sâu.
- `TestSignedDraftHTTPAndReplayPermissions`: JWT ký thật + PostgreSQL; create,
  read/edit, stale409, replay; mất quyền không replay được; redact evidence theo
  quyền hiện tại; header identity/request ID giả không được tin; DB giữ chứng cứ
  gốc. Replay đã commit vẫn hoạt động khi fixture provider unavailable; lệnh mới503.
- `TestDraftHTTPValidationAndSourceGuards`: malformed/oversized/unknown JSON,
  thiếu idempotency key400, SKU mismatch409, Batch missing404, source namespace
  chưa có adapter503; không fallback; token thiếu/sai401; key/body mismatch409.
- `TestMissingVerifierFailsClosed`: thiếu verifier401, health không yêu cầu JWT.
- `TestRuntimeLOGINAndSchemaReadiness`: LOGIN bằng role/password test riêng, đủ
  privilege policy thì PASS; migrator, mode khác và checksum sai bị chặn.
- `TestAssembledSandboxUsesRuntimeLOGIN`: composition server đọc public key/fixture,
  readiness200, signed create201 qua runtime LOGIN, protected GET qua HTTP TCP
  loopback200/no-store; checksum sai →readyz503 và từ chối startup; integration
  không dùng sandbox adapter. Không phải smoke signal/shutdown của binary main.

Tất cả credentials/key test được sinh lúc chạy và cleanup. Fixture là dữ liệu
tổng hợp trong namespace `fixture`, không phải Catalog/Inventory thật. Verifier
sandbox chưa có revocation/introspection; TTL tối đa 15 phút không thay contract X-04.

## Revision, review, publish và public đã PASS

Các tests mới chạy với DSN PostgreSQL thật, không skip:

- `TestSubmitFreezeIndependentApproval`: snapshot đóng băng, replay, self approve/
  reject bị chặn, quyết định độc lập, approve chưa public và không phát event publish.
- `TestCloneInheritsWithdrawnContributors`: thu hồi cần reason, union contributor
  của source/working/new creator, không giữ approval/snapshot cũ, QR ổn định.
- `TestConcurrentApproveRejectAndAtomicDecision`: hai quyết định đồng thời chỉ
  một thành công, một decision/audit/dedup tương ứng.
- `TestReviewAuditFailureAndExpiry`: audit failure rollback decision/root/status;
  certificate hết hạn không approve được nhưng vẫn reject được với lý do.
- `TestSubmitRejectsChangedOrStaleSource`: mismatch409/stale503, trạng thái không đổi.
- `TestSignedHTTPReviewAndRevisionHistory`: hai JWT, freeze/self-approval/reason,
  no-public-before-publish, decision replay/permission loss, withdrawal/clone/reject.
- `TestSubmitNetworkVerificationOutsideTransactionAndCAS`: provider callback
  thực hiện một edit khác thành công, chứng minh không giữ aggregate lock qua
  network; kết quả peer về sau bị chặn bởi CAS khi version đã đổi.
- `TestPublishSourceHashAndDurableEvent`: nguồn đổi chặn publish, checked_at mới
  không tự invalid hash, pending event đúng key/topic/version và không lộ evidence.
- `TestPublishOutboxFailureRollsBackEverything`: lỗi INSERT outbox rollback cả
  pointer/status/version/audit/dedup; public vẫn404.
- `TestConcurrentPublishAndImmutablePublicHistory`: một publish thắng, Draft mới
  không sửa snapshot public, publish revision tiếp theo giữ nội dung cũ/QR; suspend/
  archive và chặn clone vượt suspension.
- `TestPublicEligibilityWithoutWorker`: certificate valid_until inclusive UTC+7,
  hết hạn đúng cutoff →410 dù không có worker, không tự chuyển root sang Suspended.
- `TestSignedEndToEndPublishQRAndSuspension`: create→submit→approve→publish→QR→
  clone/edit→submit/approve/publish lại→suspend/archive qua HTTP ký thật; cùng QR,
  no-store, nullable fields, no internal evidence, 4 business events còn PENDING.
- `TestPublishRechecksSourceAndCanReplayWhileUnavailable`: publish chưa approved409,
  source correction409, provider unavailable503, không fake fallback; committed
  replay và public read hoạt động khi provider không sẵn sàng.
- `TestProfilePagination`: cursor theo created_at/id, page bounds, không trùng item.
  HTTP list có redaction và từ chối invalid/duplicate/unknown query parameters.
- Domain public tests: depleted không bị chặn, Batch restricted/expired, revoked,
  never-public404, metadata hỏng không được phục vụ; SourceHash chỉ bỏ checked_at.

Draft event schema nằm ở `services/traceability-service/contracts/events/`.
Đã validate payload thực tế từ broker như phần P6; chưa chứng minh peer compatibility.
Event giữ traceparent của request; HTTP tạo child span khi nhận parent hợp lệ và
tạo trace mới nếu parent không hợp lệ. Business event retry giữ payload/trace gốc.

## Relay, worker và metrics đã PASS

- `TestOutboxLeaseRetryAndProfileOrdering`: không vượt predecessor đang leased;
  lease hết hạn được reclaim với token mới, owner cũ không Ack; backoff; max attempts
  →FAILED; successor không vượt FAILED head; pending/failed/age stats.
- `TestOutboxConcurrentClaimAndExhaustedCrash`: 8 claim đồng thời chỉ một lease;
  crash ở attempt cuối chuyển FAILED khi lease hết hạn, không treo vô hình ở PENDING.
- `TestKafkaRelayUnavailableThenAck`: publisher kết nối tới port không phục vụ broker
  thất bại thật, giữ PENDING và public read vẫn hoạt động; retry tới broker đang chạy
  được Ack, DB mới PUBLISHED, consumer đọc được event với đúng key/version.
- `TestKafkaCrashWindowReplaysSameEvent`: broker publish đã Ack nhưng cố ý bỏ local
  Ack, cho lease hết hạn trong schema test, reclaim/gửi lại và consume được hai bản
  cùng event ID/payload/version. Đây là mô phỏng crash tại boundary, không phải kill
  OS process; chứng minh at-least-once, không tuyên bố exactly-once.
- `TestKafkaSameProfileOrder`: consume publish(version4) trước suspend(version5).
- `TestExpiryWorkerIdempotentAndDoesNotSuspend`: hai sweep đồng thời + nhiều tick
  lặp chỉ một observation/audit/event; root ACTIVE, public vẫn410, tạo revision mới được.
- `TestExpiryObservationRollsBackOnOutboxFailure`: observation/version rollback nếu
  ghi expiry event lỗi; public guard không dựa vào worker.
- `TestRuntimeLOGINCanPublishAndObserveExpiry`: runtime LOGIN hạn chế thực hiện
  create/submit/independent approve/publish/sweep, không chỉ SET ROLE.
- `TestPublicHTTPReviewDueAndBatchExpiryWithoutWorker`: anonymous public200 trước
  instant cutoff và410 đúng cutoff cho no-expiry review_due và Batch expiry.
- `TestWorkerCapabilitiesCannotSilentlyFallback`: flag sai/missing enabled broker
  bị chặn; relay off/default và worker off explicit có semantics rõ.
- `TestMetricsAndSanitizedFailure`: counters/backlog/duration; lỗi DB trả503 không
  lộ credential. Wiring test kiểm tra metric relay-disabled và không có raw profile ID.
- HTTP publish test xác nhận event traceparent bằng response child trace, giữ trace
  ID/flags của parent hợp lệ, không lấy user/role từ header.

Payload Kafka thực tế được lưu tại [artifacts/published.event.json](./artifacts/published.event.json)
(dữ liệu tổng hợp, profile test đã cleanup), validate PASS bằng Python
`jsonschema.Draft7Validator` + `FormatChecker` với service-local draft schema.
Sample đó là payload broker thật. P7 bổ sung full OpenAPI validation, bốn synthetic
event variants, privacy rejection và runtime admin-response validation; không gọi
synthetic variants là broker/peer roundtrip.

## P7 — kết quả bàn giao đã chạy

- `scripts/dev.ps1 -Action Init`: tạo `.sandbox`, migration rõ ràng, runtime LOGIN
  riêng, khóa và source synthetic. Chạy lại giữ nguyên role/password/key/data.
- `TestSeedReuseAndSmokeWithRuntimeLOGIN`: seed reuse, signed loopback HTTP với
  runtime LOGIN; full flow PASS, rerun terminal fixture không reset/restore.
- Container API build PASS (Go1.25 Alpine, CGO0, runtime UID10001). Lần đầu tải module
  bị unexpected EOF; đã tách/cache module step và build lại thành công.
- Container startup từng fail closed vì DB container cũ mất DNS alias `db`; đã
  recreate **chỉ container db của Traceability**, giữ volume. Compose API có
  readiness healthcheck; root cause không bị che bằng fixture/fake DB fallback.
- `cmd/sandbox -action smoke` trên container thật: **full_flow_passed**, profile
  `aefe5c9f-1e0f-4b13-ac30-4a15a2fb5978`. Rerun trả
  `existing_terminal_fixture_verified`; QR/history được giữ, không giả full run mới.
- `Start -Relay`: broker thật hoạt động, root outbox đã drain; health ready,
  pending0/failed0. Counter delivered reset theo process sau restart, không dùng
  counter0 để suy ra DB chưa gửi event.
- SIGTERM qua `Stop`: API exit0, đo được **0,56 giây**; restart healthy. Không đụng
  DB/broker/volume peer. Unit tests `TestShutdownDrainsAndClosesResourcesOnce`,
  `TestCleanupHangHasBoundedFailure`, `TestInFlightRequestIsCancelledOnDrainDeadline`
  chứng minh một deadline chung và timeout không bị báo thành graceful success.
- `scripts/dev.ps1 -Action Race`: **PASS** với Go1.25 Linux/GCC, PostgreSQL và broker
  thật; không có DATA RACE. Test/Race logs local ở ignored `.sandbox/final-*.log`.
- `scripts/dev.ps1 -Action Test`: tất cả tests/vet/build PASS, coverage79,1%; includes
  dependency timeout503/no-write, configured video host và `qr_url` presentation.
- Full OpenAPI3.0.3 validation PASS bằng openapi-spec-validator0.7.2; đã sửa YAML
  flow-description có dấu phẩy bị parse thành key phụ. Spec có cả `/metrics`.
- `scripts/validate_specs.py --runtime`: complete document + actual broker sample +
  four synthetic event variants + internal-evidence rejection + synthetic create
  request + actual redacted admin response đều PASS.
- Final internal fix: clone audit ghi source_revision và withdrawn_revisions;
  `TestCloneInheritsWithdrawnContributors` kiểm tra lineage ở audit append-only.
  Test/build/vet và Linux Race đã chạy lại PASS sau sửa đổi; coverage cuối79,0%.
- Strict decoder hiện từ chối null của primitive/struct/collection non-nullable;
  pointer nullable vẫn được chấp nhận. Typed Content marshals empty collections
  thành[] để body của Go client phù hợp schema. Publisher cũng từ chối unknown
  fields thay vì âm thầm gửi raw private fields ra broker.
- Đã chuẩn bị workflow CI riêng, có trigger `dev` (workflow cũ chỉ main/develop),
  race/integration/coverage>=75 và contract checks. **CI trên GitHub NOT_RUN** vì
  chưa commit/push/PR; không trình bày local PASS thành GitHub check PASS.

Runbook: [02_reviewer_runbook.md](./02_reviewer_runbook.md).

## Gate còn thiếu

- P0..P7: final audit và image check hoàn tất trong phạm vi R1; xem
  [đối chiếu theo matrix](./04_final_r1_audit.md). R2 chưa đạt.
- PR description/checklist đã chuẩn bị ở [03_review_readiness_and_pr_draft.md](./03_review_readiness_and_pr_draft.md).
- GitHub CI chưa chạy; chưa tạo/push PR theo phạm vi hiện tại.
- P8: peer adapter/roundtrip còn phụ thuộc contract X; NOT_RUN.

Chưa xác minh EXT-10, chưa production, chưa push/PR. Audit local không chứng minh
Audit Service ingest; contract producer hash còn cần owner xác nhận (X-09).

## Kiểm tra trước khi push theo yêu cầu — 2026-10-24

- `go test ./...`, `go vet ./...`, `go build ./...`: PASS.
  Lượt này không cấu hình DB/broker test nên các integration tests tương ứng skip;
  không coi kết quả này là chạy lại toàn bộ integration hoặc coverage79,0%.
- Python unittest transport: 2 tests PASS, gồm chặn 301/302/303/307/308 để không
  chuyển bearer token. `scripts/validate_specs.py`: OpenAPI/event schema PASS.
- Docker Engine không có pipe `dockerDesktopLinuxEngine`; thử khởi động Docker
  Desktop chưa làm engine sẵn sàng. `dev.ps1 -Action Test` dừng trước test ở bước
  Docker; chưa chạy lại DB/Kafka/race hoặc HTTP runtime ở lượt push này.
- Kết quả integration/race/coverage ở các phần trước là bằng chứng ngày 2026-10-17.
  Workflow GitHub sẽ kiểm tra lại trên runner sau khi push/PR; chưa suy ra PASS.
- User đã yêu cầu push code và liên kết issue `traceability-service` #21.
  Các ghi chú chưa commit/push trong checkpoint cũ mô tả thời điểm lập báo cáo.
