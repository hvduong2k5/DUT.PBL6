# ADR-003 — Decision clock, proof boundary và milestone authority

- Ngày: 17/10/2026
- Trạng thái: Accepted — pure decision guards implemented; transactional/concurrent/provider acceptance chưa đạt từ lượt này.

## Bối cảnh và gap

State graph chỉ cho biết cạnh có thể tồn tại, không chứng minh allocation, permission, remote commit hoặc cancellation. Code mới phải xử lý equality deadline, control thay đổi trong lúc finalize bay và proof tới sau deadline. PR1 chưa có DB locks/unique milestones hoặc typed authenticated provider adapter.

## Quyết định

- DecisionContext truyền current expected version và decision time từ application sau lock; domain không time.Now/network I/O. Decision không được lùi trước last applied decision, không version overflow. Persistence mapping phải lưu version/updated decision time và registry replay trước stale-version check sau khi authorize.
- Allocate chỉ exact matching receipt/allocation association, đúng receiver/currency/amount, pending và now<deadline, không hold/cancel/refund block. Equality hết hạn. Canonical receipt/proof authentication do typed port chịu trách nhiệm, không cho public caller dựng evidence struct qua JSON binding.
- ConfirmPaid cần allocation đã accepted đúng hạn, correct reservation/order/digest/warehouse và monotonic resource version, original COMMITTED decision trước resource expiry. Cho phép apply sau local payment deadline khi guard còn đủ; không yêu cầu remote và local DB clocks tuyệt đối đồng bộ.
- Block là restriction-only. Generic clear-hold/reset-budget/resume không có trong PR1; full verified controls thuộc stage7. Timeout pending vẫn chạy khi hold; pending cancel sau expiry dùng CANCELLED_TIMEOUT. Care-approved pending cancel cần binding decision/order/version/policy/scope, không trust approval ID đơn lẻ.
- Rejected business transition không xóa proof hoặc money: application/runtime vẫn phải append/query proof và tạo obligations theo Worker13. Core không tự persist external facts; return error không phải lệnh bỏ observation, release kho hoặc refund tiền.
- FIRST_PAID được đánh dấu một lần trong aggregate value; durable uniqueness/history/outbox atomic là trách nhiệm PR2/Tpaid. CompletionEligibility là pure guard trên trusted Care/settlement/window/obligations, không tự execute CompleteOrder hoặc tạo completion milestone. Full completion authority/transaction được ghép stage8.

## Hệ quả và phần còn mở

Value-returning methods giữ root cũ khi reject; hai concurrent calls trên hai copies vẫn có thể ra hai quyết định nếu application không serialize. Unit stale-version tests chứng minh re-evaluation guard, không chứng minh PostgreSQL race. Domain evidence checks là association/state/time consistency, không là HMAC/service authentication hoặc authoritative gRPC query. Remote proof validator, event/command pipeline, DB constraints, transaction locks và live provider capability tiếp tục là gates ở roadmap.

## Kiểm chứng

Tests before/equal/after deadline, valid remote commit trước expiry nhưng query sau cutoff, mismatched resource/version/decision, hold/cancel/refund blocks, stale expected version, owner denial, terminal no-revival, expired/released finalization giữ allocation, COD unpaid readiness và completion prerequisites. Xem [verification](../../../04_testing/order-service/04_order_core_foundations.md).
