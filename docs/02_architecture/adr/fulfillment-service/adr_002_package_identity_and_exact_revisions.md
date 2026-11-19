# ADR-002 — Package identity và exact source revisions

Status: Proposed. Date:30/10/2026. Owner đề xuất: Fulfillment + Order + Shipping.

## Context

Shipping tạo shipment theo package, Order cần exact packing proof khi replay. Task assignment/checklist version thay đổi độc lập; latest task không còn là original accepted/sealed revision. Reopen sau seal có thể gặp event/create shipment đang bay.

## Decision

MVP một task/order và một package/task, identity UUID riêng, seal code globally unique không tái dùng. Persist immutable task revision tại mỗi mutation, với PACKING_ACCEPTED/PACKING_SEALED checkpoints và original timestamps. Packing event bind sealed version. Additive exact package proof cần peer review vì shared TaskRevision thiếu parcel/package fields.

Reopen sau seal disabled tới khi Order/Shipping stop/fencing và replacement lineage được thống nhất. Future multi-package bỏ unique task_id qua migration cùng line quantity coverage/partial completion contracts; không reinterpret historical package.

## Alternatives

Dùng order_id làm package ID khóa đường mở rộng. Dùng latest revision khi replay phá source time/version. COMPLETED→READY trực tiếp cho phép shipment cũ xuất hiện muộn. Reject.

## Consequences

Thêm parcel/revision/outbox storage và source adapter work. Retention phải giữ proof identities, không chỉ current state. FUL-T16/17/29–35 kiểm duplicate seal, exact lookup, delayed event và disabled reopen.
