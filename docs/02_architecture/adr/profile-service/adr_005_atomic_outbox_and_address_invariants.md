# ADR-005 — Transactional Outbox và khóa theo customer cho Address Book

- Ngày: 08/10/2026
- Trạng thái: Implemented

## Bối cảnh

Profile/claim mutation và outbox từng ghi riêng, bỏ qua lỗi outbox. Khóa các address row không giải quyết được trường hợp chưa có address. Xóa default để lại Address Book không có default. Address update có thể ghi đè thay đổi ở phiên khác.

## Quyết định

Mutation và event nằm trong cùng PostgreSQL transaction. Mọi address mutation khóa hàng parent customer theo cùng thứ tự trước khi truy cập child rows. Unique partial index bảo vệ tối đa một default; transaction giữ đúng một default khi còn địa chỉ.

Default thay thế theo `updated_at DESC, id ASC`. Address có positive version; HTTP update/delete/switch yêu cầu If-Match. Retry switch tới default hiện tại là no-op. Profile hỗ trợ ETag và giữ body version cho compatibility.

## Hệ quả

Write cùng customer được tuần tự hóa; các customer khác vẫn độc lập. Profile update cũng cạnh tranh trên parent row. Migration 000003 cần được áp dụng trước binary mới. Event publish vẫn at-least-once; consumer phải idempotent. Failed outbox insert rollback mutation; không báo success giả.
