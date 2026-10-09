# MS-04 — báo cáo implementation và kiểm chứng 3.1

Ngày chạy: **09/10/2026**, workspace chưa commit. [Thiết kế hiện hành](../../02_architecture/services/order-service/08_implementation_and_acceptance.md), [cách chạy](../../../services/order-service/README.md), [runbook](../../06_operations/order_service_runbook.md). Không có commit release hoặc deployment production được tạo trong lần bàn giao này.

## 1. Kết quả đã chạy

| Gate | Kết quả thực tế | Evidence |
| --- | --- | --- |
| Build hai binary/container | PASS; Go 1.22, runtime non-root; API và simulator đã rebuild từ code cuối | Dockerfile, Compose; manifest trong `environment-evidence.json` |
| Formatting, `go vet`, unit + race integration | PASS; 2 domain tests (5 money subcases), **41 top-level integration tests**, không race report; integration 37.906s | [log nguyên bản](go-test-race-output.txt) |
| Protobuf regeneration | PASS; hash generated files trước/sau giống nhau; registered HTTP/gRPC tested | `contracts-verification.json`; test InventoryTerminalRPCs và GRPCRegisteredServerAndOwnership |
| Postman CLI 1.70.0 | PASS; 12 request definitions; lượt cuối 16 requests do polling, **31 assertions**, 0 failed; collection lint 0 errors/warnings | [CLI output](postman-cli-output.txt), [exit/target](postman-run-result.json) |
| Runtime schema smoke | PASS; operation không cần public Order, unique registry, PAID/COD guards, immutable snapshots; transaction rollback | [SQL](schema_smoke_checks.sql), [output](schema-smoke-output.txt) |
| OpenAPI / event schemas / docs links | Kết quả và số lượng cụ thể trong artifact, chỉ kiểm local | [artifact](contracts-verification.json) |
| Baseline HTTP | 20 fresh keys, concurrency 5, một SKU: acceptance p95 **30.26 ms**; client-observed completion p95 **1493.34 ms** | [raw measurements](latency-baseline.json) |

Postman polling làm số requests/assertions thay đổi giữa các lượt. Không dùng số lượng assertions để suy ra số business cases. Baseline chỉ là workload nhỏ trên máy phát triển; completion có polling delay. Không chứng minh availability 99.9%, HA hay SLO production.

## 2. Môi trường và phương pháp

PostgreSQL 16, Redis 7 và Kafka chạy **thật** bằng Compose project `order-sandbox`; provider/dependencies là **simulator có trạng thái**. Database `order_runtime` và `order_simulator` độc lập. Tests tạo schemas riêng, Redis DB1, Kafka consumer groups riêng và source topic riêng cho test consumer; API dùng DB0. Fixtures synthetic, không có giao dịch tiền thật. Kafka có volume bền vững và topic initialization; host ports chỉ bind loopback.

Harness khởi tạo service mới để kiểm restart/recovery, dùng SQL lease expiry/epoch và barriers để kiểm cạnh tranh. Mất reserve/finalize/refund ACK được tạo sau khi ledger commit. Quan sát cả Order DB và simulator ledger, không chỉ HTTP status. Outbox test publish Kafka thật rồi reset checkpoint để mô phỏng crash giữa broker ACK và SQL published; source consumer thật kiểm duplicate delivery và durable outcome.

Reproduce full gate:

```powershell
./services/order-service/scripts/test.ps1 -StartStack
python services/order-service/scripts/verify-contracts.py
```

Python contract checker cần dependencies trong `services/order-service/scripts/requirements-verification.txt` (PyYAML, jsonschema, openapi-spec-validator). `test.ps1` cần Docker và Postman CLI, dùng Go toolchain container nên không cần host Go/protoc. CI workflow đã thêm nhưng **GitHub-hosted workflow chưa chạy**; kết quả local không được ghi thành CI remote PASS.

## 3. Requirement → release → priority → code/test

Tên test dưới đây có tiền tố `Test`; log chứa toàn bộ tên và kết quả. Mỗi hàng là requirement group; số ORD-T là danh mục gốc, không phải tuyên bố toàn bộ mọi biến thể/feature trong danh mục đã được kiểm chứng.

