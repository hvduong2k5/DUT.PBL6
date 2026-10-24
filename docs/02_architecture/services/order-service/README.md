# MS-04 Order Service — baseline thiết kế 3.2

**Hoàn thiện ngày 14/10/2026.** Bộ HLD/LLD này thống nhất thiết kế mục tiêu để phối hợp các service. Runtime hiện có là sandbox 3.1; provider thật chưa đủ mọi capability. Đọc gaps trước khi tuyên bố integration-ready.

| Tài liệu | Vai trò |
| --- | --- |
| [00 — HLD Order](00_order_service_hld.md) | boundary, happy/failure flows, transport và trade-offs |
| [01 — Domain](01_order_domain_and_boundary.md) | ownership, money/snapshot và invariants |
| [02 — Lifecycle](02_state_machine_and_lifecycle.md) | operation/order/payment/stock, deadline và barriers |
| [03 — Persistence](03_database_and_persistence.md) | runtime schema mapping, target fields, transaction/locks/retention |
| [04 — Use cases](04_usecases_and_saga_orchestration.md) | checkout/payment/COD/cancel/refund/recovery |
| [05 — Contracts](05_api_contracts_and_transports.md) | provider protocol, REST/gRPC/events, auth và migration |
| [06 — Verification](06_testing_and_failure_recovery.md) | test oracles, failure matrix và feature gates |
| [07 — ADR/gaps](07_decisions_and_contract_gaps.md) | selected decisions, owners, runtime3.1→target3.2 |
| [08 — Implementation3.1](08_implementation_and_acceptance.md) | lịch sử hành vi sandbox, không thay target3.2 |
| [09 — Payment SePay + VietQR](09_payment_sepay_vietqr_design.md) | thiết kế provider-specific để review: intent/QR, HMAC, API v2, receipt, đối soát, refund và migration |
| [10 — Phản hồi review Payment](10_payment_sepay_review_resolution.md) | sáu ý kiến review, thay đổi phiên bản1.1, Staff/Refund API, Guest grant và verification timeout |
| [11 — Cart/Checkout](11_cart_checkout_design.md) | projection, selected SKU, quote consumption, durable acceptance, PLACE + PaymentIntent và cleanup có điều kiện |
| [12 — Order Core](12_order_core_design.md) | aggregate invariants, OrderSnapshot v1, transition/command matrix, concurrency, Hold/Resume và event/query contracts |
| [13 — Saga/Worker Runtime](13_order_saga_worker_runtime.md) | state layers, job registry, lease/epoch/heartbeat, stale proof, execution/reconciliation budgets, fairness và recovery |
| [17 — Persistence/Messaging](17_order_persistence_messaging_design.md) | unified physical schema, isolation/commit UNKNOWN, stream publisher, inbox/offset/replay và restore correctness |
| [18 — Implementation Roadmap](18_order_implementation_roadmap.md) | giai đoạn0–8, ba PR đầu, Shipping Quote sớm, milestones A/B/C và production-money gates |
| [Master map](ms04_order_service_design.md) | thứ tự đọc và source-of-truth map, không duplicate các chương |

## Hiệu lực

