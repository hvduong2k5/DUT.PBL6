# ADR-004 — Expand migration và cohort fencing

- Ngày: 17/10/2026
- Trạng thái: Accepted — migration004/repositories/fences implemented; target traffic chưa bật.
- Scope: PR2 theo [Roadmap18](../../services/order-service/18_order_implementation_roadmap.md).

## Bối cảnh và gap khi coding

Schema3.1 có nhiều tên bảng trùng candidate17 nhưng shape khác: snapshot là Quote, jobs.order_id bắt buộc, COD có online expiry giả, outbox v2 dùng JSONB, history chỉ một entry/version. Legacy workers/readers/recovery không lọc cohort. Chạy nguyên42-table candidate hoặc ghi target rows trước khi fence sẽ tạo schema conflict hoặc cho worker cũ giải mã/mutate dữ liệu mới sai cách.

## Quyết định

1. [Migration004](../../../../services/order-service/migrations/000004_core_foundation.up.sql) là expand trên bảng hiện có, default cohort `sandbox-3.1`; target repository ghi explicit `order-core-1`. Không rename/truncate bảng hoặc chuyển traffic. Shared planned_order_id/Order PK/operation uniqueness giữ identity xuyên cohorts.
2. Legacy placement/job claim, Order/operation reads, timeout/completion/recovery và provider matching chỉ đọc cohort legacy. Replay gặp registry target trả COHORT_ROUTING_REQUIRED, không fallback sang Redis/legacy acceptance. Target reader không fallback legacy cipher hoặc DTO.
3. orders.operation_id vẫn canonical link. c.order_id được cập nhật như compatibility projection cho sandbox schema; target lookup derive Order qua orders.operation_id. Conditional composite FK dùng generated core_operation_id/core_order_id để enforce target planned ID mà vẫn giữ historical legacy mapping khác planned ID.
4. Nullable expiry/provider_reference chỉ nới cho target theo conditional CHECK. Legacy giữ NOT NULL semantics qua CHECK; target COD có null online payment expiry, VIETQR có PaymentIntent riêng. Thêm owner/snapshot metadata và authenticated core_state_cipher, không giả legacy Quote thành v1 snapshot.
5. Physical jobs.status/key/lease_owner giữ tên/type cũ và thêm cohort/workflow/input columns; target status READY/LEASED/... map logical Job.state, key map logical_work_key. Không thêm cột state thứ hai hoặc queue có executor độc lập. Workflows và initial jobs được ghi từ T0; PR3 thêm actual executor/effects. T3 cần đúng ORDER_PLACE job/epoch, không dùng lease của VALIDATE để PLACE.
6. Shared quote-consumption table nhận cả target T0 và legacy acceptances mới trong cùng SQL transaction; quote đã consume không có marker Redis làm authority. Legacy quote history trước004 chưa tự backfill vì phải decrypt/verify source, không suy quote ID từ Order code. Trước switch target checkout phải pin quote cohort/version, drain/expire legacy previews và audit/backfill obligations theo migration plan.
7. New outbox_events/event_streams tách physical legacy outbox, cùng semantics generic aggregate identity. Legacy publisher chỉ thấy outbox v2, không tự publish target events. Chưa triển khai target Kafka publisher trong PR2.

## Hệ quả và gap còn mở

Migration là expand subset có thể thực thi, không phải toàn bộ42 bảng đã deploy. Payment config/intent và cleanup/ready durable intents được thêm sớm để T3 atomic; provider execution vẫn gated. History đang dùng existing UNIQUE(order_id,version); nhiều history entry/ordinal cùng version cần migration riêng trước feature đó. Stock REVERSED, refunds unmatched, full holds/commands/inbox và financial ledger expansion chưa được bật bởi004.

Snapshot cipher hiện vẫn chịu immutable trigger3.1 (không rewrap tại chỗ); audited rotation/archival path cần migration/grants riêng. Current owner/state hydration implemented, verified claim workflow chưa active. Production startup tiếp tục refused.

## Kiểm chứng

Real PostgreSQL16 fresh+upgrade+idempotent migration, gồm legacy Order có ID khác planned; old row vẫn update được. External package test gọi actual legacy WorkPlacement/WorkJob/Operation/GetOrder/ListOrders/SweepTimeouts và cross-cohort replay trên target storage fixtures, không chỉ mock SQL. Xem [PR2 evidence](../../../04_testing/order-service/05_persistence_foundation.md).