| Requirement / ORD-T | Release, priority | Code và tests đã PASS với simulator |
| --- | --- | --- |
| Durable operation, 202/Location/status; validation fail chưa tạo Order / T03,T10 | Slice 1 / v1.0 P0/P1 | `checkout.go`, `placement.go`; V10PlacementAcceptanceAndRestartRecovery, V10FailedValidationReplayHasNoOrder, V10RegisteredIdentityGuestAndHTTPResource |
| Scoped key/hash, concurrency, 7d response/registry không reuse / T03,T04 | Slice 1 / v1.0 P0 | V10ConcurrentAcceptanceAndRetention, V10CancelOwnershipCASAndReplay; Postman 201 replay/409 |
| Redis mất sau acceptance; lost reserve ACK, release tombstone, fenced worker / T08,T09,T14 | Slice 1 / v1.0 P0 | V10PlacementAcceptanceAndRestartRecovery, V10LostReserveACKAndTombstone, V10FencedWorkerCannotOverwriteNewLease, V10InsufficientStockCompensation |
| Server quote ownership/expiry, canonical prices/address/input / T01,T05,T06 | Slice 1 / v1.0 P0/P1 | V10QuoteExpiryEligibilityAndOwnership, V10AcceptedPreviewIsNotCommercialOverride, V10ServerRejectedQuoteAndInput; MoneyBoundaries và schema smoke |
| Provider truth độc lập callback, signed/altered duplicate, allocations / T11,T12 | Slice 2 / v1.0 P0 | `payments.go`, simulator ledger; V10PaymentLostCallbackAndFinalizeACK, V10ConcurrentReceiptAndAlteredPayload, V10SignedCallbackValidation |
| Late/unmatched/under/over payment, statement cursor, expiry races / T11,T13 | Slice 2 / v1.0 P0 | V10UnderOverUnmatchedAndLateReceipts, V10StatementFindsCancelledAndUnmatchedMoney, V10InventoryExpiryWinsFinalization, V10PaymentTimeoutRace |
| PAID chỉ sau stock commit; history/outbox atomic / T14,T15 | Slice 2 / v1.0 P0 | V10PaymentLostCallbackAndFinalizeACK, V10OrderHistoryOutboxCommitIsAtomic; Postman payment finalization |
| COD UNPAID, ownership/CAS/self cancel, receipt cạnh tranh cancel / T01,T15 | Slice 3 / v1.0 P0 | V10CODIsUnpaidAndCanCancel, V10CancelOwnershipCASAndReplay, V10CODReceiptDuringCancelIsReconciled, V10SettledCODRequiresCancellationReview |
| Registered Inventory RPC, stable Kafka IDs / T17 | Slice 3 / v1.0 P1/P0 | V10InventoryTerminalRPCsRegisteredAndIdempotent, V10OutboxCrashWindowPublishesStableEventID |
| Ready generation, cancel vs packing, hold không cho accept / T16 | v1.1 P0 | `jobs.go`, fulfillment ledger; V11ReadyCancelBarrierAndStaleAuthorization, V11CancelVersusPackingAcceptance, V11HeldOrderCannotStartPacking, V11AcceptedTaskRejectsLaterCancellation |
| Signed source, resource/version/association, DEFERRED, dedup / T17,T18 | v1.1 P0/P1 | `events.go`, `runtime.go`; V11UnsignedSourceCannotForgeDelivery, V11OutOfOrderPackingAndShipping, V11KafkaConsumerDeduplicatesRealSourceMessages, V11InboxEffectRollbackAndReplay (T19) |
| Scoped queue/hold/unhold, trusted gRPC principal / T01,T24 | v1.1 P0/P1 | V11StaffScopeQueueAndHold, V11HoldReplayAndAuthorizedResume, V11GRPCRegisteredServerAndOwnership |
| Approved concurrent refund budget, UNKNOWN query/key / T21 | v1.2 P0 | `payments.go`, `jobs.go`; V12ConcurrentRefundBudgetAndUnknownRecovery |
| Partial/full refund, no auto restock/restart, paid cancel independent / T22 | v1.2 P0/P1 | V12FullRefundAndReplay, V12FailedRefundDoesNotRestartFulfillment, V12ApprovedPaidCancellationKeepsRefundIndependent |
| Care case/completion barrier / T23 | v1.2 P0 | V12CareCaseCompletionBarrier |
| Append-only local audit/HMAC tamper evidence | v1.2 P1 | V12AuditIsAppendOnlyAndTamperEvident; content và middle deletion được phát hiện |

