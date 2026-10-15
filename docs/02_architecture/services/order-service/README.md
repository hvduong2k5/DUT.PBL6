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
| [09 — Inventory review](09_inventory_integration_review_2026-10-13.md) | findings tại SHAs của đợt review trước |
| [10 — Báo cáo trình bày liên service](10_order_design_change_report.md) | so với HLD đầu, lý do đổi, trade-offs và kế hoạch phối hợp |
| [11 — Đồng bộ contract](11_contract_alignment_report.md) | wire proposal 15/10, kiểm chứng static/build và việc provider còn phải làm |
| [12 — Giải thích từ căn bản](12_order_service_explained_from_basics.md) | lời giải thích tiếng Việt, sơ đồ từng phần, thuật ngữ và lộ trình đọc mã nguồn |
| [Master map](ms04_order_service_design.md) | thứ tự đọc và source-of-truth map, không duplicate các chương |

## Hiệu lực

- Thiết kế mục tiêu: HLD00 và LLD01–07 cùng baseline3.2; [HLD hệ thống](../../high_level_design.md) tham chiếu cùng giao thức.
- Contract target: [Order OpenAPI 3.2](../../../03_api_specs/order-service.openapi.yaml), [shared proto](../../../../packages/proto/inventory/v1/inventory.proto), [semantics profile](../../../../packages/contracts/order-3.2.md), [event schemas v3](../../../../packages/events/schemas/order/v3/).
- Runtime sandbox: [Service README](../../../../services/order-service/README.md), [OpenAPI 3.1](../../../03_api_specs/order-service.sandbox-3.1.openapi.yaml), [migrations](../../../../services/order-service/migrations/000001_order.up.sql).
- [Schema snapshot3.1](order_schema_candidate.sql) là reference runtime, không phải migration3.2. [Evidence3.1](../../../04_testing/order-service/03_implementation_verification.md) chỉ chứng minh phạm vi đã chạy.
- Các capability mới cần sign-off/provider implementation/tests; gọi simulator không thay integration thật.

Mermaid trong00/02/03 là sơ đồ chuẩn3.2. Draw.io/PNG trong workspace là tài liệu hình ảnh lịch sử chưa đồng bộ3.2; không dùng để quyết định API/flow. Không khôi phục các ảnh người dùng đang xóa.

Core flow: operation durable → reserve → Order → receipt → resource finalize → PAID/COD confirmed → fulfillment-ready/barrier. Timeout UNKNOWN giữ obligation; receipt/refund/stock không đánh đồng.
