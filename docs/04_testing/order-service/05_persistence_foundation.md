# Persistence & Transaction Foundation — implementation evidence

Ngày: 17/10/2026 · Scope: PR2 trong [Roadmap18](../../02_architecture/services/order-service/18_order_implementation_roadmap.md). Code đã implement và kiểm local trên real PostgreSQL; đây không phải rollout production hoặc target worker/provider integration acceptance.

## 1. Deliverables

| Thành phần | Implementation |
| --- | --- |
| Expand migration004 | [DDL](../../../services/order-service/migrations/000004_core_foundation.up.sql), [migration runner](../../../services/order-service/migrations/migrate.go); default legacy cohort, conditional target constraints, quote/fingerprint guard, initial workflow/jobs, payment config/intent, milestones và generic stream/outbox |
| SQL transaction boundary | [transaction.go](../../../services/order-service/internal/persistence/transaction.go): READ COMMITTED, bounded whole-tx SQLSTATE retry, explicit commit UNKNOWN và independent lookup |
| Accepted checkout | [store.go](../../../services/order-service/internal/persistence/store.go): scoped idempotency, SQL quote/fingerprint guard, encrypted immutable accepted input, workflow/job/audit atomic |
| T3 placement | [placement.go](../../../services/order-service/internal/persistence/placement.go): canonical binding và ORDER_PLACE lease fence; Order/items/Intent/history/milestone/outbox/cleanup+COD-ready intents/result/audit atomic |
| Encrypted hydration | [cipher.go](../../../services/order-service/internal/persistence/cipher.go), [read.go](../../../services/order-service/internal/persistence/read.go), [Core Restore](../../../services/order-service/internal/domain/core/restore.go); key-ID/purpose-bound AEAD, schema/invariant/hash/projection checks |
| Cohort isolation | Legacy claim/read/recovery/provider queries được fence; cross-cohort replay không fallback. Legacy accept mới ghi shared quote consumption cùng T0 |
| Tests/CI | [PG tests](../../../services/order-service/internal/persistence/store_test.go), [cross-package legacy tests](../../../services/order-service/tests/persistence/cohort_test.go), [Restore tests](../../../services/order-service/internal/domain/core/restore_test.go); compose test environment thêm ORDER_CORE_TEST_DATABASE |

## 2. Environment và kết quả

- Go1.22.12 Windows/amd64, CGO/GCC race detector; cùng toolchain tạm đã ghi trong PR1 evidence, không upgrade module dependency.
- PostgreSQL16.15 thật trong dedicated ephemeral postgres:16-alpine container; Redis7 và pinned Apache Kafka image của repository trong hai containers riêng. Random loopback ports, không dùng database/Redis/Kafka của app đang chạy.
- Mỗi test tạo/drop schema UUID riêng. Upgrade fixture thực thi schema1–3 với dữ liệu legacy rồi Apply004; fresh fixture thực thi cả bốn migration và replay Apply để kiểm idempotence.
- Provider trong41 integration tests là existing stateful simulator/registered gRPC test adapters, không phải SePay/Inventory production.

```text
go test -race -json ./... -count=1 -timeout 5m
go test -race ./internal/persistence ./tests/persistence ./internal/domain/... -count=1 -cover
go vet ./...
```

Full module run: **PASS130 test events,0 failed,0 skipped** (counts gồm subtests/fuzz seeds):

| Package scope | Passed events |
| --- | --- |
| Legacy domain |7 |
| Target Core |60 |
| Persistence (unit + real PostgreSQL) |21 |
| Legacy/target cohort integration |1 |
| Existing full-stack sandbox integration |41 |

Sau các hardening cuối về ORDER_PLACE kind/fence và replay authorization, affected persistence/Core/cohort suites chạy lại với race detector đều PASS. Statement coverage: persistence77.9%, Core81.9%, legacy domain41.9%. `go vet ./...` và gofmt check PASS. Actual VIETQR/COD placed payloads đều PASS Draft7 JSON Schema với format checker, subject/lifecycle bindings và LF-normalized schema hash pin. Không suy provider coverage từ statement coverage.