Gate P0/P1 của **phạm vi sandbox đã triển khai** vượt qua local suite. Nghiệm thu integration thật và deployment production còn tách riêng; không bật adapter thật dựa trên kết quả này.

## 4. Những case còn thiếu hoặc thuộc đợt mở rộng

| Hạng mục | Release / priority | Trạng thái và tiêu chí trước khi bật |
| --- | --- | --- |
| Guest claim/OTP verifier (T02,T20) | Mở rộng P0 | NOT_RUN; chỉ Guest session scope hiện tại. Cần proof verifier, ownership transfer và revoke tests |
| Voucher hold/finalize (T07 và phần voucher T15), loyalty | Mở rộng P0 | FEATURE_DISABLED; cần adapter và fault suite riêng; không dùng COD/stock tests thay bằng chứng voucher |
| POS/marketplace multi-channel (T25), B2B/gifting | Mở rộng P0/P1 | FEATURE_DISABLED; CreatePOSOrder trả lỗi rõ ràng, cần sale registry/terminal policy |
| Arbitrary discount/free-shipping policy (phần T06) | Mở rộng P1 | Chưa có promotion engine; money helpers/DDL guards đã test, không coi đó là feature promotions PASS |
| Production provider/VietQR, Catalog, Inventory, Shipping, Fulfillment, Care | Integration P0/P1 | BLOCKED_BY_CONTRACT; cần authentication, finality, terminal/query/refund/barrier capability và integration evidence |
| Real Profile HTTP adapter | Integration P0/P1 | Code có mapping/credential riêng; NOT_RUN với Profile thật trong lần này |
| Kong runtime proxy | Deployment P1 | Route config cập nhật/parse local; NOT_RUN qua Kong container; backend signature/ownership đã test trực tiếp |
| T26 full outages, graceful shutdown kill, PITR restore; HA/load lớn | P2 trước scale/deployment | NOT_RUN toàn bộ; suite có restart/lease/Redis loss/ACK windows nhưng chưa full outage/PITR evidence |
| External audit anchor chống tail deletion privileged actor | P2, nâng P0 nếu policy yêu cầu | NOT_RUN; local HMAC chain không chứng minh tail truncation |
| Tracing exporter, production dashboards/alerts/SLO | P2 | Histogram/backlog có code; chưa có exporter hoặc dài hạn evidence |

Các gaps không được che bằng trạng thái PASS. Production mode chủ động từ chối startup; `.env.example` chỉ dùng sandbox. Extension features giữ tắt. Retention compaction/deletion cần policy PII riêng; không xóa registry hoặc nghĩa vụ đang UNKNOWN.

## 5. Bàn giao

Source, forward migrations 000001–000003, candidate schema đồng bộ, REST/protobuf/event v2 contracts, Compose, Postman regression, fault suite, CI và runbook nằm trong repository. [Environment manifest](environment-evidence.json) ghi image/schema/config/seed metadata; chỉ chứa cấu hình synthetic và hashes, không token người dùng. Không sửa/xóa phần công việc Profile có sẵn hoặc khôi phục các viewer files người dùng đã xóa.

## 6. Packaging: Postman local-only

Theo yêu cầu bàn giao nhánh `feat/order-service`, toàn bộ folder `postman/` được Git ignore và không đưa vào commit. Các logs Postman ở trên là evidence của lượt chạy local trước đó. CI mới bỏ qua collection khi không có folder, ghi NOT_RUN; Go HTTP integration vẫn chạy và contract checker tự tạo synthetic placement fixture khi sandbox chưa có Order. Thay đổi tooling này đã kiểm tra cú pháp; lượt chạy `--fresh-fixture` sau thay đổi bị BLOCKED vì Docker Desktop engine không chạy, không ghi thêm runtime PASS. Suite Go 41 tests trước đó không thay đổi code bởi việc packaging này.
