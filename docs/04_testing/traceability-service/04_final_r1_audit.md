# Đối chiếu hoàn tất backend R1

Ngày 2026-10-17. Nhánh local `feat/traceability-service`, nền `dev@2291ef5`.
Kết luận: **R1 sẵn sàng cho review trong phạm vi sandbox Batch backend**.
R2 chưa đạt; chưa commit/push/PR, GitHub CI chưa chạy. Review của người khác và
production readiness không được suy ra từ kết luận này.

## 1. Cách đối chiếu

Đọc lại đầu ra P0..P7 và ma trận test; đối chiếu symbol test với source, báo cáo
và logs local `.sandbox/final-test.log`, `final-race.log`, `final-container.log`.
Test/vet/build PASS, coverage cuối **79,0%**; Linux Go1.25 race PASS với PostgreSQL
và broker thật. Không chạy lại toàn suite chỉ vì thay tài liệu. Logs local bị
ignore; hướng dẫn tái hiện nằm trong [runbook](./02_reviewer_runbook.md).

Ở lượt chốt, Compose báo cả API/DB/broker healthy; refresh token và chạy lại
`scripts/validate_specs.py --runtime` PASS: toàn bộ OpenAPI, event sample từ broker,
bốn synthetic variants, privacy rejection, synthetic create request và response
admin thực tế đã redaction. Chưa validate response public200 thực tế bằng script
này vì fixture demo đang Suspended; public200 có HTTP E2E tests riêng.

## 2. Ma trận hiện hành thay các checkpoint lịch sử

PASS bên dưới chỉ áp dụng guard trong phạm vi R1; không có nghĩa toàn FR của Epic
hoặc adapter peer đã PASS. Symbol rút gọn bỏ tiền tố `Test`.

| Matrix ID | Kết quả R1 | Bằng chứng chính |
| --- | --- | --- |
| CFG-01/02 | PASS | LoadBoundaries, SandboxCannotBecomeIntegration, RuntimeLOGINAndSchemaReadiness |
| OPS-01 | PASS | ReadinessAndLiveness, AssembledSandboxUsesRuntimeLOGIN |
| DB-01 | PASS | BindingAndHistory, DraftConcurrentEditsAndBinding, DraftQRCollisionAndAtomicAudit |
| DB-02 | PASS | BindingAndHistory, ApprovalNeedsDecision, ConcurrentPublishAndImmutablePublicHistory |
| DB-03 | PASS | OutboxFailureRollsBackCommand, PublishOutboxFailureRollsBackEverything |
| IDEM-01 | PASS | CommandsReplayAndIsolation, CommandsConcurrentRetry, SignedDraftHTTPAndReplayPermissions, PublishRechecksSourceAndCanReplayWhileUnavailable |
| AUTH-01/02 | PASS sandbox | SandboxIdentity, MissingVerifierFailsClosed, SignedDraftHTTPAndReplayPermissions, SignedHTTPReviewAndRevisionHistory |
| DRAFT-01 | PASS | DraftAndSubmission, DraftHTTPValidationAndSourceGuards, DraftConcurrentEditsAndBinding |
| DEP-01 | PASS fixture | DraftHTTPValidationAndSourceGuards, DependencyTimeoutIs503WithoutSubmission, SubmitRejectsChangedOrStaleSource |
| REVIEW-01 | PASS | IncompleteDraftCannotSubmit, SubmitFreezeIndependentApproval |
| REVIEW-02 | PASS | CloneInheritsWithdrawnContributors, SignedHTTPReviewAndRevisionHistory |
| REVIEW-03 | PASS | ConcurrentApproveRejectAndAtomicDecision, ReviewAuditFailureAndExpiry |
| PUB-01 | PASS | SignedEndToEndPublishQRAndSuspension, ConcurrentPublishAndImmutablePublicHistory |
| PUB-02 | PASS fixture | SubmitNetworkVerificationOutsideTransactionAndCAS, PublishSourceHashAndDurableEvent, PublishRechecksSourceAndCanReplayWhileUnavailable |
| QR-01/02 | PASS | DraftQRCollisionAndAtomicAudit, SignedEndToEndPublishQRAndSuspension, PublicEligibilityWithoutWorker |
| CERT-01/02 | PASS manual path | CertificateEligibilityBoundaries, DraftAndSubmission, PublicHTTPReviewDueAndBatchExpiryWithoutWorker |
| BATCH-01 | PASS frozen snapshot | PublicEligibilityAndNoSyntheticDefaults, PublicHTTPReviewDueAndBatchExpiryWithoutWorker |
| PRIV-01 | PASS | VideoAllowlistAndRedaction, SignedEndToEndPublishQRAndSuspension, DistributedEventRejectsPrivateOrContradictoryPayload |
| HIST-01 | PASS local snapshot | ConcurrentPublishAndImmutablePublicHistory, PublishRechecksSourceAndCanReplayWhileUnavailable |
| EVT-01/02 | PASS isolated broker | KafkaRelayUnavailableThenAck, KafkaCrashWindowReplaysSameEvent, KafkaSameProfileOrder, OutboxLeaseRetryAndProfileOrdering |
| EXP-01 | PASS | ExpiryWorkerIdempotentAndDoesNotSuspend, ExpiryObservationRollsBackOnOutboxFailure, PublicHTTPReviewDueAndBatchExpiryWithoutWorker |
| OPS-02 | PASS sandbox | ShutdownDrainsAndClosesResourcesOnce, CleanupHangHasBoundedFailure, InFlightRequestIsCancelledOnDrainDeadline, MetricsAndSanitizedFailure; Docker SIGTERM exit0 |
| E2E-01 | PASS sandbox | SignedEndToEndPublishQRAndSuspension, SeedReuseAndSmokeWithRuntimeLOGIN; container full-flow smoke và terminal rerun |
| PEER-01/02 | NOT_RUN | Chưa có adapter/roundtrip thật; xem X-01..X-09 |