## 3. Oracles thực sự được kiểm

1. Fresh/upgrade migration, legacy historical Order ID khác planned ID vẫn đọc/update được; target planned identity có conditional FK.
2.12 concurrent same-key accept calls chỉ một operation/workflow/job/audit/quote-consumption. Hash khác conflict; hai keys cùng quote race chỉ một success; khác quote nhưng fingerprint cũ bị chặn.
3. Accepted input sống trong SQL encrypted bytes, đúng scope/hash, mutation bị trigger từ chối; expired quote và late transaction fault rollback toàn T0.
4. Real COMMIT rồi fault wrapper trả EOF: lookup trả đúng operation/Order, không rerun uncertain closure. Test rollback-before-EOF giữ UNKNOWN, connectivity restored replay cùng key. Primary pool unavailable sau committed ACK loss vẫn giữ UNKNOWN rồi lookup được khi mở kết nối lại.
5. SQL40001/40P01 retry cả closure và rollback partial writes; real two-transaction deadlock recover đúng tổng updates, max3 attempts khi luôn fail.
6. VIETQR và COD T3/load/replay; Intent chỉ có cho VIETQR. Order/items/history/milestone/outbox/cleanup/result cùng commit. Fault tại Intent/outbox insert không để lại partial Order hoặc orphan cleanup.
7. Stale epoch hoặc lease hết trong T3 không commit; đúng ORDER_PLACE kind/version/workflow association mới được vào boundary. Core root từ ciphertext phải khớp owner/status/version/money/resource/time/items projections.
8. SQL mutation history/snapshot hash/outbox bytes bị chặn. AEAD reject wrong purpose/tamper; keyring đọc được key cũ. Không tuyên bố đã rewrap snapshot tại chỗ.
9. Terminal response retention trả identity reference với expired error; current owner khác không đọc replay cũ. Shared schema pin LF-normalized hash được kiểm với actual placed.v3 contract.
10. Emitted VIETQR/COD placed payloads được kiểm bằng actual JSON Schema ngoài Go, subject/key/hash semantics kiểm riêng. Không có snapshot/phone/recipient trong event.
11. Actual legacy WorkPlacement/WorkJob/read/list/timeout/replay không xử lý target storage fixture;41 full-stack legacy regression tests cũng chạy với migration004 và quote-consumption guard mới.

## 4. Gap, quyết định và activation

- [ADR004](../../02_architecture/adr/order-service/adr_004_expand_migration_and_cohort_fencing.md): shared tables, generated conditional FK, cohort fences, physical name compatibility và migration subset.
- [ADR005](../../02_architecture/adr/order-service/adr_005_sql_retry_and_commit_unknown_recovery.md): SQL retry khác commit UNKNOWN; identity lookup, retention/current rights và test fault semantics.
- [ADR006](../../02_architecture/adr/order-service/adr_006_encrypted_hydration_and_atomic_placement.md): StateRecord/AEAD/hydration, T3 closure, payment-code contract và schema hash pin.

Test fixtures chuẩn bị canonical payload/leased ORDER_PLACE job bằng SQL: chưa có target executor chạy Catalog→Reserve→PLACE. No network trong T3. Cleanup/ready là durable jobs chưa thực thi Redis Lua hoặc phát FulfillmentReady. outbox_events đã persist v3 nhưng target publisher/inbox/rebalance protocol ở stage4 chưa được implement; Kafka tests hiện chứng minh legacy v2 regression.

Chưa switch cohort, backfill production, gọi provider thật, mở payment hoặc production mode. Không claim full42-table candidate, all lifecycle projections, multi-entry history, refund/reversal hoặc key-rotation/grants/restore drill đã hoàn thành. Bước tiếp theo là PR3 Worker Runtime + Inventory adapter, sử dụng chính persistence boundaries và fences này.
