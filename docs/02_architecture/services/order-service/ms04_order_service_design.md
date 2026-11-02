# MS-04 — Master design map 3.2

Ngày13/10/2026. Nguồn chi tiết là các chương được liên kết dưới đây; master không sao chép nguyên bộ LLD để tránh hai bản mô tả khác nhau.

## Đọc để hiểu và trình bày

Để bắt đầu triển khai, dùng [Roadmap18](15_order_implementation_roadmap.md): scope/contracts → Core → persistence → worker/Inventory → messaging → placement → payment → controls/lifecycle. Roadmap tách Shipping Quote sớm khỏi shipment lifecycle và phân biệt integration milestone với quyền nhận tiền production.

Để review persistence/messaging, đọc [chương17](14_order_persistence_messaging_design.md) cùng [DDL candidate](order_persistence_messaging.candidate.sql): physical schema, transactions/isolation, commit UNKNOWN, publisher head protocol, inbox/replay và recovery sau restore. Các bảng chung ở chương03 là baseline, không thay candidate schema mới.

Để review Saga/worker, bắt đầu tại [chương13](13_order_saga_worker_runtime.md): job registry, state layers, claim/heartbeat/fencing, stale proof, retry/reconciliation và operations. Đây là runtime specification chung; các workflow giữ business guards ở chương09/11/12.

Để review Order Core bên trong, bắt đầu tại [chương12](12_order_core_design.md): aggregate/snapshot/commands/concurrency và các contract liên quan. Đây là addendum thiết kế, không phải báo cáo triển khai.

1. [Cart/Checkout chi tiết](11_cart_checkout_design.md): projection/selection, quote claim, admission boundary, Order+Intent cùng PLACE và cleanup.
2. [HLD Order](00_order_service_hld.md): scope, component boundaries, transport và checkout/payment flow.
3. [Domain](01_order_domain_and_boundary.md), [lifecycle](02_state_machine_and_lifecycle.md): định nghĩa trạng thái và điều kiện cho phép chuyển.
4. [Persistence](03_database_and_persistence.md), [use cases](04_usecases_and_saga_orchestration.md): local transactions, intent, side effect, recovery.
5. [Contracts](05_api_contracts_and_transports.md), [ADR/gaps](07_decisions_and_contract_gaps.md): protocol target, source wire, owner và rollout.
6. [Verification](06_testing_and_failure_recovery.md): evidence cần có trước khi nghiệm thu.
7. [Payment SePay + VietQR — review candidate 16/10/2026](09_payment_sepay_vietqr_design.md): lựa chọn provider, chi tiết QR/webhook/API v2, ledger/refund, API candidate và checklist review; chưa phải runtime đã triển khai.
8. [Phản hồi review Payment 1.1](10_payment_sepay_review_resolution.md): checkpoint cụ thể, manual refund staff contract, Guest access/revocation và VERIFYING có hạn.

## Baseline và evidence

HLD/LLD3.2 là thiết kế mục tiêu thống nhất. [Implementation3.1](08_implementation_and_acceptance.md) ghi runtime sandbox đã có; [report3.1](../../../04_testing/order-service/03_implementation_verification.md) ghi test lịch sử. Các báo cáo Inventory/change/alignment cũ đã được người dùng bỏ khỏi thư mục; không phục hồi hoặc coi việc sắp xếp tài liệu là đã đóng integration gaps.

Quyết định cốt lõi: operation-first; Payment module thuộc Order; gRPC finalize/query từ worker; paid là fact sau resources finalized; ready/cancel barrier; Inventory tự expire atomic; stable keys/tombstones; refund độc lập restock. Proto có tên method không chứng minh provider thật đã hỗ trợ.

[Chỉ mục toàn bộ](README.md). Sơ đồ Mermaid tại00/02/03 là source hiện hành; Draw.io/ảnh trước3.2 là reference lịch sử.
