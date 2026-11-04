# ADR-001 — Terminal admission tách packing lifecycle

Status: Proposed. Date:30/10/2026. Owner đề xuất: Fulfillment + Order + Inventory.

## Context

EPIC10 có task READY/ASSIGNED/IN_PROGRESS/BLOCKED/COMPLETED/CANCELLED, trong khi Order3.2 cần barrier OPEN/ACCEPTED/CANCELLED để chứng minh task có thể còn xuất hiện hay không. Inventory chỉ được reverse committed stock sau verified CANCELLED decision; một task bị block hoặc chưa tìm thấy không đủ.

## Decision

Lưu guard theo Order và admission theo generation. AcceptTask/CancelTask lock cùng guard kể cả absence, giữ durable tombstones/high-watermark và one-ever-accepted identity. ACCEPTED không đổi CANCELLED. Packing lifecycle riêng; local block không cấp reverse authority. Task accepted có sự cố được đưa vào operational recovery, không tự báo hủy thành công.

Replay original registry trước current eligibility/permit expiry, sau authentication/hash validation. Version admission và task tách biệt. Generation do Order cấp, không tự renew sau timeout.

## Alternatives

Một FSM task chung gây ambiguity CANCELLED và remote proof. Redis-only lock không có durable absence/uniqueness khi lease hết. Always-cancel accepted task có thể restock hàng đang được đóng gói. Reject các phương án này.

## Consequences

UI cần hai status/admission và workflow; product phải hiểu hard stop accepted chưa hỗ trợ. Tăng bảng/transaction nhưng race oracle rõ. FUL-T01–08/37 kiểm absence/late permit/generation/UNKNOWN. Future accepted-stop cần contract riêng và proof giữ nguyên historical ACCEPTED.
