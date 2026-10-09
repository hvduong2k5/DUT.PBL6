# MS-04 Order Service — implementation 3.1

**Cập nhật 09/10/2026:** Go backend đã được triển khai cho luồng sandbox MVP v1.0/v1.1/v1.2. PostgreSQL, Redis và Kafka chạy thật; Catalog/Inventory/Fulfillment/Care/Payment dùng simulator có trạng thái. Xem [hành vi implementation](08_implementation_and_acceptance.md) và [evidence](../../../04_testing/order-service/03_implementation_verification.md) trước khi dùng các mô tả candidate v3.0.

| Tài liệu | Nội dung |
| --- | --- |
| [08 — Implementation và acceptance](08_implementation_and_acceptance.md) | Operation-first checkout, retention, financial truth, release gates và giới hạn thực tế |
| [01 — Domain và boundary](01_order_domain_and_boundary.md) | Ownership, invariants và snapshots |
| [02 — Lifecycle](02_state_machine_and_lifecycle.md) | Operation/Order/payment/stock states |
| [03 — Persistence](03_database_and_persistence.md) | Schema và local transaction boundaries |
| [04 — Use cases](04_usecases_and_saga_orchestration.md) | Saga, UNKNOWN, compensation và recovery |
| [05 — Contracts](05_api_contracts_and_transports.md) | REST/gRPC/event boundaries |
| [06 — Testing](06_testing_and_failure_recovery.md) | ORD-T01..26 và requirement traceability |
| [07 — Quyết định/gaps](07_decisions_and_contract_gaps.md) | Candidate decisions và những dependency thật còn cần đóng |
| [Master](ms04_order_service_design.md) | Các chương tổng hợp, có cập nhật implementation ở đầu |
| [Schema runtime candidate](order_schema_candidate.sql) | Ghép migrations forward; dùng database rỗng khi kiểm riêng |
| [Chạy service và tests](../../../../services/order-service/README.md) | Docker, Go gates và Postman |
| [Runbook](../../../06_operations/order_service_runbook.md) | Recovery, reconciliation và manual review |

Sơ đồ Draw.io và ảnh trong `diagram-image` được giữ theo workspace hiện tại; sequence Mermaid trong chương 08 mô tả protocol runtime mới. Các viewer/online-link/tools đã bị xóa trong workspace không được khôi phục tự động.

MVP acceptance khác Order creation: POST mới trả 202 và operation Location tồn tại; worker tạo Order atomically khi đã đủ điều kiện. Source of truth sau acceptance là PostgreSQL; mất Cart Redis không xóa intent hoặc external keys. PAID/ready/refund là các quyết định độc lập có guard.

Voucher, loyalty, Guest claim, POS/marketplace, B2B/gifting còn thuộc release mở rộng. Production startup hiện fail closed; provider callback của simulator không chứng minh hợp đồng VietQR thật.
