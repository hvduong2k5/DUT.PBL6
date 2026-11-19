# ADR-004 — Peer authority và candidate contracts

Status: Proposed. Date:30/10/2026. Owner đề xuất: Fulfillment + owning providers.

## Context

Shared MVP contracts đã ở dev nhưng implementation ở nhiều nhánh, một số method mới chỉ generated stub. Permit/event thiếu packing lines/allocation detail; Order source codec không tự nhận raw CloudEvent. Tài liệu có thể dễ bị đọc thành runtime-ready.

## Decision

Pin commit/blob sources, giữ docs candidate API/proto/DDL riêng và không sửa shared wire trong lượt chuẩn bị thiết kế. Propose typed Order packing snapshot + Inventory committed allocation/validation; provider authority vẫn ở owning service. List gates/owners/evidence; UNIMPLEMENTED/missing proof fail closed. Static checks chỉ xác minh artifact tĩnh, không đóng runtime acceptance.

## Alternatives

Copy peer DB/query SQL, dùng customer detail thay privileged snapshot, fill dummy allocation, hoặc fallback simulator khi provider down đều bỏ authority hoặc giấu gap. Reject.

## Consequences

Cần peer PRs/acceptance trước E2E, không thể chỉ implement Fulfillment rồi bật toàn luồng. Tests phân mức unit/static/real infra/peer E2E; source manifest giúp tái audit khi nhánh thay đổi. Contract promotion additive vào packages cần wire regression và consumer rollout riêng.