- **Persistence coding checkpoint17/10/2026:** [PR2 evidence](../../../04_testing/order-service/05_persistence_foundation.md) ghi expand migration004, SQL identity/recovery, encrypted hydration và T3 atomicity đã kiểm trên PostgreSQL thật;41 legacy integration tests chạy trên isolated Redis/Kafka. Target executor/publisher chưa được bật; [ADR004–006](../../adr/order-service/README.md) ghi quyết định/gap implementation.
- **Core coding checkpoint17/10/2026:** target pure Core foundations đã được implement theo roadmap PR1; xem [evidence](../../../04_testing/order-service/04_order_core_foundations.md) và [coding ADRs](../../adr/order-service/README.md). Chưa chuyển sandbox routes sang target persistence/worker hoặc đóng integration gates.
- **Implementation roadmap review 17/10/2026:** [chương18](18_order_implementation_roadmap.md) thống nhất dependency và Definition of Done theo từng slice. Bắt đầu PR1 Core Foundations; Shipping Quote trước checkout, Payment race tests cùng giai đoạn6, reconciliation/manual refund và restore gates trước nhận tiền thật. Chưa đánh dấu giai đoạn nào đã triển khai chỉ từ roadmap.
- **Persistence/Messaging review 17/10/2026:** [chương17](17_order_persistence_messaging_design.md) và [DDL candidate](order_persistence_messaging.candidate.sql) thống nhất physical mapping và delivery/recovery protocol. Shared outbox hỗ trợ unmatched receipt/refund; publisher replay head và chấp nhận late duplicates, không hứa broker fencing từ DB lease. Chưa chạy migration hoặc nghiệm thu Kafka/provider fault tests.
- **Saga/Worker review 17/10/2026:** [chương13](13_order_saga_worker_runtime.md) là runtime specification chung. Execution budget tách reconciliation budget; MANUAL_REVIEW không dừng query tiền/stock UNKNOWN; stale worker mất quyền mutate nhưng external proof vẫn có đường ingest/recovery. Chưa phải runtime đã triển khai.
- **Order Core review 17/10/2026:** [chương12](12_order_core_design.md), [snapshot schema/examples](../../../03_api_specs/order-snapshot.v1.schema.json) và [Core API candidate](../../../03_api_specs/order-core.candidate.openapi.yaml) cụ thể hóa thiết kế nội bộ; là đề xuất để review. Các amendment Hold nhiều nguồn, Resume theo proof và snapshot versioning được liệt kê trong chương12.
- **Cart/Checkout review 16/10/2026:** [chương 11](11_cart_checkout_design.md) và [OpenAPI candidate](../../../03_api_specs/order-cart-checkout.candidate.openapi.yaml) đề xuất các invariant còn thiếu; chốt rõ Redis admission guard khác SQL durable commit. Chưa thay runtime hoặc bật integration thật.
- **Payment review 16/10/2026:** chọn SePay Bank Webhooks + API v2 + VietQR; [chương 09](09_payment_sepay_vietqr_design.md) và [OpenAPI candidate](../../../03_api_specs/order-payment-sepay.candidate.openapi.yaml) ghi đề xuất chi tiết. Chưa implement/activate; riêng cohort SePay đề xuất ACK sau observation durable rồi tạo canonical receipt qua API v2. Chưa đóng integration gaps hoặc thay runtime 3.1.
- Thiết kế mục tiêu: HLD00 và LLD01–07 cùng baseline3.2; [HLD hệ thống](../../high_level_design.md) tham chiếu cùng giao thức.
- Contract target: [Order OpenAPI 3.2](../../../03_api_specs/order-service.openapi.yaml), [shared proto](../../../../packages/proto/inventory/v1/inventory.proto), [semantics profile](../../../../packages/contracts/order-3.2.md), [event schemas v3](../../../../packages/events/schemas/order/v3/).
- Runtime sandbox: [Service README](../../../../services/order-service/README.md), [OpenAPI 3.1](../../../03_api_specs/order-service.sandbox-3.1.openapi.yaml), [migrations](../../../../services/order-service/migrations/000001_order.up.sql).
- [Schema snapshot3.1](order_schema_candidate.sql) là reference runtime, không phải migration3.2. [Evidence3.1](../../../04_testing/order-service/03_implementation_verification.md) chỉ chứng minh phạm vi đã chạy.
- Các capability mới cần sign-off/provider implementation/tests; gọi simulator không thay integration thật.

Mermaid trong00/02/03 là sơ đồ chuẩn3.2. Draw.io/PNG trong workspace là tài liệu hình ảnh lịch sử chưa đồng bộ3.2; không dùng để quyết định API/flow. Không khôi phục các ảnh người dùng đang xóa.

Core flow: operation durable → reserve → Order → receipt → resource finalize → PAID/COD confirmed → fulfillment-ready/barrier. Timeout UNKNOWN giữ obligation; receipt/refund/stock không đánh đồng.
