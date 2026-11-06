# Order Core Foundations — implementation evidence

Ngày: 17/10/2026 · Scope: PR1 trong [Roadmap18](../../02_architecture/services/order-service/18_order_implementation_roadmap.md). Đây là local domain implementation evidence, chưa phải production/Core API/provider integration acceptance.

## 1. Đã triển khai

- [Money](../../../services/order-service/internal/domain/core/money.go): VND integer bounds, checked addition/multiplication và immutable value; commercial bound dùng chung với legacy MaxMoney.
- [Identity](../../../services/order-service/internal/domain/core/identity.go): typed selection/manifest/payload hashes, stable command/effect scope/key/hash replay comparison; warehouse association kiểm riêng theo Inventory profile.
- [Snapshot](../../../services/order-service/internal/domain/core/snapshot.go) và [canonical JSON](../../../services/order-service/internal/domain/core/canonical.go): typed schema1.0, required/explicit-null shape, semantic arithmetic/authority/chronology checks, NFC/UTF8/int64 canonical profile, immutable bytes và defensive views.
- [Order decisions](../../../services/order-service/internal/domain/core/order.go): guarded PLACE, exact allocation, first PAID, owner/Care pending cancel, timeout, uncommitted finalization failure, restriction-only blocks, readiness/completion eligibility. Current owner field tách original snapshot owner.
- [Tests](../../../services/order-service/internal/domain/core/core_test.go): shared snapshot golden fixtures, boundary/negative/association guards, concurrent immutable views và fuzz decode roundtrip.

Pure target package nằm cạnh legacy DTO; application sandbox không được chuyển sang target pipeline trong scope này. Chưa có target repository/reconstitution, complete-order executor, owner-claim workflow, multi-source proof authentication hoặc transactional milestone/outbox writer của cohort mới.

## 2. Lệnh và kết quả

Toolchain local: Go1.22.12 Windows/amd64, đúng dòng Go1.22 của service; downloaded vào temporary directory và kiểm SHA256 bằng metadata chính thức go.dev. Không đổi toolchain requirement hoặc upgrade dependency version. golang.org/x/text v0.16.0 đã có trong module được chuyển từ indirect sang direct do dùng Unicode NFC.

```text
go test -json ./... -count=1
go test -race ./internal/domain/... -count=1 -cover
go vet ./...
go test ./internal/domain/core -run '^$' -fuzz FuzzDecodeSnapshot -fuzztime 6s -parallel 2
```

- Full module compile/tests: PASS,66 passed test events gồm subtests/fuzz seeds;41 integration tests SKIPPED vì chưa cấu hình real PostgreSQL/Redis/Kafka environment. Package integration báo ok không có nghĩa41 tests đã chạy.
- Domain race detector: PASS với CGO/GCC local; concurrent snapshot/owner view test kiểm defensive-copy safety. Target Core statement coverage82.1%; legacy domain41.9%. Coverage không đại diện provider/DB acceptance.
- Vet: PASS cho toàn module.
- Fuzz: PASS,14290 executions trong lượt6s, không có failure corpus; đây là bounded fuzz run, không chứng minh parser tuyệt đối không lỗi.
- Shared JSON fixtures tiếp tục được kiểm bằng JSON Schema; Golden digest/expected logical hashes đã đổi theo ADR002, schema shape không đổi.
- Static design-contract regression PASS:3 OpenAPI specs,8 Order v3 schemas,9 valid event examples và32 rejected event cases. Protobuf regeneration/provider integration NOT_RUN; không sửa proto/generated/events trong scope này.

## 3. Correctness được bảo vệ và giới hạn

| Phần | Evidence hiện tại | Gate còn lại |
| --- | --- | --- |
| Money/snapshot/identity | Domain boundary/golden/negative/concurrent-view tests | Provider profile/authority fixtures và persistence mapping |
| Allocation/cancel/timeout/finalize | Pure decision/version/time/owner/resource association tests | PostgreSQL serialized race tests, actual DB decision clock |
| FIRST_PAID | Aggregate không apply lần hai | Unique milestone + root/history/outbox transaction |
| Late external proof | Core không cho hold/cancel/refund bị vượt hoặc terminal revival | Runtime append/query proof và compensation obligation phải implement PR3/Payment |
| Completion | Pure eligibility predicate | Authenticated Care decision, delivery/settlement source, executor/milestone transaction stage8 |
| Current owner | Field/view tách snapshot owner-at-placement | Verified claim command + durable ownership history |
| Sandbox compatibility | Full module compile và existing domain tests | Real stack regression41 tests hiện NOT_RUN |

Chưa apply migration, gọi SePay/Inventory thật, mở feature gate hoặc gửi tiền. Quyết định/gap coding được ghi ở [ADR index](../../02_architecture/adr/order-service/README.md), gồm Core isolation, digest mismatch và domain-vs-transaction/proof boundaries.