Broker unavailable test dùng port không phục vụ broker, không dừng broker của
peer. Crash-window test bỏ local Ack và cho lease hết hạn, không kill OS process.
History PASS nói về snapshot đã lưu; không chứng minh phát hiện correction từ
Inventory khi chưa có contract/event. Coverage không thay các giới hạn này.

## 3. Đối chiếu mốc và giới hạn

- P0: domain/state/permission/transaction/schema/OpenAPI có tài liệu và source;
  không còn luật nội bộ bắt buộc để chạy R1 đang bỏ trống. Các contract ngoài là
  đề nghị của Traceability, chưa phải thỏa thuận với owner.
- P1..P6: runtime, constraints, workflow, public guard, relay/worker đã có evidence
  ở bảng trên. Image cuối đã build và chạy healthy sau sửa clone audit/strict decoder.
- P7: runbook, reusable init/smoke, Test/Race, schema checks và nội dung PR đã có.
  Script đã sửa restore biến môi trường vắng mặt bằng Remove-Item, tránh để chuỗi
  rỗng override `.env`; Tokens và Compose ps kiểm tra lại thành công.
- P8: giữ WAITING_FOR_CONTRACT. Implementation hiện từ chối toàn bộ integration/
  production ngay startup; thiết kế adapter integration trong P0 là mục tiêu P8,
  chưa phải capability đang mở có thể chạy bằng fixture.

FR-10 xác minh ngoài, FR-15 genealogy, Product/SKU profile độc lập, restore,
advanced source management và các capability đã hoãn vẫn PARTIAL/DEFERRED.
Không sửa peer source, schema chung hoặc tạo producer hash giả cho Audit.

## 4. Remote snapshot trước P8

GitHub CLI `gh api repos/hvduong2k5/DUT.PBL6/branches --paginate` kiểm tra lại
ngày trên: tất cả tám implementation refs trong file quyết định giữ nguyên SHA.
Nhánh `catalog-service@4587b763` vẫn khác `feature/catalog-service`;
`copilot/trin-khai-inventory-service@e23a9199` cũng hiện diện, không tự coi là
contract provider thay nhánh Inventory đã phân tích.

Đã đọc bổ sung tree của `copilot/trin-khai-inventory-service@e23a9199`:
`services/inventory-service/src/.gitkeep` là source duy nhất của service;
OpenAPI trong nhánh này mô tả stock/reservation, không có Batch lookup/UUID/QR.
Không có `packages/proto/inventory/v1/inventory.proto` ở ref đó. Nhánh này chưa
cung cấp lời giải implementation cho X-01/X-02, không thay thế provider thật.

Trong danh sách remote truy cập được chưa có nhánh implementation riêng tên
Identity, Fulfillment, Procurement, Audit hoặc Traceability. Không kết luận về
nhánh local chưa push, repository riêng hay quyền truy cập ngoài repo hiện tại.
Vì các refs đã đọc không đổi, không có bằng chứng mới để đóng X-01..X-06/X-09.
X-07/X-08 vẫn backlog đã hoãn. Chưa phát sinh peer runtime verification.

## 5. Bàn giao

Reviewer có thể dùng [runbook](./02_reviewer_runbook.md) và
[nội dung PR dự kiến](./03_review_readiness_and_pr_draft.md). Tạo/push PR là bước
bàn giao riêng theo yêu cầu; không được đổi GitHub CI NOT_RUN thành PASS local.
Mục tiêu toàn hệ thống còn P8; hoàn tất R1 không đóng thay mục tiêu đó.

## 6. Sửa bổ sung công cụ reviewer

Script Python kiểm tra runtime trước đây dùng urllib mặc định theo redirect,
có thể đưa bearer token tới endpoint khác nếu API local trả Location. Đã chặn
tất cả redirect, giống policy smoke Go. Hai tests HTTP local PASS: endpoint gốc
nhận request đúng; 301/302/303/307/308 bị từ chối và receiver không nhận request
hay bearer token. Không dùng credential thật trong tests này.

Sau sửa, token refresh và `validate_specs.py --runtime` PASS với API đang chạy.
Đã thêm test transport vào runbook/CI draft và ignore Python cache. Không đổi
Go source/image, nên kết quả Go tests/race trước đó giữ nguyên phạm vi bằng chứng;
CI remote vẫn NOT_RUN. `gofmt -l cmd internal migrations` không trả file cần sửa.
