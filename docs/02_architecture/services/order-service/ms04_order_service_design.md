# MS-04 — Master design map 3.2

> **Phạm vi áp dụng MVP 1 — 15/10/2026:** đọc [quyết định giao bản đầu](13_mvp_delivery_decisions.md). Chỉ khách đăng nhập, VietQR chuyển khoản qua SePay Payment Gateway; không COD/Guest/voucher, không phương thức phụ, hoàn tiền thủ công có kiểm soát và dừng tự động tại DELIVERED. Các nhánh mở rộng dưới đây không bật trong MVP. Implementation sandbox của đợt đầu được ghi ở [báo cáo](14_mvp_implementation_and_adr_report.md); baseline 3.2 chưa được nghiệm thu tích hợp production.

Ngày13/10/2026. Nguồn chi tiết là các chương được liên kết dưới đây; master không sao chép nguyên bộ LLD để tránh hai bản mô tả khác nhau.

## Đọc để hiểu và trình bày

1. [Báo cáo before/after và trade-offs](10_order_design_change_report.md): lý do thay HLD/LLD cũ, mỗi service cần phối hợp gì.
2. [HLD Order](00_order_service_hld.md): scope, component boundaries, transport và checkout/payment flow.
3. [Domain](01_order_domain_and_boundary.md), [lifecycle](02_state_machine_and_lifecycle.md): định nghĩa trạng thái và điều kiện cho phép chuyển.
4. [Persistence](03_database_and_persistence.md), [use cases](04_usecases_and_saga_orchestration.md): local transactions, intent, side effect, recovery.
5. [Contracts](05_api_contracts_and_transports.md), [ADR/gaps](07_decisions_and_contract_gaps.md): protocol target, source wire, owner và rollout.
6. [Verification](06_testing_and_failure_recovery.md): evidence cần có trước khi nghiệm thu.

## Baseline và evidence

HLD/LLD3.2 là thiết kế mục tiêu thống nhất. [Implementation3.1](08_implementation_and_acceptance.md) ghi runtime sandbox đã có; [report3.1](../../../04_testing/order-service/03_implementation_verification.md) ghi test lịch sử. [Review Inventory](09_inventory_integration_review_2026-10-13.md) giữ SHAs/findings, không tự đóng integration gaps.

Quyết định cốt lõi: operation-first; Payment module thuộc Order; gRPC finalize/query từ worker; paid là fact sau resources finalized; ready/cancel barrier; Inventory tự expire atomic; stable keys/tombstones; refund độc lập restock. Proto có tên method không chứng minh provider thật đã hỗ trợ.

[Chỉ mục toàn bộ](README.md). Sơ đồ Mermaid tại00/02/03 là source hiện hành; Draw.io/ảnh trước3.2 là reference lịch sử.
